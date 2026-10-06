package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

// ReferralHandler exposes referral endpoints for the authenticated merchant (referrer's own view).
type ReferralHandler struct {
	referralSvc          *service.ReferralService
	analyticsReferralSvc *service.AnalyticsReferralService
}

// NewReferralHandler constructs a ReferralHandler.
func NewReferralHandler(referralSvc *service.ReferralService, analyticsReferralSvc *service.AnalyticsReferralService) *ReferralHandler {
	return &ReferralHandler{
		referralSvc:          referralSvc,
		analyticsReferralSvc: analyticsReferralSvc,
	}
}

// GetMyCode handles GET /api/v1/referrals/code — returns or creates the caller's referral code.
func (h *ReferralHandler) GetMyCode(c *gin.Context) {
	memberID := mustMemberID(c)

	code, err := h.referralSvc.GetMemberReferralCode(memberID, "payminto")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"referralCode": code})
}

// RegisterReferral handles POST /api/v1/referrals/register — links caller to a referrer's code.
type registerReferralRequest struct {
	ReferralCode string `json:"referralCode" binding:"required"`
}

// Register handles POST /api/v1/referrals/register.
func (h *ReferralHandler) Register(c *gin.Context) {
	var req registerReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memberID := mustMemberID(c)

	if err := h.referralSvc.RegisterReferral(req.ReferralCode, memberID, "payminto"); err != nil {
		status := http.StatusBadRequest
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "referral registered"})
}

// ListMyRewards handles GET /api/v1/referrals/rewards — caller's earned rewards.
func (h *ReferralHandler) ListMyRewards(c *gin.Context) {
	memberID := mustMemberID(c)

	rewards, err := h.referralSvc.ListEventsByMember(memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rewards": rewards})
}

// GetMyStats handles GET /api/v1/referrals/stats — caller's referral stats.
func (h *ReferralHandler) GetMyStats(c *gin.Context) {
	memberID := mustMemberID(c)

	stats, err := h.analyticsReferralSvc.GetReferralStats(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// RecordPaymentRequest is the body for POST /api/v1/referrals/record-payment.
type recordPaymentRequest struct {
	RefereeMemberID uint   `json:"refereeMemberID" binding:"required"`
	AmountUSD       string `json:"amountUSD" binding:"required"`
}

// RecordPayment handles POST /api/v1/referrals/record-payment (internal / admin use).
func (h *ReferralHandler) RecordPayment(c *gin.Context) {
	var req recordPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	amount, err := decimal.NewFromString(req.AmountUSD)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amountUSD"})
		return
	}

	if err := h.referralSvc.RecordPayment(req.RefereeMemberID, "payminto", amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment recorded"})
}
