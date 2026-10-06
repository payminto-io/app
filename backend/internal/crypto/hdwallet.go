package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	bip39 "github.com/tyler-smith/go-bip39"
)

// NewMnemonic returns a fresh BIP-39 mnemonic phrase with the given entropy
// size in bits. Use 256 (24 words) for production wallets. Use 128 (12 words)
// only for development / tests.
func NewMnemonic(entropyBits int) (string, error) {
	if entropyBits%32 != 0 || entropyBits < 128 || entropyBits > 256 {
		return "", fmt.Errorf("entropyBits must be one of 128, 160, 192, 224, 256")
	}
	entropy, err := bip39.NewEntropy(entropyBits)
	if err != nil {
		return "", fmt.Errorf("new entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("new mnemonic: %w", err)
	}
	return mnemonic, nil
}

// SeedFromMnemonic derives the 64-byte BIP-39 seed from a mnemonic phrase
// and an optional passphrase. Passphrase may be empty.
func SeedFromMnemonic(mnemonic, passphrase string) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("invalid mnemonic")
	}
	return bip39.NewSeed(mnemonic, passphrase), nil
}

// DerivePrivateKey walks a BIP-32 derivation path from the master seed and
// returns the extended private key at the leaf. The path is a slice of
// hardened/non-hardened indices: e.g. m/44'/60'/0'/0/0 is
// []uint32{44 + 0x80000000, 60 + 0x80000000, 0 + 0x80000000, 0, 0}.
//
// Use HardenedKey(n) to mark an index as hardened.
func DerivePrivateKey(seed []byte, path []uint32, network *chaincfg.Params) (*hdkeychain.ExtendedKey, error) {
	if network == nil {
		network = &chaincfg.MainNetParams
	}
	master, err := hdkeychain.NewMaster(seed, network)
	if err != nil {
		return nil, fmt.Errorf("new master: %w", err)
	}
	key := master
	for _, idx := range path {
		key, err = key.Derive(idx)
		if err != nil {
			return nil, fmt.Errorf("derive index %d: %w", idx, err)
		}
	}
	return key, nil
}

// HardenedKey returns the index | 2^31 which marks the index as hardened.
func HardenedKey(index uint32) uint32 {
	return hdkeychain.HardenedKeyStart + index
}

// ----- Ethereum derivation -----

// DeriveEthAddress returns the 0x-prefixed checksummed Ethereum address at
// path m/44'/60'/accountIndex'/0/addressIndex. Works for both mainnet and
// testnet (Ethereum uses the same derivation for all networks).
//
// Also returns the 32-byte private key so callers can sign transactions.
func DeriveEthAddress(seed []byte, accountIndex, addressIndex uint32) (address string, privateKey []byte, err error) {
	path := []uint32{
		HardenedKey(44),
		HardenedKey(60),
		HardenedKey(accountIndex),
		0,
		addressIndex,
	}
	key, err := DerivePrivateKey(seed, path, &chaincfg.MainNetParams)
	if err != nil {
		return "", nil, err
	}

	priv, err := key.ECPrivKey()
	if err != nil {
		return "", nil, fmt.Errorf("get ecdsa key: %w", err)
	}
	ecdsaPriv := priv.ToECDSA()
	addr := ethcrypto.PubkeyToAddress(ecdsaPriv.PublicKey)

	return common.Address(addr).Hex(), ethcrypto.FromECDSA(ecdsaPriv), nil
}

// ----- Bitcoin derivation -----

// DeriveBtcAddress returns a BIP-84 (native SegWit, bech32) address at
// path m/84'/coinType'/accountIndex'/0/addressIndex. coinType is 0 for
// mainnet and 1 for testnet. The network must match the coinType.
func DeriveBtcAddress(seed []byte, accountIndex, addressIndex uint32, network *chaincfg.Params) (address string, privateKey []byte, err error) {
	if network == nil {
		network = &chaincfg.MainNetParams
	}

	// BIP-84 coin type: 0 for mainnet, 1 for testnet
	coinType := uint32(0)
	if network.Name == "testnet3" || network.Name == "signet" || network.Name == "regtest" {
		coinType = 1
	}

	path := []uint32{
		HardenedKey(84),
		HardenedKey(coinType),
		HardenedKey(accountIndex),
		0,
		addressIndex,
	}
	key, err := DerivePrivateKey(seed, path, network)
	if err != nil {
		return "", nil, err
	}

	pubKey, err := key.ECPubKey()
	if err != nil {
		return "", nil, fmt.Errorf("get pubkey: %w", err)
	}

	// BIP-84: P2WPKH — hash160 of compressed pubkey, bech32 encoded
	compressed := pubKey.SerializeCompressed()
	hash160 := btcutil.Hash160(compressed)

	addr, err := btcutil.NewAddressWitnessPubKeyHash(hash160, network)
	if err != nil {
		return "", nil, fmt.Errorf("create witness address: %w", err)
	}

	priv, err := key.ECPrivKey()
	if err != nil {
		return "", nil, fmt.Errorf("get privkey: %w", err)
	}

	return addr.EncodeAddress(), priv.Serialize(), nil
}

// ----- Tron derivation -----

// DeriveTronAddress returns the Base58Check "T"-prefixed Tron address at
// path m/44'/195'/accountIndex'/0/addressIndex. Tron uses the same
// derivation for mainnet and Nile/Shasta testnets.
func DeriveTronAddress(seed []byte, accountIndex, addressIndex uint32) (address string, privateKey []byte, err error) {
	path := []uint32{
		HardenedKey(44),
		HardenedKey(195),
		HardenedKey(accountIndex),
		0,
		addressIndex,
	}
	key, err := DerivePrivateKey(seed, path, &chaincfg.MainNetParams)
	if err != nil {
		return "", nil, err
	}

	priv, err := key.ECPrivKey()
	if err != nil {
		return "", nil, fmt.Errorf("get privkey: %w", err)
	}
	ecdsaPriv := priv.ToECDSA()
	ethAddr := ethcrypto.PubkeyToAddress(ecdsaPriv.PublicKey)

	// Tron address = 0x41 || eth_address_bytes, then base58check encoded.
	// The eth address bytes are keccak256(pubkey)[12:] which is what go-ethereum returns.
	tronBytes := make([]byte, 21)
	tronBytes[0] = 0x41
	copy(tronBytes[1:], ethAddr.Bytes())

	checksum := doubleSha256(tronBytes)[:4]
	full := append(tronBytes, checksum...)
	tronAddr := base58.Encode(full)

	return tronAddr, ethcrypto.FromECDSA(ecdsaPriv), nil
}

// ----- Internal helpers -----

func doubleSha256(data []byte) []byte {
	first := sha256Sum(data)
	second := sha256Sum(first)
	return second
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// ExportEthPrivateKeyHex returns the private key bytes as a hex string (no 0x prefix).
// Exported for tests.
func ExportEthPrivateKeyHex(privBytes []byte) string {
	return hex.EncodeToString(privBytes)
}

// _ unused ecdsa import guard
var _ = ecdsa.PublicKey{}
