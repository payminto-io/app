# 01 Double-entry ledger in Go

Status: ready-for-agent
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
