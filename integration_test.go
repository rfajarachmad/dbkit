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

func TestOpen(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Open(ctx, Config{DSN: testDSN(t), MaxOpenConns: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if s := db.Stats(); s.MaxOpenConnections != 5 || s.OpenConnections != 1 {
		t.Errorf("MaxOpenConnections=%d OpenConnections=%d, want 5 and 1", s.MaxOpenConnections, s.OpenConnections)
	}
	var one int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
		t.Fatalf("SELECT 1: %d, %v", one, err)
	}
}
