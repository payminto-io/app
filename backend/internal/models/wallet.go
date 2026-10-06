package models

// Wallet is a merchant's HD hot wallet for a specific blockchain family.
// Private keys are never stored in this row — they live in SecretsVault.
type Wallet struct {
	PaymintoModel
	Name               string `gorm:"type:varchar(100);not null" json:"name"`
	Kind               string `gorm:"type:varchar(20);not null" json:"kind"`
	Status             string `gorm:"type:varchar(20);default:'active'" json:"status"`
	BlockchainFamilyID uint   `gorm:"not null" json:"blockchainFamilyID"`
	MemberID           uint   `gorm:"not null" json:"memberID"`

	BlockchainFamily *BlockchainFamily `gorm:"foreignKey:BlockchainFamilyID" json:"blockchainFamily,omitempty"`
	Member           *Member           `gorm:"foreignKey:MemberID" json:"-"`
	AddressPools     []AddressPool     `gorm:"foreignKey:WalletID" json:"-"`
}

func (Wallet) TableName() string { return "wallets" }

// AddressPool holds pre-derived deposit addresses that are ready to be
// assigned to incoming payments. Status transitions: available → used.
type AddressPool struct {
	PaymintoModel
	Address            string  `gorm:"type:text;not null;uniqueIndex" json:"address"`
	EncryptedKey       *string `gorm:"type:text" json:"-"`
	PathIndex          uint    `json:"pathIndex"`
	Status             string  `gorm:"type:varchar(20);default:'available'" json:"status"`
	WalletID           uint    `gorm:"not null" json:"walletID"`
	BlockchainFamilyID uint    `gorm:"not null" json:"blockchainFamilyID"`

	Wallet *Wallet `gorm:"foreignKey:WalletID" json:"-"`
}

func (AddressPool) TableName() string { return "address_pools" }

// DepositAddress links a specific on-chain address to a member and optionally
// to a PaymentRequest. Created when a payment is assigned an address from the pool.
type DepositAddress struct {
	PaymintoModel
	Address              string `gorm:"type:text;not null" json:"address"`
	BlockchainCurrencyID uint   `gorm:"not null" json:"blockchainCurrencyID"`
	MemberID             uint   `gorm:"not null" json:"memberID"`
	PaymentRequestID     *uint  `json:"paymentRequestID,omitempty"`

	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"blockchainCurrency,omitempty"`
}

func (DepositAddress) TableName() string { return "deposit_addresses" }
