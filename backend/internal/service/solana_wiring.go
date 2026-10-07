package service

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/modules"
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

// wireSolana registers the module's adapter and builds the watcher and, when the module carries a
// sweep identity, the sweeper. The services stay here because they compose service-layer parts.
func (r *ServiceRegistry) wireSolana(cfg *config.Config) error {
	m, err := modules.WireSolana(modules.Deps{DB: r.db, Config: cfg})
	if errors.Is(err, modules.ErrSolanaNotSeeded) {
		return nil
	}
	if err != nil {
		return err
	}
	r.solanaModule = m
	if err := r.adapterRegistry.Register(m.Adapter); err != nil {
		return err
	}
	log.Printf("[registry] registered SOLANA adapter (%s, %d tokens validated)", m.Cluster, len(m.Tokens))
	r.depositAddressService.WithSolanaDepositAccounts(r.solanaDepositAccountRepo, r.db, m.LateWindow)

	// The watcher posts the payment journal only for payments the switch does not own (switchOwnsPayment).
	r.solanaDepositService = NewSolanaDepositService(
		r.db, m.Client, m.Chain, r.solanaDepositAccountRepo, r.depositRepo, r.depositService,
		r.missedDepositRepo, r.blockchainCurrencyRepo, r.ledgerService, nil,
		SolanaDepositConfig{LateWindow: m.LateWindow},
	)
	if m.FeePayer == nil {
		log.Printf("[registry] solana sweeps disabled (set SOLANA_HOT_WALLET_ADDRESS and SOLANA_FEE_PAYER_KEY)")
		return nil
	}
	r.solanaSweepService = NewSolanaSweepService(
		r.db, m.Client, m.Chain, r.depositRepo, r.solanaDepositAccountRepo, r.blockchainCurrencyRepo, r.missedDepositRepo,
		r.sweepRepo, r.sweepTxRepo, r.sweepService, r.sweepTransactionService, r.keyResolver, *m.FeePayer, m.HotWallet,
		r.journal,
		SolanaSweepConfig{
			BatchSize: cfg.Solana.SweepBatchSize, ComputeUnitLimit: cfg.Solana.ComputeUnitLimit,
			PriorityFeeMicroLamports: cfg.Solana.PriorityFeeMicroLamports, CloseAccounts: cfg.Solana.CloseDepositAccounts,
		},
	)
	log.Printf("[registry] solana sweeps enabled (hot wallet %s, fee payer %s, late window %s)", m.HotWallet, m.FeePayer.PublicKey(), time.Duration(m.LateWindow))
	return nil
}
