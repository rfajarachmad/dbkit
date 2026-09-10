package dbkit

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultMaxConns is used when neither the DSN nor WithMaxConns sets a limit.
const DefaultMaxConns = 10

// PoolConfig holds the pool settings read from a DSN.
type PoolConfig struct {
	// MaxConns limits open connections (max_conns). Every open connection
	// may stay idle, so the pool never churns under steady load.
	MaxConns int

	// MaxConnLifetime closes connections older than this (max_conn_lifetime).
	// Zero means connections are reused regardless of age.
	MaxConnLifetime time.Duration

	// MaxConnIdleTime closes connections idle for longer than this
	// (max_conn_idle_time). Zero means no limit.
	MaxConnIdleTime time.Duration

	// WarmPool opens MaxConns connections in Open (warm_pool, default true).
	WarmPool bool
}

func (pc PoolConfig) apply(db *sql.DB) {
	db.SetMaxOpenConns(pc.MaxConns)
	db.SetMaxIdleConns(pc.MaxConns)
	db.SetConnMaxLifetime(pc.MaxConnLifetime)
	db.SetConnMaxIdleTime(pc.MaxConnIdleTime)
}

// poolParams are the DSN parameters consumed by dbkit. They are removed from
// the connection string before it reaches the driver, which would otherwise
// send them to the server as runtime parameters.
var poolParams = map[string]func(*PoolConfig, string) error{
	"max_conns": func(pc *PoolConfig, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return fmt.Errorf("must be a positive integer")
		}
		pc.MaxConns = n
		return nil
	},
	"max_conn_lifetime": func(pc *PoolConfig, v string) error {
		return parseDuration(v, &pc.MaxConnLifetime)
	},
	"max_conn_idle_time": func(pc *PoolConfig, v string) error {
		return parseDuration(v, &pc.MaxConnIdleTime)
	},
	"warm_pool": func(pc *PoolConfig, v string) error {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("must be true or false")
		}
		pc.WarmPool = b
		return nil
	},
}

func parseDuration(v string, dst *time.Duration) error {
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 {
		return fmt.Errorf("must be a non-negative duration such as 30m")
	}
	*dst = d
	return nil
}

// ParseDSN reads the pool parameters from a PostgreSQL DSN in URL or
// keyword/value form and returns them with the DSN they were removed from.
//
// Supported parameters: max_conns, max_conn_lifetime, max_conn_idle_time and
// warm_pool. Parameters not present keep their defaults: DefaultMaxConns,
// no lifetime or idle limit, and warm-up on.
func ParseDSN(dsn string) (PoolConfig, string, error) {
	pc := PoolConfig{MaxConns: DefaultMaxConns, WarmPool: true}
	var (
		rest string
		err  error
	)
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		rest, err = extractURL(dsn, &pc)
	} else {
		rest, err = extractKeywords(dsn, &pc)
	}
	if err != nil {
		return PoolConfig{}, "", err
	}
	return pc, rest, nil
}

func extractURL(dsn string, pc *PoolConfig) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", &DSNError{Err: err}
	}
	q := u.Query()
	for key, set := range poolParams {
		if !q.Has(key) {
			continue
		}
		if err := set(pc, q.Get(key)); err != nil {
			return "", &DSNError{Param: key, Err: err}
		}
		q.Del(key)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func extractKeywords(dsn string, pc *PoolConfig) (string, error) {
	pairs, err := splitKeywords(dsn)
	if err != nil {
		return "", err
	}
	var kept []string
	for _, p := range pairs {
		if set, ok := poolParams[p.key]; ok {
			if err := set(pc, p.value); err != nil {
				return "", &DSNError{Param: p.key, Err: err}
			}
			continue
		}
		kept = append(kept, p.key+"="+quoteKeywordValue(p.value))
	}
	return strings.Join(kept, " "), nil
}

type keywordPair struct{ key, value string }

// splitKeywords parses "key=value key2='quoted value'" as libpq does.
func splitKeywords(s string) ([]keywordPair, error) {
	var pairs []keywordPair
	i := 0
	skipSpace := func() {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
			i++
		}
	}
	for {
		skipSpace()
		if i >= len(s) {
			return pairs, nil
		}
		start := i
		for i < len(s) && s[i] != '=' && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		key := s[start:i]
		skipSpace()
		if i >= len(s) || s[i] != '=' || key == "" {
			return nil, &DSNError{Err: fmt.Errorf("expected key=value near %q", s[start:])}
		}
		i++ // '='
		skipSpace()

		var b strings.Builder
		if i < len(s) && s[i] == '\'' {
			i++
			closed := false
			for i < len(s) {
				c := s[i]
				if c == '\\' && i+1 < len(s) {
					b.WriteByte(s[i+1])
					i += 2
					continue
				}
				if c == '\'' {
					i++
					closed = true
					break
				}
				b.WriteByte(c)
				i++
			}
			if !closed {
				return nil, &DSNError{Param: key, Err: fmt.Errorf("unterminated quoted value")}
			}
		} else {
			for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' {
				if s[i] == '\\' && i+1 < len(s) {
					i++
				}
				b.WriteByte(s[i])
				i++
			}
		}
		pairs = append(pairs, keywordPair{key: key, value: b.String()})
	}
}

func quoteKeywordValue(v string) string {
	if v != "" && !strings.ContainsAny(v, ` '\`+"\t\n") {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v) + "'"
}

// DSNError reports an invalid DSN or pool parameter.
type DSNError struct {
	Param string
	Err   error
}

func (e *DSNError) Error() string {
	if e.Param == "" {
		return "dbkit: invalid DSN: " + e.Err.Error()
	}
	return fmt.Sprintf("dbkit: invalid DSN parameter %s: %v", e.Param, e.Err)
}

func (e *DSNError) Unwrap() error { return e.Err }
