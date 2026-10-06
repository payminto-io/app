# middleware

Owns the cross-cutting Gin middleware used by Payminto's HTTP layer. Each file exposes a single factory (`APIKeyAuth`, `JWTAuth`, `RequirePermission`, `CORS`, `SecurityHeaders`, `RateLimit`, `RequestID`, `ActivityLog`) that returns a `gin.HandlerFunc`. Middleware here is responsible for authentication, authorisation (RBAC), rate limiting, CORS, defensive headers, request-ID propagation into `internal/observability`, and audit logging of mutating requests. Depends on `internal/service` (for auth/RBAC lookups), `internal/repository` (for activity log writes), `internal/observability`, and Redis. Route wiring lives in `internal/api`, not here.

## Files

- `auth.go` — `APIKeyAuth` and JWT bearer authentication.
- `rbac.go` — `RequirePermission` permission gate for route groups.
- `cors.go` — permissive CORS config for browser clients.
- `security.go` — standard defensive response headers (CSP, HSTS, etc.).
- `ratelimit.go` — Redis-backed per-IP rate limiter.
- `request_id.go` — injects/propagates `X-Request-ID` through context.
- `activity_log.go` — audit log recorder with sensitive-field redaction.
- `*_test.go` — per-middleware unit tests.

## See also

- `internal/api` — router that mounts these on groups
- `internal/service` — AuthService, MEPRoleService used by auth/rbac
- `internal/observability` — request-ID context helpers
