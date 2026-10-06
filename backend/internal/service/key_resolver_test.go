package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupKeyResolver(t *testing.T) (*KeyResolver, *gorm.DB, *SecretsVaultService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.SecretsVault{}, &models.SecretsVaultActivity{},
		&models.Wallet{}, &models.AddressPool{}, &models.BlockchainFamily{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	vault := NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db),
		repository.NewSecretsVaultActivityRepository(db),
	)
	if err := vault.Unlock("test-passphrase-1234567890", nil); err != nil {
		t.Fatalf("unlock vault: %v", err)
	}
	kr := NewKeyResolver(
		repository.NewAddressPoolRepository(db),
		repository.NewWalletRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		vault,
		false, // testnet
	)
	return kr, db, vault
}

func TestKeyResolver_ETH_ReturnsCorrectKey(t *testing.T) {
	kr, db, vault := setupKeyResolver(t)
	const memberID = uint(42)
	const pathIndex = uint(5)

	// Seed an ETH family + wallet + a known mnemonic in the vault.
	family := models.BlockchainFamily{Code: "ETH_Family", Family: "ETH_Family"}
	db.Create(&family)
	wallet := models.Wallet{Name: "hot", Kind: "hot", MemberID: memberID, BlockchainFamilyID: family.ID}
	db.Create(&wallet)

	mnemonic, err := crypto.NewMnemonic(256)
	if err != nil {
		t.Fatalf("mnemonic: %v", err)
	}
	if err := vault.StoreKey(mnemonicLabel(memberID, "ETH_Family"), mnemonic, models.SecretTypeMnemonic, ptr(memberID)); err != nil {
		t.Fatalf("store mnemonic: %v", err)
	}

	// Derive the address the pool would have stored at pathIndex.
	seed, _ := crypto.SeedFromMnemonic(mnemonic, "")
	wantAddr, wantPriv, err := crypto.DeriveEthAddress(seed, 0, uint32(pathIndex))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	db.Create(&models.AddressPool{
		Address:            wantAddr,
		PathIndex:          pathIndex,
		Status:             "used",
		WalletID:           wallet.ID,
		BlockchainFamilyID: family.ID,
	})

	gotPriv, fam, err := kr.PrivateKeyForAddress(wantAddr)
	if err != nil {
		t.Fatalf("PrivateKeyForAddress: %v", err)
	}
	if fam != "ETH_Family" {
		t.Errorf("family = %q, want ETH_Family", fam)
	}
	if string(gotPriv) != string(wantPriv) {
		t.Errorf("resolved private key does not match derivation")
	}
}

func TestKeyResolver_UnknownAddress_Errors(t *testing.T) {
	kr, _, _ := setupKeyResolver(t)
	if _, _, err := kr.PrivateKeyForAddress("0xdeadbeef00000000000000000000000000000000"); err == nil {
		t.Fatal("expected error for unknown address")
	}
}

func TestKeyResolver_LockedVault_Errors(t *testing.T) {
	kr, _, vault := setupKeyResolver(t)
	vault.Lock(nil)
	if _, _, err := kr.PrivateKeyForAddress("0xabc"); err == nil {
		t.Fatal("expected error when vault locked")
	}
}

func ptr[T any](v T) *T { return &v }
