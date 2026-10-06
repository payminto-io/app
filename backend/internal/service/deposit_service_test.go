package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDepositServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(
		&models.Member{},
		&models.ExternalPlatform{},
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.Wallet{},
		&models.AddressPool{},
		&models.DepositAddress{},
		&models.PaymentRequest{},
		&models.Deposit{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func seedDepositFixtures(t *testing.T, db *gorm.DB) (blockchainCurrencyID uint, member *models.Member) {
	t.Helper()
	member = &models.Member{Name: "merchant"}
	db.Create(member)

	family := &models.BlockchainFamily{Code: "evm", Name: "EVM"}
	db.Create(family)

	chain := &models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum",
		BlockchainFamilyID: family.ID,
		Status:             "active",
		MinConfirmations:   3,
	}
	db.Create(chain)

	currency := &models.Currency{Code: "ETH", Name: "Ethereum", Type: "native"}
	db.Create(currency)

	bc := &models.BlockchainCurrency{
		BlockchainID:    chain.ID,
		CurrencyID:      currency.ID,
		BlockchainCode:  "ETH",
		CurrencyCode:    "ETH",
		WalletPrecision: 18,
	}
	db.Create(bc)
	return bc.ID, member
}

func newDepositServiceForTest(t *testing.T, db *gorm.DB) *DepositService {
	t.Helper()
	return NewDepositService(
		repository.NewDepositRepository(db),
		repository.NewDepositAddressRepository(db),
		repository.NewPaymentRepository(db),
		repository.NewBlockchainCurrencyRepository(db),
		repository.NewBlockchainRepository(db),
	)
}

func TestDepositService_RecordDeposit_AddressNotFound(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, _ := seedDepositFixtures(t, db)

	_, err := svc.RecordDeposit(t.Context(), blockchain.Transaction{
		TxHash:    "0xabc",
		ToAddress: "0xunknown",
		Amount:    decimal.NewFromFloat(1),
	}, bcID)
	if !errors.Is(err, ErrDepositAddressNotFound) {
		t.Errorf("expected ErrDepositAddressNotFound, got %v", err)
	}
}

func TestDepositService_RecordDeposit_Success(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, member := seedDepositFixtures(t, db)

	// Create a deposit address
	da := &models.DepositAddress{
		Address:              "0xcafe",
		BlockchainCurrencyID: bcID,
		MemberID:             member.ID,
	}
	db.Create(da)

	dep, err := svc.RecordDeposit(t.Context(), blockchain.Transaction{
		TxHash:      "0xabc",
		FromAddress: "0xfrom",
		ToAddress:   "0xcafe",
		Amount:      decimal.NewFromFloat(1.5),
		BlockNumber: 100,
	}, bcID)
	if err != nil {
		t.Fatal(err)
	}
	if dep.ID == 0 {
		t.Error("expected non-zero deposit ID")
	}
	if dep.Status != models.DepositStatusPending {
		t.Errorf("expected pending, got %s", dep.Status)
	}
	if dep.RequiredConfirmations != 3 {
		t.Errorf("expected 3 required confs (from chain), got %d", dep.RequiredConfirmations)
	}
	if dep.MemberID != member.ID {
		t.Errorf("expected member %d, got %d", member.ID, dep.MemberID)
	}
}

func TestDepositService_RecordDeposit_Idempotent(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, member := seedDepositFixtures(t, db)

	da := &models.DepositAddress{
		Address:              "0xcafe",
		BlockchainCurrencyID: bcID,
		MemberID:             member.ID,
	}
	db.Create(da)

	tx := blockchain.Transaction{
		TxHash:      "0xabc",
		ToAddress:   "0xcafe",
		Amount:      decimal.NewFromFloat(1),
		BlockNumber: 100,
	}

	first, _ := svc.RecordDeposit(t.Context(), tx, bcID)
	second, err := svc.RecordDeposit(t.Context(), tx, bcID)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Errorf("expected same deposit ID on duplicate, got %d vs %d", first.ID, second.ID)
	}
}

