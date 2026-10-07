# Re-review: branch `fees`, fix round 1

Scope: commits after `1ead367` (`b8ae98c`, `dbd513d`, `39a1349`), excluding the merged ledger/service/database packages.
Inputs: `.superpowers/fees-review.md`, `.superpowers/fees-fix-1-report.md`, `.superpowers/fees-fix-1.diff`, worktree code.
Accepted deviations listed by the caller were not re-flagged.

## Verdict

All original findings (C1, I1-I4, M1-M8) are addressed.
No new critical or important defects found.
Six new minor findings, listed below.

## Checks run

- `cd backend && go build ./... && go vet ./... && go test ./...`: pass (exit 0, all packages ok).
- `make test-integration`: pass (exit 0, 25 packages ok; fees 113 s, ledger 114 s, database 143 s, paymentlifecycle/postgres 171 s).
- Probe integration tests supplied through `go test -overlay` (no worktree files changed): `TestProbe_DefaultScopeNullsOverlap`, `TestProbe_FutureChains`, `TestProbe_ConcurrentSnapshotAndPostFee`, `TestProbe_ClosingGUCBypass`. All ran; results below.

## Original findings

| ID | Status | Evidence |
| --- | --- | --- |
| C1 | Fixed | `NewVersion` closes every lineage row with `effective_to IS NULL OR effective_to > from` to `GREATEST(effective_from, from)` in the insert's transaction (`service.go`). Probe: v1 open, v2 +10d, v3 +20d gives v1/v2/v3 at +1m/+15d/+25d; then v4 at +5d (before both future ones) collapses v2 and v3 to empty windows and closes v1 at +5d; a bounded v5 leaves a gap, v6 reopens. Exactly one active version at every probed instant. Exclusion constraint backs it. |
| I1 | Fixed | `checkDecimal` bounds exponent (+-40), digits (40) and magnitude (<1e20) before any rescaling compare; applied to preview amount (before `resolve`), `PostFee` captured amount and every pricing field incl. slabs. Body capped at 16 KiB. |
| I2 | Fixed | `fee_rules_no_overlap EXCLUDE USING gist` on `method`, `currency`, `coalesce(connector,'')`, `coalesce(card_type,'')`, `coalesce(region,'')`, `tstzrange [)`. The `coalesce` makes NULL scopes compare equal: probe confirms a second default card/USD rule is refused by the service (`OverlapError` naming rule 1) and by direct SQL (`23P01`), while connector- and card_type-specific rules coexist with the default and a duplicate card_type scope is refused. Empty windows never conflict. |
| I3 | Fixed | `Snapshot` reads `member_id` from `payment_requests ... FOR UPDATE`; `PostFee` re-locks, checks the merchant is unchanged, loads the rule by `(id, version)` from the snapshot, recomputes on the captured amount, and posts in the snapshot's currency/asset. `feeJournal` also cross-checks breakdown rule/version against the snapshot. Currency still comes from `Query` because `payment_requests` carries no currency yet (covered by the accepted "Snapshot needs Query.Chain" item). |
| I4 | Fixed | Split into `Snapshot` (per attempt, unique `attempt_id`) and `PostFee` (keyed `fee:attempt:<id>`). Retry on another connector gets its own snapshot; nothing posts for an attempt that never succeeds. |
| M1 | Fixed | Scale checks per field; rows re-read after insert. |
| M2 | Fixed | ISO 4217 list plus configured assets; method/currency class enforced. |
| M3 | Fixed | `computeChecked` returns `ErrFeeExceedsAmount` in preview and `PostFee`. |
| M4 | Fixed, with caveat N5 | Trigger refuses closes before `transaction_timestamp()` unless `fees.closing_version` is set; `NewVersion` only closes at `>= app now` because backdating is refused in `validatePricing`. |
| M5 | Fixed | `lockPayment` filters `deleted_at IS NULL`. |
| M6 | Fixed | `/admin/fee-rules` mounts only with `Session` (`JWTAuth`) and `Admin`; preview stays `JWTOrAPIKey`. Real-RBAC router test covers 401/403/200 paths. |
| M7 | Fixed | `config.FeesConfig` via `config.Load`. |
| M8 | Fixed | Listed gaps covered by new unit and integration tests. |

Additional points the caller asked about:

