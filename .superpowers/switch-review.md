# Ticket 05 switch core: review

Branch `switch`, worktree `gateway-wt-switch`, HEAD `4a267f9`.
Reviewed against `.scratch/payments-v1/issues/05-switch-core.md`, `CLAUDE.md`, `docs/architecture/MODULES.md`, the implementer report and Hyperswitch `crates/common_enums` (attempt to intent derivation, terminal sets).
Read-only review; nothing edited or committed.

## Verdicts

- Spec: partially met.
  All four acceptance items have code and tests (transition tables, idempotent create, mock authorize-capture-refund, ledger journals on succeeded and refunded).
  The state machine is not sound: a capture that lands at the connector but is not applied on our side has no recovery edge, so captured money can be lost from the ledger (C1), and the capture claim is not exclusive (C2).
- Quality: not mergeable until C1 and C2 are fixed.
  I1, I5 and I6 should ship in the same change because they are the same family (a Sync or webhook that cannot repair, or actively damages, the row).
  The code is well structured, the module boundary is respected (`paymentswitch` imports only `connectors/port.go` and `ledger`), and Payminto's own services are untouched by the switch commits.

Checks run from the worktree:

- `cd backend && go build ./... && go vet ./... && go test ./...`: every package `ok`.
- `make test-integration` (Colima): every package that finished while reviewing is `ok`; see the end of this file for the final state.

Counts: Critical 2, Important 11, Minor 10.

Accepted deviations from the brief were not re-flagged (wiring in `main.go`, auth middleware as a route argument, public connector-verified webhook route, `partially_paid`/`overpaid`, `capture_failed`/`void_failed` keep `requires_capture`, USD-only chain deposits, merchant id = member id, no scheduler, routing deferred).

## Critical

### C1. A capture that succeeds at the connector but fails to apply here is unrecoverable, and the money never reaches the ledger

- `backend/internal/paymentswitch/statemachine.go:29` (`capture_failed` edges), `capture_cancel.go:45-71`, `sync_webhook.go:60-63`.
- Scenario A (sequential): `Capture` calls the connector, the capture lands, the response is lost or the connector returns a non-timeout error after processing (5xx, reset after send, parse failure).
  `capture_cancel.go:69` maps that to `capture_failed`; the intent returns to `requires_capture`.
  The merchant retries `Capture`: the connector reports the transaction already captured, the mock returns `ErrInvalidRequest`, the switch writes `capture_failed` again.
  `Sync` loads the attempt (`capture_failed` is not terminal), the connector answers `captured`, `MapAttemptStatus` gives `charged`, `transitionAttempt(capture_failed, charged)` fails, and `sync_webhook.go:60-63` swallows the error as "we are already past this".
  A `captured` webhook hits the same wall in `applyPaymentWebhook` and is recorded as ignored.
  Result: funds captured at the connector, `amount_captured = 0`, no payment journal, intent stuck in `requires_capture` forever.
- Scenario B (race, reproducible with the mock): two `Capture` calls on one `requires_capture` intent.
  Both read `requires_capture` before either claim commits; both pass the claim (see C2); A captures at the connector, B's capture errors, B applies `capture_failed` (allowed from `capture_initiated`), then A applies `charged` and is refused by `capture_failed -> charged`.
  Same end state as A, with A's caller getting a 409 for a capture that succeeded.
- Fix: a failed capture is a hypothesis, not a fact.
  Add `capture_failed -> charged | partial_charged` (and the same for `void_failed -> voided`, which already exists) so Sync and webhooks can apply connector evidence.
  Make `Sync` surface, not swallow, a transition it cannot apply when the connector reports money in (log at error level and keep the row); silent ignore is only right for a strictly older state.
  Add a test: capture, connector error after capture, Sync reports captured, assert `charged` and one payment journal.

### C2. The Capture and Cancel claims are not exclusive

- `backend/internal/paymentswitch/statemachine.go:93-96` (`from == to` returns nil), `apply.go:28`, `capture_cancel.go:45`, `capture_cancel.go:118-123`.
- The claim for capture is `applyToActiveAttempt(... AttemptCaptureInitiated)`.
  Because a same-status transition is a no-op, a second concurrent `Capture` whose pre-read saw `requires_capture` also "claims" `capture_initiated` successfully and calls `conn.Capture` a second time.
  The README promises "Two confirms on one intent cannot both win"; that is true for confirm (`confirm.go:64-71` is a conditional UPDATE on status and version) and false for capture.
  With a connector that does not honor `IdempotencyKey`, or when the two calls carry different amounts under the same key `attempt.ID + ".capture"`, this is a double capture.
  With the mock it degrades to C1.
