package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// PermissionHandler handles permission listing endpoints.
type PermissionHandler struct {
	permSvc *service.PermissionService
}

// NewPermissionHandler constructs a PermissionHandler.
func NewPermissionHandler(permSvc *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permSvc: permSvc}
}

// ListPermissions handles GET /api/v1/admin/permissions.
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	perms, err := h.permSvc.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}
