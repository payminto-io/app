package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/api/dto"
	"github.com/payminto/payminto/backend/internal/service"
)

// PaymentHandler handles the payment lifecycle HTTP endpoints.
type PaymentHandler struct {
	paymentSvc *service.PaymentService
	host       string
}

// NewPaymentHandler creates a PaymentHandler. host is the base URL of the
// hosted checkout (e.g. "https://pay.example.com") used in payment URLs.
func NewPaymentHandler(paymentSvc *service.PaymentService, host string) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc, host: host}
}

// CreatePayment handles POST /api/v1/payment.
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req dto.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memberID, _ := c.Get("memberID")
	platformID, _ := c.Get("externalPlatformID")

	input := service.CreatePaymentInput{
		AmountInUSD:    req.AmountInUSD,
		CustomerEmail:  req.CustomerEmail,
		CustomerID:     req.CustomerID,
		InvoiceID:      req.InvoiceID,
		BlockchainCode: req.BlockchainCode,
		CurrencyCode:   req.CurrencyCode,
	}

	result, err := h.paymentSvc.CreatePayment(input, memberID.(uint), platformID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := dto.CreatePaymentResponse{
		Host:        h.host,
		ReferenceID: result.Payment.ReferenceID,
		URL:         h.host + "/pay/" + result.Payment.ReferenceID,
	}
	if result.DepositAddress != nil {
		addr := result.DepositAddress.Address
		resp.DepositAddress = &addr
		chainCode := req.BlockchainCode
		resp.BlockchainCode = &chainCode
	}

	c.JSON(http.StatusCreated, resp)
}

// GetPayment handles GET /api/v1/payment/reference/:reference_id.
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	refID := c.Param("reference_id")
	platformID, _ := c.Get("externalPlatformID")

	payment, err := h.paymentSvc.GetPayment(refID, platformID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	c.JSON(http.StatusOK, dto.ToPaymentResponse(payment))
}

// ListPayments handles GET /api/v1/payments. Accepts an optional ?state= query param.
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	platformID, _ := c.Get("externalPlatformID")

	filter := service.ListPaymentsFilter{
		State: c.Query("state"),
	}

	payments, total, err := h.paymentSvc.ListPayments(platformID.(uint), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []dto.PaymentResponse
	for i := range payments {
		resp = append(resp, dto.ToPaymentResponse(&payments[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": resp,
		"total":    total,
	})
}
