package models

import "github.com/shopspring/decimal"

// Sweep is a batch of SmartSweep transactions that consolidate confirmed
// deposits to the cold wallet. One sweep may contain many SweepTransactions.
type Sweep struct {
	PaymintoModel
	Status       string          `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	TotalAmount  decimal.Decimal `gorm:"type:numeric(38,18)" json:"totalAmount"`
	TotalGasFee  decimal.Decimal `gorm:"type:numeric(38,18)" json:"totalGasFee"`
	BlockchainID uint            `gorm:"not null" json:"blockchainID"`

	Blockchain        *Blockchain        `gorm:"foreignKey:BlockchainID" json:"blockchain,omitempty"`
	SweepTransactions []SweepTransaction `gorm:"foreignKey:SweepID" json:"sweepTransactions,omitempty"`
}

func (Sweep) TableName() string { return "sweeps" }

// SweepTransaction is an individual on-chain transfer within a Sweep batch,
// moving funds from one deposit address to the cold wallet.
type SweepTransaction struct {
	PaymintoModel
	TxHash               string          `gorm:"type:text" json:"txHash"`
	Amount               decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	GasFee               decimal.Decimal `gorm:"type:numeric(38,18)" json:"gasFee"`
	FromAddress          string          `gorm:"type:text;not null" json:"fromAddress"`
	ToAddress            string          `gorm:"type:text;not null" json:"toAddress"`
	Status               string          `gorm:"type:varchar(20);default:'pending'" json:"status"`
	SweepID              uint            `gorm:"not null" json:"sweepID"`
	BlockchainCurrencyID uint            `gorm:"not null" json:"blockchainCurrencyID"`

	Sweep              *Sweep              `gorm:"foreignKey:SweepID" json:"-"`
	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"-"`
}

func (SweepTransaction) TableName() string { return "sweep_transactions" }

// UTXO tracks an unspent transaction output for Bitcoin deposit monitoring.
// Spent is flipped to true when the output is consumed by a sweep transaction.
type UTXO struct {
	PaymintoModel
	TxID      string          `gorm:"type:text;not null" json:"txID"`
	Vout      uint            `json:"vout"`
	Amount    decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	Address   string          `gorm:"type:text;not null" json:"address"`
	Spent     bool            `gorm:"default:false" json:"spent"`
	SpentTxID *string         `gorm:"type:text" json:"spentTxID,omitempty"`
	DepositID *uint           `json:"depositID,omitempty"`

	Deposit *Deposit `gorm:"foreignKey:DepositID" json:"-"`
}

func (UTXO) TableName() string { return "utxos" }
