package service

import (
	"crypto/ed25519"
	"testing"

	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/models"
)

func TestKeyResolver_Solana_ReturnsEd25519KeyForOwner(t *testing.T) {
	kr, db, vault := setupKeyResolver(t)
	const memberID = uint(7)
	const pathIndex = uint(3)
	family := models.BlockchainFamily{Code: "SOL_Family", Family: "SOL_Family", Name: "Solana"}
	db.Create(&family)
	wallet := models.Wallet{Name: "hd", Kind: "hd", MemberID: memberID, BlockchainFamilyID: family.ID}
	db.Create(&wallet)
	mnemonic, err := crypto.NewMnemonic(256)
	if err != nil {
		t.Fatal(err)
	}
	if err := vault.StoreKey(mnemonicLabel(memberID, "SOL_Family"), mnemonic, models.SecretTypeMnemonic, ptr(memberID)); err != nil {
		t.Fatal(err)
	}
	seed, _ := crypto.SeedFromMnemonic(mnemonic, "")
	address, want, err := crypto.DeriveSolanaAddress(seed, uint32(pathIndex))
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&models.AddressPool{Address: address, PathIndex: pathIndex, Status: "used", WalletID: wallet.ID, BlockchainFamilyID: family.ID})

	priv, code, err := kr.PrivateKeyForAddress(address)
	if err != nil {
		t.Fatal(err)
	}
	if code != "SOL_Family" || len(priv) != ed25519.PrivateKeySize || string(priv) != string(want) {
		t.Fatalf("resolved key wrong: family=%s len=%d", code, len(priv))
	}
	if _, _, err := kr.PrivateKeyForAddress("HAgk14JpMQLgt6rVgv7cBQFJWFto5Dqxi472uT3DKpqk"); err == nil {
		t.Fatal("an address outside the pool must not resolve")
	}
}
