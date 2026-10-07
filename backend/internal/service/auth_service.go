package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService handles member authentication via JWT tokens and API keys.
type AuthService struct {
	memberRepo   repository.MemberRepository
	apiKeyRepo   repository.APIKeyRepository
	mepRoleRepo  repository.MemberExternalPlatformRoleRepository
	jwtTokenSvc  *JWTTokenService
	platformSvc  *ExternalPlatformService
	eventEmitter *EventEmitterService
	jwtSecret    string
	// environment is the process environment; a key from the other one is refused (ticket 13).
	environment environment.Environment
}

// ErrAPIKeyEnvironmentMismatch wraps environment.ErrMismatch for keys that belong to the other environment.
var ErrAPIKeyEnvironmentMismatch = errors.New("API key belongs to the other environment")

// apiKeyEnvironmentError satisfies errors.Is for both the service and the environment sentinel.
type apiKeyEnvironmentError struct {
	process, key environment.Environment
}

func (e *apiKeyEnvironmentError) Error() string {
	return fmt.Sprintf("API key is a %s key but this server is %s", e.key, e.process)
}

func (e *apiKeyEnvironmentError) Is(target error) bool {
	return target == ErrAPIKeyEnvironmentMismatch || target == environment.ErrMismatch
}

// NewAuthService constructs an AuthService with the given member/API key repositories and JWT secret.
func NewAuthService(memberRepo repository.MemberRepository, apiKeyRepo repository.APIKeyRepository, jwtSecret string) *AuthService {
	return &AuthService{memberRepo: memberRepo, apiKeyRepo: apiKeyRepo, jwtSecret: jwtSecret}
}

// SetJWTTokenService injects the JWTTokenService via Pass 2 DI (avoids circular deps).
func (s *AuthService) SetJWTTokenService(svc *JWTTokenService) {
	s.jwtTokenSvc = svc
}

// SetExternalPlatformService injects the platform service for auto-creating
// a default project on root signup.
func (s *AuthService) SetExternalPlatformService(svc *ExternalPlatformService) {
	s.platformSvc = svc
}

// SetMEPRoleRepo injects the member-external-platform-role repository for
// assigning the root role on signup and resolving platforms on signin.
func (s *AuthService) SetMEPRoleRepo(repo repository.MemberExternalPlatformRoleRepository) {
	s.mepRoleRepo = repo
}

// SetEventEmitter injects the EventEmitterService for emitting welcome + password-reset events.
func (s *AuthService) SetEventEmitter(svc *EventEmitterService) {
	s.eventEmitter = svc
}

// SetEnvironment sets the process environment keys are validated and issued against.
func (s *AuthService) SetEnvironment(env environment.Environment) { s.environment = env }

// Environment is the process environment, defaulting to test so a bare service never admits live keys.
func (s *AuthService) Environment() environment.Environment {
	if s.environment == "" {
		return environment.Test
	}
	return s.environment
}

// GenerateAPIKey issues a test secret key; callers that know the process environment use GenerateAPIKeyFor.
func GenerateAPIKey() (string, error) {
	return GenerateAPIKeyFor(environment.Test)
}