- Fix: claim capture with a conditional UPDATE on the attempt row (`WHERE id = ? AND status = 'authorized' AND version = ?`) exactly like confirm, and return `ErrInvalidTransition` when zero rows change.
  Do the same for void (`capture_cancel.go:118`) when the attempt is claimable; keep the direct call only for the explicit retry-after-timeout case where the attempt is already `void_initiated`.
  Add the concurrent-capture integration test next to `TestIntegration_ConcurrentConfirm_OnlyOneAttemptWins`.

## Important

### I1. Capture or void timeout followed by "authorized" from Sync strands the intent in `processing`

- `statemachine.go:28,31`, `capture_cancel.go:66-67,138-139`, `sync_webhook.go:60-63`.
- `Capture` times out; the attempt stays `capture_initiated`, the intent `processing`.
  The capture never landed, so Sync reports `authorized`.
  `capture_initiated -> authorized` is not an edge; Sync swallows it.
  `Capture` refuses (`intent.Status != requires_capture`), `Cancel` calls `Void` without a claim (`claimed == false`), the connector voids, and `capture_initiated -> voided` is refused too.
  Void timeout then Sync "authorized" has the same shape (`void_initiated -> authorized` missing), though a repeat `Cancel` happens to work there.
- Fix: allow `capture_initiated -> authorized` and `void_initiated -> authorized` as "the initiated operation did not land" (or map them to `capture_failed`/`void_failed` inside Sync), and allow `capture_initiated -> voided`.
  Test: capture timeout (the mock needs a `Capture` timeout scenario; it has none), Sync, assert `requires_capture`, then capture again.

### I2. Ambiguous connector errors are recorded as terminal `failure`, and the retry opens a new attempt with a new idempotency key

- `confirm.go:132-136` (`default` branch), `refund.go:142-144`.
- Only `connectors.ErrTimeout` is treated as "no definitive answer".
  A context deadline, a connection reset after the request was sent, or any error a provider adapter does not wrap becomes `failure`, the intent becomes `failed`, and `Confirm` is allowed again from `failed` (`confirm.go:116`).
  The new attempt carries a new `IdempotencyKey` (`attempt.ID`), so a provider that did authorize the first call authorizes again: two holds, or two charges under automatic capture.
  A late webhook for the first attempt is ignored as "no longer active" (`sync_webhook.go:135-137`), so the first charge never reaches the ledger.
  The refund branch is worse: a refund that went through but errored on the way back is marked `failed`, which frees the refundable balance (`refund.go:92`), and the merchant's retry refunds twice.
  CLAUDE.md: "No placeholders, no inferred statuses"; `failure` here is inferred.
- Fix: in the switch, treat `context.DeadlineExceeded`, `context.Canceled` and any error that is not `ErrInvalidRequest`/`ErrUnsupported`/`ErrNotFound` as `pending` (sync later), and keep `failure` for errors the connector classifies as definitive.
  Make the connector contract explicit in `port.go`: definitive errors are sentinel-wrapped; everything else means "unknown, sync".
  For refunds, an unknown outcome must stay `pending`.

### I3. A replayed webhook returns HTTP 409, which makes providers retry it forever

- `backend/internal/api/routes_paymentswitch.go:453-454`, `sync_webhook.go:100-102`.
- Every mainstream provider treats a non-2xx as "not delivered" and retries with backoff for hours or days.
  A duplicate delivery is the normal case for at-least-once webhooks, and this handler turns each one into a retry storm.
- Fix: map `ErrWebhookReplay` to 200 with `{"event_id": ..., "ignored": true, "reason": "replay"}`.
  Keep 401 for bad signatures and 404 for unknown transactions (those should be retried once we might know the transaction).

### I4. A refund that times out stays `pending` forever on a connector without refund webhooks, and holds the refundable balance

