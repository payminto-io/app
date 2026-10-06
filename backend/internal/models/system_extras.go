package models

import "time"

// SeederLog records every database seeder run with the seeder name, status,
// and any error. Used by ops to audit which seed migrations have been
// applied to which database.
type SeederLog struct {
	PaymintoModel
	SeederName      string     `gorm:"type:varchar(200);not null;uniqueIndex" json:"seederName"`
	Status          string     `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	ErrorMsg        *string    `gorm:"type:text" json:"errorMsg,omitempty"`
	RecordsAffected int        `gorm:"default:0" json:"recordsAffected"`
}

func (SeederLog) TableName() string { return "seeder_logs" }

// GenericDataStore is a namespaced key-value store for arbitrary state that
// doesn't fit a domain table — last-seen block heights per chain, last-run
// timestamps for jobs, transient runtime data.
type GenericDataStore struct {
	PaymintoModel
	Namespace string `gorm:"type:varchar(100);not null;index:idx_gds_namespace_key,unique" json:"namespace"`
	Key       string `gorm:"type:varchar(200);not null;index:idx_gds_namespace_key,unique" json:"key"`
	Value     string `gorm:"type:text;not null" json:"value"`
	TTL       *int64 `json:"ttl,omitempty"`
}

func (GenericDataStore) TableName() string { return "generic_data_stores" }

// Tag is a generic labeling table — used by analytics, transactions, members
// to attach searchable tags. PayRam re-uses one tags table across many
// taggable entities.
type Tag struct {
	PaymintoModel
	Name        string  `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Color       *string `gorm:"type:varchar(20)" json:"color,omitempty"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
}

func (Tag) TableName() string { return "tags" }
