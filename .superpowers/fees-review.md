# Review: branch `fees` (ticket 02, versioned fee rules and preview)

Range reviewed: `b7f609e..1ead367`.
Reviewer checks: `go build ./...`, `go vet ./...`, `go test ./...` pass.
`make test-integration` passes.
Accepted deviations (admin guard argument, `FEES_OPERATOR_PLATFORM_ID`, no caller of `ApplyToPayment`, `min_fee`/`max_fee`, `tax_percent`, `(lower, upper]` slabs, UPI-only surcharge default) were not re-flagged.

## Verdicts

- Spec: **partially met.** Every acceptance bullet has code and tests, but "editing creates version n+1 and closes n" leaves version n-1 open when n was a scheduled (future) version, so a normal edit sequence makes `Resolve` fail for the whole scope (C1, reproduced).
- Quality: **good structure, not mergeable as is.** Clean port, pure core, strong DB triggers and typed errors; one critical versioning bug, an unauthenticated-cost DoS on the merchant preview route, and an `ApplyToPayment` contract that trusts caller-supplied money.

Counts: Critical 1, Important 4, Minor 8.

## Critical

### C1. Superseding a scheduled version leaves the previous live version open: two versions of one lineage overlap and the scope stops pricing

- Where: `backend/internal/fees/service.go:110-118` (close logic in `NewVersion`).
- `NewVersion` closes only the row it locked (version n).
  When n has `effective_from` in the future, `closeAt = max(new.from, n.from) = n.from`, so n gets an empty window, but n-1 was closed at `n.from` and stays active until then.
  The new version n+1 starts now, so n-1 and n+1 are both active over `[now, n.from)`, same scope, same specificity.
- Reproduced (integration, overlay test, no tree changes): v1 open; v2 effective in 10 days; v3 created now with default `effective_from`.
  `Resolve` and `Preview` for `card/USD` now return `fees: ambiguous configuration, rules 1,3 tie at specificity 0` (HTTP 409) for every payment in that scope until v2's start date.
