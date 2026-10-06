# database

Owns database connectivity and schema migration for the Payminto backend. Exposes `Connect` (opens a GORM/Postgres pool with configured limits), `AutoMigrate` (runs GORM AutoMigrate across every Payminto model, keeping dev/test schemas in sync), and — under the `integration` build tag — `NewTestDB`, which spins up a Postgres 16 testcontainer for integration tests and returns a connected `*gorm.DB` plus cleanup. Depends on `internal/config`, `internal/models`, GORM's Postgres driver, and `testcontainers-go`. All other packages take a `*gorm.DB` as a constructor argument rather than calling into this package directly.

## Files

- `database.go` — `Connect` and `AutoMigrate` for runtime use.
- `testdb.go` — `NewTestDB` testcontainer helper, gated by `//go:build integration`.

## See also

- `internal/config` — `DatabaseConfig` consumed by `Connect`
- `internal/models` — list of models migrated here
- `migrations/` — hand-written SQL migrations run outside GORM
