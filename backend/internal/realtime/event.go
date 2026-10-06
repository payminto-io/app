// Package realtime provides an in-process publish/subscribe broker used to push
// payment-status changes to connected checkout clients over Server-Sent Events.
//
// The Broker interface is deliberately transport-agnostic and small so it can be
// backed by an in-memory implementation (single instance, the default) or, in a
// future multi-instance deployment, by Redis pub/sub without touching callers.
package realtime

// PaymentEvent describes a payment-status change broadcast to subscribers of a
// payment's topic. It carries only non-sensitive fields safe for a public
// checkout client.
type PaymentEvent struct {
	ReferenceID   string `json:"referenceID"`
	State         string `json:"state"`
	Confirmations int    `json:"confirmations,omitempty"`
	Required      int    `json:"requiredConfirmations,omitempty"`
	TxID          string `json:"txID,omitempty"`
}
