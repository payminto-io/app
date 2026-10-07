package paymentswitch

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Reconciler periodically syncs every attempt and refund whose connector outcome is unknown (I4). It backs off
// per row and, past MaxAge, records a stale_in_flight anomaly and logs at error level instead of inventing a
// status. It satisfies the worker manager's Worker interface.
type Reconciler struct {
	svc      *Service
	Interval time.Duration
	MaxAge   time.Duration
	Batch    int
}

func NewReconciler(svc *Service) *Reconciler {
	return &Reconciler{svc: svc, Interval: 30 * time.Second, MaxAge: 24 * time.Hour, Batch: 100}
}

func (r *Reconciler) Name() string { return "switch_reconciler" }

func (r *Reconciler) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := r.RunOnce(ctx); err != nil {
				r.svc.logf("[switch_reconciler] run: %v", err)
			}
		}
	}
}

// RunOnce syncs one batch of due rows and returns how many it touched.
func (r *Reconciler) RunOnce(ctx context.Context) (int, error) {
	now := r.svc.now()
	db := r.svc.db.WithContext(ctx)
	touched := 0

	var attempts []AttemptRow
	err := db.Where("status IN ? AND (next_sync_at IS NULL OR next_sync_at <= ?)", []AttemptStatus{AttemptPending, AttemptCaptureInitiated, AttemptVoidInitiated}, now).
		Order("created_at").Limit(r.Batch).Find(&attempts).Error
	if err != nil {
		return 0, fmt.Errorf("reconciler: load attempts: %w", err)
	}
	for _, a := range attempts {
		touched++
		if now.Sub(a.CreatedAt) > r.MaxAge {
			if err := r.stale(ctx, "attempt", a.ID, a.IntentID, a.Status, a.CreatedAt); err != nil {
				return touched, err
			}
		}
		if _, err := r.svc.syncAttempt(ctx, a); err != nil {
			r.svc.logf("[switch_reconciler] attempt %s: %v", a.ID, err)
		}
	}

	var refunds []RefundRow
	err = db.Where("status IN ? AND (next_sync_at IS NULL OR next_sync_at <= ?)", []RefundStatus{RefundInitiated, RefundPending}, now).
		Order("created_at").Limit(r.Batch).Find(&refunds).Error
	if err != nil {
		return touched, fmt.Errorf("reconciler: load refunds: %w", err)
	}
	for _, rf := range refunds {
		touched++
		if now.Sub(rf.CreatedAt) > r.MaxAge {
			if err := r.stale(ctx, "refund", rf.ID, rf.IntentID, rf.Status, rf.CreatedAt); err != nil {
				return touched, err
			}
		}
		if err := r.svc.syncRefund(ctx, rf); err != nil {
			r.svc.logf("[switch_reconciler] refund %s: %v", rf.ID, err)
		}
	}
	return touched, nil
}

func (r *Reconciler) stale(ctx context.Context, entity, id, intentID string, status any, since time.Time) error {
	detail := fmt.Sprintf("%s %s has been %v since %s, past the %s reconciliation limit", entity, id, status, since.UTC().Format(time.RFC3339), r.MaxAge)
	return r.svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.svc.recordAnomaly(tx, entity, id, intentID, AnomalyStaleInFlight, detail)
	})
}
