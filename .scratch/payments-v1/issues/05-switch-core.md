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
- 2026-10-07 (Principal, Fable): fix round 1 against `.superpowers/switch-review.md`, all 2 critical, 11 important and 10 minor items addressed, each reproduced by a failing test first. Commits `4623cde` (merge ledger), `08be93c` (providers: evidence-first state machine, error contract, SyncRefund, idempotent mock, chaindeposit attempt-id lookup, shared backend suite), `dcfad04` (switch core: exclusive claims, unknown outcomes, anomalies, claimed capture amounts, chain deposits by delta in the received asset, reconciler, lock order, events, 200-ignored replays), `5e63177` (merge ledger round 2), `4db96cf` (claim lock order), plus the closing docs commit. Report: `.superpowers/switch-fix-1-report.md`. Not fixed by design: no invented expiry for partially paid deposits (Cancel or the reconciler's anomaly are the exits), chain deposit refunds are ticket 11, `Sync` not-found never marks an attempt failed.

