# fees

Core module that owns versioned fee rules: resolving which rule prices a payment, computing the fee, previewing it, and snapshotting the rule version on the payment with a fee journal in the ledger.
Design source: ticket `.scratch/payments-v1/issues/02-fee-rules.md` (Kuberopay ADR 0053, re-implemented).

## Port

`port.go` is the only surface other modules may import.

- `Resolve(ctx, Query) (Rule, error)` picks the most specific active rule.
- `Compute(Rule, amount) Breakdown` is a pure function: fee, tax, customer total, merchant net.
- `Preview(ctx, PreviewRequest)` resolves at the current time, applies an optional fee bearer override, and computes.
- `CreateRule`, `NewVersion`, `GetRule`, `ListRules` manage rules.
- `ApplyToPayment(ctx, tx, PaymentFee)` writes the snapshot onto `payment_requests` and posts the fee journal, both in the caller's transaction.

Typed errors: `*ValidationError` (with `Field`), `*AmbiguousRuleError` (with the tied `RuleIDs`), `ErrNoRule`, `ErrNotFound`, `ErrStaleVersion`, `ErrSurchargeForbidden`, `ErrSnapshotConflict`, `ErrPaymentNotFound`.

## Tables

`fee_rules` (migration `2026100702_fees_rules`, DDL in `schema.sql`; the migration repeats it verbatim and a unit test enforces that).

| column | meaning |
| --- | --- |
| `lineage_id`, `version` | every edit is version n+1 of the same lineage; `(lineage_id, version)` is unique |
| `method` | `card`, `upi`, `bank`, `crypto` |
| `connector`, `card_type`, `region` | nullable scope; null matches anything |
| `currency` | the asset the fee is charged in |
| `percent`, `flat` | percent is in percent (`2.9` means 2.9%); flat is in currency units |
| `slabs` | jsonb array of `{up_to, percent, flat}`; replaces rule-level percent and flat |
| `min_fee`, `max_fee` | nullable clamps |
| `taxable`, `tax_percent` | tax charged on the fee; `tax_percent` is required when taxable and 0 otherwise |
| `fee_bearer` | `merchant` or `customer` |
| `effective_from`, `effective_to` | active window `[from, to)`; `to` null means open |
| `created_by` | `member:<id>` of the admin who wrote the version |

Columns named `min`/`max` in the ticket are `min_fee`/`max_fee` here, and `tax_percent` is added because a tax flag alone cannot compute a tax.

`payment_requests` gains nullable `fee_rule_id` and `fee_rule_version`, with a composite foreign key to `fee_rules (id, version)`, a check that both or neither are set, and a trigger that fixes the pair once written.

## Rules are never mutated

A trigger rejects `DELETE`, `TRUNCATE` and any `UPDATE` other than moving `effective_to` earlier (closing a version).
`NewVersion` locks the head row with `SELECT ... FOR UPDATE`, refuses a rule that is not the lineage head (`ErrStaleVersion`), closes it, and inserts n+1 in one transaction; if the insert fails the close rolls back.
Version n is closed at `max(n+1.effective_from, n.effective_from)`, so superseding a version that has not started yet leaves it an empty window rather than an inverted one.
`effective_from` defaults to now and may not be in the past: history is not rewritten.
A version body carries pricing only; scope and currency belong to the lineage and cannot change.

## Resolution

Scope shapes form a ladder, enforced at validation and by a check constraint: `card_type` needs `connector` and method `card`; `region` needs `card_type`.
Specificity, highest first: connector + card_type + region, connector + card_type, connector, method default.
Among rules matching the method, currency and the query time, the highest specificity wins.
Two matching rules at the winning specificity are a configuration error returned as `*AmbiguousRuleError`, never a silent pick.
A tie below the winner is not an error.

## Computation

1. Base: `amount * percent / 100 + flat`, or the slab containing the amount.
   Slab `i` covers `(up_to[i-1], up_to[i]]`: the lower bound is exclusive, the upper inclusive; the last slab must have `up_to: null`.
   Slabs are tiers on the whole amount, not marginal bands.
