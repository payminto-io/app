package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite database and auto-migrates all models
// needed for payment/deposit/deposit-address tests.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	// Migrate all required models (order matters for FK resolution in SQLite).
	if err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.Member{},
		&models.ExternalPlatform{},
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.PaymentRequest{},
		&models.Deposit{},
		&models.DepositAddress{},
	); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	return db
}

// seedPlatformAndMember creates minimal ExternalPlatform + Member rows and
// returns their IDs for use in payment tests.
func seedPlatformAndMember(t *testing.T, db *gorm.DB) (platformID, memberID uint) {
	t.Helper()

	platform := models.ExternalPlatform{
		Name:            "Test Platform",
		LogoPath:        "",
		BrandColor:      "#000000",
		Website:         "https://example.com",
		SuccessEndpoint: "https://example.com/success",
	}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatalf("seed ExternalPlatform: %v", err)
	}

	member := models.Member{
		Name:       "Test Member",
		CustomerID: "cust-001",
		MemberType: "customer",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("seed Member: %v", err)
	}

	return platform.ID, member.ID
}

// seedBlockchainCurrency creates the minimal chain of records needed for a
// BlockchainCurrency and returns the currency's ID.
func seedBlockchainCurrency(t *testing.T, db *gorm.DB) uint {
	t.Helper()

	family := models.BlockchainFamily{Name: "EVM", Code: "evm"}
	if err := db.Create(&family).Error; err != nil {
		t.Fatalf("seed BlockchainFamily: %v", err)
	}

	chain := models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum",
		BlockchainFamilyID: family.ID,
	}
	if err := db.Create(&chain).Error; err != nil {
		t.Fatalf("seed Blockchain: %v", err)
	}

	currency := models.Currency{
		Name: "Ether",
		Code: "ETH",
		Type: "crypto",
	}
	if err := db.Create(&currency).Error; err != nil {
		t.Fatalf("seed Currency: %v", err)
	}

	bc := models.BlockchainCurrency{
		CurrencyCode:   "ETH",
		BlockchainCode: "ETH",
		CurrencyID:     currency.ID,
		BlockchainID:   chain.ID,
	}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatalf("seed BlockchainCurrency: %v", err)
	}

	return bc.ID
}

// ---------------------------------------------------------------------------
// PaymentRequest tests
// ---------------------------------------------------------------------------

