package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newEnvAuth builds an AuthService for a process serving env, with one active key per environment.
func newEnvAuth(t *testing.T, env environment.Environment) (*service.AuthService, map[environment.Environment]string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Member{}, &models.Role{}, &models.ExternalPlatform{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	member := models.Member{Name: "m", MemberType: "root", State: "active"}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	platform := models.ExternalPlatform{Name: "p"}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	keys := map[environment.Environment]string{}
	for _, keyEnv := range environment.All() {
		raw, err := service.GenerateAPIKeyFor(keyEnv)
		if err != nil {
			t.Fatal(err)
		}
		row := models.APIKey{Key: service.HashAPIKey(raw), Status: "active", MemberID: &member.ID, ExternalPlatformID: platform.ID, Environment: keyEnv, Prefix: service.APIKeyPrefix(raw)}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		keys[keyEnv] = raw
	}
	legacy := "pm_legacy_key_issued_before_environments"
	if err := db.Create(&models.APIKey{Key: service.HashAPIKey(legacy), Status: "active", MemberID: &member.ID, ExternalPlatformID: platform.ID}).Error; err != nil {
		t.Fatal(err)
	}
	keys["legacy"] = legacy
	auth := service.NewAuthService(repository.NewMemberRepository(db), repository.NewAPIKeyRepository(db), "jwt-secret")
	auth.SetEnvironment(env)
	return auth, keys
}

func serveWithKey(t *testing.T, mw gin.HandlerFunc, key string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", key)
	r.ServeHTTP(w, req)
	body := map[string]any{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body
}

func TestAPIKeyAuth_RejectsKeyFromTheOtherEnvironment(t *testing.T) {
	for _, processEnv := range environment.All() {
		auth, keys := newEnvAuth(t, processEnv)
		for _, mw := range []gin.HandlerFunc{APIKeyAuth(auth), JWTOrAPIKey(auth)} {
			for _, keyEnv := range environment.All() {
				w, body := serveWithKey(t, mw, keys[keyEnv])
				if keyEnv == processEnv {
					if w.Code != http.StatusOK {
						t.Errorf("process %s with %s key: status %d body %v", processEnv, keyEnv, w.Code, body)
					}
					continue
				}
				if w.Code != http.StatusUnauthorized || body["code"] != CodeAPIKeyEnvironmentMismatch {
					t.Errorf("process %s with %s key: status %d body %v, want 401 %s", processEnv, keyEnv, w.Code, body, CodeAPIKeyEnvironmentMismatch)
				}
			}
			// Keys issued before environments existed are test keys by their row default.
			w, body := serveWithKey(t, mw, keys["legacy"])
			if processEnv == environment.Test && w.Code != http.StatusOK {
				t.Errorf("legacy key on test process: status %d body %v", w.Code, body)
			}
			if processEnv == environment.Live && (w.Code != http.StatusUnauthorized || body["code"] != CodeAPIKeyEnvironmentMismatch) {
				t.Errorf("legacy key on live process: status %d body %v", w.Code, body)
			}
		}
	}
}

func TestAPIKeyAuth_PrefixMismatchIsRejectedWithoutALookup(t *testing.T) {
	auth, _ := newEnvAuth(t, environment.Test)
	w, body := serveWithKey(t, APIKeyAuth(auth), "sk_live_0000000000000000000000000000000000000000000000000000000000000000")
	if w.Code != http.StatusUnauthorized || body["code"] != CodeAPIKeyEnvironmentMismatch {
		t.Fatalf("unknown live-prefixed key on test process: status %d body %v", w.Code, body)
	}
}

func TestEnvironmentMiddleware_TagsTheRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(Environment(environment.Live))
	r.GET("/test", func(c *gin.Context) {
		env, ok := environment.FromContext(c.Request.Context())
		c.JSON(200, gin.H{"env": env, "ok": ok})
	})
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	body := map[string]any{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["env"] != "live" || body["ok"] != true {
		t.Fatalf("request context not tagged: %v", body)
	}
}

func TestJWTMiddlewares_RejectSessionsFromTheOtherEnvironment(t *testing.T) {
	for _, processEnv := range environment.All() {
		auth, _ := newEnvAuth(t, processEnv)
		other := environment.Test
		if processEnv == environment.Test {
			other = environment.Live
		}
		otherAuth, _ := newEnvAuth(t, other)
		member := &models.Member{Name: "m", MemberType: "root", State: "active"}
		member.ID = 1
		foreign, err := otherAuth.GenerateJWT(member, 1)
		if err != nil {
			t.Fatal(err)
		}
		own, err := auth.GenerateJWT(member, 1)
		if err != nil {
			t.Fatal(err)
		}
		for name, mw := range map[string]gin.HandlerFunc{"JWTAuth": JWTAuth(auth), "JWTOrAPIKey": JWTOrAPIKey(auth)} {
			w, body := serveWithBearer(t, mw, own)
			if w.Code != http.StatusOK {
				t.Errorf("%s/%s own token: %d %v", name, processEnv, w.Code, body)
			}
			w, body = serveWithBearer(t, mw, foreign)
			if w.Code != http.StatusUnauthorized || body["code"] != CodeSessionEnvironmentMismatch {
				t.Errorf("%s/%s foreign token: %d %v, want 401 %s", name, processEnv, w.Code, body, CodeSessionEnvironmentMismatch)
			}
		}
	}
}

func serveWithBearer(t *testing.T, mw gin.HandlerFunc, token string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	body := map[string]any{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body
}
