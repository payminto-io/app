# links

Core module that owns the payment link: the six-step form's object, its lifecycle, the public render model the hosted checkout reads, and paying a link.
Design source: ticket `.scratch/payments-v1/issues/03-payment-link-model.md` and the form in `../reports/Payments V1 on Kuberopay.md` (Kuberopay's link builder, re-implemented).

## Port

`port.go` is the only surface other modules may import.

- `Create`, `Get`, `List`, `Update`, `Delete` (drafts only), `Publish`, `Pause`, `Archive`, `Duplicate` for merchants, scoped by external platform.
- `Render(short_code)` returns the `RenderModel` for checkout; `Preview(form, link_id?)` returns the same model for an unsaved form plus the methods it drops and why (one builder, `render`); `Options()` lists publishable methods and currencies from an optional `Catalog` on the payment creator; `Pay(short_code, PayRequest)` creates the payment.
- Ports this module depends on:
  - `FeeQuoter` (`Resolve`, `Preview`), satisfied by `fees.Port`. Links never snapshot or post fees; the switch does that per attempt (fees README, "Payment path").
  - `PaymentCreator` turns one reserved use into a payment. `Connectors` says which connectors take a method in an environment; `CreatePayment` creates it, with `LinkPaymentID` as the payment's unique reference, so it is idempotent and a second payment for one use is impossible; `FencePayment` returns the use's payment or makes it impossible to create from then on; `CancelPayment` cancels the payment of a use the link no longer tracks; `OpenPayments` says which uses' payments are still open and unpaid. Only an error wrapping `ErrNotCreated`, or a `*links.Error`, promises nothing was created. The default is `service.LinkPaymentCreator` (Payminto's payment service, reference `pl_<LinkPaymentID>`); the switch (ticket 05) provides another.
  - `environment.Guard` (`WithGuard`): links are tagged with the process environment and `Pay` calls `guard.Require` before reserving (`link_environment_mismatch`).
  - `DestinationVerifier` answers whether a settlement override's destination is verified. The default `NoDestinations` verifies nothing, so an override cannot publish until ticket 11 supplies a real one.
- Every refusal is a `*links.Error` with a typed `Code` and the `Field` it concerns; when several rules fail, `Errors` carries all of them.

## Lifecycle

```
draft --publish--> active --pause--> paused --publish--> active
  |                  |                  |
  +----archive-------+------archive-----+--> archived (terminal)
```

- `publish` runs the full validation below and mints the short code on first publish; resuming a paused link keeps it.
- `active` and `paused` are published: amount mode, amount, min, max, currency, methods, line items and fee bearer cannot change (`link_published_immutable`); every other edit re-runs publish validation. Duplicate to change them.
- `use_limit`, `expires_after_payments` and `multi_use` cannot drop below the payments already taken (`use_limit_below_uses`).
- `archived` is not editable (`link_not_editable`); only drafts can be deleted (`link_not_deletable`).
- Writes are conditional on `revision`; `Update` takes the revision the caller read (the PATCH merge base, or `If-Match`), and a concurrent change is `link_conflict`.

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

`Pay` validates the payer's input, reserves one use under a lock, then creates the payment:

1. `Idempotency-Key` is required (1-128 characters) and scoped to the link. A replay with the same body returns the stored result (`replayed: true`) even after the link closes; another body is `idempotency_key_reused`.
2. The process guard must match the link's environment (`link_environment_mismatch`).
3. Availability: paused (`link_paused`), archived (`link_archived`), expired (`link_expired`), use limit reached (`link_use_limit_reached`). Drafts are not public (`link_not_found`).
4. Payer input: the method must be on the link (`method_not_enabled`); hidden customer fields may not be sent (`customer_field_hidden`); required ones must be sent or prefilled (`customer_field_required`); email and E.164 phone formats; addresses when required (`address_required`, `address_invalid`); answers (`answer_unknown_question`, `answer_required`, `answer_invalid`). Every required question is asked on every payment: `per_order: false` only marks the answer as about the customer rather than the order, so no response reveals whether an email has paid before and no typed email skips a question.
5. Pricing: a connector with an active fee rule is chosen again (`method_unavailable` when none). In the link's currency the use carries the preview (`fee`, `tax`, `customer_total`); for a crypto asset priced in another currency it carries the rule id and version and no fee figure. A total beyond `numeric(38,18)` is `amount_invalid`.
6. `Store.Reserve` locks the link row, re-checks availability, the fee bearer the quote used, the use limit and the open-payment caps, and records a `pending` use with `reserved_until` (the lease, `LINKS_LEASE_SECONDS`, default 300). The use row is the record of intent: its id is the `LinkPaymentID` the creator is called with.
7. `CreatePayment` runs outside that transaction, bounded to half the lease.
   - Success: the use becomes `created`.
   - `ErrNotCreated` or a `*links.Error`: the use is marked `released` (the row and its answers stay) and given back; the same key may retry.
   - Any other error, or a failed `Complete`: the use stays `pending` and the payer gets `payment_in_progress` with `Retry-After` at the lease end.
8. A pending use whose lease ended is resolved by `FencePayment`: a payment found completes the use; otherwise the reference is fenced (Payminto writes a cancelled placeholder under the unique `reference_id`) and only then is the use released, so a creation still stalled in flight fails on the unique index instead of making a live, untracked payment. A failed fence leaves the use pending. The same-key retry does this inline (and starts afresh after a release); the `link_reservation_resolver` worker does it every 30 seconds for uses nobody retries.
9. If `Complete` finds the use already settled as released, the payment just created is cancelled through `CancelPayment` and logged as an anomaly (`links: anomaly: payment created for a released use`); the payer gets `payment_creation_failed` and the same-key retry starts afresh. With a creator that fences, this path is unreachable; it exists for creators that cannot.

