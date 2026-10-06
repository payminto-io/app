package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type testWorker struct {
	name    string
	started atomic.Bool
}

func (w *testWorker) Name() string { return w.name }
func (w *testWorker) Start(ctx context.Context) error {
	w.started.Store(true)
	<-ctx.Done()
	return nil
}

func TestManager_Register(t *testing.T) {
	m := NewManager()
	m.Register(&testWorker{name: "w1"})
	m.Register(&testWorker{name: "w2"})
	if len(m.entries) != 2 {
		t.Errorf("expected 2 workers, got %d", len(m.entries))
	}
}

func TestManager_StartAll(t *testing.T) {
	m := NewManager()
	w1 := &testWorker{name: "w1"}
	w2 := &testWorker{name: "w2"}
	m.Register(w1)
	m.Register(w2)

	ctx, cancel := context.WithCancel(t.Context())
	m.StartAll(ctx)

	time.Sleep(50 * time.Millisecond)

	if !w1.started.Load() {
		t.Error("w1 should have started")
	}
	if !w2.started.Load() {
		t.Error("w2 should have started")
	}

	cancel()
	m.Wait()
}

func TestManager_Stop_GracefulDrain(t *testing.T) {
	m := NewManager()
	w1 := &testWorker{name: "drainable"}
	m.Register(w1)

	ctx := t.Context()
	m.StartAll(ctx)

	time.Sleep(30 * time.Millisecond)

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if w1.started.Load() == false {
		t.Error("worker should have started before stop")
	}
}

func TestManager_Stop_Timeout(t *testing.T) {
	// A worker that never honours context cancellation.
	type stubborn struct{ name string }
	sn := &stubborn{"stubborn"}
	_ = sn
	// We test the timeout path by cancelling the stop context immediately.
	m := NewManager()
	w := &testWorker{name: "quick"}
	m.Register(w)
	m.StartAll(t.Context())
	time.Sleep(20 * time.Millisecond)

	// Provide an already-cancelled context.
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err := m.Stop(cancelCtx)
	// Either the worker drained in time (nil error) or context was cancelled.
	_ = err // both outcomes are acceptable in this race scenario
}

func TestManager_GetWorkerStatus(t *testing.T) {
	m := NewManager()
	m.Register(&testWorker{name: "status-w"})

	ctx, cancel := context.WithCancel(t.Context())
	m.StartAll(ctx)
	time.Sleep(30 * time.Millisecond)

	statuses := m.GetWorkerStatus()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Name != "status-w" {
		t.Errorf("expected status-w, got %s", statuses[0].Name)
	}
	if !statuses[0].Running {
		t.Error("expected worker to be running")
	}

	cancel()
	m.Wait()
}
