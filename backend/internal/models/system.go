package models

// Configuration is a runtime key-value store for tunables that can be changed
// without a redeploy. It also holds the network mode stamp (key="mode",
// value="testnet"|"mainnet") used by the boot-time safety check.
type Configuration struct {
	PaymintoModel
	Key         string  `gorm:"type:varchar(200);not null;uniqueIndex" json:"key"`
	Value       string  `gorm:"type:text;not null" json:"value"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	Category    string  `gorm:"type:varchar(50)" json:"category"`
}

func (Configuration) TableName() string { return "configurations" }

// ConfigModeKey is the well-known key that stamps a database with its network
// mode. The value is "testnet" or "mainnet" and MUST match the
// BLOCKCHAIN_NETWORK_TYPE env var at boot.
const ConfigModeKey = "mode"
