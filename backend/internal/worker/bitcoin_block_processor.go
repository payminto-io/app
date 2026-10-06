package worker

import (
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
)

// NewBitcoinBlockProcessor builds a BlockchainProcessor pre-configured for
// Bitcoin. Thin wrapper for clarity in cmd/server/main.go worker registration.
func NewBitcoinBlockProcessor(
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