- Snapshot/PostFee exactly-once: probe ran 8 concurrent `Snapshot` calls for one attempt (all succeed, one row) and 8 concurrent `PostFee` calls (all succeed, one journal, merchant debited 2 once). The payment row lock serialises them and the ledger idempotency key dedupes replays.
- Fee recomputed from the snapshotted rule on the captured amount: confirmed in code and by the existing test that edits the rule after the snapshot.
- Ledger asset naming: `ledgerAsset` produces `CODE` for fiat and `CODE.CHAIN` uppercased, identical to `service.chainAsset` used by the merged ledger resolver (`registry.go:702`); within the ledger's 16-character limit.
- Migration `2026100702` is byte-identical to `internal/fees/schema.sql`; it is not on `main`, so regenerating it in place is safe.

## New findings (all minor)

### N1. A chain code is only shape-checked, so a wrong or unknown chain posts the fee into a phantom asset
- `backend/internal/fees/precision.go:92-113`.
- Probe: `Snapshot` for `crypto`/`USDC` with `Chain: "NOTACHAIN"` succeeds, so `PostFee` would debit `member/<id>/USDC.NOTACHAIN`, an asset no payment credit ever touches; a real-but-wrong chain (`ETH` for a `BASE` payment) splits the merchant's balance across assets the same way.
- The ledger's own resolver derives assets from `blockchain_currencies`; fees should check `(currency, chain)` against that table (or take a `blockchain_currency_id`) inside `Snapshot`, which already runs in a transaction.

### N2. Concurrent reuse of one attempt id on two payments surfaces as a raw unique violation, not `ErrSnapshotConflict`
- `backend/internal/fees/service.go` `Snapshot` (the pre-check is per payment lock, so two different payments race to the insert).
- Probe: one call succeeded, the other returned `duplicate key value violates unique constraint "fee_snapshots_attempt_key" (SQLSTATE 23505)`; `errors.Is(err, ErrSnapshotConflict)` is false. Map `23505` on `fee_snapshots_attempt_key` to `ErrSnapshotConflict`. Note the caller's transaction is aborted either way.

### N3. A `Snapshot` replay with a different query silently returns the original snapshot
- `backend/internal/fees/service.go` `Snapshot` early return.
- Probe: attempt snapshotted under the default rule, replayed with `Connector: "adyen"` (a different active rule), returned the default rule with no error. If the switch ever reuses an attempt id across connectors, `PostFee` prices it under the wrong connector without any signal. Compare the stored scope (or the resolved rule) on replay and return `ErrSnapshotConflict` on mismatch.

### N4. Nothing limits posted fees to one per payment
- `backend/internal/fees/journal.go` (key per attempt).
- Probe: two attempts on one payment, each snapshotted and `PostFee`d with 100, debit the merchant 4 (2 + 2). Per-attempt keys are right for I4, but nothing in fees or the schema stops two attempts of the same payment from both being posted (a switch bug, or a late success callback on a timed-out attempt). If at most one successful attempt per payment is the invariant, enforce it (for example a partial unique index on a posted marker per `payment_request_id`, or a check in `PostFee`); if multiple captures per payment are intended, document it in the README "Payment path".
- Related: `PostFee` on a zero fee posts nothing but still overwrites the legacy `fee_rule_id/version` columns, so the "latest successful attempt wins" value can come from an attempt the existing test describes as failed (accepted: legacy columns mutable).

### N5. The M4 escape hatch is a session GUC any connection can set
- `backend/internal/fees/schema.sql` trigger `fee_rules_append_only`.
- Probe: plain SQL `SET LOCAL fees.closing_version = 'on'; UPDATE fee_rules SET effective_to = effective_from ...` retroactively empties an active rule's window. Fine as an accident guard for app code, but it is not a security boundary; say so in the trigger comment or README, or restrict the bypass to a role (`current_user`) the app role cannot assume.

### N6. Two semantic gaps in the `PostFee` / `Snapshot` contract
- `Snapshot` honours a caller-supplied `Query.At`, so a caller can pin a rule active at another instant; snapshots should resolve at DB or service time (ignore `q.At`).
- For a customer-borne rule the README does not say whether `captured` is the base amount or the gross including the surcharge; computing the fee on the gross charges fee-on-fee and disagrees with the preview. State which one the switch must pass.

### Note (not a finding)
- `fees.Migrate` uses `CREATE TABLE IF NOT EXISTS`, so a developer database that ran the pre-fix schema keeps the old table without `fee_rules_no_overlap` and keeps the dropped `payment_requests_fee_snapshot_fixed` trigger, which would block `PostFee`'s legacy-column update on a second attempt. Recreate local dev databases once; not an issue for any shipped environment since `2026100702` is not on `main`.
