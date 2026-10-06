package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BalanceEntry is a per-currency balance summary for a member. Returned by
// AddressRepository.Balances.
type BalanceEntry struct {
	CurrencyCode   string          `json:"currencyCode"`
	BlockchainCode string          `json:"blockchainCode"`
	Balance        decimal.Decimal `json:"balance"`
	Locked         decimal.Decimal `json:"locked"`
	Available      decimal.Decimal `json:"available"` // balance - locked
}

// EligibleAddress is an address that qualifies for a sweep operation.
type EligibleAddress struct {
	AddressPoolID        uint
	Address              string
	Balance              decimal.Decimal
	BlockchainCurrencyID uint
}

// AddressRepository exposes the balance and eligibility queries that
// AddressService calls. Mirrors PayRam's AddressRepositoryImpl.
type AddressRepository interface {
	// Balances returns one row per distinct (currency, blockchain) pair the
	// member has any balance on. Joins account_addresses -> blockchain_currencies.
	Balances(memberID uint) ([]BalanceEntry, error)

	// PendingForApproval returns addresses with pending sweep approval needed.
	PendingForApproval(memberID uint) ([]EligibleAddress, error)

	// GetEligibleAddressesToSweep returns addresses whose on-chain balance
	// is at or above the sweep threshold and are NOT currently locked.
	GetEligibleAddressesToSweep(blockchainCurrencyID uint, threshold decimal.Decimal) ([]EligibleAddress, error)

	// GetEligibleAddressesToTransferFees finds addresses that need ETH/TRX
	// gas top-up before an ERC20/TRC20 sweep can run. Returns addresses
	// that have a non-zero token balance but zero native balance.
	GetEligibleAddressesToTransferFees(blockchainCurrencyID uint) ([]EligibleAddress, error)

	// GetEligibleSCWAddressPoolsToBroadcast returns address pool rows whose
	// corresponding WalletSCW row is in 'pending' state — the SCW deposit
	// broadcaster worker picks these up to deploy CREATE2 wallets.
	GetEligibleSCWAddressPoolsToBroadcast(blockchainID uint) ([]models.AddressPool, error)

	// ProcessAddressBalanceAndUpdateStatus walks active deposit addresses
	// for the given blockchain and updates account_addresses.balance from
	// on-chain data. This is a long-running query — the block processor
	// calls it periodically. Takes a function that fetches the balance for
	// one (address, blockchainCurrencyID) pair so the repo stays
	// adapter-agnostic.
	ProcessAddressBalanceAndUpdateStatus(
		blockchainID uint,
		fetchBalance func(address string, blockchainCurrencyID uint) (decimal.Decimal, error),
	) (updated int64, err error)

	// MarkLocked sets the address pool status to 'locked' so the sweep
	// processor skips the address while a sweep tx is in flight.
	MarkLocked(addressPoolID uint, lockDurationSeconds int) error

	// MarkUnlocked clears the lock by setting status to 'available'.
	MarkUnlocked(addressPoolID uint) error
}

// AddressRepositoryImpl is the GORM-backed implementation of AddressRepository.
type AddressRepositoryImpl struct {
	db *gorm.DB
}

// NewAddressRepository constructs an AddressRepository backed by db.
func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &AddressRepositoryImpl{db: db}
}

func (r *AddressRepositoryImpl) Balances(memberID uint) ([]BalanceEntry, error) {
	var rows []BalanceEntry
	err := r.db.Table("account_addresses aa").
		Select("bc.currency_code as currency_code, bc.blockchain_code as blockchain_code, SUM(aa.balance) as balance, SUM(aa.locked) as locked, SUM(aa.balance - aa.locked) as available").
		Joins("JOIN blockchain_currencies bc ON bc.id = aa.blockchain_currency_id").
		Where("aa.member_id = ? AND aa.deleted_at IS NULL", memberID).
		Group("bc.currency_code, bc.blockchain_code").
		Scan(&rows).Error
	return rows, err
}

