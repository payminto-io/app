package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Worker is the interface that every background goroutine must implement.
// Name returns a human-readable identifier used in log output.
// Start runs until ctx is cancelled and returns any terminal error.
type Worker interface {
	Name() string
	Start(ctx context.Context) error
}

// WorkerStatus holds runtime information about a registered worker. It is
// intentionally defined only in this package to avoid import cycles; callers
// in other packages that need this type should define their own equivalent
// struct and convert via the ManagerAdapter pattern.
type WorkerStatus struct {
	Name          string     `json:"name"`
	Running       bool       `json:"running"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	LastHeartbeat *time.Time `json:"lastHeartbeat,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// workerEntry augments a Worker with runtime tracking state.
type workerEntry struct {
	worker        Worker
	running       atomic.Bool
	startedAt     atomic.Pointer[time.Time]
	lastHeartbeat atomic.Pointer[time.Time]
	lastError     atomic.Pointer[string]
}

// Manager owns a set of Workers and starts them all concurrently.
type Manager struct {
	mu      sync.Mutex
	entries []*workerEntry
	wg      sync.WaitGroup

	// cancel is set when StartAll is called and used by Stop.
	cancel context.CancelFunc
}

// NewManager returns an empty Manager with no registered workers.
func NewManager() *Manager {
	return &Manager{}
}

// Register appends w to the manager's worker list. Call before StartAll.
func (m *Manager) Register(w Worker) {
	m.mu.Lock()
	m.entries = append(m.entries, &workerEntry{worker: w})
	m.mu.Unlock()
}

// StartAll launches every registered worker in its own goroutine. The provided
// ctx is wrapped with a cancel so Stop() can shut everything down cleanly.
func (m *Manager) StartAll(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	m.mu.Lock()
	m.cancel = cancel
	entries := m.entries
	m.mu.Unlock()

	for _, e := range entries {
		m.wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Worker %s panicked: %v", e.worker.Name(), r)
				}
				e.running.Store(false)
			}()
			now := time.Now()
			e.startedAt.Store(&now)
			e.running.Store(true)
			log.Printf("Starting worker: %s", e.worker.Name())
			if err := e.worker.Start(ctx); err != nil {
				errStr := err.Error()
				e.lastError.Store(&errStr)
				log.Printf("Worker %s error: %v", e.worker.Name(), err)
			}
		})
	}
}

// Stop cancels the shared context and waits up to 10 seconds for every worker
// goroutine to drain. Returns an error if StartAll has not been called yet or
// when one or more workers fail to stop within the deadline.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()

	if cancel == nil {
		return fmt.Errorf("worker manager not started — call StartAll first")
	}
	cancel()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	deadline := 10 * time.Second
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	select {
	case <-done:
		return nil
	case <-timer.C:
		return fmt.Errorf("worker manager: timed out waiting for workers to stop after %s", deadline)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Wait blocks until every worker goroutine has returned.
func (m *Manager) Wait() {
	m.wg.Wait()
}

// GetWorkerStatus returns a snapshot of every registered worker's runtime state.
func (m *Manager) GetWorkerStatus() []WorkerStatus {
	m.mu.Lock()
	entries := m.entries
	m.mu.Unlock()

	out := make([]WorkerStatus, 0, len(entries))
	for _, e := range entries {
		s := WorkerStatus{
			Name:    e.worker.Name(),
			Running: e.running.Load(),
		}
		if t := e.startedAt.Load(); t != nil {
			cp := *t
			s.StartedAt = &cp
		}
		if t := e.lastHeartbeat.Load(); t != nil {
			cp := *t
			s.LastHeartbeat = &cp
		}
		if errPtr := e.lastError.Load(); errPtr != nil {
			s.Error = *errPtr
		}
		out = append(out, s)
	}
	return out
}
