package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/service"
)

// TempoMemoWatcher polls the Tempo memo service. Tempo has no block processor: blocks carry
// transaction types go-ethereum cannot decode, so deposits are found from TransferWithMemo logs.
type TempoMemoWatcher struct {
	svc   *service.TempoMemoService
	every time.Duration
}

// NewTempoMemoWatcher builds the worker; the interval defaults to 2s.
func NewTempoMemoWatcher(svc *service.TempoMemoService, every time.Duration) *TempoMemoWatcher {
	if every <= 0 {
		every = 2 * time.Second
	}
	return &TempoMemoWatcher{svc: svc, every: every}
}

// Name implements Worker.
func (w *TempoMemoWatcher) Name() string { return "TEMPO_memo_watcher" }

// Start polls until ctx is cancelled.
func (w *TempoMemoWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.every)
	defer ticker.Stop()
	for {
		if n, err := w.svc.PollOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("[tempo watcher] poll: %v", err)
		} else if n > 0 {
			log.Printf("[tempo watcher] credited %d payment(s)", n)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

var _ Worker = (*TempoMemoWatcher)(nil)
