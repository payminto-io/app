# paymentswitch

The switch core: payment intents, attempts and refunds, the status vocabulary, and the map from each connector's raw statuses onto it.
The package is named `paymentswitch` because `switch` is a Go keyword.
Shape follows Hyperswitch's `payment_intent` / `payment_attempt`; design in `.scratch/payments-v1/issues/05-switch-core.md`.

## Owns

Tables (migration `2026100703_switch_intents_attempts`):

- `switch_payment_intents` - the merchant-facing payment; unique `(merchant_id, idempotency_key)` with a canonical `request_hash`.
- `switch_payment_attempts` - one row per connector try; unique `(connector_code, connector_transaction_id)`.
- `switch_refunds` - unique `(merchant_id, idempotency_key)` and `(connector_code, connector_refund_id)`.
- `switch_webhook_events` - replay guard, unique `(connector_code, event_id)`.
- `switch_status_transitions` - audit trail of every status change with its reason (raw connector errors live here, never on the merchant-facing row).
- `switch_anomalies` - contradictions and stalls recorded for an operator (`evidence_after_terminal`, `stale_in_flight`, `amount_unknown`, `unmapped_status`); never turned into a status.

Ids are text with a type prefix (`pi_`, `pa_`, `re_`), safe to expose.
Amounts are `numeric(38,18)` plus a `varchar(16)` asset; never floats.

## Vocabulary

Intent: `requires_payment_method`, `requires_confirmation`, `requires_action`, `processing`, `requires_capture`, `partially_captured`, `partially_paid`, `succeeded`, `failed`, `cancelled`.

Attempt: `started`, `pending`, `authentication_pending`, `authorized`, `capture_initiated`, `charged`, `partial_charged`, `partially_paid`, `underpaid`, `overpaid`, `capture_failed`, `authorization_failed`, `void_initiated`, `voided`, `void_failed`, `failure`.

Refund: `initiated`, `pending`, `succeeded`, `failed`.

Connector evidence wins: `capture_failed`, `capture_initiated`, `void_initiated` and `void_failed` all have edges to what the connector later proves (charged, partial_charged, voided, or back to authorized when nothing landed).
Evidence that contradicts a terminal state is recorded as an anomaly, not applied and not swallowed.

The transition tables live in `statemachine.go` and every allowed and disallowed edge is enumerated in `statemachine_test.go`.
`IntentStatusFor` derives the intent status from the active attempt.
Deviations from Hyperswitch: `capture_failed` and `void_failed` keep the authorization, so the intent stays `requires_capture`; `partially_paid`, `underpaid` and `overpaid` exist because a chain deposit can arrive short or over. An open deposit is `requires_action` (the customer must send funds), and `partially_paid` closes as the terminal intent status `partially_paid` with the shortfall in metadata when the merchant cancels; the received funds stay on the books and await refund (ticket 11).

`status_map.go` holds one map per connector code from its raw statuses to ours.
An unmapped raw status is `ErrUnmappedStatus`, never a guess, and `CheckStatusMap` runs at wiring so a connector with a gap never starts.

## Flows

Every command is claim, call, apply: an exclusive compare-and-set moves the row into an in-flight status (`processing` with a new attempt, `capture_initiated`, `void_initiated`, refund `initiated`; a same-status transition is a conflict), the connector is called outside any transaction, then a second transaction applies the mapped result.
Two confirms, two captures or two cancels on one intent cannot both win; the loser sees `ErrInvalidTransition`.

Only a typed `connectors.ErrDeclined` is terminal. Any other error is "outcome unknown": the attempt stays in flight with a typed error code and a redacted message (the raw error goes to the transition reason and the log), no new attempt opens, a retry of the same operation re-sends with the same connector idempotency key and amount, and the reconciler resolves it with `Sync`/`SyncRefund`.
The capture amount is stored at claim time (`amount_to_capture`); a captured event without an amount settles that, never the full authorization, and with neither the switch refuses and records an `amount_unknown` anomaly.
Lock order everywhere: intent, then attempt, then refund; webhooks read the attempt or refund unlocked to find the intent, then lock in order.

- `Create` - idempotent on `(merchant, key)`; same body replays, different body is `ErrIdempotencyConflict`. `Confirm: true` runs the first attempt.
- `Confirm` - selects a connector through `ConnectorSelector` (default `FirstEnabledSelector`, ticket 06 replaces it), creates the attempt, calls `Authorize`.
- `Capture` - full or partial; partial is terminal.
- `Cancel` - local before any attempt, `Void` at the connector otherwise.
- `Refund` - keyed like `Create`; the refundable balance counts pending refunds.
- `Sync` - resolves unknown outcomes for the active attempt and every open refund; connectors answer by our attempt or refund id when no connector id was stored. Sync and webhooks keep `next_action` unless the connector supplies a replacement or the status leaves the customer nothing to do.
- `HandleWebhook` - connector verifies authenticity (signature and timestamp window), the switch records the event id and applies it; a replay or an out-of-order event is answered 200 with an `ignored` marker, money-in evidence after a terminal state becomes an anomaly.
- `Reconciler` (`reconciler.go`) - a worker registered with the manager; syncs attempts in `pending`/`capture_initiated`/`void_initiated` and refunds in `initiated`/`pending` with per-row backoff, and past `MaxAge` records a `stale_in_flight` anomaly and logs at error level rather than inventing a status.

## Ledger

The first time a card attempt has money in (`charged`, `partial_charged`) the switch posts one `payment` journal in the same transaction: debit `connector/<code>` asset, credit `member/<merchant>` liability, key `switch.payment.<attempt_id>`.
A chain deposit posts in the asset actually received, using the ledger's chain-qualified code (`USDC.ETH`), into `platform/crypto_assets` against the merchant's liability in that asset, by delta as confirmations arrive (`partially_paid`, `charged`, `overpaid`), keyed by the cumulative received amount; the USD price is journal metadata only. Conversion is ticket 10.
A successful refund posts the reverse as a `refund` journal keyed `switch.refund.<refund_id>`.
Exactly once is guaranteed by the shared transaction and the ledger's own idempotency key.

Domain events `switch.payment.succeeded.v1`, `switch.payment.failed.v1` and `switch.refund.succeeded.v1` are emitted after commit through the existing event emitter (`ee_events`).

## Config

`SWITCH_CONNECTORS` (default `mock,chaindeposit`) and `SWITCH_MOCK_WEBHOOK_SECRET`; see `internal/modules/paymentswitch.go`.
The mock refuses to wire in staging and production.

## Routes

`internal/api/routes_paymentswitch.go`: `POST /api/v2/payments`, `GET /api/v2/payments/:id` (`?sync=true` syncs first), `POST .../confirm`, `/capture`, `/cancel`, `/refunds`, and the public `POST /api/v2/webhooks/:connector`.
Bodies are snake_case, amounts are decimal strings, success is `{"payment": ...}` or `{"refund": ...}`, errors are `{"error": {"code", "message"}}`.
