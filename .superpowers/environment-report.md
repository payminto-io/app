# Ticket 13: live and test environment isolation

Branch `environment`, worktree `gateway-wt-environment`. Owner role: Principal engineer (Fable). Date: 2026-10-07.

## What shipped

### Core module `backend/internal/environment/`

- `port.go`: `Environment` (`test` | `live`), `Parse`, `WithContext` / `FromContext`, the `Guard` interface, `MismatchError` (satisfies `errors.Is(err, ErrMismatch)`), API key prefix helpers (`sk_test_`, `sk_live_`, `pk_test_`, `pk_live_`; `KeyEnvironment` reads a raw key's prefix and reports `ok=false` for legacy `pm_` keys).
- `service.go`: `ProcessGuard` (`Require(ctx, env)` checks the explicit environment, then the context, against the process), `Resolve` (explicit, then context, then fallback), `BootFacts` + `CheckBoot` (the boot gate; returns a `BootError` listing every refusal at once).
- The package reads no configuration; `config.Config.BootFacts()` projects the loaded config onto it.

### Process mode and boot gate

- `GATEWAY_ENVIRONMENT=test|live`, default `test`, canonicalised and validated in `config.Load`.
- `cmd/server/main.go` calls `modules.WireEnvironment` right after `config.Load`, before a database is opened, so every refusal exits non-zero with `environment: refusing to boot as live: ...` and the full list of reasons.
- Live refuses: database equal to `POSTGRES_TEST_DATABASE` (default `payminto_test`) or ending in `_test`; `DEV_KEYSTORE=true` or `AES_KEY` set (development keystore / local vault master key); custody enabled without a strong `VAULT_PASSPHRASE` (vault dev mode); any `<SLOT>_PROVIDER=mock` (`CUSTODY_PROVIDER`, `CONNECTORS_PROVIDER`, `CONVERSION_PROVIDER`, `PAYOUT_PROVIDER`, `KYC_PROVIDER`, `FRAUD_PROVIDER`, `BRIDGE_PROVIDER`); `SERVER` outside staging/production (that is the switch the existing config keys secure cookies, HSTS and secret strength on); `POSTGRES_SSL_MODE` other than `verify-full` except the loopback opt-out the config already models; `BLOCKCHAIN_NETWORK_TYPE` other than mainnet (ADR 0033: live money is mainnet custody).
- Test refuses a database that is neither `*_test` nor on a loopback host, and a mainnet network.
- After `PrepareSchema`, `EnvironmentModule.VerifyDatabase` refuses a schema missing the `environment` columns (names migration `2026100705`) and any `api_keys` row whose visible prefix disagrees with its environment column (the "API key prefix mismatch" refusal).

### API keys

- `models.APIKey` gains `environment` (default `test`) and `prefix` (default `''`), additive. Existing keys are test keys and keep working on a test process.
- `service.GenerateAPIKeyFor(env)` issues `sk_<env>_<64 hex>`; `GenerateAPIKey()` now returns a test key for existing callers. `NewAPIKeyRow` builds the stored row. The merchant `POST /api/v1/api-keys`, platform `RegenerateAPIKey`/`Create`, and `cmd/devseed` all issue through it in the process environment.
- `AuthService.ValidateAPIKey` refuses the other environment by prefix before the lookup and by row after it; `APIKeyAuth` and `JWTOrAPIKey` answer `401 {"error": ..., "code": "api_key_environment_mismatch"}` (`middleware.CodeAPIKeyEnvironmentMismatch`). Other failures are unchanged.
- `middleware.Environment` tags every request context with the process environment (registered globally in `NewRouter`).
- The API key list response now carries `environment` and shows the stored visible prefix instead of the first characters of the hash.

### Ledger

- `AccountKey.Environment` (zero value resolves to the context, then the service default). `AccountRow.Environment` joins the uniqueness key (`ledger_accounts_env_owner_asset_kind_key`); the old four-column index is dropped by the idempotent `constraints.sql` and by the migration.
- `ledger.New(db, ledger.WithEnvironment(env), ledger.WithGuard(guard))`; a bare `ledger.New(db)` serves test. `PostIn`, `EnsureAccount`, `AccountID`, `Balances`, `Statement` (and `Reconcile` through them) ask the guard first; a journal whose lines name two environments is `ErrInvalid`.
- The request hash only includes an environment when a key names one explicitly, so journals posted before this change still replay (pinned in `TestRequestHash_UnchangedForKeysWithoutEnvironment`).
- `service.LedgerService` is untouched except for its wiring in the registry.

### Migration and schema manifest

`backend/internal/database/migrations/2026100705_environment_isolation.up.sql`: additive, idempotent, registered in `migrationManifest`. As with the ledger migration, the new columns stay out of `currentSchemaManifest` so `ApplyMigrations` can run on a database that predates them; `VerifyDatabase` is the runtime check that they exist. Dev/test converge through AutoMigrate plus `constraints.sql`; `TestIntegration_SchemaConvergesFromAutoMigrateAndChecksummedMigration` proves both paths end with exactly the environment-scoped index.

### Wiring and route

- `backend/internal/modules/deps.go` (`Deps{Config, DB}`) and `modules/environment.go` (`WireEnvironment`, `VerifyDatabase`).
- `service/registry.go` gains the one line `r.environmentModule, err = modules.WireEnvironment(...)`, exposes `EnvironmentModule()`, and passes the environment to `AuthService`, `ExternalPlatformService` and the ledger journal.
- `backend/internal/api/routes_environment.go`: `GET /v2/environment` returns `{"environment": "test"}` behind `JWTOrAPIKey`.
- Both compose stacks declare `GATEWAY_ENVIRONMENT: test` and default the database to `payminto_test` (they reach Postgres by service name, so the loopback allowance does not apply).

## Tests

- `internal/environment`: parse, prefixes, context, guard, boot policy (every live refusal, all reasons at once, test rules).
- `internal/config`: defaults, canonical environment, slot providers, `BootFacts` projection.
- `internal/ledger`: sqlite unit tests for per-environment accounts and balances, zero-value resolution, guard refusal, mixed-environment validation, pinned pre-change hash; Postgres integration test `TestIntegration_EnvironmentsNeverShareAccountsOrBalances` (two guarded services, 18-decimal amounts, cross-environment writes and reads refused, CHECK constraint on unknown environment).
- `internal/api/middleware`: both middlewares, both process environments, both key environments plus a legacy `pm_` key; prefix mismatch refused without a lookup; request context tagging.
- `internal/modules`: live refuses dev keystore and mock; test defaults; `VerifyDatabase` missing column and prefix mismatch.
- `cmd/server/main_test.go` (helper subprocess, no Docker): live with dev keystore, with `AES_KEY`, with a mock provider, against a `_test` database; test against a remote non-`_test` database; unknown environment. Each exits non-zero with the specific message and is proven to come from the environment gate, not the database.
- `cmd/server/boot_integration_test.go` (Docker): a test process boots against `payminto_test`; a live process boots against `payminto_live` in validate mode and answers `GET /v2/environment` 200 for an `sk_live_` key and 401 `api_key_environment_mismatch` for an `sk_test_` key and a legacy key; live refuses the reachable test database.
- `database.NewTestDBConfig` was added (integration tag) so a whole process can be booted against a container; `NewEmptyTestDB` now builds on it. The container password changed to a deployment-strength value for the live boot.

Checks run from the worktree: `cd backend && go build ./... && go vet ./... && go test ./...` and `make test-integration`. See the ticket comment for the final result.

## Deviations and judgement calls

- "Development keystore" has no existing switch in this code base, so it is `DEV_KEYSTORE=true` (new, read by config, refused in live; ticket 15's mocks can set it) plus `AES_KEY` (the legacy local master key the config still models). "SecretsVault in dev mode" is custody enabled with a non-strong passphrase; config already rejects that in every profile, so live cannot reach it, but the fact is still projected and checked.
- Live additionally requires `BLOCKCHAIN_NETWORK_TYPE=mainnet` and test refuses mainnet. Not in the fixed decisions, but it is what ADR 0033 says and it is cheap. Existing deployments default to `test`, so a production-profile testnet deployment keeps booting unless it opts into `live`.
- Newly issued keys are `sk_test_...` instead of `pm_...`. The one existing test asserting `pm_` was updated. The hash, lookup and header handling are unchanged, so existing integrations keep working.
- `pk_` publishable keys are defined (prefix helpers, validation) but nothing issues them yet; there is no publishable-key surface in this code base.
- The migration drops `ledger_accounts_owner_asset_kind_key` and creates the five-column index. That is the only non-additive DDL, and it is required by the decision to put the environment in the uniqueness key. The append-only triggers are row-level and do not fire on DDL.

## Follow-ups

- Frontend: the dashboard's test/live indicator should read `GET /v2/environment` (not done here; frontend engineer).
- `docker/postgres/init.sql` creates a `payminto` database and a `payminto` user that the compose `POSTGRES_USER` already creates; it predates this ticket and was left alone.
- Future money modules (switch, settlement, custody) must take the guard from `modules.EnvironmentModule` and call `Require` before writing; the ledger already does, so anything that posts through it is covered.
- Settlement eligibility by merchant approval state (`test_only` vs `live`, ADR 0033) belongs to ticket 11, which can now key it on `environment.FromContext`.
