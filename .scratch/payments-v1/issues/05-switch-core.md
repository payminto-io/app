# 05 Switch core: intents, attempts, connector interface, status map

Status: done
Owner: Principal engineer (Fable)
Blocked by: 01

## Goal
`backend/internal/switch/`: `payment_intents` and `payment_attempts` (Hyperswitch shape), a `Connector` interface (`Authorize`, `Capture`, `Void`, `Refund`, `Sync`, `VerifyWebhook`), a status map table from connector status to our vocabulary (requires_payment_method, requires_action, processing, succeeded, failed, cancelled, partially_captured), and a mock connector. Payminto's crypto deposit flow is wrapped as a connector so a USDC payment is an attempt like any other.

## Acceptance
State machine tests for every transition; idempotency on intent creation; mock connector drives a full authorize-capture-refund; ledger journals posted on succeeded, refunded.

## Comments

- 2026-10-07 (Principal, Fable): done on branch `switch`. Commits `19fd9e3` (connector port, registry, vocabularies and exhaustive transition tables), `4429bc2` (scriptable mock connector, per-connector status maps), `4a50b6b` (intents/attempts/refunds service: idempotent create, claim-then-call flows, ledger posting, webhook replay guard), `f4dc450` (conformance suite, chaindeposit provider over Payminto's deposit flow), `6171709` (migration `2026100703_switch_intents_attempts`, config, module wiring, v2 routes), `63adcae` (Postgres integration tests, READMEs). Full report: `.superpowers/switch-report.md`. Deviations: package is `paymentswitch` (`switch` is a keyword); the wiring line is in `cmd/server/main.go` because `service/registry.go` cannot import `modules` without a cycle; `RegisterPaymentSwitchRoutes` takes the auth middleware as a third argument; two attempt statuses (`partially_paid`, `overpaid`) added for chain deposits; `capture_failed`/`void_failed` keep the intent in `requires_capture`.

