package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/payminto/payminto/backend/internal/environment"
	"math/big"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// ErrOTPNotFound is returned when no active OTP can be found for the member+purpose.
var ErrOTPNotFound = errors.New("OTP not found")

// ErrOTPExpired is returned when the OTP has passed its ExpiresAt.
var ErrOTPExpired = errors.New("OTP expired")

// ErrOTPInvalid is returned when the provided code does not match.
var ErrOTPInvalid = errors.New("OTP invalid")

// ErrOTPMaxAttempts is returned when the OTP has been tried too many times.
var ErrOTPMaxAttempts = errors.New("OTP max attempts exceeded")

// ErrOTPAlreadyUsed is returned when the OTP has already been consumed.
var ErrOTPAlreadyUsed = errors.New("OTP already used")

const (
	otpTTL         = 5 * time.Minute
	otpMaxAttempts = 5
)

// OTPService generates, stores, and validates one-time passwords for withdrawal
// approval, password reset, email verification, and other sensitive flows. Codes
// are 6 digits, hashed via sha256 before storage, expire after 5 minutes, and are
// invalidated after 5 failed attempts.
type OTPService struct {
	otpRepo repository.OTPRepository
	// environment is folded into every code hash so a code issued by test never verifies on live.
	environment environment.Environment
}

// NewOTPService constructs an OTPService.
func NewOTPService(otpRepo repository.OTPRepository) *OTPService {
	return &OTPService{otpRepo: otpRepo}
}

// SetEnvironment sets the process environment; Generate and Verify fail closed until it is set.
func (s *OTPService) SetEnvironment(env environment.Environment) { s.environment = env }

func (s *OTPService) env() (environment.Environment, error) {
	if !s.environment.Valid() {
		return "", environment.ErrUnconfigured
	}
	return s.environment, nil
}

// Generate creates a new OTP row with a hashed 6-digit code for the given member
// and purpose. Any previous pending OTPs for the same member+purpose are
// invalidated. Returns the plaintext code for delivery via email.
func (s *OTPService) Generate(memberID uint, purpose string) (string, error) {
	// Invalidate any existing pending OTPs for this member+purpose.
	if err := s.otpRepo.InvalidateByMemberAndPurpose(memberID, purpose); err != nil {
		return "", fmt.Errorf("invalidate old OTPs: %w", err)
	}

	code, err := generateOTPCode()
	if err != nil {
		return "", fmt.Errorf("generate OTP code: %w", err)
	}

	env, err := s.env()
	if err != nil {
		return "", err
	}
	codeHash := hashOTP(env, code)
	now := time.Now()
	otp := &models.OTP{
		MemberID:    memberID,
		Purpose:     purpose,
		CodeHash:    codeHash,
		Attempts:    0,
		MaxAttempts: otpMaxAttempts,
		ExpiresAt:   now.Add(otpTTL),
	}
	if err := s.otpRepo.Create(otp); err != nil {
		return "", fmt.Errorf("create OTP: %w", err)
	}
	return code, nil
}

// Verify checks the provided code against the latest pending OTP for the given
// member and purpose. Uses ClaimAttempt for an atomic increment that prevents
// concurrent callers from bypassing the max-attempts limit. Returns sentinel
// errors ErrOTPExpired, ErrOTPInvalid, ErrOTPMaxAttempts, or ErrOTPAlreadyUsed.
func (s *OTPService) Verify(memberID uint, purpose, code string) error {
	env, err := s.env()
	if err != nil {
		return err
	}
	otp, err := s.otpRepo.GetLatestByMemberAndPurpose(memberID, purpose)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOTPInvalid
		}
		return fmt.Errorf("fetch OTP: %w", err)
	}

	// Atomic claim: increment attempts only if OTP is still valid (not used,
	// not expired, attempts < max). If rowsAffected == 0 the OTP has already
	// been exhausted/expired/used — re-read to return the right sentinel.
	rowsAffected, err := s.otpRepo.ClaimAttempt(otp.ID)
	if err != nil {
		return fmt.Errorf("claim attempt: %w", err)
	}
	if rowsAffected == 0 {
		// Re-fetch to determine which sentinel applies.
		refreshed, fetchErr := s.otpRepo.GetByID(otp.ID)
		if fetchErr != nil || refreshed == nil {
			return ErrOTPNotFound
		}
		if refreshed.UsedAt != nil {
			return ErrOTPAlreadyUsed
		}
		if time.Now().After(refreshed.ExpiresAt) {
			return ErrOTPExpired
		}
		return ErrOTPMaxAttempts
	}

	// Constant-time comparison on the hash.
	inputHash := hashOTP(env, code)
	if !constantTimeEqualOTP(inputHash, otp.CodeHash) {
		return ErrOTPInvalid
	}

	// Success — mark as used.
	if err := s.otpRepo.MarkUsed(otp.ID); err != nil {
		return fmt.Errorf("mark OTP used: %w", err)
	}
	return nil
}

// Invalidate marks all pending OTPs for the given member+purpose as used.
func (s *OTPService) Invalidate(memberID uint, purpose string) error {
	return s.otpRepo.InvalidateByMemberAndPurpose(memberID, purpose)
}

// generateOTPCode returns a cryptographically random 6-digit decimal string.
func generateOTPCode() (string, error) {
	// Generate a random number in [0, 1000000).
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hashOTP returns the SHA-256 hex digest of the OTP code bound to its environment.
func hashOTP(env environment.Environment, code string) string {
	sum := sha256.Sum256([]byte("payminto/otp/" + string(env) + "/" + code))
	return hex.EncodeToString(sum[:])
}

// constantTimeEqualOTP compares two OTP hashes in constant time.
func constantTimeEqualOTP(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
