package constants

import "time"

// ── Real-time payment streaming (SSE) ────────────────────────────────────────

const (
	// SSEKeepAliveInterval is how often a heartbeat comment is sent to keep
	// idle SSE connections (and intermediary proxies) alive.
	SSEKeepAliveInterval = 15 * time.Second

	// SSESubscriberBuffer is the per-subscriber channel buffer. Events beyond
	// this depth for a slow consumer are dropped rather than blocking the
	// publisher.
	SSESubscriberBuffer = 8

	// PaymentEventTopicPrefix namespaces per-payment topics in the broker.
	// A payment's topic is PaymentEventTopicPrefix + referenceID.
	PaymentEventTopicPrefix = "payment:"
)
