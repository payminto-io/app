package models

import "github.com/shopspring/decimal"

// InternalBlockchainTransaction records gas-fee transfers and other internal
// moves between Payminto-controlled addresses (e.g., funding a deposit
// address with gas before sweeping ERC-20 tokens out). Used by the
// double-entry ledger to record gas as expenses.
type InternalBlockchainTransaction struct {
	PaymintoModel
	TxHash               string          `gorm:"type:varchar(100);not null;index" json:"txHash"`
	BlockchainCurrencyID uint            `gorm:"not null;index" json:"blockchainCurrencyID"`
	FromAddress          string          `gorm:"type:varchar(100);not null" json:"fromAddress"`
	ToAddress            string          `gorm:"type:varchar(100);not null" json:"toAddress"`
	Amount               decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	GasFee               decimal.Decimal `gorm:"type:numeric(38,18);default:0" json:"gasFee"`
	Purpose              string          `gorm:"type:varchar(50);not null" json:"purpose"`
	Status               string          `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	BlockNumber          *int64          `json:"blockNumber,omitempty"`

	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"-"`
}

func (InternalBlockchainTransaction) TableName() string { return "internal_blockchain_transactions" }
