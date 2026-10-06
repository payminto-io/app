package service

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// WorkerStatus holds runtime information about a registered worker.
type WorkerStatus struct {
	// Name is the worker's human-readable identifier.
	Name string `json:"name"`
	// Running reports whether the worker goroutine is currently active.
	Running bool `json:"running"`
	// StartedAt is when the worker was last started.
	StartedAt *time.Time `json:"startedAt,omitempty"`
	// LastHeartbeat is the time the worker last completed a loop iteration.
	LastHeartbeat *time.Time `json:"lastHeartbeat,omitempty"`
	// Error is the terminal error from the last run (if any).
	Error string `json:"error,omitempty"`
}

// WorkerManager is the interface SystemService uses to interact with the worker
// manager. It matches the public surface of worker.Manager without importing
// that package (avoiding import cycles).
type WorkerManager interface {
	GetWorkerStatus() []WorkerStatus
	Stop(ctx context.Context) error
}

// SystemHealth aggregates the runtime health of workers and the database.
type SystemHealth struct {
	// Status is "ok" when all subsystems are healthy, "degraded" otherwise.
	Status string `json:"status"`
	// Workers holds per-worker status snapshots.
	Workers []WorkerStatus `json:"workers"`
	// DBStatus is "ok" when the database is reachable.
	DBStatus string `json:"dbStatus"`
}

// SystemService exposes runtime control over worker goroutines and overall
// platform health.
type SystemService struct {
	manager WorkerManager
	db      *gorm.DB
}

// NewSystemService constructs a SystemService.
func NewSystemService(manager WorkerManager, db *gorm.DB) *SystemService {
	return &SystemService{manager: manager, db: db}
}

// ListWorkers returns a snapshot of every registered worker's runtime status.
func (s *SystemService) ListWorkers() []WorkerStatus {
	return s.manager.GetWorkerStatus()
}

// StopAllWorkers gracefully stops every registered worker by cancelling the
// shared context. In the current architecture all workers share one context, so
// per-worker stop is not supported — use this to shut everything down cleanly.
func (s *SystemService) StopAllWorkers(ctx context.Context) error {
	return s.manager.Stop(ctx)
}

// GetSystemHealth returns the aggregated health of workers and the database.
func (s *SystemService) GetSystemHealth(ctx context.Context) SystemHealth {
	health := SystemHealth{
		Status:  "ok",
		Workers: s.manager.GetWorkerStatus(),
	}

	// DB ping (guard against nil db used in tests).
	if s.db == nil {
		health.DBStatus = "error"
		health.Status = "degraded"
	} else if sqlDB, err := s.db.DB(); err != nil || sqlDB.PingContext(ctx) != nil {
		health.DBStatus = "error"
		health.Status = "degraded"
	} else {
		health.DBStatus = "ok"
	}

	// Any stopped worker degrades overall status.
	for _, ws := range health.Workers {
		if !ws.Running {
			health.Status = "degraded"
			break
		}
	}

	return health
}
