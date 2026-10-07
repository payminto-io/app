package environment

import "time"

// StampRow is gateway_environment, the one-row table that names the environment a database serves.
// It is written on first boot and checked on every boot by the server, cmd/migrate and cmd/devseed.
type StampRow struct {
	ID          int16        `gorm:"primaryKey;check:gateway_environment_singleton,id = 1"`
	Environment Environment  `gorm:"type:varchar(8);not null"`
	StampedAt   time.Time    `gorm:"not null"`
	AdoptedFrom *Environment `gorm:"type:varchar(8)"`
	AdoptedAt   *time.Time
}

func (StampRow) TableName() string { return "gateway_environment" }

// StampID is the only primary key gateway_environment ever holds.
const StampID int16 = 1
