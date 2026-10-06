package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// DepositAddressHandler handles deposit address assignment and listing for
// payment references. All routes require API key authentication (applied in
// router.go).
type DepositAddressHandler struct {
	paymentSvc        *service.PaymentService
	depositAddressSvc *service.DepositAddressService
}

// NewDepositAddressHandler constructs the handler with the required services.
func NewDepositAddressHandler(
	paymentSvc *service.PaymentService,
	depositAddressSvc *service.DepositAddressService,
) *DepositAddressHandler {
	return &DepositAddressHandler{
		paymentSvc:        paymentSvc,
		depositAddressSvc: depositAddressSvc,
	}
}

type assignDepositAddressRequest struct {
	BlockchainCode string `json:"blockchainCode" binding:"required"`
	CurrencyCode   string `json:"currencyCode"`
}

type depositAddressResponse struct {
	Address              string `json:"address"`
	BlockchainCode       string `json:"blockchainCode"`
	CurrencyCode         string `json:"currencyCode"`
	BlockchainCurrencyID uint   `json:"blockchainCurrencyID"`
	PaymentRequestID     *uint  `json:"paymentRequestID,omitempty"`
}

// AssignForReference handles POST /api/v1/deposit-address/reference/:reference_id.
// Body: { blockchainCode, currencyCode? }. Looks up the payment, assigns a
// fresh deposit address from the pool, and returns the address + chain details.
func (h *DepositAddressHandler) AssignForReference(c *gin.Context) {
	refID := c.Param("reference_id")
	platformID, _ := c.Get("externalPlatformID")

	var req assignDepositAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.paymentSvc.GetPayment(refID, platformID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	currencyCode := req.CurrencyCode
	if currencyCode == "" {
		currencyCode = req.BlockchainCode
	}

	da, err := h.depositAddressSvc.AssignForPayment(payment, req.BlockchainCode, currencyCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := depositAddressResponse{
		Address:              da.Address,
		BlockchainCode:       req.BlockchainCode,
		CurrencyCode:         currencyCode,
		BlockchainCurrencyID: da.BlockchainCurrencyID,
		PaymentRequestID:     da.PaymentRequestID,
	}
	c.JSON(http.StatusCreated, resp)
}

// ListForReference handles GET /api/v1/deposit-address/reference/:reference_id.
// Returns every deposit address currently assigned to the payment.
func (h *DepositAddressHandler) ListForReference(c *gin.Context) {
	refID := c.Param("reference_id")
	platformID, _ := c.Get("externalPlatformID")

	payment, err := h.paymentSvc.GetPayment(refID, platformID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	list, err := h.depositAddressSvc.ListForPayment(payment.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]depositAddressResponse, 0, len(list))
	for _, da := range list {
		chainCode := ""
		currencyCode := ""
		if da.BlockchainCurrency != nil {
			chainCode = da.BlockchainCurrency.BlockchainCode
			currencyCode = da.BlockchainCurrency.CurrencyCode
		}
		out = append(out, depositAddressResponse{
			Address:              da.Address,
			BlockchainCode:       chainCode,
			CurrencyCode:         currencyCode,
			BlockchainCurrencyID: da.BlockchainCurrencyID,
			PaymentRequestID:     da.PaymentRequestID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"depositAddresses": out})
}
