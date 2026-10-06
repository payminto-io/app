package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAddressPoolTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.BlockchainFamily{},
		&models.Wallet{},
		&models.WalletXpub{},
		&models.WalletFunction{},
		&models.SecretsVault{},
		&models.SecretsVaultActivity{},
		&models.AddressPool{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newPoolServiceForTest(t *testing.T, db *gorm.DB) (*AddressPoolService, uint) {
	t.Helper()

	// Set up vault + WalletService
	vault := NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db),
		repository.NewSecretsVaultActivityRepository(db),
	)
	if err := vault.Unlock("test-passphrase-12345", nil); err != nil {
		t.Fatalf("unlock vault: %v", err)
	}

	walletSvc := NewWalletService(
		repository.NewWalletRepository(db),
		repository.NewWalletXpubRepository(db),
		repository.NewWalletFunctionRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		vault,
	)

	family := &models.BlockchainFamily{Code: "evm", Name: "EVM"}
	db.Create(family)

	wallet, err := walletSvc.CreateHDWallet(1, family)
	if err != nil {
		t.Fatal(err)
	}

	poolSvc := NewAddressPoolService(
		repository.NewAddressPoolRepository(db),
		repository.NewWalletRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		walletSvc,
	)
	return poolSvc, wallet.ID
}

func TestAddressPoolService_GenerateBatch_Creates5(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, walletID := newPoolServiceForTest(t, db)

	n, err := svc.GenerateBatch(walletID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("expected 5 created, got %d", n)
	}

	count, _ := svc.CountAvailable(walletID)
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestAddressPoolService_GenerateBatch_InvalidCount(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, walletID := newPoolServiceForTest(t, db)

	if _, err := svc.GenerateBatch(walletID, 0); err == nil {
		t.Error("expected error for count=0")
	}
	if _, err := svc.GenerateBatch(walletID, -1); err == nil {
		t.Error("expected error for negative count")
	}
}

func TestAddressPoolService_GetAvailable_Sequential(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, walletID := newPoolServiceForTest(t, db)

	_, _ = svc.GenerateBatch(walletID, 3)

	p1, err := svc.GetAvailable(walletID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkUsed(p1.ID); err != nil {
		t.Fatal(err)
	}

	p2, err := svc.GetAvailable(walletID)
	if err != nil {
		t.Fatal(err)
	}
	if p2.ID == p1.ID {
		t.Error("expected different pool row after MarkUsed")
	}
}

func TestAddressPoolService_EnsurePoolSize_TopsUp(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, walletID := newPoolServiceForTest(t, db)

	_, _ = svc.GenerateBatch(walletID, 2)

	added, err := svc.EnsurePoolSize(walletID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if added != 8 {
		t.Errorf("expected 8 added to reach 10, got %d", added)
	}

	count, _ := svc.CountAvailable(walletID)
	if count != 10 {
		t.Errorf("expected count 10, got %d", count)
	}
}

func TestAddressPoolService_EnsurePoolSize_NoOp(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, walletID := newPoolServiceForTest(t, db)

	_, _ = svc.GenerateBatch(walletID, 20)

	added, err := svc.EnsurePoolSize(walletID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if added != 0 {
		t.Errorf("expected no-op when pool already large, got added=%d", added)
	}
}

func TestAddressPoolService_GenerateBatch_WalletNotFound(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, _ := newPoolServiceForTest(t, db)

	if _, err := svc.GenerateBatch(9999, 3); err == nil {
		t.Error("expected error for missing wallet")
	}
}

func TestChainCodeForFamily(t *testing.T) {
	cases := map[string]string{
		"evm": "ETH",
		"btc": "BTC",
		"trx": "TRX",
	}
	for family, want := range cases {
		got, err := chainCodeForFamily(family)
		if err != nil {
			t.Errorf("unexpected error for %s: %v", family, err)
		}
		if got != want {
			t.Errorf("family %s: want %s got %s", family, want, got)
		}
	}

	if _, err := chainCodeForFamily("unknown"); err == nil {
		t.Error("expected error for unknown family")
	}
}

func TestAddressPoolService_GetByAddress(t *testing.T) {
	db := setupAddressPoolTestDB(t)
	svc, walletID := newPoolServiceForTest(t, db)

	_, _ = svc.GenerateBatch(walletID, 1)

	pool, _ := svc.GetAvailable(walletID)
	got, err := svc.GetByAddress(pool.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != pool.ID {
		t.Errorf("expected same pool, got different IDs")
	}
}
