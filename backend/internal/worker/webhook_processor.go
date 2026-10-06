package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/gorm"
)

var retrySchedule = []time.Duration{
	30 * time.Minute, 1 * time.Hour, 2 * time.Hour,
	4 * time.Hour, 8 * time.Hour, 24 * time.Hour, 48 * time.Hour,
}

// WebhookProcessor is a background worker that retries failed webhook
// deliveries using an exponential back-off schedule defined by retrySchedule.
type WebhookProcessor struct {
	db         *gorm.DB
	webhookSvc *service.WebhookService
}

// NewWebhookProcessor creates a WebhookProcessor backed by db and webhookSvc.
func NewWebhookProcessor(db *gorm.DB, webhookSvc *service.WebhookService) *WebhookProcessor {
	return &WebhookProcessor{db: db, webhookSvc: webhookSvc}
}

// Name implements Worker and returns the human-readable identifier for this worker.
func (wp *WebhookProcessor) Name() string { return "webhook_processor" }

// Start implements Worker. It retries failed webhook deliveries on the
// configured back-off schedule. Runs until ctx is cancelled.
func (wp *WebhookProcessor) Start(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			wp.retryFailed()
		}
	}
}

func (wp *WebhookProcessor) retryFailed() {
	var logs []models.WebhookDeliveryLog
	wp.db.Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", "failed", time.Now()).
		Where("attempts < ?", len(retrySchedule)+1).
		Preload("Webhook").
		Limit(50).
		Find(&logs)

	for _, dl := range logs {
		if dl.Webhook == nil || !dl.Webhook.Active {
			continue
		}
		err := wp.webhookSvc.Deliver(dl.Webhook, dl.Event, dl.Payload)
		if err != nil {
			nextAttempt := dl.Attempts
			if nextAttempt < len(retrySchedule) {
				nextRetry := time.Now().Add(retrySchedule[nextAttempt])
				wp.db.Model(&dl).Updates(map[string]any{
					"attempts":      dl.Attempts + 1,
					"next_retry_at": nextRetry,
				})
			} else {
				wp.db.Model(&dl).Update("status", "dead_letter")
			}
			log.Printf("Webhook retry failed for delivery %d: %v", dl.ID, err)
		}
	}
}
