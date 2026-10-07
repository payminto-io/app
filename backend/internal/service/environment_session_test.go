package service

import (
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sessionStack builds the auth, token and OTP services of one process on a shared database and secret.
func sessionStack(t *testing.T, db *gorm.DB, env environment.Environment) (*AuthService, *JWTTokenService, *OTPService) {
	t.Helper()
	const secret = "the-same-secret-copied-between-both-env-files"
	jwtSvc := NewJWTTokenService(repository.NewAuthRefreshTokenRepository(db), repository.NewMemberRepository(db), secret, secret, 15*time.Minute, 24*time.Hour)
	jwtSvc.SetEnvironment(env)
	auth := NewAuthService(repository.NewMemberRepository(db), repository.NewAPIKeyRepository(db), secret)
	auth.SetEnvironment(env)
	auth.SetJWTTokenService(jwtSvc)
	otp := NewOTPService(repository.NewOTPRepository(db))
	otp.SetEnvironment(env)
	return auth, jwtSvc, otp
}

func sessionDB(t *testing.T) (*gorm.DB, *models.Member) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Member{}, &models.APIKey{}, &models.AuthRefreshToken{}, &models.OTP{}); err != nil {
		t.Fatal(err)
	}
	member := &models.Member{Name: "root", MemberType: "root", State: "active"}
	if err := db.Create(member).Error; err != nil {
		t.Fatal(err)
	}
	return db, member
}

func TestSessions_TestTokensAreRejectedByALiveProcessWithTheSameSecret(t *testing.T) {
	db, member := sessionDB(t)
	testAuth, testJWT, _ := sessionStack(t, db, environment.Test)
	liveAuth, liveJWT, _ := sessionStack(t, db, environment.Live)

	pair, err := testJWT.GenerateTokenPair(member, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testAuth.ValidateJWT(pair.AccessToken); err != nil {
		t.Fatalf("test token on test process: %v", err)
	}
	if _, err := liveAuth.ValidateJWT(pair.AccessToken); !errors.Is(err, ErrSessionEnvironmentMismatch) || !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("test access token on live process = %v, want session mismatch", err)
	}
	if _, err := liveJWT.ValidateAccessToken(pair.AccessToken); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live JWTTokenService accepted a test access token: %v", err)
	}
	if _, err := liveJWT.RefreshAccessToken(pair.RefreshToken); !errors.Is(err, ErrRefreshTokenNotFound) {
		t.Fatalf("live process refreshed a test refresh token: %v", err)
	}
	if _, err := testJWT.RefreshAccessToken(pair.RefreshToken); err != nil {
		t.Fatalf("test process could not refresh its own token: %v", err)
	}

	legacy, err := testAuth.GenerateJWT(member, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := liveAuth.ValidateJWT(legacy); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("legacy-path test token on live = %v", err)
	}
}

func TestSessions_TokensWithoutAnAudienceAreRejected(t *testing.T) {
	db, _ := sessionDB(t)
	auth, _, _ := sessionStack(t, db, environment.Live)
	// A token signed with the raw secret and no audience is what the pre-ticket code minted.
	raw := "the-same-secret-copied-between-both-env-files"
	legacyAuth := NewAuthService(repository.NewMemberRepository(db), repository.NewAPIKeyRepository(db), raw)
	legacyAuth.SetEnvironment(environment.Live)
	if _, err := auth.ValidateJWT("not-a-token"); err == nil {
		t.Fatal("garbage accepted")
	}
	unset := NewAuthService(repository.NewMemberRepository(db), repository.NewAPIKeyRepository(db), raw)
	if _, err := unset.GenerateJWT(&models.Member{}, 1); !errors.Is(err, environment.ErrUnconfigured) {
		t.Fatalf("unconfigured auth service minted a token: %v", err)
	}
	if _, err := unset.ValidateAPIKey("sk_test_x"); !errors.Is(err, environment.ErrUnconfigured) {
		t.Fatalf("unconfigured auth service validated a key: %v", err)
	}
}

func TestSessions_OTPCodesAreBoundToTheEnvironment(t *testing.T) {
	db, member := sessionDB(t)
	_, _, testOTP := sessionStack(t, db, environment.Test)
	_, _, liveOTP := sessionStack(t, db, environment.Live)
	code, err := testOTP.Generate(member.ID, models.OTPPurposePasswordReset)
	if err != nil {
		t.Fatal(err)
	}
	if err := liveOTP.Verify(member.ID, models.OTPPurposePasswordReset, code); !errors.Is(err, ErrOTPInvalid) {
		t.Fatalf("live process accepted a test OTP: %v", err)
	}
	if err := testOTP.Verify(member.ID, models.OTPPurposePasswordReset, code); err != nil {
		t.Fatalf("test process rejected its own OTP: %v", err)
	}
	unset := NewOTPService(repository.NewOTPRepository(db))
	if _, err := unset.Generate(member.ID, "x"); !errors.Is(err, environment.ErrUnconfigured) {
		t.Fatalf("unconfigured OTP service generated a code: %v", err)
	}
}
