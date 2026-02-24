// Package dbkit opens PostgreSQL connection pools for Go services.
//
// It wraps database/sql with the pgx driver and applies the pool settings
// our services share, so each service doesn't tune them on its own.
package dbkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
)

const (
	// DefaultMaxOpenConns is used when Config.MaxOpenConns is zero.
	DefaultMaxOpenConns = 10

	// DefaultDriverName is the database/sql driver registered by pgx.
	DefaultDriverName = "pgx"

	connMaxLifetime = 10 * time.Minute
)

// ErrEmptyDSN is returned by Open when Config.DSN is empty.
var ErrEmptyDSN = errors.New("dbkit: empty DSN")

// Config describes a connection pool.
type Config struct {
	// DSN is a PostgreSQL connection string in URL or keyword/value form.
	DSN string

	// MaxOpenConns limits the number of open connections.
	// Zero means DefaultMaxOpenConns.
	MaxOpenConns int

	// DriverName selects a registered database/sql driver, for example one
	// wrapped with tracing. Empty means DefaultDriverName.
	DriverName string

	// SkipWarmUp stops Open from opening every connection up front.
	SkipWarmUp bool

	// Logger receives pool lifecycle messages. Nil means slog.Default().
	Logger *slog.Logger
}

// poolSettings are the database/sql pool limits derived from a Config.
type poolSettings struct {
	maxOpen     int
	maxIdle     int
	maxLifetime time.Duration
}

func (c Config) pool() poolSettings {
	n := c.MaxOpenConns
	if n <= 0 {
		n = DefaultMaxOpenConns
	}
	// Keep every open connection idle-able: with the database/sql default of
	// two idle connections the pool churns connections under steady load.
	return poolSettings{maxOpen: n, maxIdle: n, maxLifetime: connMaxLifetime}
}

func (p poolSettings) apply(db *sql.DB) {
	db.SetMaxOpenConns(p.maxOpen)
	db.SetMaxIdleConns(p.maxIdle)
	db.SetConnMaxLifetime(p.maxLifetime)
}

// Open creates a connection pool, applies the pool settings and, unless
// SkipWarmUp is set, opens MaxOpenConns connections before returning.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, ErrEmptyDSN
	}
	driver := cfg.DriverName
	if driver == "" {
		driver = DefaultDriverName
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	db, err := sql.Open(driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("dbkit: open: %w", err)
	}
	p := cfg.pool()
	p.apply(db)

	if !cfg.SkipWarmUp {
		start := time.Now()
		if err := warmUp(ctx, db, p.maxOpen); err != nil {
			db.Close()
			return nil, fmt.Errorf("dbkit: warm up: %w", err)
		}
		logger.Info("database pool warmed", "conns", p.maxOpen, "took", time.Since(start).Round(time.Millisecond))
	} else if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("dbkit: ping: %w", err)
	}

	logger.Info("database pool ready", "max_open_conns", p.maxOpen)
	return db, nil
}

// MustOpen is like Open but panics on error.
//
// Deprecated: use Open and handle the error.
func MustOpen(ctx context.Context, cfg Config) *sql.DB {
	db, err := Open(ctx, cfg)
	if err != nil {
		panic(err)
	}
	return db
}

// warmUp checks out n connections at once, pings each, then returns them
// to the pool so the first requests don't pay for connection setup.
func warmUp(ctx context.Context, db *sql.DB, n int) error {
	conns := make([]*sql.Conn, 0, n)
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()
	for range n {
		c, err := db.Conn(ctx)
		if err != nil {
			return err
		}
		conns = append(conns, c)
		if err := c.PingContext(ctx); err != nil {
			return err
		}
	}
	return nil
}
