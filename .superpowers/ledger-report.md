# Ticket 01 - double-entry ledger: report

Branch `ledger`, worktree `gateway-wt-ledger`.
Commits (oldest first): `601d3dd`, `ada0679`, `99b6e31`, `340f385`, `dcd2277`, plus the docs commit that adds this file.

## Files

New package `backend/internal/ledger/`:

- `journal.go` - input types (`Journal`, `Line`, `AccountKey`, `Reference`), enums, sentinel errors, `Validate`, canonical `requestHash`.
- `model.go` - GORM rows `AccountRow`, `JournalRow`, `LineRow`, the `Metadata` jsonb type, `Models()`.
- `service.go` - `Service` with `Post`, `PostIn`, `Transaction`, `EnsureAccount`, `AccountID`, `Balance`, `Balances`, `Statement`, `DB`.
- `reconcile.go` - `Reconcile`, `Expected`, `Drift`, `NaturalBalance`.
- `constraints.sql` / `constraints.go` - Postgres triggers and constraints, `InstallConstraints`, `Migrate` (dev/test).
- Tests: `journal_test.go`, `service_test.go`, `property_test.go`, `reconcile_test.go` (SQLite in-memory, the repo's unit-test convention) and `integration_test.go` (`//go:build integration`, external package `ledger_test`, testcontainer).

Changed:

- `backend/internal/database/migrations/2026100701_ledger_double_entry.up.sql` - the checksummed migration (tables + the same constraint block), registered in `migrations.go`.
- `backend/internal/database/startup.go` - `MigrateExpandSchema` called from `PrepareSchema` in auto-migrate mode.
- `backend/internal/database/testdb.go` - `NewTestDB` also runs `MigrateExpandSchema`.
- `backend/internal/database/migrations_integration_test.go` - expects two applied migrations now.
- `backend/internal/database/README.md` - one line on the expand hook.
- `backend/internal/service/ledger_service.go` - dual-write adapter, `WithJournal` option, `ReconcileStoredBalances`.
- `backend/internal/service/ledger_service_journal_test.go` (+ `_helpers_test.go`) - adapter tests.
- `backend/internal/service/registry.go` - wires `ledger.New(db)` and `currencyAssetResolver`.
- `backend/internal/repository/account_repo_impl.go` - `TxBinder` interface and `WithTx` so the legacy write shares the journal's transaction.

## Schema

`ledger_accounts (id bigserial, owner_type, owner_id, asset varchar(16), kind, created_at)`
unique `(owner_type, owner_id, asset, kind)`, unique `(id, asset)`, CHECKs on the enums and on trimmed non-empty owner_id/asset.

`ledger_journals (id bigserial, kind, reference_type, reference_id, idempotency_key unique, request_hash char(64), posted_at, metadata jsonb, created_at)`
CHECK on kind, key length, hash shape, `jsonb_typeof(metadata) = 'object'`; index on `(reference_type, reference_id)` and `posted_at`.

`ledger_lines (id bigserial, journal_id, account_id, asset, amount numeric(38,18) signed, created_at)`
`CHECK (amount <> 0)`, FK `journal_id -> ledger_journals`, composite FK `(account_id, asset) -> ledger_accounts (id, asset)` so a line can never sit on an account of another asset.

Triggers (all three tables): `BEFORE UPDATE OR DELETE` raise `append-only`.
Deferred constraint triggers: `ledger_lines_balance` (per `(journal_id, asset)` sum must be zero at commit) and `ledger_journals_has_lines` (a journal cannot commit empty).

Signed amounts: positive is a debit, negative a credit. `Balance` returns the raw signed sum; `NaturalBalance` flips liability/income for display and reconciliation.

## Decisions

- **Primary keys are `uint` autoincrement (bigserial)**, matching `PaymintoModel` across the repo. The only non-uint tables (`payment_lifecycle_*`) use text ids chosen for a different reason; the repo's convention is uint, so the ledger follows it.
- **Migration location.** `backend/migrations/` contains only a seed file and nothing executes SQL from it. The checksummed runner (`database.ApplyMigrations`) embeds `backend/internal/database/migrations/*.up.sql` with a `YYYYMMDDNN_name` version, so the ledger migration lives there as `2026100701_ledger_double_entry.up.sql`. A file in `backend/migrations/` would be config set but never reaching the code that reads it.
- **Ledger tables stay out of `currentSchemaManifest` and `database.AutoMigrate`.** `TestCurrentSchemaManifestCoversEveryAutoMigrateTable` requires every AutoMigrate table to be in the manifest, and `ApplyMigrations` validates the manifest before applying anything, so listing the ledger tables there would make the migration unable to run on any database that predates it. Dev/test get the tables through `database.MigrateExpandSchema` (which `PrepareSchema` and `NewTestDB` call), the same stance the `payment_lifecycle_*` tables take. The migration is idempotent (`IF NOT EXISTS`, guarded `ADD CONSTRAINT`, `DROP TRIGGER IF EXISTS`) so a dev database already shaped by AutoMigrate converges when `cmd/migrate` runs; `TestIntegration_SchemaConvergesFromAutoMigrateAndChecksummedMigration` proves that.
- **`asset` is `varchar(16)`, not `char(16)`.** `char` pads with spaces, which breaks equality in Go and the composite FK. Length is still capped at 16.
- **Journal kind `transfer` added** to the ticket's list. Sweeps move value between the platform's own accounts; labelling them `settlement` or `adjustment` would be a false claim, and the honesty rule outranks the enum.
- **Idempotency stores a request hash.** Same key + same canonical payload (line order and decimal formatting ignored) returns the original id; same key + different payload is `ErrIdempotencyConflict`. Concurrency uses `INSERT ... ON CONFLICT DO NOTHING` then a re-read; the unique index serialises racers.
- **Existing `LedgerService` keeps its name and every public signature.** The ticket said "rename"; the brief said "do not change their signatures", and there is no Go collision between `service.LedgerService` and package `ledger`, so renaming would be churn with no benefit. `NewLedgerService(repo)` still compiles for every caller; `WithJournal` is a variadic option.
- **Dual-write order:** journal first, legacy entries second, one transaction. A replayed key returns early so the legacy tables are not duplicated either (that is stricter than before, where a retry inserted again).
- **Legacy mapping:** every old `code` becomes a `platform`-owned account whose `owner_id` is the code (`crypto_assets`, `merchant_balance`, `sweep_gas`, ...). The old API carries no member id, so member-scoped accounts cannot be produced honestly yet.
- **Asset resolution fails closed.** `currencyAssetResolver` looks the code up in `currencies`; an unknown id fails the post and nothing is written in either table set.
- **Reconciliation never writes.** `ledger.Reconcile` compares stored balances with natural derived balances; a missing account derives to zero so an unposted stored balance is reported rather than hidden.

## Test output

Unit (`go build ./... && go vet ./... && go test ./...`): all packages `ok`, no failures.
`internal/ledger`: 17 unit tests incl. the property test (300 random balanced journals over 4 assets and 7 owners with random replays; every asset sums to zero globally and every derived balance equals the replayed sum).
`internal/service`: 4 new adapter tests (every Record* method posts a balanced journal of the right kind, replay writes nothing, unresolvable asset writes nothing, reconcile reports drift without mutating).

Integration (`go test -tags=integration ./internal/ledger/... ./internal/service/... ./internal/repository/...`, Colima Docker): `ok` for all three.
`internal/ledger` integration: schema convergence (AutoMigrate then ApplyMigrations, triggers and composite FK present), UPDATE/DELETE rejected on all three tables, unbalanced commit / empty journal / wrong-asset line rejected at the database, 12 concurrent posts with one key produce one journal and two lines, 18-decimal amounts exact (`1e-18 + 2e-18 + 99999999999999999999.999999999999999997 = 1e20`).
`internal/database` integration also run: `ok` (the manifest test now expects both migrations).

Note for whoever runs these: `testcontainers` skips silently when it cannot find the Docker socket. On this machine Docker is Colima, so `DOCKER_HOST=unix://$HOME/.colima/default/docker.sock` is needed; without it every integration test reports SKIP and the package still says `ok`.

## Pre-existing issues seen, not touched

- `gofmt -l .` lists 31 files outside this change (handlers, models, blockchain adapters, workers). Left alone; a formatting-only commit is a separate decision.
- `WithdrawalProcessingService` passes `withdrawal.BlockchainCurrencyID` where `RecordWithdrawal` expects a currency id (`withdrawal_processing_service.go:212`). The old entries already stored that id in `currency_id`; the new resolver will now resolve it as a currency id too. Caller should map through `BlockchainCurrency.CurrencyID`.
- `WithdrawalProcessingService` logs and continues when the ledger write fails ("reconciler will fix"); with the ledger as source of truth that path should fail the step instead.

## Follow-ups

1. Remove the legacy `assets`/`liabilities`/`revenues`/`expenses` dual-write and `Account.Balance` once every reader is on `internal/ledger` (tickets 02, 05, 13 will add the readers).
2. Give the Record* callers a member id so merchant balances post to `member/<id>` liability accounts; until then `ReconcileStoredBalances` reports every non-zero stored balance as drift, which is the truthful state.
3. Fix the two withdrawal issues above.
4. `Balances` sums across kinds per asset for one owner; if an owner ever holds both an asset and a liability account in one asset, split the map by kind.
5. Wire `ReconcileStoredBalances` into a worker or admin endpoint; today it is callable but not scheduled.
