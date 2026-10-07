package service

import (
	"errors"
	"github.com/payminto/payminto/backend/internal/environment"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newAuthServiceForTest builds an AuthService wired to a JWTTokenService backed
// by the provided in-memory DB. Also requires an APIKey table.
func newAuthServiceForTest(t *testing.T, db *gorm.DB) (*AuthService, *JWTTokenService) {
	t.Helper()
	if err := db.AutoMigrate(&models.APIKey{}); err != nil {
		t.Fatalf("migrate APIKey: %v", err)
	}
	memberRepo := repository.NewMemberRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	jwtSvc, _, _ := newTestJWTService(t, db)
	authSvc := NewAuthService(memberRepo, apiKeyRepo, "test-secret")
	authSvc.SetJWTTokenService(jwtSvc)
	return authSvc, jwtSvc
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.AuthRefreshToken{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestJWTService(t *testing.T, db *gorm.DB) (*JWTTokenService, repository.AuthRefreshTokenRepository, repository.MemberRepository) {
	t.Helper()
	refreshRepo := repository.NewAuthRefreshTokenRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	svc := NewJWTTokenService(refreshRepo, memberRepo, "access-secret", "refresh-secret",
		15*time.Minute, 30*24*time.Hour)
	svc.SetEnvironment(environment.Test)
	return svc, refreshRepo, memberRepo
}

func insertTestMember(t *testing.T, memberRepo repository.MemberRepository) *models.Member {
	t.Helper()
	email := "test@example.com"
	m := &models.Member{
		Name:       "Test User",
		Email:      &email,
		MemberType: "root",
		State:      "active",
	}
	if err := memberRepo.Create(m); err != nil {
		t.Fatalf("create member: %v", err)
	}
	return m
}

func TestJWTTokenService_GenerateTokenPair(t *testing.T) {
	db := newTestDB(t)
	svc, _, memberRepo := newTestJWTService(t, db)
	member := insertTestMember(t, memberRepo)

	pair, err := svc.GenerateTokenPair(member, 0)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if pair.ExpiresIn <= 0 {
		t.Error("expected positive ExpiresIn")
	}
}

func TestJWTTokenService_ValidateAccessToken(t *testing.T) {
	db := newTestDB(t)
	svc, _, memberRepo := newTestJWTService(t, db)
	member := insertTestMember(t, memberRepo)

	pair, err := svc.GenerateTokenPair(member, 0)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.MemberID != member.ID {
		t.Errorf("claims.MemberID = %d, want %d", claims.MemberID, member.ID)
	}
}

func TestJWTTokenService_RefreshHappyPath(t *testing.T) {
	db := newTestDB(t)
	svc, _, memberRepo := newTestJWTService(t, db)
	member := insertTestMember(t, memberRepo)

	pair, err := svc.GenerateTokenPair(member, 0)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	newPair, err := svc.RefreshAccessToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if newPair.AccessToken == "" || newPair.RefreshToken == "" {
		t.Error("expected non-empty tokens after refresh")
	}
	// Old token should be revoked now — reuse should fail.
	_, err = svc.RefreshAccessToken(pair.RefreshToken)
	if !errors.Is(err, ErrRefreshTokenReused) {
		t.Errorf("expected ErrRefreshTokenReused, got %v", err)
	}
}

func TestJWTTokenService_RevokeRefreshToken(t *testing.T) {
	db := newTestDB(t)
	svc, _, memberRepo := newTestJWTService(t, db)
	member := insertTestMember(t, memberRepo)

	pair, err := svc.GenerateTokenPair(member, 0)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if err := svc.RevokeRefreshToken(pair.RefreshToken); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}

	// Revoked token should not rotate.
	_, err = svc.RefreshAccessToken(pair.RefreshToken)
	if !errors.Is(err, ErrRefreshTokenReused) {
		t.Errorf("expected ErrRefreshTokenReused after explicit revocation, got %v", err)
	}
}

func TestJWTTokenService_RevokeAllForMember(t *testing.T) {
	db := newTestDB(t)
	svc, _, memberRepo := newTestJWTService(t, db)
	member := insertTestMember(t, memberRepo)

	pair1, _ := svc.GenerateTokenPair(member, 0)
	pair2, _ := svc.GenerateTokenPair(member, 0)

	if err := svc.RevokeAllForMember(member.ID); err != nil {
		t.Fatalf("RevokeAllForMember: %v", err)
	}

	for _, tok := range []string{pair1.RefreshToken, pair2.RefreshToken} {
		_, err := svc.RefreshAccessToken(tok)
		if err == nil {
			t.Error("expected error after RevokeAllForMember, got nil")
		}
	}
}

// TestResetPassword_InvalidatesAllRefreshTokens verifies that after a successful
// ResetPassword, previously-issued refresh tokens are revoked and cannot be used
// to obtain new access tokens.
func TestResetPassword_InvalidatesAllRefreshTokens(t *testing.T) {
	db := newTestDB(t)
	authSvc, jwtSvc := newAuthServiceForTest(t, db)
	memberRepo := repository.NewMemberRepository(db)
	member := insertTestMember(t, memberRepo)

	// Issue two token pairs (simulating two active sessions).
	pair1, err := jwtSvc.GenerateTokenPair(member, 0)
	if err != nil {
		t.Fatalf("GenerateTokenPair session 1: %v", err)
	}
	pair2, err := jwtSvc.GenerateTokenPair(member, 0)
	if err != nil {
		t.Fatalf("GenerateTokenPair session 2: %v", err)
	}

	// Simulate ForgotPassword storing a reset token.
	resetToken := "test-reset-token-abc123"
	expiry := time.Now().Add(time.Hour)
	member.ResetPasswordToken = &resetToken
	member.ResetPasswordExpiry = &expiry
	if err := memberRepo.Update(member); err != nil {
		t.Fatalf("store reset token: %v", err)
	}

	// Reset the password.
	if err := authSvc.ResetPassword(resetToken, "newSecurePassword123!"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Both previously-issued refresh tokens must now be invalid.
	for i, tok := range []string{pair1.RefreshToken, pair2.RefreshToken} {
		_, err := jwtSvc.RefreshAccessToken(tok)
		if err == nil {
			t.Errorf("session %d: expected error after password reset, got nil", i+1)
		}
		if !errors.Is(err, ErrRefreshTokenReused) && !errors.Is(err, ErrRefreshTokenNotFound) {
			t.Errorf("session %d: expected ErrRefreshTokenReused or ErrRefreshTokenNotFound, got %v", i+1, err)
		}
	}
}

func TestHashToken_ConstantTime(t *testing.T) {
	raw := "some-token"
	h := hashToken(raw)
	if !compareTokenHash(raw, h) {
		t.Error("compareTokenHash should return true for matching raw+hash")
	}
	if compareTokenHash("wrong", h) {
		t.Error("compareTokenHash should return false for non-matching input")
	}
}
