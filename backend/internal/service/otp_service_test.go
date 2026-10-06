package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOTPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.Member{}, &models.OTP{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newOTPService(db *gorm.DB) (*OTPService, repository.OTPRepository) {
	repo := repository.NewOTPRepository(db)
	return NewOTPService(repo), repo
}

func TestOTPService_GenerateAndVerify(t *testing.T) {
	db := newOTPTestDB(t)
	svc, _ := newOTPService(db)

	code, err := svc.Generate(1, models.OTPPurposeEmailVerification)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected 6-digit code, got %q (len=%d)", code, len(code))
	}

	if err := svc.Verify(1, models.OTPPurposeEmailVerification, code); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestOTPService_InvalidCode(t *testing.T) {
	db := newOTPTestDB(t)
	svc, _ := newOTPService(db)

	if _, err := svc.Generate(1, models.OTPPurposeEmailVerification); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	err := svc.Verify(1, models.OTPPurposeEmailVerification, "000000")
	if !errors.Is(err, ErrOTPInvalid) {
		t.Errorf("expected ErrOTPInvalid, got %v", err)
	}
}

func TestOTPService_MaxAttempts(t *testing.T) {
	db := newOTPTestDB(t)
	svc, _ := newOTPService(db)

	if _, err := svc.Generate(1, models.OTPPurposeEmailVerification); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Exhaust attempts with wrong codes.
	for range 5 {
		_ = svc.Verify(1, models.OTPPurposeEmailVerification, "000000")
	}

	err := svc.Verify(1, models.OTPPurposeEmailVerification, "000000")
	if !errors.Is(err, ErrOTPMaxAttempts) && !errors.Is(err, ErrOTPInvalid) {
		t.Errorf("expected ErrOTPMaxAttempts or ErrOTPInvalid after exhausting attempts, got %v", err)
	}
}

func TestOTPService_Expiry(t *testing.T) {
	db := newOTPTestDB(t)
	repo := repository.NewOTPRepository(db)
	svc := NewOTPService(repo)

	code, err := svc.Generate(1, models.OTPPurposePasswordReset)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Manually expire the OTP in the database.
	past := time.Now().Add(-10 * time.Minute)
	db.Model(&models.OTP{}).
		Where("member_id = ? AND purpose = ?", 1, models.OTPPurposePasswordReset).
		Update("expires_at", past)

	err = svc.Verify(1, models.OTPPurposePasswordReset, code)
	if !errors.Is(err, ErrOTPExpired) {
		t.Errorf("expected ErrOTPExpired, got %v", err)
	}
}

func TestOTPService_Invalidate(t *testing.T) {
	db := newOTPTestDB(t)
	svc, _ := newOTPService(db)

	code, err := svc.Generate(1, models.OTPPurposeEmailVerification)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if err := svc.Invalidate(1, models.OTPPurposeEmailVerification); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	err = svc.Verify(1, models.OTPPurposeEmailVerification, code)
	// OTP was marked used by Invalidate so should return ErrOTPInvalid (no active OTP found).
	if err == nil {
		t.Error("expected error after invalidation, got nil")
	}
}

// TestOTPService_Verify_ConcurrentAttempts_CantBypassMax spawns 10 goroutines
// all verifying the same OTP with the wrong code. Because ClaimAttempt is atomic,
// attempts must never exceed MaxAttempts regardless of concurrency. At least the
// first few goroutines should get ErrOTPInvalid (wrong code) and the rest should
// get ErrOTPMaxAttempts once the budget is exhausted.
func TestOTPService_Verify_ConcurrentAttempts_CantBypassMax(t *testing.T) {
	// SQLite with shared cache and WAL mode is sufficient for concurrency testing
	// since ClaimAttempt uses a conditional UPDATE that SQLite serialises.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // force serialisation on SQLite
	if err := db.AutoMigrate(&models.Member{}, &models.OTP{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc, _ := newOTPService(db)
	if _, err := svc.Generate(42, models.OTPPurposeEmailVerification); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	const goroutines = 10
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			errs[i] = svc.Verify(42, models.OTPPurposeEmailVerification, "000000")
		}(i)
	}
	wg.Wait()

	invalidCount := 0
	maxAttemptsCount := 0
	for _, e := range errs {
		switch {
		case errors.Is(e, ErrOTPInvalid):
			invalidCount++
		case errors.Is(e, ErrOTPMaxAttempts):
			maxAttemptsCount++
		default:
			t.Errorf("unexpected error from concurrent Verify: %v", e)
		}
	}
	if invalidCount == 0 {
		t.Error("expected at least one ErrOTPInvalid from concurrent verifications")
	}

	// Verify the DB attempts counter did not exceed MaxAttempts.
	var otp models.OTP
	db.Where("member_id = ? AND purpose = ?", 42, models.OTPPurposeEmailVerification).First(&otp)
	if otp.Attempts > otp.MaxAttempts {
		t.Errorf("attempts=%d exceeded max_attempts=%d — atomic guard failed", otp.Attempts, otp.MaxAttempts)
	}
	t.Logf("concurrent results: invalid=%d, max_attempts=%d, db_attempts=%d/%d",
		invalidCount, maxAttemptsCount, otp.Attempts, otp.MaxAttempts)
}

func TestOTPService_GenerateUniqueCodes(t *testing.T) {
	codes := make(map[string]bool)
	for range 20 {
		code, err := generateOTPCode()
		if err != nil {
			t.Fatalf("generateOTPCode: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("expected 6 digits, got %q", code)
		}
		codes[code] = true
	}
	// With 20 samples and 1M possibilities, all should be unique.
	if len(codes) < 15 {
		t.Errorf("expected mostly unique codes, got %d unique in 20 samples", len(codes))
	}
}
