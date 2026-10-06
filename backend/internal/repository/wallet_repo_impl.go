package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// WalletRepository defines the persistence contract for hot-wallet records.
type WalletRepository interface {
	Create(w *models.Wallet) error
	Update(w *models.Wallet) error
	Delete(id uint) error
	GetByID(id uint) (*models.Wallet, error)
	GetByMemberAndFamily(memberID, blockchainFamilyID uint) (*models.Wallet, error)
	ListByMember(memberID uint, opts ...QueryOption) ([]models.Wallet, error)
	ListByFamily(blockchainFamilyID uint, opts ...QueryOption) ([]models.Wallet, error)
	ListAll(opts ...QueryOption) ([]models.Wallet, error)
	Count() (int64, error)
}

// WalletRepositoryImpl is the GORM-backed implementation of WalletRepository.
type WalletRepositoryImpl struct {
	db *gorm.DB
}

// NewWalletRepository constructs a WalletRepository backed by db.
func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &WalletRepositoryImpl{db: db}
}

// Create inserts a new Wallet row.
func (r *WalletRepositoryImpl) Create(w *models.Wallet) error {
	return r.db.Create(w).Error
}

// Update saves all non-zero fields on the wallet (full save).
func (r *WalletRepositoryImpl) Update(w *models.Wallet) error {
	return r.db.Save(w).Error
}

// Delete soft-deletes the wallet with the given id.
func (r *WalletRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.Wallet{}, id).Error
}

// GetByID fetches a wallet by primary key, preloading BlockchainFamily.
func (r *WalletRepositoryImpl) GetByID(id uint) (*models.Wallet, error) {
	var w models.Wallet
	err := r.db.Preload("BlockchainFamily").First(&w, id).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetByMemberAndFamily returns the wallet for a given member and blockchain family.
func (r *WalletRepositoryImpl) GetByMemberAndFamily(memberID, blockchainFamilyID uint) (*models.Wallet, error) {
	var w models.Wallet
	err := r.db.
		Where("member_id = ? AND blockchain_family_id = ?", memberID, blockchainFamilyID).
		First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ListByMember returns all wallets belonging to the given member.
func (r *WalletRepositoryImpl) ListByMember(memberID uint, opts ...QueryOption) ([]models.Wallet, error) {
	var wallets []models.Wallet
	q := Apply(r.db.Where("member_id = ?", memberID), opts...)
	err := q.Find(&wallets).Error
	return wallets, err
}

// ListByFamily returns all wallets in the given blockchain family.
func (r *WalletRepositoryImpl) ListByFamily(blockchainFamilyID uint, opts ...QueryOption) ([]models.Wallet, error) {
	var wallets []models.Wallet
	q := Apply(r.db.Where("blockchain_family_id = ?", blockchainFamilyID), opts...)
	err := q.Find(&wallets).Error
	return wallets, err
}

// ListAll returns every non-deleted wallet, optionally filtered by QueryOptions.
// Used by AddressPoolWarmer to iterate over all wallets regardless of member.
func (r *WalletRepositoryImpl) ListAll(opts ...QueryOption) ([]models.Wallet, error) {
	var out []models.Wallet
	q := Apply(r.db, opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Count returns the total number of (non-deleted) wallets.
func (r *WalletRepositoryImpl) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Wallet{}).Count(&count).Error
	return count, err
}
