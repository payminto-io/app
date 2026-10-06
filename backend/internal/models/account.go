package models

import "github.com/shopspring/decimal"

// Account represents the double-entry ledger account for a member and currency pair,
// tracking spendable balance and locked (in-flight) funds.
type Account struct {
	PaymintoModel
	Balance    decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"balance"`
	Locked     decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"locked"`
	CurrencyID uint            `gorm:"not null" json:"currencyID"`
	MemberID   uint            `gorm:"not null" json:"memberID"`

	Currency *Currency `gorm:"foreignKey:CurrencyID" json:"currency,omitempty"`
	Member   *Member   `gorm:"foreignKey:MemberID" json:"-"`
}

func (Account) TableName() string { return "accounts" }

// AccountAddress tracks the on-chain balance of a deposit address for a
// specific blockchain currency and member.
type AccountAddress struct {
	PaymintoModel
	Balance              decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"balance"`
	Locked               decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"locked"`
	Address              string          `gorm:"type:text;not null" json:"address"`
	BlockchainCurrencyID uint            `gorm:"not null" json:"blockchainCurrencyID"`
	MemberID             uint            `gorm:"not null" json:"memberID"`

	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"blockchainCurrency,omitempty"`
}

func (AccountAddress) TableName() string { return "account_addresses" }

// Asset is a double-entry ledger entry representing an asset (debit-normal) line item.
type Asset struct {
	PaymintoModel
	Code        string          `gorm:"type:varchar(50);not null" json:"code"`
	Debit       decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"debit"`
	Credit      decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"credit"`
	CurrencyID  uint            `gorm:"not null" json:"currencyID"`
	ReferenceID *uint           `json:"referenceID,omitempty"`
	Reference   string          `gorm:"type:varchar(50)" json:"reference"`
}

func (Asset) TableName() string { return "assets" }

// Liability is a double-entry ledger entry representing a liability (credit-normal) line item.
type Liability struct {
	PaymintoModel
	Code        string          `gorm:"type:varchar(50);not null" json:"code"`
	Debit       decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"debit"`
	Credit      decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"credit"`
	CurrencyID  uint            `gorm:"not null" json:"currencyID"`
	ReferenceID *uint           `json:"referenceID,omitempty"`
	Reference   string          `gorm:"type:varchar(50)" json:"reference"`
}

func (Liability) TableName() string { return "liabilities" }

// Revenue is a double-entry ledger entry recording income recognized by the platform.
type Revenue struct {
	PaymintoModel
	Code        string          `gorm:"type:varchar(50);not null" json:"code"`
	Debit       decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"debit"`
	Credit      decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"credit"`
	CurrencyID  uint            `gorm:"not null" json:"currencyID"`
	ReferenceID *uint           `json:"referenceID,omitempty"`
	Reference   string          `gorm:"type:varchar(50)" json:"reference"`
}

func (Revenue) TableName() string { return "revenues" }

// Expense is a double-entry ledger entry recording costs incurred by the platform.
type Expense struct {
	PaymintoModel
	Code        string          `gorm:"type:varchar(50);not null" json:"code"`
	Debit       decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"debit"`
	Credit      decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"credit"`
	CurrencyID  uint            `gorm:"not null" json:"currencyID"`
	ReferenceID *uint           `json:"referenceID,omitempty"`
	Reference   string          `gorm:"type:varchar(50)" json:"reference"`
}

func (Expense) TableName() string { return "expenses" }
