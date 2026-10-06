# observability

Owns structured logging and request-scoped context for the Payminto backend. Exposes `Init` (configure the default `slog.Logger` per environment — JSON output, debug level in dev, info in prod), `Logger` (access the configured logger), and the context helpers `WithRequestID` / `WithMemberID` / `FromContext` used by middleware and workers to propagate correlation IDs through `context.Context` and into log records. Depends only on `log/slog` and the standard library. Every other package logs through this rather than calling `slog.Default()` directly so log shape stays consistent.

## Files

- `logger.go` — default logger init, accessors, and context key helpers.

## See also

- `internal/api/middleware/request_id.go` — sets the request ID this package reads
- `cmd/server` — calls `observability.Init` at boot
