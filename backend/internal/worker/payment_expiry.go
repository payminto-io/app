package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/service"
)

// PaymentExpiryWorker is a background worker that periodically calls
// PaymentService.ExpireStalePayments to mark timed-out invoices as expired.
type PaymentExpiryWorker struct {
	paymentSvc *service.PaymentService
}

// NewPaymentExpiryWorker constructs a PaymentExpiryWorker backed by paymentSvc.
func NewPaymentExpiryWorker(paymentSvc *service.PaymentService) *PaymentExpiryWorker {
	return &PaymentExpiryWorker{paymentSvc: paymentSvc}
}

// Name implements Worker and returns the human-readable identifier for this worker.
func (w *PaymentExpiryWorker) Name() string { return "payment_expiry" }

// Start implements Worker. It periodically calls PaymentService.ExpireStalePayments
// to mark timed-out invoices as expired. Runs until ctx is cancelled.
func (w *PaymentExpiryWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			expired, err := w.paymentSvc.ExpireStalePayments()
			if err != nil {
				log.Printf("Payment expiry error: %v", err)
			} else if expired > 0 {
				log.Printf("Expired %d stale payments", expired)
			}
		}
	}
}
