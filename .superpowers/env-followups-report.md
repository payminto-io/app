# Ticket 13 follow-ups (final review M1-M6)

Worktree `gateway-wt-env-followups`, branch `env-followups` from `main` at `391e224`, 2026-10-07. Review: `gateway-wt-environment/.superpowers/environment-final-review.md`.

## Commit

- `231095c` adoption checks the network mode; no write before every boot check passes; every merchant and money table counts as data; DSN message names PGAPPNAME/PGOPTIONS/PGSERVICE; docs and import grouping

## Per finding

- M1: `EnvironmentModule` carries the configured network type (`BootFacts.NetworkType`). `AdoptLive` reads `configurations.mode` inside its transaction; the row, or the configured type when no row exists, must be `mainnet`, else it refuses ("live money is mainnet; this database's network mode is ..."). `AdoptTest` refuses when the row is `mainnet`. Tests: `TestAdoption_ChecksTheNetworkMode` (sqlite); on Postgres `TestIntegration_AdoptionRefusesTheWrongNetworkMode` (adopt-test refused on mainnet, accepted on testnet, no stamp left by the refusal) and `TestIntegration_AdoptLiveRefusesATestnetDatabaseWithALiveName` (a `payminto_prod` database with `mode = testnet` and merchant data: adopt-live refused naming the network mode, no stamp written; accepted once the row says mainnet).
- M2: `EnvironmentModule.Finalize(ctx, db, networkType)` is the first write and the last boot step: in one transaction it re-reads the stamp, reads the mode row, refuses a mismatch, refuses an unstamped populated database, and only then writes the stamp and (when absent) the mode row. `cmd/server` now runs `config.CheckModeMatch` (read-only) and `Finalize` after the registry and before the vault unlock and RBAC seeding; `cmd/migrate up` does the same after `ApplyMigrations` and `VerifySchema`. `Stamp` is `Finalize` without a mode write (used by `devseed`). Tests: `TestFinalize_WritesStampAndModeTogetherOnlyAfterEveryCheck` (refusals leave neither row; the happy path writes both; idempotent) and the boot-level `TestIntegration_LiveBootRefusesUnstampedDataUntilAdopted` now asserts the refused real server left no mode row and no stamp.
- M3: `dataTables` is one list of fourteen: `members`, `external_platforms`, `api_keys`, `payment_requests`, `deposits`, `deposit_addresses`, `withdrawals`, `sweeps`, `wallets`, `address_pools`, `secrets_vaults`, `ledger_accounts`, `ledger_journals`, `fee_rules`; absent tables are skipped. `DataTables()` exposes it and `TestStamp_EveryDataTableCountsAsData` proves a single row in each one refuses the stamp naming the table, and pins the list.
- M4: `VerifiedDSN`'s refusal names `PGAPPNAME`, `PGOPTIONS` and `PGSERVICE`; `docs/OPERATIONS.md` "Environments" and `internal/database/README.md` document it.
- M5: OPERATIONS.md says to rerun `cmd/migrate up` after adoption to narrow the application role, documents the mode checks, and has the blank line before "Fee rules". The database README's duplicated `testdb.go` and `migrations/` bullets are merged.
- M6: internal imports moved into the second group in `router_analytics_test.go` and `routes_fees_test.go`.

## Checks

- `cd backend && go build ./... && go vet ./...` and unit tests for `internal/modules`, `internal/api/...`, `internal/database`, `internal/config`, `internal/environment`, `cmd/...`: exit 0 (two older sqlite fixtures that build a live module by struct literal gained `networkType: "mainnet"`, since the real wiring always sets it from the live gate).
- A full disk interrupted the first build (`ld: No space left on device`); `go clean -cache` and Docker prune freed it. Nothing in any worktree was touched.
- `DOCKER_HOST=... go test -tags=integration -count=1 -timeout 45m ./internal/environment/... ./internal/database/... ./internal/modules/... ./cmd/...`: all ok (`database` 122s, `cmd/server` 48s, `modules` 21s on the rerun after the two older Postgres adoption fixtures, which build a live module by struct literal, gained `networkType: "mainnet"`; the real wiring always sets it from the live gate).
