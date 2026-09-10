package dbkit

import (
	"context"
	"database/sql"
	"errors"
	"sync"
)

// warmUp checks out n connections concurrently, pings each, then returns them
// to the pool so the first requests don't pay for connection setup.
func warmUp(ctx context.Context, db *sql.DB, n int) error {
	conns := make([]*sql.Conn, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			c, err := db.Conn(ctx)
			if err != nil {
				errs[i] = err
				return
			}
			conns[i] = c
			errs[i] = c.PingContext(ctx)
		})
	}
	wg.Wait()

	for _, c := range conns {
		if c != nil {
			c.Close()
		}
	}
	return errors.Join(errs...)
}
