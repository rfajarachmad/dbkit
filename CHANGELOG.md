# Changelog

All notable changes to this module are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the module uses
[Semantic Versioning](https://semver.org/).

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
