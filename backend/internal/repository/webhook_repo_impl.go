package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// WebhookRepository defines all persistence operations for Webhook records.
type WebhookRepository interface {
	Create(w *models.Webhook) error
	Update(w *models.Webhook) error
	Delete(id uint) error
	GetByID(id uint) (*models.Webhook, error)
	GetActiveByPlatform(platformID uint) ([]models.Webhook, error)
	ListByPlatform(platformID uint, opts ...QueryOption) ([]models.Webhook, error)
	SetActive(id uint, active bool) error
	UpdateSecret(id uint, newSecret string) error
	Count() (int64, error)
}

// WebhookRepositoryImpl is the GORM-backed implementation of WebhookRepository.
type WebhookRepositoryImpl struct {
	db *gorm.DB
}

// NewWebhookRepository constructs a new WebhookRepository backed by the provided *gorm.DB.
func NewWebhookRepository(db *gorm.DB) WebhookRepository {
	return &WebhookRepositoryImpl{db: db}
}

// Create inserts a new Webhook record.
func (r *WebhookRepositoryImpl) Create(w *models.Webhook) error {
	return r.db.Create(w).Error
}

// Update saves all fields of the given Webhook.
func (r *WebhookRepositoryImpl) Update(w *models.Webhook) error {
	return r.db.Save(w).Error
}

// Delete soft-deletes a Webhook by primary key.
func (r *WebhookRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.Webhook{}, id).Error
}

// GetByID fetches a Webhook by primary key.
func (r *WebhookRepositoryImpl) GetByID(id uint) (*models.Webhook, error) {
	var w models.Webhook
	if err := r.db.First(&w, id).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// GetActiveByPlatform returns all active webhooks for a given platform.
func (r *WebhookRepositoryImpl) GetActiveByPlatform(platformID uint) ([]models.Webhook, error) {
	var results []models.Webhook
	if err := r.db.
		Where("external_platform_id = ? AND active = ?", platformID, true).
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// ListByPlatform returns all Webhooks for a given platform, applying the
// provided QueryOptions. Results default to created_at DESC if no order is specified.
func (r *WebhookRepositoryImpl) ListByPlatform(platformID uint, opts ...QueryOption) ([]models.Webhook, error) {
	if len(opts) == 0 {
		opts = append(opts, WithDescendingOrder("created_at"))
	}
	q := Apply(r.db.Where("external_platform_id = ?", platformID), opts...)
	var results []models.Webhook
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// SetActive updates the active flag of a Webhook.
func (r *WebhookRepositoryImpl) SetActive(id uint, active bool) error {
	return r.db.Model(&models.Webhook{}).
		Where("id = ?", id).
		Update("active", active).Error
}

// UpdateSecret sets the secret field and updated_at timestamp for a Webhook.
func (r *WebhookRepositoryImpl) UpdateSecret(id uint, newSecret string) error {
	return r.db.Model(&models.Webhook{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"secret":     newSecret,
			"updated_at": time.Now(),
		}).Error
}

// Count returns the total number of Webhook records.
func (r *WebhookRepositoryImpl) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&models.Webhook{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
