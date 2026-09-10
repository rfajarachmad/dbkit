package dbkit

import (
	"context"
	"errors"
	"testing"
)

func TestOpenEmptyDSN(t *testing.T) {
	_, err := Open(context.Background(), "")
	if !errors.Is(err, ErrEmptyDSN) {
		t.Fatalf("err = %v, want ErrEmptyDSN", err)
	}
}

func TestOpenInvalidPoolParam(t *testing.T) {
	_, err := Open(context.Background(), "postgres://localhost/app?max_conns=lots")
	var dsnErr *DSNError
	if !errors.As(err, &dsnErr) || dsnErr.Param != "max_conns" {
		t.Fatalf("err = %v, want a DSNError for max_conns", err)
	}
}

func TestOpenUnknownDriver(t *testing.T) {
	_, err := Open(context.Background(), "postgres://localhost/app", WithDriverName("nope"))
	if err == nil {
		t.Fatal("expected an error for an unregistered driver")
	}
}
