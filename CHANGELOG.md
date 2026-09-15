# Changelog

All notable changes to this module are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the module uses
[Semantic Versioning](https://semver.org/).

## [2.0.0] - 2026-09-15

Configuration now comes from the DSN, so services can retune their pool
through the connection string alone. This release changes the module path and
the `Open` signature; see "Upgrading from v1" in the README.

### Changed
- **Breaking:** module path is now `github.com/rfajarachmad/dbkit/v2`.
- **Breaking:** `Open(ctx, dsn, ...Option)` replaces `Open(ctx, Config)`.
  `Config` is removed; use `WithDriverName`, `WithLogger`, `WithMaxConns` and
  `WithWarmUp`.
- Pool settings (`max_conns`, `max_conn_lifetime`, `max_conn_idle_time`,
  `warm_pool`) are read from the DSN instead of being set in code. Parameters
  that are not set use `database/sql` defaults.
- Pool parameters are removed from the connection string before it reaches the
  driver, so they are no longer sent to the server as runtime parameters.
- Warm-up opens connections concurrently. A 50-connection pool now warms in
  about the time of one connection.
- Errors from `Open` wrap the underlying cause; use `errors.Is` and `errors.As`.
- Requires Go 1.27.

### Added
- `ParseDSN` returns the pool settings and the connection string without them.
- `PoolConfig` describes the settings read from a DSN.
- `DSNError` reports which parameter is invalid.
- `Redact` masks the password in a DSN for logging. `Open` logs the redacted
  DSN at startup.
- `DefaultMaxConns` (10), replacing `DefaultMaxOpenConns`.

### Removed
- `MustOpen`, deprecated since 1.2.0.
- `Config.SkipWarmUp`. Use `WithWarmUp(false)` or `warm_pool=false`.

## [1.3.0] - 2026-03-02

### Added
- Warm pool on start: `Open` opens `MaxOpenConns` connections before it returns,
  so the first requests after a deploy don't pay for connection setup.
- `Config.SkipWarmUp` to turn warm-up off, for CLIs and migrations.

### Changed
- Idle connection limit now equals `MaxOpenConns`. The `database/sql` default of
  two idle connections closed and reopened connections under steady load.

## [1.2.1] - 2025-11-18

### Fixed
- `Open` closes the pool when the initial ping fails instead of returning it.

## [1.2.0] - 2025-09-09

### Added
- `Config.DriverName` to use a wrapped driver, for example one registered by an
  OpenTelemetry instrumentation package.
- `ErrEmptyDSN`.

### Deprecated
- `MustOpen`. Use `Open` and handle the error.

## [1.1.0] - 2025-06-23

### Added
- `Config.Logger` (`*slog.Logger`) for pool lifecycle messages.

### Changed
- Upgraded to pgx v5.

## [1.0.0] - 2025-04-14

### Added
- `Open` and `Config`: PostgreSQL pool on database/sql with shared defaults
  (10 open connections, 10 minute connection lifetime).
