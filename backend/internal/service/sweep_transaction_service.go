package service

import (
	"fmt"
	"log"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// SweepTransactionStatus constants for on-chain sweep tx lifecycle.
const (
	SweepTxStatusPending    = "pending"
	SweepTxStatusBroadcast  = "broadcast"
	SweepTxStatusConfirming = "confirming"
	SweepTxStatusConfirmed  = "confirmed"
	SweepTxStatusFailed     = "failed"
)

// SweepTransactionService builds and tracks individual on-chain sweep
// transactions within a Sweep batch.
type SweepTransactionService struct {
	sweepTxRepo   repository.SweepTransactionRepository
	sweepRepo     repository.SweepRepository
	ledgerService *LedgerService
}

// NewSweepTransactionService constructs a SweepTransactionService.
func NewSweepTransactionService(
	sweepTxRepo repository.SweepTransactionRepository,
	sweepRepo repository.SweepRepository,
	ledgerService *LedgerService,
) *SweepTransactionService {
	return &SweepTransactionService{
		sweepTxRepo:   sweepTxRepo,
		sweepRepo:     sweepRepo,
		ledgerService: ledgerService,
	}
}

// CreateSweepTransactionPayload builds a SweepTransaction row in pending state
// without broadcasting it. The actual broadcast happens in a later phase once
// the SecretsVault signs the transaction.
//
// This is the write path used by createSweepTransactionPayload sub-goroutine
// in AccountProcessorJob.
func (s *SweepTransactionService) CreateSweepTransactionPayload(
	sweepID uint,
	blockchainCurrencyID uint,
	fromAddress string,
	toAddress string,
	amount decimal.Decimal,
) (*models.SweepTransaction, error) {
	st := &models.SweepTransaction{
		Amount:               amount,
		GasFee:               decimal.Zero,
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		Status:               SweepTxStatusPending,
		SweepID:              sweepID,
		BlockchainCurrencyID: blockchainCurrencyID,
	}
	if err := s.sweepTxRepo.Create(st); err != nil {
		return nil, fmt.Errorf("create sweep transaction payload: %w", err)
	}
	return st, nil
}

// EthAutoSweepTransaction builds a SweepTransaction row for an EVM native-coin
// sweep (ETH/BASE/POLYGON). It records the pending transaction in the database
// but does NOT broadcast it on-chain — the real broadcast with full tx signing
// is wired in Phase K when SecretsVault provides private key access.
//
// TODO(phase-k-signing): Replace the log stub below with an actual ethclient
// SendTransaction call after retrieving the decrypted private key from
// SecretsVaultService.DecryptKey(depositAddress.WalletID).
func (s *SweepTransactionService) EthAutoSweepTransaction(
	sweepID uint,
	blockchainCurrencyID uint,
	fromAddress string,
	toAddress string,
	amount decimal.Decimal,
	estimatedGas decimal.Decimal,
) (*models.SweepTransaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("eth auto sweep: amount must be positive, got %s", amount)
	}

	st := &models.SweepTransaction{
		Amount:               amount,
		GasFee:               estimatedGas,
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		Status:               SweepTxStatusPending,
		SweepID:              sweepID,
		BlockchainCurrencyID: blockchainCurrencyID,
	}
	if err := s.sweepTxRepo.Create(st); err != nil {
		return nil, fmt.Errorf("eth auto sweep: create sweep transaction: %w", err)
	}

	// TODO(phase-k-signing): Obtain decrypted private key from SecretsVault,
	// sign and broadcast the transaction via ethclient.SendTransaction, then
	// call sweepTxRepo.UpdateTxHash(st.ID, txHash) and
	// sweepTxRepo.UpdateStatus(st.ID, SweepTxStatusBroadcast).
	log.Printf("[SweepTransactionService] EthAutoSweepTransaction: sweep_tx_id=%d from=%s to=%s amount=%s gas=%s — PENDING PHASE-K SIGNING",
		st.ID, fromAddress, toAddress, amount.String(), estimatedGas.String())

	return st, nil
}

