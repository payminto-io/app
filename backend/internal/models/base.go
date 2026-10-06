package models

import (
	"time"

	"gorm.io/gorm"
)

// PaymintoModel is the base struct embedded in every Payminto model. It
// provides auto-increment ID, audit timestamps, and GORM soft-delete support.
type PaymintoModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// BaseModel is used by activity logs, campaigns, events, and rewards — tables
// that PayRam maps to a distinct base type. Structurally identical to
// PaymintoModel but kept as a separate type for clarity and future divergence.
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}
