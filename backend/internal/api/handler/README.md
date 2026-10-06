# handler

Owns the Gin HTTP handlers for every Payminto REST endpoint. Each file defines one handler struct (`AuthHandler`, `PaymentHandler`, `WithdrawalHandler`, `MemberHandler`, `WebhookHandler`, etc.), a `New…Handler` constructor, and the methods registered by `internal/api/router`. Handlers are thin: they parse request bodies, call the corresponding `internal/service` method, map errors to HTTP status codes, and encode responses. They depend on `internal/service` for business logic, `internal/api/dto` for wire types, and Gin for HTTP primitives. No persistence, blockchain, or cryptographic logic lives here — that belongs in services.

## Files

- `auth_handler.go` — signup, signin, refresh, logout.
- `analytics_handler.go` — dashboard metrics and reporting endpoints.
- `configuration_handler.go` — merchant configuration CRUD.
- `deposit_address_handler.go` — deposit address allocation and lookup.
- `external_platform_handler.go` — external platform admin endpoints.
- `health_handler.go` — liveness and readiness probes.
- `member_handler.go` — member (user) CRUD and profile.
- `missed_deposit_handler.go` — missed deposit reconciliation.
- `onramper_handler.go` — fiat-to-crypto onramp (Onramper) endpoints.
- `otp_handler.go` — one-time password issuance and verification.
- `payment_handler.go` — payment request creation and lookup.
- `permission_handler.go` — RBAC permission CRUD.
- `public_api_handler.go` — unauthenticated public endpoints.
- `recipient_handler.go` — withdrawal recipient management.
- `referral_admin_handler.go` — admin-only referral campaign ops.
- `referral_handler.go` — merchant referral endpoints.
- `role_handler.go` — RBAC role CRUD.
- `system_handler.go` — system info and admin system endpoints.
- `ticker_handler.go` — price ticker endpoints.
- `webhook_handler.go` — webhook CRUD and delivery log endpoints.
- `withdrawal_handler.go` — withdrawal request lifecycle endpoints.
- `*_test.go` — per-handler unit tests with mocked services.

## See also

- `internal/api/router` — mounts these handlers on route groups
- `internal/service` — business logic invoked by handlers
- `internal/api/dto` — request/response shapes
