package models

// AddressContractSignature stores ECDSA signatures used to authorize a
// smart-contract wallet's CREATE2 deployment or upgrade. PayRam queues
// signatures so the SCW broadcaster worker can spend them in batches.
type AddressContractSignature struct {
	PaymintoModel
	BlockchainContractID uint    `gorm:"not null;index" json:"blockchainContractID"`
	Address              string  `gorm:"type:varchar(100);not null;index" json:"address"`
	PathIndex            uint    `gorm:"not null" json:"pathIndex"`
	ContractType         string  `gorm:"type:varchar(50);not null" json:"contractType"`
	Signature            string  `gorm:"type:text;not null" json:"signature"`
	Status               string  `gorm:"type:varchar(20);default:'pending';not null" json:"status"`

	BlockchainContract *BlockchainContract `gorm:"foreignKey:BlockchainContractID" json:"-"`
}

func (AddressContractSignature) TableName() string { return "address_contract_signatures" }