func TestPaymentRepo_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	platformID, memberID := seedPlatformAndMember(t, db)
	repo := NewPaymentRepository(db)

	pr := &models.PaymentRequest{
		ReferenceID:        "ref-001",
		AmountInUSD:        decimal.NewFromFloat(100.00),
		State:              models.PaymentStateOpen,
		MemberID:           memberID,
		ExternalPlatformID: platformID,
	}

	if err := repo.Create(pr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pr.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByID(pr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ReferenceID != "ref-001" {
		t.Errorf("expected ReferenceID 'ref-001', got %q", got.ReferenceID)
	}
}

func TestPaymentRepo_GetByReferenceID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewPaymentRepository(db)

	_, err := repo.GetByReferenceID("does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestPaymentRepo_ListByPlatform_OrdersNewest(t *testing.T) {
	db := openTestDB(t)
	platformID, memberID := seedPlatformAndMember(t, db)
	repo := NewPaymentRepository(db)

	// Insert three payments with intentionally spread-out creation times.
	for i, ref := range []string{"ref-A", "ref-B", "ref-C"} {
		pr := &models.PaymentRequest{
			ReferenceID:        ref,
			AmountInUSD:        decimal.NewFromFloat(float64(i+1) * 10),
			State:              models.PaymentStateOpen,
			MemberID:           memberID,
			ExternalPlatformID: platformID,
		}
		// Manually set CreatedAt so ordering is deterministic.
		pr.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		if err := db.Create(pr).Error; err != nil {
			t.Fatalf("seed payment %s: %v", ref, err)
		}
	}

	list, err := repo.ListByPlatform(platformID)
	if err != nil {
		t.Fatalf("ListByPlatform: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 results, got %d", len(list))
	}
	// Newest first: ref-C > ref-B > ref-A
	if list[0].ReferenceID != "ref-C" {
		t.Errorf("expected newest first (ref-C), got %q", list[0].ReferenceID)
	}
	if list[2].ReferenceID != "ref-A" {
		t.Errorf("expected oldest last (ref-A), got %q", list[2].ReferenceID)
	}
}

func TestPaymentRepo_ExpireStale(t *testing.T) {
	db := openTestDB(t)
	platformID, memberID := seedPlatformAndMember(t, db)
	repo := NewPaymentRepository(db)

	expiredAt := time.Now().Add(-2 * time.Hour)
	pr := &models.PaymentRequest{
		ReferenceID:        "ref-expired",
		AmountInUSD:        decimal.NewFromFloat(50.00),
		State:              models.PaymentStateOpen,
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		ExpiresAt:          &expiredAt,
	}
	if err := repo.Create(pr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	cutoff := time.Now().Add(-1 * time.Hour)
	affected, err := repo.ExpireStale(cutoff)
	if err != nil {
		t.Fatalf("ExpireStale: %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}

	updated, err := repo.GetByID(pr.ID)
	if err != nil {
		t.Fatalf("GetByID after expire: %v", err)
	}
	if updated.State != models.PaymentStateCancelled {
		t.Errorf("expected state CANCELLED, got %q", updated.State)
	}
}

// ---------------------------------------------------------------------------
// Deposit tests
// ---------------------------------------------------------------------------

func TestDepositRepo_CreateAndGetByTxID(t *testing.T) {
	db := openTestDB(t)
	_, memberID := seedPlatformAndMember(t, db)
	bcID := seedBlockchainCurrency(t, db)
	repo := NewDepositRepository(db)

	d := &models.Deposit{
		TxID:                 "0xabcdef1234",
		Amount:               decimal.NewFromFloat(0.5),
		Status:               models.DepositStatusPending,
		ToAddress:            "0xRecipient",
		BlockchainCurrencyID: bcID,
		MemberID:             memberID,
	}
	if err := repo.Create(d); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	deposits, err := repo.GetByTxID("0xabcdef1234")
	if err != nil {
		t.Fatalf("GetByTxID: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deposits))
	}
	if deposits[0].ToAddress != "0xRecipient" {
		t.Errorf("unexpected ToAddress: %q", deposits[0].ToAddress)
	}
}

func TestDepositRepo_UpdateConfirmations(t *testing.T) {
	db := openTestDB(t)
	_, memberID := seedPlatformAndMember(t, db)
	bcID := seedBlockchainCurrency(t, db)
	repo := NewDepositRepository(db)

	d := &models.Deposit{
		TxID:                 "0xdeadbeef",
		Amount:               decimal.NewFromFloat(1.0),
		Status:               models.DepositStatusConfirming,
		ToAddress:            "0xRecipient2",
		BlockchainCurrencyID: bcID,
		MemberID:             memberID,
		Confirmations:        0,
	}
	if err := repo.Create(d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateConfirmations(d.ID, 6); err != nil {
		t.Fatalf("UpdateConfirmations: %v", err)
	}

	got, err := repo.GetByID(d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Confirmations != 6 {
		t.Errorf("expected 6 confirmations, got %d", got.Confirmations)
	}
}

// ---------------------------------------------------------------------------
// FinalizeFromConfirmedDeposits tests (C8)
// ---------------------------------------------------------------------------

func TestPaymentRepo_FinalizeFromConfirmedDeposits_Filled(t *testing.T) {
	db := openTestDB(t)
	platformID, memberID := seedPlatformAndMember(t, db)
	bcID := seedBlockchainCurrency(t, db)
	repo := NewPaymentRepository(db)
	depositRepo := NewDepositRepository(db)

	payment := &models.PaymentRequest{
		ReferenceID:        "ref-finalize-1",
		AmountInUSD:        decimal.NewFromFloat(50),
		State:              models.PaymentStateOpen,
		MemberID:           memberID,
		ExternalPlatformID: platformID,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	if err := depositRepo.Create(&models.Deposit{
		TxID:                 "0xabc",
		Amount:               decimal.NewFromFloat(50),
		Status:               models.DepositStatusConfirmed,
		PaymentRequestID:     &payment.ID,
		ToAddress:            "0xcafe",
		BlockchainCurrencyID: bcID,
		MemberID:             memberID,
	}); err != nil {
		t.Fatalf("seed deposit: %v", err)
	}

	state, total, changed, err := repo.FinalizeFromConfirmedDeposits(payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if state != models.PaymentStateFilled {
		t.Errorf("expected FILLED, got %s", state)
	}
	if !total.Equal(decimal.NewFromFloat(50)) {
		t.Errorf("expected total 50, got %s", total)
	}

	var refreshed models.PaymentRequest
	db.First(&refreshed, payment.ID)
	if refreshed.State != models.PaymentStateFilled {
		t.Errorf("payment row not updated in DB: got %s", refreshed.State)
	}
}

func TestPaymentRepo_FinalizeFromConfirmedDeposits_AlreadyFilled_NoChange(t *testing.T) {
	db := openTestDB(t)
	platformID, memberID := seedPlatformAndMember(t, db)
	repo := NewPaymentRepository(db)

	payment := &models.PaymentRequest{
		ReferenceID:        "ref-already-filled",
		AmountInUSD:        decimal.NewFromFloat(50),
		State:              models.PaymentStateFilled,
		MemberID:           memberID,
		ExternalPlatformID: platformID,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("seed payment: %v", err)
	}

	_, _, changed, err := repo.FinalizeFromConfirmedDeposits(payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false for already-filled payment")
	}
}

// ---------------------------------------------------------------------------
// ConfirmIfPending tests (C7)
// ---------------------------------------------------------------------------

func TestDepositRepo_ConfirmIfPending_OnlyAdvancesPending(t *testing.T) {
	db := openTestDB(t)
	_, memberID := seedPlatformAndMember(t, db)
	bcID := seedBlockchainCurrency(t, db)
	repo := NewDepositRepository(db)

	// Deposit already confirmed — ConfirmIfPending must not update it.
	d := &models.Deposit{
		TxID:                  "0xconfirmed",
		Amount:                decimal.NewFromFloat(1),
		Status:                models.DepositStatusConfirmed,
		ToAddress:             "0xAddr",
		BlockchainCurrencyID:  bcID,
		MemberID:              memberID,
		RequiredConfirmations: 3,
		Confirmations:         3,
	}
	if err := repo.Create(d); err != nil {
		t.Fatalf("seed deposit: %v", err)
	}

	rows, err := repo.ConfirmIfPending(d.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Errorf("expected 0 rows affected for already-confirmed deposit, got %d", rows)
	}
}

// ---------------------------------------------------------------------------
// ListByStatusAndMember isolation tests (C6)
// ---------------------------------------------------------------------------

func TestDepositRepo_ListByStatusAndMember_Isolation(t *testing.T) {
	db := openTestDB(t)
	_, memberID := seedPlatformAndMember(t, db)
	bcID := seedBlockchainCurrency(t, db)
	repo := NewDepositRepository(db)

	// A second member.
	other := models.Member{Name: "Other Merchant", CustomerID: "cust-other", MemberType: "customer"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other member: %v", err)
	}

	// 2 confirmed deposits for memberID, 1 for other.
	for i, addr := range []string{"0xA", "0xB"} {
		_ = i
		if err := repo.Create(&models.Deposit{
			TxID: addr, Amount: decimal.NewFromFloat(1),
			Status: models.DepositStatusConfirmed, ToAddress: addr,
			MemberID: memberID, BlockchainCurrencyID: bcID,
		}); err != nil {
			t.Fatalf("seed deposit: %v", err)
		}
	}
	if err := repo.Create(&models.Deposit{
		TxID: "0xC", Amount: decimal.NewFromFloat(1),
		Status: models.DepositStatusConfirmed, ToAddress: "0xC",
		MemberID: other.ID, BlockchainCurrencyID: bcID,
	}); err != nil {
		t.Fatalf("seed other deposit: %v", err)
	}

	list, err := repo.ListByStatusAndMember(models.DepositStatusConfirmed, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 deposits for member %d, got %d", memberID, len(list))
	}
	for _, d := range list {
		if d.MemberID != memberID {
			t.Errorf("result includes deposit for wrong member %d", d.MemberID)
		}
	}
}

// ---------------------------------------------------------------------------
// DepositAddress tests
// ---------------------------------------------------------------------------

func TestDepositAddressRepo_CreateAndGetByAddress(t *testing.T) {
	db := openTestDB(t)
	_, memberID := seedPlatformAndMember(t, db)
	bcID := seedBlockchainCurrency(t, db)
	repo := NewDepositAddressRepository(db)

	da := &models.DepositAddress{
		Address:              "0xDepositAddr1",
		BlockchainCurrencyID: bcID,
		MemberID:             memberID,
	}
	if err := repo.Create(da); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if da.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByAddress("0xDepositAddr1", bcID)
	if err != nil {
		t.Fatalf("GetByAddress: %v", err)
	}
	if got.ID != da.ID {
		t.Errorf("expected ID %d, got %d", da.ID, got.ID)
	}
	if got.BlockchainCurrency == nil {
		t.Error("expected BlockchainCurrency to be preloaded")
	}
}
