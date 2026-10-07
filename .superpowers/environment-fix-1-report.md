# Ticket 13, fix round 1

Branch `environment`, worktree `gateway-wt-environment`, 2026-10-07. Review: `.superpowers/environment-review.md` (1 critical, 4 important, 8 minor). Step 0 merged `ledger` (sealed journals, chain-qualified assets, role ownership) at `79e73fc`; conflicts were in `ledger/service.go`, `constraints.sql`, `auth_service.go`, `registry.go` and the database README, all resolved by re-applying the environment changes on top of the ledger branch's versions.

## Commits

- `79e73fc` Merge branch 'ledger' into environment
- `dd65cb1` C1: sessions bound to the environment (HKDF keys, required audience, per-environment refresh and OTP hashes, live refuses dev `JWT_SECRET`)
- `1f9f02a` I3, M5, M6: the database is the authority (`current_database()`, one-row stamp, gates in `migrate` and `devseed`, quoted DSN, one wired module)
- `5f7124d` I1, M1, M2, M8: environment-scoped idempotency, both account indexes kept, `NOT VALID` checks, `SECURITY DEFINER` relabel function
- `d617f1c` I2: `cmd/migrate adopt-live`, documented in `docs/OPERATIONS.md`
- `075b977` I4, M4, M7: resolved providers judged in live, wiring fails closed, webhook payloads name the environment
- `72d3b65` docs: module README, security addendum, CLAUDE.md, ticket 17

## What changed per finding

### C1 (critical): sessions crossed environments

`environment.DeriveKey(secret, env, purpose)` (HKDF-SHA256, `crypto/hkdf`) and `Environment.Audience()` (`payminto:<env>`). `AuthService.GenerateJWT`/`ValidateJWT` and `JWTTokenService.signAccessToken`/`ValidateAccessToken` share `signSessionClaims`/`parseSessionClaims`: the key is derived per environment, `aud` is set on issue and required on verify (`jwt.WithAudience`), and the audience is read before signature verification so a foreign token is reported as `*sessionEnvironmentError` (`errors.Is` both `ErrSessionEnvironmentMismatch` and `environment.ErrMismatch`). Refresh-token hashes are HMAC-SHA256 keyed with the derived `refresh-token` key, so a test refresh token is `ErrRefreshTokenNotFound` on live. OTP hashes fold the environment into the digest. Both middlewares answer 401 `session_environment_mismatch`. `CheckBoot` in live refuses a `JWT_SECRET` that is a shipped dev default or shorter than 32 bytes (`BootFacts.JWTSecretWeak`). Consequence: sessions and refresh tokens issued before this change stop verifying after the upgrade (users sign in again); API keys are unaffected.

Tests: `service/environment_session_test.go` (access, refresh, legacy path, OTP, unconfigured services), `middleware/environment_test.go` (`JWTAuth` and `JWTOrAPIKey`, both process environments), `environment/port_test.go` (`DeriveKey`, audience), `cmd/server/main_test.go` (live with the compose JWT secret exits non-zero).

### I1 (important): index swap broke old binaries

The migration and `constraints.sql` create `ledger_accounts_env_owner_asset_kind_key` and keep `ledger_accounts_owner_asset_kind_key`. `TestIntegration_OldBinariesStillPostDuringARollingDeploy` runs the pre-ticket `ON CONFLICT (owner_type, owner_id, asset, kind)` statement verbatim after migrations. Ticket 17 (`.scratch/payments-v1/issues/17-drop-legacy-ledger-index.md`) drops the old index later.

### I2 (important): no adoption path

`EnvironmentModule.AdoptLive(ctx, db, confirm)` and `cmd/migrate adopt-live --confirm-adopt-live=<name>`. Guards: live-configured process (full live boot gate), `confirm` equals `current_database()` (trimmed, case-insensitive), live name policy, stamp row present with `environment = test` and `adopted_from IS NULL`. In one transaction: `SELECT accounts, journals FROM ledger_adopt_environment('live')`, a `SECURITY DEFINER` function (`constraints.sql` and the migration) that refuses once any live row exists, pauses only `ledger_accounts_append_only`/`ledger_journals_append_only`, relabels, and re-enables them; the migration moves the function to `ledger_owner` when that role exists and grants `EXECUTE` to the migrator only (`REVOKE ... FROM PUBLIC`); then `api_keys` rows with an empty prefix become live (`sk_test_` keys stay test keys); then the stamp becomes `live` with `adopted_from = test`. `cmd/migrate` skips the pre-action stamp check for this one action. Documented in `docs/OPERATIONS.md`, "Environments".

Tests: `modules/environment_test.go` (`TestAdoptLive_Guards`, sqlite) and `modules/adopt_integration_test.go` (Postgres: populated pre-ticket database, counts, live balances visible, test environment sees none, second run refused, append-only trigger back on, application role gets `permission denied` on the function).

### I3 (important): name and host heuristics

