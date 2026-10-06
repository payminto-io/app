package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// OTPHandler handles OTP generation and verification endpoints.
type OTPHandler struct {
	otpSvc *service.OTPService
}

// NewOTPHandler constructs an OTPHandler.
func NewOTPHandler(otpSvc *service.OTPService) *OTPHandler {
	return &OTPHandler{otpSvc: otpSvc}
}

// generateOTPRequest is the request body for POST /otp/generate.
type generateOTPRequest struct {
	Purpose string `json:"purpose" binding:"required"`
}

// verifyOTPRequest is the request body for POST /otp/verify.
type verifyOTPRequest struct {
	Purpose string `json:"purpose" binding:"required"`
	Code    string `json:"code"    binding:"required"`
}

// Generate handles POST /api/v1/otp/generate. Requires JWT auth.
// Returns HTTP 200 with a stub response — the OTP code is sent via email,
// not returned in the response body.
func (h *OTPHandler) Generate(c *gin.Context) {
	var req generateOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	if _, err := h.otpSvc.Generate(memberID, req.Purpose); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

// Verify handles POST /api/v1/otp/verify.
func (h *OTPHandler) Verify(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	if err := h.otpSvc.Verify(memberID, req.Purpose, req.Code); err != nil {
		switch {
		case errors.Is(err, service.ErrOTPExpired):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "OTP expired"})
		case errors.Is(err, service.ErrOTPMaxAttempts):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "max OTP attempts exceeded"})
		case errors.Is(err, service.ErrOTPInvalid):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid OTP"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified"})
}
