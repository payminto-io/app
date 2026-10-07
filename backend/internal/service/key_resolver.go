package service

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/repository"
)

// KeyResolver derives the private key for a previously-generated deposit/hot
// address by reproducing the exact HD derivation used when the address was
// created: vault mnemonic → BIP-39 seed → BIP-44/84 path at the address's
// stored PathIndex. It is the single integration point for "give me the signing
// key for this address", used by the withdrawal and sweep paths.
//
// Security: the mnemonic and seed live only transiently in memory and the seed
// is zeroed after use. The vault must be unlocked.
type KeyResolver struct {
	addressPoolRepo repository.AddressPoolRepository
	walletRepo      repository.WalletRepository
	familyRepo      repository.BlockchainFamilyRepository
	vault           *SecretsVaultService
	// mainnet selects BTC network params (mainnet vs testnet3) for derivation.
	mainnet bool
}

// NewKeyResolver constructs a KeyResolver. mainnet must match the running
// network mode so BTC addresses derive on the correct network params.
func NewKeyResolver(
	addressPoolRepo repository.AddressPoolRepository,
	walletRepo repository.WalletRepository,
	familyRepo repository.BlockchainFamilyRepository,
	vault *SecretsVaultService,
	mainnet bool,
) *KeyResolver {
	return &KeyResolver{
		addressPoolRepo: addressPoolRepo,
		walletRepo:      walletRepo,
		familyRepo:      familyRepo,
		vault:           vault,
		mainnet:         mainnet,
	}
}

// PrivateKeyForAddress returns the raw private key bytes for a managed address
// plus the blockchain-family code ("ETH_Family"|"BTC_Family"|"TRX_Family").
//
// It verifies the re-derived address equals the requested one and errors
// otherwise — a mismatch means the mnemonic/index no longer reproduces this
// address (e.g. after an un-stashed mnemonic rotation) and signing must abort
// rather than risk signing with the wrong key.
func (r *KeyResolver) PrivateKeyForAddress(address string) (privKey []byte, familyCode string, err error) {
	if r.vault == nil || !r.vault.IsUnlocked() {
		return nil, "", errors.New("secrets vault is locked")
	}

	pool, err := r.addressPoolRepo.GetByAddress(address)
	if err != nil {
		return nil, "", fmt.Errorf("address not found in pool: %w", err)
	}
	wallet, err := r.walletRepo.GetByID(pool.WalletID)
	if err != nil {
		return nil, "", fmt.Errorf("wallet %d: %w", pool.WalletID, err)
	}
	family, err := r.familyRepo.GetByID(wallet.BlockchainFamilyID)
	if err != nil {
		return nil, "", fmt.Errorf("family %d: %w", wallet.BlockchainFamilyID, err)
	}

	// Guard against silent uint→uint32 truncation: BIP-44 path indices must fit
	// in uint32. A truncated index would derive a DIFFERENT key for the same
	// stored address — caught later by the address-match check, but reject early.
	if pool.PathIndex > math.MaxUint32 {
		return nil, "", fmt.Errorf("path index %d exceeds BIP-44 max (uint32)", pool.PathIndex)
	}

	mnemonic, err := r.vault.GetKey(mnemonicLabel(wallet.MemberID, family.Code), &wallet.MemberID)
	if err != nil {
		return nil, "", fmt.Errorf("fetch mnemonic: %w", err)
	}
	seed, err := crypto.SeedFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, "", fmt.Errorf("seed: %w", err)
	}
	defer func() {
		for i := range seed {
			seed[i] = 0
		}
	}()

	var derivedAddr string
	switch family.Code {
	case "ETH_Family":
		derivedAddr, privKey, err = crypto.DeriveEthAddress(seed, 0, uint32(pool.PathIndex))
	case "TRX_Family":
		derivedAddr, privKey, err = crypto.DeriveTronAddress(seed, 0, uint32(pool.PathIndex))
	case "BTC_Family":
		net := &chaincfg.TestNet3Params
		if r.mainnet {
			net = &chaincfg.MainNetParams
		}
		derivedAddr, privKey, err = crypto.DeriveBtcAddress(seed, 0, uint32(pool.PathIndex), net)
	case "SOL_Family", "sol", "SOL":
		derivedAddr, privKey, err = crypto.DeriveSolanaAddress(seed, uint32(pool.PathIndex))
	default:
		return nil, "", fmt.Errorf("unsupported family %q", family.Code)
	}
	if err != nil {
		return nil, "", fmt.Errorf("derive key: %w", err)
	}

	if !addressesEqual(family.Code, derivedAddr, address) {
		return nil, "", fmt.Errorf("derived address %s does not match requested %s (key mismatch)", derivedAddr, address)
	}
	return privKey, family.Code, nil
}

// addressesEqual compares two addresses with chain-appropriate semantics. EVM
// addresses are hex and case-insensitive (checksum is presentational), so they
// compare case-insensitively. Bitcoin (bech32/base58) and Tron (base58check)
// addresses are case-sensitive and must match exactly.
func addressesEqual(familyCode, a, b string) bool {
	if familyCode == "ETH_Family" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