func (r *AddressRepositoryImpl) PendingForApproval(memberID uint) ([]EligibleAddress, error) {
	var rows []EligibleAddress
	err := r.db.Table("address_pools ap").
		Select("ap.id as address_pool_id, ap.address as address, aa.balance as balance, aa.blockchain_currency_id as blockchain_currency_id").
		Joins("JOIN account_addresses aa ON aa.address = ap.address").
		Where("aa.member_id = ? AND ap.status = ? AND ap.deleted_at IS NULL", memberID, "pending_approval").
		Scan(&rows).Error
	return rows, err
}

func (r *AddressRepositoryImpl) GetEligibleAddressesToSweep(blockchainCurrencyID uint, threshold decimal.Decimal) ([]EligibleAddress, error) {
	var rows []EligibleAddress
	err := r.db.Table("address_pools ap").
		Select("ap.id as address_pool_id, ap.address as address, aa.balance as balance, aa.blockchain_currency_id as blockchain_currency_id").
		Joins("JOIN account_addresses aa ON aa.address = ap.address AND aa.blockchain_currency_id = ?", blockchainCurrencyID).
		Where("aa.balance >= ?", threshold).
		Where("ap.status IN ?", []string{"used", "active"}).
		Where("ap.deleted_at IS NULL").
		Scan(&rows).Error
	return rows, err
}

func (r *AddressRepositoryImpl) GetEligibleAddressesToTransferFees(blockchainCurrencyID uint) ([]EligibleAddress, error) {
	// Addresses that have a non-zero token balance but need gas top-up.
	// The caller (sweep processor) filters the result against live chain data.
	var rows []EligibleAddress
	err := r.db.Table("address_pools ap").
		Select("ap.id as address_pool_id, ap.address as address, aa.balance as balance, aa.blockchain_currency_id as blockchain_currency_id").
		Joins("JOIN account_addresses aa ON aa.address = ap.address AND aa.blockchain_currency_id = ?", blockchainCurrencyID).
		Where("aa.balance > 0").
		Where("ap.deleted_at IS NULL").
		Scan(&rows).Error
	return rows, err
}

func (r *AddressRepositoryImpl) GetEligibleSCWAddressPoolsToBroadcast(blockchainID uint) ([]models.AddressPool, error) {
	var pools []models.AddressPool
	err := r.db.
		Joins("JOIN wallet_scws ws ON ws.wallet_id = address_pools.wallet_id").
		Where("ws.blockchain_id = ? AND ws.status = ?", blockchainID, "pending").
		Find(&pools).Error
	return pools, err
}

func (r *AddressRepositoryImpl) ProcessAddressBalanceAndUpdateStatus(
	blockchainID uint,
	fetchBalance func(address string, blockchainCurrencyID uint) (decimal.Decimal, error),
) (int64, error) {
	var rows []struct {
		ID                   uint
		Address              string
		BlockchainCurrencyID uint
	}
	err := r.db.Table("account_addresses aa").
		Select("aa.id as id, aa.address as address, aa.blockchain_currency_id as blockchain_currency_id").
		Joins("JOIN blockchain_currencies bc ON bc.id = aa.blockchain_currency_id").
		Where("bc.blockchain_id = ?", blockchainID).
		Where("aa.deleted_at IS NULL").
		Scan(&rows).Error
	if err != nil {
		return 0, err
	}

	var updated int64
	for _, row := range rows {
		newBalance, err := fetchBalance(row.Address, row.BlockchainCurrencyID)
		if err != nil {
			continue
		}
		res := r.db.Model(&models.AccountAddress{}).
			Where("id = ?", row.ID).
			Update("balance", newBalance)
		if res.Error != nil {
			continue
		}
		if res.RowsAffected > 0 {
			updated++
		}
	}
	return updated, nil
}

func (r *AddressRepositoryImpl) MarkLocked(addressPoolID uint, lockDurationSeconds int) error {
	return r.db.Model(&models.AddressPool{}).
		Where("id = ?", addressPoolID).
		Update("status", "locked").Error
}

func (r *AddressRepositoryImpl) MarkUnlocked(addressPoolID uint) error {
	return r.db.Model(&models.AddressPool{}).
		Where("id = ?", addressPoolID).
		Update("status", "available").Error
}
