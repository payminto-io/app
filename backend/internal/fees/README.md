# fees

Core module that owns versioned fee rules: resolving which rule prices a payment attempt, computing the fee, previewing it, snapshotting the rule version per attempt, and posting the fee journal on capture.
Design source: ticket `.scratch/payments-v1/issues/02-fee-rules.md` (Kuberopay ADR 0053, re-implemented).

## Port

`port.go` is the only surface other modules may import.

- `Resolve(ctx, Query) (Rule, error)` picks the most specific active rule.
- `Compute(Rule, amount) Breakdown` is a pure function: fee, tax, customer total, merchant net.
- `Preview(ctx, PreviewRequest)` resolves at the current time, applies an optional fee bearer override, and computes.
- `CreateRule`, `NewVersion`, `GetRule`, `ListRules` manage rules.
- `Snapshot(ctx, tx, AttemptRef, Query)` and `PostFee(ctx, tx, AttemptRef, captured)`: see "Payment path".

Typed errors: `*ValidationError` (with `Field`), `*OverlapError` and `*AmbiguousRuleError` (with `RuleIDs`), `ErrNoRule`, `ErrNotFound`, `ErrStaleVersion`, `ErrSurchargeForbidden`, `ErrFeeExceedsAmount`, `ErrSnapshotNotFound`, `ErrSnapshotConflict`, `ErrPaymentNotFound`.

## Payment path

Both calls run in the caller's transaction and are made by the switch (ticket 05).

1. **At attempt creation:** `Snapshot(ctx, tx, AttemptRef{PaymentRequestID, AttemptID}, Query{method, connector, card_type, region, currency, chain})`.
   Fees resolves the rule itself, reads the merchant from `payment_requests.member_id` (locking the row, refusing a soft-deleted payment), derives the ledger asset, and stores one `fee_snapshots` row per attempt.
   A retry on another connector is a new attempt with its own snapshot.
   Calling it again for the same attempt returns the stored snapshot; the same attempt id on another payment is `ErrSnapshotConflict`.
2. **When the attempt succeeds or is captured:** `PostFee(ctx, tx, AttemptRef, capturedAmount)`.
   The fee is recomputed from the snapshotted rule version on the captured amount; merchant, currency and ledger asset come from the stored rows, never from the caller.
   It posts one fee journal keyed `fee:attempt:<attempt id>` and sets the legacy `payment_requests.fee_rule_id/fee_rule_version` to that attempt's rule (the latest successful attempt wins).
   Replaying with the same amount is a no-op; a different amount is refused by the ledger (`ledger.ErrIdempotencyConflict`).
   A fee plus tax above the captured amount is `ErrFeeExceedsAmount`.

Nothing is posted for an attempt that never succeeds, so an expired or cancelled payment carries no fee.

## Tables

`fee_rules` and `fee_snapshots` (migration `2026100702_fees_rules`, DDL in `schema.sql`; the migration repeats it verbatim and a unit test enforces that).

| `fee_rules` column | meaning |
| --- | --- |
| `lineage_id`, `version` | every edit is version n+1 of the same lineage; `(lineage_id, version)` is unique |
| `method` | `card`, `upi`, `bank`, `crypto` |
| `connector`, `card_type`, `region` | nullable scope; null matches anything |
| `currency` | ISO 4217 code for fiat methods, an on-chain asset code for `crypto` |
| `minor_units` | the currency's precision when the version was written; `Compute` rounds to it |
| `percent`, `flat` | percent is in percent (`2.9` means 2.9%), at most 6 decimals; flat is in currency units on the currency's grid |
| `slabs` | jsonb array of `{up_to, percent, flat}`; replaces rule-level percent and flat |
| `min_fee`, `max_fee` | nullable clamps on the currency's grid |
| `taxable`, `tax_percent` | tax charged on the fee; `tax_percent` is required when taxable and 0 otherwise |
| `fee_bearer` | `merchant` or `customer` |
| `effective_from`, `effective_to` | active window `[from, to)`; `to` null means open |
| `created_by` | `member:<id>` of the admin who wrote the version |

`fee_snapshots`: `attempt_id` (unique), `payment_request_id`, `merchant_id`, `fee_rule_id`, `fee_rule_version` (composite FK to `fee_rules`), `currency`, `ledger_asset`, `fee_bearer`, `created_at`. Append-only by trigger.

`payment_requests` gains nullable `fee_rule_id` and `fee_rule_version` (composite FK, both-or-neither check), kept for compatibility and set by `PostFee`.

## Rules are never mutated

A trigger rejects `DELETE`, `TRUNCATE` and any `UPDATE` other than moving `effective_to` earlier, and refuses closing a version into the past unless the transaction set `fees.closing_version` (only `NewVersion` does).
`NewVersion` locks the head row, refuses a rule that is not the lineage head (`ErrStaleVersion`), closes every version of the lineage whose window extends past the new start at `GREATEST(effective_from, new start)`, and inserts n+1 in one transaction.
A superseded version that had not started yet becomes an empty window.
`effective_from` defaults to now and may not be backdated.
A version body carries pricing only; scope and currency belong to the lineage.

## No overlap

The exclusion constraint `fee_rules_no_overlap` (btree_gist) makes it impossible for two rules with the same method, currency, connector, card type and region to be active at the same instant, whatever path writes them.
A conflicting create or version is `*OverlapError` naming the existing rules (HTTP 409 `overlapping_fee_rule`).
To schedule a price change, version the existing rule with a future `effective_from`.
`*AmbiguousRuleError` remains as a defensive check in `resolve` and should be unreachable.

## Resolution

