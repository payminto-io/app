package models

// WalletFunction is the audit log of every operation performed on a Wallet
// — generation, rotation, signing, locking. Uses BaseModel because it's an
// audit table.
type WalletFunction struct {
	BaseModel
	WalletID    uint    `gorm:"not null;index" json:"walletID"`
	MemberID    *uint   `json:"memberID,omitempty"`
	Action      string  `gorm:"type:varchar(50);not null" json:"action"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	Successful  bool    `gorm:"default:true" json:"successful"`
	Metadata    *string `gorm:"type:text" json:"metadata,omitempty"`
}

func (WalletFunction) TableName() string { return "wallet_functions" }
