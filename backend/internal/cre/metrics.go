package cre

import (
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collectors from SPEC section 9; registered on import like internal/metrics.
var (
	runsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: constants.MetricsNamespace, Subsystem: "cre", Name: "runs_total",
		Help: "Workflow triggers sent, labeled by kind, provider and result.",
	}, []string{"kind", "provider", "result"})
	attestationAge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: constants.MetricsNamespace, Subsystem: "cre", Name: "attestation_age_seconds",
		Help: "Age of the newest attested record per kind at the time it was recorded.",
	}, []string{"kind"})
	verifyFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: constants.MetricsNamespace, Subsystem: "cre", Name: "verify_failures_total",
		Help: "Reports refused by the verifier, labeled by reason.",
	}, []string{"reason"})
	triggerLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: constants.MetricsNamespace, Subsystem: "cre", Name: "trigger_latency_seconds",
		Help: "Time to get a workflow trigger accepted.", Buckets: prometheus.DefBuckets,
	})
)

func recordRun(run Run, took time.Duration) {
	runsTotal.WithLabelValues(string(run.Kind), run.Provider, run.Status).Inc()
	triggerLatency.Observe(took.Seconds())
}

func recordAttestation(a Attestation) {
	if a.Status == StatusAttested {
		attestationAge.WithLabelValues(string(a.Kind)).Set(a.RecordedAt.Sub(a.ObservedAt).Seconds())
	}
}

func recordVerifyFailure(err error) {
	reason := "other"
	for _, c := range []struct {
		err  error
		name string
	}{
		{ErrForged, "forged"}, {ErrReplayed, "replayed"}, {ErrWrongWorkflow, "wrong_workflow"}, {ErrWrongOwner, "wrong_owner"},
		{ErrWrongGateway, "wrong_gateway"}, {ErrWrongEmitter, "wrong_emitter"}, {ErrUnconfirmed, "unconfirmed"},
		{ErrStale, "stale"}, {ErrInvalidReport, "invalid"},
	} {
		if errors.Is(err, c.err) {
			reason = c.name
			break
		}
	}
	verifyFailures.WithLabelValues(reason).Inc()
}
