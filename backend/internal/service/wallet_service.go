package service

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// WalletService manages HD wallets for merchants. Wraps SecretsVault for
// mnemonic storage, WalletRepository for the wallet row, and
// WalletXpubRepository for derivation path tracking.
//
// All address derivation flows through DeriveNextAddress which:
//  1. Fetches the encrypted mnemonic from the vault
//  2. Seeds from the mnemonic
//  3. Derives the address at the next path index
//  4. Atomically increments the xpub's next_idx
//  5. Returns the address to the caller
//  6. Zeroes the in-memory seed (relies on Go GC)
//
// The plaintext mnemonic is never logged or persisted outside the vault.
type WalletService struct {
	walletRepo           repository.WalletRepository
	walletXpubRepo       repository.WalletXpubRepository
	walletFunctionRepo   repository.WalletFunctionRepository
	blockchainFamilyRepo repository.BlockchainFamilyRepository
	vault                *SecretsVaultService
}

func NewWalletService(
	walletRepo repository.WalletRepository,
	walletXpubRepo repository.WalletXpubRepository,
	walletFunctionRepo repository.WalletFunctionRepository,
	blockchainFamilyRepo repository.BlockchainFamilyRepository,
	vault *SecretsVaultService,
) *WalletService {
	return &WalletService{
		walletRepo:           walletRepo,
		walletXpubRepo:       walletXpubRepo,
		walletFunctionRepo:   walletFunctionRepo,
		blockchainFamilyRepo: blockchainFamilyRepo,
		vault:                vault,
	}
}

// CreateHDWallet generates a fresh 256-bit BIP-39 mnemonic, stores it in the
// secrets vault, and creates a Wallet row with an initial xpub for the
// given blockchain family. Returns the new wallet.
func (s *WalletService) CreateHDWallet(memberID uint, family *models.BlockchainFamily) (*models.Wallet, error) {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return nil, errors.New("secrets vault is locked")
	}
	if family == nil {
		return nil, errors.New("blockchain family is nil")
	}

	// 1. Generate a new mnemonic
	mnemonic, err := crypto.NewMnemonic(256)
	if err != nil {
		s.logFunc(0, &memberID, "create_hd_wallet", false, "mnemonic generation failed")
		return nil, fmt.Errorf("generate mnemonic: %w", err)
	}

	// 2. Create the wallet row first so we have an ID for the vault label
	wallet := &models.Wallet{
		Name:               fmt.Sprintf("HD Wallet (%s)", family.Code),
		Kind:               "hd",
		Status:             "active",
		BlockchainFamilyID: family.ID,
		MemberID:           memberID,
	}
	if err := s.walletRepo.Create(wallet); err != nil {
		s.logFunc(0, &memberID, "create_hd_wallet", false, err.Error())
		return nil, fmt.Errorf("create wallet: %w", err)
	}

	// 3. Store the mnemonic in the vault
	label := mnemonicLabel(memberID, family.Code)
	if err := s.vault.StoreKey(label, mnemonic, models.SecretTypeMnemonic, &memberID); err != nil {
		// Roll back the wallet row on vault failure
		_ = s.walletRepo.Delete(wallet.ID)
		s.logFunc(wallet.ID, &memberID, "create_hd_wallet", false, "vault store failed")
		return nil, fmt.Errorf("store mnemonic: %w", err)
	}

	// 4. Derive the first xpub
	seed, err := crypto.SeedFromMnemonic(mnemonic, "")
	if err != nil {
		s.logFunc(wallet.ID, &memberID, "create_hd_wallet", false, "seed derive failed")
		return nil, fmt.Errorf("seed: %w", err)
	}

	path := hdPathForFamily(family.Code)
	xpub := &models.WalletXpub{
		WalletID:   wallet.ID,
		Xpub:       "", // we don't expose the xpub publicly for now
		Path:       path,
		AccountIdx: 0,
		NextIdx:    0,
		Status:     "active",
	}
	if err := s.walletXpubRepo.Create(xpub); err != nil {
		s.logFunc(wallet.ID, &memberID, "create_hd_wallet", false, "xpub create failed")
		return nil, fmt.Errorf("create xpub: %w", err)
	}

	// Zero the in-memory seed — Go will GC, but being explicit
	for i := range seed {
		seed[i] = 0
	}

	s.logFunc(wallet.ID, &memberID, "create_hd_wallet", true, fmt.Sprintf("family=%s", family.Code))
	return wallet, nil
}

