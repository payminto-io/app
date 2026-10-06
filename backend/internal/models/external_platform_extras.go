package models

import "github.com/shopspring/decimal"

// ExternalPlatformBlockchainCurrency is the per-project per-token enable
// list. Lets a project decide "we accept USDC on Base but not Polygon" and
// configure per-token withdrawal limits.
type ExternalPlatformBlockchainCurrency struct {
	PaymintoModel
	ExternalPlatformID   uint             `gorm:"not null;index" json:"externalPlatformID"`
	BlockchainCurrencyID uint             `gorm:"not null;index" json:"blockchainCurrencyID"`
	Enabled              bool             `gorm:"default:true" json:"enabled"`
	AutoApproveThreshold *decimal.Decimal `gorm:"type:numeric(38,18)" json:"autoApproveThreshold,omitempty"`
	HourlyCap            *decimal.Decimal `gorm:"type:numeric(38,18)" json:"hourlyCap,omitempty"`
	DailyCap             *decimal.Decimal `gorm:"type:numeric(38,18)" json:"dailyCap,omitempty"`
	MinAmount            *decimal.Decimal `gorm:"type:numeric(38,18)" json:"minAmount,omitempty"`
	MaxAmount            *decimal.Decimal `gorm:"type:numeric(38,18)" json:"maxAmount,omitempty"`

	ExternalPlatform   *ExternalPlatform   `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"-"`
}

func (ExternalPlatformBlockchainCurrency) TableName() string {
	return "external_platform_blockchain_currencies"
}

// ExternalPlatformWalletBlockchainFamily maps an external platform's wallet
// to a blockchain family. PayRam needs this so each project can have its own
// HD wallet per family.
type ExternalPlatformWalletBlockchainFamily struct {
	PaymintoModel
	ExternalPlatformID uint `gorm:"not null;index" json:"externalPlatformID"`
	WalletID           uint `gorm:"not null;index" json:"walletID"`
	BlockchainFamilyID uint `gorm:"not null;index" json:"blockchainFamilyID"`
	IsPrimary          bool `gorm:"default:false" json:"isPrimary"`

	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	Wallet           *Wallet           `gorm:"foreignKey:WalletID" json:"-"`
	BlockchainFamily *BlockchainFamily `gorm:"foreignKey:BlockchainFamilyID" json:"-"`
}

func (ExternalPlatformWalletBlockchainFamily) TableName() string {
	return "external_platform_wallet_blockchain_families"
}
