package models

import "github.com/shopspring/decimal"

// MissedDeposit holds incoming on-chain transactions that couldn't be matched
// to an open PaymentRequest (wrong amount, expired, address reuse). Operators
// resolve these manually via the dashboard — refund, claim as revenue, or
// link to an existing payment.
//
// C5: BlockchainCurrencyID is nullable (*uint) because when an on-chain tx
// arrives for an address that is not in the watched set we cannot determine
// which blockchain_currency it belongs to. BlockchainID (required) always
// identifies the chain.
type MissedDeposit struct {
	PaymintoModel
	TxHash               string          `gorm:"type:varchar(100);not null;index" json:"txHash"`
	BlockchainID         uint            `gorm:"not null;index" json:"blockchainId"`
	BlockchainCurrencyID *uint           `gorm:"index" json:"blockchainCurrencyId,omitempty"`
	FromAddress          string          `gorm:"type:varchar(100)" json:"fromAddress"`
	ToAddress            string          `gorm:"type:varchar(100);not null" json:"toAddress"`
	Amount               decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	BlockNumber          int64           `json:"blockNumber"`
	Reason               string          `gorm:"type:varchar(100);not null" json:"reason"`
	Status               string          `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	ResolvedAt           *string         `gorm:"type:varchar(100)" json:"resolvedAt,omitempty"`
	ResolvedBy           *uint           `json:"resolvedBy,omitempty"`
	Resolution           *string         `gorm:"type:text" json:"resolution,omitempty"`

	Blockchain         *Blockchain         `gorm:"foreignKey:BlockchainID" json:"-"`
	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"-"`
}

func (MissedDeposit) TableName() string { return "missed_deposits" }

const (
	MissedDepositStatusPending  = "pending"
	MissedDepositStatusRefunded = "refunded"
	MissedDepositStatusClaimed  = "claimed"
	MissedDepositStatusLinked   = "linked"
	MissedDepositStatusIgnored  = "ignored"
)
