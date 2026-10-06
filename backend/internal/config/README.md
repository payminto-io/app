# config

Owns the typed application configuration loaded from environment variables at startup. Exposes the top-level `Config` struct and its nested groups (`ServerConfig`, `DatabaseConfig`, `RedisConfig`, `BlockchainConfig`, `SecurityConfig`) together with `Load` for parsing env vars and `DatabaseConfig.DSN` for building the Postgres connection string. This package has no runtime dependencies on other Payminto packages — it is imported by `cmd/server`, `internal/database`, and `internal/service` to receive their settings. Validation and defaulting of individual fields lives here so the rest of the codebase can treat config as already-correct.

## Files

- `config.go` — `Config` struct, nested config groups, `Load`, and DSN helpers.
- `config_test.go` — env parsing and default-value tests.

## See also

- `internal/database` — consumes `DatabaseConfig`
- `internal/service/registry.go` — receives the full `*Config`
