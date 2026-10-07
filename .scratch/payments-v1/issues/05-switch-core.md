# 05 Switch core: intents, attempts, connector interface, status map

Status: ready-for-agent
Owner: Principal engineer (Fable)
Blocked by: 01

## Goal
`backend/internal/switch/`: `payment_intents` and `payment_attempts` (Hyperswitch shape), a `Connector` interface (`Authorize`, `Capture`, `Void`, `Refund`, `Sync`, `VerifyWebhook`), a status map table from connector status to our vocabulary (requires_payment_method, requires_action, processing, succeeded, failed, cancelled, partially_captured), and a mock connector. Payminto's crypto deposit flow is wrapped as a connector so a USDC payment is an attempt like any other.

## Acceptance
State machine tests for every transition; idempotency on intent creation; mock connector drives a full authorize-capture-refund; ledger journals posted on succeeded, refunded.
