package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

// SLIP-0010 ed25519 test vector 1.
func TestDeriveSLIP10Ed25519_SpecVector(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	cases := []struct {
		path []uint32
		priv string
	}{
		{nil, "2b4be7f19ee27bbf30c667b642d5f4aa69fd169872f8fc3059c08ebae2eb19e7"},
		{[]uint32{HardenedKey(0)}, "68e0fe46dfb67e368c75379acec591dad19df3cde26e63b93a8e704f1dade7a3"},
		{[]uint32{HardenedKey(0), HardenedKey(1)}, "b1d0bad404bf35da785a64ca1ac54b2617211d2777696fbffaf208f746ae84f2"},
	}
	for _, c := range cases {
		got, err := DeriveSLIP10Ed25519(seed, c.path)
		if err != nil {
			t.Fatalf("path %v: %v", c.path, err)
		}
		if hex.EncodeToString(got) != c.priv {
			t.Fatalf("path %v: key %x, want %s", c.path, got, c.priv)
		}
	}
}

func TestDeriveSLIP10Ed25519_RejectsNonHardened(t *testing.T) {
	if _, err := DeriveSLIP10Ed25519([]byte("seed"), []uint32{1}); err == nil {
		t.Fatal("expected an error for a non-hardened ed25519 child")
	}
}

// Vectors produced by `solana-keygen pubkey "prompt://?full-path=m/44'/501'/n'/0'"` for the
// BIP-39 "abandon ... about" mnemonic with an empty passphrase.
func TestDeriveSolanaAddress_MatchesSolanaCLI(t *testing.T) {
	seed, err := SeedFromMnemonic("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[uint32]string{
		0: "HAgk14JpMQLgt6rVgv7cBQFJWFto5Dqxi472uT3DKpqk",
		7: "9h1cLBiraaUqM1CdJTaVaew1oQtgQUW24FZ8YdnLLgJY",
	}
	for idx, want := range cases {
		addr, priv, err := DeriveSolanaAddress(seed, idx)
		if err != nil {
			t.Fatalf("index %d: %v", idx, err)
		}
		if addr != want {
			t.Fatalf("index %d: address %s, want %s", idx, addr, want)
		}
		if len(priv) != ed25519.PrivateKeySize {
			t.Fatalf("index %d: private key length %d", idx, len(priv))
		}
		msg := []byte("hello")
		if !ed25519.Verify(ed25519.PrivateKey(priv).Public().(ed25519.PublicKey), msg, ed25519.Sign(ed25519.PrivateKey(priv), msg)) {
			t.Fatalf("index %d: key does not sign", idx)
		}
	}
}
