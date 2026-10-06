package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// WalletSCWRepository defines the persistence contract for Smart Contract Wallet records.
type WalletSCWRepository interface {
	Create(s *models.WalletSCW) error
	Update(s *models.WalletSCW) error
	GetByID(id uint) (*models.WalletSCW, error)
	GetByWalletAndBlockchain(walletID, blockchainID uint) (*models.WalletSCW, error)
	GetByContractAddress(address string, blockchainID uint) (*models.WalletSCW, error)
	ListByWallet(walletID uint) ([]models.WalletSCW, error)
	ListPending(blockchainID uint) ([]models.WalletSCW, error)
	MarkDeployed(id uint, contractAddress, txHash string) error
}

// WalletSCWRepositoryImpl is the GORM-backed implementation of WalletSCWRepository.
type WalletSCWRepositoryImpl struct {
	db *gorm.DB
}

// NewWalletSCWRepository constructs a WalletSCWRepository backed by db.
func NewWalletSCWRepository(db *gorm.DB) WalletSCWRepository {
	return &WalletSCWRepositoryImpl{db: db}
}

// Create inserts a new WalletSCW row.
func (r *WalletSCWRepositoryImpl) Create(s *models.WalletSCW) error { return r.db.Create(s).Error }

// Update saves all fields on the SCW (full save).
func (r *WalletSCWRepositoryImpl) Update(s *models.WalletSCW) error { return r.db.Save(s).Error }

// GetByID fetches a WalletSCW by primary key.
func (r *WalletSCWRepositoryImpl) GetByID(id uint) (*models.WalletSCW, error) {
	var s models.WalletSCW
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByWalletAndBlockchain fetches the SCW for a specific wallet and blockchain pair.
func (r *WalletSCWRepositoryImpl) GetByWalletAndBlockchain(walletID, blockchainID uint) (*models.WalletSCW, error) {
	var s models.WalletSCW
	err := r.db.Where("wallet_id = ? AND blockchain_id = ?", walletID, blockchainID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByContractAddress fetches the SCW by its deployed contract address on a specific blockchain.
func (r *WalletSCWRepositoryImpl) GetByContractAddress(address string, blockchainID uint) (*models.WalletSCW, error) {
	var s models.WalletSCW
	err := r.db.Where("contract_address = ? AND blockchain_id = ?", address, blockchainID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListByWallet returns all SCW records for a given wallet.
func (r *WalletSCWRepositoryImpl) ListByWallet(walletID uint) ([]models.WalletSCW, error) {
	var out []models.WalletSCW
	err := r.db.Where("wallet_id = ?", walletID).Find(&out).Error
	return out, err
}

// ListPending returns all SCW records in 'pending' status for a given blockchain.
func (r *WalletSCWRepositoryImpl) ListPending(blockchainID uint) ([]models.WalletSCW, error) {
	var out []models.WalletSCW
	err := r.db.Where("blockchain_id = ? AND status = ?", blockchainID, "pending").Find(&out).Error
	return out, err
}

// MarkDeployed updates the SCW record with contract address, tx hash, and sets status to 'deployed'.
func (r *WalletSCWRepositoryImpl) MarkDeployed(id uint, contractAddress, txHash string) error {
	return r.db.Model(&models.WalletSCW{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"contract_address": contractAddress,
			"deployed_at":      txHash,
			"status":           "deployed",
		}).Error
}
