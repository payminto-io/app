# database

Owns database connectivity and schema migration for the Payminto backend. Exposes `Connect` (opens a GORM/Postgres pool with configured limits), `AutoMigrate` (runs GORM AutoMigrate across every Payminto model, keeping dev/test schemas in sync), and — under the `integration` build tag — `NewTestDB`, which spins up a Postgres 16 testcontainer for integration tests and returns a connected `*gorm.DB` plus cleanup. Depends on `internal/config`, `internal/models`, GORM's Postgres driver, and `testcontainers-go`. All other packages take a `*gorm.DB` as a constructor argument rather than calling into this package directly.

## Files

- `database.go` — `Connect` and `AutoMigrate` for runtime use.
- `startup.go` — `PrepareSchema` and `MigrateExpandSchema` (migration-managed tables such as `internal/ledger` and `internal/paymentswitch`, kept out of the manifest). Boot also validates the ledger schema and, in live environments, that the connected role cannot rewrite it.
- `testdb.go` (integration tag) — `NewTestDB`, `NewEmptyTestDB` and `NewTestDBConfig` (a container's connection config, for tests that boot a whole process against it).
- `testdb.go` — `NewTestDB` testcontainer helper, gated by `//go:build integration`.

## See also

- `internal/config` — `DatabaseConfig` consumed by `Connect`
- `internal/models` — list of models migrated here
- `migrations/` — hand-written SQL migrations run outside GORM; `2026100705_environment_isolation` adds `environment` to `api_keys` and `ledger_accounts` (columns stay out of the manifest so `ApplyMigrations` can run on a database that predates them)
- `migrations/` — hand-written SQL migrations run outside GORM. After `2026100701` the ledger tables and trigger functions belong to `ledger_owner`; a later migration that alters them must `SET ROLE ledger_owner` first.
