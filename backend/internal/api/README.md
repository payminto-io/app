# api

Owns the HTTP surface of the Payminto backend: the Gin router wiring that binds services, middleware, and handlers into a single `*gin.Engine`. Exposes `RouterConfig` (the DI struct carrying every service the router needs) and `NewRouter` (builds the engine, mounts route groups, applies middleware stacks). It depends on `internal/api/handler`, `internal/api/middleware`, and `internal/service`, and is consumed by `cmd/server` at boot. This package contains no business logic — it is purely declarative wiring that determines which handler serves which path and which middleware guards which group.

## Files

- `router.go` — `RouterConfig` struct and `NewRouter` constructor that wires every route group.
- `routes_fees.go` - `RegisterFeesRoutes`: fee preview and `/admin/fee-rules` (see `internal/fees/README.md`).
- `router_test.go` — integration tests that boot the full router against a test service registry.

## See also

- `internal/api/handler` — endpoint implementations mounted here
- `internal/api/middleware` — middleware applied to route groups
- `internal/service` — services injected via `RouterConfig`
