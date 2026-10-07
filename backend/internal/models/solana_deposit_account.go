package models

import "time"

// SolanaDepositAccount records, per Solana deposit address, the owner keypair's address and the
// associated token account the payment watches, plus the signature cursors of the watcher.
// deposit_addresses.address holds the token account; payers may also send to OwnerAddress.
// Design: .scratch/payments-v1/issues/09-solana-usdc.md.
type SolanaDepositAccount struct {
	PaymintoModel
	DepositAddressID     uint   `gorm:"not null;uniqueIndex" json:"depositAddressID"`
	PaymentRequestID     *uint  `gorm:"index" json:"paymentRequestID,omitempty"`
	BlockchainCurrencyID uint   `gorm:"not null;index" json:"blockchainCurrencyID"`
	OwnerAddress         string `gorm:"type:varchar(64);not null;index" json:"ownerAddress"`
	TokenAccount         string `gorm:"type:varchar(64);not null;uniqueIndex" json:"tokenAccount"`
	Mint                 string `gorm:"type:varchar(64);not null" json:"mint"`
	TokenProgram         string `gorm:"type:varchar(64);not null" json:"tokenProgram"`
	Decimals             uint8  `gorm:"not null" json:"decimals"`
	Status               string `gorm:"type:varchar(20);default:'watching';not null;index" json:"status"`
	// Cursors are the newest signature already processed for each address (getSignaturesForAddress until).
	TokenAccountCursor string     `gorm:"type:varchar(128)" json:"tokenAccountCursor"`
	OwnerCursor        string     `gorm:"type:varchar(128)" json:"ownerCursor"`
	LastPolledAt       *time.Time `json:"lastPolledAt,omitempty"`
	LastSeenSlot       int64      `gorm:"default:0" json:"lastSeenSlot"`

	DepositAddress *DepositAddress `gorm:"foreignKey:DepositAddressID" json:"-"`
}

func (SolanaDepositAccount) TableName() string { return "solana_deposit_accounts" }

const (
	// SolanaDepositAccountWatching is polled for new signatures.
	SolanaDepositAccountWatching = "watching"
	// SolanaDepositAccountClosed is no longer polled; its token account was swept and closed.
	SolanaDepositAccountClosed = "closed"
)
