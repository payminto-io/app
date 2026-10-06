# email

Owns the embedded email template bundle for the Payminto email pipeline. Uses Go's `embed.FS` to compile every file under `templates/*.tmpl` into the binary and expose them as the package-level `FS` variable, which `service.EmailService` imports at boot to parse and render transactional messages (signup verification, OTP, withdrawal confirmations, etc.). The package has no runtime dependencies and contains no rendering logic — rendering, SMTP transport, and delivery queuing live in `internal/service` and `internal/worker/email_processor`. Treat this directory as a static asset bundle with a Go facade.

## Files

- `templates.go` — `//go:embed templates/*.tmpl` FS declaration.
- `templates/` — the actual `.tmpl` files compiled into the binary.

## See also

- `internal/service/email_service.go` — parses and renders these templates
- `internal/worker/email_processor.go` — sends rendered emails
