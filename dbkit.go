// Package dbkit opens PostgreSQL connection pools for Go services.
//
// It wraps database/sql with the pgx driver. Pool settings come from the DSN,
// so a service can be retuned through its connection string alone:
//
//	postgres://app@db:5432/app?sslmode=require&max_conns=20&max_conn_lifetime=30m
//
// See ParseDSN for the supported parameters.
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

// DefaultDriverName is the database/sql driver registered by pgx.
const DefaultDriverName = "pgx"

// ErrEmptyDSN is returned by Open when the DSN is empty.
var ErrEmptyDSN = errors.New("dbkit: empty DSN")

type options struct {
	driverName string
	logger     *slog.Logger
	maxConns   int
	warmUp     *bool
}

// Option customises Open.
type Option func(*options)

// WithDriverName selects a registered database/sql driver, for example one
// wrapped with tracing.
func WithDriverName(name string) Option {
	return func(o *options) { o.driverName = name }
}

// WithLogger sets the logger for pool lifecycle messages.
func WithLogger(l *slog.Logger) Option {
	return func(o *options) { o.logger = l }
}

// WithMaxConns sets the maximum number of open connections. It takes
// precedence over max_conns in the DSN.
func WithMaxConns(n int) Option {
	return func(o *options) { o.maxConns = n }
}

// WithWarmUp turns warm-up on or off. It takes precedence over warm_pool in
// the DSN.
func WithWarmUp(enabled bool) Option {
	return func(o *options) { o.warmUp = &enabled }
}

// Open creates a connection pool from dsn, applies its pool settings and,
// unless warm-up is off, opens every connection before returning.
func Open(ctx context.Context, dsn string, opts ...Option) (*sql.DB, error) {
	if dsn == "" {
		return nil, ErrEmptyDSN
	}
	o := options{driverName: DefaultDriverName, logger: slog.Default()}
	for _, opt := range opts {
		opt(&o)
	}

	pc, connString, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	if o.maxConns > 0 {
		pc.MaxConns = o.maxConns
	}
	if o.warmUp != nil {
		pc.WarmPool = *o.warmUp
	}

	db, err := sql.Open(o.driverName, connString)
	if err != nil {
		return nil, fmt.Errorf("dbkit: open: %w", err)
	}
	pc.apply(db)

	if pc.WarmPool {
		start := time.Now()
		if err := warmUp(ctx, db, pc.MaxConns); err != nil {
			db.Close()
			return nil, fmt.Errorf("dbkit: warm up: %w", err)
		}
		o.logger.Info("database pool warmed", "conns", pc.MaxConns, "took", time.Since(start).Round(time.Millisecond))
	} else if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("dbkit: ping: %w", err)
	}

	o.logger.Info("database pool ready", "max_open_conns", pc.MaxConns, "dsn", Redact(connString))
	return db, nil
}
