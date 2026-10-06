package crypto

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestNewMnemonic_12Words(t *testing.T) {
	m, err := NewMnemonic(128)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(m)); n != 12 {
		t.Errorf("expected 12 words, got %d", n)
	}
}

func TestNewMnemonic_24Words(t *testing.T) {
	m, err := NewMnemonic(256)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(m)); n != 24 {
		t.Errorf("expected 24 words, got %d", n)
	}
}

func TestNewMnemonic_InvalidEntropy(t *testing.T) {
	if _, err := NewMnemonic(100); err == nil {
		t.Error("expected error for 100 bits")
	}
}

func TestSeedFromMnemonic_KnownVector(t *testing.T) {
	seed, err := SeedFromMnemonic(testMnemonic, "TREZOR")
	if err != nil {
		t.Fatal(err)
	}
	want := "c55257c360c07c72029aebc1b53c05ed0362ada38ead3e3e9efa3708e53495531f09a6987599d18264c1e1c92f2cf141630c7a3c4ab7c81b2f001698e7463b04"
	if got := hex.EncodeToString(seed); got != want {
		t.Errorf("seed = %s\nwant = %s", got, want)
	}
}

func TestSeedFromMnemonic_Invalid(t *testing.T) {
	if _, err := SeedFromMnemonic("not valid words here", ""); err == nil {
		t.Error("expected error")
	}
}

func TestDeriveEthAddress_Vector0(t *testing.T) {
	seed, _ := SeedFromMnemonic(testMnemonic, "")
	addr, privKey, err := DeriveEthAddress(seed, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
	if !strings.EqualFold(addr, want) {
		t.Errorf("addr = %s\nwant = %s", addr, want)
	}
	if len(privKey) != 32 {
		t.Errorf("expected 32-byte private key, got %d bytes", len(privKey))
	}
}

func TestDeriveEthAddress_Vector1(t *testing.T) {
	seed, _ := SeedFromMnemonic(testMnemonic, "")
	addr, _, err := DeriveEthAddress(seed, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	// different index should give different address
	addr0, _, _ := DeriveEthAddress(seed, 0, 0)
	if addr == addr0 {
		t.Error("different addresses expected for index 0 and 1")
	}
}

func TestDeriveBtcAddress_Mainnet(t *testing.T) {
	seed, _ := SeedFromMnemonic(testMnemonic, "")
	addr, privKey, err := DeriveBtcAddress(seed, 0, 0, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(addr, "bc1") {
		t.Errorf("expected bc1-prefixed address for mainnet, got %s", addr)
	}
	if len(privKey) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(privKey))
	}
}

func TestDeriveBtcAddress_Testnet(t *testing.T) {
	seed, _ := SeedFromMnemonic(testMnemonic, "")
	addr, _, err := DeriveBtcAddress(seed, 0, 0, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(addr, "tb1") {
		t.Errorf("expected tb1-prefixed address for testnet, got %s", addr)
	}
}

func TestDeriveTronAddress_Format(t *testing.T) {
	seed, _ := SeedFromMnemonic(testMnemonic, "")
	addr, privKey, err := DeriveTronAddress(seed, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(addr, "T") {
		t.Errorf("expected T-prefixed Tron address, got %s", addr)
	}
	if len(addr) != 34 {
		t.Errorf("expected 34-char Tron address, got %d chars: %s", len(addr), addr)
	}
	if len(privKey) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(privKey))
	}
}

func TestHardenedKey(t *testing.T) {
	if HardenedKey(0) != 0x80000000 {
		t.Error("hardened 0 should be 2^31")
	}
	if HardenedKey(44) != 0x80000000+44 {
		t.Error("hardened 44 should be 2^31 + 44")
	}
}
