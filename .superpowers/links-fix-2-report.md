# Links fix round 2 report

Branch `links`. Re-review: `.superpowers/links-rereview.md`.

## Commits

- `de68606` N1 to N5.
- `016080c` docs/OPERATIONS.md: client IPs behind a proxy, `TRUSTED_PROXIES` in the production checklist.
- the merge of main (switch core, ticket 05) is the commit before this report; manifest order 03, 05, 06.

## Findings

- N1 fixed. The reviewer's probe (a creator that ignores ctx, blocked past the lease, resolver releases, then the call commits, then a same-key retry) is `TestStalledCreatorAfterReleaseNeverLeavesASecondLivePayment`. It failed with two live payments before the fix and passes now.
  - A use is released only after `FencePayment` makes its payment impossible. Payminto claims the unique `pl_<LinkPaymentID>` reference with a cancelled placeholder, so a stalled `PaymentService.CreatePayment` hits the `reference_id` unique index and the creator reports `ErrNotCreated`.
  - A released use keeps its row with status `released`; a partial unique index keeps one live use per key, and the key can start afresh.
  - When `Complete` finds the use already released, the just-created payment is cancelled through the new `CancelPayment` and logged as `links: anomaly: payment created for a released use`. `TestCreatorThatCannotFenceHasItsLatePaymentCancelled` covers a creator that cannot fence.
  - On Postgres with the real Payminto creator, `TestIntegration_StalledPaymintoCreationAfterReleaseLeavesOneLivePayment` shows exactly one live `payment_requests` row for the key and `uses_count` 1. A creator unit test shows a fenced reference refuses a later payment.
- N2 fixed. `docker-compose.yml` gives the network a fixed `172.29.86.0/24`, pins nginx to `172.29.86.10` and sets `TRUSTED_PROXIES` to that one address, so the host's port mapping is not trusted. `docker-compose.dev.yml` sets it empty with a note. OPERATIONS.md warns that an unset value behind a proxy collapses every client to one IP and adds it to the production checklist. The router logs once when `X-Forwarded-For` arrives from an untrusted peer. A router test with a trusted proxy shows distinct forwarded clients get distinct budgets and the warning fires once for an untrusted forwarder.
- N3 fixed. Before each reservation on a capped multi-use link, the link's counted uses are checked with the new `OpenPayments`; paid, cancelled and expired payments stop counting (`CloseUses`). For Payminto, open means `OPEN` or `PARTIALLY_FILLED` and unexpired. Client keys group IPv6 by /64. Tests: a fourth payer after three paid from one IP, IPv6 grouping, and a store contract case.
- N4 fixed. `PaymentService` returns the cancel error when its address rollback fails instead of ignoring it. The creator, finding a live payment after a failed creation, cancels it (`ErrNotCreated`), or logs an anomaly and returns an ambiguous error when it cannot (for example once funds arrived), so the resolver handles it. A test covers both.
- N5 fixed. Both limiters count through one Lua script (`INCR`, then `PEXPIRE` when the key has no TTL), so a key always expires and one that lost its TTL gets it back. The Redis integration test covers a key without TTL.

## Checks

- `go build ./... && go vet ./... && go test ./...`: pass, before and after the merge.
- Integration for `./internal/links/... ./internal/api/...` (Postgres, Redis) and the migration manifest test: pass after the merge.

## Residual

- The fence placeholder appears in Payminto's payment list as a cancelled payment with the use's reference.
- A fence that finds a payment whose address is still being assigned completes the use; if that assignment then fails, Payminto cancels the payment and it stops counting as open, but the use is not given back.
- The switch tables from main are not in the environment module's data-table list (not part of this ticket).
