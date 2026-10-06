package models

import "github.com/shopspring/decimal"

// AddressDeployment tracks the on-chain deployment of a smart contract
// deposit wallet (CREATE2 deploy). Holds the broadcast tx hash, gas cost,
// and lifecycle status.
type AddressDeployment struct {
	PaymintoModel
	BlockchainID         uint             `gorm:"not null;index" json:"blockchainID"`
	BlockchainCurrencyID *uint            `gorm:"index" json:"blockchainCurrencyID,omitempty"`
	Address              string           `gorm:"type:varchar(100);not null;index" json:"address"`
	TransactionHash      *string          `gorm:"type:varchar(100);index" json:"transactionHash,omitempty"`
	BroadcastedAt        *string          `gorm:"type:varchar(100)" json:"broadcastedAt,omitempty"`
	TransactionFee       *decimal.Decimal `gorm:"type:numeric(38,18)" json:"transactionFee,omitempty"`
	Status               string           `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	ErrorMessage         *string          `gorm:"type:text" json:"errorMessage,omitempty"`

	Blockchain         *Blockchain         `gorm:"foreignKey:BlockchainID" json:"-"`
	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"-"`
}

func (AddressDeployment) TableName() string { return "address_deployments" }
