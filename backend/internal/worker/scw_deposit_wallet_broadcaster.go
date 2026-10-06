package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/repository"
)

// SCWDepositWalletBroadcaster picks up AddressDeployment rows in 'pending'
// state and triggers a CREATE2 deploy for each one.
//
// STUB: The real on-chain broadcast (building the CREATE2 calldata, signing via
// SecretsVault, sending via ethclient) lands in Phase K when the full
// transaction-signing pipeline from SecretsVault is wired. For now, this
// worker logs the pending deployment and marks it 'deployed' so downstream
// code sees a terminal state without blocking.
//
// TODO(phase-k-signing): Replace the log + status update with:
//  1. SecretsVaultService.DecryptKey(hotwallet) → private key
//  2. Build CREATE2 factory calldata for the deployment address
//  3. ethclient.SendTransaction → receive txHash
//  4. Poll for tx confirmation
//  5. Call deploymentRepo.UpdateStatus(id, "deployed") with the txHash recorded
type SCWDepositWalletBroadcaster struct {
	deploymentRepo repository.AddressDeploymentRepository
	pollInterval   time.Duration
	batchSize      int
}

// NewSCWDepositWalletBroadcaster constructs the broadcaster. pollInterval
// controls how often it checks for pending deployments; batchSize caps how
// many it processes per tick to bound RPC traffic.
func NewSCWDepositWalletBroadcaster(
	deploymentRepo repository.AddressDeploymentRepository,
	pollInterval time.Duration,
	batchSize int,
) *SCWDepositWalletBroadcaster {
	if pollInterval <= 0 {
		pollInterval = 15 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 10
	}
	return &SCWDepositWalletBroadcaster{
		deploymentRepo: deploymentRepo,
		pollInterval:   pollInterval,
		batchSize:      batchSize,
	}
}

// Name implements Worker.
func (b *SCWDepositWalletBroadcaster) Name() string { return "scw_deposit_wallet_broadcaster" }

// Start implements Worker. It polls for pending AddressDeployment rows on
// pollInterval and processes them until ctx is cancelled.
func (b *SCWDepositWalletBroadcaster) Start(ctx context.Context) error {
	ticker := time.NewTicker(b.pollInterval)
	defer ticker.Stop()

	log.Printf("[SCWDepositWalletBroadcaster] started (interval=%s, batchSize=%d)", b.pollInterval, b.batchSize)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[SCWDepositWalletBroadcaster] ctx cancelled, exiting")
			return nil
		case <-ticker.C:
			b.processPending()
		}
	}
}

// processPending picks up AddressDeployment rows in 'pending' state and marks
// them 'deployed'. The real CREATE2 broadcast is wired in Phase K.
func (b *SCWDepositWalletBroadcaster) processPending() {
	deployments, err := b.deploymentRepo.ListByStatus("pending", repository.WithLimit(b.batchSize))
	if err != nil {
		log.Printf("[SCWDepositWalletBroadcaster] list pending: %v", err)
		return
	}
	if len(deployments) == 0 {
		return
	}

	log.Printf("[SCWDepositWalletBroadcaster] processing %d pending deployments", len(deployments))
	for _, d := range deployments {
		// TODO(phase-k-signing): Replace this block with real CREATE2 broadcast:
		//   key, err := secretsVaultSvc.DecryptKey(d.BlockchainID)
		//   txHash, err := ethAdapter.DeploySCW(ctx, key, d.Address)
		//   deploymentRepo.Update(&d{TxHash: &txHash, Status: "deploying"})
		//   — then poll separately for confirmation.
		//
		// For now: log and mark deployed so the system makes forward progress.
		log.Printf("[SCWDepositWalletBroadcaster] STUB: marking deployment id=%d address=%s as deployed (real broadcast pending Phase K)",
			d.ID, d.Address)

		if err := b.deploymentRepo.UpdateStatus(d.ID, "deployed"); err != nil {
			log.Printf("[SCWDepositWalletBroadcaster] update status deployment %d: %v", d.ID, err)
		}
	}
}
