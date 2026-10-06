# Payminto Go Backend Production Audit

Date: 2026-08-19  
Scope: `payminto/backend` (read-only review of product code)  
Perspective: senior backend engineer / production Go systems

## Executive verdict

The backend has a substantial implementation, good use of exact decimal arithmetic, repository seams, contextual worker shutdown, atomic row claims in several places, tenant-scoped payment reads, refresh-token rotation, and a broad unit-test suite. It is **not safe to operate as a self-hosted payment processor yet**.

The release-blocking problem is not simply missing polish: the invoice settlement invariant is undefined and currently violated. An invoice stores USD, a deposit stores units of a cryptoasset, and settlement directly sums the crypto units and compares that sum with USD. At the same time, a second worker bypasses that comparison and marks any sufficiently confirmed deposit as `FILLED`, even if it is zero or only a partial payment. Newly detected deposits can also remain `pending` forever because the production confirmation scan selects only `confirming` rows. No immutable server-side quote exists to say how much of which asset was required.

Additional release blockers are: confirmed deposits never post the payment ledger entry; the payment-expiry, email, webhook, and account/withdrawal orchestration workers are not registered; production accepts an empty JWT secret; defensive HTTP middleware exists but is not mounted; Redis is never constructed; the production Docker image cannot build because it uses Go 1.22 while `go.mod` requires Go 1.25.4; and the webhook retry implementation can duplicate or loop deliveries.

**Recommendation: NO-GO for mainnet, real customer funds, or public Internet exposure.** Treat the current build as an integration prototype until P0/P1 items below are fixed and exercised by PostgreSQL-backed end-to-end tests.

## What works today

- All packages compile locally on Go 1.25.4. `go test ./...`, `go vet ./...`, `go build ./cmd/server ./cmd/migrate`, and focused `-race` tests passed during this audit.
- Monetary fields use `shopspring/decimal` and PostgreSQL `numeric(38,18)`, avoiding binary floating-point for stored amounts (`internal/models/payment.go:15,47`).
- Payment list/detail repository methods scope by `external_platform_id` (`internal/repository/payment_repo_impl.go:85-94,122-127,132-173`).
- API keys are generated from `crypto/rand` and stored as SHA-256 hashes (`internal/service/auth_service.go:57-69`). Refresh tokens are random, hashed at rest, rotated, and grouped for replay-family revocation (`internal/service/jwt_token_service.go:71-104`).
- Deposit confirmation has an atomic conditional-update primitive, and payment finalization has a PostgreSQL `SELECT ... FOR UPDATE` seam (`internal/repository/deposit_repo_impl.go:137-150`; `internal/repository/payment_repo_impl.go:196-252`).
- Address-pool claims use a transaction with `FOR UPDATE SKIP LOCKED`, a good concurrency primitive (`internal/repository/address_pool_repo_impl.go:79-99`).
- Chain scanning deliberately avoids advancing the persisted height after block-processing failure (`internal/worker/blockchain_processor.go:124-187`).
- The email queue consumer has an atomic pending-to-processing claim and bounded retry/dead-letter logic (`internal/worker/email_processor.go:75-160`; `internal/repository/ee_event_repo_impl.go:82-131`).
- The worker manager propagates cancellation and waits for goroutines during normal shutdown (`internal/worker/manager.go:63-124`).
- Request IDs, JSON logging, Prometheus collectors, Sentry initialization, liveness/readiness routes, CORS, and graceful HTTP shutdown exist, although their production coverage has gaps described below.

Passing unit tests are not a production-readiness signal here: several tests encode the faulty behavior. For example, `TestDepositProcessor_ConfirmsAndPublishes` creates a payment and deposit with zero monetary amounts, then requires the payment to become `FILLED` (`internal/worker/deposit_processor_test.go:28-64`).

## Findings

### P0 — Invoice settlement compares different units and has two conflicting state machines

Evidence:

- `PaymentRequest.AmountInUSD` is USD (`internal/models/payment.go:12-16`), while `Deposit.Amount` is the native/token amount (`internal/models/payment.go:42-58`).
- Finalization sums raw confirmed `Deposit.Amount` values and compares them directly with `AmountInUSD`; the code itself leaves a TODO to convert to USD (`internal/repository/payment_repo_impl.go:211-237`). A deposit of `0.05 ETH` against a `$100` invoice becomes partial; `100 USDT` happens to work only by coincidence; `100 ETH` becomes exactly filled rather than enormously overpaid.
- No quote snapshot or required crypto amount exists on the payment model or creation path (`internal/models/payment.go:12-30`; `internal/service/payment_service.go:43-91`). Ticker data is fetched independently and cached, not locked into the invoice.
- `DepositProcessor` directly marks a confirmed deposit and its payment `FILLED`, bypassing `DepositService.FinalizePayment`, partial/overpayment calculation, row locking, `confirmed_at`, and terminal-state guards (`internal/worker/deposit_processor.go:49-67` versus `internal/service/deposit_service.go:166-172`).

Impact: false-positive settlement, false under/overpayment states, inconsistent results depending on which worker wins, and an inability to reconcile the promised checkout quote after market movement.

Required fix: create an immutable `PaymentQuote` (or equivalent fields) with `asset_id`, `network_id`, `fiat_amount`, `fiat_currency`, `rate`, `required_asset_amount`, `quote_provider`, `quoted_at`, `expires_at`, rounding policy, and tolerance. Sum confirmed deposits by the quoted asset and compare like units only. Expose one transition method; no worker may write payment state directly.

### P0 — The production deposit confirmation pipeline can strand every new deposit

`RecordDeposit` creates rows with status `pending` (`internal/service/deposit_service.go:100-116`). The per-chain confirmation loop calls `ListConfirmingForChain` (`internal/worker/blockchain_processor.go:353-381`), but that repository query selects only status `confirming` (`internal/repository/deposit_repo_impl.go:162-175`). Nothing in the production path advances the initial `pending` row. Unit tests hide this by calling `UpdateConfirmations` directly (`internal/service/deposit_service_test.go:186-205`).

Impact: a real detected payment can remain pending indefinitely.

Required fix: select both `pending` and `confirming`, add an integration test that starts with `RecordDeposit`, drives the actual confirmation loop seam, and observes settlement.

### P0 — Confirmed deposits are not posted to the ledger

`LedgerService.RecordPaymentDeposit` exists, but the only non-test occurrence is its declaration (`internal/service/ledger_service.go:26-50`). Neither deposit confirmation nor payment finalization calls it. Consequently the customer-facing payment may be filled without increasing the merchant liability or crypto asset ledger.

Impact: custody and accounting diverge; withdrawal availability, reconciliation, and financial statements cannot be trusted.

Required fix: atomically couple deposit confirmation, invoice transition, ledger journal, and outbox event in one PostgreSQL transaction. Ledger idempotency needs database-enforced journal keys, not comments or caller convention.

### P0 — Essential workers are implemented but never run

Production registration includes address-pool warming, `DepositProcessor`, optional EVM sweeping, and per-chain block processors only (`cmd/server/main.go:115-165`). The following implementations are not registered:

- `PaymentExpiryWorker` (`internal/worker/payment_expiry.go:11-43`): open invoices never expire.
- `EmailProcessor` (`internal/worker/email_processor.go:36-73`): signup/password-reset/withdrawal events accumulate, and password-reset email never arrives.
- `WebhookProcessor` (`internal/worker/webhook_processor.go:18-45`): retry queue never runs.
- `AccountProcessorJob` (`internal/worker/account_processor.go:55-67,186-215`): BTC/ERC-20 sweeping, withdrawal processing, rewards, and stale recovery do not run. Several of its loops are explicitly still stubs (`:59-60`).

No producer was found that invokes `EmitWebhook` for `payment.confirmed`, and no non-test call to `WebhookService.Deliver` exists outside `WebhookProcessor`.

Impact: core lifecycle promises are silently absent despite routes and services appearing complete.

Required fix: make worker composition a validated boot module. Startup should fail when a mandatory worker or dependency is missing, and readiness must report registered/running/last-success/error status for each required worker.

### P1 — Payment/address creation is not an atomic, idempotent operation

Payment creation inserts the payment, auto-creates a customer, and claims/creates an address across separate operations (`internal/service/payment_service.go:73-136`). On address failure it changes the payment to `CANCELLED` rather than rolling back (`:125-132`). Address assignment commits the pool claim before creating `DepositAddress`; a failed create permanently burns the address (`internal/service/deposit_address_service.go:90-118`; claim transaction ends at `internal/repository/address_pool_repo_impl.go:83-99`).

