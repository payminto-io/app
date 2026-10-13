package service_test

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/api/middleware"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func signupAuthorityFixture(t *testing.T) (*gorm.DB, *service.AuthService, *service.MemberExternalPlatformRoleService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.Member{}, &models.ExternalPlatform{}, &models.APIKey{}, &models.Role{}, &models.Permission{}, &models.MemberExternalPlatformRole{}); err != nil {
		t.Fatal(err)
	}
	members := repository.NewMemberRepository(db)
	roles := repository.NewRoleRepository(db)
	permissions := repository.NewPermissionRepository(db)
	assignments := repository.NewMemberExternalPlatformRoleRepository(db)
	if err := roles.Create(&models.Role{Name: "unrelated-role", DisplayName: "Existing role"}); err != nil {
		t.Fatal(err)
	}
	if err := service.NewPermissionService(permissions).SeedDefaults(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := service.NewRoleService(roles, permissions).SeedDefaults(t.Context()); err != nil {
		t.Fatal(err)
	}
	auth := service.NewAuthService(members, repository.NewAPIKeyRepository(db), "isolated-signup-test-session-secret")
	auth.SetEnvironment(environment.Test)
	platforms := service.NewExternalPlatformService(repository.NewExternalPlatformRepository(db), repository.NewAPIKeyRepository(db))
	platforms.SetEnvironment(environment.Test)
	auth.SetExternalPlatformService(platforms)
	auth.SetMEPRoleRepo(assignments)
	auth.SetRoleRepo(roles)
	guard := service.NewMemberExternalPlatformRoleService(assignments)
	guard.SetMemberRepository(members)
	return db, auth, guard
}

