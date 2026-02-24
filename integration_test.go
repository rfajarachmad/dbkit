package dbkit

import (
	"context"
	"os"
	"testing"
	"time"
)

// testDSN returns the DSN of a PostgreSQL server for integration tests,
// or skips the test when DBKIT_TEST_DSN is not set.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DBKIT_TEST_DSN")
	if dsn == "" {
		t.Skip("DBKIT_TEST_DSN not set")
	}
	return dsn
}

func TestOpenWarmsPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Open(ctx, Config{DSN: testDSN(t), MaxOpenConns: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := db.Stats()
	if s.MaxOpenConnections != 5 {
		t.Errorf("MaxOpenConnections = %d, want 5", s.MaxOpenConnections)
	}
	if s.OpenConnections != 5 || s.Idle != 5 {
		t.Errorf("after warm up: open=%d idle=%d, want 5 and 5", s.OpenConnections, s.Idle)
	}
}

func TestOpenSkipWarmUp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Open(ctx, Config{DSN: testDSN(t), MaxOpenConns: 5, SkipWarmUp: true})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections = %d, want 1", got)
	}
}

func TestQueriesKeepConnectionsOpen(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Open(ctx, Config{DSN: testDSN(t), MaxOpenConns: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for range 50 {
		var one int
		if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
			t.Fatal(err)
		}
	}
	s := db.Stats()
	if s.OpenConnections != 3 || s.MaxIdleClosed != 0 {
		t.Errorf("open=%d maxIdleClosed=%d, want 3 and 0", s.OpenConnections, s.MaxIdleClosed)
	}
}
