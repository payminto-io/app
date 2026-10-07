package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/service"
)

// SolanaDepositWatcher replaces the block-scanning processor for Solana: it polls signatures per
// watched account and advances deposits to finalized. See internal/blockchain/solana/README.md.
type SolanaDepositWatcher struct {
	svc             *service.SolanaDepositService
	pollInterval    time.Duration
	confirmInterval time.Duration
}

// NewSolanaDepositWatcher builds the worker; intervals default to 5s (poll) and 5s (confirm).
func NewSolanaDepositWatcher(svc *service.SolanaDepositService, poll, confirm time.Duration) *SolanaDepositWatcher {
	if poll <= 0 {
		poll = 5 * time.Second
	}
	if confirm <= 0 {
		confirm = 5 * time.Second
	}
	return &SolanaDepositWatcher{svc: svc, pollInterval: poll, confirmInterval: confirm}
}

// Name implements Worker.
func (w *SolanaDepositWatcher) Name() string { return "SOLANA_deposit_watcher" }

// Start runs the poll and confirmation loops until ctx is cancelled.
func (w *SolanaDepositWatcher) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Go(func() { w.loop(ctx, w.pollInterval, "poll", w.svc.PollOnce) })
	wg.Go(func() { w.loop(ctx, w.confirmInterval, "confirm", w.svc.ConfirmOnce) })
	wg.Wait()
	return nil
}

func (w *SolanaDepositWatcher) loop(ctx context.Context, every time.Duration, name string, fn func(context.Context) (int, error)) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if _, err := fn(ctx); err != nil && ctx.Err() == nil {
			log.Printf("[solana watcher] %s: %v", name, err)
		}
	}
}

var _ Worker = (*SolanaDepositWatcher)(nil)
