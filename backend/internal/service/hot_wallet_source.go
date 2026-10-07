package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/repository"
)

// familyForChain maps a chain code to its blockchain-family code. EVM chains
// share one family (and one hot wallet) since they use the same key/address.
func familyForChain(chainCode string) (string, bool) {
	switch strings.ToUpper(chainCode) {
	case "ETH", "BASE", "POLYGON":
		return "ETH_Family", true
	case "BTC":
		return "BTC_Family", true
	case "TRX":
		return "TRX_Family", true
	case "SOLANA", "SOL":
		return "SOL_Family", true
	}
	return "", false
}

// HotWalletSource resolves the payout hot wallet for a merchant + chain: its
// public address and decrypted private key. Withdrawals are funded from this
// operator-managed hot wallet (registered via POST /wallets/hot), whose key is
// stored in the vault under "hot_wallet.{walletID}" — distinct from the derived
// deposit-address keys that KeyResolver handles.
type HotWalletSource struct {
	walletRepo repository.WalletRepository
	familyRepo repository.BlockchainFamilyRepository
	configSvc  *ConfigurationService
	vault      *SecretsVaultService
}

// NewHotWalletSource constructs a HotWalletSource.
func NewHotWalletSource(
	walletRepo repository.WalletRepository,
	familyRepo repository.BlockchainFamilyRepository,
	configSvc *ConfigurationService,
	vault *SecretsVaultService,
) *HotWalletSource {
	return &HotWalletSource{
		walletRepo: walletRepo,
		familyRepo: familyRepo,
		configSvc:  configSvc,
		vault:      vault,
	}
}

// Resolve returns the hot wallet address and raw private-key bytes for a member
// on the family that owns chainCode. Caller MUST zero the returned key after use.
func (h *HotWalletSource) Resolve(ctx context.Context, memberID uint, chainCode string) (address string, privKey []byte, err error) {
	if h.vault == nil || !h.vault.IsUnlocked() {
		return "", nil, errors.New("secrets vault is locked")
	}
	familyCode, ok := familyForChain(chainCode)
	if !ok {
		return "", nil, fmt.Errorf("unsupported chain %q", chainCode)
	}
	family, err := h.familyRepo.GetByCode(familyCode)
	if err != nil {
		return "", nil, fmt.Errorf("family %s: %w", familyCode, err)
	}

	wallets, err := h.walletRepo.ListByMember(memberID)
	if err != nil {
		return "", nil, fmt.Errorf("list wallets: %w", err)
	}
	var walletID uint
	for i := range wallets {
		w := &wallets[i]
		if w.Kind == "hot" && w.BlockchainFamilyID == family.ID && w.Status == "active" {
			walletID = w.ID
			break
		}
	}
	if walletID == 0 {
		return "", nil, fmt.Errorf("no active hot wallet for member %d on %s", memberID, familyCode)
	}

	addrCfg, err := h.configSvc.Get(ctx, fmt.Sprintf("hot_wallet_address.%d", walletID))
	if err != nil || addrCfg == nil || addrCfg.Value == "" {
		return "", nil, fmt.Errorf("hot wallet %d address not configured", walletID)
	}
	address = addrCfg.Value

	keyHex, err := h.vault.GetKey(fmt.Sprintf("hot_wallet.%d", walletID), &memberID)
	if err != nil {
		return "", nil, fmt.Errorf("fetch hot wallet key: %w", err)
	}
	privKey, err = hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(keyHex), "0x"))
	if err != nil {
		return "", nil, fmt.Errorf("decode hot wallet key: %w", err)
	}
	return address, privKey, nil
}
