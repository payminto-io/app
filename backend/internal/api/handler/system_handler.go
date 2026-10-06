package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// SystemHandler handles worker control and health endpoints.
type SystemHandler struct {
	systemSvc *service.SystemService
}

// NewSystemHandler constructs a SystemHandler.
func NewSystemHandler(systemSvc *service.SystemService) *SystemHandler {
	return &SystemHandler{systemSvc: systemSvc}
}

// ListWorkers handles GET /api/v1/admin/system/workers.
func (h *SystemHandler) ListWorkers(c *gin.Context) {
	statuses := h.systemSvc.ListWorkers()
	c.JSON(http.StatusOK, gin.H{"workers": statuses})
}

// StopAllWorkers handles POST /api/v1/admin/system/workers/stop-all.
// It cancels the shared worker context, gracefully stopping every worker.
func (h *SystemHandler) StopAllWorkers(c *gin.Context) {
	if err := h.systemSvc.StopAllWorkers(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "all workers stopped"})
}

// GetHealth handles GET /api/v1/admin/system/health.
func (h *SystemHandler) GetHealth(c *gin.Context) {
	health := h.systemSvc.GetSystemHealth(c.Request.Context())
	status := http.StatusOK
	if health.Status != "ok" {
		status = http.StatusMultiStatus
	}
	c.JSON(status, health)
}
