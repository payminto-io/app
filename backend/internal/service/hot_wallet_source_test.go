package service

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHotWalletSource(t *testing.T) (*HotWalletSource, *gorm.DB, *SecretsVaultService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.SecretsVault{}, &models.SecretsVaultActivity{},
		&models.Wallet{}, &models.BlockchainFamily{}, &models.Configuration{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	vault := NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db),
		repository.NewSecretsVaultActivityRepository(db),
	)
	if err := vault.Unlock("test-passphrase-1234567890", nil); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	src := NewHotWalletSource(
		repository.NewWalletRepository(db),
		repository.NewBlockchainFamilyRepository(db),
		NewConfigurationService(repository.NewConfigurationRepository(db)),
		vault,
	)
	return src, db, vault
}

func TestHotWalletSource_ResolvesKeyAndAddress(t *testing.T) {
	src, db, vault := setupHotWalletSource(t)
	const memberID = uint(7)
	const wantHex = "4c0883a69102937d6231471b5dbb6204fe512961708279f1f542a4f3e2f1f1f1"
	const wantAddr = "0xHotWalletAddr"

	fam := models.BlockchainFamily{Code: "ETH_Family", Family: "ETH_Family"}
	db.Create(&fam)
	w := models.Wallet{Name: "hot", Kind: "hot", Status: "active", MemberID: memberID, BlockchainFamilyID: fam.ID}
	db.Create(&w)
	if err := vault.StoreKey("hot_wallet."+itoa(w.ID), wantHex, models.SecretTypePrivateKey, ptr(memberID)); err != nil {
		t.Fatalf("store key: %v", err)
	}
	cfgSvc := NewConfigurationService(repository.NewConfigurationRepository(db))
	if err := cfgSvc.Set(context.Background(), "hot_wallet_address."+itoa(w.ID), wantAddr, "addr"); err != nil {
		t.Fatalf("set addr: %v", err)
	}

	addr, priv, err := src.Resolve(context.Background(), memberID, "ETH")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if addr != wantAddr {
		t.Errorf("address = %q, want %q", addr, wantAddr)
	}
	wantBytes, _ := hex.DecodeString(wantHex)
	if string(priv) != string(wantBytes) {
		t.Error("resolved private key does not match stored key")
	}
}

func TestHotWalletSource_NoHotWallet_Errors(t *testing.T) {
	src, _, _ := setupHotWalletSource(t)
	if _, _, err := src.Resolve(context.Background(), 99, "ETH"); err == nil {
		t.Fatal("expected error when no hot wallet exists")
	}
}

func TestHotWalletSource_UnsupportedChain_Errors(t *testing.T) {
	src, _, _ := setupHotWalletSource(t)
	if _, _, err := src.Resolve(context.Background(), 1, "DOGE"); err == nil {
		t.Fatal("expected error for unsupported chain")
	}
}

func TestFamilyForChain(t *testing.T) {
	cases := map[string]string{"ETH": "ETH_Family", "BASE": "ETH_Family", "POLYGON": "ETH_Family", "BTC": "BTC_Family", "TRX": "TRX_Family"}
	for chain, want := range cases {
		got, ok := familyForChain(chain)
		if !ok || got != want {
			t.Errorf("familyForChain(%q) = %q,%v want %q", chain, got, ok, want)
		}
	}
	if _, ok := familyForChain("DOGE"); ok {
		t.Error("DOGE should be unsupported")
	}
}

// itoa is a tiny uint→string helper for vault labels in tests.
func itoa(v uint) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
