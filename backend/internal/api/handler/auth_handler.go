package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// AuthHandler handles authentication lifecycle endpoints.
type AuthHandler struct {
	authSvc     *service.AuthService
	jwtTokenSvc *service.JWTTokenService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authSvc *service.AuthService, jwtTokenSvc *service.JWTTokenService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, jwtTokenSvc: jwtTokenSvc}
}

// signupRequest is the request body for POST /auth/signup.
type signupRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"     binding:"required"`
}

// signinRequest is the request body for POST /auth/signin.
type signinRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// refreshRequest is the request body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// signoutRequest is the request body for POST /auth/signout.
type signoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// forgotPasswordRequest is the request body for POST /auth/forgot-password.
type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// resetPasswordRequest is the request body for POST /auth/reset-password.
type resetPasswordRequest struct {
	Token       string `json:"token"       binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// Signup handles POST /api/v1/auth/signup.
func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, pair, err := h.authSvc.Signup(req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"member": member,
		"tokens": pair,
	})
}

// Signin handles POST /api/v1/auth/signin.
func (h *AuthHandler) Signin(c *gin.Context) {
	var req signinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, pair, err := h.authSvc.Signin(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"member": member,
		"tokens": pair,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.jwtTokenSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token service unavailable"})
		return
	}

	pair, err := h.jwtTokenSvc.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": pair})
}

// SignOut handles POST /api/v1/auth/signout.
func (h *AuthHandler) SignOut(c *gin.Context) {
	var req signoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authSvc.SignOut(req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "signed out"})
}

// SignOutAll handles POST /api/v1/auth/signout-all. Requires JWT auth.
func (h *AuthHandler) SignOutAll(c *gin.Context) {
	memberIDVal, exists := c.Get("memberID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	memberID, ok := memberIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid member context"})
		return
	}

	if err := h.authSvc.SignOutAll(memberID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always return success to avoid email enumeration.
	_, _ = h.authSvc.ForgotPassword(req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "if the account exists, a reset link has been sent"})
}

// ResetPassword handles POST /api/v1/auth/reset-password.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authSvc.ResetPassword(req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}
