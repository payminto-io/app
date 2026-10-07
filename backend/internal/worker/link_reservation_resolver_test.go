package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeLinkResolver struct {
	calls atomic.Int64
	full  atomic.Int64
}

func (f *fakeLinkResolver) ResolveExpired(context.Context, int) (int, int, error) {
	n := f.calls.Add(1)
	if n <= f.full.Load() {
		return 60, 40, nil
	}
	if n == 4 {
		return 0, 0, errors.New("lookup down")
	}
	return 1, 0, nil
}

func TestLinkReservationResolverDrainsFullBatchesAndStops(t *testing.T) {
	f := &fakeLinkResolver{}
	f.full.Store(2)
	w := NewLinkReservationResolver(f, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()
	deadline := time.Now().Add(2 * time.Second)
	for f.calls.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if f.calls.Load() < 5 {
		t.Fatalf("resolver ran %d times; it stopped after an error", f.calls.Load())
	}
	if w.Name() != "link_reservation_resolver" {
		t.Fatal(w.Name())
	}
}
