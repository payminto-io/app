package realtime

import (
	"sync"

	"github.com/payminto/payminto/backend/internal/constants"
)

// Broker is a transport-agnostic publish/subscribe hub keyed by topic.
type Broker interface {
	// Subscribe registers a subscriber for a topic and returns a receive-only
	// channel of events plus an unsubscribe function. The caller MUST invoke
	// unsubscribe when done to avoid leaking the subscription.
	Subscribe(topic string) (<-chan PaymentEvent, func())
	// Publish delivers an event to all current subscribers of the topic.
	// Delivery is non-blocking: a subscriber whose buffer is full misses the
	// event rather than stalling the publisher.
	Publish(topic string, event PaymentEvent)
	// PaymentTopic returns the canonical topic name for a payment reference.
	PaymentTopic(referenceID string) string
}

// MemoryBroker is an in-process Broker safe for concurrent use. Suitable for a
// single backend instance; swap for a Redis-backed Broker to scale horizontally.
type MemoryBroker struct {
	mu     sync.RWMutex
	nextID uint64
	topics map[string]map[uint64]chan PaymentEvent
}

// NewMemoryBroker constructs an empty in-memory broker.
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{topics: make(map[string]map[uint64]chan PaymentEvent)}
}

// PaymentTopic implements Broker.
func (b *MemoryBroker) PaymentTopic(referenceID string) string {
	return constants.PaymentEventTopicPrefix + referenceID
}

// Subscribe implements Broker.
func (b *MemoryBroker) Subscribe(topic string) (<-chan PaymentEvent, func()) {
	ch := make(chan PaymentEvent, constants.SSESubscriberBuffer)

	b.mu.Lock()
	id := b.nextID
	b.nextID++
	subs, ok := b.topics[topic]
	if !ok {
		subs = make(map[uint64]chan PaymentEvent)
		b.topics[topic] = subs
	}
	subs[id] = ch
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subs, ok := b.topics[topic]; ok {
			if _, ok := subs[id]; ok {
				delete(subs, id)
				close(ch)
			}
			if len(subs) == 0 {
				delete(b.topics, topic)
			}
		}
	}
	return ch, unsubscribe
}

// Publish implements Broker. Non-blocking per subscriber.
func (b *MemoryBroker) Publish(topic string, event PaymentEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.topics[topic] {
		select {
		case ch <- event:
		default:
			// Slow subscriber: drop rather than block the publisher. The
			// client's polling fallback reconciles any missed event.
		}
	}
}

// SubscriberCount returns how many subscribers a topic currently has. Useful
// for tests and metrics.
func (b *MemoryBroker) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.topics[topic])
}
