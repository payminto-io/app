package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/modules"
)

// RegisterEnvironmentRoutes exposes the process environment for the dashboard's test/live indicator.
func RegisterEnvironmentRoutes(rg *gin.RouterGroup, m *modules.EnvironmentModule) {
	rg.GET("/environment", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"environment": m.Environment})
	})
}