// ProcessERC20Sweep is the two-phase ERC-20 sweep: first fund the deposit
// address with ETH for gas (via InternalBlockchainTransactionService), then
// trigger the ERC-20 token transfer.
//
// Phase 1 (fund gas): record an InternalBlockchainTransaction for the gas
// funding transfer. Phase 2 (sweep tokens): create the SweepTransaction row.
//
// TODO(phase-k-signing): Both broadcast calls are stubbed; replace with real
// ethclient.SendTransaction calls after SecretsVault integration in Phase K.
func (s *SweepTransactionService) ProcessERC20Sweep(
	sweepID uint,
	blockchainCurrencyID uint,
	fromAddress string,
	toAddress string,
	tokenAmount decimal.Decimal,
	gasFundAmount decimal.Decimal,
) (*models.SweepTransaction, error) {
	if tokenAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("process erc20 sweep: token amount must be positive")
	}

	// Phase 1: gas funding is recorded as an InternalBlockchainTransaction in
	// InternalBlockchainTransactionService.RecordGasFunding (called by the
	// processERC20Sweeps goroutine in AccountProcessorJob).
	log.Printf("[SweepTransactionService] ProcessERC20Sweep: fund gas for %s amount=%s — PENDING PHASE-K SIGNING",
		fromAddress, gasFundAmount.String())

	// Phase 2: create the ERC-20 sweep transaction payload.
	st := &models.SweepTransaction{
		Amount:               tokenAmount,
		GasFee:               gasFundAmount,
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		Status:               SweepTxStatusPending,
		SweepID:              sweepID,
		BlockchainCurrencyID: blockchainCurrencyID,
	}
	if err := s.sweepTxRepo.Create(st); err != nil {
		return nil, fmt.Errorf("process erc20 sweep: create sweep transaction: %w", err)
	}

	// TODO(phase-k-signing): After gas tx confirms, sign and broadcast the ERC-20
	// transfer (approve + transferFrom or direct transfer). Update st.TxHash and
	// transition status to SweepTxStatusBroadcast.
	log.Printf("[SweepTransactionService] ProcessERC20Sweep: sweep_tx_id=%d ERC-20 transfer from=%s to=%s tokens=%s — PENDING PHASE-K SIGNING",
		st.ID, fromAddress, toAddress, tokenAmount.String())

	return st, nil
}

// RecordBroadcastSweep persists a SweepTransaction that has ALREADY been signed
// and broadcast on-chain, in the 'broadcast' state with its real tx hash and
// gas fee. Used by EVMSweepService after a successful broadcast.
func (s *SweepTransactionService) RecordBroadcastSweep(
	sweepID, blockchainCurrencyID uint,
	fromAddress, toAddress, txHash string,
	amount, gasFee decimal.Decimal,
) (*models.SweepTransaction, error) {
	st := &models.SweepTransaction{
		TxHash:               txHash,
		Amount:               amount,
		GasFee:               gasFee,
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		Status:               SweepTxStatusBroadcast,
		SweepID:              sweepID,
		BlockchainCurrencyID: blockchainCurrencyID,
	}
	if err := s.sweepTxRepo.Create(st); err != nil {
		return nil, fmt.Errorf("record broadcast sweep: %w", err)
	}
	return st, nil
}

// UpdateStatus transitions a SweepTransaction to the given status.
func (s *SweepTransactionService) UpdateStatus(id uint, status string) error {
	return s.sweepTxRepo.UpdateStatus(id, status)
}

// UpdateTxHash records the on-chain hash for a broadcast SweepTransaction.
func (s *SweepTransactionService) UpdateTxHash(id uint, txHash string) error {
	return s.sweepTxRepo.UpdateTxHash(id, txHash)
}

// ListPending returns all SweepTransactions in pending state.
func (s *SweepTransactionService) ListPending() ([]models.SweepTransaction, error) {
	return s.sweepTxRepo.ListByStatus(SweepTxStatusPending)
}

// ListBySweep returns all SweepTransactions belonging to a sweep batch.
func (s *SweepTransactionService) ListBySweep(sweepID uint) ([]models.SweepTransaction, error) {
	return s.sweepTxRepo.ListBySweep(sweepID)
}

// ListByStatus returns SweepTransactions with the given status string.
// Used by AccountProcessorJob sub-loops to find pending/broadcast/confirming txs.
func (s *SweepTransactionService) ListByStatus(status string) ([]models.SweepTransaction, error) {
	return s.sweepTxRepo.ListByStatus(status)
}