2. Clamp to `[min_fee, max_fee]`.
3. Round (see below).
4. Tax: `fee * tax_percent / 100`, rounded the same way, computed on the rounded fee.
5. Bearer:
   - `customer`: `customer_total = amount + fee + tax`, `merchant_net = amount` (a surcharge).
   - `merchant`: `customer_total = amount`, `merchant_net = amount - fee - tax`. A fee larger than the amount is shown as a negative net rather than hidden.

## Rounding

Fee and tax are rounded half-up to the currency's minor unit (`MinorUnits` in `compute.go`).

- Fiat (any three-letter ISO 4217 code): 2 places by default; 0 for JPY, KRW, VND, CLP, ISK, UGX, XAF, XOF; 3 for BHD, KWD, OMR, JOD, TND, IQD, LYD.
- Crypto: the asset's on-chain decimals from the precision map: USDC, USDT, PYUSD, EURC, TRX 6; BTC 8; SOL 9; ETH, DAI, POL, MATIC, BNB 18.
- Any other code is rejected as an unknown currency; a new asset is added to the map in code, with a test.

`min_fee`, `max_fee` and preview amounts must not be finer than the minor unit.

## Surcharge policy

A customer-borne fee is a surcharge.
Methods listed in `FEES_SURCHARGE_FORBIDDEN_METHODS` refuse it with `ErrSurchargeForbidden`, both when a rule is created or versioned and when a preview asks for `fee_bearer: customer`.
The default forbids `upi` (NPCI does not permit surcharging UPI).

## Ledger

`ApplyToPayment` posts one `fee` journal through `ledger.PostIn`, idempotency key `fee:payment_request:<id>`, reference `payment_request/<id>`:

| account | amount |
| --- | --- |
| `member/<merchant id>/<currency>/liability` | `+(fee + tax)` (debit: we owe the merchant less) |
| `fees/platform/<currency>/income` | `-fee` |
| `fees/tax/<currency>/liability` | `-tax` (only when tax is non-zero) |

The journal is the same for both bearers; the bearer changes what the payment journal credits the merchant (the customer total when the customer bears the fee, the amount when the merchant does).
A zero fee posts no journal but still records the snapshot.
Replaying with the same rule is a no-op; a different rule returns `ErrSnapshotConflict`.

## HTTP

Routes live in `internal/api/routes_fees.go`, all under `/api/v1`, JSON snake_case, decimals as strings (numbers are accepted on input).

- `POST /fees/preview` (JWT or API key): `{amount, currency, method, connector?, card_type?, region?, fee_bearer?}` returns `{rule_id, version, currency, fee_bearer, amount, fee, tax, customer_total, merchant_net}`.
- `GET /admin/fee-rules?method=&currency=&lineage_id=`, `POST /admin/fee-rules`, `GET /admin/fee-rules/:id`, `POST /admin/fee-rules/:id/versions` (JWT or API key, plus `system.admin`, plus the operator gate below).

Errors are `{error, code}` with `field` for validation and `rule_ids` for ambiguity: 400 `invalid_json`/`invalid_request`, 404 `no_fee_rule`/`not_found`, 409 `ambiguous_fee_rule`/`stale_version`, 422 `surcharge_forbidden`.

## Access

Fee rules are platform-wide, but `system.admin` is granted per external platform to every platform owner.
`FEES_OPERATOR_PLATFORM_ID` restricts management to the operator's platform.
In `DEVELOPMENT` and `TEST` it may be unset (any `system.admin`); in `STAGING` and `PRODUCTION` the admin routes answer 403 `fee_admin_not_configured` until it is set.

## Configuration

| key | default | meaning |
| --- | --- | --- |
| `FEES_SURCHARGE_FORBIDDEN_METHODS` | `upi` | comma list of methods that refuse a customer-borne fee, or `none` |
| `FEES_OPERATOR_PLATFORM_ID` | unset | the only platform allowed to manage fee rules |

## Events

None yet.
`ApplyToPayment` is called by the payment path (switch, ticket 05) inside its own transaction.

## Tests

Unit tests next to the code; `integration_test.go` (`-tags=integration`) covers versioning atomicity, concurrent edits, append-only enforcement, ties, decimal round trips, and the snapshot plus fee journal.
