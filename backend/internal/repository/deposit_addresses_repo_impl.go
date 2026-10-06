package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// DepositAddressRepository defines all persistence operations for DepositAddress records.
type DepositAddressRepository interface {
	Create(d *models.DepositAddress) error
	Update(d *models.DepositAddress) error
	GetByID(id uint) (*models.DepositAddress, error)
	GetByAddress(address string, blockchainCurrencyID uint) (*models.DepositAddress, error)
	GetByPaymentRequestID(paymentRequestID uint) ([]models.DepositAddress, error)
	ListByMember(memberID uint, opts ...QueryOption) ([]models.DepositAddress, error)
	ListByBlockchainCurrency(blockchainCurrencyID uint, opts ...QueryOption) ([]models.DepositAddress, error)

	// ListAllByBlockchainID returns a page of deposit addresses on the given chain,
	// joined through blockchain_currencies. The returned rows preload
	// BlockchainCurrency (including its nested Currency) so downstream code
	// can read Decimals/WalletPrecision without additional DB round-trips.
	// Used by BlockchainProcessor.refreshWatchedAddresses to build the watched
	// address map in a single paginated scan (I1 — eliminates N+1 queries).
	ListAllByBlockchainID(blockchainID uint, limit, offset int) ([]models.DepositAddress, error)
}

// DepositAddressRepositoryImpl is the GORM-backed implementation of DepositAddressRepository.
type DepositAddressRepositoryImpl struct {
	db *gorm.DB
}

// NewDepositAddressRepository constructs a new DepositAddressRepository backed by the provided *gorm.DB.
func NewDepositAddressRepository(db *gorm.DB) DepositAddressRepository {
	return &DepositAddressRepositoryImpl{db: db}
}

// Create inserts a new DepositAddress record.
func (r *DepositAddressRepositoryImpl) Create(d *models.DepositAddress) error {
	return r.db.Create(d).Error
}

// Update saves all fields of the given DepositAddress.
func (r *DepositAddressRepositoryImpl) Update(d *models.DepositAddress) error {
	return r.db.Save(d).Error
}

// GetByID fetches a DepositAddress by primary key.
func (r *DepositAddressRepositoryImpl) GetByID(id uint) (*models.DepositAddress, error) {
	var da models.DepositAddress
	if err := r.db.First(&da, id).Error; err != nil {
		return nil, err
	}
	return &da, nil
}

// GetByAddress fetches a DepositAddress by its on-chain address and blockchain
// currency, preloading BlockchainCurrency. The address comparison is
// case-insensitive (LOWER) to handle EVM EIP-55 checksum vs lowercase
// normalisation — the same address stored in checksum form in the DB will be
// found when looked up in lowercase form (C6).
func (r *DepositAddressRepositoryImpl) GetByAddress(address string, blockchainCurrencyID uint) (*models.DepositAddress, error) {
	var da models.DepositAddress
	err := r.db.
		Preload("BlockchainCurrency").
		Where("LOWER(address) = LOWER(?) AND blockchain_currency_id = ?", address, blockchainCurrencyID).
		First(&da).Error
	if err != nil {
		return nil, err
	}
	return &da, nil
}

// GetByPaymentRequestID returns all DepositAddresses associated with a payment request.
func (r *DepositAddressRepositoryImpl) GetByPaymentRequestID(paymentRequestID uint) ([]models.DepositAddress, error) {
	var addrs []models.DepositAddress
	if err := r.db.
		Preload("BlockchainCurrency").
		Preload("BlockchainCurrency.Blockchain").
		Preload("BlockchainCurrency.Currency").
		Where("payment_request_id = ?", paymentRequestID).
		Find(&addrs).Error; err != nil {
		return nil, err
	}
	return addrs, nil
}

// ListByMember returns all DepositAddresses belonging to a member,
// applying any provided QueryOptions.
func (r *DepositAddressRepositoryImpl) ListByMember(memberID uint, opts ...QueryOption) ([]models.DepositAddress, error) {
	q := Apply(r.db.Where("member_id = ?", memberID), opts...)
	var addrs []models.DepositAddress
	if err := q.Find(&addrs).Error; err != nil {
		return nil, err
	}
	return addrs, nil
}

// ListByBlockchainCurrency returns all DepositAddresses for a given blockchain
// currency, applying any provided QueryOptions.
func (r *DepositAddressRepositoryImpl) ListByBlockchainCurrency(blockchainCurrencyID uint, opts ...QueryOption) ([]models.DepositAddress, error) {
	q := Apply(r.db.Where("blockchain_currency_id = ?", blockchainCurrencyID), opts...)
	var addrs []models.DepositAddress
	if err := q.Find(&addrs).Error; err != nil {
		return nil, err
	}
	return addrs, nil
}

// ListAllByBlockchainID returns a paginated page of DepositAddresses on the given
// chain. It joins deposit_addresses → blockchain_currencies filtered by
// blockchain_id, preloading BlockchainCurrency and its nested Currency so
// WalletPrecision (decimals) and MinDepositAmount are available without
// additional DB round-trips (I1).
func (r *DepositAddressRepositoryImpl) ListAllByBlockchainID(blockchainID uint, limit, offset int) ([]models.DepositAddress, error) {
	var addrs []models.DepositAddress
	err := r.db.
		Joins("JOIN blockchain_currencies ON blockchain_currencies.id = deposit_addresses.blockchain_currency_id").
		Where("blockchain_currencies.blockchain_id = ? AND blockchain_currencies.deleted_at IS NULL", blockchainID).
		Preload("BlockchainCurrency").
		Preload("BlockchainCurrency.Currency").
		Limit(limit).
		Offset(offset).
		Find(&addrs).Error
	if err != nil {
		return nil, err
	}
	return addrs, nil
}
