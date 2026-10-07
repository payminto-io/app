package service

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// WithdrawalProcessingService orchestrates approved withdrawals going to
// merchant wallets. It picks up Withdrawal rows in state 'pending', builds
// on-chain transactions via the AdapterRegistry (signing is stubbed pending
// Phase K), records a Withdraw row, updates state, and calls the ledger.
type WithdrawalProcessingService struct {
	withdrawalRepo repository.WithdrawalRepository
	withdrawRepo   repository.WithdrawRepository
	ledgerSvc      *LedgerService
	adapterReg     *blockchain.AdapterRegistry
	// hotWallets and bcCurrencyRepo enable real EVM broadcast. When nil (or for
	// non-EVM chains), Execute falls back to the legacy recorded-only path.
	hotWallets     *HotWalletSource
	bcCurrencyRepo repository.BlockchainCurrencyRepository
}

// NewWithdrawalProcessingService constructs a WithdrawalProcessingService.
// hotWallets and bcCurrencyRepo may be nil to disable real EVM broadcast.
func NewWithdrawalProcessingService(
	withdrawalRepo repository.WithdrawalRepository,
	withdrawRepo repository.WithdrawRepository,
	ledgerSvc *LedgerService,
	adapterReg *blockchain.AdapterRegistry,
	hotWallets *HotWalletSource,
	bcCurrencyRepo repository.BlockchainCurrencyRepository,
) *WithdrawalProcessingService {
	return &WithdrawalProcessingService{
		withdrawalRepo: withdrawalRepo,
		withdrawRepo:   withdrawRepo,
		ledgerSvc:      ledgerSvc,
		adapterReg:     adapterReg,
		hotWallets:     hotWallets,
		bcCurrencyRepo: bcCurrencyRepo,
	}
}

func (s *WithdrawalProcessingService) markSentWithLedger(ctx context.Context, withdrawal *models.Withdrawal, txHash string, gasFee decimal.Decimal) error {
	if s.ledgerSvc == nil {
		if _, err := s.withdrawalRepo.MarkSent(withdrawal.ID, txHash); err != nil {
			return fmt.Errorf("mark sent for withdrawal %d: %w", withdrawal.ID, err)
		}
		return nil
	}
	return s.ledgerSvc.InTransaction(ctx, func(tx *gorm.DB) error {
		repo := s.withdrawalRepo
		binder, ok := repo.(repository.WithdrawalTxBinder)
		if !ok {
			return fmt.Errorf("withdrawal repository %T cannot join the ledger transaction", repo)
		}
		repo = binder.WithTx(tx)
		if _, err := repo.MarkSent(withdrawal.ID, txHash); err != nil {
			return fmt.Errorf("mark sent for withdrawal %d: %w", withdrawal.ID, err)
		}
		if err := s.ledgerSvc.RecordWithdrawalIn(ctx, tx, withdrawal.ID, withdrawal.BlockchainCurrencyID, withdrawal.Amount, gasFee); err != nil {
			return fmt.Errorf("ledger for withdrawal %d: %w", withdrawal.ID, err)
		}
		return nil
	})
}

// canBroadcastEVM reports whether real EVM broadcast is wired and applicable.
func (s *WithdrawalProcessingService) canBroadcastEVM(chainCode string) bool {
	if s.hotWallets == nil || s.bcCurrencyRepo == nil || s.adapterReg == nil {
		return false
	}
	fam, ok := familyForChain(chainCode)
	return ok && fam == "ETH_Family"
}

// broadcastEVM signs and broadcasts a withdrawal from the merchant's hot wallet.
// Returns the real tx hash, the hot-wallet from-address, and the gas fee paid
// (native wei → decimal). The private key is zeroed before returning.
func (s *WithdrawalProcessingService) broadcastEVM(ctx context.Context, w *models.Withdrawal) (txHash, fromAddr string, gasFee decimal.Decimal, err error) {
	adapter, err := s.adapterReg.Get(w.BlockchainCode)
	if err != nil {
		return "", "", decimal.Zero, fmt.Errorf("no adapter for %s: %w", w.BlockchainCode, err)
	}
	evm, ok := adapter.(blockchain.EVMSigner)
	if !ok {
		return "", "", decimal.Zero, fmt.Errorf("%s adapter does not support EVM signing", w.BlockchainCode)
	}
	bc, err := s.bcCurrencyRepo.GetByID(w.BlockchainCurrencyID)
	if err != nil {
		return "", "", decimal.Zero, fmt.Errorf("load currency: %w", err)
	}

	addr, privKey, err := s.hotWallets.Resolve(ctx, w.MemberID, w.BlockchainCode)
	if err != nil {
		return "", "", decimal.Zero, err
	}
	defer zeroBytes(privKey)

	decimals := int32(18)
	if bc.WalletPrecision > 0 {
		decimals = int32(bc.WalletPrecision)
	}
	amount := w.Amount.Mul(decimal.New(1, decimals)).BigInt()

	var feeWei *big.Int
	if bc.Address == "" { // native coin
		txHash, feeWei, err = evm.SendNative(ctx, privKey, w.ToAddress, amount)
	} else { // ERC-20
		txHash, feeWei, err = evm.SendERC20(ctx, privKey, bc.Address, w.ToAddress, amount)
	}
	if err != nil {
		return "", "", decimal.Zero, err
	}
	// Gas fee is always paid in the native coin (18 decimals), not token units.
	return txHash, addr, decimal.NewFromBigInt(feeWei, -18), nil
}

// ProcessApproved picks up approved withdrawals (state='pending') and submits
// them on-chain. Called by AccountProcessorJob goroutine 8.
// ctx is honoured between each withdrawal for graceful shutdown.
func (s *WithdrawalProcessingService) ProcessApproved(ctx context.Context) error {
	return s.ProcessPending(ctx)
}

