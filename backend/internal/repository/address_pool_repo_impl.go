package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AddressPoolRepository defines the persistence contract for address-pool records.
type AddressPoolRepository interface {
	Create(a *models.AddressPool) error
	BulkCreate(addresses []models.AddressPool) error
	GetByAddress(address string) (*models.AddressPool, error)
	GetByID(id uint) (*models.AddressPool, error)
	GetAvailableByWallet(walletID uint) (*models.AddressPool, error)
	CountAvailableByWallet(walletID uint) (int64, error)
	ClaimNextAvailable(walletID uint) (*models.AddressPool, error)
	MarkUsed(id uint) error
	MarkLocked(id uint) error
	GetLastPathIndex(walletID uint) (uint, error)
	GetByWalletAndStatus(walletID uint, status string, opts ...QueryOption) ([]models.AddressPool, error)
	ListWithUnassignedKeys(blockchainFamilyID uint, limit int) ([]models.AddressPool, error)
}

// AddressPoolRepositoryImpl is the GORM-backed implementation of AddressPoolRepository.
type AddressPoolRepositoryImpl struct {
	db *gorm.DB
}

// NewAddressPoolRepository constructs an AddressPoolRepository backed by db.
func NewAddressPoolRepository(db *gorm.DB) AddressPoolRepository {
	return &AddressPoolRepositoryImpl{db: db}
}

// Create inserts a single AddressPool row.
func (r *AddressPoolRepositoryImpl) Create(a *models.AddressPool) error {
	return r.db.Create(a).Error
}

// BulkCreate inserts addresses in batches of 500.
func (r *AddressPoolRepositoryImpl) BulkCreate(addresses []models.AddressPool) error {
	return r.db.CreateInBatches(addresses, 500).Error
}

// GetByAddress fetches an address-pool record by its on-chain address.
func (r *AddressPoolRepositoryImpl) GetByAddress(address string) (*models.AddressPool, error) {
	var a models.AddressPool
	err := r.db.Where("address = ?", address).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetByID fetches an address-pool record by primary key.
func (r *AddressPoolRepositoryImpl) GetByID(id uint) (*models.AddressPool, error) {
	var a models.AddressPool
	err := r.db.First(&a, id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetAvailableByWallet returns the lowest path_index available address for a wallet.
func (r *AddressPoolRepositoryImpl) GetAvailableByWallet(walletID uint) (*models.AddressPool, error) {
	var a models.AddressPool
	err := r.db.
		Where("wallet_id = ? AND status = ?", walletID, "available").
		Order("path_index ASC").
		Limit(1).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ClaimNextAvailable atomically selects and marks used the lowest path_index
// available address for a wallet. It runs inside a single transaction with a
// SELECT ... FOR UPDATE SKIP LOCKED so concurrent callers never claim the same
// row. Returns gorm.ErrRecordNotFound when the pool is exhausted.
func (r *AddressPoolRepositoryImpl) ClaimNextAvailable(walletID uint) (*models.AddressPool, error) {
	var a models.AddressPool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("wallet_id = ? AND status = ?", walletID, "available").
			Order("path_index ASC").
			Limit(1).
			First(&a).Error; err != nil {
			return err
		}
		return tx.Model(&a).Update("status", "used").Error
	})
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CountAvailableByWallet counts rows with status='available' for a wallet.
func (r *AddressPoolRepositoryImpl) CountAvailableByWallet(walletID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.AddressPool{}).
		Where("wallet_id = ? AND status = ?", walletID, "available").
		Count(&count).Error
	return count, err
}

// MarkUsed sets status='used' on the address-pool record with the given id.
func (r *AddressPoolRepositoryImpl) MarkUsed(id uint) error {
	return r.db.Model(&models.AddressPool{}).
		Where("id = ?", id).
		Update("status", "used").Error
}

// MarkLocked sets status='locked' on the address-pool record with the given id.
func (r *AddressPoolRepositoryImpl) MarkLocked(id uint) error {
	return r.db.Model(&models.AddressPool{}).
		Where("id = ?", id).
		Update("status", "locked").Error
}

// GetLastPathIndex returns the highest path_index for a wallet, or 0 if none exist.
func (r *AddressPoolRepositoryImpl) GetLastPathIndex(walletID uint) (uint, error) {
	var result struct {
		MaxIndex *uint
	}
	err := r.db.Model(&models.AddressPool{}).
		Select("MAX(path_index) as max_index").
		Where("wallet_id = ?", walletID).
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if result.MaxIndex == nil {
		return 0, nil
	}
	return *result.MaxIndex, nil
}

// GetByWalletAndStatus returns address-pool records filtered by wallet and status.
func (r *AddressPoolRepositoryImpl) GetByWalletAndStatus(walletID uint, status string, opts ...QueryOption) ([]models.AddressPool, error) {
	var addresses []models.AddressPool
	q := Apply(
		r.db.Where("wallet_id = ? AND status = ?", walletID, status),
		opts...,
	)
	err := q.Find(&addresses).Error
	return addresses, err
}

// ListWithUnassignedKeys returns addresses that have no encrypted_key yet for a given blockchain family.
func (r *AddressPoolRepositoryImpl) ListWithUnassignedKeys(blockchainFamilyID uint, limit int) ([]models.AddressPool, error) {
	var addresses []models.AddressPool
	err := r.db.
		Where("encrypted_key IS NULL AND blockchain_family_id = ?", blockchainFamilyID).
		Limit(limit).
		Find(&addresses).Error
	return addresses, err
}
