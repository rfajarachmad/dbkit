# dbkit

Shared PostgreSQL connection helper for Go services. It opens a `database/sql`
pool on the [pgx](https://github.com/jackc/pgx) driver, reads pool settings
from the DSN and warms the pool on start.

## Install

```bash
go get github.com/rfajarachmad/dbkit/v2
```

## Usage

```go
db, err := dbkit.Open(ctx, os.Getenv("DB_DSN"),
	dbkit.WithMaxConns(20),
	dbkit.WithLogger(logger),
)
if err != nil {
	return err
}
defer db.Close()
```

### Pool settings

Set these in the DSN, in URL or keyword/value form. They are removed before
the DSN is passed to the driver.

| Parameter | Default | Meaning |
| --- | --- | --- |
| `max_conns` | 10 | Maximum open connections. All of them may stay idle. `WithMaxConns` takes precedence |
| `max_conn_lifetime` | none | Close connections older than this, for example `30m` |
| `max_conn_idle_time` | none | Close connections idle longer than this |
| `warm_pool` | `true` | Open `max_conns` connections in `Open`. `WithWarmUp` takes precedence |

```text
postgres://app@db:5432/app?sslmode=require&max_conns=20&max_conn_idle_time=5m
host=db user=app dbname=app max_conns=20 warm_pool=false
```

### Tracing

Register a wrapped driver and pass its name:

```go
driverName, err := otelsql.Register("pgx")
if err != nil {
	return err
}
db, err := dbkit.Open(ctx, dsn, dbkit.WithDriverName(driverName))
```

## Upgrading from v1

| v1 | v2 |
| --- | --- |
| `import "github.com/rfajarachmad/dbkit"` | `import "github.com/rfajarachmad/dbkit/v2"` |
| `dbkit.Open(ctx, dbkit.Config{DSN: dsn, ...})` | `dbkit.Open(ctx, dsn, opts...)` |
| `Config.MaxOpenConns` | `WithMaxConns` or `max_conns` |
| `Config.DriverName` | `WithDriverName` |
| `Config.Logger` | `WithLogger` |
| `Config.SkipWarmUp` | `WithWarmUp(false)` or `warm_pool=false` |
| `MustOpen` | removed |

See [CHANGELOG.md](CHANGELOG.md) for the full list of changes.

## Tests

Unit tests run with `go test ./...`. Integration tests need a PostgreSQL server
and a DSN in URL form:

```bash
DBKIT_TEST_DSN='postgres://postgres@localhost:5432/postgres?sslmode=disable' go test ./...
```
