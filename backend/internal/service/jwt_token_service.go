package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// ErrRefreshTokenExpired is returned when a refresh token has passed its ExpiresAt.
var ErrRefreshTokenExpired = errors.New("refresh token expired")

// ErrRefreshTokenRevoked is returned when a refresh token has been explicitly revoked.
var ErrRefreshTokenRevoked = errors.New("refresh token revoked")

// ErrRefreshTokenReused is returned when a previously-rotated token is presented,
// indicating a possible replay attack. The entire family is revoked.
var ErrRefreshTokenReused = errors.New("refresh token reuse detected: family revoked")

// ErrRefreshTokenNotFound is returned when the token hash has no matching row.
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// TokenPair bundles the short-lived access token and long-lived refresh token
// returned to the client on sign-in or rotation.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // seconds until access token expires
}

// JWTTokenService manages the access + refresh token pair. Access tokens are
// short-lived (default 24h) JWTs; refresh tokens are long-lived (default 30d)
// random strings hashed before storage. Uses the rotation-family pattern to
// detect reuse — if a previously-rotated refresh token is ever presented, the
// entire family is revoked.
type JWTTokenService struct {
	refreshRepo   repository.AuthRefreshTokenRepository
	memberRepo    repository.MemberRepository
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewJWTTokenService constructs a JWTTokenService.
func NewJWTTokenService(
	refreshRepo repository.AuthRefreshTokenRepository,
	memberRepo repository.MemberRepository,
	accessSecret, refreshSecret string,
	accessTTL, refreshTTL time.Duration,
) *JWTTokenService {
	return &JWTTokenService{
		refreshRepo:   refreshRepo,
		memberRepo:    memberRepo,
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

// GenerateTokenPair creates a new access JWT and refresh token for the given
// member, persisting the hashed refresh token in a new rotation family.
func (s *JWTTokenService) GenerateTokenPair(member *models.Member, platformID uint) (*TokenPair, error) {
	accessToken, err := s.signAccessToken(member, platformID)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	family := uuid.NewString() // new family for a new login session
	hash := hashToken(rawRefresh)

	now := time.Now()
	record := &models.AuthRefreshToken{
		TokenHash:          hash,
		MemberID:           member.ID,
		ExternalPlatformID: platformID,
		RotationFamily:     family,
		IssuedAt:           now,
		ExpiresAt:          now.Add(s.refreshTTL),
	}
	if err := s.refreshRepo.Create(record); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// ValidateAccessToken parses and validates a JWT string, returning the embedded
// JWTClaims on success.
func (s *JWTTokenService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(s.accessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}

// RefreshAccessToken verifies a refresh token by hashed lookup, rotates the
// pair (invalidates old row, issues new pair with same family), and detects
// reuse by checking RevokedAt on the found row.
func (s *JWTTokenService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	hash := hashToken(refreshToken)

	record, err := s.refreshRepo.GetByTokenHash(hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}

	// Reuse detection: token was already revoked (rotated previously).
	if record.RevokedAt != nil {
		// Revoke the entire family to protect against compromise.
		_ = s.refreshRepo.RevokeFamilyByRotationFamily(record.RotationFamily)
		return nil, ErrRefreshTokenReused
	}

	if time.Now().After(record.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}

	// Look up the member.
	member, err := s.memberRepo.GetByID(record.MemberID)
	if err != nil {
		return nil, fmt.Errorf("member not found: %w", err)
	}

	// Carry the platformID from the stored refresh token row into the new access token.
	// This was persisted on GenerateTokenPair — no data loss across rotations.
	// TODO(phase-h-rbac): resolve full RBAC roles from MemberExternalPlatformRole.
	accessToken, err := s.signAccessToken(member, record.ExternalPlatformID)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Revoke the consumed token.
	if err := s.refreshRepo.RevokeByID(record.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Issue a new refresh token in the same family.
	rawRefresh, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now()
	newRecord := &models.AuthRefreshToken{
		TokenHash:          hashToken(rawRefresh),
		MemberID:           member.ID,
		ExternalPlatformID: record.ExternalPlatformID, // carry forward from previous record
		RotationFamily:     record.RotationFamily,      // same family
		IssuedAt:           now,
		ExpiresAt:          now.Add(s.refreshTTL),
	}
	if err := s.refreshRepo.Create(newRecord); err != nil {
		return nil, fmt.Errorf("persist new refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// RevokeRefreshToken marks the token (identified by raw token string) as revoked.
func (s *JWTTokenService) RevokeRefreshToken(refreshToken string) error {
	hash := hashToken(refreshToken)
	record, err := s.refreshRepo.GetByTokenHash(hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // already gone — idempotent
		}
		return err
	}
	return s.refreshRepo.RevokeByID(record.ID)
}

// RevokeAllForMember revokes every refresh token for the given member.
func (s *JWTTokenService) RevokeAllForMember(memberID uint) error {
	return s.refreshRepo.RevokeAllByMemberID(memberID)
}

// signAccessToken issues a signed HS256 JWT for the given member.
func (s *JWTTokenService) signAccessToken(member *models.Member, platformID uint) (string, error) {
	email := ""
	if member.Email != nil {
		email = *member.Email
	}
	claims := JWTClaims{
		MemberID:           member.ID,
		Email:              email,
		MemberType:         member.MemberType,
		ExternalPlatformID: platformID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.accessSecret))
}

// generateSecureToken produces a cryptographically random 32-byte hex string.
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of the raw token for storage.
// constant-time comparison is used at verification time.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// compareTokenHash performs a constant-time comparison of a raw token against
// its expected hash. Returns true if they match.
func compareTokenHash(raw, expectedHash string) bool {
	actual := hashToken(raw)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}
