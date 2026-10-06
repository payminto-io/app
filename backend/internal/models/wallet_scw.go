package models

// WalletSCW (Smart Contract Wallet) tracks deployed account-abstraction
// wallets per blockchain. Used by PayRam's CREATE2 deposit-address pattern.
type WalletSCW struct {
	PaymintoModel
	WalletID        uint    `gorm:"not null;index" json:"walletID"`
	BlockchainID    uint    `gorm:"not null;index" json:"blockchainID"`
	OwnerAddress    string  `gorm:"type:varchar(100);not null" json:"ownerAddress"`
	ContractAddress *string `gorm:"type:varchar(100)" json:"contractAddress,omitempty"`
	FactoryAddress  string  `gorm:"type:varchar(100);not null" json:"factoryAddress"`
	Salt            string  `gorm:"type:varchar(100);not null" json:"salt"`
	Status          string  `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	DeployedAt      *string `gorm:"type:varchar(100)" json:"deployedTxHash,omitempty"`

	Wallet     *Wallet     `gorm:"foreignKey:WalletID" json:"-"`
	Blockchain *Blockchain `gorm:"foreignKey:BlockchainID" json:"-"`
}

func (WalletSCW) TableName() string { return "wallet_scws" }
