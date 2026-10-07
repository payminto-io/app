package models

// BlockchainFamily groups related blockchains (e.g. "ETH_Family" covers ETH, Base, Polygon).
// Wallets are created per family, not per individual chain.
// PayRam columns: id, family, path, gas_path, supports_hd_wallet, supports_sc_wallet
type BlockchainFamily struct {
	PaymintoModel
	Name             string `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Code             string `gorm:"type:varchar(20);not null;uniqueIndex" json:"code"`
	Family           string `gorm:"type:varchar(50)" json:"family"`        // PayRam: ETH_Family, BTC_Family, TRX_Family
	Path             string `gorm:"type:varchar(100)" json:"path"`         // HD derivation path template e.g. m/44'/60'/0'/0/%d
	GasPath          string `gorm:"type:varchar(100)" json:"gasPath"`      // Gas derivation path (if different)
	SupportsHDWallet bool   `gorm:"default:false" json:"supportsHDWallet"` // BTC uses HD, EVM uses SCW
	SupportsSCWallet bool   `gorm:"default:false" json:"supportsSCWallet"` // EVM uses smart contract wallets

	Blockchains []Blockchain `gorm:"foreignKey:BlockchainFamilyID" json:"blockchains,omitempty"`
}

func (BlockchainFamily) TableName() string { return "blockchain_families" }

// Blockchain represents a single network (e.g. Ethereum Sepolia, Base) within a
// BlockchainFamily. Tracks the current block height and explorer URL templates.
// PayRam columns: id, code, name, family, client, height, height_timestamp,
// explorer_address, explorer_transaction, min_confirmations, chain_id, status,
// is_scw, blockchain_family_id, is_create2_supported
type Blockchain struct {
	PaymintoModel
	Code               string  `gorm:"type:varchar(20);not null;uniqueIndex" json:"code"`
	Name               string  `gorm:"type:varchar(100);not null" json:"name"`
	Family             string  `gorm:"type:varchar(50)" json:"family"` // PayRam: ETH_Family, BTC_Family, TRX_Family
	Client             string  `gorm:"type:varchar(50)" json:"client"` // PayRam: geth, bitcoin_core, trongrid
	BlockchainFamilyID uint    `gorm:"not null" json:"blockchainFamilyID"`
	Height             int64   `gorm:"default:0" json:"height"`
	HeightTimestamp    *string `gorm:"type:timestamptz" json:"heightTimestamp,omitempty"` // Last block timestamp
	Status             string  `gorm:"type:varchar(20);default:'active'" json:"status"`
	MinConfirmations   int     `gorm:"default:12" json:"minConfirmations"`
	ChainID            *int64  `gorm:"type:bigint" json:"chainID,omitempty"`    // EVM chain ID (11155111 for Sepolia, etc.)
	IsSCW              bool    `gorm:"default:false" json:"isSCW"`              // Supports smart contract wallets
	IsCreate2Supported bool    `gorm:"default:false" json:"isCreate2Supported"` // Supports CREATE2 deterministic deployment
	ExplorerTx         string  `gorm:"type:text" json:"explorerTx"`
	ExplorerAddress    string  `gorm:"type:text" json:"explorerAddress"`

	BlockchainFamily *BlockchainFamily `gorm:"foreignKey:BlockchainFamilyID" json:"blockchainFamily,omitempty"`
}

func (Blockchain) TableName() string { return "blockchains" }

// BlockchainContract stores the ABI and type of a smart contract deployed on a
// specific blockchain (e.g. SmartSweep factory, ERC-20 proxy).
type BlockchainContract struct {
	PaymintoModel
	BlockchainID uint   `gorm:"not null" json:"blockchainID"`
	ContractType string `gorm:"type:varchar(50);not null" json:"contractType"`
	ABI          string `gorm:"type:text" json:"-"`

	Blockchain        *Blockchain       `gorm:"foreignKey:BlockchainID" json:"blockchain,omitempty"`
	ContractAddresses []ContractAddress `gorm:"foreignKey:BlockchainContractID" json:"contractAddresses,omitempty"`
}

func (BlockchainContract) TableName() string { return "blockchain_contracts" }

// ContractAddress is a deployed instance of a BlockchainContract at a specific
// on-chain address on a given network.
type ContractAddress struct {
	PaymintoModel
	BlockchainContractID uint   `gorm:"not null" json:"blockchainContractID"`
	Address              string `gorm:"type:text;not null" json:"address"`
	Status               string `gorm:"type:varchar(20);default:'active'" json:"status"`
}

func (ContractAddress) TableName() string { return "contract_addresses" }
