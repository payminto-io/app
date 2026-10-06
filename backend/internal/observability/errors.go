package observability

import (
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

// sentryEnabled records whether Sentry was successfully initialized, so Capture
// and Flush are no-ops otherwise.
var sentryEnabled bool

// InitErrorReporting initializes Sentry error reporting when dsn is non-empty.
// When dsn is empty (the dev/test default) it is a no-op and CaptureError falls
// back to structured logging only. Call once at startup; pair with FlushErrors
// on shutdown.
func InitErrorReporting(dsn, environment string) {
	if dsn == "" {
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		EnableTracing:    false,
		AttachStacktrace: true,
	})
	if err != nil {
		slog.Warn("sentry init failed; continuing without error reporting", "error", err)
		return
	}
	sentryEnabled = true
	slog.Info("sentry error reporting enabled", "environment", environment)
}

// CaptureError logs an error via slog and, when Sentry is enabled, forwards it
// for aggregation. Optional key/value context pairs are attached as tags.
func CaptureError(err error, contextPairs ...any) {
	if err == nil {
		return
	}
	slog.Error("captured error", append([]any{"error", err}, contextPairs...)...)
	if !sentryEnabled {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for i := 0; i+1 < len(contextPairs); i += 2 {
			if k, ok := contextPairs[i].(string); ok {
				scope.SetTag(k, toString(contextPairs[i+1]))
			}
		}
		sentry.CaptureException(err)
	})
}

// FlushErrors blocks briefly to deliver buffered Sentry events on shutdown.
func FlushErrors() {
	if sentryEnabled {
		sentry.Flush(2 * time.Second)
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return slogValue(v)
}

func slogValue(v any) string {
	return slog.AnyValue(v).String()
}
