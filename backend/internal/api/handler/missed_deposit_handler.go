package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// MissedDepositHandler exposes the operator reconciliation endpoints.
type MissedDepositHandler struct {
	svc *service.MissedDepositService
}

// NewMissedDepositHandler wires the handler.
func NewMissedDepositHandler(svc *service.MissedDepositService) *MissedDepositHandler {
	return &MissedDepositHandler{svc: svc}
}

// ListPending handles GET /admin/missed-deposits.
func (h *MissedDepositHandler) ListPending(c *gin.Context) {
	rows, err := h.svc.ListPending()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"missedDeposits": rows})
}

// Get handles GET /admin/missed-deposits/:id.
func (h *MissedDepositHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}

type resolveRequest struct {
	Action string `json:"action" binding:"required"`
	Reason string `json:"reason"`
}

// Resolve handles POST /admin/missed-deposits/:id/resolve.
func (h *MissedDepositHandler) Resolve(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	memberID, _ := c.Get("memberID")
	mid, _ := memberID.(uint)

	if err := h.svc.Resolve(id, req.Action, req.Reason, mid); err != nil {
		switch {
		case errors.Is(err, service.ErrMissedDepositAlreadyResolved):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrMissedDepositNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}
