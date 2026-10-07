package models

import (
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
)

// APIKey rows carry the environment they were issued in; keys older than the column are test (ticket 13).
type APIKey struct {
	PaymintoModel
	Key                string                  `gorm:"type:text;not null;uniqueIndex" json:"-"`
	Status             string                  `gorm:"type:varchar(20);default:'active';not null" json:"status"`
	Environment        environment.Environment `gorm:"type:varchar(8);not null;default:'test';index" json:"environment"`
	Prefix             string                  `gorm:"type:varchar(16);not null;default:''" json:"prefix"`
	MemberID           *uint                   `json:"memberID,omitempty"`
	ExternalPlatformID uint                    `gorm:"not null" json:"externalPlatformID"`
	RoleID             *uint                   `json:"roleID,omitempty"`
	ExpireAt           *time.Time              `json:"expireAt,omitempty"`
	Description        *string                 `gorm:"type:text" json:"description,omitempty"`

	Member           *Member           `gorm:"foreignKey:MemberID" json:"-"`
	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	Role             *Role             `gorm:"foreignKey:RoleID" json:"-"`
}

func (APIKey) TableName() string { return "api_keys" }