// ProcessPending fetches up to 20 withdrawals in state='pending' and executes
// each one. Honours ctx cancellation between iterations. Returns the first
// execution error encountered (others continue).
func (s *WithdrawalProcessingService) ProcessPending(ctx context.Context) error {
	withdrawals, err := s.withdrawalRepo.ListPendingForProcessing(20)
	if err != nil {
		return fmt.Errorf("list pending withdrawals: %w", err)
	}

	var lastErr error
	for i := range withdrawals {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.Execute(ctx, &withdrawals[i]); err != nil {
			log.Printf("[WithdrawalProcessingService] Execute withdrawal %d: %v", withdrawals[i].ID, err)
			lastErr = err
		}
	}
	return lastErr
}

// Execute processes a single Withdrawal using a guarded state machine:
//
//  1. ClaimForProcessing: atomic pending → initiated (only one worker wins).
//  2. Create Withdraw stub row (rolls back claim on failure).
//  3. MarkSent: initiated → sent, in one transaction with
//  4. the ledger journal; a failed post rolls the state change back.
//  5. MarkProcessed: sent → processed.
//
// PHASE-K-STUB: Steps 3–5 execute synchronously and the tx hash is a stub.
// In Phase K this will wait for on-chain confirmation before calling MarkProcessed.
func (s *WithdrawalProcessingService) Execute(ctx context.Context, withdrawal *models.Withdrawal) error {
	// Step 1: Atomic claim — transition pending → initiated.
	// If rowsAffected == 0, another worker already claimed it — skip silently.
	rowsAffected, err := s.withdrawalRepo.ClaimForProcessing(withdrawal.ID)
	if err != nil {
		return fmt.Errorf("claim withdrawal %d: %w", withdrawal.ID, err)
	}
	if rowsAffected == 0 {
		log.Printf("[WithdrawalProcessingService] withdrawal %d already claimed by another worker, skipping", withdrawal.ID)
		return nil
	}

	// Step 2: Broadcast on-chain. For EVM chains with a configured hot wallet
	// this signs and submits a real transaction; otherwise it falls back to a
	// recorded-only entry (TRX/BTC are not yet wired — see docs/EVM_SETTLEMENT.md).
	now := time.Now()
	var txHash, fromAddress string
	gasFee := decimal.Zero

	if s.canBroadcastEVM(withdrawal.BlockchainCode) {
		h, from, fee, berr := s.broadcastEVM(ctx, withdrawal)
		if berr != nil {
			// Broadcast failed — release the claim so it retries next round and
			// no Withdraw row / ledger entry is written for an un-sent payout.
			if _, rerr := s.withdrawalRepo.RevertClaim(withdrawal.ID); rerr != nil {
				log.Printf("[WithdrawalProcessingService] revert claim for withdrawal %d: %v", withdrawal.ID, rerr)
			}
			return fmt.Errorf("broadcast withdrawal %d: %w", withdrawal.ID, berr)
		}
		txHash, fromAddress, gasFee = h, from, fee
		log.Printf("[WithdrawalProcessingService] withdrawal %d broadcast on %s tx=%s from=%s", withdrawal.ID, withdrawal.BlockchainCode, txHash, fromAddress)
	} else {
		// Non-EVM / unconfigured: recorded-only (no real on-chain movement yet).
		txHash = fmt.Sprintf("pending_%d_%d", withdrawal.ID, now.Unix())
		fromAddress = "unconfigured"
		log.Printf("[WithdrawalProcessingService] withdrawal %d recorded (no broadcast for %s)", withdrawal.ID, withdrawal.BlockchainCode)
	}

	// Create the on-chain Withdraw record. Roll back the claim on failure
	// so the worker loop can retry on the next tick.
	withdraw := &models.Withdraw{
		WithdrawalID:         withdrawal.ID,
		TxHash:               &txHash,
		BlockchainCurrencyID: withdrawal.BlockchainCurrencyID,
		FromAddress:          fromAddress,
		ToAddress:            withdrawal.ToAddress,
		Amount:               withdrawal.Amount,
		GasFee:               gasFee,
		Status:               "pending",
		BroadcastedAt:        &now,
	}
	if err := s.withdrawRepo.Create(withdraw); err != nil {
		// Roll back the claim so the withdrawal re-enters the pending queue.
		if _, revertErr := s.withdrawalRepo.RevertClaim(withdrawal.ID); revertErr != nil {
			log.Printf("[WithdrawalProcessingService] revert claim for withdrawal %d: %v", withdrawal.ID, revertErr)
		}
		return fmt.Errorf("create withdraw row for withdrawal %d: %w", withdrawal.ID, err)
	}

	// Record the broadcast details on the parent Withdrawal row.
	if err := s.withdrawalRepo.RecordBroadcast(withdrawal.ID, txHash, now); err != nil {
		log.Printf("[WithdrawalProcessingService] record broadcast for withdrawal %d: %v", withdrawal.ID, err)
	}

	// Steps 3 and 4: initiated -> sent and the ledger journal commit together; a failed post leaves the row initiated.
	if err := s.markSentWithLedger(ctx, withdrawal, txHash, gasFee); err != nil {
		return err
	}

	// Step 5: sent → processed.
	// NOTE: confirmation-gating (advance only after N confirmations, mirroring
	// EVMSweepConfirmer) is the remaining refinement tracked in EVM_SETTLEMENT.md.
	if _, err := s.withdrawalRepo.MarkProcessed(withdrawal.ID); err != nil {
		return fmt.Errorf("mark processed for withdrawal %d: %w", withdrawal.ID, err)
	}

	log.Printf("[WithdrawalProcessingService] withdrawal %d processed (tx=%s)", withdrawal.ID, txHash)
	return nil
}