There is no unique `(external_platform_id, invoice_id)` idempotency constraint; `InvoiceID` is nullable free text (`internal/models/payment.go:19,24`). A retried merchant request creates a second invoice. Public address assignment uses list-then-create without a payment/asset uniqueness constraint (`internal/api/handler/public_api_handler.go:129-175`; `internal/models/wallet.go:38-48`), so concurrent calls can consume multiple addresses.

Required fix: use a merchant-supplied idempotency key and a single transaction for invoice + quote + address claim + deposit-address link + outbox. Add database uniqueness for deposit identity and assignment identity.

### P1 — Deposit deduplication is check-then-insert without a database invariant

`RecordDeposit` first queries, then separately inserts (`internal/service/deposit_service.go:62-69,100-116`). `Deposit` has no unique index for chain/asset/transaction/log-or-output identity (`internal/models/payment.go:44-62`). Concurrent/replayed scanners can both pass the check. The chosen key `(tx hash, to address, currency)` is also insufficient to represent repeated logs/outputs to the same destination in one transaction.

Required fix: model a canonical chain event identity (`chain_id + tx_hash + log_index` for EVM, `txid + vout` for Bitcoin, equivalent Tron index) with a unique constraint and insert-on-conflict semantics.

### P1 — Webhook delivery/retry semantics are unsafe and incomplete

- `Deliver` creates a new delivery-log row for every attempt (`internal/service/webhook_service.go:50-98`). The retry worker then updates the old failed row (`internal/worker/webhook_processor.go:48-73`), splitting one logical delivery across rows.
- HTTP 4xx/5xx sets the new log to failed but returns `nil` (`internal/service/webhook_service.go:76-103`). The retry worker therefore does not increment/schedule/dead-letter the original row, making it eligible every 30 seconds indefinitely if the worker is ever registered.
- Multiple process replicas select retry rows without claim locking (`internal/worker/webhook_processor.go:48-54`).
- There is no event/delivery idempotency key in payload or headers, no SSRF policy for merchant URLs, and response bodies are not bounded/read for diagnosis.
- Payment finalization does not enqueue webhook events at all.

Required fix: one outbox event and one logical delivery row per `(event_id, webhook_id)`, atomically claimed with `SKIP LOCKED`; update that row on every attempt; classify HTTP response codes; include stable event/delivery IDs and signature version/timestamp; block private/link-local/metadata destinations and revalidate redirects.

### P1 — Production accepts empty secrets and insecure database transport

`config.Load` defaults `JWT_SECRET`, `AES_KEY`, database password, and vault passphrase to empty and performs no validation (`internal/config/config.go:83-130`). Empty JWT strings are then used to sign and validate access tokens (`internal/service/auth_service.go:93-125`; `internal/service/registry.go:298-300,420-428`). Production can run with the vault locked rather than fail boot (`cmd/server/main.go:76-89`). The database DSN always uses `sslmode=disable` (`internal/config/config.go:132-140`).

Additional auth concerns:

- The access and refresh signing secrets are the same (`internal/service/registry.go:420-428`).
- JWT validation does not explicitly restrict the expected signing algorithm (`internal/service/auth_service.go:113-117`).
- Password-reset tokens are stored raw on the member record (`internal/service/auth_service.go:294-304`).
- Public signup uses a global `memberRepo.Count()` check to select the root user (`internal/service/auth_service.go:147-160`), a race on first boot and not a per-platform invitation model.

Required fix: environment-aware `Validate()` that refuses production unless distinct high-entropy secrets, vault credentials/KMS, TLS DB mode, explicit origins, valid destinations, and required RPC/SMTP settings exist. Hash reset tokens, pin HS256, shorten access TTL, and provision the initial administrator through a one-time transactional bootstrap.

### P1 — Rate limiting and HTTP security middleware exist but are not mounted

The router mounts only request ID, CORS, and optional metrics globally (`internal/api/router.go:81-93`). `SecurityHeaders` and Redis `RateLimit` exist and are unit-tested but are never used (`internal/api/middleware/security.go:5-16`; `internal/api/middleware/ratelimit.go:13-42`). The registry receives `nil` Redis from main (`cmd/server/main.go:66`), and no Redis client is created even though Redis is deployed.

