package models

import "time"

// OTP holds one-time passwords used for withdrawal approval, password reset,
// email verification, etc. The Code is stored as a SHA-256 hash so the
// plaintext is never persisted. Each OTP has a maximum-attempt budget and an
// expiry timestamp.
type OTP struct {
	PaymintoModel
	MemberID    uint       `gorm:"not null;index" json:"memberID"`
	Purpose     string     `gorm:"type:varchar(50);not null;index" json:"purpose"`
	CodeHash    string     `gorm:"type:varchar(128);not null" json:"-"`
	Attempts    int        `gorm:"default:0;not null" json:"attempts"`
	MaxAttempts int        `gorm:"default:5;not null" json:"maxAttempts"`
	ExpiresAt   time.Time  `gorm:"not null" json:"expiresAt"`
	UsedAt      *time.Time `json:"usedAt,omitempty"`

	Member *Member `gorm:"foreignKey:MemberID" json:"-"`
}

func (OTP) TableName() string { return "otps" }

// Well-known OTP purposes.
const (
	OTPPurposeWithdrawalApproval = "withdrawal_approval"
	OTPPurposePasswordReset      = "password_reset"
	OTPPurposeEmailVerification  = "email_verification"
	OTPPurposeSensitiveAction    = "sensitive_action"
)