func TestDepositService_UpdateConfirmations_TransitionsStates(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, member := seedDepositFixtures(t, db)

	db.Create(&models.ExternalPlatform{Name: "Test"})
	payment := &models.PaymentRequest{
		ReferenceID:        "ref-1",
		AmountInUSD:        decimal.NewFromFloat(1.5),
		State:              models.PaymentStateOpen,
		MemberID:           member.ID,
		ExternalPlatformID: 1,
	}
	db.Create(payment)

	da := &models.DepositAddress{
		Address:              "0xcafe",
		BlockchainCurrencyID: bcID,
		MemberID:             member.ID,
		PaymentRequestID:     &payment.ID,
	}
	db.Create(da)

	dep, _ := svc.RecordDeposit(t.Context(), blockchain.Transaction{
		TxHash:      "0xabc",
		ToAddress:   "0xcafe",
		Amount:      decimal.NewFromFloat(1.5),
		BlockNumber: 100,
	}, bcID)

	// 1 confirmation → confirming
	if err := svc.UpdateConfirmations(dep.ID, 100); err != nil {
		t.Fatal(err)
	}
	var d1 models.Deposit
	db.First(&d1, dep.ID)
	if d1.Status != models.DepositStatusConfirming {
		t.Errorf("expected confirming after 1 conf, got %s", d1.Status)
	}

	// 3 confirmations → confirmed, payment FILLED
	if err := svc.UpdateConfirmations(dep.ID, 102); err != nil {
		t.Fatal(err)
	}
	var d3 models.Deposit
	db.First(&d3, dep.ID)
	if d3.Status != models.DepositStatusConfirmed {
		t.Errorf("expected confirmed after 3 confs, got %s", d3.Status)
	}

	var pay models.PaymentRequest
	db.First(&pay, payment.ID)
	if pay.State != models.PaymentStateFilled {
		t.Errorf("expected payment FILLED, got %s", pay.State)
	}
}

func TestDepositService_GetUnspentDeposits(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, member := seedDepositFixtures(t, db)

	db.Create(&models.Deposit{
		TxID:                 "0xabc",
		Amount:               decimal.NewFromFloat(1),
		Status:               models.DepositStatusConfirmed,
		ToAddress:            "0xcafe",
		MemberID:             member.ID,
		BlockchainCurrencyID: bcID,
	})
	db.Create(&models.Deposit{
		TxID:                 "0xdef",
		Amount:               decimal.NewFromFloat(0.5),
		Status:               models.DepositStatusPending,
		ToAddress:            "0xcafe",
		MemberID:             member.ID,
		BlockchainCurrencyID: bcID,
	})

	list, err := svc.GetUnspentDeposits(member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 confirmed deposit, got %d", len(list))
	}
}

// TestDepositService_GetUnspentDeposits_TenantIsolated verifies that confirmed
// deposits belonging to one merchant are never returned for another merchant.
func TestDepositService_GetUnspentDeposits_TenantIsolated(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, member := seedDepositFixtures(t, db)

	other := &models.Member{Name: "other"}
	db.Create(other)

	// 2 confirmed deposits for the original member, 1 for the other.
	db.Create(&models.Deposit{TxID: "0x1", Amount: decimal.NewFromFloat(1), Status: "confirmed", ToAddress: "0x1", MemberID: member.ID, BlockchainCurrencyID: bcID})
	db.Create(&models.Deposit{TxID: "0x2", Amount: decimal.NewFromFloat(1), Status: "confirmed", ToAddress: "0x2", MemberID: member.ID, BlockchainCurrencyID: bcID})
	db.Create(&models.Deposit{TxID: "0x3", Amount: decimal.NewFromFloat(1), Status: "confirmed", ToAddress: "0x3", MemberID: other.ID, BlockchainCurrencyID: bcID})

	list, err := svc.GetUnspentDeposits(member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 deposits, got %d", len(list))
	}
	for _, d := range list {
		if d.MemberID != member.ID {
			t.Errorf("leaked deposit from member %d", d.MemberID)
		}
	}
}

// TestDepositService_GetUnspentDeposits_ZeroMemberID verifies that memberID=0
// returns an empty slice rather than leaking all merchants' deposits.
func TestDepositService_GetUnspentDeposits_ZeroMemberID(t *testing.T) {
	db := setupDepositServiceDB(t)
	svc := newDepositServiceForTest(t, db)
	bcID, member := seedDepositFixtures(t, db)

	db.Create(&models.Deposit{TxID: "0x1", Amount: decimal.NewFromFloat(1), Status: "confirmed", ToAddress: "0x1", MemberID: member.ID, BlockchainCurrencyID: bcID})

	list, err := svc.GetUnspentDeposits(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected zero results for member_id=0, got %d", len(list))
	}
}
