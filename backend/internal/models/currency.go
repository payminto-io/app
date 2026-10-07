package models

import "github.com/shopspring/decimal"

type Currency struct {
	PaymintoModel
	Name              string           `gorm:"type:varchar(100);not null" json:"name"`
	Code              string           `gorm:"type:varchar(20);not null;uniqueIndex" json:"code"`
	Description       *string          `gorm:"type:text" json:"description,omitempty"`
	Type              string           `gorm:"type:varchar(20);not null" json:"type"`
	Visible           bool             `gorm:"default:true" json:"visible"`
	DepositEnabled    bool             `gorm:"default:true" json:"depositEnabled"`
	WithdrawalEnabled bool             `gorm:"default:true" json:"withdrawalEnabled"`
	WalletPrecision   uint             `gorm:"default:18" json:"walletPrecision"`
	BasePrecision     uint32           `gorm:"default:18" json:"basePrecision"`
	IconURL           *string          `gorm:"type:text" json:"iconURL,omitempty"`
	Price             *decimal.Decimal `gorm:"type:numeric(38,18)" json:"price,omitempty"`
}

func (Currency) TableName() string { return "currencies" }

// BlockchainCurrency maps a currency to a blockchain with chain-specific configuration.
// PayRam columns: address, standard, currency_code, blockchain_code, deposit_fee,
// min_deposit_amount, min_collection_amount, withdraw_fee, min_withdraw_amount,
// withdraw_limit24hr, withdraw_limit72hr, visible, deposit_enabled, withdrawal_enabled,
// wallet_precision, abi, currency_id, blockchain_id, approval_fee_amount
type BlockchainCurrency struct {
	PaymintoModel
	Address               string           `gorm:"type:text" json:"address"`
	Standard              string           `gorm:"type:varchar(20)" json:"standard"`
	CurrencyCode          string           `gorm:"type:varchar(20);not null" json:"currencyCode"`
	BlockchainCode        string           `gorm:"type:varchar(20);not null" json:"blockchainCode"`
	DepositFee            *decimal.Decimal `gorm:"type:numeric(38,18)" json:"depositFee,omitempty"`
	MinDepositAmount      *decimal.Decimal `gorm:"type:numeric(38,18)" json:"minDepositAmount,omitempty"`
	MinCollectionAmount   *decimal.Decimal `gorm:"type:numeric(38,18)" json:"minCollectionAmount,omitempty"`
	WithdrawFee           *decimal.Decimal `gorm:"type:numeric(38,18)" json:"withdrawFee,omitempty"`
	MinWithdrawAmount     *decimal.Decimal `gorm:"type:numeric(38,18)" json:"minWithdrawAmount,omitempty"`
	WithdrawLimit24hr     *decimal.Decimal `gorm:"type:numeric(38,18)" json:"withdrawLimit24hr,omitempty"` // PayRam: 24h withdrawal limit
	WithdrawLimit72hr     *decimal.Decimal `gorm:"type:numeric(38,18)" json:"withdrawLimit72hr,omitempty"` // PayRam: 72h withdrawal limit
	ApprovalFeeAmount     *decimal.Decimal `gorm:"type:numeric(38,18)" json:"approvalFeeAmount,omitempty"` // PayRam: ERC20 approval gas cost
	Visible               bool             `gorm:"default:true" json:"visible"`
	DepositEnabled        bool             `gorm:"default:true" json:"depositEnabled"`
	WithdrawalEnabled     bool             `gorm:"default:true" json:"withdrawalEnabled"`
	WalletPrecision       uint             `gorm:"default:18" json:"walletPrecision"`
	MinBalanceForSweep    *decimal.Decimal `gorm:"type:numeric(38,18)" json:"minBalanceForSweep,omitempty"`
	SweepBatchSize        *int             `json:"sweepBatchSize,omitempty"`
	SweepMaxWaitTimeInMin *uint            `json:"sweepMaxWaitTimeInMinutes,omitempty"`
	ABI                   *string          `gorm:"type:text" json:"-"`

	CurrencyID   uint `gorm:"not null" json:"currencyID"`
	BlockchainID uint `gorm:"not null" json:"blockchainID"`

	Currency   *Currency   `gorm:"foreignKey:CurrencyID" json:"currency,omitempty"`
	Blockchain *Blockchain `gorm:"foreignKey:BlockchainID" json:"blockchain,omitempty"`
}

func (BlockchainCurrency) TableName() string { return "blockchain_currencies" }