Impact: signup/signin/forgot-password/reset, public ticker, checkout lookup, SSE, and address assignment have no distributed abuse control. Public `/metrics` may disclose operational data. Rate limiting fails open on Redis errors (`internal/api/middleware/ratelimit.go:25-29`).

Required fix: mount layered limits (global, IP, identity, endpoint), use Redis with an atomic Lua/token-bucket algorithm, define deliberate failure behavior, protect metrics, and mount appropriate security headers (HSTS only behind confirmed TLS).

### P1 — Tenant ownership is inconsistent between platform and member

Payments and webhooks are platform-scoped, but wallets and deposit addresses are member-scoped (`internal/models/payment.go:23-29`; `internal/models/wallet.go:3-15,36-46`). Address assignment selects a wallet by `payment.MemberID + family`, not by the payment's platform (`internal/service/deposit_address_service.go:83-87`). A member belonging to multiple projects can therefore mix project funds and addresses. API keys can be created by any caller of the general protected group without a `settings.write` permission, accept an arbitrary `RoleID`, and actual authorization checks the member's MEP role rather than enforcing key-scoped permissions (`internal/api/router.go:177-182`; `internal/api/handler/api_key_handler.go:91-145`; `internal/api/middleware/rbac.go:22-55`).

Required fix: choose and enforce one custody tenant aggregate (normally external platform/project), include `tenant_id` in every wallet/address/deposit/account/journal row, use compound foreign keys or transaction-time ownership checks, and make API-key scopes authoritative and non-escalatable.

### P1 — Shipping image and CI cannot build the module

`go.mod:3` requires Go 1.25.4. `Dockerfile:1` uses `golang:1.22-alpine`, and `.github/workflows/ci.yml:29-31` selects Go 1.22. The audit's real Docker build failed at `go mod download`:

```text
go: go.mod requires go >= 1.25.4 (running go 1.22.12; GOTOOLCHAIN=local)
```

The development Compose image also tries to run `air`, but the Dockerfile never installs it and the bind mount replaces `/app` (`docker-compose.dev.yml:31-54`). Production Compose has no health-based DB dependency, no backend healthcheck, no explicit network mode, no vault/SMTP/RPC/cold-wallet configuration, and no backend port declaration (routing depends entirely on nginx correctness) (`docker-compose.yml:18-33`).

Required fix: pin one supported Go toolchain across module, Docker, CI and developer tooling; add a non-root runtime user, read-only filesystem, healthcheck, SBOM/vulnerability scan, reproducible version metadata, and a separate dev target that installs `air`.

### P1 — Boot-time AutoMigrate is not an auditable production migration strategy

The server mutates all 73 tables on every boot (`cmd/server/main.go:57-64`; `internal/database/database.go:34-136`). The migration command also uses AutoMigrate, does not support rollback, and says operators must drop the DB manually (`cmd/migrate/main.go:46-62`). The migration directory contains seeds, not an ordered schema history. `runSeeds` claims each file is executed in its own transaction, but it calls plain `db.Exec` without an explicit transaction (`cmd/migrate/main.go:66-99`).

Impact: schema drift is not reviewable or reliably reversible; rolling deployments can run incompatible code/schema combinations; startup can acquire DDL locks.

Required fix: versioned forward migrations with expand/migrate/contract discipline, checksums, advisory lock, separate deploy step, backups and tested restore/rollback plans. Application containers should have no DDL permission.

### P2 — State-machine and expiry semantics are incomplete

States are untyped strings and any repository caller can set any value (`internal/models/payment.go:34-40`; `internal/repository/payment_repo_impl.go:185-190`). Expiry changes only `OPEN`, not `PARTIALLY_FILLED` (`internal/repository/payment_repo_impl.go:176-182`). The payment does not capture cancellation/expiry reason, late-payment state, refund disposition, reorg/reversal, risk hold, or finality version. The two settlement writers produce different side effects and metrics.

Required fix: centralize permitted transitions with compare-and-swap/versioning and an append-only transition history. Explicitly model `EXPIRED`, late deposits, refund/manual review, and reorg reversal.

### P2 — Public/API contracts need hardening

