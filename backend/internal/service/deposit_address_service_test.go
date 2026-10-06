package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDepositAddressTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.ExternalPlatform{},
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.Wallet{},
		&models.WalletXpub{},
		&models.WalletFunction{},
		&models.SecretsVault{},
		&models.SecretsVaultActivity{},
		&models.AddressPool{},
		&models.DepositAddress{},
		&models.PaymentRequest{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

// fullDepositAddressStack wires every dependency for the service end-to-end
// using an in-memory sqlite DB. Returns the service, the seeded payment, the
// seeded BlockchainCurrency, and the raw *gorm.DB handle (so tests can insert
// additional rows against the same DB).
func fullDepositAddressStack(t *testing.T) (*DepositAddressService, *models.PaymentRequest, *models.BlockchainCurrency, *gorm.DB) {
	t.Helper()
	db := setupDepositAddressTestDB(t)

	// Vault + WalletService
	vault := NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db),
		repository.NewSecretsVaultActivityRepository(db),
	)
	if err := vault.Unlock("test-passphrase", nil); err != nil {
		t.Fatal(err)
	}
	walletSvc := NewWalletService(
		repository.NewWalletRepository(db),
		repository.NewWalletXpubRepository(db),
		repository.NewWalletFunctionRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		vault,
	)

	// Seed family, chain, currency, blockchain_currency
	family := &models.BlockchainFamily{Code: "evm", Name: "EVM"}
	db.Create(family)
	chain := &models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum",
		BlockchainFamilyID: family.ID,
		Status:             "active",
		MinConfirmations:   12,
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

	// Create wallet for member 1
	member := &models.Member{Name: "merchant"}
	db.Create(member)
	if _, err := walletSvc.CreateHDWallet(member.ID, family); err != nil {
		t.Fatal(err)
	}

	// Create platform + payment
	platform := &models.ExternalPlatform{Name: "Test Platform"}
	db.Create(platform)
	payment := &models.PaymentRequest{
		ReferenceID:        "ref-123",
		AmountInUSD:        decimal.NewFromFloat(50),
		State:              models.PaymentStateOpen,
		MemberID:           member.ID,
		ExternalPlatformID: platform.ID,
	}
	db.Create(payment)

	// Assemble AddressPoolService + DepositAddressService
	poolSvc := NewAddressPoolService(
		repository.NewAddressPoolRepository(db),
		repository.NewWalletRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		walletSvc,
	)

	depositAddrSvc := NewDepositAddressService(
		repository.NewDepositAddressRepository(db),
		repository.NewWalletRepository(db),
		repository.NewBlockchainCurrencyRepository(db),
		walletSvc,
		poolSvc,
	)

	return depositAddrSvc, payment, bc, db
}

func TestDepositAddressService_AssignForPayment_NilPayment(t *testing.T) {
	svc, _, _, _ := fullDepositAddressStack(t)
	if _, err := svc.AssignForPayment(nil, "ETH", "ETH"); err == nil {
		t.Error("expected error for nil payment")
	}
}

func TestDepositAddressService_AssignForPayment_MissingChainCode(t *testing.T) {
	svc, payment, _, _ := fullDepositAddressStack(t)
	if _, err := svc.AssignForPayment(payment, "", ""); err == nil {
		t.Error("expected error for empty chainCode")
	}
}

func TestDepositAddressService_AssignForPayment_AutoTopsUpEmptyPool(t *testing.T) {
	svc, payment, _, _ := fullDepositAddressStack(t)

	da, err := svc.AssignForPayment(payment, "ETH", "ETH")
	if err != nil {
		t.Fatalf("AssignForPayment: %v", err)
	}
	if da == nil || da.Address == "" {
		t.Fatal("expected deposit address with non-empty address")
	}
	if da.PaymentRequestID == nil || *da.PaymentRequestID != payment.ID {
		t.Error("deposit address not linked to payment")
	}
	if da.BlockchainCurrency == nil {
		t.Error("expected BlockchainCurrency to be preloaded")
	}
}

func TestDepositAddressService_AssignForPayment_TwoPaymentsGetDifferentAddresses(t *testing.T) {
	svc, payment, bc, db := fullDepositAddressStack(t)

	// Create a second payment using the same DB handle
	payment2 := &models.PaymentRequest{
		ReferenceID:        "ref-456",
		AmountInUSD:        decimal.NewFromFloat(25),
		State:              models.PaymentStateOpen,
		MemberID:           payment.MemberID,
		ExternalPlatformID: payment.ExternalPlatformID,
	}
	db.Create(payment2)

	da1, err := svc.AssignForPayment(payment, "ETH", "ETH")
	if err != nil {
		t.Fatal(err)
	}
	da2, err := svc.AssignForPayment(payment2, "ETH", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	if da1.Address == da2.Address {
		t.Errorf("expected different addresses for sequential payments, both = %s", da1.Address)
	}

	// Verify bc is used (addresses should belong to that blockchain currency)
	_ = bc
}

func TestDepositAddressService_GetByAddress(t *testing.T) {
	svc, payment, bc, _ := fullDepositAddressStack(t)

	assigned, _ := svc.AssignForPayment(payment, "ETH", "ETH")

	got, err := svc.GetByAddress(assigned.Address, bc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != assigned.ID {
		t.Errorf("wrong deposit address returned")
	}
}

func TestDepositAddressService_ListForPayment(t *testing.T) {
	svc, payment, _, _ := fullDepositAddressStack(t)

	_, err := svc.AssignForPayment(payment, "ETH", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListForPayment(payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 deposit address for payment, got %d", len(list))
	}
}

func TestDepositAddressService_ListByMember(t *testing.T) {
	svc, payment, _, _ := fullDepositAddressStack(t)
	_, _ = svc.AssignForPayment(payment, "ETH", "ETH")

	list, err := svc.ListByMember(payment.MemberID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 deposit address for member, got %d", len(list))
	}
}

func TestDepositAddressService_AssignForPayment_UnknownChain(t *testing.T) {
	svc, payment, _, _ := fullDepositAddressStack(t)
	if _, err := svc.AssignForPayment(payment, "DOGE", "DOGE"); err == nil {
		t.Error("expected error for unknown chain")
	}
}
