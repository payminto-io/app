package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// MemberHandler handles member management endpoints.
type MemberHandler struct {
	memberSvc *service.MemberService
}

// NewMemberHandler constructs a MemberHandler.
func NewMemberHandler(memberSvc *service.MemberService) *MemberHandler {
	return &MemberHandler{memberSvc: memberSvc}
}

// GetMember handles GET /api/v1/admin/members/:id.
func (h *MemberHandler) GetMember(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member id"})
		return
	}
	m, err := h.memberSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	c.JSON(http.StatusOK, m)
}

// ListMembers handles GET /api/v1/admin/members — returns all members
// assigned to the caller's external platform. The JWT/API-key middleware
// populates externalPlatformID in the Gin context; we scope the query to
// that platform so tenants only see their own members.
func (h *MemberHandler) ListMembers(c *gin.Context) {
	platformIDVal, ok := c.Get("externalPlatformID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing platform context"})
		return
	}
	platformID, ok := platformIDVal.(uint)
	if !ok || platformID == 0 {
		c.JSON(http.StatusOK, []any{})
		return
	}

	members, err := h.memberSvc.ListByPlatform(c.Request.Context(), platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return a flat array so the frontend hook can consume it directly.
	c.JSON(http.StatusOK, members)
}

// inviteMemberRequest is the payload for POST /api/v1/admin/members.
type inviteMemberRequest struct {
	Name       string `json:"name" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	MemberType string `json:"memberType"`
}

// InviteMember handles POST /api/v1/admin/members (requires members.invite).
func (h *MemberHandler) InviteMember(c *gin.Context) {
	var req inviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.memberSvc.CreateMember(c.Request.Context(), service.CreateMemberInput{
		Name:       req.Name,
		Email:      req.Email,
		Password:   req.Password,
		MemberType: req.MemberType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

// GetMe handles GET /api/v1/members/me — returns the authenticated caller's
// own member record. Accessible to any caller on the merchant route group
// (JWTOrAPIKey). Returns 401 when the context is missing a member ID, 404
// when the stored member has been deleted.
func (h *MemberHandler) GetMe(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	m, err := h.memberSvc.GetByID(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	c.JSON(http.StatusOK, m)
}

// changePasswordRequest is the payload for PUT /api/v1/members/me/password.
type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
}

// ChangeOwnPassword handles PUT /api/v1/members/me/password.
func (h *MemberHandler) ChangeOwnPassword(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.memberSvc.ChangePassword(c.Request.Context(), memberID, req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// ListCustomers handles GET /api/v1/customers — returns paginated customer members.
func (h *MemberHandler) ListCustomers(c *gin.Context) {
	limit := 25
	offset := 0
	search := c.Query("search")
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	customers, total, err := h.memberSvc.ListCustomers(c.Request.Context(), service.ListCustomersFilter{
		Search: search,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"customers": customers, "total": total})
}

// RemoveMember handles DELETE /api/v1/admin/members/:id (requires members.remove).
func (h *MemberHandler) RemoveMember(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member id"})
		return
	}
	if err := h.memberSvc.Deactivate(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member deactivated"})
}
