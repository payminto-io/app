# Ticket 05 - switch core: report

Branch `switch`, worktree `gateway-wt-switch`.
Commits (oldest first): `19fd9e3`, `4429bc2`, `4a50b6b`, `f4dc450`, `6171709`, `63adcae`, plus the docs commit that adds this file and closes the ticket.

## Files

New module `backend/internal/paymentswitch/` (the Go keyword forces the name):

- `port.go` - intent, attempt and refund status vocabularies, `Intent`/`Attempt`/`Refund`/`View`, sentinel errors, `ConnectorSelector`, `MerchantConnectors`, `Ledger` ports.
- `statemachine.go` - explicit transition tables, terminal sets, `IntentStatusFor` (attempt -> intent derivation), `MoneyIn`.
- `status_map.go` - per-connector raw status -> vocabulary maps (`mock`, `chaindeposit`), `MapAttemptStatus`, `MapRefundStatus`, `RegisterStatusMap`, `CheckStatusMap`.
- `model.go` - GORM rows for the five tables, `JSONMap`, row -> value conversions.
- `service.go` - `Service`, options, `Create` (idempotent), `Get`, versioned saves, transition audit rows.
- `confirm.go` - `Confirm` (claim, authorize, apply), `FirstEnabledSelector`, `StaticMerchantConnectors`.
- `capture_cancel.go` - `Capture` (full and partial), `Cancel` (local or `Void`).
- `refund.go` - `Refund` (idempotent, refundable balance counts pending refunds).
- `sync_webhook.go` - `Sync` and `HandleWebhook` with the replay guard.
- `apply.go` - `applyAttempt` / `applyRefund` and the two ledger journals.
- `constraints.sql` / `constraints.go` - Postgres CHECKs and FKs, `Migrate` for dev/test.
- Tests: `statemachine_test.go` (every pair of every vocabulary), `status_map_test.go`, `service_test.go` (SQLite, mock connector, real ledger behind a recording port), `integration_test.go` (`//go:build integration`, testcontainer).

New slot module `backend/internal/connectors/`:

- `port.go` - `Connector` interface, `Capabilities` (incl. `RawStatuses`), request/response types, `WebhookEvent`, errors, `Lookup`.
- `registry.go` - `Registry` (register by code, no duplicates, must declare raw statuses).
- `mock/` - scriptable provider: success, decline, requires_action, timeout (then success or decline), async, refunds immediate/async/fail, HMAC-signed webhooks, `Settle`, `SignWebhook`, lookup by our attempt id for post-timeout `Sync`.
- `conformance/` - `Run(t, Harness)` suite; `mock_test.go` runs it against the mock.
- `chaindeposit/` - provider over Payminto's deposit flow: `Backend` port, `PaymintoBackend` (adapts `PaymentService.CreatePayment`, `PaymentRepository`, `DepositRepository`; no Payminto file changed), `MemoryBackend` test double, conformance and under/over-payment tests.

Changed:

- `backend/internal/database/migrations/2026100703_switch_intents_attempts.up.sql` - checksummed migration (tables, indexes, the same constraint block), registered in `migrations.go`; `migrations_integration_test.go` now expects three migrations.
- `backend/internal/database/startup.go` - `MigrateExpandSchema` also runs `paymentswitch.Migrate` (same stance as the ledger: outside the manifest). README line.
- `backend/internal/config/config.go` - `Switch` section: `SWITCH_CONNECTORS` (default `mock,chaindeposit`), `SWITCH_MOCK_WEBHOOK_SECRET`; `envCSVDefault` helper.
- `backend/internal/modules/paymentswitch.go` - `WirePaymentSwitch(Deps)`: builds the registry from config, refuses the mock in staging/production, `CheckStatusMap` per connector, default selector. `paymentswitch_test.go`.
- `backend/internal/api/routes_paymentswitch.go` - `RegisterPaymentSwitchRoutes(rg, m, auth)`; `router.go` mounts it under `/api/v2` when `RouterConfig.PaymentSwitch` is set. `routes_paymentswitch_test.go`.
- `backend/cmd/server/main.go` - the one wiring call.

## Schema

`switch_payment_intents (id text pk, merchant_id, platform_id, idempotency_key, request_hash char(64), status, amount numeric(38,18), asset, amount_captured, amount_refunded, capture_method, payment_method_type, payment_method jsonb, connector_code, active_attempt_id, description, return_url, metadata jsonb, next_action jsonb, last_error_code, last_error_message, version, created_at, updated_at)` - unique `(merchant_id, idempotency_key)`; CHECKs on status, capture method, amounts (`amount_refunded <= amount_captured`), asset shape, hash shape.

`switch_payment_attempts (id, intent_id fk, merchant_id, connector_code, status, raw_status, amount, asset, amount_captured, amount_received, connector_transaction_id, selection_reason, error_code, error_message, next_action, version, ...)` - unique `(connector_code, connector_transaction_id)`; status CHECK.

`switch_refunds (id, intent_id fk, attempt_id fk, merchant_id, connector_code, idempotency_key, request_hash, status, raw_status, amount, asset, connector_refund_id, reason, error_*, version, ...)` - unique `(merchant_id, idempotency_key)` and `(connector_code, connector_refund_id)`.

`switch_webhook_events (id, connector_code, event_id, received_at)` - unique `(connector_code, event_id)`.

`switch_status_transitions (id, entity, entity_id, from_status, to_status, reason, created_at)`.

## Decisions

