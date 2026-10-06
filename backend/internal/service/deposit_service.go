package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// DepositService is the write-side for deposits detected by the block
// processors. It records new deposits, dedups against existing ones, links
// them to open PaymentRequests, and advances confirmation counts.
type DepositService struct {
	depositRepo        repository.DepositRepository
	depositAddressRepo repository.DepositAddressRepository
	paymentRepo        repository.PaymentRepository
	blockchainCurRepo  repository.BlockchainCurrencyRepository
	blockchainRepo     repository.BlockchainRepository
}

// NewDepositService constructs a DepositService with the required repositories.
func NewDepositService(
	depositRepo repository.DepositRepository,
	depositAddressRepo repository.DepositAddressRepository,
	paymentRepo repository.PaymentRepository,
	blockchainCurRepo repository.BlockchainCurrencyRepository,
	blockchainRepo repository.BlockchainRepository,
) *DepositService {
	return &DepositService{
		depositRepo:        depositRepo,
		depositAddressRepo: depositAddressRepo,
		paymentRepo:        paymentRepo,
		blockchainCurRepo:  blockchainCurRepo,
		blockchainRepo:     blockchainRepo,
	}
}

// RecordDeposit creates a Deposit row from a detected chain transaction.
// Returns the created Deposit, or the existing one if the (tx_hash,
// to_address, blockchain_currency_id) tuple already exists (idempotent).
//
// The method:
//  1. Checks for an existing deposit with the same tx_hash + to_address +
//     blockchain_currency_id (duplicate detection)
//  2. Looks up the DepositAddress by ToAddress to find the owning member
//     and linked PaymentRequest
//  3. Looks up the Blockchain to get the required confirmations
//  4. Creates the Deposit row in 'pending' state with 0 confirmations
//  5. If there's a matching PaymentRequest, sets PaymentRequestID on the deposit
func (s *DepositService) RecordDeposit(ctx context.Context, tx blockchain.Transaction, blockchainCurrencyID uint) (*models.Deposit, error) {
	if tx.TxHash == "" {
		return nil, errors.New("tx hash is empty")
	}
	if tx.ToAddress == "" {
		return nil, errors.New("to address is empty")
	}

	// Duplicate detection
	existing, err := s.depositRepo.GetByTxIDAndToAddress(tx.TxHash, tx.ToAddress, blockchainCurrencyID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("duplicate check: %w", err)
	}

	// Look up the deposit address
	depositAddr, err := s.depositAddressRepo.GetByAddress(tx.ToAddress, blockchainCurrencyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepositAddressNotFound
		}
		return nil, fmt.Errorf("lookup deposit address: %w", err)
	}

	// Resolve required confirmations from the blockchain row
	bc, err := s.blockchainCurRepo.GetByID(blockchainCurrencyID)
	if err != nil {
		return nil, fmt.Errorf("lookup blockchain_currency: %w", err)
	}

	var requiredConf int
	if bc.Blockchain != nil {
		requiredConf = bc.Blockchain.MinConfirmations
	} else {
		// Fetch separately if not preloaded
		chain, chainErr := s.blockchainRepo.GetByID(bc.BlockchainID)
		if chainErr == nil && chain != nil {
			requiredConf = chain.MinConfirmations
		}
	}
	if requiredConf < 1 {
		requiredConf = 1
	}

	deposit := &models.Deposit{
		TxID:                  tx.TxHash,
		Amount:                tx.Amount,
		Status:                models.DepositStatusPending,
		Confirmations:         0,
		RequiredConfirmations: requiredConf,
		FromAddress:           tx.FromAddress,
		ToAddress:             tx.ToAddress,
		BlockNumber:           int64(tx.BlockNumber),
		BlockHash:             tx.BlockHash,
		BlockchainCurrencyID:  blockchainCurrencyID,
		MemberID:              depositAddr.MemberID,
		PaymentRequestID:      depositAddr.PaymentRequestID,
	}
	if err := s.depositRepo.Create(deposit); err != nil {
		return nil, fmt.Errorf("create deposit: %w", err)
	}

	return deposit, nil
}

// UpdateConfirmations computes the current confirmation count from the
// deposit's block number and the latest chain tip, then atomically advances
// the deposit state via ConfirmIfPending. Calling FinalizePayment is gated on
// ConfirmIfPending reporting that exactly this goroutine performed the final
// state transition, preventing double-finalization when two block processors
// race on the same deposit.
func (s *DepositService) UpdateConfirmations(depositID uint, currentBlockHeight int64) error {
	deposit, err := s.depositRepo.GetByID(depositID)
	if err != nil {
		return err
	}

	// Already in a terminal state or no block to anchor to — nothing to do.
	if deposit.Status == models.DepositStatusConfirmed {
		return nil
	}
	if deposit.BlockNumber == 0 || currentBlockHeight < deposit.BlockNumber {
		return nil
	}

	confirmations := int(currentBlockHeight - deposit.BlockNumber + 1)
	if confirmations < 0 {
		confirmations = 0
	}

	// ConfirmIfPending performs a single conditional UPDATE that advances both
	// the confirmations count and the status in one round-trip. If two
	// goroutines race, only the first one gets rowsAffected == 1; the second
	// sees 0 and must not re-trigger finalization.
	rowsAffected, err := s.depositRepo.ConfirmIfPending(depositID, confirmations)
	if err != nil {
		return fmt.Errorf("confirm if pending: %w", err)
	}

	// Only the winning goroutine (rowsAffected == 1) should trigger downstream
	// work. The threshold check uses the locally-computed confirmations value
	// because ConfirmIfPending already encoded the same logic in SQL.
	if rowsAffected == 1 &&
		confirmations >= deposit.RequiredConfirmations &&
		deposit.PaymentRequestID != nil {
		_ = s.FinalizePayment(*deposit.PaymentRequestID)
	}
	return nil
}

// FinalizePayment is a thin wrapper over PaymentRepository.FinalizeFromConfirmedDeposits.
// All concurrency control (SELECT FOR UPDATE, sum-in-tx, idempotency guard) is
// handled inside the repo method. This service method exists so that higher-level
// callers (e.g. UpdateConfirmations, block processors) have a stable entry point.
func (s *DepositService) FinalizePayment(paymentRequestID uint) error {
	_, _, _, err := s.paymentRepo.FinalizeFromConfirmedDeposits(paymentRequestID)
	return err
}

// GetUnspentDeposits returns confirmed deposits that haven't been swept yet,
// scoped to a single merchant. Returns an empty slice when memberID is 0 as a
// defensive measure — passing 0 would otherwise enumerate every merchant's
// deposits (P0 data leak).
func (s *DepositService) GetUnspentDeposits(memberID uint) ([]models.Deposit, error) {
	if memberID == 0 {
		return []models.Deposit{}, nil
	}
	return s.depositRepo.ListByStatusAndMember(models.DepositStatusConfirmed, memberID)
}

// ListConfirmingForChain returns deposits in 'pending' or 'confirming' state
// for the given blockchain. Used by BlockchainProcessor to advance
// confirmation counts on each block tick.
func (s *DepositService) ListConfirmingForChain(blockchainID uint, limit int) ([]models.Deposit, error) {
	return s.depositRepo.ListConfirmingForChain(blockchainID, limit)
}

// ErrDepositAddressNotFound signals that a detected transaction hit an
// address Payminto does not own — the caller should route it to
// MissedDepositService.
var ErrDepositAddressNotFound = errors.New("deposit address not found")
