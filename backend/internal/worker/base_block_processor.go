package worker

import (
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
)

// NewBaseBlockProcessor builds a BlockchainProcessor pre-configured for Base
// (Coinbase's L2, chain ID 8453). Thin wrapper for clarity in
// cmd/server/main.go worker registration.
func NewBaseBlockProcessor(
	chain *models.Blockchain,
	deps ProcessorDeps,
	adapter blockchain.ChainAdapter,
) *BlockchainProcessor {
	return NewBlockchainProcessor(
		chain,
		deps.BlockchainRepo,
		deps.DepositService,
		deps.DepositAddressRepo,
		deps.MissedDepositRepo,
		deps.BlockchainCurrencyRepo,
		adapter,
	)
}
