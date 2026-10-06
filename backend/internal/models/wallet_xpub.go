package models

// WalletXpub holds a BIP-32 extended public key for a Wallet, allowing
// derivation of new deposit addresses without exposing the master private
// key. PayRam uses this to pre-generate address pools.
type WalletXpub struct {
	PaymintoModel
	WalletID   uint   `gorm:"not null;index" json:"walletID"`
	Xpub       string `gorm:"type:text;not null" json:"xpub"`
	Path       string `gorm:"type:varchar(100);not null" json:"path"`
	AccountIdx uint   `gorm:"not null" json:"accountIdx"`
	NextIdx    uint   `gorm:"default:0;not null" json:"nextIdx"`
	Status     string `gorm:"type:varchar(20);default:'active';not null" json:"status"`

	Wallet *Wallet `gorm:"foreignKey:WalletID" json:"-"`
}

func (WalletXpub) TableName() string { return "wallet_xpubs" }
