package models

import "time"

// SolanaDepositAccount records, per Solana deposit address, the owner keypair's address and the
// associated token account the payment watches, the watch window and cadence, and the signature
// cursors of the watcher. deposit_addresses.address holds the token account; payers may also send
// to OwnerAddress. Design: .scratch/payments-v1/issues/09-solana-usdc.md.
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
	// PaymentExpiresAt switches to the late cadence; WatchUntil ends polling (status expired).
	PaymentExpiresAt *time.Time `json:"paymentExpiresAt,omitempty"`
	WatchUntil       *time.Time `gorm:"index" json:"watchUntil,omitempty"`
	// LastBalanceRaw is the token account balance last seen by getMultipleAccounts; a change triggers a signature poll.
	LastBalanceRaw string     `gorm:"type:varchar(40)" json:"lastBalanceRaw"`
	TokenPollAfter *time.Time `gorm:"index" json:"tokenPollAfter,omitempty"`
	OwnerPollAfter *time.Time `json:"ownerPollAfter,omitempty"`
	// HeldSignature is a listed signature whose transaction no node has returned yet; the cursor
	// stays below it for HeldAttempts ticks, then it moves to UnresolvedSignatures (JSON array)
	// which is retried on its own so later signatures can proceed.
	HeldSignature        string `gorm:"type:varchar(128)" json:"heldSignature"`
	HeldAttempts         int    `gorm:"default:0" json:"heldAttempts"`
	UnresolvedSignatures string `gorm:"type:text" json:"unresolvedSignatures"`
	// UnresolvedAttempts counts polls with unresolved signatures outstanding; past the budget expiry proceeds.
	UnresolvedAttempts int `gorm:"default:0" json:"unresolvedAttempts"`
	// BalanceHoldAttempts counts ticks a balance movement went unexplained by any listed signature.
	BalanceHoldAttempts int `gorm:"default:0" json:"balanceHoldAttempts"`

	DepositAddress *DepositAddress `gorm:"foreignKey:DepositAddressID" json:"-"`
}

func (SolanaDepositAccount) TableName() string { return "solana_deposit_accounts" }

const (
	// SolanaDepositAccountWatching is polled for new signatures.
	SolanaDepositAccountWatching = "watching"
	// SolanaDepositAccountExpired passed watch_until; confirmed deposits are still swept.
	SolanaDepositAccountExpired = "expired"
	// SolanaDepositAccountClosed is no longer polled; its token account was swept and closed.
	SolanaDepositAccountClosed = "closed"
	// SolanaDepositAccountDrained holds nothing on chain though its deposits were never booked swept; an anomaly records it.
	SolanaDepositAccountDrained = "drained"
)

// SolanaSweepAttempt is one signed transaction of a sweep, persisted before it is sent; a rebuild after
// blockhash expiry adds another. The tracker checks every attempt so a late-landing earlier attempt is
// booked, never lost. (sweep_id, attempt_no) is unique so two workers cannot both rebuild one sweep.
type SolanaSweepAttempt struct {
	PaymintoModel
	SweepID              uint   `gorm:"not null;index;uniqueIndex:idx_solana_sweep_attempts_sweep_attempt,priority:1" json:"sweepID"`
	AttemptNo            int    `gorm:"not null;default:1;uniqueIndex:idx_solana_sweep_attempts_sweep_attempt,priority:2" json:"attemptNo"`
	Signature            string `gorm:"type:varchar(128);not null;uniqueIndex" json:"signature"`
	Blockhash            string `gorm:"type:varchar(64);not null" json:"blockhash"`
	LastValidBlockHeight uint64 `gorm:"not null" json:"lastValidBlockHeight"`
	Status               string `gorm:"type:varchar(20);default:'sent';not null" json:"status"`
}

func (SolanaSweepAttempt) TableName() string { return "solana_sweep_attempts" }

const (
	// SolanaSweepAttemptSigned is persisted before the send; the node may or may not have taken it.
	SolanaSweepAttemptSigned  = "signed"
	SolanaSweepAttemptSent    = "sent"
	SolanaSweepAttemptLanded  = "landed"
	SolanaSweepAttemptExpired = "expired"
	SolanaSweepAttemptFailed  = "failed"
)

// SolanaSweepDeposit links the deposits a sweep claimed, so a failed sweep releases exactly those.
// A deposit released by a failed sweep is claimed again by a later sweep, so uniqueness is per sweep.
type SolanaSweepDeposit struct {
	PaymintoModel
	SweepID   uint `gorm:"not null;uniqueIndex:idx_solana_sweep_deposits_sweep_deposit,priority:1" json:"sweepID"`
	DepositID uint `gorm:"not null;index;uniqueIndex:idx_solana_sweep_deposits_sweep_deposit,priority:2" json:"depositID"`
}

func (SolanaSweepDeposit) TableName() string { return "solana_sweep_deposits" }

// SolanaSweepLock is the database's guarantee of one sweep in flight per token account: inserted
// with the sweep rows, hard-deleted when the sweep completes or fails (no soft delete: a deleted row
// must not keep holding the unique index). See service/SOLANA_SWEEPS.md.
type SolanaSweepLock struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	TokenAccount string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"tokenAccount"`
	SweepID      uint      `gorm:"not null;index" json:"sweepID"`
}

func (SolanaSweepLock) TableName() string { return "solana_sweep_locks" }
