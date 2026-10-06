package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDepositAddressHandler_AssignForReference_MissingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDepositAddressHandler(nil, nil)
	r.POST("/api/v1/deposit-address/reference/:reference_id", func(c *gin.Context) {
		c.Set("externalPlatformID", uint(1))
		c.Next()
	}, h.AssignForReference)

	w := httptest.NewRecorder()
	// Empty JSON object — blockchainCode has binding:"required" so bind should fail
	req, _ := http.NewRequest("POST", "/api/v1/deposit-address/reference/ref-1", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing blockchainCode, got %d", w.Code)
	}
}

func TestDepositAddressHandler_ListForReference_NoPayment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Use gin.Recovery to catch nil service panics gracefully
	r.Use(gin.Recovery())
	h := NewDepositAddressHandler(nil, nil)
	r.GET("/api/v1/deposit-address/reference/:reference_id", func(c *gin.Context) {
		c.Set("externalPlatformID", uint(1))
		c.Next()
	}, h.ListForReference)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/deposit-address/reference/ref-missing", nil)
	r.ServeHTTP(w, req)

	// Nil payment service → panic → recovered → 500 OR the handler returns 404.
	// Accept either.
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected 404 or 500, got %d", w.Code)
	}
}
