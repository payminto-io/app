# Ticket 05 switch core - fix round 1

Branch `switch`. Review: `.superpowers/switch-review.md`. Every item was reproduced by a test that failed before the fix (the I11 list, plus the scenarios named under each item below).

Commits this round (oldest first):

- `4623cde` merge `ledger` (sealed journals, chain-qualified assets, `Record*In`); one README conflict resolved, `Balances` assertion moved to natural balances.
- `08be93c` providers: evidence-first state machine, connector error contract, `SyncRefund`, idempotent mock with scripted timeouts and a webhook window, chaindeposit attempt-id lookup and `cancelled_underpaid`, shared backend suite.
- `dcfad04` switch core: exclusive claims, outcome-unknown handling with same-key retries, evidence-first apply with anomalies, claimed capture amounts, chain deposits posted by delta in the received asset, refund sync and reconciler worker, lock order, domain events, 200-ignored webhook replays.
- `5e63177` merge `ledger` round 2 (trigger search_path, seal by txid and start time, assets widened to 32, migrator role handling); no conflicts, no API change that reaches the switch.
- `4db96cf` the I10 test caught the last inversion: `claimAttempt` updated the attempt before locking the intent; it now locks the intent first, and the webhook race test counts 200-ignored replays.
- the closing commit (this report and the ticket comment).

## Critical

**C1 - connector evidence wins.** `statemachine.go`: `capture_failed -> charged | partial_charged`, `void_failed -> charged | partial_charged | voided`, `capture_initiated -> authorized | voided`, `void_initiated -> authorized | charged | partial_charged | underpaid`. `applyAttempt` applies any allowed edge from Sync or a webhook and posts the journal exactly once (`switch.payment.<attempt>` keyed; `settleAmount` returns the already-posted amount when the attempt is already money-in). "Already past" is now only for evidence that contradicts a terminal attempt or refund: it is written to `switch_anomalies` as `evidence_after_terminal`, logged at error level, and answered as ignored; a strictly older state is ignored without an anomaly. Tests: `TestCapture_ErrorAfterLandingIsRecoveredBySync` (capture landed, 502 on the way back, Sync -> charged, one journal, a second Sync posts nothing), `TestCapture_DeclinedThenConnectorEvidenceOfCaptureWins` (scenario A), `TestWebhook_CapturedAfterVoidIsAnAnomaly` (M10), `TestFlow_RequiresActionThenWebhookSuccess` (authorized after captured -> anomaly).

**C2 - exclusive claims.** `claimAttempt` is a conditional `UPDATE switch_payment_attempts ... WHERE id = ? AND status = ? AND version = ?` into `capture_initiated` or `void_initiated`, with the intent moved in the same transaction; a same-status transition is a conflict for claims (`ErrInvalidTransition`). Refund's claim is the `initiated` row insert under the intent lock. Tests: `TestCapture_ClaimIsExclusiveAndRetriesReuseTheKey` (unit), `TestIntegration_ConcurrentCapture_ExactlyOneConnectorCall` (10 racers with different amounts on Postgres: one wins, `mock.Calls("capture") == 1`, one journal, captured equals the claimed amount).

## Important

**I1** - the edges above plus `TestCapture_TimeoutLostThenSyncRestoresAuthorizationAndCaptureSucceeds`, `TestCapture_TimeoutLandedThenSyncCharges`, `TestCancel_VoidTimeoutLostThenSyncThenCancelAgain`, `TestCancel_VoidTimeoutLandedThenRetryReusesKey`. The mock gained `capture_timeout_landed`, `capture_timeout_lost`, `capture_error_landed`, `capture_declined`, `void_timeout_landed`, `void_timeout_lost` (chosen by the authorize token, consumed by the first Capture/Void) and `refund_timeout_landed`, `refund_timeout_lost`, `refund_error_landed` (by refund reason).

**I2** - `connectors/port.go` documents the contract: only `ErrDeclined` is definitive (`connectors.Definitive`); `ErrUnsupported`/`ErrInvalidRequest` are raised before any call; every other error, `ErrTimeout`, context deadlines and raw errors included, is "outcome unknown". `authorizeOutcome`, `Capture`, `Cancel` and `sendRefund` keep the row in flight (`pending`, `capture_initiated`, `void_initiated`, refund `initiated`) via `markUnknown`/`markRefundUnknown`, which store a typed code and a redacted message, write the raw error to the transition audit as a note, and schedule the reconciler. `Confirm` refuses while an attempt is in flight (no second attempt). Retries: `Capture` on `capture_initiated` re-sends with `attempt.ID + ".capture"` and the stored `amount_to_capture` (a different amount is refused), `Cancel` on `void_initiated` re-sends `attempt.ID + ".void"`, a replayed refund key whose row is `initiated` re-sends with `refund.ID`. The mock honours keys and replays the landed result for a repeat. Tests: `TestConfirm_UntypedErrorIsUnknownNotFailed` (connection reset via the `flaky` wrapper: pending, no failed event, raw text only in the audit, Sync resolves, one authorize call), `TestFlow_RefundUnknownOutcomeHoldsBalanceAndResolves`, `TestFlow_RefundDeclinedAtConnector`, mock `TestIdempotencyKeys_ReplayFirstOutcomeEvenAfterAnUnansweredCall`.

