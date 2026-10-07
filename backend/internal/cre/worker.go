package cre

import (
	"context"
	"time"

	"github.com/payminto/payminto/backend/internal/observability"
)

// Runner is the background loop; it satisfies worker.Worker without importing it. Nil when the module is off.
type Runner struct {
	svc *Service
}

func (s *Service) Worker() *Runner {
	if !s.Enabled() {
		return nil
	}
	return &Runner{svc: s}
}

func (r *Runner) Name() string { return "cre-attester" }

// Start polls every provider cursor, refreshes the liabilities checkpoint, and sweeps for staleness.
// Only the mock provider is triggered on a schedule; deployed workflows pull on their own cron (SPEC section 5).
func (r *Runner) Start(ctx context.Context) error {
	cfg := r.svc.cfg
	poll := time.NewTicker(cfg.PollInterval)
	solvency := time.NewTicker(cfg.SolvencyInterval)
	finality := time.NewTicker(cfg.FinalityBatchInterval)
	conversion := time.NewTicker(cfg.ConversionInterval)
	sweep := time.NewTicker(maxDuration(cfg.FinalityBatchInterval, time.Minute))
	defer poll.Stop()
	defer solvency.Stop()
	defer finality.Stop()
	defer conversion.Stop()
	defer sweep.Stop()

	r.tick(ctx, KindSolvency, true)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-poll.C:
			for _, kind := range Kinds {
				if _, err := r.svc.Poll(ctx, kind); err != nil {
					observability.Logger().Error("cre: poll", "kind", kind, "err", err)
				}
			}
		case <-solvency.C:
			r.tick(ctx, KindSolvency, true)
		case <-finality.C:
			r.tick(ctx, KindDepositFinality, false)
		case <-conversion.C:
			r.tick(ctx, KindConversionReference, false)
		case <-sweep.C:
			if err := r.svc.SweepStale(ctx); err != nil {
				observability.Logger().Error("cre: stale sweep", "err", err)
			}
		}
	}
}

func (r *Runner) tick(ctx context.Context, kind Kind, publish bool) {
	if publish {
		if _, err := r.svc.PublishCheckpoint(ctx); err != nil {
			observability.Logger().Error("cre: publish checkpoint", "err", err)
		}
	}
	if r.svc.cfg.Provider != ProviderMock {
		return
	}
	if _, err := r.svc.Run(ctx, kind); err != nil {
		observability.Logger().Warn("cre: mock run", "kind", kind, "err", err)
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
