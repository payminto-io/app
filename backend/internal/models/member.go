package models

import "time"

// Member represents a Payminto user account (merchant or customer). Members
// authenticate via password or OAuth and may hold multiple API keys and roles.
type Member struct {
	PaymintoModel
	Name                  string     `gorm:"type:varchar(100);not null" json:"name"`
	Email                 *string    `gorm:"type:text" json:"email,omitempty"`
	CustomerID            string     `gorm:"type:text" json:"customerID"`
	Level                 int        `gorm:"default:0;not null" json:"level"`
	Group                 string     `gorm:"type:varchar(20);default:'vip-0';not null" json:"group"`
	State                 string     `gorm:"type:varchar(20);default:'active';not null" json:"state"`
	Username              *string    `gorm:"type:varchar(20)" json:"username,omitempty"`
	Password              *string    `gorm:"type:text" json:"-"`
	ResetPasswordRequired bool       `gorm:"default:false;not null" json:"resetPasswordRequired"`
	ResetPasswordToken    *string    `gorm:"type:text" json:"-"`
	ResetPasswordExpiry   *time.Time `json:"-"`
	MemberType            string     `gorm:"type:varchar(20);default:'customer';not null" json:"memberType"`
	ReferredByMemberID    *uint      `json:"referredByMemberID,omitempty"`

	ReferredByMember *Member  `gorm:"foreignKey:ReferredByMemberID" json:"-"`
	Roles            []Role   `gorm:"many2many:member_roles;" json:"roles,omitempty"`
	APIKeys          []APIKey `gorm:"foreignKey:MemberID" json:"-"`
}

func (Member) TableName() string { return "members" }
