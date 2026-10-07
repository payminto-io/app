package handler

import (
	"bytes"
	"encoding/json"
	"github.com/payminto/payminto/backend/internal/environment"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuthHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.APIKey{},
		&models.AuthRefreshToken{},
		&models.EEEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newAuthHandler(t *testing.T, db *gorm.DB) (*AuthHandler, *service.AuthService, *service.JWTTokenService) {
	t.Helper()
	memberRepo := repository.NewMemberRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	refreshRepo := repository.NewAuthRefreshTokenRepository(db)
	eeRepo := repository.NewEEEventRepository(db)

	jwtSvc := service.NewJWTTokenService(refreshRepo, memberRepo, "test-secret", "test-secret",
		15*time.Minute, 30*24*time.Hour)
	jwtSvc.SetEnvironment(environment.Test)
	emitter := service.NewEventEmitterService(eeRepo)

	authSvc := service.NewAuthService(memberRepo, apiKeyRepo, "test-secret")
	authSvc.SetEnvironment(environment.Test)
	authSvc.SetJWTTokenService(jwtSvc)
	authSvc.SetEventEmitter(emitter)

	h := NewAuthHandler(authSvc, jwtSvc)
	return h, authSvc, jwtSvc
}

func TestAuthHandler_Signup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthHandlerDB(t)
	h, _, _ := newAuthHandler(t, db)

	r := gin.New()
	r.POST("/auth/signup", h.Signup)

	body, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
		"name":     "Alice",
	})
	req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("signup: got %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Signin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthHandlerDB(t)
	h, authSvc, _ := newAuthHandler(t, db)

	// Pre-create member via Signup.
	if _, _, err := authSvc.Signup("bob@example.com", "password123", "Bob"); err != nil {
		t.Fatalf("Signup: %v", err)
	}

	r := gin.New()
	r.POST("/auth/signin", h.Signin)

	body, _ := json.Marshal(map[string]string{
		"email":    "bob@example.com",
		"password": "password123",
	})
	req, _ := http.NewRequest("POST", "/auth/signin", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("signin: got %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["tokens"]; !ok {
		t.Error("expected 'tokens' in response")
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthHandlerDB(t)
	h, authSvc, _ := newAuthHandler(t, db)

	_, pair, err := authSvc.Signup("carol@example.com", "password123", "Carol")
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	body, _ := json.Marshal(map[string]string{
		"refreshToken": pair.RefreshToken,
	})
	req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("refresh: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Signup_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthHandlerDB(t)
	h, _, _ := newAuthHandler(t, db)

	r := gin.New()
	r.POST("/auth/signup", h.Signup)

	req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestAuthHandler_ForgotPassword_NeverEnumerates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthHandlerDB(t)
	h, _, _ := newAuthHandler(t, db)

	r := gin.New()
	r.POST("/auth/forgot-password", h.ForgotPassword)

	// Non-existent email — should still return 200.
	body, _ := json.Marshal(map[string]string{"email": "ghost@example.com"})
	req, _ := http.NewRequest("POST", "/auth/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (anti-enumeration), got %d", w.Code)
	}
}

func TestAuthHandler_SignOut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthHandlerDB(t)
	h, authSvc, _ := newAuthHandler(t, db)

	_, pair, err := authSvc.Signup("dave@example.com", "password123", "Dave")
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	r := gin.New()
	r.POST("/auth/signout", h.SignOut)

	body, _ := json.Marshal(map[string]string{"refreshToken": pair.RefreshToken})
	req, _ := http.NewRequest("POST", "/auth/signout", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("signout: got %d, want 200", w.Code)
	}
}
