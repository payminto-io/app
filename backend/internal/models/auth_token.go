package models

import "time"

// AuthRefreshToken stores hashed refresh tokens with rotation family tracking
// for reuse-detection. Mirrors PayRam's auth_refresh_tokens table.
type AuthRefreshToken struct {
	PaymintoModel
	TokenHash          string     `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	MemberID           uint       `gorm:"not null;index" json:"memberID"`
	ExternalPlatformID uint       `gorm:"not null;default:0;index" json:"externalPlatformID"`
	RotationFamily     string     `gorm:"type:varchar(64);not null;index" json:"-"`
	IssuedAt           time.Time  `gorm:"not null" json:"issuedAt"`
	ExpiresAt          time.Time  `gorm:"not null;index" json:"expiresAt"`
	LastUsedAt         *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt          *time.Time `json:"revokedAt,omitempty"`
	UserAgent          *string    `gorm:"type:text" json:"userAgent,omitempty"`
	IPAddress          *string    `gorm:"type:varchar(45)" json:"ipAddress,omitempty"`

	Member *Member `gorm:"foreignKey:MemberID" json:"-"`
}

func (AuthRefreshToken) TableName() string { return "auth_refresh_tokens" }
