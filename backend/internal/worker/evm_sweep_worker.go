package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/service"
)

// EVMSweepWorker periodically (1) sweeps confirmed native-EVM deposits into cold
// storage and (2) tracks broadcast sweep txs to confirmation. It is a thin
// scheduler — all logic lives in the services — registered only when an EVM
// cold wallet is configured.
type EVMSweepWorker struct {
	svc       *service.EVMSweepService
	confirmer *service.EVMSweepConfirmer
	interval  time.Duration
}

// NewEVMSweepWorker constructs the worker. interval defaults to 30s when zero.
// confirmer may be nil to disable confirmation tracking.
func NewEVMSweepWorker(svc *service.EVMSweepService, confirmer *service.EVMSweepConfirmer, interval time.Duration) *EVMSweepWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &EVMSweepWorker{svc: svc, confirmer: confirmer, interval: interval}
}

// Name implements Worker.
func (w *EVMSweepWorker) Name() string { return "evm_sweep" }

// Start implements Worker — each tick broadcasts new sweeps then advances
// pending sweeps' confirmation state, until ctx is cancelled.
func (w *EVMSweepWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	log.Printf("[evm_sweep] started (interval=%s)", w.interval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := w.svc.SweepConfirmedNative(ctx); err != nil {
				log.Printf("[evm_sweep] sweep round error: %v", err)
			}
			if w.confirmer != nil {
				if _, err := w.confirmer.TrackConfirmations(ctx); err != nil {
					log.Printf("[evm_sweep] confirmation tracking error: %v", err)
				}
			}
		}
	}
}
