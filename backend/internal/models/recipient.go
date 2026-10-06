package models

// Recipient is a saved payee in a merchant's address book. Used to populate
// the withdrawal form with frequently-used addresses and validate against a
// known set.
type Recipient struct {
	PaymintoModel
	MemberID           uint    `gorm:"not null;index" json:"memberID"`
	ExternalPlatformID uint    `gorm:"not null;index" json:"externalPlatformID"`
	Name               string  `gorm:"type:varchar(100);not null" json:"name"`
	Email              *string `gorm:"type:text" json:"email,omitempty"`
	BlockchainCode     string  `gorm:"type:varchar(20);not null" json:"blockchainCode"`
	CurrencyCode       string  `gorm:"type:varchar(20);not null" json:"currencyCode"`
	Address            string  `gorm:"type:varchar(100);not null" json:"address"`
	Memo               *string `gorm:"type:text" json:"memo,omitempty"`
	Verified           bool    `gorm:"default:false" json:"verified"`

	Member           *Member           `gorm:"foreignKey:MemberID" json:"-"`
	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
}

func (Recipient) TableName() string { return "recipients" }
