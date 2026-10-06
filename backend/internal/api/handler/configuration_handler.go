package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// ConfigurationHandler handles runtime configuration endpoints.
type ConfigurationHandler struct {
	configSvc *service.ConfigurationService
}

// NewConfigurationHandler constructs a ConfigurationHandler.
func NewConfigurationHandler(configSvc *service.ConfigurationService) *ConfigurationHandler {
	return &ConfigurationHandler{configSvc: configSvc}
}

// ListConfigurations handles GET /api/v1/admin/configurations.
func (h *ConfigurationHandler) ListConfigurations(c *gin.Context) {
	cfgs, err := h.configSvc.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"configurations": cfgs})
}

// GetConfiguration handles GET /api/v1/admin/configurations/:key.
func (h *ConfigurationHandler) GetConfiguration(c *gin.Context) {
	key := c.Param("key")
	cfg, err := h.configSvc.Get(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "configuration key not found"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// setConfigurationRequest is the payload for PUT /api/v1/admin/configurations/:key.
type setConfigurationRequest struct {
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
}

// SetConfiguration handles PUT /api/v1/admin/configurations/:key.
func (h *ConfigurationHandler) SetConfiguration(c *gin.Context) {
	key := c.Param("key")
	var req setConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.configSvc.Set(c.Request.Context(), key, req.Value, req.Description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "configuration updated"})
}