// GenerateAPIKeyFor generates a random 32-byte secret key with the visible prefix of env ("sk_live_...").
func GenerateAPIKeyFor(env environment.Environment) (string, error) {
	if !env.Valid() {
		return "", fmt.Errorf("%w: %q", environment.ErrInvalid, env)
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return env.KeyPrefix(environment.SecretKey) + hex.EncodeToString(b), nil
}

// APIKeyPrefix is the part of a raw key safe to show in a list ("sk_live_ab12").
func APIKeyPrefix(rawKey string) string {
	const shown = 12
	if len(rawKey) <= shown {
		return rawKey
	}
	return rawKey[:shown]
}

// NewAPIKeyRow builds the stored row for a freshly generated key in env.
func NewAPIKeyRow(rawKey string, env environment.Environment, platformID uint) *models.APIKey {
	return &models.APIKey{
		Key:                HashAPIKey(rawKey),
		Status:             "active",
		ExternalPlatformID: platformID,
		Environment:        env,
		Prefix:             APIKeyPrefix(rawKey),
	}
}

// HashAPIKey returns the SHA-256 hex digest of the raw API key. This hash is stored in the DB.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// HashPassword returns a bcrypt hash of the password using DefaultCost.
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword reports whether password matches the bcrypt hash.
func (s *AuthService) CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// JWTClaims extends jwt.RegisteredClaims with Payminto-specific member identity fields.
type JWTClaims struct {
	MemberID           uint   `json:"memberID"`
	Email              string `json:"email"`
	MemberType         string `json:"memberType"`
	ExternalPlatformID uint   `json:"externalPlatformID"`
	jwt.RegisteredClaims
}

// GenerateJWT issues a signed 24-hour HS256 JWT for the given member and external platform.
func (s *AuthService) GenerateJWT(member *models.Member, platformID uint) (string, error) {
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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// ValidateJWT parses and validates a JWT string, returning the embedded JWTClaims on success.
func (s *AuthService) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ValidateAPIKey looks up and validates a raw API key, returning the active APIKey record.
// A key whose prefix or row names the other environment is refused before anything else.
func (s *AuthService) ValidateAPIKey(key string) (*models.APIKey, error) {
	process := s.Environment()
	if keyEnv, _, ok := environment.KeyEnvironment(key); ok && keyEnv != process {
		return nil, &apiKeyEnvironmentError{process: process, key: keyEnv}
	}
	hash := HashAPIKey(key)
	apiKey, err := s.apiKeyRepo.GetByKey(hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid API key")
		}
		return nil, errors.New("invalid API key")
	}
	if apiKey.Status != "active" {
		return nil, errors.New("invalid API key")
	}
	if apiKey.ExpireAt != nil && apiKey.ExpireAt.Before(time.Now()) {
		return nil, errors.New("API key expired")
	}
	rowEnv := apiKey.Environment
	if rowEnv == "" {
		rowEnv = environment.Test
	}
	if rowEnv != process {
		return nil, &apiKeyEnvironmentError{process: process, key: rowEnv}
	}
	return apiKey, nil
}

// Signup creates a new member account, hashes the password, generates a token
// pair, and emits a welcome email event. If a root member already exists for the
// platform, the new member is created as type "internal".
func (s *AuthService) Signup(emailAddr, password, name string) (*models.Member, *TokenPair, error) {
	// Check whether a root member already exists.
	count, err := s.memberRepo.Count()
	if err != nil {
		return nil, nil, fmt.Errorf("count members: %w", err)
	}

	memberType := "root"
	if count > 0 {
		memberType = "internal"
	}

	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	member := &models.Member{
		Name:       name,
		Email:      &emailAddr,
		Password:   &hash,
		MemberType: memberType,
		State:      "active",
	}
	if err := s.memberRepo.Create(member); err != nil {
		return nil, nil, fmt.Errorf("create member: %w", err)
	}

	// For root members: auto-create a default ExternalPlatform ("Default Project")
	// and assign the root role. This ensures the JWT carries a valid platformID
	// so payment/withdrawal APIs work immediately after signup.
	var platformID uint
	if memberType == "root" && s.platformSvc != nil {
		platform, _, err := s.platformSvc.Create(ExternalPlatformInput{
			MemberID: member.ID,
			Name:     "Default Project",
			Website:  "https://payminto.local",
		})
		if err != nil {
			log.Printf("[AuthService] Signup: auto-create platform for root member %d: %v", member.ID, err)
		} else {
			platformID = platform.ID
			// Assign root role to the member on this platform.
			if s.mepRoleRepo != nil {
				rootRole := &models.MemberExternalPlatformRole{
					MemberID:           member.ID,
					ExternalPlatformID: platform.ID,
					RoleID:             1, // root role (seeded as ID 1)
				}
				if err := s.mepRoleRepo.Assign(rootRole); err != nil {
					log.Printf("[AuthService] Signup: assign root role for member %d platform %d: %v", member.ID, platform.ID, err)
				}
			}
		}
	}

	pair, err := s.issueTokenPair(member, platformID)
	if err != nil {
		return nil, nil, err
	}

	// Emit welcome email event (best-effort).
	if s.eventEmitter != nil {
		_ = s.eventEmitter.EmitEmail(EmailPayload{
			To:       emailAddr,
			Template: "welcome",
			Subject:  "Welcome to Payminto",
			Data: map[string]any{
				"Name": name,
			},
		})
	}

	return member, pair, nil
}

// Signin authenticates a member by email+password and issues a token pair.
// Resolves the member's first linked ExternalPlatform so the JWT carries
// the correct platformID for merchant routes.
func (s *AuthService) Signin(emailAddr, password string) (*models.Member, *TokenPair, error) {
	member, err := s.memberRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, fmt.Errorf("lookup member: %w", err)
	}

	if member.Password == nil || !s.CheckPassword(*member.Password, password) {
		return nil, nil, errors.New("invalid credentials")
	}

	if member.State != "active" {
		log.Printf("[AuthService] Signin: member %d (email=%s) is in state %q — returning generic error",
			member.ID, emailAddr, member.State)
		return nil, nil, errors.New("invalid credentials")
	}

	// Resolve the member's linked platform for the JWT.
	var platformID uint
	if s.mepRoleRepo != nil {
		roles, err := s.mepRoleRepo.ListByMember(member.ID)
		if err == nil && len(roles) > 0 {
			platformID = roles[0].ExternalPlatformID
		}
	}

	pair, err := s.issueTokenPair(member, platformID)
	if err != nil {
		return nil, nil, err
	}

	return member, pair, nil
}

