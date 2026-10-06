package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Withdrawal is a merchant payout request. State machine:
//
//	pending-otp → pending-approval → pending → initiated → sent → processed
//	                                                                 ↘ failed
type Withdrawal struct {
	PaymintoModel
	ReferenceID          string           `gorm:"type:varchar(100);not null;uniqueIndex" json:"referenceID"`
	State                string           `gorm:"type:varchar(30);default:'pending-approval';not null;index" json:"state"`
	Email                *string          `gorm:"type:text" json:"email,omitempty"`
	BlockchainCode       string           `gorm:"type:varchar(20);not null" json:"blockchainCode"`
	CurrencyCode         string           `gorm:"type:varchar(20);not null" json:"currencyCode"`
	Amount               decimal.Decimal  `gorm:"type:numeric(38,18);not null" json:"amount"`
	AmountInUSD          *decimal.Decimal `gorm:"type:numeric(38,18)" json:"amountInUSD,omitempty"`
	NetworkFee           *decimal.Decimal `gorm:"type:numeric(38,18)" json:"networkFee,omitempty"`
	ToAddress            string           `gorm:"type:varchar(100);not null" json:"toAddress"`
	Memo                 *string          `gorm:"type:text" json:"memo,omitempty"`
	CustomerID           *string          `gorm:"type:text" json:"customerID,omitempty"`

	MemberID             uint  `gorm:"not null;index" json:"memberID"`
	ExternalPlatformID   uint  `gorm:"not null;index" json:"externalPlatformID"`
	BlockchainCurrencyID uint  `gorm:"not null" json:"blockchainCurrencyID"`
	OTPID                *uint `json:"otpID,omitempty"`
	ApprovedByMemberID   *uint `json:"approvedByMemberID,omitempty"`
	ApprovedAt           *time.Time `json:"approvedAt,omitempty"`
	InitiatedAt          *time.Time `json:"initiatedAt,omitempty"`
	SentAt               *time.Time `json:"sentAt,omitempty"`
	ProcessedAt          *time.Time `json:"processedAt,omitempty"`
	FailedAt             *time.Time `json:"failedAt,omitempty"`
	FailureReason        *string    `gorm:"type:text" json:"failureReason,omitempty"`

	Member             *Member             `gorm:"foreignKey:MemberID" json:"-"`
	ExternalPlatform   *ExternalPlatform   `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"blockchainCurrency,omitempty"`
}

func (Withdrawal) TableName() string { return "withdrawals" }

// Withdrawal state constants — match PayRam state machine.
const (
	WithdrawalStatePendingOTP      = "pending-otp"
	WithdrawalStatePendingApproval = "pending-approval"
	WithdrawalStatePending         = "pending"
	WithdrawalStateInitiated       = "initiated"
	WithdrawalStateSent            = "sent"
	WithdrawalStateProcessed       = "processed"
	WithdrawalStateFailed          = "failed"
	WithdrawalStateCancelled       = "cancelled"
)

// Withdraw is the internal blockchain transaction created when a Withdrawal
// gets broadcast. PayRam splits these so a single Withdrawal can produce
// multiple chain-level transactions (UTXO consolidation, retry, etc.).
type Withdraw struct {
	PaymintoModel
	WithdrawalID         uint            `gorm:"not null;index" json:"withdrawalID"`
	TxHash               *string         `gorm:"type:varchar(100);index" json:"txHash,omitempty"`
	BlockchainCurrencyID uint            `gorm:"not null;index" json:"blockchainCurrencyID"`
	FromAddress          string          `gorm:"type:varchar(100);not null" json:"fromAddress"`
	ToAddress            string          `gorm:"type:varchar(100);not null" json:"toAddress"`
	Amount               decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	GasFee               decimal.Decimal `gorm:"type:numeric(38,18);default:0" json:"gasFee"`
	Status               string          `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	BlockNumber          *int64          `json:"blockNumber,omitempty"`
	Confirmations        int             `gorm:"default:0" json:"confirmations"`
	BroadcastedAt        *time.Time      `json:"broadcastedAt,omitempty"`
	ConfirmedAt          *time.Time      `json:"confirmedAt,omitempty"`

	Withdrawal         *Withdrawal         `gorm:"foreignKey:WithdrawalID" json:"-"`
	BlockchainCurrency *BlockchainCurrency `gorm:"foreignKey:BlockchainCurrencyID" json:"-"`
}

func (Withdraw) TableName() string { return "withdraws" }
