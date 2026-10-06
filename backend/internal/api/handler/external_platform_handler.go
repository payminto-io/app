package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

func float64PtrToDecimalPtr(v *float64) *decimal.Decimal {
	if v == nil {
		return nil
	}
	d := decimal.NewFromFloat(*v)
	return &d
}

// ExternalPlatformHandler serves the admin CRUD surface for merchant projects
// (external platforms) plus their per-currency enable/limit config.
type ExternalPlatformHandler struct {
	platformSvc *service.ExternalPlatformService
	epbcSvc     *service.ExternalPlatformBlockchainCurrencyService
}

// NewExternalPlatformHandler wires the handler.
func NewExternalPlatformHandler(
	platformSvc *service.ExternalPlatformService,
	epbcSvc *service.ExternalPlatformBlockchainCurrencyService,
) *ExternalPlatformHandler {
	return &ExternalPlatformHandler{platformSvc: platformSvc, epbcSvc: epbcSvc}
}

type externalPlatformRequest struct {
	Name             string  `json:"name" binding:"required"`
	Website          string  `json:"website"`
	LogoPath         string  `json:"logoPath"`
	BrandColor       string  `json:"brandColor"`
	SuccessEndpoint  string  `json:"successEndpoint"`
	FrontendEndpoint *string `json:"frontendEndpoint"`
	CancelEndpoint   *string `json:"cancelEndpoint"`
	SupportEmail     *string `json:"supportEmail"`
}

func (r externalPlatformRequest) toInput() service.ExternalPlatformInput {
	return service.ExternalPlatformInput{
		Name:             r.Name,
		Website:          r.Website,
		LogoPath:         r.LogoPath,
		BrandColor:       r.BrandColor,
		SuccessEndpoint:  r.SuccessEndpoint,
		FrontendEndpoint: r.FrontendEndpoint,
		CancelEndpoint:   r.CancelEndpoint,
		SupportEmail:     r.SupportEmail,
	}
}

// Create handles POST /admin/external-platforms — returns the plaintext
// API key exactly once.
func (h *ExternalPlatformHandler) Create(c *gin.Context) {
	var req externalPlatformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	platform, plainKey, err := h.platformSvc.Create(req.toInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"platform": platform, "apiKey": plainKey})
}

// List handles GET /admin/external-platforms.
func (h *ExternalPlatformHandler) List(c *gin.Context) {
	rows, err := h.platformSvc.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"platforms": rows})
}

// Get handles GET /admin/external-platforms/:id.
func (h *ExternalPlatformHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	platform, err := h.platformSvc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, platform)
}

// Update handles PUT /admin/external-platforms/:id.
func (h *ExternalPlatformHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req externalPlatformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	platform, err := h.platformSvc.Update(id, req.toInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, platform)
}

// Delete handles DELETE /admin/external-platforms/:id.
func (h *ExternalPlatformHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.platformSvc.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// RegenerateKey handles POST /admin/external-platforms/:id/regenerate-key.
// The plaintext key is returned exactly once.
func (h *ExternalPlatformHandler) RegenerateKey(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	key, err := h.platformSvc.RegenerateAPIKey(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"apiKey": key})
}

// ListCurrencies handles GET /admin/external-platforms/:id/currencies.
func (h *ExternalPlatformHandler) ListCurrencies(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.epbcSvc.ListByPlatform(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currencies": rows})
}

type epbcRequest struct {
	BlockchainCurrencyID uint     `json:"blockchainCurrencyID" binding:"required"`
	AutoApproveThreshold *float64 `json:"autoApproveThreshold"`
	HourlyCap            *float64 `json:"hourlyCap"`
	DailyCap             *float64 `json:"dailyCap"`
	MinAmount            *float64 `json:"minAmount"`
	MaxAmount            *float64 `json:"maxAmount"`
}

func (r epbcRequest) toConfig() service.EPBCConfig {
	return service.EPBCConfig{
		AutoApproveThreshold: float64PtrToDecimalPtr(r.AutoApproveThreshold),
		HourlyCap:            float64PtrToDecimalPtr(r.HourlyCap),
		DailyCap:             float64PtrToDecimalPtr(r.DailyCap),
		MinAmount:            float64PtrToDecimalPtr(r.MinAmount),
		MaxAmount:            float64PtrToDecimalPtr(r.MaxAmount),
	}
}

// EnableCurrency handles POST /admin/external-platforms/:id/currencies.
func (h *ExternalPlatformHandler) EnableCurrency(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req epbcRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.epbcSvc.Enable(id, req.BlockchainCurrencyID, req.toConfig())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}

// UpdateCurrencyLimits handles PUT /admin/external-platforms/:id/currencies/:currencyID.
func (h *ExternalPlatformHandler) UpdateCurrencyLimits(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	currencyID, err := parseUintParam(c, "currencyID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req epbcRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.BlockchainCurrencyID = currencyID
	row, err := h.epbcSvc.UpdateLimits(id, currencyID, req.toConfig())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}

// DisableCurrency handles DELETE /admin/external-platforms/:id/currencies/:currencyID.
func (h *ExternalPlatformHandler) DisableCurrency(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	currencyID, err := parseUintParam(c, "currencyID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.epbcSvc.Disable(id, currencyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
