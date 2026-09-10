package dbkit

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"
)

// testDSN returns the DSN of a PostgreSQL server for integration tests with
// params added to its query string, or skips the test when DBKIT_TEST_DSN is
// not set. DBKIT_TEST_DSN must be in URL form.
func testDSN(t *testing.T, params url.Values) string {
	t.Helper()
	dsn := os.Getenv("DBKIT_TEST_DSN")
	if dsn == "" {
		t.Skip("DBKIT_TEST_DSN not set")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("DBKIT_TEST_DSN: %v", err)
	}
	q := u.Query()
	for k, v := range params {
		q[k] = v
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestOpenWarmsPool(t *testing.T) {
	ctx := testContext(t)
	db, err := Open(ctx, testDSN(t, url.Values{"max_conns": {"5"}}))
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

func TestWithMaxConnsOverridesDSN(t *testing.T) {
	ctx := testContext(t)
	db, err := Open(ctx, testDSN(t, url.Values{"max_conns": {"5"}}), WithMaxConns(3))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if got := db.Stats().OpenConnections; got != 3 {
		t.Errorf("OpenConnections = %d, want 3", got)
	}
}

func TestWarmPoolOff(t *testing.T) {
	ctx := testContext(t)
	db, err := Open(ctx, testDSN(t, url.Values{"max_conns": {"5"}, "warm_pool": {"false"}}))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if got := db.Stats().OpenConnections; got != 1 {
		t.Errorf("OpenConnections = %d, want 1", got)
	}
}

// TestMaxConnLifetime checks that connections are replaced once they reach
// max_conn_lifetime, and are kept when it is not set.
func TestMaxConnLifetime(t *testing.T) {
	tests := []struct {
		name       string
		params     url.Values
		wantClosed bool
	}{
		{"set", url.Values{"max_conns": {"3"}, "max_conn_lifetime": {"1s"}}, true},
		{"unset", url.Values{"max_conns": {"3"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext(t)
			db, err := Open(ctx, testDSN(t, tt.params))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				var one int
				if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
					t.Fatal(err)
				}
				time.Sleep(50 * time.Millisecond)
			}

			closed := db.Stats().MaxLifetimeClosed
			if tt.wantClosed && closed == 0 {
				t.Error("MaxLifetimeClosed = 0, want connections replaced after 1s")
			}
			if !tt.wantClosed && closed != 0 {
				t.Errorf("MaxLifetimeClosed = %d, want 0", closed)
			}
		})
	}
}

func TestQueriesKeepConnectionsOpen(t *testing.T) {
	ctx := testContext(t)
	db, err := Open(ctx, testDSN(t, url.Values{"max_conns": {"3"}}))
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
