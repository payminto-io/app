package service

import (
	"context"
	"log"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// defaultRequiredConfirmations is used when a blockchain row has no explicit
// MinConfirmations (0) — a conservative fallback.
const defaultRequiredConfirmations = 12

// confirmationChecker reports how many confirmations a tx has on a chain.
// Implemented by AdapterConfirmationChecker over the adapter registry; an
// interface so EVMSweepConfirmer is unit-testable with a fake.
type confirmationChecker interface {
	Confirmations(ctx context.Context, chainCode, txHash string) (uint64, error)
}

// AdapterConfirmationChecker adapts the chain-adapter registry to
// confirmationChecker. Works for any chain (GetConfirmations is on the base
// adapter interface), not just EVM.
type AdapterConfirmationChecker struct {
	adapters adapterSource
}

// NewAdapterConfirmationChecker wires a confirmation checker over the adapters.
func NewAdapterConfirmationChecker(adapters adapterSource) *AdapterConfirmationChecker {
	return &AdapterConfirmationChecker{adapters: adapters}
}

// Confirmations implements confirmationChecker.
func (c *AdapterConfirmationChecker) Confirmations(ctx context.Context, chainCode, txHash string) (uint64, error) {
	adapter, err := c.adapters.Get(chainCode)
	if err != nil {
		return 0, err
	}
	return adapter.GetConfirmations(ctx, txHash)
}

// EVMSweepConfirmer advances broadcast sweep transactions through their
// confirmation lifecycle: broadcast → confirming → confirmed. When a sweep tx
// reaches its chain's required confirmation depth it marks the parent Sweep
// batch completed (which records the ledger entries, idempotently).
//
// This is the reconciliation half of the claim-first sweep design: a sweep tx
// that was broadcast but whose DB record was written is reconciled here against
// real on-chain state.
type EVMSweepConfirmer struct {
	sweepTxRepo repository.SweepTransactionRepository
	currencies  repository.BlockchainCurrencyRepository
	sweepSvc    *SweepService
	checker     confirmationChecker
}

// NewEVMSweepConfirmer constructs an EVMSweepConfirmer.
func NewEVMSweepConfirmer(
	sweepTxRepo repository.SweepTransactionRepository,
	currencies repository.BlockchainCurrencyRepository,
	sweepSvc *SweepService,
	checker confirmationChecker,
) *EVMSweepConfirmer {
	return &EVMSweepConfirmer{
		sweepTxRepo: sweepTxRepo,
		currencies:  currencies,
		sweepSvc:    sweepSvc,
		checker:     checker,
	}
}

// TrackConfirmations polls every broadcast/confirming sweep tx and advances its
// state. Returns the number newly marked confirmed this round. Per-tx errors
// (e.g. tx not yet mined) are logged and skipped.
func (c *EVMSweepConfirmer) TrackConfirmations(ctx context.Context) (int, error) {
	var pending []models.SweepTransaction
	for _, status := range []string{SweepTxStatusBroadcast, SweepTxStatusConfirming} {
		txs, err := c.sweepTxRepo.ListByStatus(status)
		if err != nil {
			return 0, err
		}
		pending = append(pending, txs...)
	}

	confirmed := 0
	for i := range pending {
		if ctx.Err() != nil {
			return confirmed, ctx.Err()
		}
		if c.trackOne(ctx, &pending[i]) {
			confirmed++
		}
	}
	return confirmed, nil
}

// trackOne advances a single sweep tx; returns true if newly confirmed.
func (c *EVMSweepConfirmer) trackOne(ctx context.Context, tx *models.SweepTransaction) bool {
	if tx.TxHash == "" {
		return false // nothing to track (orphaned payload — left for manual reconcile)
	}
	bc, err := c.currencies.GetByID(tx.BlockchainCurrencyID)
	if err != nil {
		log.Printf("[EVMSweepConfirmer] sweep tx %d: load currency: %v", tx.ID, err)
		return false
	}
	if !evmChains[bc.BlockchainCode] {
		return false // non-EVM tracked elsewhere
	}

	confs, err := c.checker.Confirmations(ctx, bc.BlockchainCode, tx.TxHash)
	if err != nil {
		// Not yet mined / transient RPC error — try again next round.
		return false
	}

	required := uint64(defaultRequiredConfirmations)
	if bc.Blockchain != nil && bc.Blockchain.MinConfirmations > 0 {
		required = uint64(bc.Blockchain.MinConfirmations)
	}

	switch {
	case confs >= required:
		if err := c.sweepTxRepo.UpdateStatus(tx.ID, SweepTxStatusConfirmed); err != nil {
			log.Printf("[EVMSweepConfirmer] sweep tx %d mark confirmed: %v", tx.ID, err)
			return false
		}
		// MarkCompleted is idempotent (atomic conditional update + ledger).
		if err := c.sweepSvc.MarkCompleted(tx.SweepID, tx.Amount, tx.GasFee, tx.BlockchainCurrencyID); err != nil {
			log.Printf("[EVMSweepConfirmer] sweep %d mark completed: %v", tx.SweepID, err)
		}
		log.Printf("[EVMSweepConfirmer] sweep tx %d confirmed (%d/%d) tx=%s", tx.ID, confs, required, tx.TxHash)
		return true
	case confs > 0 && tx.Status != SweepTxStatusConfirming:
		if err := c.sweepTxRepo.UpdateStatus(tx.ID, SweepTxStatusConfirming); err != nil {
			log.Printf("[EVMSweepConfirmer] sweep tx %d mark confirming: %v", tx.ID, err)
		}
		return false
	default:
		return false
	}
}
