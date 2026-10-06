package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/api/middleware"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRBACTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.Member{},
		&models.ExternalPlatform{},
		&models.MemberExternalPlatformRole{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func setupRBACTest(t *testing.T) (*service.MemberExternalPlatformRoleService, uint, uint) {
	t.Helper()
	db := newRBACTestDB(t)
	repo := repository.NewMemberExternalPlatformRoleRepository(db)
	svc := service.NewMemberExternalPlatformRoleService(repo)

	perm := &models.Permission{Name: "payments.read", DisplayName: "Read", Description: "view"}
	db.Create(perm)

	role := &models.Role{Name: "viewer", DisplayName: "Viewer", Permissions: []models.Permission{*perm}}
	db.Create(role)

	email := "u@example.com"
	member := &models.Member{Name: "User", Email: &email, State: "active", MemberType: "merchant"}
	db.Create(member)

	platform := &models.ExternalPlatform{Name: "P"}
	db.Create(platform)

	_ = svc.Assign(t.Context(), member.ID, platform.ID, role.ID)

	return svc, member.ID, platform.ID
}

func TestRequirePermission_Unauthenticated(t *testing.T) {
	mepSvc, _, _ := setupRBACTest(t)

	r := gin.New()
	r.Use(middleware.RequirePermission(mepSvc, "payments.read"))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequirePermission_MissingPermission(t *testing.T) {
	mepSvc, memberID, platformID := setupRBACTest(t)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("memberID", memberID)
		c.Set("externalPlatformID", platformID)
		c.Next()
	})
	r.Use(middleware.RequirePermission(mepSvc, "system.admin"))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	mepSvc, memberID, platformID := setupRBACTest(t)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("memberID", memberID)
		c.Set("externalPlatformID", platformID)
		c.Next()
	})
	r.Use(middleware.RequirePermission(mepSvc, "payments.read"))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
