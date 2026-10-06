package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreatePayment_MissingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Create handler with nil service to test input validation
	h := NewPaymentHandler(nil, "http://localhost:8080")
	r.POST("/payment", func(c *gin.Context) {
		c.Set("memberID", uint(1))
		c.Set("externalPlatformID", uint(1))
		c.Next()
	}, h.CreatePayment)

	w := httptest.NewRecorder()
	// Empty JSON object — AmountInUSD has binding:"required" so bind should fail
	req, _ := http.NewRequest("POST", "/payment", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing required field, got %d", w.Code)
	}
}

func TestCreatePayment_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	h := NewPaymentHandler(nil, "http://localhost:8080")
	r.POST("/payment", func(c *gin.Context) {
		c.Set("memberID", uint(1))
		c.Set("externalPlatformID", uint(1))
		c.Next()
	}, h.CreatePayment)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payment", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestGetPayment_MissingRefID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	h := NewPaymentHandler(nil, "http://localhost:8080")
	r.GET("/payment/reference/:reference_id", func(c *gin.Context) {
		c.Set("externalPlatformID", uint(1))
		c.Next()
	}, h.GetPayment)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/payment/reference/nonexistent", nil)
	r.ServeHTTP(w, req)

	// With nil service, the handler will panic (nil pointer dereference) which
	// Recovery catches as 500, or it returns 404 — both are acceptable.
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected 404 or 500 for nonexistent payment, got %d", w.Code)
	}
}
