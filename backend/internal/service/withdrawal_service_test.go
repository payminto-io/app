package service

import (
	"sync"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWithdrawalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.OTP{},
		&models.EEEvent{},
		&models.Withdrawal{},
		&models.Withdraw{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.ExternalPlatform{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed a blockchain + currency + blockchain_currency so withdrawal tests
	// can resolve "ETH"/"USDT" via GetByBlockchainCodeAndCurrencyCode.
	bc := &models.Blockchain{Code: "ETH", Name: "Ethereum", BlockchainFamilyID: 1}
	db.Create(bc)
	cur := &models.Currency{Code: "USDT", Name: "Tether USD"}
	db.Create(cur)
	bcc := &models.BlockchainCurrency{
		BlockchainID:   bc.ID,
		CurrencyID:     cur.ID,
		BlockchainCode: "ETH",
		CurrencyCode:   "USDT",
	}
	db.Create(bcc)

	return db
}

func newWithdrawalService(db *gorm.DB) *WithdrawalService {
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	blockchainCcyRepo := repository.NewBlockchainCurrencyRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	eeRepo := repository.NewEEEventRepository(db)
	otpSvc := NewOTPService(otpRepo)
	emitterSvc := NewEventEmitterService(eeRepo)
	return NewWithdrawalService(withdrawalRepo, blockchainCcyRepo, otpSvc, emitterSvc)
}

func TestWithdrawalService_CreateLowAmount(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	// Amount below auto-approve threshold → no OTP.
	input := CreateWithdrawalInput{
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Amount:             decimal.NewFromInt(50),
		ToAddress:          "0xdeadbeef",
		MemberID:           1,
		ExternalPlatformID: 1,
	}

	w, prompt, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil withdrawal")
	}
	if w.State != models.WithdrawalStatePendingApproval {
		t.Errorf("state = %q, want %q", w.State, models.WithdrawalStatePendingApproval)
	}
	if prompt.OTPRequired {
		t.Error("expected OTPRequired=false for low amount")
	}
}

func TestWithdrawalService_CreateHighAmount_OTPRequired(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	// Amount above auto-approve threshold → OTP required.
	input := CreateWithdrawalInput{
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Amount:             decimal.NewFromInt(500),
		ToAddress:          "0xdeadbeef",
		MemberID:           1,
		ExternalPlatformID: 1,
	}

	w, prompt, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.State != models.WithdrawalStatePendingOTP {
		t.Errorf("state = %q, want %q", w.State, models.WithdrawalStatePendingOTP)
	}
	if !prompt.OTPRequired {
		t.Error("expected OTPRequired=true for high amount")
	}
}

func TestWithdrawalService_ValidationErrors(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	tests := []struct {
		name  string
		input CreateWithdrawalInput
	}{
		{
			name: "empty ToAddress",
			input: CreateWithdrawalInput{
				BlockchainCode: "ETH", CurrencyCode: "USDT",
				Amount: decimal.NewFromInt(10), MemberID: 1,
			},
		},
		{
			name: "zero amount",
			input: CreateWithdrawalInput{
				BlockchainCode: "ETH", CurrencyCode: "USDT",
				Amount: decimal.Zero, ToAddress: "0x1234", MemberID: 1,
			},
		},
		{
			name: "missing blockchain code",
			input: CreateWithdrawalInput{
				CurrencyCode: "USDT", Amount: decimal.NewFromInt(10),
				ToAddress: "0x1234", MemberID: 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.Create(tc.input)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestWithdrawalService_Approve(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	input := CreateWithdrawalInput{
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Amount:             decimal.NewFromInt(50), // below threshold
		ToAddress:          "0xdeadbeef",
		MemberID:           1,
		ExternalPlatformID: 1,
	}

	w, _, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Approve(w.ID, 99); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	updated, err := svc.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.State != models.WithdrawalStatePending {
		t.Errorf("state after approve = %q, want %q", updated.State, models.WithdrawalStatePending)
	}
	if updated.ApprovedByMemberID == nil || *updated.ApprovedByMemberID != 99 {
		t.Errorf("ApprovedByMemberID = %v, want 99", updated.ApprovedByMemberID)
	}
}

func TestWithdrawalService_Cancel(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	input := CreateWithdrawalInput{
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Amount:             decimal.NewFromInt(50),
		ToAddress:          "0xdeadbeef",
		MemberID:           1,
		ExternalPlatformID: 1,
	}

	w, _, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Cancel(w.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	updated, err := svc.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.State != models.WithdrawalStateCancelled {
		t.Errorf("state after cancel = %q, want %q", updated.State, models.WithdrawalStateCancelled)
	}
}

func TestWithdrawalService_ListByPlatform(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	for range 3 {
		input := CreateWithdrawalInput{
			BlockchainCode:     "ETH",
			CurrencyCode:       "USDT",
			Amount:             decimal.NewFromInt(10),
			ToAddress:          "0xdeadbeef",
			MemberID:           1,
			ExternalPlatformID: 5,
		}
		if _, _, err := svc.Create(input); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := svc.ListByPlatform(5)
	if err != nil {
		t.Fatalf("ListByPlatform: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 withdrawals, got %d", len(list))
	}
}

// TestWithdrawalService_Approve_Concurrent_OnlyOneWins spawns two goroutines that
// both race to approve the same withdrawal. Because Approve uses an atomic
// conditional UPDATE, exactly one must succeed and the other must return an error.
func TestWithdrawalService_Approve_Concurrent_OnlyOneWins(t *testing.T) {
	// Use a shared-cache SQLite DB with a single connection to force serialisation.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&models.Member{},
		&models.OTP{},
		&models.EEEvent{},
		&models.Withdrawal{},
		&models.Withdraw{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.ExternalPlatform{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed blockchain currency data for ETH/USDT lookup.
	bc := &models.Blockchain{Code: "ETH", Name: "Ethereum", BlockchainFamilyID: 1}
	db.Create(bc)
	cur := &models.Currency{Code: "USDT", Name: "Tether USD"}
	db.Create(cur)
	bcc := &models.BlockchainCurrency{
		BlockchainID:   bc.ID,
		CurrencyID:     cur.ID,
		BlockchainCode: "ETH",
		CurrencyCode:   "USDT",
	}
	db.Create(bcc)

	svc := newWithdrawalService(db)

	// Create a withdrawal in pending-approval state (amount < threshold).
	input := CreateWithdrawalInput{
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Amount:             decimal.NewFromInt(50),
		ToAddress:          "0xdeadbeef",
		MemberID:           1,
		ExternalPlatformID: 1,
	}
	w, _, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	errs := make([]error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for i := range 2 {
		go func(i int) {
			defer wg.Done()
			errs[i] = svc.Approve(w.ID, uint(100+i))
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, e := range errs {
		if e == nil {
			successCount++
		}
	}
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful Approve, got %d (errors: %v, %v)", successCount, errs[0], errs[1])
	}
}

func TestWithdrawalService_TenantIsolation(t *testing.T) {
	db := newWithdrawalTestDB(t)
	svc := newWithdrawalService(db)

	for platformID := uint(1); platformID <= 3; platformID++ {
		input := CreateWithdrawalInput{
			BlockchainCode:     "ETH",
			CurrencyCode:       "USDT",
			Amount:             decimal.NewFromInt(10),
			ToAddress:          "0xabc",
			MemberID:           platformID,
			ExternalPlatformID: platformID,
		}
		if _, _, err := svc.Create(input); err != nil {
			t.Fatalf("Create platform %d: %v", platformID, err)
		}
	}

	// Platform 2 should only see its own withdrawal.
	list, err := svc.ListByPlatform(2)
	if err != nil {
		t.Fatalf("ListByPlatform: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 withdrawal for platform 2, got %d", len(list))
	}
}
