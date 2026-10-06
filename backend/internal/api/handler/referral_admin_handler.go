package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

// ReferralAdminHandler exposes campaign management endpoints for system admins.
type ReferralAdminHandler struct {
	campaignSvc          *service.ReferralCampaignService
	analyticsReferralSvc *service.AnalyticsReferralService
}

// NewReferralAdminHandler constructs a ReferralAdminHandler.
func NewReferralAdminHandler(
	campaignSvc *service.ReferralCampaignService,
	analyticsReferralSvc *service.AnalyticsReferralService,
) *ReferralAdminHandler {
	return &ReferralAdminHandler{
		campaignSvc:          campaignSvc,
		analyticsReferralSvc: analyticsReferralSvc,
	}
}

// createCampaignRequest is the body for POST /api/v1/admin/referrals/campaigns.
type createCampaignRequest struct {
	Name         string  `json:"name" binding:"required"`
	Description  string  `json:"description,omitempty"`
	RewardType   string  `json:"rewardType,omitempty"`
	RewardValue  string  `json:"rewardValue,omitempty"`
	CurrencyCode string  `json:"currencyCode,omitempty"`
	Budget       *string `json:"budget,omitempty"`
}

// CreateCampaign handles POST /api/v1/admin/referrals/campaigns.
func (h *ReferralAdminHandler) CreateCampaign(c *gin.Context) {
	var req createCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rewardVal := decimal.Zero
	if req.RewardValue != "" {
		var err error
		rewardVal, err = decimal.NewFromString(req.RewardValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rewardValue"})
			return
		}
	}

	var budget *decimal.Decimal
	if req.Budget != nil && *req.Budget != "" {
		b, err := decimal.NewFromString(*req.Budget)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid budget"})
			return
		}
		budget = &b
	}

	campaign, err := h.campaignSvc.CreateCampaign(service.CreateCampaignInput{
		Project:      "payminto",
		Name:         req.Name,
		Description:  req.Description,
		RewardType:   req.RewardType,
		RewardValue:  rewardVal,
		CurrencyCode: req.CurrencyCode,
		Budget:       budget,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"campaign": campaign})
}

// ListCampaigns handles GET /api/v1/admin/referrals/campaigns.
func (h *ReferralAdminHandler) ListCampaigns(c *gin.Context) {
	campaigns, err := h.campaignSvc.ListCampaigns("payminto")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": campaigns})
}

// GetCampaign handles GET /api/v1/admin/referrals/campaigns/:id.
func (h *ReferralAdminHandler) GetCampaign(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign id"})
		return
	}
	campaign, err := h.campaignSvc.GetCampaign(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": campaign})
}

// ActivateCampaign handles POST /api/v1/admin/referrals/campaigns/:id/activate.
func (h *ReferralAdminHandler) ActivateCampaign(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign id"})
		return
	}
	if err := h.campaignSvc.ActivateCampaign(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "campaign activated"})
}

// DeactivateCampaign handles POST /api/v1/admin/referrals/campaigns/:id/deactivate.
func (h *ReferralAdminHandler) DeactivateCampaign(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign id"})
		return
	}
	if err := h.campaignSvc.DeactivateCampaign(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "campaign deactivated"})
}

// DeleteCampaign handles DELETE /api/v1/admin/referrals/campaigns/:id.
func (h *ReferralAdminHandler) DeleteCampaign(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign id"})
		return
	}
	if err := h.campaignSvc.DeleteCampaign(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "campaign deleted"})
}

// GetTopReferrers handles GET /api/v1/admin/referrals/top.
func (h *ReferralAdminHandler) GetTopReferrers(c *gin.Context) {
	rows, err := h.analyticsReferralSvc.GetTopReferrers(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"topReferrers": rows})
}

// GetCampaignPerformance handles GET /api/v1/admin/referrals/campaigns/:id/performance.
func (h *ReferralAdminHandler) GetCampaignPerformance(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign id"})
		return
	}
	perf, err := h.analyticsReferralSvc.GetCampaignPerformance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"performance": perf})
}
