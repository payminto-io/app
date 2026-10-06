package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/service"
)

// WebhookHandler handles webhook CRUD endpoints for merchants.
type WebhookHandler struct {
	mgmtSvc *service.WebhookManagementService
}

// NewWebhookHandler constructs a WebhookHandler.
func NewWebhookHandler(mgmtSvc *service.WebhookManagementService) *WebhookHandler {
	return &WebhookHandler{mgmtSvc: mgmtSvc}
}

// createWebhookRequest is the body for POST /api/v1/webhooks.
type createWebhookRequest struct {
	URL    string   `json:"url" binding:"required,url"`
	Events []string `json:"events" binding:"required,min=1,dive,required"`
	Active bool     `json:"active"`
	// UnsupportedDescription detects the removed field so old clients receive
	// an explicit error instead of silently losing operator-authored text.
	UnsupportedDescription *json.RawMessage `json:"description"`
}

// updateWebhookRequest is the body for PUT /api/v1/webhooks/:id.
type updateWebhookRequest struct {
	URL                    string           `json:"url,omitempty"`
	Events                 *[]string        `json:"events,omitempty"`
	Active                 *bool            `json:"active,omitempty"`
	RegenerateSecret       bool             `json:"regenerateSecret,omitempty"`
	UnsupportedDescription *json.RawMessage `json:"description"`
}

// webhookResponse is the canonical HTTP representation. The persistence
// model keeps subscriptions in one text column for compatibility, while the
// HTTP interface consistently exposes a JSON array.
type webhookResponse struct {
	ID                 uint      `json:"id"`
	ExternalPlatformID uint      `json:"externalPlatformID"`
	URL                string    `json:"url"`
	Events             []string  `json:"events"`
	Active             bool      `json:"active"`
	Secret             string    `json:"secret,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func webhookToResponse(w *models.Webhook, revealSecret bool) webhookResponse {
	response := webhookResponse{
		ID:                 w.ID,
		ExternalPlatformID: w.ExternalPlatformID,
		URL:                w.URL,
		Events:             decodeWebhookEvents(w.Events),
		Active:             w.Active,
		CreatedAt:          w.CreatedAt,
		UpdatedAt:          w.UpdatedAt,
	}
	if revealSecret {
		response.Secret = w.Secret
	}
	return response
}

func canonicalWebhookEvents(events []string) ([]string, error) {
	if len(events) == 0 {
		return nil, errors.New("events must contain at least one event")
	}
	seen := make(map[string]struct{}, len(events))
	canonical := make([]string, 0, len(events))
	for _, raw := range events {
		event := strings.TrimSpace(raw)
		if event == "" {
			return nil, errors.New("events must not contain empty values")
		}
		if strings.Contains(event, ",") {
			return nil, errors.New("each event must be a separate array value")
		}
		if _, duplicate := seen[event]; duplicate {
			continue
		}
		seen[event] = struct{}{}
		canonical = append(canonical, event)
	}
	return canonical, nil
}

func encodeWebhookEvents(events []string) string {
	return strings.Join(events, ",")
}

func decodeWebhookEvents(events string) []string {
	if strings.TrimSpace(events) == "" {
		return []string{}
	}
	parts := strings.Split(events, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, event := range parts {
		if event = strings.TrimSpace(event); event != "" {
			if _, duplicate := seen[event]; duplicate {
				continue
			}
			seen[event] = struct{}{}
			result = append(result, event)
		}
	}
	return result
}

// CreateWebhook handles POST /api/v1/webhooks.
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	var req createWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.UnsupportedDescription != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description is not supported"})
		return
	}
	events, err := canonicalWebhookEvents(req.Events)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	platformID := mustPlatformID(c)

	w, err := h.mgmtSvc.Create(service.CreateWebhookInput{
		PlatformID: platformID,
		URL:        req.URL,
		Events:     encodeWebhookEvents(events),
		Active:     req.Active,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"webhook": webhookToResponse(w, true)})
}

// ListWebhooks handles GET /api/v1/webhooks.
func (h *WebhookHandler) ListWebhooks(c *gin.Context) {
	platformID := mustPlatformID(c)

	webhooks, err := h.mgmtSvc.ListByPlatform(platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]webhookResponse, 0, len(webhooks))
	for i := range webhooks {
		responses = append(responses, webhookToResponse(&webhooks[i], false))
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": responses})
}

// GetWebhook handles GET /api/v1/webhooks/:id.
func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}
	platformID := mustPlatformID(c)

	w, err := h.mgmtSvc.GetByID(id, platformID)
	if err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook": webhookToResponse(w, false)})
}

// UpdateWebhook handles PUT /api/v1/webhooks/:id.
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}
	platformID := mustPlatformID(c)

	var req updateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.UnsupportedDescription != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description is not supported"})
		return
	}
	var events *string
	if req.Events != nil {
		canonical, canonicalErr := canonicalWebhookEvents(*req.Events)
		if canonicalErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": canonicalErr.Error()})
			return
		}
		encoded := encodeWebhookEvents(canonical)
		events = &encoded
	}

	w, err := h.mgmtSvc.Update(id, platformID, service.UpdateWebhookInput{
		URL:              req.URL,
		Events:           events,
		Active:           req.Active,
		RegenerateSecret: req.RegenerateSecret,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook": webhookToResponse(w, req.RegenerateSecret)})
}

// DeleteWebhook handles DELETE /api/v1/webhooks/:id.
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}
	platformID := mustPlatformID(c)

	if err := h.mgmtSvc.Delete(id, platformID); err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook deleted"})
}

// RegenerateSecret handles POST /api/v1/webhooks/:id/regenerate-secret.
func (h *WebhookHandler) RegenerateSecret(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}
	platformID := mustPlatformID(c)

	secret, err := h.mgmtSvc.RegenerateSecret(id, platformID)
	if err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

// ListDeliveries handles GET /api/v1/webhooks/:id/deliveries.
func (h *WebhookHandler) ListDeliveries(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}
	platformID := mustPlatformID(c)

	logs, err := h.mgmtSvc.ListDeliveries(id, platformID)
	if err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deliveries": logs})
}

// isNotFound returns true for webhook-not-found or tenant-mismatch errors.
func isNotFound(err error) bool {
	return err == service.ErrWebhookNotFound || err == service.ErrWebhookTenantMismatch
}
