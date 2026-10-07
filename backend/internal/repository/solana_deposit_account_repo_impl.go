package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// SolanaDepositAccountRepository persists the owner/ATA pairs the Solana watcher polls.
type SolanaDepositAccountRepository interface {
	Create(a *models.SolanaDepositAccount) error
	GetByDepositAddressID(depositAddressID uint) (*models.SolanaDepositAccount, error)
	GetByTokenAccount(tokenAccount string) (*models.SolanaDepositAccount, error)
	ListByStatus(status string, limit int) ([]models.SolanaDepositAccount, error)
	// ListDue returns watching accounts whose token poll is due, least recently polled first.
	ListDue(now time.Time, limit int) ([]models.SolanaDepositAccount, error)
	// ListExpiring returns watching accounts past watch_until.
	ListExpiring(now time.Time, limit int) ([]models.SolanaDepositAccount, error)
	// ListExpiredDue returns expired accounts whose slow balance scan is due.
	ListExpiredDue(now time.Time, limit int) ([]models.SolanaDepositAccount, error)
	// ExpireWatching moves watching accounts past watch_until to expired; returns how many.
	ExpireWatching(now time.Time) (int64, error)
	// Update applies column updates by id.
	Update(id uint, updates map[string]any) error
	UpdateStatus(id uint, status string) error
	// WithTx binds the repository to a transaction.
	WithTx(tx *gorm.DB) SolanaDepositAccountRepository
}

type solanaDepositAccountRepository struct{ db *gorm.DB }

// NewSolanaDepositAccountRepository constructs the GORM implementation.
func NewSolanaDepositAccountRepository(db *gorm.DB) SolanaDepositAccountRepository {
	return &solanaDepositAccountRepository{db: db}
}

func (r *solanaDepositAccountRepository) WithTx(tx *gorm.DB) SolanaDepositAccountRepository {
	return &solanaDepositAccountRepository{db: tx}
}

func (r *solanaDepositAccountRepository) Create(a *models.SolanaDepositAccount) error {
	return r.db.Create(a).Error
}

func (r *solanaDepositAccountRepository) GetByDepositAddressID(depositAddressID uint) (*models.SolanaDepositAccount, error) {
	var a models.SolanaDepositAccount
	if err := r.db.Where("deposit_address_id = ?", depositAddressID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *solanaDepositAccountRepository) GetByTokenAccount(tokenAccount string) (*models.SolanaDepositAccount, error) {
	var a models.SolanaDepositAccount
	if err := r.db.Where("token_account = ?", tokenAccount).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *solanaDepositAccountRepository) ListByStatus(status string, limit int) ([]models.SolanaDepositAccount, error) {
	var out []models.SolanaDepositAccount
	q := r.db.Where("status = ?", status).Order("last_polled_at ASC NULLS FIRST").Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *solanaDepositAccountRepository) ListDue(now time.Time, limit int) ([]models.SolanaDepositAccount, error) {
	var out []models.SolanaDepositAccount
	q := r.db.Where("status = ? AND (token_poll_after IS NULL OR token_poll_after <= ?)", models.SolanaDepositAccountWatching, now).
		Order("last_polled_at ASC NULLS FIRST").Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *solanaDepositAccountRepository) ListExpiring(now time.Time, limit int) ([]models.SolanaDepositAccount, error) {
	var out []models.SolanaDepositAccount
	q := r.db.Where("status = ? AND watch_until IS NOT NULL AND watch_until < ?", models.SolanaDepositAccountWatching, now).Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *solanaDepositAccountRepository) ListExpiredDue(now time.Time, limit int) ([]models.SolanaDepositAccount, error) {
	var out []models.SolanaDepositAccount
	q := r.db.Where("status = ? AND (token_poll_after IS NULL OR token_poll_after <= ?)", models.SolanaDepositAccountExpired, now).Order("token_poll_after ASC NULLS FIRST").Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *solanaDepositAccountRepository) ExpireWatching(now time.Time) (int64, error) {
	res := r.db.Model(&models.SolanaDepositAccount{}).
		Where("status = ? AND watch_until IS NOT NULL AND watch_until < ?", models.SolanaDepositAccountWatching, now).
		Update("status", models.SolanaDepositAccountExpired)
	return res.RowsAffected, res.Error
}

func (r *solanaDepositAccountRepository) Update(id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&models.SolanaDepositAccount{}).Where("id = ?", id).Updates(updates).Error
}

func (r *solanaDepositAccountRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.SolanaDepositAccount{}).Where("id = ?", id).Update("status", status).Error
}
