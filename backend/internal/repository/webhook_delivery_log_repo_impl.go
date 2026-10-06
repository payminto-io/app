package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// WebhookDeliveryLogRepository defines all persistence operations for WebhookDeliveryLog records.
type WebhookDeliveryLogRepository interface {
	Create(l *models.WebhookDeliveryLog) error
	Update(l *models.WebhookDeliveryLog) error
	GetByID(id uint) (*models.WebhookDeliveryLog, error)
	ListByWebhookID(webhookID uint, opts ...QueryOption) ([]models.WebhookDeliveryLog, error)
	ListByStatus(status string, limit int) ([]models.WebhookDeliveryLog, error)
	ListPendingRetries(now time.Time, limit int) ([]models.WebhookDeliveryLog, error)
	UpdateStatus(id uint, status string, responseCode *int, responseBody *string) error
	IncrementAttempts(id uint, nextRetryAt *time.Time) error
	MarkDelivered(id uint, responseCode int) error
}

// WebhookDeliveryLogRepositoryImpl is the GORM-backed implementation of WebhookDeliveryLogRepository.
type WebhookDeliveryLogRepositoryImpl struct {
	db *gorm.DB
}

// NewWebhookDeliveryLogRepository constructs a new WebhookDeliveryLogRepository backed by the provided *gorm.DB.
func NewWebhookDeliveryLogRepository(db *gorm.DB) WebhookDeliveryLogRepository {
	return &WebhookDeliveryLogRepositoryImpl{db: db}
}

// Create inserts a new WebhookDeliveryLog record.
func (r *WebhookDeliveryLogRepositoryImpl) Create(l *models.WebhookDeliveryLog) error {
	return r.db.Create(l).Error
}

// Update saves all fields of the given WebhookDeliveryLog.
func (r *WebhookDeliveryLogRepositoryImpl) Update(l *models.WebhookDeliveryLog) error {
	return r.db.Save(l).Error
}

// GetByID fetches a WebhookDeliveryLog by primary key, preloading its Webhook.
func (r *WebhookDeliveryLogRepositoryImpl) GetByID(id uint) (*models.WebhookDeliveryLog, error) {
	var l models.WebhookDeliveryLog
	if err := r.db.Preload("Webhook").First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// ListByWebhookID returns all delivery logs for a given webhook, applying the
// provided QueryOptions. Results default to created_at DESC if no order is specified.
func (r *WebhookDeliveryLogRepositoryImpl) ListByWebhookID(webhookID uint, opts ...QueryOption) ([]models.WebhookDeliveryLog, error) {
	if len(opts) == 0 {
		opts = append(opts, WithDescendingOrder("created_at"))
	}
	q := Apply(r.db.Where("webhook_id = ?", webhookID), opts...)
	var results []models.WebhookDeliveryLog
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// ListByStatus returns delivery logs filtered by status, preloading Webhook,
// ordered by created_at ASC, up to limit rows.
func (r *WebhookDeliveryLogRepositoryImpl) ListByStatus(status string, limit int) ([]models.WebhookDeliveryLog, error) {
	var results []models.WebhookDeliveryLog
	q := r.db.Preload("Webhook").
		Where("status = ?", status).
		Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// ListPendingRetries returns failed delivery logs eligible for retry, preloading
// Webhook. A log is eligible when next_retry_at IS NULL or next_retry_at <= now.
func (r *WebhookDeliveryLogRepositoryImpl) ListPendingRetries(now time.Time, limit int) ([]models.WebhookDeliveryLog, error) {
	var results []models.WebhookDeliveryLog
	q := r.db.Preload("Webhook").
		Where("status = 'failed' AND (next_retry_at IS NULL OR next_retry_at <= ?)", now).
		Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateStatus sets status, response_code, and response_body on a delivery log.
func (r *WebhookDeliveryLogRepositoryImpl) UpdateStatus(id uint, status string, responseCode *int, responseBody *string) error {
	updates := map[string]any{
		"status":        status,
		"response_code": responseCode,
		"response_body": responseBody,
	}
	return r.db.Model(&models.WebhookDeliveryLog{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// IncrementAttempts increments the attempts counter and sets next_retry_at.
func (r *WebhookDeliveryLogRepositoryImpl) IncrementAttempts(id uint, nextRetryAt *time.Time) error {
	return r.db.Model(&models.WebhookDeliveryLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":      gorm.Expr("attempts + 1"),
			"next_retry_at": nextRetryAt,
		}).Error
}

// MarkDelivered sets status to 'delivered', records delivered_at and response_code.
func (r *WebhookDeliveryLogRepositoryImpl) MarkDelivered(id uint, responseCode int) error {
	now := time.Now()
	return r.db.Model(&models.WebhookDeliveryLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        "delivered",
			"delivered_at":  now,
			"response_code": responseCode,
		}).Error
}
