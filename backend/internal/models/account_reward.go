package models

import "github.com/shopspring/decimal"

// AccountReward is a separate ledger account for referral rewards. PayRam
// keeps these separate from the main Account so reward balances don't mix
// with operational funds.
type AccountReward struct {
	PaymintoModel
	MemberID    uint            `gorm:"not null;index" json:"memberID"`
	CurrencyID  uint            `gorm:"not null;index" json:"currencyID"`
	Balance     decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"balance"`
	Locked      decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"locked"`
	TotalEarned decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"totalEarned"`
	TotalSpent  decimal.Decimal `gorm:"type:numeric(38,18);default:0;not null" json:"totalSpent"`

	Member   *Member   `gorm:"foreignKey:MemberID" json:"-"`
	Currency *Currency `gorm:"foreignKey:CurrencyID" json:"-"`
}

func (AccountReward) TableName() string { return "account_rewards" }
