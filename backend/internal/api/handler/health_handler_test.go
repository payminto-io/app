package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler_NilDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Use Recovery so a nil-pointer panic in the handler becomes a 500 rather
	// than crashing the test process.  The important assertion is that we do NOT
	// get a 200 when no real DB is wired up.
	r.Use(gin.Recovery())

	h := NewHealthHandler(nil)
	r.GET("/healthz", h.Health)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	// With nil DB the handler either returns 503 or panics (caught as 500).
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusInternalServerError {
		t.Errorf("expected 503 or 500 with nil DB, got %d", w.Code)
	}
}
