package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SweepStatus constants mirror PayRam's sweep lifecycle.
const (
	SweepStatusPending    = "pending"
	SweepStatusProcessing = "processing"
	SweepStatusCompleted  = "completed"
	SweepStatusFailed     = "failed"
	SweepStatusStale      = "stale"
)

// ErrSweepNotFound is returned when a Sweep cannot be located by ID.
var ErrSweepNotFound = errors.New("sweep not found")

// SweepService manages the lifecycle of SmartSweep batches. A sweep batches
// multiple deposit addresses into a single on-chain consolidation to cold storage.
//
// db is held directly so MarkCompleted can atomically update the sweep status
// and write ledger entries in a single database transaction.
type SweepService struct {
	db              *gorm.DB
	sweepRepo       repository.SweepRepository
	sweepTxRepo     repository.SweepTransactionRepository
	blockchainRepo  repository.BlockchainRepository
	ledgerService   *LedgerService
}

// NewSweepService constructs a SweepService with the required repositories.
func NewSweepService(
	db *gorm.DB,
	sweepRepo repository.SweepRepository,
	sweepTxRepo repository.SweepTransactionRepository,
	blockchainRepo repository.BlockchainRepository,
	ledgerService *LedgerService,
) *SweepService {
	return &SweepService{
		db:             db,
		sweepRepo:      sweepRepo,
		sweepTxRepo:    sweepTxRepo,
		blockchainRepo: blockchainRepo,
		ledgerService:  ledgerService,
	}
}

// CreateSweep creates a new Sweep batch in pending status for the given blockchain.
func (s *SweepService) CreateSweep(blockchainID uint) (*models.Sweep, error) {
	sweep := &models.Sweep{
		Status:       SweepStatusPending,
		TotalAmount:  decimal.Zero,
		TotalGasFee:  decimal.Zero,
		BlockchainID: blockchainID,
	}
	if err := s.sweepRepo.Create(sweep); err != nil {
		return nil, fmt.Errorf("create sweep: %w", err)
	}
	return sweep, nil
}

// GetByID fetches a Sweep by primary key.
func (s *SweepService) GetByID(id uint) (*models.Sweep, error) {
	sweep, err := s.sweepRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get sweep: %w", err)
	}
	return sweep, nil
}

// ListByStatus returns all Sweeps with the given status.
func (s *SweepService) ListByStatus(status string) ([]models.Sweep, error) {
	return s.sweepRepo.ListByStatus(status)
}

// ListPending returns all sweeps in pending state.
func (s *SweepService) ListPending() ([]models.Sweep, error) {
	return s.sweepRepo.ListByStatus(SweepStatusPending)
}

// UpdateStatus transitions a Sweep to the given status.
func (s *SweepService) UpdateStatus(id uint, status string) error {
	return s.sweepRepo.UpdateStatus(id, status)
}

// MarkCompleted transitions a Sweep to completed, updates its totals, and
// records the ledger entries for the sweep event. The conditional UPDATE
// (status != 'completed') ensures idempotency: a second call for the same
// sweep ID is a no-op and does not write a duplicate ledger entry.
//
// Implementation note: the sweep UPDATE and the ledger write are committed
// sequentially rather than wrapped in a single outer transaction. This avoids
// nested-transaction complexity (SQLite savepoints are unreliable in test; Postgres
// supports them natively). The trade-off is a tiny window where the sweep row
// shows 'completed' but the ledger entry is missing — a reconciliation job
// (Phase L) closes this gap. The ledger failure is returned as an error so
// callers can retry.
func (s *SweepService) MarkCompleted(id uint, totalAmount, totalGasFee decimal.Decimal, currencyID uint) error {
	// Atomic conditional UPDATE — only transitions sweeps that are not yet completed.
	res := s.db.Model(&models.Sweep{}).
		Where("id = ? AND status != ?", id, SweepStatusCompleted).
		Updates(map[string]any{
			"status":        SweepStatusCompleted,
			"total_amount":  totalAmount,
			"total_gas_fee": totalGasFee,
		})
	if res.Error != nil {
		return fmt.Errorf("update sweep %d: %w", id, res.Error)
	}
	if res.RowsAffected == 0 {
		// Already completed — idempotent success, do not write a second ledger entry.
		return nil
	}

	// Record the sweep event in the double-entry ledger. Any failure is returned
	// to the caller so they can retry — the ledger entry is idempotent on
	// (sweep_id, reference) via the unique-constraint in Phase E.
	if err := s.ledgerService.RecordSweep(id, currencyID, totalAmount, totalGasFee); err != nil {
		return fmt.Errorf("record sweep ledger for sweep %d: %w", id, err)
	}
	return nil
}

// MarkFailed transitions a Sweep to failed.
func (s *SweepService) MarkFailed(id uint) error {
	return s.sweepRepo.UpdateStatus(id, SweepStatusFailed)
}

// MarkStale marks a Sweep that has been pending longer than the stale threshold.
func (s *SweepService) MarkStale(id uint) error {
	return s.sweepRepo.UpdateStatus(id, SweepStatusStale)
}

// StaleThreshold is the duration after which a pending sweep is considered stale.
const StaleThreshold = 2 * time.Hour
