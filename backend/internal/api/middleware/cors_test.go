package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_AllowsOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	acaoHeader := w.Header().Get("Access-Control-Allow-Origin")
	if acaoHeader == "" {
		t.Error("expected Access-Control-Allow-Origin header")
	}
}

func TestCORS_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)

	if w.Code != 204 && w.Code != 200 {
		t.Errorf("expected 204 or 200 for preflight, got %d", w.Code)
	}
}

// TestCORS_DefaultNeverWildcardWithCredentials guards the security invariant:
// the default (no configured origins) must NOT emit
// Access-Control-Allow-Credentials:true, because that combined with a wildcard
// origin is rejected by browsers and is a credential-leak footgun.
func TestCORS_DefaultNeverWildcardWithCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS()) // no allowed origins → public wildcard, no credentials
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got == "true" {
		t.Errorf("default CORS must not allow credentials, got %q", got)
	}
}

// TestCORS_ConfiguredOriginAllowsCredentials verifies that an explicit allow
// list permits exactly the listed origin WITH credentials, and rejects others.
func TestCORS_ConfiguredOriginAllowsCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("https://app.payminto.com"))
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Allowed origin → reflected with credentials.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.payminto.com")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.payminto.com" {
		t.Errorf("expected allowed origin reflected, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credentials allowed for configured origin, got %q", got)
	}

	// Disallowed origin → not reflected.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w2, req2)
	if got := w2.Header().Get("Access-Control-Allow-Origin"); got == "https://evil.example.com" {
		t.Error("disallowed origin must not be reflected")
	}
}
