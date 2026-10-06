# Payminto Security Review — 2026-08-29

## Release verdict

**NO-GO for public/mainnet use or real funds.** The isolated checkout is suitable for local and testnet evaluation, but the backend settlement and authorization findings below must be remediated and independently retested before production custody is enabled.

This review intentionally contains no passwords, API keys, JWT secrets, encryption keys, database credentials, webhook secrets, or custody secrets.

## P0 — Release blockers

### Settlement can mark an underpaid invoice as paid

- `backend/internal/models/payment.go:12-25` stores the fiat amount but not an immutable asset/network quote in atomic units.
- `backend/internal/repository/payment_repo_impl.go:211-237` sums raw crypto amounts and compares them directly with `AmountInUSD`.
- `backend/internal/worker/deposit_processor.go:49-67` can independently mark a payment `FILLED` after a linked deposit reaches the confirmation threshold, regardless of the required amount.
- The unsafe processor is registered in `backend/cmd/server/main.go:124-126`.

Required remediation: integrate one transactional settlement state machine. Persist immutable chain, asset, decimals, required atomic units, rate/quote identity, expiry, rounding, and tolerance. Match only the selected asset/network and compare integers in atomic units. Do not let the deposit processor write final payment state independently.

### Withdrawal authorization lacks maker/checker separation

- Withdrawal routes have authentication but no action-specific permission checks: `backend/internal/api/router.go:127-153`.
- The request supplies the OTP destination email: `backend/internal/api/handler/withdrawal_handler.go:23-32` and `backend/internal/service/withdrawal_service.go:220-235`.
- The initiating principal can also approve the withdrawal: `backend/internal/service/withdrawal_service.go:289-301`.

Required remediation: add create/read/approve/cancel permissions, send step-up authentication only to the authenticated member's verified channel, require a different approver, and enforce address allowlists and velocity/value caps.

### Existing API keys must never be published in the browser

- The widget reads `data-api-key` from public page source: `widget/src/index.ts:5-11`.
- API-key authentication reaches broad merchant routes, including withdrawals, wallet administration, and key creation: `backend/internal/api/router.go:127-206`.
- The key's `RoleID` is not enforced by RBAC: `backend/internal/api/middleware/auth.go:55-93` and `backend/internal/api/middleware/rbac.go:22-55`.

Required remediation: keep merchant secret keys server-side. Give checkout a short-lived, one-use, capability-limited session token bound to tenant, invoice, origin, amount, allowed assets, and expiry. Enforce explicit API-key scopes on every endpoint.

## P1 — High-priority findings

- Quote calculation must be server-owned and integer-based. Never authorize settlement from a browser `Number`, display rounding, or a fixed stablecoin-equals-one-dollar assumption.
- Customer and member queries need mandatory platform scoping to prevent cross-tenant data exposure.
- Payment creation, quote selection, and address allocation need idempotency keys, unique constraints, and transactional allocation.
- Chain-event identity must include log index for EVM and output index for UTXO chains, with reorg handling before final settlement.
- Webhooks need SSRF-resistant URL validation and egress, a transactional outbox, stable event IDs and signed bytes, replay protection, and encrypted endpoint secrets.
- Bootstrap signup must not let the first public visitor become root. Access tokens should be short-lived, refresh rotation atomic, and reset tokens hashed.
- Onramp sessions must derive every material field from the server-owned invoice and reject replayed/out-of-window provider events.

## Completed hardening in this change

- Checkout now runs as a separate minimal application and origin with a same-origin public BFF; it does not receive dashboard cookies or merchant keys.
- The checkout sends a nonce-based CSP plus `Referrer-Policy`, `Permissions-Policy`, `X-Content-Type-Options`, and frame-denial headers.
- Public address assignment now requires an open, unexpired payment and locks an invoice to its previously selected method.
- API-key identity extraction was corrected to avoid an invalid member-ID context value.
- Local environment and generated credential files are mode `0600`.
- Runtime Next.js dependencies were upgraded and production dependency audits report no known vulnerabilities.

## Production gate

Do not enable mainnet RPCs, custody workers, withdrawal processing, or real-value settlement until all P0 items are fixed, P1 tenant/webhook/idempotency/auth paths are tested, secret history has been scanned and rotated where necessary, and a second independent security review has approved the resulting implementation.
