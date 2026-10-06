package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockWorkerManager implements WorkerManager for testing.
type mockWorkerManager struct {
	statuses []WorkerStatus
	stopErr  error
	stopped  bool
}

func (m *mockWorkerManager) GetWorkerStatus() []WorkerStatus {
	return m.statuses
}

func (m *mockWorkerManager) Stop(_ context.Context) error {
	m.stopped = true
	return m.stopErr
}

func TestSystemService_ListWorkers(t *testing.T) {
	mgr := &mockWorkerManager{
		statuses: []WorkerStatus{
			{Name: "w1", Running: true},
			{Name: "w2", Running: false},
		},
	}
	svc := NewSystemService(mgr, nil)

	workers := svc.ListWorkers()
	if len(workers) != 2 {
		t.Errorf("expected 2 workers, got %d", len(workers))
	}
}

func TestSystemService_StopAllWorkers(t *testing.T) {
	mgr := &mockWorkerManager{
		statuses: []WorkerStatus{
			{Name: "sweep-worker", Running: true},
			{Name: "deposit-monitor", Running: true},
		},
	}
	svc := NewSystemService(mgr, nil)

	if err := svc.StopAllWorkers(context.Background()); err != nil {
		t.Fatalf("StopAllWorkers: %v", err)
	}
	if !mgr.stopped {
		t.Error("expected manager.Stop to be called")
	}
}

func TestSystemService_StopAllWorkers_PropagatesError(t *testing.T) {
	mgr := &mockWorkerManager{
		statuses: []WorkerStatus{{Name: "w1", Running: true}},
		stopErr:  errors.New("timeout"),
	}
	svc := NewSystemService(mgr, nil)

	err := svc.StopAllWorkers(context.Background())
	if err == nil {
		t.Error("expected error to be propagated")
	}
}

func TestSystemService_GetSystemHealth_NoWorkers(t *testing.T) {
	mgr := &mockWorkerManager{statuses: []WorkerStatus{}}
	svc := NewSystemService(mgr, nil)

	// db is nil — health check will mark db as error.
	health := svc.GetSystemHealth(context.Background())
	if health.DBStatus != "error" {
		t.Errorf("expected db error with nil db, got %s", health.DBStatus)
	}
	if health.Status != "degraded" {
		t.Errorf("expected degraded with nil db, got %s", health.Status)
	}
}

func TestSystemService_GetSystemHealth_DegradedWorker(t *testing.T) {
	now := time.Now()
	mgr := &mockWorkerManager{
		statuses: []WorkerStatus{
			{Name: "w1", Running: true, StartedAt: &now},
			{Name: "w2", Running: false}, // stopped worker
		},
	}
	svc := NewSystemService(mgr, nil)

	health := svc.GetSystemHealth(context.Background())
	if health.Status != "degraded" {
		t.Errorf("expected degraded due to stopped worker, got %s", health.Status)
	}
}