- Failure scenario: an operator schedules a price change for next month, then corrects it today. From that moment every card/USD payment fails to price for a month (once the switch calls `Resolve`, payments fail outright).
- Fix: in the same transaction, close every row of the lineage whose window extends past `new.from`: `UPDATE fee_rules SET effective_to = GREATEST(effective_from, $from) WHERE lineage_id = $l AND (effective_to IS NULL OR effective_to > $from)`.
  Back it with a DB guarantee (see I2's exclusion constraint) so no future path can reintroduce overlap.
  Add the integration test above (v1 open, v2 future, v3 now: exactly one active version at every instant).

## Important

### I1. Preview (any merchant, API key) can be made to burn seconds-to-minutes of CPU and GBs of memory with one small body

- Where: `backend/internal/fees/validate.go:239-244` (`req.Amount.Round(places)` and the positivity check run on the raw parsed decimal); `backend/internal/api/routes_fees.go:80,218-232`.
- `shopspring/decimal` accepts exponents up to int32 from JSON (`"amount":"1e100000000"`), and `Round`/`Cmp` rescale to the common exponent by materialising `10^exp` as a big.Int.
  Measured on this machine: `1e1000000` takes 108 ms, `1e10000000` takes 5.9 s, and each further order of magnitude is roughly 50-100x more with memory to match. A 14-byte body ties up a core per request; a few concurrent ones exhaust the server. `1e-100000000` does the same in the other direction.
- The 1 MiB body limit does not help; the cost is in the exponent, not the length.
- Fix: before any arithmetic or comparison, bound the representation directly: reject if `amount.Exponent() < -18 || amount.Exponent() > 0 && amount.NumDigits()+int(amount.Exponent()) > 20` (or simply reject when `len(coefficient digits) + |exponent|` exceeds ~40), then do the precision check. Apply the same guard in `validatePricing` to `percent`, `flat`, `min_fee`, `max_fee`, `tax_percent` and every slab `up_to`/`percent`/`flat`, because a stored slab `up_to` of `1e100000000` makes every subsequent preview in that scope pay the rescale cost inside `slabFor`. Add a unit test that a huge-exponent amount is rejected in well under a millisecond.

### I2. No write-time guard against overlapping same-scope rules: one `POST /admin/fee-rules` takes a scope offline

- Where: `backend/internal/fees/service.go:74-84` (`CreateRule`), `backend/internal/fees/schema.sql:4-36` (no exclusion constraint).
- Creating a second lineage for a scope that already has an active rule succeeds, and from then on every `Resolve` for that scope returns `AmbiguousRuleError`. Two operators creating the "same" rule concurrently, or an operator who meant "edit" and used "create", has the same effect.
  The ticket asks Resolve to pick the most specific active rule; it does not ask for overlap to be legal. The report calls this "as specified"; it is not in the ticket.
- Fix: make overlap impossible in the database: `CREATE EXTENSION IF NOT EXISTS btree_gist;` plus `EXCLUDE USING gist (method WITH =, currency WITH =, coalesce(connector,'') WITH =, coalesce(card_type,'') WITH =, coalesce(region,'') WITH =, tstzrange(effective_from, effective_to, '[)') WITH &&)` (empty ranges never conflict, so superseded zero-width versions are fine). Map the exclusion violation (`23P01`) to a 409 `overlapping_fee_rule` naming the existing rule. This also closes C1 at the DB layer and makes `AmbiguousRuleError` a can't-happen.

### I3. `ApplyToPayment` trusts caller-supplied merchant, currency and amounts instead of deriving them from the payment and the snapshotted rule

- Where: `backend/internal/fees/service.go:173-207`, `backend/internal/fees/journal.go:16-62`, `backend/internal/fees/port.go:163-168`.
- The snapshot writes `(rule_id, version)` from `pf.Breakdown`, but the journal posts `pf.Breakdown.Fee`/`Tax` in `pf.Breakdown.Currency` against `pf.MerchantID`, none of which is checked against anything:
  - `payment_requests.member_id` exists and is ignored; a caller bug (or a string formatted as `member:5` instead of `5`) debits the wrong or a phantom `member/<id>` account.
  - The fee is not recomputed from the snapshotted rule, so a breakdown computed under rule A (or under a stale bearer override from preview) can be posted with rule B's snapshot; the ledger then disagrees with the snapshot it claims to derive from.
  - `Breakdown.Currency` is not compared with the rule's currency or the payment's.
  This conflicts with CLAUDE.md "every number shown is a ledger line" and "fee rule snapshot on every payment": the snapshot no longer explains the line.
- Fix: change the port to `ApplyToPayment(ctx, tx, paymentRequestID uint, q Query, amount decimal.Decimal)` (or take the rule id/version plus amount) and inside it: read `member_id` (and currency/amount once the payment carries them) from `payment_requests ... FOR UPDATE`, load the rule by `(id, version)`, check currency, call `Compute`, then snapshot and post. Keep `MerchantID` typed as `uint` if it stays in the API.

### I4. Snapshot and fee posting are fused, keyed per payment request, and run at creation time: wrong for retries across connectors and for unpaid payments

- Where: `backend/internal/fees/port.go:178-179` (doc: "the snapshot, the fee journal and the payment commit together"), `backend/internal/fees/service.go:180-200`, `backend/internal/fees/journal.go:36` (key `fee:payment_request:<id>`).
- The report says the switch (05) should call `ApplyToPayment` per attempt. With the snapshot fixed per payment request and the idempotency key per payment request:
  - Attempt 1 on connector A snapshots rule A; attempt 2 cascades to connector B (different rule) and fails with `ErrSnapshotConflict`, so a routing fallback cannot complete.
  - The fee is debited from the merchant when the payment is created, before any money arrives; an expired or cancelled payment leaves a posted fee with no reversal path, and a partially/over-filled crypto payment is charged on the hypothetical amount.
- Fix: split into `Snapshot` (at attempt creation, stored per attempt or on the payment once the winning attempt is known) and `PostFee` (at capture/confirmation, on the captured amount, keyed by the capture event). Settle this now in the port, before ticket 05 builds on the fused shape; record it in the README.

## Minor

### M1. Values with more precision than the column are silently rounded by Postgres while the API echoes the unrounded input
- `backend/internal/fees/validate.go:112-117,150-155,181-187`; `schema.sql:13-19`.
- `percent: "2.9999999"` passes validation, Postgres stores `3.000000` (numeric(9,6)), and `CreateRule` returns the in-memory row, so the 201 response says 2.9999999 while every later read and every fee uses 3. Same for `tax_percent`, and `flat` beyond 18 places; `flat` with more than 20 integer digits is a 500.
- Fix: reject more than 6 decimals on percentages and more than the currency's minor units on `flat` and slab values (as is already done for `min_fee`/`max_fee`), and re-read the row after insert (`RETURNING *`).

### M2. `MinorUnits` treats any unknown three-letter code as a two-decimal fiat currency
- `backend/internal/fees/compute.go:21,31-33`.
- `XRP`, `TON`, `ADA`, `DOT`, `ARB` (or a typo like `USF`) are accepted as fiat with 2 places, so fees in those assets are rounded to 0.01 units. The README says unknown codes are rejected; they are not.
- Fix: use a real ISO 4217 list for fiat, or read `currencies.wallet_precision`, and reject everything else.

### M3. `min_fee` larger than the amount produces a negative merchant net and a debit larger than the payment
- `backend/internal/fees/compute.go:62-64,86-88`; documented in README line 66.
- Documented, so not a spec break, but the fee journal will push the merchant liability below zero for that payment. Consider capping the fee at the amount for merchant-borne rules, or refusing the payment, and test whichever is chosen (no test covers fee > amount today).

### M4. The append-only trigger allows retroactive closure
- `backend/internal/fees/schema.sql:47-49`.
- `UPDATE fee_rules SET effective_to = now() - interval '30 days'` is accepted, rewriting which rule "was active" historically (snapshots still pin payments, but audit and `Resolve(At: past)` change).
- Fix: also require `NEW.effective_to >= least(OLD.effective_to, statement_timestamp()) - small tolerance`, or `>= NEW.effective_from` and `>= transaction_timestamp()` with the app passing DB time.

### M5. Soft-deleted payments can receive a snapshot
- `backend/internal/fees/service.go:180,190`: raw SQL ignores `payment_requests.deleted_at`. Add `AND deleted_at IS NULL` to both statements.

### M6. Fee rule management accepts API keys
- `backend/internal/api/router.go:421-428`: admin routes ride on `JWTOrAPIKey`, so an API key belonging to an operator-platform admin can create and version global pricing. Matches existing admin groups, but pricing is higher stakes than referrals; prefer `JWTAuth` for `/admin/fee-rules`.

### M7. Config read with `os.Getenv` inside the wiring function
- `backend/internal/modules/fees.go:26,31`: MODULES.md rule 6 puts module config under a config section; reading env directly bypasses `config.Load` validation and test injection. Move `FEES_*` into `config.Config`.

### M8. Test gaps
- No test for: superseding a future version (C1); overlapping creates (I2); huge-exponent or out-of-range decimals (I1); fee > amount (M3); precision beyond column scale (M1); unknown 3-letter currency (M2).
- `TestNewRouterMountsFeeRoutesBehindAuth` (`routes_fees_test.go:232`) proves 401 without credentials but nothing proves a real, authenticated non-admin merchant gets 403 from `RequirePermission`; the operator-gate tests use a fake guard. Add one router-level test with a member lacking `system.admin` and one with `system.admin` on a non-operator platform.
- The compute tests are otherwise good: boundaries at, below and above each slab bound, half-up per currency, clamp and tax order.

## What is solid

- Pure `Compute`/`resolve`/`preview` with typed errors; specificity is a total order enforced as data shape.
- Clamp-then-round is safe because `min_fee`/`max_fee` are validated onto the currency grid; tax is stated (on the rounded fee) and consistent between README, code and tests.
- Window semantics `[from, to)` match between Go (`ActiveAt`) and SQL; all times normalised to UTC.
- `NewVersion` concurrency (row lock, head check, unique index backstop) is correct for concurrent edits of the head; the 8-editor test proves it.
- The append-only and snapshot-fixed triggers block in-place edits from every path, including direct SQL; the fee journal's accounts and signs follow the ledger's debit-positive convention and balance.

## Checks run

- `cd backend && go build ./... && go vet ./... && go test ./...`: pass.
- `make test-integration`: pass (all packages ok; fees 278 s).
- C1 reproduced with an integration test supplied through `go test -overlay` (no files in the worktree were changed).
- I1 measured with a unit test supplied through `go test -overlay`.
