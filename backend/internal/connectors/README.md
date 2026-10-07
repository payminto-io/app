# connectors

Slot module for payment processors.
`port.go` is the only file other modules import; providers live in subfolders and register in the `Registry` by code.
Selection is configuration (`SWITCH_CONNECTORS`), never an import change.

## Port

`Connector`: `Code`, `Capabilities`, `Authorize`, `Capture`, `Void`, `Refund`, `Sync`, `VerifyWebhook`.

Contract:

- Statuses are returned raw (`RawStatus`), exactly as the provider says them; the switch maps them. `Capabilities.RawStatuses` must list every status the provider can report, and the switch verifies the map covers them at wiring.
- An operation the provider cannot do returns `ErrUnsupported`; the capability flags say so up front.
- A call that reached the provider without a definitive answer returns `ErrTimeout`; the switch keeps the attempt pending and calls `Sync`, passing our attempt id as well as any connector id.
- `VerifyWebhook(ctx, headers, body)` checks authenticity and decodes the event. It must return a stable, provider-unique `EventID`; replay protection is the switch's job, which persists `(code, event_id)` and rejects a second delivery. A connector never dedupes itself.
- No card data: `PaymentMethod` carries a token or reference and non-sensitive details only.

## Providers

- `mock/` - scriptable by payment method token: `success`, `decline`, `requires_action`, `timeout`, `timeout_then_decline`, `async`; refund reason `async` or `fail`. Signs webhooks with HMAC-SHA256 over the body (`X-Mock-Signature`). Runs with no keys.
- `chaindeposit/` - wraps Payminto's crypto deposit flow: `Authorize` opens a payment request with a deposit address, the attempt stays `open` (pending) until the block processors confirm, `partially_filled` and `over_filled` are explicit statuses, `Void` cancels an open request. Prices in USD because `payment_requests.amount_in_usd` does. No capture, refund or webhooks; state comes from `Sync`. `payminto.go` adapts the real services, `memory.go` is the test double.

## Conformance

`conformance.Run(t, Harness)` is the suite every provider must pass; see `conformance/mock_test.go` and `chaindeposit/chaindeposit_test.go`.
Tests a provider cannot do are skipped by capability and the skip is visible in `go test -v`.
