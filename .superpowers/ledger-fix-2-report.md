# Ticket 01 ledger - fix round 2

Branch `ledger`. Re-review: `.superpowers/ledger-rereview.md`. Each finding has a test that failed before its fix; N1 and N2 are reproduced on Postgres.

Commits this round (oldest first):

- `0bfef86` ledger package and migration: N2, N5, N7, N8, N9, N10
- `a337d0e` service, repositories, seeds: N3, N4, N6, N10
- `8a57cd8` migration role handling: N1
- the docs commit that adds this file and the ticket comment

## Important

**N1 - ownership block aborted for a CREATEROLE migrator.** The block now: creates `ledger_owner` when the migrator is superuser or has CREATEROLE; for a non-superuser grants `ledger_owner TO current_user WITH SET TRUE, INHERIT TRUE` (caught and reduced to a NOTICE if the migrator is not an admin of a pre-existing role); gates the transfer on `pg_has_role(current_user, 'ledger_owner', 'SET')`; grants `ledger_owner` `USAGE, CREATE` on the current schema (Postgres requires the new owner to hold CREATE there; only a superuser bypasses it) and gates on `has_schema_privilege`; any gate failing takes the NOTICE path and leaves ownership with the migrator. `ledger_migrator_integration_test.go` builds the schema as a non-superuser that owns the database (`AutoMigrate` + `MigrateExpandSchema` as that role) and runs `ApplyMigrations` as it: with CREATEROLE the tables end up owned by `ledger_owner`, the migrator can still post and can `SET ROLE ledger_owner`; with neither attribute the migration succeeds on the NOTICE path and the tables stay with the migrator. The superuser path is covered by the existing `migratedWithAppRole` tests. `docs/OPERATIONS.md` lists the manual statements including the membership grant and the schema grant.

**N2 - search_path bypass.** All five trigger functions are created through `format()` inside a `DO` block so the installing schema is baked into every table reference (`%I.ledger_lines`), with `SET search_path = pg_catalog, pg_temp` and `pg_catalog.txid_current()` / `pg_catalog.transaction_timestamp()` qualified (`COALESCE` is parser syntax and cannot be shadowed). They stay SECURITY INVOKER: nothing in them needs elevated rights, and the app role already has SELECT on the tables they read. `ValidatePrivileges` additionally refuses an app role that owns any of the five functions, holds CREATE on the database, or holds CREATE on any schema (so nothing it creates can ever be on a search path); TEMP is deliberately not refused because `pg_temp` is pinned last and every reference is qualified, which the doc states. `TestLedgerTriggersCannotBeShadowedThroughSearchPath` reproduces both reviewer bypasses as the narrowed app role granted CREATE on a schema: the zero-amount `ledger_lines` view and the forged `txid_current()`; both now fail to commit, the balance is unchanged, and `PrepareSchema` refuses that role.

**N3 - Polygon has no native row.** `POL` is added to the currencies and a native `POL`/`POLYGON` row to `blockchain_currencies` in both `mainnet` and `testnet` seeds. The resolver returns `Native: ""` when a chain has no native row; `gasLines` fails only when there is gas to book, so gas-free journals post regardless. Tests: SQLite unit tests for a gas-free withdrawal on a chain without a native row (posts) and a sweep with gas on it (fails naming the native row); Postgres `polygon_seeds_integration_test.go` applies the real seed files and runs a Polygon sweep and a Polygon withdrawal through the services, asserting `USDC.POLYGON` custody lines and `POL.POLYGON` gas lines, plus a check over both seed sets that every chain carrying a token row has exactly one native row.

**N4 - sweep confirmation outside the completion transaction.** `SweepService.CompleteConfirmedTransaction` binds the sweep-transaction repository to the transaction (`SweepTransactionRepositoryImpl.WithTx`), marks the tx confirmed, applies the conditional sweep completion and posts the journal, all in one transaction; the confirmer calls it instead of updating status first. `TrackConfirmations` additionally lists confirmed transactions whose sweep is not completed (`ListConfirmedAwaitingCompletion`, a join on `sweeps`) and finishes them without a chain round trip, which also covers rows written before this change. Tests: a resolver that fails once leaves the tx broadcast and the sweep pending with no journal, and the next round confirms, completes and posts four lines; a pre-seeded confirmed tx with a pending sweep is completed on the next round and not again.

## Minor

- **N5** asset codes are `varchar(32)` in the models, migration and CHECK (`^[A-Z0-9_-]+(\.[A-Z0-9_-]+)*$`), validated by the same pattern and length in `Validate`; unit test over accepted and rejected shapes including `USDC.E.AVALANCHE`.
- **N6** `RecordGasFunding` returns an error when the IBT repository cannot bind to the transaction; test with a wrapped repository.
- **N7** `ValidateSchema` joins `pg_trigger` to the three ledger tables in `current_schema()` and requires `tgenabled = 'O'`; functions must exist in `current_schema()`, return `trigger` and carry `search_path=pg_catalog, pg_temp` in `proconfig`. `ValidatePrivileges` checks function ownership. `TestLedgerBootChecksEffectNotNames` disables a trigger, hands a function to the app role, and resets a function's `search_path`; each is refused by name.
- **N8** a key already posted replays (hash match) even when its `PostedAt` is outside the window; only a new journal is bound by it. Test with a one-hour window.
- **N9** journals also carry `posting_started_at` stamped with `transaction_timestamp()`; the seal check requires both the txid and the start time to match. `docs/OPERATIONS.md` says to restore the ledger only through `pg_upgrade` or physical backups.
- **N10** `internal/database/README.md` notes `SET ROLE ledger_owner` for future ledger migrations; `ReconcileStoredBalances` compares `ledger.CurrencyOfAsset(asset) == code` (split on the last dot), with a test that `USDC.E.AVALANCHE` does not count toward `USDC`.
- Cosmetic: the duplicated header comment in the migration is gone with the regeneration.

## Test summary

- `go build ./... && go vet ./... && go vet -tags=integration ./... && go test ./...`: all ok.
- `make test-integration`: all packages ok (database rerun in full after the N1 amend: ok, 189s).
- New tests this round: 3 ledger unit (asset format, late replay, CurrencyOfAsset), 2 database integration for N2 and N7 (both bypasses, effect checks), 2 database integration for N1 (CREATEROLE migrator, plain migrator), 4 service unit (gas-free native, gas without native, unbindable IBT repo, exact reconcile), 2 confirmer unit (failed post retried, stranded confirmed tx completed), 2 service integration (seed native-row invariant for both networks, Polygon sweep and withdrawal end to end).

## Not fixed, with reason

- Nothing from the re-review is left open. The migration file was regenerated in place again (asset width, `posting_started_at`, pinned functions, ownership block); same justification as round 1: unmerged branch, never applied outside test containers.
