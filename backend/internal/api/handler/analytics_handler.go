package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// AnalyticsHandler exposes read-only aggregation endpoints for the merchant dashboard.
type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

// NewAnalyticsHandler constructs an AnalyticsHandler.
func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// parseTimeRange extracts start/end query params. Defaults: last 30 days.
func parseTimeRange(c *gin.Context) (start, end time.Time) {
	end = time.Now().UTC()
	start = end.AddDate(0, 0, -30)

	if s := c.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if e := c.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	}
	return start, end
}

// GetVolume handles GET /api/v1/analytics/volume?start=&end=&interval=day.
func (h *AnalyticsHandler) GetVolume(c *gin.Context) {
	platformID := mustPlatformID(c)
	start, end := parseTimeRange(c)
	interval := c.DefaultQuery("interval", "day")

	buckets, err := h.svc.GetVolumeOverTime(c.Request.Context(), platformID, start, end, interval)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"volume": buckets})
}

// GetTopCustomers handles GET /api/v1/analytics/customers/top?limit=10.
func (h *AnalyticsHandler) GetTopCustomers(c *gin.Context) {
	platformID := mustPlatformID(c)

	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	customers, err := h.svc.GetTopCustomers(c.Request.Context(), platformID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customers": customers})
}

// GetRevenueBreakdown handles GET /api/v1/analytics/revenue?start=&end=.
func (h *AnalyticsHandler) GetRevenueBreakdown(c *gin.Context) {
	platformID := mustPlatformID(c)
	start, end := parseTimeRange(c)

	breakdown, err := h.svc.GetRevenueBreakdown(c.Request.Context(), platformID, start, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"revenue": breakdown})
}

// GetSweepStats handles GET /api/v1/analytics/sweeps?start=&end=.
func (h *AnalyticsHandler) GetSweepStats(c *gin.Context) {
	platformID := mustPlatformID(c)
	start, end := parseTimeRange(c)

	stats, err := h.svc.GetSweepStats(c.Request.Context(), platformID, start, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sweeps": stats})
}

// GetWithdrawalStats handles GET /api/v1/analytics/withdrawals?start=&end=.
func (h *AnalyticsHandler) GetWithdrawalStats(c *gin.Context) {
	platformID := mustPlatformID(c)
	start, end := parseTimeRange(c)

	stats, err := h.svc.GetWithdrawalStats(c.Request.Context(), platformID, start, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawals": stats})
}

// GetSummary handles GET /api/v1/analytics/summary.
func (h *AnalyticsHandler) GetSummary(c *gin.Context) {
	platformID := mustPlatformID(c)

	summary, err := h.svc.GetDashboardSummary(c.Request.Context(), platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}
