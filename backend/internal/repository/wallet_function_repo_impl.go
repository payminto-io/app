package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// WalletFunctionRepository defines the persistence contract for wallet audit log entries.
type WalletFunctionRepository interface {
	Create(f *models.WalletFunction) error
	ListByWallet(walletID uint, opts ...QueryOption) ([]models.WalletFunction, error)
	ListByMember(memberID uint, opts ...QueryOption) ([]models.WalletFunction, error)
	ListByAction(action string, limit int) ([]models.WalletFunction, error)
}

// WalletFunctionRepositoryImpl is the GORM-backed implementation of WalletFunctionRepository.
type WalletFunctionRepositoryImpl struct {
	db *gorm.DB
}

// NewWalletFunctionRepository constructs a WalletFunctionRepository backed by db.
func NewWalletFunctionRepository(db *gorm.DB) WalletFunctionRepository {
	return &WalletFunctionRepositoryImpl{db: db}
}

func (r *WalletFunctionRepositoryImpl) Create(f *models.WalletFunction) error {
	return r.db.Create(f).Error
}

func (r *WalletFunctionRepositoryImpl) ListByWallet(walletID uint, opts ...QueryOption) ([]models.WalletFunction, error) {
	var out []models.WalletFunction
	q := Apply(r.db.Where("wallet_id = ?", walletID).Order("created_at DESC"), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *WalletFunctionRepositoryImpl) ListByMember(memberID uint, opts ...QueryOption) ([]models.WalletFunction, error) {
	var out []models.WalletFunction
	q := Apply(r.db.Where("member_id = ?", memberID).Order("created_at DESC"), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *WalletFunctionRepositoryImpl) ListByAction(action string, limit int) ([]models.WalletFunction, error) {
	var out []models.WalletFunction
	q := r.db.Where("action = ?", action).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
