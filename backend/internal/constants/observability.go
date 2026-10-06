package constants

// ── Observability ────────────────────────────────────────────────────────────

const (
	// MetricsNamespace prefixes every Prometheus metric exported by the app.
	MetricsNamespace = "payminto"

	// MetricsPath is the HTTP path the Prometheus exposition is served on.
	MetricsPath = "/metrics"
)
