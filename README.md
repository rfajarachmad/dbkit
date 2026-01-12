# dbkit

Shared PostgreSQL connection helper for Go services. It opens a `database/sql`
pool on the [pgx](https://github.com/jackc/pgx) driver and applies the pool
settings our services share.

## Install

```bash
go get github.com/rfajarachmad/dbkit
```

## Usage

```go
db, err := dbkit.Open(ctx, dbkit.Config{
	DSN:          os.Getenv("DB_DSN"),
	MaxOpenConns: 20,
	Logger:       logger,
})
if err != nil {
	return err
}
defer db.Close()
```

### Tracing

Register a wrapped driver and pass its name:

```go
driverName, err := otelsql.Register("pgx")
if err != nil {
	return err
}
db, err := dbkit.Open(ctx, dbkit.Config{DSN: dsn, DriverName: driverName})
```

## Tests

Unit tests run with `go test ./...`. Integration tests need a PostgreSQL server:

```bash
DBKIT_TEST_DSN='postgres://postgres@localhost:5432/postgres?sslmode=disable' go test ./...
```
