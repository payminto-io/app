// Package observability provides structured logging and request-scoped
// context helpers used across the HTTP and worker layers.
package observability

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	memberIDKey  ctxKey = "member_id"
)

var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// Init configures the default logger. Call once at program start.
func Init(environment string) {
	level := slog.LevelInfo
	if environment == "development" || environment == "dev" {
		level = slog.LevelDebug
	}
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	})).With("app", "payminto", "env", environment)
	slog.SetDefault(defaultLogger)
}

// Logger returns the configured default logger.
func Logger() *slog.Logger { return defaultLogger }

// WithRequestID returns a context annotated with the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extracts the request ID from the context, or empty string.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithMemberID returns a context annotated with the authenticated member ID.
func WithMemberID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, memberIDKey, id)
}

// FromContext returns a logger enriched with request-scoped fields.
func FromContext(ctx context.Context) *slog.Logger {
	l := defaultLogger
	if rid := RequestID(ctx); rid != "" {
		l = l.With("request_id", rid)
	}
	if mid, ok := ctx.Value(memberIDKey).(uint); ok {
		l = l.With("member_id", mid)
	}
	return l
}