Scope shapes form a ladder, enforced at validation and by a check constraint: `card_type` needs `connector` and method `card`; `region` needs `card_type`.
Specificity, highest first: connector + card_type + region, connector + card_type, connector, method default.
Among rules matching the method, currency and the query time, the highest specificity wins.

## Computation

1. Base: `amount * percent / 100 + flat`, or the slab containing the amount.
   Slab `i` covers `(up_to[i-1], up_to[i]]`: the lower bound is exclusive, the upper inclusive; the last slab must have `up_to: null`.
   Slabs are tiers on the whole amount, not marginal bands.
2. Clamp to `[min_fee, max_fee]`.
3. Round (see below).
4. Tax: `fee * tax_percent / 100`, rounded the same way, computed on the rounded fee.
5. Bearer:
   - `customer`: `customer_total = amount + fee + tax`, `merchant_net = amount` (a surcharge).
   - `merchant`: `customer_total = amount`, `merchant_net = amount - fee - tax`.
6. A fee plus tax above the amount (for example a `min_fee` larger than a small payment) is refused with `ErrFeeExceedsAmount` (HTTP 422 `fee_exceeds_amount`) in preview and in `PostFee`, never reported as a negative net.

## Rounding and precision

Fee and tax are rounded half-up to the version's `minor_units`.

- Fiat: the ISO 4217 list in `precision.go` with its minor units (2 for most; 0 for JPY, KRW, VND, CLP and others; 3 for BHD, KWD, OMR, JOD, TND, IQD, LYD; 4 for CLF, UYW). Fiat methods (`card`, `upi`, `bank`) accept only these.
- On-chain assets (method `crypto` only): USDC, USDT, PYUSD, EURC, TRX 6; BTC 8; SOL 9; ETH, DAI, POL, MATIC, BNB 18; extended by `FEES_ASSET_PRECISION`.
- Any other code is rejected.

Every decimal input is bounded before any arithmetic (exponent within plus or minus 40, at most 40 digits, below 1e20), so a value like `1e100000000` is refused in microseconds instead of costing seconds and gigabytes to rescale.
Percentages may have at most 6 decimals; money values may not be finer than the currency's minor unit. Nothing is silently rounded by Postgres.
Request bodies on fee routes are capped at 16 KiB.

## Surcharge policy

A customer-borne fee is a surcharge.
Methods listed in `FEES_SURCHARGE_FORBIDDEN_METHODS` refuse it with `ErrSurchargeForbidden`, at rule creation and versioning, at preview, and at snapshot.
The default forbids `upi` (NPCI does not permit surcharging UPI).

## Ledger

`PostFee` posts one `fee` journal through `ledger.PostIn`, key `fee:attempt:<attempt id>`, reference `payment_attempt/<attempt id>`, in the snapshot's ledger asset: the bare code for fiat (`USD`), `CODE.CHAIN` for on-chain assets (`USDC.BASE`), matching ticket 01's naming.

| account | amount |
| --- | --- |
| `member/<merchant id>/<asset>/liability` | `+(fee + tax)` (debit: we owe the merchant less) |
| `fees/platform/<asset>/income` | `-fee` |
| `fees/tax/<asset>/liability` | `-tax` (only when tax is non-zero) |

The journal is the same for both bearers; the bearer changes what the payment journal credits the merchant.
A zero fee posts no journal.

## HTTP

Routes live in `internal/api/routes_fees.go`, all under `/api/v1`, JSON snake_case, decimals as strings (numbers are accepted on input).

- `POST /fees/preview` (dashboard session or API key): `{amount, currency, method, connector?, card_type?, region?, fee_bearer?}` returns `{rule_id, version, currency, fee_bearer, amount, fee, tax, customer_total, merchant_net}`.
- `GET /admin/fee-rules?method=&currency=&lineage_id=`, `POST /admin/fee-rules`, `GET /admin/fee-rules/:id`, `POST /admin/fee-rules/:id/versions`: dashboard session only (API keys are refused), plus `system.admin`, plus the operator gate below.

Errors are `{error, code}` with `field` for validation and `rule_ids` for overlap and ambiguity: 400 `invalid_json`/`invalid_request`, 404 `no_fee_rule`/`not_found`, 409 `overlapping_fee_rule`/`ambiguous_fee_rule`/`stale_version`, 422 `surcharge_forbidden`/`fee_exceeds_amount`.

## Access

Fee rules are platform-wide, but `system.admin` is granted per external platform to every platform owner.
`FEES_OPERATOR_PLATFORM_ID` restricts management to the operator's platform.
In `DEVELOPMENT` and `TEST` it may be unset (any `system.admin`); in `STAGING` and `PRODUCTION` the admin routes answer 403 `fee_admin_not_configured` until it is set.

## Configuration

Read by `config.Load` into `config.FeesConfig`.

| key | default | meaning |
| --- | --- | --- |
| `FEES_SURCHARGE_FORBIDDEN_METHODS` | `upi` | comma list of methods that refuse a customer-borne fee, or `none` |
| `FEES_ASSET_PRECISION` | empty | extra on-chain assets, `CODE:decimals,...` (decimals 0..18; ISO codes refused) |
| `FEES_OPERATOR_PLATFORM_ID` | unset | the only platform allowed to manage fee rules |

The migration needs the `btree_gist` extension; see `docs/OPERATIONS.md`, "Fee rules".

## Events

None yet.

## Tests

Unit tests next to the code; `integration_test.go` (`-tags=integration`) covers versioning atomicity, superseding a scheduled version, concurrent edits, overlap refusal (service, direct SQL and concurrent creates), append-only enforcement including retroactive closes and snapshots, exact decimal round trips, snapshot per attempt with retry on another connector, fee posted on the captured amount from the snapshotted version, chain-qualified assets, and soft-deleted payments.
