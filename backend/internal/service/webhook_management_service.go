package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// ErrWebhookNotFound is returned when a webhook ID does not exist or belongs to
// a different platform.
var ErrWebhookNotFound = errors.New("webhook not found")

// ErrWebhookTenantMismatch is returned when the caller's platform does not match
// the webhook's platform.
var ErrWebhookTenantMismatch = errors.New("webhook belongs to a different platform")

// CreateWebhookInput contains the fields required to register a new webhook.
type CreateWebhookInput struct {
	PlatformID uint
	URL        string
	Events     string
	Active     bool
}

// UpdateWebhookInput contains the fields that can be mutated on an existing webhook.
type UpdateWebhookInput struct {
	URL              string
	Events           *string
	Active           *bool
	RegenerateSecret bool
}

// WebhookManagementService handles webhook endpoint CRUD for merchants. Delivery
// (the outbound HTTP POST) is performed by WebhookService; this service manages
// the configuration layer.
type WebhookManagementService struct {
	webhookRepo     repository.WebhookRepository
	deliveryLogRepo repository.WebhookDeliveryLogRepository
}

// NewWebhookManagementService constructs a WebhookManagementService.
func NewWebhookManagementService(
	webhookRepo repository.WebhookRepository,
	deliveryLogRepo repository.WebhookDeliveryLogRepository,
) *WebhookManagementService {
	return &WebhookManagementService{
		webhookRepo:     webhookRepo,
		deliveryLogRepo: deliveryLogRepo,
	}
}

// generateSecret produces a 32-byte cryptographically random hex string.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate webhook secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create registers a new webhook endpoint for the given platform, generating a
// signing secret automatically.
func (s *WebhookManagementService) Create(input CreateWebhookInput) (*models.Webhook, error) {
	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}

	w := &models.Webhook{
		ExternalPlatformID: input.PlatformID,
		URL:                input.URL,
		Secret:             secret,
		Events:             input.Events,
		Active:             input.Active,
	}

	if err := s.webhookRepo.Create(w); err != nil {
		return nil, fmt.Errorf("webhook create: %w", err)
	}
	return w, nil
}

// GetByID returns a webhook by ID, enforcing tenant isolation via platformID.
func (s *WebhookManagementService) GetByID(id, platformID uint) (*models.Webhook, error) {
	w, err := s.webhookRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	if w.ExternalPlatformID != platformID {
		return nil, ErrWebhookTenantMismatch
	}
	return w, nil
}

// ListByPlatform returns all webhooks registered for a platform.
func (s *WebhookManagementService) ListByPlatform(platformID uint) ([]models.Webhook, error) {
	return s.webhookRepo.ListByPlatform(platformID)
}

// Update mutates allowed fields on an existing webhook. If RegenerateSecret is
// true a new signing secret is generated and stored.
func (s *WebhookManagementService) Update(id, platformID uint, input UpdateWebhookInput) (*models.Webhook, error) {
	w, err := s.GetByID(id, platformID)
	if err != nil {
		return nil, err
	}

	if input.URL != "" {
		w.URL = input.URL
	}
	if input.Events != nil {
		w.Events = *input.Events
	}
	if input.Active != nil {
		w.Active = *input.Active
	}

	if input.RegenerateSecret {
		secret, err := generateSecret()
		if err != nil {
			return nil, err
		}
		w.Secret = secret
	}

	if err := s.webhookRepo.Update(w); err != nil {
		return nil, fmt.Errorf("webhook update: %w", err)
	}
	return w, nil
}

// Delete soft-deletes the webhook after verifying platform ownership.
func (s *WebhookManagementService) Delete(id, platformID uint) error {
	if _, err := s.GetByID(id, platformID); err != nil {
		return err
	}
	return s.webhookRepo.Delete(id)
}

// RegenerateSecret rotates the signing secret for a webhook.
func (s *WebhookManagementService) RegenerateSecret(id, platformID uint) (string, error) {
	if _, err := s.GetByID(id, platformID); err != nil {
		return "", err
	}
	secret, err := generateSecret()
	if err != nil {
		return "", err
	}
	if err := s.webhookRepo.UpdateSecret(id, secret); err != nil {
		return "", fmt.Errorf("rotate secret: %w", err)
	}
	return secret, nil
}

// ListDeliveries returns delivery logs for a webhook, enforcing platform ownership.
func (s *WebhookManagementService) ListDeliveries(id, platformID uint) ([]models.WebhookDeliveryLog, error) {
	if _, err := s.GetByID(id, platformID); err != nil {
		return nil, err
	}
	return s.deliveryLogRepo.ListByWebhookID(id)
}
