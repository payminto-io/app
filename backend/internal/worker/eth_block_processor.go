package worker

import (
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
)

// NewEthereumBlockProcessor builds a BlockchainProcessor pre-configured for
// Ethereum. Thin wrapper for clarity in cmd/server/main.go worker registration.
func NewEthereumBlockProcessor(
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
