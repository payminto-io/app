package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// OnramperPayments tracks Onramper card-to-crypto sessions. Created when a
// merchant initiates a fiat-onramp payment, updated when Onramper's webhook
// reports settlement.
type OnramperPayments struct {
	PaymintoModel
	SessionID          string           `gorm:"type:varchar(100);not null;uniqueIndex" json:"sessionID"`
	PaymentRequestID   *uint            `gorm:"index" json:"paymentRequestID,omitempty"`
	ExternalPlatformID uint             `gorm:"not null;index" json:"externalPlatformID"`
	Status             string           `gorm:"type:varchar(30);default:'created';not null" json:"status"`
	FiatAmount         decimal.Decimal  `gorm:"type:numeric(38,18);not null" json:"fiatAmount"`
	FiatCurrency       string           `gorm:"type:varchar(10);not null" json:"fiatCurrency"`
	CryptoAmount       *decimal.Decimal `gorm:"type:numeric(38,18)" json:"cryptoAmount,omitempty"`
	CryptoCurrency     string           `gorm:"type:varchar(20);not null" json:"cryptoCurrency"`
	BlockchainCode     string           `gorm:"type:varchar(20);not null" json:"blockchainCode"`
	WalletAddress      string           `gorm:"type:varchar(100);not null" json:"walletAddress"`
	OnrampProvider     string           `gorm:"type:varchar(50);default:'onramper';not null" json:"onrampProvider"`
	ProviderTxID       *string          `gorm:"type:varchar(100);index" json:"providerTxID,omitempty"`
	OnchainTxHash      *string          `gorm:"type:varchar(100);index" json:"onchainTxHash,omitempty"`
	CompletedAt        *time.Time       `json:"completedAt,omitempty"`
	FailedAt           *time.Time       `json:"failedAt,omitempty"`
	FailureReason      *string          `gorm:"type:text" json:"failureReason,omitempty"`
	WebhookPayload     *string          `gorm:"type:text" json:"webhookPayload,omitempty"`

	PaymentRequest   *PaymentRequest   `gorm:"foreignKey:PaymentRequestID" json:"-"`
	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
}

func (OnramperPayments) TableName() string { return "onramper_payments" }

// Onramper status constants.
const (
	OnramperStatusCreated   = "created"
	OnramperStatusPending   = "pending"
	OnramperStatusCompleted = "completed"
	OnramperStatusFailed    = "failed"
	OnramperStatusExpired   = "expired"
)
