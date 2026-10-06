package models

import "time"

type Role struct {
	PaymintoModel
	Name        string  `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	DisplayName string  `gorm:"type:varchar(100);not null" json:"displayName"`
	Description *string `gorm:"type:text" json:"description,omitempty"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	Members     []Member     `gorm:"many2many:member_roles;" json:"-"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	PaymintoModel
	Name        string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	DisplayName string `gorm:"type:varchar(200);not null" json:"displayName"`
	Description string `gorm:"type:text;not null" json:"description"`
}

func (Permission) TableName() string { return "permissions" }

type MemberExternalPlatformRole struct {
	CreatedAt          time.Time `json:"createdAt"`
	MemberID           uint      `gorm:"primaryKey" json:"memberID"`
	ExternalPlatformID uint      `gorm:"primaryKey" json:"externalPlatformID"`
	RoleID             uint      `gorm:"primaryKey" json:"roleID"`

	Member           *Member           `gorm:"foreignKey:MemberID" json:"-"`
	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	Role             *Role             `gorm:"foreignKey:RoleID" json:"-"`
}
