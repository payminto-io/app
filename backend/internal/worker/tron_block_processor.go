package worker

import (
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
)

// NewTronBlockProcessor builds a BlockchainProcessor pre-configured for Tron.
// Thin wrapper for clarity in cmd/server/main.go worker registration.
func NewTronBlockProcessor(
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
