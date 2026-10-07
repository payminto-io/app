package ledger

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/shopspring/decimal"
)

type (
	AccountID uint
	JournalID uint
)

// Metadata is a jsonb column; a map so GORM binds it as text rather than bytea.
type Metadata map[string]any

func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (m *Metadata) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*m = nil
		return nil
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("ledger: cannot scan %T into Metadata", src)
	}
}

// AccountRow is ledger_accounts. Primary keys follow the repo's uint convention (PaymintoModel).
// Environment is part of the uniqueness key so test and live never share an account (ticket 13).
type AccountRow struct {
	ID          AccountID               `gorm:"primarykey"`
	Environment environment.Environment `gorm:"type:varchar(8);not null;default:'test';uniqueIndex:ledger_accounts_env_owner_asset_kind_key,priority:1"`
	OwnerType   OwnerType               `gorm:"type:varchar(16);not null;uniqueIndex:ledger_accounts_env_owner_asset_kind_key,priority:2"`
	OwnerID     string                  `gorm:"type:varchar(128);not null;uniqueIndex:ledger_accounts_env_owner_asset_kind_key,priority:3"`
	Asset       string                  `gorm:"type:varchar(16);not null;uniqueIndex:ledger_accounts_env_owner_asset_kind_key,priority:4"`
	Kind        AccountKind             `gorm:"type:varchar(16);not null;uniqueIndex:ledger_accounts_env_owner_asset_kind_key,priority:5"`
	CreatedAt   time.Time               `gorm:"not null"`
}

func (AccountRow) TableName() string { return "ledger_accounts" }

func (a AccountRow) Key() AccountKey {
	return AccountKey{OwnerType: a.OwnerType, OwnerID: a.OwnerID, Asset: a.Asset, Kind: a.Kind, Environment: a.Environment}
}

// JournalRow is ledger_journals. RequestHash lets a replayed key be checked against its original payload.
type JournalRow struct {
	ID             JournalID   `gorm:"primarykey"`
	Kind           JournalKind `gorm:"type:varchar(16);not null"`
	ReferenceType  string      `gorm:"type:varchar(128);not null;index:ledger_journals_reference_idx,priority:1"`
	ReferenceID    string      `gorm:"type:varchar(128);not null;index:ledger_journals_reference_idx,priority:2"`
	IdempotencyKey string      `gorm:"type:varchar(128);not null;uniqueIndex:ledger_journals_idempotency_key_key"`
	// Environment scopes the idempotency key: a key reused across environments is a conflict, never a replay.
	Environment environment.Environment `gorm:"type:varchar(8);not null;default:'test';index:ledger_journals_environment_idx"`
	RequestHash string                  `gorm:"type:char(64);not null"`
	PostedAt    time.Time               `gorm:"not null;index"`
	Metadata    Metadata                `gorm:"type:jsonb;not null"`
	// PostingTxID is stamped by a Postgres trigger; lines may only join a journal from the same transaction.
	PostingTxID int64     `gorm:"type:bigint;not null;default:0"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (JournalRow) TableName() string { return "ledger_journals" }

// LineRow is ledger_lines. Asset is denormalised so the balance trigger and the (account_id, asset) FK can check it.
type LineRow struct {
	ID        uint            `gorm:"primarykey"`
	JournalID JournalID       `gorm:"not null;index"`
	AccountID AccountID       `gorm:"not null;index"`
	Asset     string          `gorm:"type:varchar(16);not null"`
	Amount    decimal.Decimal `gorm:"type:numeric(38,18);not null"`
	CreatedAt time.Time       `gorm:"not null"`
}

func (LineRow) TableName() string { return "ledger_lines" }

// Models lists the GORM models for AutoMigrate; constraints and triggers come from InstallConstraints.
func Models() []any {
	return []any{&AccountRow{}, &JournalRow{}, &LineRow{}}
}
