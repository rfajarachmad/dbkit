package dbkit

import (
	"errors"
	"testing"
	"time"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		wantPC   PoolConfig
		wantConn string
	}{
		{
			name:     "url defaults",
			dsn:      "postgres://app:secret@db:5432/app?sslmode=disable",
			wantPC:   PoolConfig{MaxConns: DefaultMaxConns, WarmPool: true},
			wantConn: "postgres://app:secret@db:5432/app?sslmode=disable",
		},
		{
			name: "url with pool params",
			dsn:  "postgresql://app@db/app?sslmode=require&max_conns=20&max_conn_lifetime=10m&max_conn_idle_time=90s&warm_pool=false",
			wantPC: PoolConfig{
				MaxConns:        20,
				MaxConnLifetime: 10 * time.Minute,
				MaxConnIdleTime: 90 * time.Second,
				WarmPool:        false,
			},
			wantConn: "postgresql://app@db/app?sslmode=require",
		},
		{
			name:     "keyword defaults",
			dsn:      "host=db port=5432 user=app dbname=app",
			wantPC:   PoolConfig{MaxConns: DefaultMaxConns, WarmPool: true},
			wantConn: "host=db port=5432 user=app dbname=app",
		},
		{
			name: "keyword with pool params and quoting",
			dsn:  `host=db user=app password='it\'s' max_conn_lifetime = 10m application_name='px api' max_conns=50`,
			wantPC: PoolConfig{
				MaxConns:        50,
				MaxConnLifetime: 10 * time.Minute,
				WarmPool:        true,
			},
			wantConn: `host=db user=app password='it\'s' application_name='px api'`,
		},
		{
			name:     "keyword escaped quote",
			dsn:      `host=db password='a\'b c'`,
			wantPC:   PoolConfig{MaxConns: DefaultMaxConns, WarmPool: true},
			wantConn: `host=db password='a\'b c'`,
		},
		{
			name:     "lifetime zero",
			dsn:      "postgres://db/app?max_conn_lifetime=0s",
			wantPC:   PoolConfig{MaxConns: DefaultMaxConns, WarmPool: true},
			wantConn: "postgres://db/app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc, conn, err := ParseDSN(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}
			if pc != tt.wantPC {
				t.Errorf("PoolConfig = %+v, want %+v", pc, tt.wantPC)
			}
			if conn != tt.wantConn {
				t.Errorf("conn string = %q, want %q", conn, tt.wantConn)
			}
		})
	}
}

func TestParseDSNErrors(t *testing.T) {
	tests := []struct {
		dsn       string
		wantParam string
	}{
		{"postgres://db/app?max_conns=0", "max_conns"},
		{"postgres://db/app?max_conn_lifetime=10", "max_conn_lifetime"},
		{"postgres://db/app?max_conn_idle_time=-1m", "max_conn_idle_time"},
		{"postgres://db/app?warm_pool=maybe", "warm_pool"},
		{"host=db max_conns=x", "max_conns"},
		{"host=db password='open", "password"},
		{"host=db =x", ""},
		{"host", ""},
	}
	for _, tt := range tests {
		_, _, err := ParseDSN(tt.dsn)
		var dsnErr *DSNError
		if !errors.As(err, &dsnErr) {
			t.Errorf("ParseDSN(%q) err = %v, want a DSNError", tt.dsn, err)
			continue
		}
		if dsnErr.Param != tt.wantParam {
			t.Errorf("ParseDSN(%q) param = %q, want %q", tt.dsn, dsnErr.Param, tt.wantParam)
		}
	}
}

func TestRedact(t *testing.T) {
	tests := map[string]string{
		"postgres://app:secret@db/app":       "postgres://app:xxxxx@db/app",
		"host=db user=app password=secret":   "host=db user=app password=xxxxx",
		"host=db password='two words' x=1":   "host=db password=xxxxx x=1",
		"host=db password='unterminated":     "<invalid dsn>",
		"postgres://app@db/app?sslmode=none": "postgres://app@db/app?sslmode=none",
	}
	for in, want := range tests {
		if got := Redact(in); got != want {
			t.Errorf("Redact(%q) = %q, want %q", in, got, want)
		}
	}
}