// SignOut revokes the provided refresh token.
func (s *AuthService) SignOut(refreshToken string) error {
	if s.jwtTokenSvc == nil {
		return errors.New("JWTTokenService not configured")
	}
	return s.jwtTokenSvc.RevokeRefreshToken(refreshToken)
}

// SignOutAll revokes every refresh token for the member.
func (s *AuthService) SignOutAll(memberID uint) error {
	if s.jwtTokenSvc == nil {
		return errors.New("JWTTokenService not configured")
	}
	return s.jwtTokenSvc.RevokeAllForMember(memberID)
}

// ForgotPassword generates a password-reset token, stores it on the member row,
// and emits a password_reset email event. Returns the raw token (callers store
// it temporarily; the email delivery happens async via EventEmitter).
func (s *AuthService) ForgotPassword(emailAddr string) (string, error) {
	member, err := s.memberRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return no error to avoid email enumeration.
			return "", nil
		}
		return "", fmt.Errorf("lookup member: %w", err)
	}

	token, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}

	expiry := time.Now().Add(time.Hour)
	member.ResetPasswordToken = &token
	member.ResetPasswordExpiry = &expiry
	if err := s.memberRepo.Update(member); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	// Emit password reset email event (best-effort).
	if s.eventEmitter != nil && member.Email != nil {
		_ = s.eventEmitter.EmitEmail(EmailPayload{
			To:       *member.Email,
			Template: "password_reset",
			Subject:  "Reset your Payminto password",
			Data: map[string]any{
				"ResetURL": "https://app.payminto.com/auth/reset-password?token=" + token,
			},
		})
	}

	return token, nil
}

// ResetPassword validates the reset token, hashes the new password, and clears
// the reset fields on the member row.
func (s *AuthService) ResetPassword(token, newPassword string) error {
	member, err := s.memberRepo.GetByResetPasswordToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid or expired reset token")
		}
		return fmt.Errorf("lookup reset token: %w", err)
	}

	if member.ResetPasswordExpiry != nil && time.Now().After(*member.ResetPasswordExpiry) {
		return errors.New("reset token expired")
	}

	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	member.Password = &hash
	member.ResetPasswordToken = nil
	member.ResetPasswordExpiry = nil
	member.ResetPasswordRequired = false

	if err := s.memberRepo.Update(member); err != nil {
		return fmt.Errorf("update member: %w", err)
	}

	// Invalidate all existing refresh tokens so that sessions that existed
	// before the password reset cannot continue. This prevents a compromised
	// session from persisting after a successful password reset.
	if s.jwtTokenSvc != nil {
		if err := s.jwtTokenSvc.RevokeAllForMember(member.ID); err != nil {
			// Log but don't fail the reset — the password change is the critical operation.
			log.Printf("[AuthService] ResetPassword: revoke tokens for member %d: %v", member.ID, err)
		}
	}
	return nil
}

// issueTokenPair uses JWTTokenService if available, otherwise falls back to
// the legacy GenerateJWT path.
func (s *AuthService) issueTokenPair(member *models.Member, platformID uint) (*TokenPair, error) {
	if s.jwtTokenSvc != nil {
		return s.jwtTokenSvc.GenerateTokenPair(member, platformID)
	}
	// Fallback: access-only pair (no refresh token persisted).
	access, err := s.GenerateJWT(member, platformID)
	if err != nil {
		return nil, fmt.Errorf("generate JWT: %w", err)
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: "",
		ExpiresIn:    int64((24 * time.Hour).Seconds()),
	}, nil
}

// generateResetToken produces a cryptographically random 32-byte hex token.
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
