package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// PaymentRequest is the core invoice model. Merchants create one per customer
// checkout; it tracks the required amount, state machine, and optional deposit
// address assignment.
type PaymentRequest struct {
	PaymintoModel
	ReferenceID        string           `gorm:"type:varchar(100);not null;uniqueIndex" json:"referenceID"`
	AmountInUSD        decimal.Decimal  `gorm:"type:numeric(38,18);not null" json:"amountInUSD"`
	State              string           `gorm:"type:varchar(20);default:'OPEN';not null" json:"state"`
	CustomerEmail      *string          `gorm:"type:text" json:"customerEmail,omitempty"`
	CustomerID         *string          `gorm:"type:text" json:"customerID,omitempty"`
	InvoiceID          *string          `gorm:"type:text" json:"invoiceID,omitempty"`
	ExpiresAt          *time.Time       `json:"expiresAt,omitempty"`
	ConfirmedAt        *time.Time       `json:"confirmedAt,omitempty"`

	MemberID           uint `gorm:"not null" json:"memberID"`
	ExternalPlatformID uint `gorm:"not null" json:"externalPlatformID"`
	DepositAddressID   *uint `json:"depositAddressID,omitempty"`

	Member           *Member           `gorm:"foreignKey:MemberID" json:"-"`
	ExternalPlatform *ExternalPlatform `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	Deposits         []Deposit         `gorm:"foreignKey:PaymentRequestID" json:"deposits,omitempty"`
}

func (PaymentRequest) TableName() string { return "payment_requests" }

const (
	PaymentStateOpen            = "OPEN"
	PaymentStateCancelled       = "CANCELLED"
	PaymentStateFilled          = "FILLED"
	PaymentStatePartiallyFilled = "PARTIALLY_FILLED"
	PaymentStateOverFilled      = "OVER_FILLED"
)

// Deposit records a single on-chain transaction detected by a block processor.
// Multiple deposits can be linked to one PaymentRequest (partial fills).
type Deposit struct {
	PaymintoModel
	TxID                  string          `gorm:"type:text;not null" json:"txID"`
	Amount                decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	Status                string          `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	Confirmations         int             `gorm:"default:0" json:"confirmations"`
	RequiredConfirmations int             `json:"requiredConfirmations"`
	FromAddress           string          `gorm:"type:text" json:"fromAddress"`
	ToAddress             string          `gorm:"type:text;not null" json:"toAddress"`
	BlockNumber           int64           `json:"blockNumber"`
	BlockHash             string          `gorm:"type:text" json:"blockHash"`

	BlockchainCurrencyID uint  `gorm:"not null" json:"blockchainCurrencyID"`
	PaymentRequestID     *uint `json:"paymentRequestID,omitempty"`
	MemberID             uint  `gorm:"not null" json:"memberID"`

	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"blockchainCurrency,omitempty"`
	PaymentRequest     *PaymentRequest     `gorm:"foreignKey:PaymentRequestID" json:"-"`
}

func (Deposit) TableName() string { return "deposits" }

const (
	DepositStatusPending    = "pending"
	DepositStatusConfirming = "confirming"
	DepositStatusConfirmed  = "confirmed"
	DepositStatusFailed     = "failed"
	// DepositStatusSwept marks a confirmed deposit whose funds have been swept
	// to cold storage. Terminal — makes sweeping idempotent (never swept twice).
	DepositStatusSwept = "swept"
)
