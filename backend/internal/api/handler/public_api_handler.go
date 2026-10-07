package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
)

// PublicAPIHandler serves read-only endpoints for the embeddable widget. No
// auth — responses redact sensitive fields.
type PublicAPIHandler struct {
	paymentSvc        *service.PaymentService
	depositAddressSvc *service.DepositAddressService
	platformSvc       *service.ExternalPlatformService
	tickerSvc         *service.TickerService
	blockchainCurRepo repository.BlockchainCurrencyRepository
}

// NewPublicAPIHandler wires the handler.
func NewPublicAPIHandler(
	paymentSvc *service.PaymentService,
	depositAddressSvc *service.DepositAddressService,
	platformSvc *service.ExternalPlatformService,
	tickerSvc *service.TickerService,
	blockchainCurRepo repository.BlockchainCurrencyRepository,
) *PublicAPIHandler {
	return &PublicAPIHandler{
		paymentSvc:        paymentSvc,
		depositAddressSvc: depositAddressSvc,
		platformSvc:       platformSvc,
		tickerSvc:         tickerSvc,
		blockchainCurRepo: blockchainCurRepo,
	}
}

// GetPaymentByReference handles GET /api/v1/public/payment/:reference_id.
// Returns a narrowed projection suitable for the checkout widget including
// deposit address, blockchain/currency codes, and merchant name.
func (h *PublicAPIHandler) GetPaymentByReference(c *gin.Context) {
	referenceID := c.Param("reference_id")
	if referenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reference_id is required"})
		return
	}
	pr, err := h.paymentSvc.GetByReferenceIDPublic(referenceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	resp := gin.H{
		"referenceID": pr.ReferenceID,
		"amountInUSD": pr.AmountInUSD,
		"state":       pr.State,
		"expiresAt":   pr.ExpiresAt,
	}

	// Enrich with deposit address data if assigned.
	if pr.ID > 0 && h.depositAddressSvc != nil {
		addresses, _ := h.depositAddressSvc.ListForPayment(pr.ID)
		if len(addresses) > 0 {
			da := addresses[0]
			resp["depositAddress"] = da.Address
			if da.BlockchainCurrency != nil {
				resp["blockchainCode"] = da.BlockchainCurrency.BlockchainCode
				resp["currencyCode"] = da.BlockchainCurrency.CurrencyCode
			}
			if owner, ok := h.depositAddressSvc.SolanaOwnerAddress(da.ID); ok {
				resp["depositOwnerAddress"] = owner
			}
		}
	}

	// Enrich with merchant name from ExternalPlatform.
	if h.platformSvc != nil {
		platform, err := h.platformSvc.GetByID(pr.ExternalPlatformID)
		if err == nil {
			resp["merchantName"] = platform.Name
		}
	}

	c.JSON(http.StatusOK, resp)
}

// GetTicker handles GET /api/v1/public/ticker?symbols=BTC,ETH and returns a
// cached price map for the requested symbols.
func (h *PublicAPIHandler) GetTicker(c *gin.Context) {
	if h.tickerSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ticker unavailable"})
		return
	}
	symbolsParam := c.Query("symbols")
	if symbolsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbols query param required"})
		return
	}
	symbols := splitAndTrim(symbolsParam)
	prices, err := h.tickerSvc.GetPrices(c.Request.Context(), symbols)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

// AssignDepositAddress handles POST /api/v1/public/deposit-address/reference/:reference_id.
// It assigns a deposit address for the given payment+blockchain so the checkout
// page can display a QR code. No auth required — the reference_id acts as the
// capability token (same as PayRam's public checkout flow).
func (h *PublicAPIHandler) AssignDepositAddress(c *gin.Context) {
	referenceID := c.Param("reference_id")
	if referenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reference_id is required"})
		return
	}

	var body struct {
		BlockchainCode string `json:"blockchainCode"`
		CurrencyCode   string `json:"currencyCode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if body.BlockchainCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "blockchainCode is required"})
		return
	}

	pr, err := h.paymentSvc.GetByReferenceIDPublic(referenceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if !strings.EqualFold(pr.State, models.PaymentStateOpen) || (pr.ExpiresAt != nil && !pr.ExpiresAt.After(time.Now())) {
		c.JSON(http.StatusConflict, gin.H{"error": "payment is not open"})
		return
	}

	// If the payment already has a deposit address, return it (idempotent).
	if h.depositAddressSvc != nil && pr.ID > 0 {
		existing, _ := h.depositAddressSvc.ListForPayment(pr.ID)
		if len(existing) > 0 {
			da := existing[0]
			if da.BlockchainCurrency == nil || !strings.EqualFold(da.BlockchainCurrency.BlockchainCode, body.BlockchainCode) || !strings.EqualFold(da.BlockchainCurrency.CurrencyCode, currencyCodeOrDefault(body.CurrencyCode, body.BlockchainCode)) {
				c.JSON(http.StatusConflict, gin.H{"error": "payment method is already locked"})
				return
			}
			resp := gin.H{
				"address": da.Address,
			}
			if da.BlockchainCurrency != nil {
				resp["blockchainCode"] = da.BlockchainCurrency.BlockchainCode
				resp["currencyCode"] = da.BlockchainCurrency.CurrencyCode
			}
			if owner, ok := h.depositAddressSvc.SolanaOwnerAddress(da.ID); ok {
				resp["ownerAddress"] = owner
			}
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	if h.depositAddressSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "deposit address service unavailable"})
		return
	}

	currencyCode := body.CurrencyCode
	if currencyCode == "" {
		currencyCode = body.BlockchainCode // default to native currency
	}

	da, err := h.depositAddressSvc.AssignForPayment(pr, body.BlockchainCode, currencyCode)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment method is temporarily unavailable"})
		return
	}

	resp := gin.H{
		"address": da.Address,
	}
	if da.BlockchainCurrency != nil {
		resp["blockchainCode"] = da.BlockchainCurrency.BlockchainCode
		resp["currencyCode"] = da.BlockchainCurrency.CurrencyCode
	}
	if owner, ok := h.depositAddressSvc.SolanaOwnerAddress(da.ID); ok {
		resp["ownerAddress"] = owner
	}
	c.JSON(http.StatusOK, resp)
}

func currencyCodeOrDefault(currencyCode, blockchainCode string) string {
	if currencyCode == "" {
		return blockchainCode
	}
	return currencyCode
}

// ListBlockchainCurrencies handles GET /api/v1/public/blockchain-currencies.
// Returns all deposit-enabled blockchain currencies so the checkout page can
// present a chain selector to the customer.
func (h *PublicAPIHandler) ListBlockchainCurrencies(c *gin.Context) {
	if h.blockchainCurRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "blockchain currency data unavailable"})
		return
	}
	currencies, err := h.blockchainCurRepo.ListDepositEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currencies": currencies})
}

func splitAndTrim(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			tok := s[start:i]
			// trim spaces
			for len(tok) > 0 && tok[0] == ' ' {
				tok = tok[1:]
			}
			for len(tok) > 0 && tok[len(tok)-1] == ' ' {
				tok = tok[:len(tok)-1]
			}
			if tok != "" {
				out = append(out, tok)
			}
			start = i + 1
		}
	}
	return out
}