Open-payment caps (multi-use links only): a use counts as open while pending and then while its payment is open and unpaid. Before each reservation the link's counted uses are checked with `OpenPayments`, and paid, cancelled or expired ones stop counting; a payment's expiry (the creator's `expires_at`, else the link's quote expiry) also ends it. IPv6 payers are grouped by /64. At most `LINKS_MAX_OPEN_PAYMENTS` (default 100) per link and `LINKS_MAX_OPEN_PAYMENTS_PER_CLIENT` (default 3) per payer IP (stored as a hash) are open at once; beyond that `open_payments_limit` (HTTP 429). This keeps an anonymous payer from draining the deposit-address pool.

The effective use limit is 1 for a single-use link, else the lower of `use_limit` and `expires_after_payments`, else unlimited.
A use is counted when the payment is created, not when it is paid; an abandoned payment keeps its use (see "Open questions").

## Fee preview

`FeePreview(link)` lists, per method, the connector, rule id and version, fee bearer and fee currency that will apply, and the breakdown (`amount`, `fee`, `tax`, `customer_total`, `merchant_net`) when the link has a real amount in the fee's currency; an unusable method carries its refusal code. Merchant single-link responses include it as `fee_preview`.

## Short codes and QR

12 characters from `[A-Za-z0-9]` drawn with `crypto/rand` and rejection sampling (about 71 bits), unique by constraint, retried five times on collision.
The link URL is `<CHECKOUT_BASE_URL>/l/<short_code>`. `GET /api/v2/public/links/:short_code/qr.svg` serves it as an SVG QR code (error correction M, four-module quiet zone), so a printed code needs no client library.

## Render model

`RenderModel` (`render.go`) is the public contract with checkout; `render_test.go` pins its key set.
It carries no ids beyond the short code, no metadata, reference, category, webhook, settlement, receipt note or success URL, and no prefill of a hidden field.
Methods that lost their connector or fee rule since publish are left out; with none left the model says `available: false, unavailable_reason: "no_methods_available"`.

## Tables

Migration `2026100710_links_payment_links` repeats `schema.sql` verbatim (`schema_test.go`); `links.Migrate` runs it in dev and test through `database.MigrateExpandSchema`.

- `payment_links`: every form field as a column (see `schema.sql`), plus `environment` (`live`/`test`, default `test`), `status`, `short_code` (unique), `uses_count`, `revision`, `published_at`. Check constraints keep `uses_count` within the single-use, `use_limit` and `expires_after_payments` bounds, and require a short code on active and paused links.
- `payment_link_line_items`, `payment_link_questions`: children ordered by `position`, replaced on save.
- `payment_link_payments`: one use of a link (pending, then created or released), one live use per key (partial unique index excluding released), unique on `(link_id, idempotency_key)`, with the request hash, method, connector, amount, fee preview and `(fee_rule_id, fee_rule_version)` (foreign key to `fee_rules`), customer fields, addresses, the processor's response, `client_key` (hash of the payer IP), `reserved_until` and `open_until`.

`payment_links` and `payment_link_payments` carry `environment`; the environment module's `VerifySchema` requires the column and counts both tables as data. The Payminto creator also writes the use's fee rule onto `payment_requests.fee_rule_id/fee_rule_version`.
- `payment_link_answers`: one row per answered question per use, with the question's label as asked.

## Configuration

| key | default | meaning |
| --- | --- | --- |
| `LINKS_LEASE_SECONDS` | 300 | lease of a pending use (at least 60) |
| `LINKS_MAX_OPEN_PAYMENTS` | 100 | open payments per multi-use link; 0 disables |
| `LINKS_MAX_OPEN_PAYMENTS_PER_CLIENT` | 3 | open payments per payer IP per link; 0 disables |
| `TRUSTED_PROXIES` | empty | proxies whose `X-Forwarded-For` sets the client IP; empty trusts none |

It also reads `CHECKOUT_BASE_URL` and `FEES_ASSET_PRECISION`. The environment is the process's (`GATEWAY_ENVIRONMENT`), from the environment module.
The public routes are limited per client IP (120 reads, 20 pays per minute) in Redis when it answers (count and TTL set in one script) and in process memory when it does not; they never run unlimited. Behind a proxy `TRUSTED_PROXIES` must name it (docs/OPERATIONS.md, "Client IPs behind a proxy").

## HTTP

Routes in `internal/api/routes_links.go`, documented in `docs/API_SPECIFICATION.md` section 4.44.

## Events

None yet. Link-paid and use-limit events belong with the switch's payment events.

## Open questions

- A use is consumed when the payment is created. For a single-use link an abandoned payment closes the link; releasing the use when the payment expires needs the switch's payment events.
- `fees.Snapshot` takes the rule's own fee bearer; the switch must carry the link's bearer into the attempt's snapshot.
- The Payminto creator honours the quote expiry, but Payminto's payment requests have no amount tolerance and no per-payment webhook; `chain_tolerance_bps` and `webhook_id` are passed to the creator for the switch to honour.
- If writing the fee rule onto the payment request fails after creation, the use stays pending and is completed by the resolver, but that payment request keeps no fee rule; the use row still records it.
- A fence that finds a payment whose deposit address is still being assigned completes the use with that payment; if the assignment then fails, Payminto cancels the payment and the use counts no longer as open, but the link's use is not given back.