- The create response uses `reference_id`, detail uses `referenceID`, and detail's state field is `paymentState` while public detail uses `state` (`internal/api/dto/payment_dto.go:17-33`; `internal/api/handler/public_api_handler.go:53-58`).
- The create URL host is always built as `http://localhost:<port>` in production (`cmd/server/main.go:169-180`).
- Public address assignment treats the UUID reference as a bearer capability but does not reject expired/cancelled/filled invoices and allows the first caller to choose the settlement chain/asset (`internal/api/handler/public_api_handler.go:105-175`).
- Several handlers expose raw internal error strings, creating an unstable contract and potential information leakage.
- No OpenAPI document/versioning/standard error envelope/idempotency-header contract was found.

Required fix: version and generate the contract, use a configured external base URL, stable error codes, explicit pagination, idempotency semantics, and state checks on all mutating public operations.

### P2 — Runtime resilience and observability are insufficient for custody

- `http.Server` sets only `ReadHeaderTimeout`; `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes` are absent (`cmd/server/main.go:215-220`). Long-lived SSE needs a deliberate separate timeout policy.
- A worker panic is logged and permanently stops that worker; there is no restart policy or readiness failure (`internal/worker/manager.go:73-90`). `LastHeartbeat` is never updated (`internal/worker/manager.go:32-38,132-157`).
- `/healthz` and `/readyz` are identical DB pings and do not assess workers, Redis, RPC lag, vault state, migration level, or queue backlog (`internal/api/router.go:95-98`; `internal/api/handler/health_handler.go:20-32`).
- DB logging is always `Info`, potentially noisy and sensitive, and connection lifetime/idle lifetime are unset (`internal/database/database.go:13-31`).
- `metrics.WebhookDelivered` is defined but not called; finalization paths do not consistently increment payment metrics (`internal/metrics/metrics.go:52-83`).
- Much code uses the standard unstructured `log` package, bypassing request/tenant/chain/payment correlation and Sentry capture.
- `log.Fatalf` on server failure bypasses deferred cleanup (`cmd/server/main.go:239-254`).

Required fix: SLO-oriented metrics and alerts for chain head lag, oldest pending deposit, confirmation lag, quote/fill mismatches, outbox age, webhook/email attempts, sweep/withdrawal backlog, ledger imbalance, RPC failover, and worker heartbeat. Make readiness dependency-aware and logging consistently structured/redacted.

## Red-capable verification commands

These are deterministic checks that currently expose the exact wiring/invariant defects and can be converted directly into CI gates.

```bash
cd payminto/backend

# Fails while any worker directly writes FILLED.
! rg 'Update\("state", models.PaymentStateFilled\)' internal/worker/deposit_processor.go

# Fails while settlement explicitly admits raw crypto-vs-USD comparison.
! rg 'TODO\(phase-I\): convert deposit amounts to USD before comparison' internal/repository/payment_repo_impl.go

# Fails until the quote snapshot is modeled.
rg 'RequiredCrypto|RequiredAssetAmount|LockedRate|PaymentQuote' internal/models internal/service/payment_service.go

# Fails until both defensive middleware are mounted.
rg 'r\.Use\(middleware\.SecurityHeaders' internal/api/router.go && \
rg 'middleware\.RateLimit' internal/api/router.go

# Fails until lifecycle workers are registered.
for worker in PaymentExpiryWorker EmailProcessor WebhookProcessor AccountProcessorJob; do
  rg "New${worker}" cmd/server/main.go || exit 1
done

# Real release artifact check; currently fails due Go 1.22/1.25.4 mismatch.
docker build -t payminto-backend-audit .
```

Audit checks run:

```text
go test ./... -count=1                                           PASS (24.95s)
go vet ./...                                                     PASS
go build ./cmd/server ./cmd/migrate                              PASS
go test -race ./internal/service/... ./internal/repository/... \
  ./internal/worker/... ./internal/api/... -count=1              PASS (48.54s slowest package)
docker build -t payminto-backend-audit .                         FAIL (toolchain mismatch)
```

Missing high-value test layers:

