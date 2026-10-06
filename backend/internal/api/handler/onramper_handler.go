package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

// OnramperHandler serves the Onramper session + webhook endpoints.
type OnramperHandler struct {
	svc *service.OnramperPaymentsService
}

// NewOnramperHandler wires the handler.
func NewOnramperHandler(svc *service.OnramperPaymentsService) *OnramperHandler {
	return &OnramperHandler{svc: svc}
}

type createOnramperSessionRequest struct {
	PaymentRequestID *uint  `json:"paymentRequestID"`
	FiatAmount       string `json:"fiatAmount" binding:"required"`
	FiatCurrency     string `json:"fiatCurrency" binding:"required"`
	CryptoCurrency   string `json:"cryptoCurrency" binding:"required"`
	BlockchainCode   string `json:"blockchainCode" binding:"required"`
	WalletAddress    string `json:"walletAddress" binding:"required"`
}

// CreateSession handles POST /onramper/session (API-key protected).
func (h *OnramperHandler) CreateSession(c *gin.Context) {
	var req createOnramperSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	platformIDVal, ok := c.Get("externalPlatformID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing platform context"})
		return
	}
	platformID, _ := platformIDVal.(uint)

	fiat, err := decimal.NewFromString(req.FiatAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fiatAmount"})
		return
	}

	row, redirectURL, err := h.svc.CreateSession(service.CreateOnramperSessionInput{
		PlatformID:       platformID,
		PaymentRequestID: req.PaymentRequestID,
		FiatAmount:       fiat,
		FiatCurrency:     req.FiatCurrency,
		CryptoCurrency:   req.CryptoCurrency,
		BlockchainCode:   req.BlockchainCode,
		WalletAddress:    req.WalletAddress,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session": row, "redirectURL": redirectURL})
}

// Webhook handles POST /onramper/webhook (no auth, HMAC-verified).
func (h *OnramperHandler) Webhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var parsed service.OnramperWebhookPayload
	if err := json.Unmarshal(body, &parsed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	signature := c.GetHeader("X-Onramper-Signature")
	if err := h.svc.HandleWebhook(body, signature, parsed); err != nil {
		switch {
		case errors.Is(err, service.ErrOnramperSignatureInvalid):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrOnramperAlreadyTerminal):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// List handles GET /onramper/payments for the calling merchant.
func (h *OnramperHandler) List(c *gin.Context) {
	platformIDVal, ok := c.Get("externalPlatformID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing platform context"})
		return
	}
	platformID, _ := platformIDVal.(uint)

	rows, err := h.svc.ListByPlatform(platformID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"payments": rows})
}
