package dbkit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPoolDefaults(t *testing.T) {
	got := Config{DSN: "postgres://localhost/app"}.pool()
	want := poolSettings{maxOpen: DefaultMaxOpenConns, maxIdle: DefaultMaxOpenConns, maxLifetime: 10 * time.Minute}
	if got != want {
		t.Fatalf("pool() = %+v, want %+v", got, want)
	}
}

func TestPoolMaxOpenConns(t *testing.T) {
	for _, n := range []int{1, 20, 50} {
		got := Config{MaxOpenConns: n}.pool()
		if got.maxOpen != n || got.maxIdle != n {
			t.Errorf("MaxOpenConns=%d: got open=%d idle=%d", n, got.maxOpen, got.maxIdle)
		}
	}
}

func TestPoolNegativeMaxOpenConns(t *testing.T) {
	if got := (Config{MaxOpenConns: -1}).pool().maxOpen; got != DefaultMaxOpenConns {
		t.Fatalf("maxOpen = %d, want %d", got, DefaultMaxOpenConns)
	}
}

func TestOpenEmptyDSN(t *testing.T) {
	_, err := Open(context.Background(), Config{})
	if !errors.Is(err, ErrEmptyDSN) {
		t.Fatalf("err = %v, want ErrEmptyDSN", err)
	}
}

func TestOpenUnknownDriver(t *testing.T) {
	_, err := Open(context.Background(), Config{DSN: "postgres://localhost/app", DriverName: "nope"})
	if err == nil {
		t.Fatal("expected an error for an unregistered driver")
	}
}