- PostgreSQL-backed settlement tests (SQLite does not exercise PostgreSQL locking/decimal/constraint behavior).
- Full detected-block -> pending -> confirming -> confirmed -> quote settlement -> ledger -> outbox -> webhook flow.
- Two-process concurrency tests for duplicate deposit, address assignment, invoice idempotency, worker claims, and webhook delivery.
- Reorg and late/partial/overpayment fixtures for BTC, native EVM, ERC-20, and Tron.
- Crash-injection tests at every DB/external-call boundary.
- Production-container boot and Compose smoke tests in CI.

## Recommended architecture: deepen the payment lifecycle module

Today the payment lifecycle is a shallow cluster: handlers, workers, services, and repositories each know pieces of state ordering, units, locking, side effects, and retry behavior. The same transition is implemented in multiple places, so the interface callers must understand is nearly as complex as the implementation.

Create one deep `PaymentLifecycle` module at a transactional seam:

```go
type PaymentLifecycle interface {
    Create(ctx context.Context, cmd CreatePayment) (PaymentView, error)
    ObserveDeposit(ctx context.Context, event ChainDeposit) (SettlementResult, error)
    AdvanceFinality(ctx context.Context, event ChainFinality) (SettlementResult, error)
    Expire(ctx context.Context, paymentID PaymentID, now time.Time) (SettlementResult, error)
}
```

Its implementation should own quote units/rounding, permitted state transitions, row locking/versioning, deposit identity, ledger journals, and outbox records in one transaction. Chain monitors become adapters that report canonical observations; HTTP and workers cross the same small interface. This produces locality: a settlement invariant is fixed once rather than in every caller.

Add two other deep modules:

1. `OutboxDispatcher`: one interface over transactional events, claims, retries, delivery idempotency, email/webhook adapters, and dead-letter policy.
2. `WorkerRuntime`: declarative required-worker registration, leader/partition strategy, health/heartbeats, supervision, and shutdown. Startup and readiness consume the same registry.

Avoid adding repository methods as new public seams for every query. Keep persistence adapters internal to these implementations. The interface is the test surface; run production-like PostgreSQL adapters and deterministic in-memory fakes through the same lifecycle contract.

## Prioritized remediation plan

### Stop-ship (before any funds)

1. Disable direct payment-state writes; remove or rework `DepositProcessor` so one lifecycle transition owns settlement.
2. Introduce immutable quotes and compare required/received amounts in the same asset unit with explicit rounding/tolerance.
3. Fix pending-deposit selection and add the end-to-end confirmation test.
4. Atomically write settlement + ledger + outbox; add a balance invariant check.
5. Add database unique identities for chain deposits and merchant idempotency keys.
6. Register and health-check mandatory workers; implement missing webhook producer/consumer and withdrawal orchestration.
7. Fail production boot on empty/weak secrets, locked vault, insecure DB mode, missing RPC/destinations, and invalid origin/base URL.
8. Align Go toolchain and make the production Docker/CI smoke test green.

### Before private testnet beta

9. Make payment/address creation one transaction; enforce project-level wallet/deposit/account ownership.
10. Replace webhook retry logic with a claimed outbox delivery state machine and SSRF defense.
11. Mount rate/security middleware and actually connect Redis.
12. Replace AutoMigrate with versioned migrations and a backup/restore-tested deployment runbook.
13. Define late payment, expiry of partial payments, cancellation, manual review/refund, and reorg semantics.
14. Add API schema, consistent field names/errors, configured external URL, scoped API keys, and idempotency headers.

### Before mainnet/general availability

15. PostgreSQL multi-process concurrency, chaos/crash, reorg, and provider-failure suites.
16. Worker leadership/partitioning and horizontal broker/outbox support; the current SSE broker is process-local (`cmd/server/main.go:110-113`).
17. Full custody reconciliation: chain balances vs deposit events vs ledger vs sweeps/withdrawals, with alerting and operator repair tools.
18. Secret rotation/KMS or HSM strategy, dependency/container scanning, non-root hardened runtime, audit log retention, disaster recovery, and independently reviewed threat model.

## Bottom line

There is valuable implementation here, and many low-level primitives are heading in the right direction. The project is nevertheless at **prototype/integration-test maturity**, not self-hosted payment-gateway maturity. The fastest safe route is to stop adding surface area, consolidate settlement behind one deep transactional module, make accounting/outbox effects atomic, and promote production wiring/configuration into testable first-class code.
