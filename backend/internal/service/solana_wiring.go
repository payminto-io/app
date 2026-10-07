package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
)

// familyOfChain returns the family label the adapter switch expects. Seeds fill
// blockchain_families.code (evm, btc, trx, sol); blockchains.family is the legacy label.
func familyOfChain(chain *models.Blockchain) string {
	if chain.Family != "" {
		return chain.Family
	}
	if chain.BlockchainFamily != nil {
		if chain.BlockchainFamily.Family != "" {
			return chain.BlockchainFamily.Family
		}
		switch strings.ToLower(chain.BlockchainFamily.Code) {
		case "evm":
			return "ETH_Family"
		case "btc":
			return "BTC_Family"
		case "trx":
			return "TRX_Family"
		case "sol":
			return "SOL_Family"
		}
	}
	if chain.Code == solana.ChainCode {
		return "SOL_Family"
	}
	return ""
}

// solanaCluster derives the cluster from SOLANA_CLUSTER or the network mode.
func solanaCluster(cfg *config.Config) string {
	if c := strings.TrimSpace(cfg.Solana.Cluster); c != "" {
		return c
	}
	if cfg.Blockchain.NetworkType == "mainnet" {
		return solana.ClusterMainnet
	}
	return solana.ClusterDevnet
}

// wireSolana builds the watcher and, when a hot wallet and fee payer are configured, the sweeper.
func (r *ServiceRegistry) wireSolana(cfg *config.Config) error {
	adapter, err := r.adapterRegistry.Get(solana.ChainCode)
	if err != nil {
		return nil
	}
	sol, ok := adapter.(*solana.Adapter)
	if !ok {
		return nil
	}
	chain, err := r.blockchainRepo.GetByCode(solana.ChainCode)
	if err != nil {
		return fmt.Errorf("solana chain row: %w", err)
	}
	if !sol.IsMainnet() {
		r.applyDevnetUSDTMint(chain, cfg.Solana.DevnetUSDTMint)
	}
	r.solanaDepositService = NewSolanaDepositService(
		r.db, sol.Client(), chain, r.solanaDepositAccountRepo, r.depositRepo, r.depositService,
		r.missedDepositRepo, r.blockchainCurrencyRepo, r.ledgerService, nil, SolanaDepositConfig{},
	)

	if cfg.Solana.HotWalletAddress == "" || cfg.Solana.FeePayerKey == "" {
		log.Printf("[registry] solana sweeps disabled (set SOLANA_HOT_WALLET_ADDRESS and SOLANA_FEE_PAYER_KEY)")
		return nil
	}
	hot, err := solana.ParsePublicKey(cfg.Solana.HotWalletAddress)
	if err != nil {
		return fmt.Errorf("SOLANA_HOT_WALLET_ADDRESS: %w", err)
	}
	feePayer, err := solana.ParseKeypair(cfg.Solana.FeePayerKey)
	if err != nil {
		return fmt.Errorf("SOLANA_FEE_PAYER_KEY: %w", err)
	}
	r.solanaSweepService = NewSolanaSweepService(
		r.db, sol.Client(), chain, r.depositRepo, r.solanaDepositAccountRepo, r.blockchainCurrencyRepo,
		r.sweepRepo, r.sweepTxRepo, r.sweepService, r.sweepTransactionService, r.keyResolver, feePayer, hot,
		ledger.New(r.db), SolanaSweepConfig{
			BatchSize: cfg.Solana.SweepBatchSize, ComputeUnitLimit: cfg.Solana.ComputeUnitLimit,
			PriorityFeeMicroLamports: cfg.Solana.PriorityFeeMicroLamports, CloseAccounts: cfg.Solana.CloseDepositAccounts,
			Send: solana.SendOptions{Wait: 45 * time.Second},
		},
	)
	log.Printf("[registry] solana sweeps enabled (hot wallet %s, fee payer %s)", hot, feePayer.PublicKey())
	return nil
}

// applyDevnetUSDTMint fills the seeded devnet USDT row, which ships disabled with no mint because
// devnet has no official USDT. Logged either way so the state is never silent.
func (r *ServiceRegistry) applyDevnetUSDTMint(chain *models.Blockchain, mint string) {
	row, err := r.blockchainCurrencyRepo.GetByBlockchainCodeAndCurrencyCode(solana.ChainCode, "USDT")
	if err != nil {
		return
	}
	if mint == "" {
		if row.Address == "" {
			log.Printf("[registry] solana devnet USDT has no mint; set SOLANA_DEVNET_USDT_MINT to enable it")
		}
		return
	}
	if _, err := solana.ParsePublicKey(mint); err != nil {
		log.Printf("[registry] SOLANA_DEVNET_USDT_MINT ignored: %v", err)
		return
	}
	if row.Address == mint && row.DepositEnabled {
		return
	}
	row.Address = mint
	row.DepositEnabled = true
	row.Visible = true
	if err := r.blockchainCurrencyRepo.Update(row); err != nil {
		log.Printf("[registry] solana devnet USDT mint update: %v", err)
		return
	}
	log.Printf("[registry] solana devnet USDT mint set to %s on chain %s", mint, chain.Code)
}
