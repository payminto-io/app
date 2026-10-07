# 02 Versioned fee rules and fee preview

Status: done
Owner: Backend engineer (Opus)
Blocked by: 01

## Goal
Fee rules per method, connector, card type and region; percent, flat, slab, min, max; tax flag; fee bearer; effective dates; versioned lineage, never mutated (Kuberopay ADR 0053). A preview endpoint applies the rule to a hypothetical amount. Every payment snapshots the rule version it used.

## Tables
`fee_rules` (id, lineage_id, version, method, connector nullable, card_type nullable, region nullable, currency, percent numeric, flat numeric, slabs jsonb nullable, min, max, taxable bool, fee_bearer merchant|customer, effective_from, effective_to nullable, created_by, created_at). Editing creates version n+1 and closes n.

## API
- `GET/POST /admin/fee-rules`, `POST /admin/fee-rules/:id/versions`.
- `POST /fees/preview` body `{amount, currency, method, connector?, card_type?, region?, fee_bearer?}` returns `{rule_id, version, fee, tax, customer_total, merchant_net}`.
- `fees.Resolve(ctx, Query) (Rule, error)` picks the most specific active rule; `fees.Compute(Rule, amount) Breakdown`.

## Acceptance
Specificity order tested; slab boundaries tested; min and max clamp; surcharge forbidden where method disallows; snapshot stored on `payment_requests.fee_rule_id/fee_rule_version`; fee lines posted to ledger via ticket 01.

## Comments

- 2026-10-07 (Backend, Opus): done on branch `fees`. Commits `10ad400` (port, Compute, resolution with typed ties), `7fa8d7e` (validation, surcharge policy, preview), `6211723` (fee_rules table, migration `2026100702_fees_rules`, versioning service, payment snapshot, fee journal, integration tests), `30eb457` (WireFees, registry line, `/fees/preview` and `/admin/fee-rules` routes), `acce8ec` (module README). Full report: `.superpowers/fees-report.md`. Deviations: `min`/`max` are `min_fee`/`max_fee`; `tax_percent` added; `RegisterFeesRoutes` takes the admin permission guard as a third argument (import cycle otherwise); `FEES_OPERATOR_PLATFORM_ID` gates rule management because `system.admin` is per platform. `ApplyToPayment` has no caller yet; the switch (05) should call it per attempt.
