package worker

import (
	"context"
	"log/slog"
	"time"
)

// linkResolver is the slice of links.Service the worker needs.
type linkResolver interface {
	ResolveExpired(ctx context.Context, limit int) (completed, released int, err error)
}

// LinkReservationResolver settles payment-link uses whose lease ended without an outcome
// (internal/links/README.md, "Paying a link").
type LinkReservationResolver struct {
	links linkResolver
	every time.Duration
	batch int
}

func NewLinkReservationResolver(links linkResolver, every time.Duration) *LinkReservationResolver {
	return &LinkReservationResolver{links: links, every: every, batch: 100}
}

func (w *LinkReservationResolver) Name() string { return "link_reservation_resolver" }

// Start resolves on every tick until ctx is cancelled; a full batch is followed at once by the next.
func (w *LinkReservationResolver) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for {
				completed, released, err := w.links.ResolveExpired(ctx, w.batch)
				if err != nil {
					slog.Error("link reservation resolver", "error", err)
				}
				if completed+released > 0 {
					slog.Info("link reservations resolved", "completed", completed, "released", released)
				}
				if err != nil || completed+released < w.batch || ctx.Err() != nil {
					break
				}
			}
		}
	}
}
