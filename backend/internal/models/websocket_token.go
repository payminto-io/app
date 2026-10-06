package models

import "time"

// WebSocketToken is a short-lived token issued via REST that a client uses to
// open an authenticated WebSocket connection. Mirrors PayRam's pattern of
// avoiding JWTs in WebSocket URLs (which can leak through proxies and access
// logs).
type WebSocketToken struct {
	PaymintoModel
	Token              string     `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	MemberID           uint       `gorm:"not null;index" json:"memberID"`
	ExternalPlatformID *uint      `json:"externalPlatformID,omitempty"`
	ExpiresAt          time.Time  `gorm:"not null;index" json:"expiresAt"`
	UsedAt             *time.Time `json:"usedAt,omitempty"`
	IPAddress          *string    `gorm:"type:varchar(45)" json:"ipAddress,omitempty"`

	Member           *Member           `gorm:"foreignKey:MemberID" json:"-"`
	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
}

func (WebSocketToken) TableName() string { return "web_socket_tokens" }
