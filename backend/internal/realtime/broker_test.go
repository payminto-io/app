package realtime

import (
	"testing"
	"time"
)

func TestMemoryBroker_PublishReachesSubscriber(t *testing.T) {
	b := NewMemoryBroker()
	topic := b.PaymentTopic("ref-123")

	ch, unsub := b.Subscribe(topic)
	defer unsub()

	if got := b.SubscriberCount(topic); got != 1 {
		t.Fatalf("subscriber count = %d, want 1", got)
	}

	want := PaymentEvent{ReferenceID: "ref-123", State: "FILLED", Confirmations: 12, Required: 12}
	b.Publish(topic, want)

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("event = %+v, want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestMemoryBroker_UnsubscribeRemovesTopic(t *testing.T) {
	b := NewMemoryBroker()
	topic := b.PaymentTopic("ref-x")
	_, unsub := b.Subscribe(topic)
	unsub()
	if got := b.SubscriberCount(topic); got != 0 {
		t.Fatalf("after unsubscribe count = %d, want 0", got)
	}
}

func TestMemoryBroker_PublishNoSubscribersIsNoop(t *testing.T) {
	b := NewMemoryBroker()
	// Should not panic or block.
	b.Publish(b.PaymentTopic("nobody"), PaymentEvent{ReferenceID: "nobody"})
}

func TestMemoryBroker_MultipleSubscribers(t *testing.T) {
	b := NewMemoryBroker()
	topic := b.PaymentTopic("multi")
	ch1, unsub1 := b.Subscribe(topic)
	ch2, unsub2 := b.Subscribe(topic)
	defer unsub1()
	defer unsub2()

	b.Publish(topic, PaymentEvent{ReferenceID: "multi", State: "EXPIRED"})

	for i, ch := range []<-chan PaymentEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.State != "EXPIRED" {
				t.Fatalf("subscriber %d state = %q, want EXPIRED", i, got.State)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out", i)
		}
	}
}

// TestMemoryBroker_SlowSubscriberDoesNotBlock fills a subscriber's buffer and
// confirms further publishes neither block nor panic.
func TestMemoryBroker_SlowSubscriberDoesNotBlock(t *testing.T) {
	b := NewMemoryBroker()
	topic := b.PaymentTopic("slow")
	_, unsub := b.Subscribe(topic)
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			b.Publish(topic, PaymentEvent{ReferenceID: "slow", Confirmations: i})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publisher blocked on slow subscriber")
	}
}
