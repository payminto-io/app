# links

Core module that owns the payment link: the six-step form's object, its lifecycle, the public render model the hosted checkout reads, and paying a link.
Design source: ticket `.scratch/payments-v1/issues/03-payment-link-model.md` and the form in `../reports/Payments V1 on Kuberopay.md` (Kuberopay's link builder, re-implemented).

## Port

`port.go` is the only surface other modules may import.

- `Create`, `Get`, `List`, `Update`, `Delete` (drafts only), `Publish`, `Pause`, `Archive`, `Duplicate` for merchants, scoped by external platform.
- `Render(short_code)` returns the `RenderModel` for checkout; `Pay(short_code, PayRequest)` creates the payment.
- Ports this module depends on:
  - `FeeQuoter` (`Resolve`, `Preview`), satisfied by `fees.Port`. Links never snapshot or post fees; the switch does that per attempt (fees README, "Payment path").
  - `PaymentCreator` turns one reserved use into a payment. `Connectors` says which connectors take a method in an environment; `CreatePayment` creates it. The default is `service.LinkPaymentCreator` (Payminto's payment service); the switch (ticket 05) provides another.
  - `DestinationVerifier` answers whether a settlement override's destination is verified. The default `NoDestinations` verifies nothing, so an override cannot publish until ticket 11 supplies a real one.
- Every refusal is a `*links.Error` with a typed `Code` and the `Field` it concerns; when several rules fail, `Errors` carries all of them.

## Lifecycle

```
draft --publish--> active --pause--> paused --publish--> active
  |                  |                  |
  +----archive-------+------archive-----+--> archived (terminal)
```

- `publish` runs the full validation below and mints the short code on first publish; resuming a paused link keeps it.
- `active` and `paused` are published: amount mode, amount, min, max, currency, methods and line items cannot change (`link_published_immutable`); every other edit re-runs publish validation. Duplicate to change them.
- `archived` is not editable (`link_not_editable`); only drafts can be deleted (`link_not_deletable`).
- Writes are conditional on `revision`; a concurrent change is `link_conflict`.

## Validation

Saving a draft refuses malformed values and contradictions (shape rules); a draft may be incomplete.
Publishing (and any edit of a published link) also requires completeness and checks every external reference.

| Rule | Code | When |
| --- | --- | --- |
| Title at most 200, description at most 5000 | `title_too_long`, `description_too_long` | save |
| Amount mode is fixed, customer or line_items | `amount_mode_invalid` | save |
| Currency is ISO 4217 or a known on-chain asset (fees precision) | `currency_unsupported` | save |
| Money is positive, bounded, on the currency's minor unit | `amount_invalid` | save |
| Fixed takes no min/max and no line items; customer takes no amount; line_items takes no client amount | `amount_bounds_not_allowed`, `line_items_not_allowed`, `amount_not_allowed` | save |
| Customer min at most max | `amount_range_invalid` | save |
| Line item: name, quantity 1-100000, unit price at least 0 on the grid, tax 0-100% | `line_item_invalid` | save |
| Reference at most 100, category at most 64, metadata at most 20 keys | `reference_id_too_long`, `category_too_long`, `metadata_invalid` | save |
| Customer field modes and prefills (email, E.164 phone) | `customer_field_policy_invalid` | save |
| Questions: key `[a-z0-9_]`, unique, label, type, select options | `question_invalid`, `question_key_duplicate` | save |
| Use limit and payment cap at least 1, only on multi-use links | `use_limit_invalid`, `use_limit_requires_multi_use` | save |
| Methods: known, crypto has chain and asset, no duplicates | `method_unknown`, `method_invalid`, `method_chain_required`, `method_asset_unsupported`, `method_duplicate` | save |
| Capture mode, 3DS policy, tolerance 0-1000 bps, quote expiry 60-86400 s, fee bearer | `capture_mode_invalid`, `three_ds_policy_invalid`, `chain_tolerance_invalid`, `quote_expiry_invalid`, `fee_bearer_invalid` | save |
| Success mode; success URL absolute http(s); text fields at most 1000 | `success_mode_invalid`, `success_url_invalid`, `text_too_long` | save |
| Receipt email needs an email field that is asked or prefilled | `receipt_requires_email` | save |
| Webhook belongs to the platform | `webhook_not_found` | save, publish |
| Settlement override shape; timing | `settlement_override_invalid`, `settlement_timing_invalid` | save |
| Logo https, accent `#RRGGBB`, language tag | `logo_url_invalid`, `accent_color_invalid`, `language_invalid` | save |
| Title, currency, amount (fixed), line items (line_items, sum above 0), at least one method | `title_required`, `currency_required`, `amount_required`, `line_items_required`, `amount_invalid`, `methods_required` | publish |
| Success message or URL for the chosen mode | `success_message_required`, `success_url_required` | publish |
| Expiry in the future | `expires_at_in_past` | publish |
| Manual capture needs card; hold in asset needs crypto | `capture_mode_requires_card`, `hold_in_asset_requires_crypto` | publish |
| Override destination verified | `settlement_destination_unverified` | publish |
| Each method has a connector in the link's environment | `method_no_connector` | publish |
| Each method has an active fee rule on one of those connectors | `method_no_fee_rule`, `fee_rule_ambiguous` | publish |
| A customer-borne fee is allowed for the method | `surcharge_forbidden` | publish |
| A surcharge needs the fee in the link's currency (no conversion quotes yet) | `surcharge_needs_quote` | publish |
| The fee does not exceed the amount (fixed, line items, or customer minimum) | `fee_exceeds_amount` | publish |

## Amounts

- `fixed`: the merchant's amount.
- `customer`: the payer's amount, positive, on the currency grid, inside `[amount_min, amount_max]` when set.
- `line_items`: computed server side, never taken from a client: per line `quantity x unit_price`, plus tax `round_half_up(subtotal x tax_rate / 100)` to the currency's minor units, summed. The `amount` column stores this sum; the API exposes it as `total`.

## Paying a link

`Pay` validates the payer's input, then reserves one use, then creates the payment:

1. `Idempotency-Key` is required (1-128 characters) and scoped to the link. A replay with the same body returns the stored result (`replayed: true`) even after the link closes; another body is `idempotency_key_reused`; a key whose payment is still being created is `payment_in_progress`.
2. Availability: paused (`link_paused`), archived (`link_archived`), expired (`link_expired`), use limit reached (`link_use_limit_reached`). Drafts are not public (`link_not_found`).
3. Payer input: the method must be on the link (`method_not_enabled`); hidden customer fields may not be sent (`customer_field_hidden`); required ones must be sent or prefilled (`customer_field_required`); email and E.164 phone formats; billing and shipping addresses when required (`address_required`, `address_invalid`); answers (`answer_unknown_question`, `answer_required`, `answer_invalid`). A required question with `per_order: false` is asked once per customer email on the link.
4. Pricing: a connector with an active fee rule is chosen again at pay time (`method_unavailable` when none). When the fee is in the link's currency the payment carries the preview (`fee`, `tax`, `customer_total`); for a crypto asset priced in another currency it carries the rule id and version but no fee figure, because none can be computed without a quote.
5. `Store.Reserve` locks the link row, re-checks availability and the limit, records the use with its answers and fee rule, and increments `uses_count`. Exactly one payer gets the last use.
6. `PaymentCreator.CreatePayment` runs outside that transaction. If it fails the reservation is deleted and the use given back, so the same key can retry.

The effective use limit is 1 for a single-use link, else the lower of `use_limit` and `expires_after_payments`, else unlimited.
A use is counted when the payment is created, not when it is paid; an abandoned payment keeps its use (see "Open questions").

## Short codes and QR

12 characters from `[A-Za-z0-9]` drawn with `crypto/rand` and rejection sampling (about 71 bits), unique by constraint, retried five times on collision.
The link URL is `<CHECKOUT_BASE_URL>/l/<short_code>`; the QR encodes that URL and is rendered client side.

## Render model

`RenderModel` (`render.go`) is the public contract with checkout; `render_test.go` pins its key set.
It carries no ids beyond the short code, no metadata, reference, category, webhook, settlement, receipt note or success URL, and no prefill of a hidden field.
Methods that lost their connector or fee rule since publish are left out; with none left the model says `available: false, unavailable_reason: "no_methods_available"`.

## Tables

Migration `2026100706_links_payment_links` repeats `schema.sql` verbatim (`schema_test.go`); `links.Migrate` runs it in dev and test through `database.MigrateExpandSchema`.

- `payment_links`: every form field as a column (see `schema.sql`), plus `environment` (`live`/`test`, default `test`), `status`, `short_code` (unique), `uses_count`, `revision`, `published_at`. Check constraints keep `uses_count` within the single-use, `use_limit` and `expires_after_payments` bounds, and require a short code on active and paused links.
- `payment_link_line_items`, `payment_link_questions`: children ordered by `position`, replaced on save.
- `payment_link_payments`: one use of a link (pending, then created), unique on `(link_id, idempotency_key)`, with the request hash, method, connector, amount, fee preview and `(fee_rule_id, fee_rule_version)` (foreign key to `fee_rules`), customer fields, addresses and the processor's response.
- `payment_link_answers`: one row per answered question per use, with the question's label as asked.

## Configuration

No keys of its own. It reads `CHECKOUT_BASE_URL` for link URLs and `FEES_ASSET_PRECISION` for on-chain asset precision.
`environment` is `live` when `SERVER=PRODUCTION` and `test` otherwise, until ticket 13 owns live and test isolation.

## HTTP

Routes in `internal/api/routes_links.go`, documented in `docs/API_SPECIFICATION.md` section 4.44.

## Events

None yet. Link-paid and use-limit events belong with the switch's payment events.

## Open questions

- A use is consumed when the payment is created. For a single-use link an abandoned payment closes the link; releasing the use when the payment expires needs the switch's payment events.
- If the process dies between `CreatePayment` and `Complete`, the reservation stays `pending` and its key answers `payment_in_progress`; a reconciler should complete or release pending rows older than the payment expiry.
- `fees.Snapshot` takes the rule's own fee bearer; a link whose `fee_bearer` differs from its rule's needs the switch to carry the link's bearer into the attempt's snapshot.