// GetByMember returns the wallet for a member on the given blockchain family.
func (s *WalletService) GetByMember(memberID, blockchainFamilyID uint) (*models.Wallet, error) {
	return s.walletRepo.GetByMemberAndFamily(memberID, blockchainFamilyID)
}

// EnsureXpubs verifies the wallet has at least one active xpub. Creates one
// at account 0 if missing. Idempotent.
func (s *WalletService) EnsureXpubs(walletID uint) error {
	xpubs, err := s.walletXpubRepo.ListByWallet(walletID)
	if err != nil {
		return err
	}
	if len(xpubs) > 0 {
		return nil
	}

	wallet, err := s.walletRepo.GetByID(walletID)
	if err != nil {
		return err
	}

	// Fetch family code for path
	family, err := s.blockchainFamilyRepo.GetByID(wallet.BlockchainFamilyID)
	if err != nil {
		return err
	}

	return s.walletXpubRepo.Create(&models.WalletXpub{
		WalletID:   walletID,
		Xpub:       "",
		Path:       hdPathForFamily(family.Code),
		AccountIdx: 0,
		NextIdx:    0,
		Status:     "active",
	})
}

// DeriveNextAddress returns a newly-derived address for the wallet and
// atomically advances the xpub's next_idx so subsequent calls never return
// the same address.
//
// The chainCode argument ("ETH", "BTC", "TRX", "BASE", "POLYGON") picks the
// derivation method. Callers should validate chainCode belongs to the
// wallet's family.
func (s *WalletService) DeriveNextAddress(memberID, walletID uint, chainCode string) (address string, pathIndex uint, err error) {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return "", 0, errors.New("secrets vault is locked")
	}

	wallet, err := s.walletRepo.GetByID(walletID)
	if err != nil {
		return "", 0, err
	}
	if wallet.MemberID != memberID {
		return "", 0, errors.New("wallet does not belong to member")
	}

	family, err := s.blockchainFamilyRepo.GetByID(wallet.BlockchainFamilyID)
	if err != nil {
		return "", 0, err
	}

	xpub, err := s.walletXpubRepo.GetByWalletAndAccount(walletID, 0)
	if err != nil {
		return "", 0, fmt.Errorf("xpub not found (call EnsureXpubs first): %w", err)
	}

	// Fetch the mnemonic from the vault
	label := mnemonicLabel(memberID, family.Code)
	mnemonic, err := s.vault.GetKey(label, &memberID)
	if err != nil {
		s.logFunc(walletID, &memberID, "derive_address", false, "mnemonic fetch failed")
		return "", 0, fmt.Errorf("fetch mnemonic: %w", err)
	}

	seed, err := crypto.SeedFromMnemonic(mnemonic, "")
	if err != nil {
		return "", 0, fmt.Errorf("seed: %w", err)
	}
	defer func() {
		for i := range seed {
			seed[i] = 0
		}
	}()

	// Atomically claim the next derivation index before touching the seed.
	// If derivation fails after this point we burn one path index — that is
	// acceptable; handing out duplicate addresses is not.
	next, err := s.walletXpubRepo.ClaimNextIdx(xpub.ID)
	if err != nil {
		s.logFunc(walletID, &memberID, "derive_address", false, "claim idx failed")
		return "", 0, err
	}

	switch chainCode {
	case "ETH", "BASE", "POLYGON":
		addr, _, derr := crypto.DeriveEthAddress(seed, 0, uint32(next))
		if derr != nil {
			s.logFunc(walletID, &memberID, "derive_address", false, derr.Error())
			return "", 0, derr
		}
		address = addr
	case "BTC":
		addr, _, derr := crypto.DeriveBtcAddress(seed, 0, uint32(next), &chaincfg.MainNetParams)
		if derr != nil {
			s.logFunc(walletID, &memberID, "derive_address", false, derr.Error())
			return "", 0, derr
		}
		address = addr
	case "BTC_TESTNET":
		addr, _, derr := crypto.DeriveBtcAddress(seed, 0, uint32(next), &chaincfg.TestNet3Params)
		if derr != nil {
			s.logFunc(walletID, &memberID, "derive_address", false, derr.Error())
			return "", 0, derr
		}
		address = addr
	case "TRX":
		addr, _, derr := crypto.DeriveTronAddress(seed, 0, uint32(next))
		if derr != nil {
			s.logFunc(walletID, &memberID, "derive_address", false, derr.Error())
			return "", 0, derr
		}
		address = addr
	case "SOLANA":
		addr, _, derr := crypto.DeriveSolanaAddress(seed, uint32(next))
		if derr != nil {
			s.logFunc(walletID, &memberID, "derive_address", false, derr.Error())
			return "", 0, derr
		}
		address = addr
	default:
		return "", 0, fmt.Errorf("unsupported chain code %q", chainCode)
	}

	s.logFunc(walletID, &memberID, "derive_address", true, fmt.Sprintf("chain=%s idx=%d", chainCode, next))
	return address, next, nil
}

