package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// RecipientHandler handles merchant payee book endpoints.
type RecipientHandler struct {
	recipientSvc *service.RecipientService
}

// NewRecipientHandler constructs a RecipientHandler.
func NewRecipientHandler(recipientSvc *service.RecipientService) *RecipientHandler {
	return &RecipientHandler{recipientSvc: recipientSvc}
}

// callerMemberID extracts the authenticated member ID from the Gin context.
func callerMemberID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("memberID")
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok && id != 0
}

// ListRecipients handles GET /api/v1/recipients.
func (h *RecipientHandler) ListRecipients(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	recs, err := h.recipientSvc.ListByMember(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recipients": recs})
}

// GetRecipient handles GET /api/v1/recipients/:id.
func (h *RecipientHandler) GetRecipient(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient id"})
		return
	}
	rec, err := h.recipientSvc.GetByID(c.Request.Context(), uint(id), memberID)
	if err != nil {
		if errors.Is(err, service.ErrRecipientNotOwned) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipient not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// createRecipientRequest is the payload for POST /api/v1/recipients. The
// external platform is derived from the authenticated caller's token — never
// the request body — so recipients always belong to the correct tenant.
type createRecipientRequest struct {
	Name           string `json:"name" binding:"required"`
	Email          string `json:"email"`
	BlockchainCode string `json:"blockchainCode" binding:"required"`
	CurrencyCode   string `json:"currencyCode" binding:"required"`
	Address        string `json:"address" binding:"required"`
	Memo           string `json:"memo"`
}

// CreateRecipient handles POST /api/v1/recipients.
func (h *RecipientHandler) CreateRecipient(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	platformIDVal, _ := c.Get("externalPlatformID")
	platformID, _ := platformIDVal.(uint)
	if platformID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing platform context"})
		return
	}
	var req createRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rec, err := h.recipientSvc.Create(c.Request.Context(), service.CreateRecipientInput{
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		Name:               req.Name,
		Email:              req.Email,
		BlockchainCode:     req.BlockchainCode,
		CurrencyCode:       req.CurrencyCode,
		Address:            req.Address,
		Memo:               req.Memo,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rec)
}

// updateRecipientRequest is the payload for PUT /api/v1/recipients/:id.
type updateRecipientRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Memo    string `json:"memo"`
	Address string `json:"address"`
}

// UpdateRecipient handles PUT /api/v1/recipients/:id.
func (h *RecipientHandler) UpdateRecipient(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient id"})
		return
	}
	var req updateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rec, err := h.recipientSvc.Update(c.Request.Context(), uint(id), memberID, service.UpdateRecipientInput{
		Name:    req.Name,
		Email:   req.Email,
		Memo:    req.Memo,
		Address: req.Address,
	})
	if err != nil {
		if errors.Is(err, service.ErrRecipientNotOwned) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipient not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// DeleteRecipient handles DELETE /api/v1/recipients/:id.
func (h *RecipientHandler) DeleteRecipient(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient id"})
		return
	}
	if err := h.recipientSvc.Delete(c.Request.Context(), uint(id), memberID); err != nil {
		if errors.Is(err, service.ErrRecipientNotOwned) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipient not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "recipient deleted"})
}
