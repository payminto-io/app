package models

type ExternalPlatform struct {
	PaymintoModel
	Name             string  `gorm:"type:varchar(200);not null" json:"name"`
	LogoPath         string  `gorm:"type:text" json:"logoPath"`
	BrandColor       string  `gorm:"type:varchar(20)" json:"brandColor"`
	Website          string  `gorm:"type:text" json:"website"`
	SuccessEndpoint  string  `gorm:"type:text" json:"successEndpoint"`
	FrontendEndpoint *string `gorm:"type:text" json:"frontendEndpoint,omitempty"`
	CancelEndpoint   *string `gorm:"type:text" json:"cancelEndpoint,omitempty"`
	SupportEmail     *string `gorm:"type:text" json:"supportEmail,omitempty"`

	DefaultPaymentBlockchainID         *uint `json:"defaultPaymentBlockchainID,omitempty"`
	DefaultPaymentBlockchainCurrencyID *uint `json:"defaultPaymentBlockchainCurrencyID,omitempty"`

	APIKeys       []APIKey                     `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	PlatformRoles []MemberExternalPlatformRole `gorm:"foreignKey:ExternalPlatformID" json:"-"`
}

func (ExternalPlatform) TableName() string { return "external_platforms" }