// RotateMnemonic generates a new mnemonic and stores it under the wallet's
// label. The old mnemonic is preserved under a rotation-stamped label so
// legacy addresses can still be signed. Xpubs are NOT touched — new
// derivations use the new mnemonic going forward but old addresses remain
// valid for deposits.
//
// In production this would also bump an internal rotation counter so the
// derivation path encodes which mnemonic produced a given address.
func (s *WalletService) RotateMnemonic(memberID, walletID uint) error {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return errors.New("secrets vault is locked")
	}

	wallet, err := s.walletRepo.GetByID(walletID)
	if err != nil {
		return err
	}
	if wallet.MemberID != memberID {
		return errors.New("wallet does not belong to member")
	}

	family, err := s.blockchainFamilyRepo.GetByID(wallet.BlockchainFamilyID)
	if err != nil {
		return err
	}

	oldLabel := mnemonicLabel(memberID, family.Code)
	oldMnemonic, err := s.vault.GetKey(oldLabel, &memberID)
	if err != nil {
		return fmt.Errorf("fetch old mnemonic: %w", err)
	}

	// Stash the old mnemonic under a rotation label
	stashLabel := fmt.Sprintf("%s.rotated.%d", oldLabel, wallet.UpdatedAt.Unix())
	if err := s.vault.StoreKey(stashLabel, oldMnemonic, models.SecretTypeMnemonic, &memberID); err != nil {
		return fmt.Errorf("stash old mnemonic: %w", err)
	}

	newMnemonic, err := crypto.NewMnemonic(256)
	if err != nil {
		return err
	}
	if err := s.vault.StoreKey(oldLabel, newMnemonic, models.SecretTypeMnemonic, &memberID); err != nil {
		return fmt.Errorf("store new mnemonic: %w", err)
	}

	s.logFunc(walletID, &memberID, "rotate_mnemonic", true, fmt.Sprintf("stashed=%s", stashLabel))
	return nil
}

func (s *WalletService) logFunc(walletID uint, memberID *uint, action string, successful bool, desc string) {
	if s.walletFunctionRepo == nil {
		return
	}
	var d *string
	if desc != "" {
		d = &desc
	}
	_ = s.walletFunctionRepo.Create(&models.WalletFunction{
		WalletID:    walletID,
		MemberID:    memberID,
		Action:      action,
		Description: d,
		Successful:  successful,
	})
}

func mnemonicLabel(memberID uint, familyCode string) string {
	return fmt.Sprintf("hd_wallet.%d.%s.mnemonic", memberID, familyCode)
}

// hdPathForFamily returns the canonical BIP-44/84 derivation path template
// for a blockchain family. The last two indices (change + address) are
// filled in at derivation time.
func hdPathForFamily(familyCode string) string {
	switch familyCode {
	case "evm", "EVM":
		return "m/44'/60'/0'"
	case "btc", "BTC":
		return "m/84'/0'/0'"
	case "trx", "TRX":
		return "m/44'/195'/0'"
	case "sol", "SOL", "SOL_Family":
		return "m/44'/501'/{index}'/0'"
	default:
		return "m/44'/60'/0'"
	}
}
