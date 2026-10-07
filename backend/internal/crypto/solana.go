package crypto

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/base58"
)

// SolanaCoinType is the SLIP-0044 coin type for Solana.
const SolanaCoinType = 501

// slip10MasterKey derives the ed25519 master key and chain code per SLIP-0010.
func slip10MasterKey(seed []byte) (key, chainCode []byte) {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	sum := mac.Sum(nil)
	return sum[:32], sum[32:]
}

// slip10ChildKey derives a hardened child; ed25519 in SLIP-0010 only defines hardened children.
func slip10ChildKey(key, chainCode []byte, index uint32) (childKey, childChain []byte, err error) {
	if index < hardenedOffset {
		return nil, nil, fmt.Errorf("slip10: ed25519 child %d must be hardened", index)
	}
	data := make([]byte, 0, 1+32+4)
	data = append(data, 0x00)
	data = append(data, key...)
	data = binary.BigEndian.AppendUint32(data, index)
	mac := hmac.New(sha512.New, chainCode)
	mac.Write(data)
	sum := mac.Sum(nil)
	return sum[:32], sum[32:], nil
}

const hardenedOffset = uint32(0x80000000)

// DeriveSLIP10Ed25519 walks a fully hardened SLIP-0010 path and returns the 32-byte ed25519 seed at the leaf.
func DeriveSLIP10Ed25519(seed []byte, path []uint32) ([]byte, error) {
	if len(seed) == 0 {
		return nil, errors.New("slip10: empty seed")
	}
	key, chain := slip10MasterKey(seed)
	for _, idx := range path {
		var err error
		key, chain, err = slip10ChildKey(key, chain, idx)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

// DeriveSolanaAddress returns the base58 public key and the 64-byte ed25519 private key at
// m/44'/501'/addressIndex'/0', the path Phantom and the Solana CLI use for account addressIndex.
func DeriveSolanaAddress(seed []byte, addressIndex uint32) (address string, privateKey []byte, err error) {
	path := []uint32{
		HardenedKey(44),
		HardenedKey(SolanaCoinType),
		HardenedKey(addressIndex),
		HardenedKey(0),
	}
	leaf, err := DeriveSLIP10Ed25519(seed, path)
	if err != nil {
		return "", nil, err
	}
	priv := ed25519.NewKeyFromSeed(leaf)
	for i := range leaf {
		leaf[i] = 0
	}
	pub := priv.Public().(ed25519.PublicKey)
	return base58.Encode(pub), []byte(priv), nil
}
