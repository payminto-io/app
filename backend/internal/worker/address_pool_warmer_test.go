package worker

import (
	"context"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWarmerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(
		&models.Member{},
		&models.BlockchainFamily{},
		&models.Wallet{},
		&models.WalletXpub{},
		&models.WalletFunction{},
		&models.SecretsVault{},
		&models.SecretsVaultActivity{},
		&models.AddressPool{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func fullWarmerStack(t *testing.T) (*AddressPoolWarmer, *service.AddressPoolService, uint, *gorm.DB) {
	t.Helper()
	db := setupWarmerTestDB(t)

	// Vault + WalletService
	vault := service.NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db),
		repository.NewSecretsVaultActivityRepository(db),
	)
	if err := vault.Unlock("test-passphrase-12345", nil); err != nil {
		t.Fatal(err)
	}

	walletSvc := service.NewWalletService(
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

	poolSvc := service.NewAddressPoolService(
		repository.NewAddressPoolRepository(db),
		repository.NewWalletRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		walletSvc,
	)

	warmer := NewAddressPoolWarmer(
		repository.NewWalletRepository(db),
		poolSvc,
	).WithInterval(10 * time.Millisecond).WithMinSize(5)

	return warmer, poolSvc, wallet.ID, db
}

func TestAddressPoolWarmer_Name(t *testing.T) {
	warmer, _, _, _ := fullWarmerStack(t)
	if warmer.Name() != "address_pool_warmer" {
		t.Errorf("expected name 'address_pool_warmer', got %q", warmer.Name())
	}
}

func TestAddressPoolWarmer_TickOnce_FillsEmptyPool(t *testing.T) {
	warmer, poolSvc, walletID, _ := fullWarmerStack(t)

	before, _ := poolSvc.CountAvailable(walletID)
	if before != 0 {
		t.Fatalf("expected empty pool at start, got %d", before)
	}

	warmer.TickOnce(t.Context())

	after, _ := poolSvc.CountAvailable(walletID)
	if after != 5 {
		t.Errorf("expected 5 addresses after tick, got %d", after)
	}
}

func TestAddressPoolWarmer_TickOnce_NoopIfAlreadyFull(t *testing.T) {
	warmer, poolSvc, walletID, _ := fullWarmerStack(t)

	// Prime the pool to 5
	if _, err := poolSvc.GenerateBatch(walletID, 5); err != nil {
		t.Fatal(err)
	}
	before, _ := poolSvc.CountAvailable(walletID)
	if before != 5 {
		t.Fatalf("expected 5, got %d", before)
	}

	warmer.TickOnce(t.Context())

	after, _ := poolSvc.CountAvailable(walletID)
	if after != 5 {
		t.Errorf("expected pool unchanged (5), got %d", after)
	}
}

func TestAddressPoolWarmer_Start_StopsOnContextCancel(t *testing.T) {
	warmer, _, _, _ := fullWarmerStack(t)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- warmer.Start(ctx)
	}()

	// Let one tick land
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected clean shutdown, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("worker did not exit after context cancel")
	}
}

func TestAddressPoolWarmer_TickOnce_SkipsInactiveWallet(t *testing.T) {
	warmer, poolSvc, walletID, db := fullWarmerStack(t)

	// Mark the wallet inactive
	db.Model(&models.Wallet{}).Where("id = ?", walletID).Update("status", "inactive")

	warmer.TickOnce(t.Context())

	count, _ := poolSvc.CountAvailable(walletID)
	if count != 0 {
		t.Errorf("expected inactive wallet skipped, pool should be 0, got %d", count)
	}
}

func TestAddressPoolWarmer_TickOnce_MultipleWallets(t *testing.T) {
	warmer, poolSvc, wallet1ID, db := fullWarmerStack(t)

	// Create a second wallet for the same member (different family)
	family2 := &models.BlockchainFamily{Code: "btc", Name: "Bitcoin"}
	db.Create(family2)

	// Manually create a second active wallet (skipping the HD derive since
	// we don't need real addresses — the warmer will call the service)
	wallet2 := &models.Wallet{
		Name:               "BTC wallet",
		Kind:               "hd",
		Status:             "active",
		BlockchainFamilyID: family2.ID,
		MemberID:           1,
	}
	db.Create(wallet2)

	// Create a WalletXpub for wallet2 so DeriveNextAddress works
	db.Create(&models.WalletXpub{
		WalletID:   wallet2.ID,
		Path:       "m/84'/0'/0'",
		AccountIdx: 0,
		NextIdx:    0,
		Status:     "active",
	})

	// NOTE: wallet2 has no mnemonic in the vault, so DeriveNextAddress will
	// fail. TickOnce should log the failure and continue. wallet1 should
	// still get topped up.
	warmer.TickOnce(t.Context())

	count1, _ := poolSvc.CountAvailable(wallet1ID)
	if count1 != 5 {
		t.Errorf("expected wallet1 topped up to 5, got %d", count1)
	}

	count2, _ := poolSvc.CountAvailable(wallet2.ID)
	if count2 != 0 {
		t.Errorf("expected wallet2 to have 0 (derivation failed), got %d", count2)
	}
}
