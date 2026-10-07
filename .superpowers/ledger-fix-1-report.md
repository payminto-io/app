# Ticket 01 ledger - fix round 1

Branch `ledger`. Review: `.superpowers/ledger-review.md`. Every item was reproduced by a test that failed before the fix.

Commits this round (oldest first):

- `10109ea` ledger package: I1, I4, M1, M2, M3, M4, M5
- `bfd1cbe` service layer: C1, I2, I6, I7, M6, M7
- `203bc4f` database/config/migrate: I3, I5
- the docs commit that adds this file and the ticket comment

## Critical

**C1 - asset resolved from the wrong table.** `AssetResolver` now takes a `blockchain_currencies` id and reads that row (`backend/internal/service/registry.go`, `blockchainCurrencyAssetResolver`). It emits chain-qualified assets, `USDC.BASE`, `ETH.BASE`, because custody is per chain and a USDC balance on Base is not the same holding as one on Ethereum. The native asset for gas comes from the chain's own `standard = 'native'` row; a chain without one fails the post. Every `Record*` parameter is renamed `blockchainCurrencyID`. The resolver reads through the posting transaction's handle rather than a repository, so it never takes a second pool connection (that was the I7 pattern in a second place, found by the single-connection test below). Tests per caller path in `ledger_callers_test.go`: a fixture whose `currencies` and `blockchain_currencies` ids deliberately differ (and asserts they differ), then `SweepService.MarkCompleted`, `WithdrawalProcessingService.Execute` and `RecordGasFunding` each assert the exact posted lines and assets. `TestBlockchainCurrencyAssetResolver_ResolvesFromTheTableTheIdRefersTo` covers native, token, chain-without-native and unknown id.

## Important

**I1 - scale and magnitude.** `Validate` returns `ErrScale` for any amount that is not equal to itself truncated to 18 places (trailing zeros beyond 18 are not precision and pass) and `ErrMagnitude` for `|amount| >= 1e20`. Both are `ErrInvalid` and run before hashing. No rounding on the caller's behalf.

**I2 - no money event may be lost. Choice: single transaction everywhere.** Reasoning: a durable "ledger pending" marker is a second write model with its own failure modes and a worker nobody owns yet; joining the caller's transaction costs one parameter and removes the window entirely.
- Sweep: `MarkCompleted(ctx, ...)` wraps the conditional status UPDATE and `RecordSweepIn` in one `db.Transaction`; a failed post rolls `completed` back (asserted in `TestSweepService_MarkCompleted_LedgerFailureReturnsError` and the caller-path test).
- Withdrawal: `markSentWithLedger` runs `MarkSent` (through a new `WithdrawalRepositoryImpl.WithTx`) and `RecordWithdrawalIn` in one transaction opened by `LedgerService.InTransaction`; a failed post leaves the row `initiated` and `Execute` returns the error instead of logging it (asserted). Pre-existing and unchanged: a withdrawal stuck in `initiated` has no automatic retry; that was already true for a `MarkSent` failure and needs its own ticket.
- Gas funding: `RecordGasFeeIn(ctx, tx, ...)` joins `RecordGasFunding`'s transaction and the IBT repository is bound to it too (it previously wrote outside the transaction it was nominally inside).
- `LedgerService` keeps every old signature; each `Record*` is now a wrapper over `Record*In(ctx, tx, ...)` with `context.Background()` and its own transaction.

**I3 - boot without ledger tables.** `PrepareSchema` now calls `ledger.ValidateSchema` (tables; on Postgres also the ten triggers and five functions) in every environment and mode, `ledger.ValidateMigrationRecorded` (migration `2026100701` applied and not dirty) in validate mode, and `ledger.ValidatePrivileges` in staging and production. These run after the manifest check and are not part of `ApplyMigrations`, so the runner can still migrate a database that predates the ledger. Tests: SQLite unit test for the missing-table case in both environments; Postgres tests for the missing migration record and the live privilege rules.

**I4 - TRUNCATE and late lines.** Statement-level `BEFORE TRUNCATE` triggers on all three tables. `ledger_journals.posting_txid` is stamped by a `BEFORE INSERT` trigger with `txid_current()` (the client cannot forge it; the test asserts no journal carries 0), and a `BEFORE INSERT` trigger on `ledger_lines` refuses a line whose journal was posted by another transaction ("sealed"). `txid_current()` is the top-level id, so savepoints inside the posting transaction still match. Tests: four TRUNCATE shapes rejected; a balanced +1000/-1000 pair appended to a committed journal rejected with the balance unchanged. The migration file was regenerated in place; it is on an unmerged branch and has never been applied outside test containers, so the checksum change is safe. The convergence test still proves AutoMigrate-then-migration converges.

