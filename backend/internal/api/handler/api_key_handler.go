package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
)

// APIKeyHandler serves the merchant-facing /api-keys CRUD surface.
// Merchants manage API keys scoped to their external platform.
type APIKeyHandler struct {
	apiKeyRepo repository.APIKeyRepository
}

// NewAPIKeyHandler wires the handler.
func NewAPIKeyHandler(apiKeyRepo repository.APIKeyRepository) *APIKeyHandler {
	return &APIKeyHandler{apiKeyRepo: apiKeyRepo}
}

// apiKeyResponse is the public projection returned to the frontend.
// The raw key is included only on create/regenerate (revealed once).
type apiKeyResponse struct {
	ID                 uint    `json:"id"`
	Name               string  `json:"name"`
	Prefix             string  `json:"prefix"` // first 8 chars e.g. "pm_abcd..."
	Active             bool    `json:"active"`
	Status             string  `json:"status"`
	ExternalPlatformID uint    `json:"externalPlatformID"`
	MemberID           *uint   `json:"memberID,omitempty"`
	Description        *string `json:"description,omitempty"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
	RawKey             *string `json:"rawKey,omitempty"` // populated only on create/regenerate
}

func toAPIKeyResponse(k *models.APIKey, rawKey string) apiKeyResponse {
	prefix := ""
	if len(k.Key) >= 10 {
		prefix = k.Key[:10] + "..."
	} else if rawKey != "" && len(rawKey) >= 10 {
		prefix = rawKey[:10] + "..."
	}
	name := "API Key"
	if k.Description != nil && *k.Description != "" {
		name = *k.Description
	}
	resp := apiKeyResponse{
		ID:                 k.ID,
		Name:               name,
		Prefix:             prefix,
		Active:             k.Status == "active",
		Status:             k.Status,
		ExternalPlatformID: k.ExternalPlatformID,
		MemberID:           k.MemberID,
		Description:        k.Description,
		CreatedAt:          k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          k.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if rawKey != "" {
		resp.RawKey = &rawKey
	}
	return resp
}

// ListAPIKeys handles GET /api/v1/api-keys — returns all API keys for the
// caller's external platform.
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	platformID := mustPlatformID(c)
	if platformID == 0 {
		c.JSON(http.StatusOK, gin.H{"apiKeys": []apiKeyResponse{}})
		return
	}

	keys, err := h.apiKeyRepo.ListByExternalPlatformID(platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]apiKeyResponse, 0, len(keys))
	for i := range keys {
		out = append(out, toAPIKeyResponse(&keys[i], ""))
	}
	c.JSON(http.StatusOK, gin.H{"apiKeys": out})
}

type createAPIKeyRequest struct {
	Name   string `json:"name"`
	RoleID *uint  `json:"roleID,omitempty"`
}

// CreateAPIKey handles POST /api/v1/api-keys — generates a new API key,
// stores its SHA-256 hash, and returns the plaintext key exactly once.
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	platformID := mustPlatformID(c)
	if platformID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no platform associated with your account"})
		return
	}
	memberIDVal, _ := c.Get("memberID")
	memberID, _ := memberIDVal.(uint)

	var req createAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		req.Name = "API Key"
	}

	rawKey, err := service.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	hashed := service.HashAPIKey(rawKey)

	mid := memberID
	var mPtr *uint
	if mid > 0 {
		mPtr = &mid
	}
	name := req.Name
	apiKey := &models.APIKey{
		Key:                hashed,
		Status:             "active",
		ExternalPlatformID: platformID,
		MemberID:           mPtr,
		RoleID:             req.RoleID,
		Description:        &name,
	}
	if err := h.apiKeyRepo.Create(apiKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the plaintext key exactly once (revealed on create).
	// Replace the stored hash with the raw key in the response prefix.
	resp := toAPIKeyResponse(apiKey, rawKey)
	c.JSON(http.StatusCreated, resp)
}

// RevokeAPIKey handles POST /api/v1/api-keys/:id/revoke — sets status to "revoked".
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	platformID := mustPlatformID(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	key, err := h.apiKeyRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
		return
	}
	if key.ExternalPlatformID != platformID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	key.Status = "revoked"
	if err := h.apiKeyRepo.Update(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toAPIKeyResponse(key, ""))
}
