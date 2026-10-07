package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/service"
)

// SolanaSweepWorker sweeps confirmed SPL deposits into the hot wallet and completes finalized sweeps.
type SolanaSweepWorker struct {
	svc      *service.SolanaSweepService
	interval time.Duration
}

// NewSolanaSweepWorker builds the worker; interval defaults to 30s.
func NewSolanaSweepWorker(svc *service.SolanaSweepService, interval time.Duration) *SolanaSweepWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &SolanaSweepWorker{svc: svc, interval: interval}
}

// Name implements Worker.
func (w *SolanaSweepWorker) Name() string { return "solana_sweep" }

// Start implements Worker.
func (w *SolanaSweepWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	log.Printf("[solana_sweep] started (interval=%s)", w.interval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := w.svc.SweepConfirmed(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[solana_sweep] sweep round: %v", err)
			}
			if _, err := w.svc.TrackConfirmations(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[solana_sweep] confirmation tracking: %v", err)
			}
		}
	}
}

var _ Worker = (*SolanaSweepWorker)(nil)