- `refund.go:140-141`, `connectors/port.go:200-209` (no refund sync), `sync_webhook.go:15` (Sync only covers the attempt).
- The refundable balance counts pending refunds (correct), but nothing ever resolves a pending refund except a refund webhook.
  The mock has webhooks; a card connector with sync but without refund webhooks (common) leaves the merchant unable to refund the remainder.
- Fix: add `SyncRefund(ctx, SyncRefundRequest)` to the port with a `RefundSync` capability, have `Sync` resolve pending refunds of the intent, and let the conformance suite cover it.

### I5. Sync and pending webhooks erase `next_action`

- `apply.go:49` (`attempt.NextAction = nextActionJSON(u.nextAction)` unconditionally), `apply.go:73`, `sync_webhook.go:54-58,134`, `chaindeposit.go:152-168` (Sync never returns `NextAction`), `mock.go:256-276` (same).
- `GET /v2/payments/:id?sync=true` on an open chain deposit (`open -> pending`, same status) deletes the deposit address from the attempt and the intent.
  A `pending` webhook for a 3DS payment deletes the redirect URL.
  The customer-facing instruction disappears while the customer still needs it.
- Fix: only replace `next_action` when the update carries one or when the new status is one where no action is possible (terminal, authorized, charged); keep it on same-status and pending updates.
  Test: open chain deposit, Sync while still open, assert `next_action.address` unchanged.

### I6. The requested capture amount is not persisted at claim, so a capture confirmed by webhook or Sync without an amount posts the full authorization

- `apply.go:51-59` (falls back to `attempt.Amount`), `capture_cancel.go:45` (claim carries no amount), `sync_webhook.go:134`.
- Partial capture of 40 on a 100 authorization: claim `capture_initiated`, connector call times out, the provider's `captured` webhook (or Sync) arrives without `amount_captured`.
  `captured` is `attempt.Amount` = 100; `intent.amount_captured = 100`; the ledger credits the merchant 100 for a 40 capture.
  The mock always reports the amount, so the tests cannot see this.
- Fix: store `amount_to_capture` on the attempt at claim time and use it as the fallback; if neither the claim nor the connector gives an amount, refuse to post rather than infer (Honesty rule).

### I7. The chaindeposit connector posts a token quantity to the ledger under asset `USD`, against a `connector/chaindeposit` asset account

