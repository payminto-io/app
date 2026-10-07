package models

// PaymentChannel represents a payment method (crypto or card onramp).
// PayRam columns: name, channel_type, status, config_value, metadata_value,
// api_key, display_name, description, icon, display_order, is_default
type PaymentChannel struct {
	PaymintoModel
	Name          string  `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	ChannelType   string  `gorm:"type:varchar(50);not null" json:"channelType"`    // PayRam: blockchain, app
	Status        string  `gorm:"type:varchar(20);default:'active'" json:"status"` // PayRam: active, inactive
	ConfigValue   *string `gorm:"type:text" json:"configValue,omitempty"`          // PayRam: JSON config (fees, geography, networks)
	MetadataValue *string `gorm:"type:text" json:"metadataValue,omitempty"`        // PayRam: additional metadata JSON
	APIKey        *string `gorm:"type:varchar(255)" json:"-"`                      // PayRam: API key for onramp providers
	DisplayName   string  `gorm:"type:varchar(100)" json:"displayName"`            // PayRam: human-readable name
	Description   *string `gorm:"type:text" json:"description,omitempty"`
	Icon          *string `gorm:"type:text" json:"icon,omitempty"` // PayRam: icon path
	DisplayOrder  int     `gorm:"default:0" json:"displayOrder"`   // PayRam: UI sort order
	IsDefault     bool    `gorm:"default:false" json:"isDefault"`  // PayRam: default channel
}

func (PaymentChannel) TableName() string { return "payment_channels" }

// DisabledPaymentChannelProject is the per-project opt-out — a channel can
// be disabled for one specific external platform without affecting others.
type DisabledPaymentChannelProject struct {
	PaymintoModel
	ExternalPlatformID uint    `gorm:"not null;index" json:"externalPlatformID"`
	PaymentChannelID   uint    `gorm:"not null;index" json:"paymentChannelID"`
	Reason             *string `gorm:"type:text" json:"reason,omitempty"`

	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	PaymentChannel   *PaymentChannel   `gorm:"foreignKey:PaymentChannelID" json:"-"`
}

func (DisabledPaymentChannelProject) TableName() string { return "disabled_payment_channel_projects" }

// PaymentsApp tracks mobile/web payment apps that integrate with the gateway
// (e.g., a third-party POS terminal). Each app has its own credentials.
type PaymentsApp struct {
	PaymintoModel
	Name               string  `gorm:"type:varchar(100);not null" json:"name"`
	AppType            string  `gorm:"type:varchar(50);not null" json:"appType"`
	APIKey             string  `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	ExternalPlatformID uint    `gorm:"not null;index" json:"externalPlatformID"`
	Status             string  `gorm:"type:varchar(20);default:'active';not null" json:"status"`
	Config             *string `gorm:"type:text" json:"config,omitempty"`

	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
}

func (PaymentsApp) TableName() string { return "payments_apps" }
