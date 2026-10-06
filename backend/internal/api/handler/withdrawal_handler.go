package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

// WithdrawalHandler handles merchant withdrawal endpoints.
type WithdrawalHandler struct {
	withdrawalSvc *service.WithdrawalService
}

// NewWithdrawalHandler constructs a WithdrawalHandler.
func NewWithdrawalHandler(withdrawalSvc *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{withdrawalSvc: withdrawalSvc}
}

// createWithdrawalRequest is the request body for POST /withdrawal/merchant.
type createWithdrawalRequest struct {
	BlockchainCode string `json:"blockchainCode" binding:"required"`
	CurrencyCode   string `json:"currencyCode"   binding:"required"`
	Amount         string `json:"amount"         binding:"required"`
	ToAddress      string `json:"toAddress"      binding:"required"`
	Memo           string `json:"memo,omitempty"`
	CustomerID     string `json:"customerID,omitempty"`
	Email          string `json:"email,omitempty"`
}

// verifyWithdrawalOTPRequest is the request body for POST /withdrawal/:id/otp/verify.
type verifyWithdrawalOTPRequest struct {
	Code string `json:"code" binding:"required"`
}

// Create handles POST /api/v1/withdrawal/merchant.
func (h *WithdrawalHandler) Create(c *gin.Context) {
	var req createWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memberID := mustMemberID(c)
	platformID := mustPlatformID(c)

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	input := service.CreateWithdrawalInput{
		BlockchainCode:     req.BlockchainCode,
		CurrencyCode:       req.CurrencyCode,
		Amount:             amount,
		ToAddress:          req.ToAddress,
		Memo:               req.Memo,
		CustomerID:         req.CustomerID,
		Email:              req.Email,
		MemberID:           memberID,
		ExternalPlatformID: platformID,
	}

	withdrawal, prompt, err := h.withdrawalSvc.Create(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"withdrawal": withdrawal,
		"otp":        prompt,
	})
}

// VerifyOTP handles POST /api/v1/withdrawal/:id/otp/verify.
func (h *WithdrawalHandler) VerifyOTP(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal id"})
		return
	}

	var req verifyWithdrawalOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memberID := mustMemberID(c)

	if err := h.withdrawalSvc.VerifyOTP(withdrawalID, memberID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified, withdrawal pending approval"})
}

// Approve handles POST /api/v1/withdrawal/:id/approve.
func (h *WithdrawalHandler) Approve(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal id"})
		return
	}

	memberID := mustMemberID(c)
	platformID := mustPlatformID(c)

	if err := h.withdrawalSvc.ApproveForPlatform(withdrawalID, platformID, memberID); err != nil {
		if errors.Is(err, service.ErrWithdrawalNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrWithdrawalNotFound.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal approved"})
}

// Cancel handles POST /api/v1/withdrawal/:id/cancel.
func (h *WithdrawalHandler) Cancel(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal id"})
		return
	}

	platformID := mustPlatformID(c)
	if err := h.withdrawalSvc.CancelForPlatform(withdrawalID, platformID); err != nil {
		if errors.Is(err, service.ErrWithdrawalNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrWithdrawalNotFound.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal cancelled"})
}

// GetByID handles GET /api/v1/withdrawal/:id.
func (h *WithdrawalHandler) GetByID(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal id"})
		return
	}

	platformID := mustPlatformID(c)
	w, err := h.withdrawalSvc.GetByIDForPlatform(withdrawalID, platformID)
	if err != nil {
		if errors.Is(err, service.ErrWithdrawalNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrWithdrawalNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawal": w})
}

// List handles GET /api/v1/withdrawal/merchant.
func (h *WithdrawalHandler) List(c *gin.Context) {
	platformID := mustPlatformID(c)

	withdrawals, err := h.withdrawalSvc.ListByPlatform(platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawals": withdrawals})
}

// mustMemberID extracts memberID from the Gin context (set by middleware).
// Returns 0 if not present.
func mustMemberID(c *gin.Context) uint {
	v, _ := c.Get("memberID")
	id, _ := v.(uint)
	return id
}

// mustPlatformID extracts externalPlatformID from the Gin context.
func mustPlatformID(c *gin.Context) uint {
	v, _ := c.Get("externalPlatformID")
	id, _ := v.(uint)
	return id
}

// parseIDParam parses a route parameter as uint.
func parseIDParam(c *gin.Context, param string) (uint, error) {
	s := c.Param(param)
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