**I3** - `HandleWebhook` returns `WebhookResult{Ignored: true, IgnoreWhy: "replay"}` with no error; the route answers 200 `{"ignored": true, "reason": "replay"}`. Out-of-order events and evidence after terminal states are also 200 ignored. Tests in `service_test.go` and `routes_paymentswitch_test.go`.

**I4** - `Connector.SyncRefund` with `Capabilities.RefundSync`; `Sync` resolves every `initiated`/`pending` refund of the intent; `Reconciler` (`reconciler.go`, registered in `main.go` with the worker manager as `switch_reconciler`) syncs attempts in `pending`/`capture_initiated`/`void_initiated` and refunds in `initiated`/`pending` whose `next_sync_at` is due, with doubling backoff per row (`sync_count`, `next_sync_at`, `last_synced_at`), and past `MaxAge` (24h) records a `stale_in_flight` anomaly and logs at error level without touching the status. Conformance covers `SyncRefund` (by our id and by the connector's). Tests: `TestFlow_RefundPendingResolvedBySync`, `TestReconciler_SyncsInFlightRowsAndFlagsStaleOnes`.

**I5** - `applyAttempt` keeps `next_action` when the new status still lets the customer act (`pending`, `authentication_pending`, `partially_paid`) and the update carries none; a same-status update only replaces it when the connector supplies one. Chaindeposit `Sync` now returns the address for open and partially filled requests; the mock returns the redirect while action is required. Tests: pending webhook and Sync in `TestFlow_RequiresActionThenWebhookSuccess`, Sync while open and while partially paid in `TestChainDeposit_PartialThenFilledPostsByDeltaInTheReceivedAsset`.

**I6** - `amount_to_capture` is stored at claim time (the full amount for automatic capture at confirm, the requested amount at a manual capture claim). `settleAmount` uses the connector's amount, else the claimed amount, else refuses with `ErrAmountUnknown` and an `amount_unknown` anomaly. Test: `TestWebhook_CapturedWithoutAmountUsesTheClaimedCaptureAmount` (40 claimed, amount-less captured webhook posts 40; with no claim and no amount nothing is posted and the anomaly is recorded).

**I7** - chaindeposit accepts only `StableUSD` (`USDC`, `USDT`, `DAI`, `PYUSD`) and reports `AmountReceived` with `ReceivedAsset = <CURRENCY>.<CHAIN>` (`chaindeposit.ReceivedAsset`, same shape as the ledger's `chainAsset`). The switch posts deposits with `postDeposit`: debit `platform/crypto_assets` in that asset, credit `member/<merchant>` liability in that asset, metadata `priced_asset`/`priced_amount`/`received_asset`/`cumulative_received`. `attempt.received_asset` and `amount_received` are new columns; the route omits `amount_received` unless reported (M2). Test: `TestChainDeposit_PartialThenFilledPostsByDeltaInTheReceivedAsset` asserts the accounts, asset and merchant balance in `USDC.ETH`.

**I8** - `OpenRequest.AttemptID` travels as `CreatePaymentInput.InvoiceID`, which `PaymentService.CreatePayment` persists in the same insert; `PaymintoBackend.PaymentStatusByAttempt` queries `payment_requests.invoice_id`; the memory backend picks its own reference like Payminto and indexes by attempt. `chaindeposit.RunBackendSuite` runs against both (`TestMemoryBackendSuite`, `TestPaymintoBackendSuite` on SQLite with a `dbOpener` that inserts the way `CreatePayment` does and confirms deposits through Payminto's own `FinalizeFromConfirmedDeposits`), asserting the reference is the backend's, lookup by attempt works, deposits move state like the finalizer, cancel rules. `TestPaymintoBackend_OpenRequiresADepositAddressAndTheConnectorUsesIt` syncs by attempt id through the Payminto backend.

**I9** - `partially_paid` is money-in: each Sync posts the received delta (keyed `switch.payment.<attempt>.<cumulative>`). `Cancel` on `partially_paid` claims `void_initiated` with the reason "received funds await refund", chaindeposit `Void` cancels `PARTIALLY_FILLED` and reports `cancelled_underpaid`, mapped to the terminal attempt status `underpaid` and the terminal intent status `partially_paid`, with `shortfall` and `received_asset` in the intent metadata. Expiry: Payminto's own expiry only cancels `OPEN` requests and its finalizer still fills a `PARTIALLY_FILLED` one, so the switch does not invent an expiry; a partially paid attempt past `MaxAge` is flagged by the reconciler and closed by `Cancel`. Tests: `TestChainDeposit_CancelWhilePartiallyPaidClosesShortWithFundsOnTheBooks`, `TestChainDeposit_OverpaidPostsWhatArrived`, `TestChainDeposit_CancelWhileOpen`.

**I10** - one order everywhere: intent, then attempt, then refund. `applyToActiveAttempt`, `applyToRefund`, `markUnknown`, `Refund` (intent lock before the sum) and both webhook paths (read the attempt/refund unlocked, then lock in order) follow it. Test: `TestIntegration_CaptureApplyAndWebhookNeverDeadlock` (15 rounds of a capture apply racing the captured webhook on Postgres; no 40P01, one journal per intent).

**I11** - every listed gap has a test above; `grep` now finds `partially_paid`, `charged` and `overpaid` driven through the service with ledger assertions.

## Minor

- **M1** `validateAmount`: more than 18 decimals or `>= 1e20` is `ErrInvalid`, for create, capture and refund amounts.
- **M2** `amount_received` is a pointer with `omitempty`; `received_asset` and `amount_to_capture` added to the attempt DTO.
- **M3** webhook kinds outside `payment`/`refund` are `ErrWebhookMalformed`.
- **M4** `Create` returns the created intent alongside a confirm error; the route puts `payment_id` in the error envelope; a replay of a create with `confirm_requested` and no attempt re-runs the confirm. Tests in both layers.
- **M5** a `captured` response with a zero amount no longer falls back to the request; it goes through I6's rule.
- **M6** `open` maps to `authentication_pending` (intent `requires_action`) with the `pay_to_address` next action.
- **M7** merchants see typed codes (`declined`, `connector_error`, `connector_timeout`, `not_found_at_connector`) and neutral messages; raw connector and backend text goes to `switch_status_transitions.reason` and the log. `invalid_state`/`amount_exceeds`/`invalid_request` messages are the switch's own wording.
- **M8** the mock signs over `timestamp.body` with a five-minute window (`ErrWebhookStale`); the conformance suite requires every webhook-capable provider to supply a stale delivery and refuse it.
- **M9** `switch.payment.succeeded.v1`, `switch.payment.failed.v1`, `switch.refund.succeeded.v1` are emitted after commit through `service.EventEmitterService.EmitDomain` (one additive method) via `modules.EmitterEvents`; `NoEvents` for tests and tools. Tests assert one event per transition.
- **M10** see C1.

## Schema

Migration `2026100703` regenerated in place (unapplied outside test containers, same reasoning as the ledger round): `switch_payment_intents.confirm_requested`; `switch_payment_attempts.amount_to_capture`, `amount_received` (nullable), `received_asset`, `sync_count`, `next_sync_at`, `last_synced_at`; the same sync columns on `switch_refunds`; new `switch_anomalies`; CHECKs extended with `partially_paid`, `underpaid`, refund `initiated`. `constraints.sql` matches and the convergence test asserts the anomalies constraint.

## Test summary

- `cd backend && go build ./... && go vet ./... && go test ./...`: every package ok.
- `make test-integration` (Colima Docker, after the ledger round 2 merge): every package ok, exit 0 (`internal/database` 121s, `internal/ledger` 75s, `internal/paymentlifecycle/postgres` 125s, `internal/paymentswitch` 60s). The first run of this round failed exactly where the new I10 test aims: `claimAttempt` updated the attempt row before locking the intent (40P01 in `TestIntegration_CaptureApplyAndWebhookNeverDeadlock`); fixed in `4db96cf`, rerun green.
- New or rewritten tests this round: 11 state machine edge updates, 3 mock, 3 conformance subtests, backend suite (4 subtests x 2 backends) + 4 chaindeposit, 21 service scenarios, 2 route, 2 integration.

## Not fixed, with reason

- Expiry of a partially paid deposit is not invented by the switch (I9): Payminto has no expiry for `PARTIALLY_FILLED` and would still fill it; the honest exits are the merchant's `Cancel` and the reconciler's `stale_in_flight` anomaly. A connector-side expiry needs Payminto's finalizer to agree, which is outside this ticket.
- Chain deposit refunds (`underpaid` funds awaiting refund) are ticket 11; the switch records the state and refuses `Refund` for chaindeposit.
- `Sync` returning `ErrNotFound` keeps the attempt in flight and only advances the backoff; it never marks the attempt failed (the review's I8 showed that inference was false in production). The reconciler's max age surfaces a transaction a provider never admits to.