- `chaindeposit.go:163-166` (`AmountReceived = st.Received`), `payminto.go:60-66` (sums `deposits.amount`, which is in the deposit token's unit; `models/payment.go:47`), `apply.go:57-58,64,97` (posts `amount_received` under `attempt.Asset = "USD"`).
- The accepted deviation is "chain deposits price in USD".
  What is new here is that the received amount is a raw token sum (Payminto's own `TODO(phase-I)` at `payment_repo_impl.go:213`) and the switch records it as USD on an asset account named after a connector, when the funds are actually on-chain in a token (the ledger has `OwnerChain` for that).
  For USDC the number happens to match; for anything else the ledger line is a unit error, and `over_filled` posts `received` verbatim.
- Fix until conversion (ticket 10) lands: allow-list the currency codes the connector will open (USD-pegged stablecoins) and refuse the rest explicitly; post `filled` at the intent amount (the finalizer's own definition of filled is `received == amount_in_usd`), and record the received token amount in metadata, not as the USD line.
  Decide the asset-side account with the ledger owner: `chain/<chain>` in the token asset plus an explicit conversion journal is the honest shape under "conversions are explicit ledger trades".

### I8. `PaymintoBackend.OpenPayment` drops `OpenRequest.Reference`; the memory backend keys on it, so the tests prove a lookup production cannot do

- `chaindeposit/payminto.go:32-44` (never passes `req.Reference`), `chaindeposit/memory.go:29-33` (uses it as the key), `chaindeposit.go:152-156` (Sync falls back to `AttemptID` as the reference), `sync_webhook.go:43-46`.
- With the memory backend, Sync by attempt id works, and the conformance suite passes.
  With Payminto, a crash between `CreatePayment` committing and the switch persisting the response leaves an `OPEN` payment request with a deposit address, and `Sync` by attempt id returns `ErrNotFound`, which `sync_webhook.go:45` turns into `failure: authorization never reached the connector`.
  That message is false, and the orphan request will claim any deposit sent to it.
  The report's follow-up 5 names this, but the test double hides it (recurring-defect class: fixtures indistinguishable from real behaviour).
- Fix: either carry our attempt id into Payminto (a nullable `payment_requests.switch_attempt_id` or the existing metadata) so `PaymentStatus` can resolve by it, or make `MemoryBackend` ignore `Reference` like production and let the conformance suite fail honestly.
  Minimum: `payminto_test.go` must assert that `OpenPayment` returns Payminto's reference, not the attempt id.

### I9. A partially paid deposit has money in custody with no ledger line, no terminal state, and no way to cancel

- `statemachine.go:118` (`partially_paid -> processing`), `statemachine.go:137-139` (`MoneyIn` excludes it), `chaindeposit.go:139-141` (Void refuses anything but `open`), `payment_repo_impl.go:206` (Payminto has no expiry state).
- A customer sends 40 of 100 and walks away.
  The 40 is confirmed on-chain, in our custody, and the ledger does not know.
  The intent stays `processing` forever, `Cancel` fails with `ErrInvalidRequest`, and the merchant has no number to act on.
  CLAUDE.md: "Every number shown is a ledger line or a connector or chain receipt" and "Only ledger writes balances".
- Fix: make `partially_paid` money-in for the received amount (post the journal, allow later increments to `charged`/`overpaid` as further journals keyed by deposit), and give it an exit: the chaindeposit Void should cancel a `PARTIALLY_FILLED` request too (Payminto's finalizer only moves `OPEN`/`PARTIALLY_FILLED`), mapping to a terminal attempt status with the received funds still on the books.

### I10. Lock order inversion between attempt apply and payment webhook

- `confirm.go:143-147` (`applyToActiveAttempt`: lock intent, then attempt), `sync_webhook.go:118-125` (`applyPaymentWebhook`: lock attempt, then intent).
- Under Postgres, a capture response apply and the provider's `captured` webhook for the same attempt can deadlock; Postgres aborts one with 40P01.
  If the loser is the capture apply, the caller gets a 503, the attempt stays `capture_initiated`, and recovery depends on a later Sync (which works only when the connector still reports `captured`, otherwise I1).
  The refund paths are consistent with each other (refund then intent) but not with the attempt paths.
- Fix: one lock order everywhere, intent first (it is the aggregate root), then attempt or refund; the webhook can read the attempt without a lock to find the intent id, then lock in order.
  Document the order in one comment at `applyToActiveAttempt`.

### I11. Test gaps around the failure paths the review focuses on

- `service_test.go`, `integration_test.go`, `mock/mock.go`.
- Not covered: concurrent captures (C2), a connector error after a successful capture then Sync (C1), capture or void timeout then Sync (I1; the mock has no capture/void timeout scenario), a non-timeout error from Authorize/Capture/Refund (I2), `next_action` after Sync (I5), a `captured` webhook without an amount after a partial capture (I6), and any chain deposit driven through the switch to `partially_paid`, `charged` or `overpaid` (I7, I9: the ledger posting on `overpaid` via `amount_received` is never executed; `grep` finds those statuses only in `statemachine_test.go` and `status_map_test.go`).
  The mock's `Capture` and `Refund` ignore `IdempotencyKey`, so the switch's retry-with-same-key assumption is never checked; the mock should record keys and return the first result for a repeat.
  The routes test and the module test wire `chaindeposit` with the memory backend only (see I8).
- Fix: add the scenarios above; add `ScenarioCaptureTimeout`/`ScenarioVoidTimeout` to the mock; run one service-level chain deposit flow against `MemoryBackend` asserting the journal lines.

## Minor

### M1. Amount scale is not validated; Postgres rounds to 18 decimals silently

- `service.go:124-132`, `routes_paymentswitch.go:342-347`.
- A 19-decimal amount is accepted, hashed into `request_hash` at full precision, posted to the ledger from the in-memory value, and stored rounded; a replay with the same body then matches a row whose amount differs from the hash's.
- Fix: reject `Amount.Exponent() < -18` and more than 38 significant digits in `validateMoney`, and in `Capture`/`Refund` amounts.

### M2. `amount_received: "0"` is emitted for every attempt, including card attempts where nothing was received

- `routes_paymentswitch.go:91,403`.
- A number not derived from data; omit (`omitempty` with a pointer) unless the connector reported it.

### M3. Webhook kind is not validated

- `sync_webhook.go:103-108`: anything that is not `refund` is applied as a payment event.
- Fix: refuse kinds outside `{payment, refund}` with `ErrWebhookMalformed`.

### M4. `Create` with `Confirm: true` loses the intent id when the confirm fails before the claim

- `service.go:239-241`.
- A selector error (`ErrNoConnector`) after the insert returns only the error; the client never learns the id, and a replay returns the intent unconfirmed with 201.
- Fix: return the intent alongside the error, or have the replay re-run confirm when the stored request asked for it and no attempt exists.

### M5. Capture infers the captured amount when the connector reports zero

- `capture_cancel.go:61-64`.
- `captured.IsZero()` falls back to the requested amount; a connector that returns `captured` with no amount should be an error, not a guess.

### M6. An open chain deposit is `pending`/`processing` although the customer still has to act

- `status_map.go:40`, `statemachine.go:118`.
- `requires_action` with the `pay_to_address` next action is the honest merchant-facing status and matches Hyperswitch's `RequiresCustomerAction`.

### M7. Raw connector and backend error strings are stored in `error_message` and returned to the merchant

- `confirm.go:135`, `capture_cancel.go:69`, `refund.go:143`.
- For chaindeposit the `Authorize` error is a local database or service error (`chaindeposit: open payment: ...`), exposed verbatim through `GET`.
- Fix: store the raw error in the transition reason or a log, and expose a classified code and a neutral message.

### M8. The mock's `VerifyWebhook` has no timestamp window and mutates connector state before the switch dedupes

- `mock/mock.go:305-338`; `port.go:197-199` promises a timestamp window.
- Harmless for tests, but the conformance suite should assert the window for real providers (a stale signed body must be refused), and the mock should not be the only provider that passes without one.

### M9. No domain events on `succeeded`, `failed`, `refunded`

- MODULES.md rule 9; not in the ticket's acceptance, but tickets 02 and 11 will need them and adding them later touches every apply path.

### M10. A `captured` webhook after a void is silently ignored

- `sync_webhook.go:139-143`, `statemachine.go:37` (`voided` terminal).
- When a void races a capture at the connector, money is captured with no ledger line and no signal; Hyperswitch has `VoidedPostCharge` for exactly this.
  At minimum, log and surface `ignored` reasons that carry money-in evidence.

## What is right and worth keeping

- Confirm's claim (`confirm.go:62-89`) is the correct shape: conditional UPDATE on status and version, attempt inserted in the same transaction, connector called outside it; the 12-racer integration test proves it.
- Ledger posting is in the same transaction as the status change, keyed per attempt and per refund; the 10-delivery webhook test and the 8-racer refund test prove exactly-once on Postgres.
- Signs and accounts for the card path are correct: payment debits `connector/<code>` asset and credits `member/<merchant>` liability; refund reverses both; journals balance per asset.
- Refund over-refund protection counts pending refunds under the intent row lock (`refund.go:87-103`); `amount_refunded <= amount_captured` is also a database CHECK.
- Merchant scoping: every command loads by `(id, merchant_id)`; webhooks derive the merchant from the attempt; idempotency keys are unique per `(merchant_id, key)` with a canonical request hash; a refund key is bound to its intent through the hash.
- Status maps use string literals so `paymentswitch` never imports a provider folder; `CheckStatusMap` at wiring and the conformance `declared` check close the unmapped-status gap from both sides.
- The switch commits change nothing under `internal/service`, `internal/repository` or `internal/models`; the chaindeposit adapter only calls existing methods.
- Decimal handling is `shopspring/decimal` end to end with string amounts on the wire and `numeric(38,18)` in every table; the integration test proves an 18-decimal capture round-trips and matches the ledger balance.

## Integration run

`make test-integration` from the worktree root (Colima Docker): every package `ok`, exit 0.
`internal/database` 384s, `internal/ledger` 252s, `internal/paymentswitch` 245s, `internal/paymentlifecycle/postgres` 470s.
Unit run (`go build ./... && go vet ./... && go test ./...`): every package `ok`, exit 0.
Green tests do not cover C1, C2, I1, I2, I5 or I6; see I11.
