package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// RoleHandler handles RBAC role management endpoints.
type RoleHandler struct {
	roleSvc *service.RoleService
}

// NewRoleHandler constructs a RoleHandler.
func NewRoleHandler(roleSvc *service.RoleService) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc}
}

// ListRoles handles GET /api/v1/admin/roles.
func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roleSvc.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// GetRole handles GET /api/v1/admin/roles/:id.
func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	role, err := h.roleSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	c.JSON(http.StatusOK, role)
}

// createRoleRequest is the payload for POST /api/v1/admin/roles.
type createRoleRequest struct {
	Name          string `json:"name" binding:"required"`
	DisplayName   string `json:"displayName" binding:"required"`
	Description   string `json:"description"`
	PermissionIDs []uint `json:"permissionIDs"`
}

// CreateRole handles POST /api/v1/admin/roles.
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role, err := h.roleSvc.CreateRole(c.Request.Context(), service.CreateRoleInput{
		Name:          req.Name,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, role)
}

// updateRolePermissionsRequest is the payload for PUT /api/v1/admin/roles/:id/permissions.
type updateRolePermissionsRequest struct {
	PermissionIDs []uint `json:"permissionIDs" binding:"required"`
}

// UpdateRolePermissions handles PUT /api/v1/admin/roles/:id/permissions.
func (h *RoleHandler) UpdateRolePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	var req updateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.roleSvc.UpdatePermissions(c.Request.Context(), uint(id), req.PermissionIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "permissions updated"})
}
