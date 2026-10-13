package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/config"
)

// wireTempo builds the Tempo memo watcher when TEMPO_MERCHANT_ADDRESS is set and the TEMPO chain is
// seeded. Unset means Tempo is off and nothing else changes; a set but broken config fails boot.
func (r *ServiceRegistry) wireTempo(cfg *config.Config) error {
	tc := cfg.Tempo
	if tc.MerchantAddress == "" {
		return nil
	}
	if !common.IsHexAddress(tc.MerchantAddress) {
		return fmt.Errorf("TEMPO_MERCHANT_ADDRESS %q is not an address", tc.MerchantAddress)
	}
	chain, err := r.blockchainRepo.GetByCode(tempo.ChainCode)
	if err != nil {
		return fmt.Errorf("TEMPO_MERCHANT_ADDRESS is set but blockchain %s is not seeded: %w", tempo.ChainCode, err)
	}
	rows, err := r.blockchainCurrencyRepo.ListByBlockchainID(chain.ID)
	if err != nil {
		return fmt.Errorf("tempo currencies: %w", err)
	}
	tokens := TempoTokensFromCurrencies(rows)
	if len(tokens) == 0 {
		return fmt.Errorf("tempo: no deposit-enabled TIP-20 currency seeded for %s", tempo.ChainCode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := tempo.Dial(ctx, tc.RPCURL)
	if err != nil {
		return err
	}
	id, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("tempo chain id: %w", err)
	}
	if chain.ChainID != nil && id.Int64() != *chain.ChainID {
		return fmt.Errorf("tempo: RPC %s is chain %d, the %s row says %d", tc.RPCURL, id.Int64(), tempo.ChainCode, *chain.ChainID)
	}
	r.tempoMemoService = NewTempoMemoService(r.db, client, chain, tokens, r.depositService, r.ledgerService, nil, TempoMemoConfig{
		Merchant: common.HexToAddress(tc.MerchantAddress), MemberID: uint(max(tc.MemberID, 0)), StartBlock: uint64(max(tc.StartBlock, 0)),
	})
	log.Printf("[registry] tempo memo watcher wired (chain %d, merchant %s, %d tokens)", id.Int64(), tc.MerchantAddress, len(tokens))
	return nil
}

// TempoMemoService returns the Tempo memo watcher, or nil when Tempo is not configured.
func (r *ServiceRegistry) TempoMemoService() *TempoMemoService { return r.tempoMemoService }
