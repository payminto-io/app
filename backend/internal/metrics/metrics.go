// Package metrics defines the Payminto Prometheus collectors and helpers to
// record them. All metric names are prefixed with constants.MetricsNamespace so
// they share a consistent namespace in the exposition.
//
// Collectors are registered on the default Prometheus registry at package init
// via promauto, so importing this package is enough to expose them on /metrics.
package metrics

import (
	"strconv"
	"time"

	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// httpRequests counts HTTP requests by method, route template, and status.
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: constants.MetricsNamespace,
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total HTTP requests processed, labeled by method, route and status.",
	}, []string{"method", "route", "status"})

	// httpDuration observes request latency by route.
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: constants.MetricsNamespace,
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request latency in seconds, labeled by method and route.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "route"})

	// paymentsCreated counts created payments.
	paymentsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: constants.MetricsNamespace,
		Subsystem: "payments",
		Name:      "created_total",
		Help:      "Total payment requests created.",
	})

	// paymentsConfirmed counts payments that reached the FILLED state.
	paymentsConfirmed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: constants.MetricsNamespace,
		Subsystem: "payments",
		Name:      "confirmed_total",
		Help:      "Total payments confirmed on-chain (FILLED).",
	})

	// webhookDeliveries counts webhook delivery attempts by outcome.
	webhookDeliveries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: constants.MetricsNamespace,
		Subsystem: "webhooks",
		Name:      "deliveries_total",
		Help:      "Total webhook delivery attempts, labeled by outcome.",
	}, []string{"outcome"})

	// sseConnections tracks currently-open checkout SSE streams.
	sseConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: constants.MetricsNamespace,
		Subsystem: "realtime",
		Name:      "sse_connections",
		Help:      "Number of currently-open checkout SSE connections.",
	})
)

// ObserveHTTP records one HTTP request's outcome and latency.
func ObserveHTTP(method, route string, status int, dur time.Duration) {
	httpRequests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	httpDuration.WithLabelValues(method, route).Observe(dur.Seconds())
}

// PaymentCreated increments the created-payments counter.
func PaymentCreated() { paymentsCreated.Inc() }

// PaymentConfirmed increments the confirmed-payments counter.
func PaymentConfirmed() { paymentsConfirmed.Inc() }

// WebhookDelivered records a webhook delivery outcome ("delivered"/"failed").
func WebhookDelivered(outcome string) { webhookDeliveries.WithLabelValues(outcome).Inc() }

// SSEConnectionOpened / SSEConnectionClosed track live SSE stream count.
func SSEConnectionOpened() { sseConnections.Inc() }
func SSEConnectionClosed() { sseConnections.Dec() }
