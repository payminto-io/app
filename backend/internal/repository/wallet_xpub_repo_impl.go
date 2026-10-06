package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletXpubRepository defines the persistence contract for BIP-32 extended
// public keys associated with wallets.
type WalletXpubRepository interface {
	Create(x *models.WalletXpub) error
	Update(x *models.WalletXpub) error
	Delete(id uint) error
	GetByID(id uint) (*models.WalletXpub, error)
	GetByWalletAndAccount(walletID, accountIdx uint) (*models.WalletXpub, error)
	ListByWallet(walletID uint, opts ...QueryOption) ([]models.WalletXpub, error)
	// Deprecated: prefer ClaimNextIdx for atomic claim semantics.
	IncrementNextIdx(xpubID uint) error // atomic UPDATE next_idx = next_idx + 1
	ClaimNextIdx(xpubID uint) (uint, error)
}

// WalletXpubRepositoryImpl is the GORM-backed implementation of WalletXpubRepository.
type WalletXpubRepositoryImpl struct {
	db *gorm.DB
}

// NewWalletXpubRepository constructs a WalletXpubRepository backed by db.
func NewWalletXpubRepository(db *gorm.DB) WalletXpubRepository {
	return &WalletXpubRepositoryImpl{db: db}
}

func (r *WalletXpubRepositoryImpl) Create(x *models.WalletXpub) error { return r.db.Create(x).Error }
func (r *WalletXpubRepositoryImpl) Update(x *models.WalletXpub) error { return r.db.Save(x).Error }

func (r *WalletXpubRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.WalletXpub{}, id).Error
}

func (r *WalletXpubRepositoryImpl) GetByID(id uint) (*models.WalletXpub, error) {
	var x models.WalletXpub
	if err := r.db.First(&x, id).Error; err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *WalletXpubRepositoryImpl) GetByWalletAndAccount(walletID, accountIdx uint) (*models.WalletXpub, error) {
	var x models.WalletXpub
	err := r.db.Where("wallet_id = ? AND account_idx = ?", walletID, accountIdx).First(&x).Error
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *WalletXpubRepositoryImpl) ListByWallet(walletID uint, opts ...QueryOption) ([]models.WalletXpub, error) {
	var out []models.WalletXpub
	q := Apply(r.db.Where("wallet_id = ?", walletID), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// IncrementNextIdx atomically increments next_idx by 1 for the given xpub.
//
// Deprecated: prefer ClaimNextIdx for atomic claim semantics.
func (r *WalletXpubRepositoryImpl) IncrementNextIdx(xpubID uint) error {
	return r.db.Model(&models.WalletXpub{}).
		Where("id = ?", xpubID).
		UpdateColumn("next_idx", gorm.Expr("next_idx + 1")).Error
}

// ClaimNextIdx atomically claims the current next_idx and increments it.
// Returns the value that was current before the increment — use this as the
// derivation index to ensure concurrent callers never receive the same index.
func (r *WalletXpubRepositoryImpl) ClaimNextIdx(xpubID uint) (uint, error) {
	var claimed uint
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var x models.WalletXpub
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", xpubID).
			First(&x).Error; err != nil {
			return err
		}
		claimed = x.NextIdx
		return tx.Model(&x).Update("next_idx", x.NextIdx+1).Error
	})
	return claimed, err
}
