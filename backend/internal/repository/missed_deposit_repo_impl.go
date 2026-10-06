package repository

import (
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// MissedDepositRepository persists deposits detected on-chain that did not
// match any open payment request. Operators reconcile these manually.
type MissedDepositRepository interface {
	Create(d *models.MissedDeposit) error
	Update(d *models.MissedDeposit) error
	GetByID(id uint) (*models.MissedDeposit, error)
	GetByTxHash(txHash string) ([]models.MissedDeposit, error)
	ListByStatus(status string, opts ...QueryOption) ([]models.MissedDeposit, error)
	ListUnresolved(opts ...QueryOption) ([]models.MissedDeposit, error)
	// MarkResolved atomically transitions a missed deposit from 'pending' to
	// the given terminal status. Returns (rowsAffected, error). rowsAffected==0
	// signals the row was not in 'pending' state — the service translates this
	// to ErrMissedDepositAlreadyResolved.
	MarkResolved(id uint, status, resolution string, resolvedBy uint) (int64, error)
}

// MissedDepositRepositoryImpl is the GORM-backed implementation of MissedDepositRepository.
type MissedDepositRepositoryImpl struct {
	db *gorm.DB
}

// NewMissedDepositRepository constructs a new MissedDepositRepository backed by the provided *gorm.DB.
func NewMissedDepositRepository(db *gorm.DB) MissedDepositRepository {
	return &MissedDepositRepositoryImpl{db: db}
}

// Create inserts a new MissedDeposit record.
func (r *MissedDepositRepositoryImpl) Create(d *models.MissedDeposit) error {
	return r.db.Create(d).Error
}

// Update saves all fields of the given MissedDeposit.
func (r *MissedDepositRepositoryImpl) Update(d *models.MissedDeposit) error {
	return r.db.Save(d).Error
}

// GetByID fetches a MissedDeposit by primary key, preloading BlockchainCurrency.
func (r *MissedDepositRepositoryImpl) GetByID(id uint) (*models.MissedDeposit, error) {
	var d models.MissedDeposit
	err := r.db.Preload("BlockchainCurrency").First(&d, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &d, nil
}

// GetByTxHash returns all MissedDeposits sharing a transaction hash. Multiple
// outputs in one transaction may each produce a separate missed-deposit row.
func (r *MissedDepositRepositoryImpl) GetByTxHash(txHash string) ([]models.MissedDeposit, error) {
	var deposits []models.MissedDeposit
	if err := r.db.Where("tx_hash = ?", txHash).Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// ListByStatus returns MissedDeposits with the given status, applying any
// provided QueryOptions (e.g. Limit, Offset, Order).
func (r *MissedDepositRepositoryImpl) ListByStatus(status string, opts ...QueryOption) ([]models.MissedDeposit, error) {
	q := Apply(r.db.Where("status = ?", status), opts...)
	var deposits []models.MissedDeposit
	if err := q.Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// ListUnresolved returns all MissedDeposits in 'pending' status, applying any
// provided QueryOptions.
func (r *MissedDepositRepositoryImpl) ListUnresolved(opts ...QueryOption) ([]models.MissedDeposit, error) {
	q := Apply(r.db.Where("status = ?", models.MissedDepositStatusPending), opts...)
	var deposits []models.MissedDeposit
	if err := q.Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// MarkResolved sets the status, resolution text, resolved_by member ID, and
// resolved_at timestamp on a MissedDeposit. Uses a targeted UPDATE to avoid
// loading the full row.
func (r *MissedDepositRepositoryImpl) MarkResolved(id uint, status, resolution string, resolvedBy uint) (int64, error) {
	now := time.Now().UTC()
	res := r.db.Model(&models.MissedDeposit{}).
		Where("id = ? AND status = ?", id, models.MissedDepositStatusPending).
		Updates(map[string]any{
			"status":      status,
			"resolution":  resolution,
			"resolved_by": resolvedBy,
			"resolved_at": now,
			"updated_at":  now,
		})
	return res.RowsAffected, res.Error
}
