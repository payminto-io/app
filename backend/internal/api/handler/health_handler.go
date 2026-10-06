package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler exposes the /healthz liveness probe.
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a HealthHandler backed by db.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health handles GET /healthz. Returns 200 when the database is reachable and
// 503 otherwise.
func (h *HealthHandler) Health(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": "db unavailable"})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": "db ping failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
