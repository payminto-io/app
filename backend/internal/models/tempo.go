package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// TempoMemoCredit records every TransferWithMemo log the Tempo watcher has decided on, keyed by
// (tx hash, log index) so a log is credited at most once however often its block range is rescanned.
// Outcome is "credited" (DepositID set) or the rejection reason; rejected logs are kept for audit.
type TempoMemoCredit struct {
	ID               uint            `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time       `json:"createdAt"`
	TxHash           string          `gorm:"type:varchar(66);not null;uniqueIndex:idx_tempo_memo_credit_log" json:"txHash"`
	LogIndex         uint            `gorm:"not null;uniqueIndex:idx_tempo_memo_credit_log" json:"logIndex"`
	BlockNumber      int64           `json:"blockNumber"`
	Token            string          `gorm:"type:varchar(42);not null" json:"token"`
	FromAddress      string          `gorm:"type:varchar(42)" json:"fromAddress"`
	ToAddress        string          `gorm:"type:varchar(42);not null" json:"toAddress"`
	Amount           decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	Memo             string          `gorm:"type:varchar(66);not null;index" json:"memo"`
	Outcome          string          `gorm:"type:varchar(32);not null" json:"outcome"`
	PaymentRequestID *uint           `json:"paymentRequestID,omitempty"`
	DepositID        *uint           `json:"depositID,omitempty"`
}

func (TempoMemoCredit) TableName() string { return "tempo_memo_credits" }
