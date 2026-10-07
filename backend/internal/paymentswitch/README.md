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
- `switch_status_transitions` - audit trail of every status change with its reason.

Ids are text with a type prefix (`pi_`, `pa_`, `re_`), safe to expose.
Amounts are `numeric(38,18)` plus a `varchar(16)` asset; never floats.

## Vocabulary

Intent: `requires_payment_method`, `requires_confirmation`, `requires_action`, `processing`, `requires_capture`, `partially_captured`, `succeeded`, `failed`, `cancelled`.

Attempt: `started`, `pending`, `authentication_pending`, `authorized`, `capture_initiated`, `charged`, `partial_charged`, `partially_paid`, `overpaid`, `capture_failed`, `authorization_failed`, `void_initiated`, `voided`, `void_failed`, `failure`.

Refund: `pending`, `succeeded`, `failed`.

The transition tables live in `statemachine.go` and every allowed and disallowed edge is enumerated in `statemachine_test.go`.
`IntentStatusFor` derives the intent status from the active attempt.
Deviations from Hyperswitch: `capture_failed` and `void_failed` keep the authorization, so the intent stays `requires_capture`; `partially_paid` and `overpaid` exist because a chain deposit can arrive short or over.

`status_map.go` holds one map per connector code from its raw statuses to ours.
An unmapped raw status is `ErrUnmappedStatus`, never a guess, and `CheckStatusMap` runs at wiring so a connector with a gap never starts.

## Flows

Every command is claim, call, apply: a short transaction moves the row (conditional `UPDATE ... WHERE status = ? AND version = ?`), the connector is called outside any transaction, then a second transaction applies the mapped result.
Two confirms on one intent cannot both win; the loser sees `ErrInvalidTransition`.

- `Create` - idempotent on `(merchant, key)`; same body replays, different body is `ErrIdempotencyConflict`. `Confirm: true` runs the first attempt.
- `Confirm` - selects a connector through `ConnectorSelector` (default `FirstEnabledSelector`, ticket 06 replaces it), creates the attempt, calls `Authorize`.
- `Capture` - full or partial; partial is terminal.
- `Cancel` - local before any attempt, `Void` at the connector otherwise.
- `Refund` - keyed like `Create`; the refundable balance counts pending refunds.
- `Sync` - resolves a timed-out call; the mock answers by our attempt id when no connector id was stored.
- `HandleWebhook` - connector verifies authenticity, the switch records the event id and applies it; a repeat is `ErrWebhookReplay`, an out-of-order event is recorded and ignored.

## Ledger

The first time an attempt has money in (`charged`, `partial_charged`, `overpaid`) the switch posts one `payment` journal in the same transaction: debit `connector/<code>` asset, credit `member/<merchant>` liability, key `switch.payment.<attempt_id>`.
A successful refund posts the reverse as a `refund` journal keyed `switch.refund.<refund_id>`.
Exactly once is guaranteed by the shared transaction and the ledger's own idempotency key.

## Config

`SWITCH_CONNECTORS` (default `mock,chaindeposit`) and `SWITCH_MOCK_WEBHOOK_SECRET`; see `internal/modules/paymentswitch.go`.
The mock refuses to wire in staging and production.

## Routes

`internal/api/routes_paymentswitch.go`: `POST /api/v2/payments`, `GET /api/v2/payments/:id` (`?sync=true` syncs first), `POST .../confirm`, `/capture`, `/cancel`, `/refunds`, and the public `POST /api/v2/webhooks/:connector`.
Bodies are snake_case, amounts are decimal strings, success is `{"payment": ...}` or `{"refund": ...}`, errors are `{"error": {"code", "message"}}`.