- **Vocabulary.** Intent statuses are the nine from the brief. Attempt statuses are modelled on Hyperswitch's `AttemptStatus`, trimmed to what the switch can honestly assert, plus `partially_paid` and `overpaid` for chain deposits that arrive short or over (the brief asked for explicit statuses). `IntentStatusFor` deviates from Hyperswitch in two places, both documented in code: `capture_failed` and `void_failed` keep the authorization and leave the intent in `requires_capture` rather than `failed`.
- **Claim, call, apply.** No connector call runs inside a database transaction. The claim is a conditional `UPDATE ... WHERE status = ? AND version = ?`; on Postgres the apply also takes `FOR UPDATE`. Twelve concurrent confirms on one intent produce one attempt and one authorization at the connector (`TestIntegration_ConcurrentConfirm_OnlyOneAttemptWins`).
- **Idempotency as in the ledger.** `(merchant_id, idempotency_key)` unique, canonical request hash stored, `INSERT ... ON CONFLICT DO NOTHING` then re-read; same body replays (a create+confirm replay runs no second attempt), different body is `ErrIdempotencyConflict`. An empty key gets a generated one so the column is never null. Refunds use the same scheme.
- **Exactly-once ledger posting.** The payment journal is posted inside the same transaction that moves the attempt into a money-in status, keyed `switch.payment.<attempt_id>`; refunds likewise with `switch.refund.<refund_id>`. Tests assert through a recording wrapper on the ledger port and count `ledger_journals` rows. Ten concurrent identical webhook deliveries post once.
- **Webhook replay contract.** The connector verifies authenticity and returns the provider's event id; the switch persists `(code, event_id)` and rejects a repeat with `ErrWebhookReplay` (HTTP 409). A failed delivery (bad signature, unknown transaction) is not recorded, so the provider's retry gets a fresh chance. An out-of-order event (e.g. `authorized` after `captured`) is recorded and ignored rather than retried forever.
- **Status map ownership.** Maps live in `paymentswitch/status_map.go` keyed by connector code and use raw string literals, so `paymentswitch` imports nothing from any provider folder (MODULES.md rule 1). `Capabilities.RawStatuses` lets `CheckStatusMap` prove coverage at wiring; the conformance suite checks every status a provider reports is declared.
- **Merchant identity.** `merchant_id` is the member id and `platform_id` the external platform, both from the existing auth middleware. The ledger liability account is `member/<merchant_id>`, matching the ledger report's follow-up 2.
- **Chain deposits price in USD** because `payment_requests.amount_in_usd` does; an intent in another asset is refused rather than converted silently. Payminto's finalizer compares raw deposit amounts with the USD amount (its own `TODO(phase-I)`); the connector reports what it finds and does not paper over that.
- **Wiring line in `main.go`, not `registry.go`.** `modules` imports `service` (for the chaindeposit backend), so `service/registry.go` cannot import `modules` without a cycle. The one line lives in `cmd/server/main.go`.
- **Routes take the auth middleware as a third argument.** `RegisterPaymentSwitchRoutes(rg, m, auth)`: the module knows nothing about auth and `router.go` passes `middleware.JWTOrAPIKey(cfg.AuthSvc)`. The webhook route is public and verified by the connector.
- **Mock refused in deployment.** `WirePaymentSwitch` returns `ErrMockInDeployment` for staging and production (MODULES.md rule 6).

## Test output

Unit (`cd backend && go build ./... && go vet ./... && go test ./...`): every package `ok`.
`internal/paymentswitch`: transition tables (72 intent pairs, 210 attempt pairs, refunds), status maps, and 22 service tests: capture-now, authorize+capture, partial capture, void, cancel before attempt, refund full/partial/pending-then-webhook/failed, decline then retry, requires_action then webhook (plus replay and out-of-order), timeout then sync (success and decline), async then webhook then capture, second confirm loses, merchant scoping, selector.
`internal/connectors/...`: mock unit tests, conformance for mock (all subtests run) and chaindeposit (settled-sync, declined, manual capture, refund, webhook skipped by capability, visibly), chaindeposit under/over payment, Payminto backend adapter.
`internal/modules`, `internal/api` (route tests over the real module on SQLite): `ok`.

Integration (`make test-integration`, Colima Docker): every package `ok` (`internal/database` 402s, `internal/ledger` 156s, `internal/paymentswitch` 168s, `internal/paymentlifecycle/postgres` 536s).
`internal/paymentswitch`: schema converges from AutoMigrate then `ApplyMigrations` (constraints, indexes present; out-of-vocabulary status rejected by the database), 18-decimal amounts exact with the merchant ledger balance matching, 12 concurrent confirms -> 1 attempt, 10 concurrent webhook deliveries -> 1 journal, 8 concurrent refunds of 60 on 100 -> 1 succeeds.

## Pre-existing issues seen, not touched

- `gofmt -l` still lists the ~31 files the ledger report named; none are mine.
- `PaymentRepository.FinalizeFromConfirmedDeposits` compares raw deposit amounts against `amount_in_usd` (its own TODO). The chaindeposit connector inherits that until conversion (ticket 10) lands.

## Follow-ups

1. Ticket 06 replaces `FirstEnabledSelector` and `StaticMerchantConnectors` with the routing engine and per-merchant connector configuration; `Attempt.SelectionReason` is already recorded for its decisions.
2. Ticket 07's card connector registers a status map via `RegisterStatusMap` (or a built-in entry) and runs `conformance.Run`.
3. A scheduler that calls `Sync` on attempts left `pending` is not part of this ticket; today `GET /v2/payments/:id?sync=true` and the block processors drive it.
4. Fee snapshot on the intent (ticket 02) and the per-intent settlement preference (ticket 11) are columns to add, additively.
5. The chaindeposit connector cannot resolve a payment by our attempt id because Payminto generates the reference; an `OpenPayment` failure after Payminto committed the row would need a reconciliation pass. Today the call is local and transactional enough that this has not happened in tests.