func TestSignupMerchantOwnsSeparatePlatformWithoutGlobalAuthority(t *testing.T) {
	db, auth, guard := signupAuthorityFixture(t)
	root, rootTokens, err := auth.Signup("bootstrap@example.test", "signup-test-password", "Installation")
	if err != nil {
		t.Fatal(err)
	}
	rootClaims, err := auth.ValidateJWT(rootTokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if root.MemberType != "root" || !guard.HasPermission(t.Context(), root.ID, rootClaims.ExternalPlatformID, "system.admin") {
		t.Fatal("installation root lost its global authority")
	}
	merchant, tokens, err := auth.Signup("merchant@example.test", "signup-test-password", "Merchant")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ValidateJWT(tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if merchant.MemberType == "root" || claims.ExternalPlatformID == 0 || claims.ExternalPlatformID == rootClaims.ExternalPlatformID {
		t.Fatal("merchant platform is missing or shares installation platform")
	}
	for _, permission := range []string{"payments.create", "wallets.read", "withdrawals.create", "analytics.read"} {
		if !guard.HasPermission(t.Context(), merchant.ID, claims.ExternalPlatformID, permission) {
			t.Fatalf("merchant lost commerce permission %s", permission)
		}
	}
	for _, permission := range []string{"system.admin", "roles.manage", "settings.read", "settings.write", "members.read", "members.invite", "members.remove"} {
		if guard.HasPermission(t.Context(), merchant.ID, claims.ExternalPlatformID, permission) {
			t.Fatalf("ordinary signup obtained global permission %s", permission)
		}
	}
	if guard.HasPermission(t.Context(), merchant.ID, rootClaims.ExternalPlatformID, "payments.create") {
		t.Fatal("merchant obtained another tenant's commerce permission")
	}
	_, signinTokens, err := auth.Signin("merchant@example.test", "signup-test-password")
	if err != nil {
		t.Fatal(err)
	}
	signinClaims, err := auth.ValidateJWT(signinTokens.AccessToken)
	if err != nil || signinClaims.ExternalPlatformID != claims.ExternalPlatformID {
		t.Fatalf("signin did not restore merchant project: %v", err)
	}
	var platform models.ExternalPlatform
	if err := db.First(&platform, claims.ExternalPlatformID).Error; err != nil || platform.Name != "Merchant" {
		t.Fatalf("merchant platform not created: %v", err)
	}
}

func TestLegacyMerchantOwnerCannotUseGlobalAuthority(t *testing.T) {
	db, auth, guard := signupAuthorityFixture(t)
	_, _, err := auth.Signup("bootstrap@example.test", "signup-test-password", "Installation")
	if err != nil {
		t.Fatal(err)
	}
	merchant, pair, err := auth.Signup("legacy@example.test", "signup-test-password", "Legacy merchant")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ValidateJWT(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := repository.NewRoleRepository(db).GetByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.NewMemberExternalPlatformRoleRepository(db).Assign(&models.MemberExternalPlatformRole{MemberID: merchant.ID, ExternalPlatformID: claims.ExternalPlatformID, RoleID: owner.ID}); err != nil {
		t.Fatal(err)
	}
	if guard.HasPermission(t.Context(), merchant.ID, claims.ExternalPlatformID, "system.admin") {
		t.Fatal("existing non-root owner still has global authority")
	}
	if !guard.HasPermission(t.Context(), merchant.ID, claims.ExternalPlatformID, "payments.create") {
		t.Fatal("existing merchant lost commerce access")
	}
}

func TestInstallationPermissionsRecheckPersistedIdentity(t *testing.T) {
	db, auth, guard := signupAuthorityFixture(t)
	root, tokens, err := auth.Signup("bootstrap@example.test", "signup-test-password", "Installation")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ValidateJWT(tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	globals := []string{"system.admin", "roles.manage", "settings.read", "settings.write", "members.read", "members.invite", "members.remove"}
	for _, p := range globals {
		if !guard.HasPermission(t.Context(), root.ID, claims.ExternalPlatformID, p) {
			t.Fatalf("root lacks %s", p)
		}
	}
	for _, change := range []map[string]any{{"state": "inactive"}, {"state": "active", "member_type": "internal"}} {
		if err := db.Model(root).Updates(change).Error; err != nil {
			t.Fatal(err)
		}
		for _, p := range globals {
			if guard.HasPermission(t.Context(), root.ID, claims.ExternalPlatformID, p) {
				t.Fatalf("cached permission bypasses persisted identity: %s", p)
			}
		}
	}
	if err := db.Model(root).Updates(map[string]any{"state": "active", "member_type": "root"}).Error; err != nil {
		t.Fatal(err)
	}
	withoutIdentity := service.NewMemberExternalPlatformRoleService(repository.NewMemberExternalPlatformRoleRepository(db))
	if withoutIdentity.HasPermission(t.Context(), root.ID, claims.ExternalPlatformID, "system.admin") {
		t.Fatal("missing identity dependency must fail closed even for an active root with the owner role")
	}
	if err := db.Delete(root).Error; err != nil {
		t.Fatal(err)
	}
	if guard.HasPermission(t.Context(), root.ID, claims.ExternalPlatformID, "system.admin") {
		t.Fatal("deleted root retains global authority")
	}

}

func TestSignupFailsClosedWhenProvisioningIncomplete(t *testing.T) {
	for _, scenario := range []string{"missing platform service", "missing role catalogue", "missing assignment repository", "missing seeded role", "invalid environment", "platform key environment", "token issuance", "platform creation", "role assignment", "member activation"} {
		t.Run(scenario, func(t *testing.T) {
			db, auth, _ := signupAuthorityFixture(t)
			switch scenario {
			case "missing platform service":
				auth.SetExternalPlatformService(nil)
			case "missing role catalogue":
				auth.SetRoleRepo(nil)
			case "missing assignment repository":
				auth.SetMEPRoleRepo(nil)
			case "missing seeded role":
				if err := db.Where("name = ?", "owner").Delete(&models.Role{}).Error; err != nil {
					t.Fatal(err)
				}
			case "invalid environment":
				auth.SetEnvironment("")
			case "platform key environment":
				auth.SetExternalPlatformService(service.NewExternalPlatformService(repository.NewExternalPlatformRepository(db), repository.NewAPIKeyRepository(db)))
			case "token issuance":
				auth.SetJWTTokenService(service.NewJWTTokenService(repository.NewAuthRefreshTokenRepository(db), repository.NewMemberRepository(db), "test", "test", 0, 0))
			case "platform creation", "role assignment":
				table := "external_platforms"
				if scenario == "role assignment" {
					table = "member_external_platform_roles"
				}
				if err := db.Callback().Create().Before("gorm:create").Register("fail_signup", func(tx *gorm.DB) {
					if tx.Statement.Table == table {
						_ = tx.AddError(errors.New("injected provisioning failure"))
					}
				}); err != nil {
					t.Fatal(err)
				}
			case "member activation":
				if err := db.Callback().Update().Before("gorm:update").Register("fail_activation", func(tx *gorm.DB) {
					if tx.Statement.Table == "members" {
						_ = tx.AddError(errors.New("injected activation failure"))
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			member, pair, err := auth.Signup("incomplete@example.test", "signup-test-password", "Incomplete")
			if err == nil || member != nil || pair != nil {
				t.Fatal("incomplete signup returned member/session")
			}
			var active int64
			if err := db.Model(&models.Member{}).Where("state = ?", "active").Count(&active).Error; err != nil {
				t.Fatal(err)
			}
			if active != 0 {
				t.Fatal("incomplete signup left authenticatable member")
			}
			if _, pair, err := auth.Signin("incomplete@example.test", "signup-test-password"); err == nil || pair != nil {
				t.Fatal("incomplete account can sign in")
			}
		})
	}
}

func TestSignupSessionsEnforceGlobalRBACAtHTTPBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, auth, guard := signupAuthorityFixture(t)
	_, rootTokens, err := auth.Signup("bootstrap@example.test", "signup-test-password", "Installation")
	if err != nil {
		t.Fatal(err)
	}
	merchant, merchantTokens, err := auth.Signup("merchant@example.test", "signup-test-password", "Merchant")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ValidateJWT(merchantTokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := repository.NewRoleRepository(db).GetByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.NewMemberExternalPlatformRoleRepository(db).Assign(&models.MemberExternalPlatformRole{MemberID: merchant.ID, ExternalPlatformID: claims.ExternalPlatformID, RoleID: owner.ID}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	for _, permission := range []string{"system.admin", "roles.manage", "settings.write", "members.remove", "payments.create"} {
		router.GET("/"+permission, middleware.JWTAuth(auth), middleware.RequirePermission(guard, permission), func(c *gin.Context) { c.Status(http.StatusOK) })
	}
	for _, check := range []struct {
		token, permission string
		want              int
	}{
		{rootTokens.AccessToken, "system.admin", http.StatusOK},
		{merchantTokens.AccessToken, "system.admin", http.StatusForbidden},
		{merchantTokens.AccessToken, "roles.manage", http.StatusForbidden},
		{merchantTokens.AccessToken, "settings.write", http.StatusForbidden},
		{merchantTokens.AccessToken, "members.remove", http.StatusForbidden},
		{merchantTokens.AccessToken, "payments.create", http.StatusOK},
	} {
		request := httptest.NewRequest(http.MethodGet, "/"+check.permission, nil)
		request.Header.Set("Authorization", "Bearer "+check.token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != check.want {
			t.Fatalf("%s status = %d, want %d", check.permission, response.Code, check.want)
		}
	}
}
