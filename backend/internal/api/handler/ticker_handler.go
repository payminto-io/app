package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// TickerHandler exposes price-feed endpoints. No auth required — price data is public.
type TickerHandler struct {
	svc *service.TickerService
}

// NewTickerHandler constructs a TickerHandler.
func NewTickerHandler(svc *service.TickerService) *TickerHandler {
	return &TickerHandler{svc: svc}
}

// GetPrice handles GET /api/v1/ticker/:symbol.
func (h *TickerHandler) GetPrice(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	price, err := h.svc.GetPrice(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"price":  price,
	})
}

// GetPrices handles GET /api/v1/ticker?symbols=BTC,ETH,USDT.
func (h *TickerHandler) GetPrices(c *gin.Context) {
	raw := c.Query("symbols")
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbols query parameter is required"})
		return
	}

	parts := strings.Split(raw, ",")
	symbols := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(strings.ToUpper(p)); s != "" {
			symbols = append(symbols, s)
		}
	}
	if len(symbols) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid symbols provided"})
		return
	}

	prices, err := h.svc.GetPrices(c.Request.Context(), symbols)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert decimal.Decimal map to string map for clean JSON.
	out := make(map[string]string, len(prices))
	for sym, price := range prices {
		out[sym] = price.String()
	}

	c.JSON(http.StatusOK, gin.H{"prices": out})
}
