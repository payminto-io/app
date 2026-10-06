package models

// EntrypointSCAddress represents the ERC-4337 EntryPoint contract addresses
// used by smart contract wallets. PayRam stores one row per chain.
type EntrypointSCAddress struct {
	PaymintoModel
	BlockchainID uint   `gorm:"not null;uniqueIndex" json:"blockchainID"`
	Address      string `gorm:"type:varchar(100);not null" json:"address"`
	Version      string `gorm:"type:varchar(20)" json:"version"`
	Status       string `gorm:"type:varchar(20);default:'active';not null" json:"status"`

	Blockchain *Blockchain `gorm:"foreignKey:BlockchainID" json:"-"`
}

func (EntrypointSCAddress) TableName() string { return "entrypoint_sc_addresses" }
