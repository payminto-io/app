package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// InternalBlockchainTxPurpose constants for InternalBlockchainTransaction rows.
const (
	IBTPurposeGasFunding = "gas_funding" // fund deposit address with ETH for ERC-20 sweep
	IBTPurposeInternal   = "internal"    // generic internal move
)

// InternalBlockchainTransactionService records gas-fee transfers and other
// internal moves between Payminto-controlled addresses. These records feed the
// double-entry ledger via LedgerService.RecordGasFee.
//
// db is held directly so RecordGasFunding can create the IBT row and the
// ledger entry in a single database transaction.
type InternalBlockchainTransactionService struct {
	db            *gorm.DB
	ibtRepo       repository.InternalBlockchainTransactionRepository
	ledgerService *LedgerService
}

// NewInternalBlockchainTransactionService constructs an
// InternalBlockchainTransactionService with the required dependencies.
func NewInternalBlockchainTransactionService(
	db *gorm.DB,
	ibtRepo repository.InternalBlockchainTransactionRepository,
	ledgerService *LedgerService,
) *InternalBlockchainTransactionService {
	return &InternalBlockchainTransactionService{
		db:            db,
		ibtRepo:       ibtRepo,
		ledgerService: ledgerService,
	}
}

// RecordGasFunding records a gas-funding transfer in the database. The tx is
// created in 'pending' state; the worker updates it to 'confirmed' once the
// on-chain tx is included in a block. The IBT row and ledger entry are written
// atomically — if the ledger write fails the whole operation rolls back.
//
// The placeholder TxHash uses a UUID to avoid collisions between concurrent
// pending IBTs for the same (from, to) pair.
//
// TODO(phase-k-signing): The actual on-chain send is stubbed here. After Phase K
// SecretsVault integration, call the Ethereum adapter's SendTransaction and
// populate tx.TxHash before persisting.
func (s *InternalBlockchainTransactionService) RecordGasFunding(
	blockchainCurrencyID uint,
	fromAddress string,
	toAddress string,
	amount decimal.Decimal,
	estimatedGas decimal.Decimal,
) (*models.InternalBlockchainTransaction, error) {
	ibt := &models.InternalBlockchainTransaction{
		// Use a UUID so concurrent pending IBTs for the same (from,to) pair
		// don't collide on the tx_hash unique index, and slicing long addresses
		// cannot panic.
		TxHash:               "pending_" + uuid.NewString(),
		BlockchainCurrencyID: blockchainCurrencyID,
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		Amount:               amount,
		GasFee:               estimatedGas,
		Purpose:              IBTPurposeGasFunding,
		Status:               "pending",
	}

	ctx := context.Background()
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		binder, ok := s.ibtRepo.(repository.InternalBlockchainTransactionTxBinder)
		if !ok {
			return fmt.Errorf("record gas funding: repository %T cannot join the transaction", s.ibtRepo)
		}
		if err := binder.WithTx(tx).Create(ibt); err != nil {
			return fmt.Errorf("record gas funding: create ibt: %w", err)
		}

		// TODO(phase-k-signing): Sign and broadcast the ETH gas transfer here.
		// Replace the placeholder TxHash with the real one returned by SendTransaction.
		log.Printf("[InternalBlockchainTransactionService] gas funding: id=%d from=%s to=%s amount=%s — PENDING PHASE-K SIGNING",
			ibt.ID, fromAddress, toAddress, amount.String())

		// Record the gas cost in the ledger as an expense.
		if err := s.ledgerService.RecordGasFeeIn(ctx, tx, ibt.ID, blockchainCurrencyID, estimatedGas); err != nil {
			return fmt.Errorf("record gas funding: ledger gas fee for ibt %d: %w", ibt.ID, err)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return ibt, nil
}

// Confirm transitions an InternalBlockchainTransaction to confirmed and records
// the block number at which it was included.
func (s *InternalBlockchainTransactionService) Confirm(id uint, blockNumber int64) error {
	if err := s.ibtRepo.UpdateStatus(id, "confirmed"); err != nil {
		return fmt.Errorf("confirm ibt: update status: %w", err)
	}
	return s.ibtRepo.UpdateBlockNumber(id, blockNumber)
}

// GetByTxHash fetches an InternalBlockchainTransaction by on-chain hash.
// Returns nil, nil when not found (callers should check for gorm.ErrRecordNotFound).
func (s *InternalBlockchainTransactionService) GetByTxHash(txHash string) (*models.InternalBlockchainTransaction, error) {
	tx, err := s.ibtRepo.GetByTxHash(txHash)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return tx, err
}

// ListPending returns all InternalBlockchainTransactions in pending state.
func (s *InternalBlockchainTransactionService) ListPending() ([]models.InternalBlockchainTransaction, error) {
	return s.ibtRepo.ListByStatus("pending")
}