**I5 - application role owns the tables.** Migration `2026100701` ends with a block that, when the migrator is superuser or has `CREATEROLE`, creates `ledger_owner NOLOGIN`, moves the three tables (and their sequences) and five trigger functions to it, and grants the migrator `SELECT, INSERT` plus sequence use. Without that privilege it raises a NOTICE and leaves ownership alone; `docs/OPERATIONS.md` "Ledger roles" documents the manual statements. `cmd/migrate --ledger-app-role <role>` (or `POSTGRES_LEDGER_APP_ROLE`) runs `ledger.GrantAppRole`: `REVOKE ALL`, `GRANT SELECT, INSERT`, sequence `USAGE, SELECT`. Boot check in staging/production (`ledger.ValidatePrivileges`): not superuser, not owner or member of the owner, no UPDATE/DELETE/TRUNCATE/TRIGGER, can SELECT and INSERT. Integration tests (`ledger_roles_integration_test.go`) apply the migration as the container superuser, create an app role with ordinary DML on every table, narrow it, reconnect as it (`ConnectTestDBAs`), and prove: ownership of tables and functions is `ledger_owner`; the app role can post and read balances; UPDATE, DELETE, TRUNCATE are "permission denied"; `ALTER TABLE ... DISABLE TRIGGER` and `DROP TRIGGER` are "must be owner"; replacing the balance function is refused; `PrepareSchema` accepts the app role and refuses the owner in staging and production while development accepts both. The existing production validate test now runs as the app role, which is what production will look like.

**I6 - gas in the token's asset.** Each journal is built explicitly (no longer derived from the legacy entries): token custody moves only by the token amount, and gas is its own balanced pair in the native asset (`sweep_gas`/`withdrawal_gas`/`gas_fee`/`deployment_gas` expense against `crypto_assets` in `ETH.BASE`). The -106 USDC assertion is replaced by per-asset assertions: `crypto_assets USDC.BASE = -100`, `crypto_assets ETH.BASE = -12`, every asset sums to zero. The legacy entry tables still carry the old mixed rows; they are the dual-write that is scheduled for removal, not a source of truth.

**I7 - second connection inside the caller's transaction.** Covered by I2's `RecordGasFeeIn`. `TestIntegration_RecordGasFundingNeedsOnlyTheCallerConnection` sets `SetMaxOpenConns(1)` on a Postgres pool and requires the call to finish; before the fix it hung on the second connection (and it also caught the resolver taking a second connection, fixed under C1).

## Minor

- **M1** `RunPropertyTest(t, s, maxScale)` generates amounts with 0..maxScale decimals, corrupts one journal in four into an unbalanced, 19-decimal or zero-amount journal and asserts the typed rejection and that nothing was written; the SQLite unit run uses scale 0 (SQLite stores numeric as float), the Postgres run under the integration tag uses scale 18.
- **M2** Accounts are resolved or created in sorted key order before the journal row. `TestIntegration_ConcurrentFirstUseOfAccountsInOppositeOrder` posts 40 pairs of journals touching two fresh accounts in opposite order concurrently.
- **M3** `PostedAt`, when set, is in the canonical payload (UTC, microsecond precision); a replay with a different `PostedAt` is `ErrIdempotencyConflict`.
- **M4** `PostedAt` is the accounting date and is bounded: default 7 days back, 5 minutes ahead (`WithPostedAtWindow`), `ErrPostedAt` otherwise. Statement order stays `posted_at, journal id, line id`.
- **M5** `Statement` running balance is per account, with `RunningNatural` alongside the signed value. `Balances` returns natural balances per asset and returns `ErrMixedKinds` when an owner holds two kinds in one asset rather than netting them; `AccountBalances` lists every account with signed and natural values. `ReconcileStoredBalances` now takes a currency-code resolver and sums the member's liability accounts across every chain instance of the currency (stored balances are chain-agnostic).
- **M6** `ctx` threads through the `Record*In` variants and `InTransaction`; the legacy signatures use `context.Background()` and say so.
- **M7** `boundRepo` returns an error when the account repository is not a `TxBinder`; `TestLedgerService_RefusesARepositoryItCannotBindToTheTransaction` wraps the repository and asserts refusal with nothing written.
- **M8** Not changed, per ruling: `service.LedgerService` keeps its name.

## Test summary

- `go build ./... && go vet ./... && go test ./...`: all packages ok.
- `make test-integration`: all packages ok (database 302s, ledger 255s, paymentlifecycle/postgres 332s, service 64s); the one assertion that needed adjusting was a wording difference (PG16 refuses the function replacement with "permission denied for schema public" before reaching "must be owner"), and both wordings are now accepted as refusals.
- New tests this round: 9 ledger unit, 4 ledger integration (property on Postgres, truncate, sealed journal, opposite-order concurrency), 6 service unit (resolver, sweep, withdrawal, gas funding, unbindable repo, reworked dual-write and reconcile), 1 service integration (single connection), 1 database unit, 5 database integration (ownership, app role capabilities, live accept/refuse, migration record, production validate as app role).

## Not fixed, with reason

- Withdrawals that fail after `initiated` (now including a failed ledger post) have no automatic retry. That is pre-existing behaviour for any `MarkSent` failure and belongs to the withdrawal state machine, not the ledger; noted for a follow-up ticket.
- The legacy `assets`/`liabilities`/`revenues`/`expenses` rows still book gas in the token's `currency_id`, as they always did. They are the dual-write slated for removal once ticket 02/05/13 readers are on `internal/ledger`; correcting a table that is being deleted would be churn.