`config.validate` trims `POSTGRES_DATABASE`/`POSTGRES_TEST_DATABASE` and rejects any of host, user, database or test database containing whitespace, quotes, backslashes or `=`; `DatabaseConfig.DSN()` single-quotes and escapes every value. `environment.CheckDatabase(env, name, testName, host, stamp)` normalises names (trim, case-fold); `CheckBoot` runs it on the configured name with no stamp, `EnvironmentModule.VerifyDatabase` runs it on `current_database()` plus the `gateway_environment` stamp right after connecting and before `PrepareSchema`; `VerifySchema` (columns, exact prefix check) runs after; `Stamp` writes the row once. Loopback test databases are allowed only while the stamp says test or the database is new. `cmd/migrate` and `cmd/devseed` run the same three steps; `devseed` additionally refuses any environment but test. `EnforceModeMatch` is unchanged.

Tests: `environment/database_test.go` (fourteen name/host/stamp cases incl. trailing whitespace and case variants), `config_test.go` (unsafe names rejected, trimming, quoted DSN), `modules/environment_test.go` (`VerifyDatabaseAndStamp`, `VerifySchema`), `cmd/server/boot_integration_test.go` (`LiveRefusesAReachableDatabaseStampedTest` now asserts on the reported name and stamp; `TestProcessRefusesALoopbackDatabaseStampedLive`; `FirstBootStampsTheDatabase`). The live boot test now migrates with `ApplyMigrations` and runs the server as a narrowed application role, as the merged ledger branch requires in production.

### I4 (important): mock by default in live

`environment.ResolveProvider(env, configured)` (unset is mock in test, nothing in live), `RequireRealProvider` and `ProcessGuard.RequireProvider(slot, resolved)` (`ErrProvider` for mock or empty in live). `CheckBoot` judges `ResolveProvider` for every configured slot; `MODULES.md` rule 6 now requires every `Wire<Slot>` to resolve and then call `RequireProvider`. Tests in `environment/service_test.go` and `modules/environment_test.go` cover every known slot.

### Minors

- M1: `ledger_journals.environment`; the stored hash is `requestHashFor(env)` with every line carrying the resolved environment; a replay against a row of the other environment is `ErrIdempotencyConflict` ("was posted in the test environment"), never a silent replay; rows with the pre-ticket hash still replay through `legacyRequestHash` (pinned). Tests: `TestIdempotency_IsScopedToTheEnvironment`, `TestIdempotency_RowsWithTheLegacyHashStillReplay`.
- M2: `Balance(accountID)` resolves the environment and filters on it; `AccountBalances` (hence `Balances`) and `Statement` filter too.
- M3: prefix check is `substr(prefix, 1, 8) NOT IN ('sk_<env>_', 'pk_<env>_')`; live rows with an empty prefix are legacy keys adopted through `adopt-live` and stay allowed. Test covers a prefix that merely contains "test".
- M4: `NewRouter` panics without a wired environment, `NewAPIKeyHandler` panics on an invalid one, `ExternalPlatformService` and `AuthService` return `environment.ErrUnconfigured`; `JWTTokenService`, `OTPService` and `WebhookService` likewise. Tests set `Test` explicitly.
- M5: `cmd/devseed` and `cmd/migrate` run `WireEnvironment`, `VerifyDatabase`, `VerifySchema`, `Stamp`.
- M6: `main` wires once and passes the module through `service.WithEnvironmentModule`; the registry wires its own only when none is given (tests).
- M7: `WebhookPayload.Environment`, inside the signed body. Test: `service/webhook_environment_test.go`.
- M8: every new CHECK is `ADD CONSTRAINT ... NOT VALID` then `VALIDATE CONSTRAINT`.

## Test-quality items from the review

- A test process cannot reach live rows: proven structurally now, since a test process refuses a live-stamped database even on loopback (`TestProcessRefusesALoopbackDatabaseStampedLive`) and a live process refuses a test-stamped one by what the database reports.
- JWT cross-environment rejection, rolling-deploy index compatibility, upgrade of a populated pre-ticket database, whitespace and case variants: all listed above.

## Not fixed, with reason

- Payments, deposits, withdrawals, wallets, members and platforms still carry no `environment` column. Under one database per environment, enforced by the stamp, they cannot mix; adding columns to every money table is ticket-11/15 territory and would not change what the stamp already guarantees.
- Existing dashboard sessions and refresh tokens are invalidated by C1's key derivation; there is no migration for a signed token and a forced re-login is the correct outcome.

## Checks

- `cd backend && go build ./... && go vet ./... && go test ./...`: exit 0.
- `make test-integration` (Docker testcontainers): every package `ok` except `internal/paymentlifecycle/postgres`, which hit `go test`'s default 10-minute timeout while 26 packages shared one Docker daemon (it took 443s in the previous round and 462s alone now). It is untouched by this ticket and passes alone; the Makefile target now passes `-timeout 45m` so the gate measures correctness, not daemon contention (`a8f7c7f`).
- Integration packages exercised by this round: `cmd/server` (311s: six subprocess refusals, test and live boots, stamp refusals both ways, first-boot stamp, key refusal over HTTP), `internal/modules` (adoption on Postgres), `internal/ledger` (separation, rolling-deploy arbiter, schema convergence), `internal/database`.
