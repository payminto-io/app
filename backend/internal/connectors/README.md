# connectors

Slot module for payment processors.
`port.go` is the only file other modules import; providers live in subfolders and register in the `Registry` by code.
Selection is configuration (`SWITCH_CONNECTORS`), never an import change.

## Port

`Connector`: `Code`, `Capabilities`, `Authorize`, `Capture`, `Void`, `Refund`, `Sync`, `VerifyWebhook`.

Contract:

- Statuses are returned raw (`RawStatus`), exactly as the provider says them; the switch maps them. `Capabilities.RawStatuses` must list every status the provider can report, and the switch verifies the map covers them at wiring.
- An operation the provider cannot do returns `ErrUnsupported`; the capability flags say so up front.
- Only `ErrDeclined` is definitive (the provider refused, nothing happened). Every other error, `ErrTimeout` included, means "outcome unknown": the switch keeps the operation in flight and calls `Sync` or `SyncRefund`, passing our attempt or refund id as well as any connector id, and retries with the same `IdempotencyKey`.
- `Capture`, `Void` and `Refund` honour `IdempotencyKey`: a repeat with the same key returns the first outcome.
- `VerifyWebhook(ctx, headers, body)` checks authenticity (signature and a timestamp window, `ErrWebhookStale` outside it) and decodes the event. It must return a stable, provider-unique `EventID`; replay protection is the switch's job, which persists `(code, event_id)` and answers a repeat as ignored. A connector never dedupes itself.
- A received amount (chain deposits) names its asset with the ledger's chain-qualified code (`USDC.SOLANA`).
- No card data: `PaymentMethod` carries a token or reference and non-sensitive details only.

## Providers

- `mock/` - scriptable by payment method token: `success`, `decline`, `requires_action`, `timeout`, `timeout_then_decline`, `async`, and for the first later Capture/Void `capture_timeout_landed`, `capture_timeout_lost`, `capture_error_landed`, `capture_declined`, `void_timeout_landed`, `void_timeout_lost`; refund reason `async`, `fail`, `refund_timeout_landed`, `refund_timeout_lost`, `refund_error_landed`. Honours idempotency keys, counts calls per operation, signs webhooks with HMAC-SHA256 over timestamp and body (`X-Mock-Timestamp`, `X-Mock-Signature`, five-minute window). Runs with no keys.
- `chaindeposit/` - wraps Payminto's crypto deposit flow: `Authorize` opens a payment request with a deposit address (our attempt id travels as `invoice_id` in the same insert, so `Sync` finds it by attempt after a crash), the request is `open` (customer action) until the block processors confirm, `partially_filled`, `over_filled` and `cancelled_underpaid` are explicit statuses, `Void` cancels an open or partially filled request. Prices in USD because `payment_requests.amount_in_usd` does and only USD stablecoins are accepted until conversion lands; the received amount is reported in the token with its chain-qualified asset. No capture, refund or webhooks; state comes from `Sync`. `payminto.go` adapts the real services, `memory.go` is the test double, and both pass `RunBackendSuite`.

## Conformance

`conformance.Run(t, Harness)` is the suite every provider must pass; see `conformance/mock_test.go` and `chaindeposit/chaindeposit_test.go`.
Tests a provider cannot do are skipped by capability and the skip is visible in `go test -v`.
