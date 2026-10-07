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
	UpdateCursors(id uint, tokenAccountCursor, ownerCursor string, lastSeenSlot int64, polledAt time.Time) error
	UpdateStatus(id uint, status string) error
}

type solanaDepositAccountRepository struct{ db *gorm.DB }

// NewSolanaDepositAccountRepository constructs the GORM implementation.
func NewSolanaDepositAccountRepository(db *gorm.DB) SolanaDepositAccountRepository {
	return &solanaDepositAccountRepository{db: db}
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

// ListByStatus returns the least recently polled accounts first so every account gets a turn.
func (r *solanaDepositAccountRepository) ListByStatus(status string, limit int) ([]models.SolanaDepositAccount, error) {
	var out []models.SolanaDepositAccount
	q := r.db.Where("status = ?", status).Order("last_polled_at ASC NULLS FIRST").Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *solanaDepositAccountRepository) UpdateCursors(id uint, tokenAccountCursor, ownerCursor string, lastSeenSlot int64, polledAt time.Time) error {
	updates := map[string]any{"last_polled_at": polledAt}
	if tokenAccountCursor != "" {
		updates["token_account_cursor"] = tokenAccountCursor
	}
	if ownerCursor != "" {
		updates["owner_cursor"] = ownerCursor
	}
	if lastSeenSlot > 0 {
		updates["last_seen_slot"] = gorm.Expr("GREATEST(last_seen_slot, ?)", lastSeenSlot)
	}
	return r.db.Model(&models.SolanaDepositAccount{}).Where("id = ?", id).Updates(updates).Error
}

func (r *solanaDepositAccountRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.SolanaDepositAccount{}).Where("id = ?", id).Update("status", status).Error
}
