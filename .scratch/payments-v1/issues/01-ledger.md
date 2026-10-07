# 01 Double-entry ledger in Go

Status: done
Owner: Principal engineer (Fable)
Blocked by: -

## Goal
A multi-asset, append-only, double-entry ledger in `backend/internal/ledger/`. Balances are derived by summing lines, never stored. Every later ticket writes to it.

## Design (from Kuberopay ledger, ADR 0030/0053, re-implemented)
- Tables: `ledger_accounts` (id, owner_type member|platform|connector|chain|fees|reserve, owner_id, asset char(16), kind asset|liability|income|expense, unique(owner_type, owner_id, asset, kind)), `ledger_journals` (id, kind payment|fee|conversion|settlement|refund|adjustment, reference_type, reference_id, idempotency_key unique, posted_at, metadata jsonb), `ledger_lines` (id, journal_id, account_id, amount numeric(38,18) signed, asset, created_at). A journal's lines must sum to zero per asset; enforced in code and by a deferred constraint trigger.
- `Post(ctx, Journal{Kind, Reference, IdempotencyKey, Lines []Line}) (JournalID, error)`: one transaction, rejects unbalanced journals, idempotent on key.
- `Balance(ctx, accountID) (decimal.Decimal, error)` and `Balances(ctx, ownerType, ownerID) map[asset]decimal`.
- `Statement(ctx, ownerType, ownerID, from, to)` returning lines with running balance.
- Rename Payminto's existing `ledger` service to avoid collision; keep its callers working by adapting them to post journals.

## Acceptance
- Unbalanced journal rejected; duplicate idempotency key returns the original id.
- Property test: random sequences of journals keep every asset summing to zero across all accounts.
- Integration test with testcontainer; migration is additive.
- `go test ./internal/ledger/...` and `-tags=integration` pass.

## Plan
`docs/superpowers/plans/2026-10-07-ledger.md`

## Comments

- 2026-10-07 (Principal, Fable): done on branch `ledger`. Commits `601d3dd` (journal types + validation), `ada0679` (models + service), `99b6e31` (property test), `340f385` (Postgres constraints, migration `2026100701_ledger_double_entry`, integration tests), `dcd2277` (LedgerService dual-write + reconciliation + registry wiring). Full report: `.superpowers/ledger-report.md`. Deviations from the ticket text: `asset` is `varchar(16)` not `char(16)` (padding breaks equality); journal kind `transfer` added for sweeps; the existing `service.LedgerService` keeps its name because there is no collision and its callers are untouched. Follow-ups listed in the report.
- 2026-10-07 fix round 1 (Principal, Fable): review `.superpowers/ledger-review.md` addressed in `10109ea` (ledger package: scale/magnitude, sealed journals, TRUNCATE, per-account statements, PostedAt bounds, sorted account creation, property test on Postgres), `bfd1cbe` (callers: assets resolved from blockchain_currencies as chain-qualified codes, gas in the native asset, every post inside the caller's transaction, no second connection), `203bc4f` (boot refuses a missing ledger; ledger_owner role, app role limited to SELECT/INSERT, live privilege check). Report: `.superpowers/ledger-fix-1-report.md`.
