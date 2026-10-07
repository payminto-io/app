package fees

import (
	"database/sql/driver"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

//go:embed schema.sql
var schemaSQL string

// SchemaSQL is the DDL migration 2026100702 must repeat verbatim.
func SchemaSQL() string { return schemaSQL }

// Migrate creates fee_rules and the payment_requests snapshot columns in dev/test; Postgres only.
func Migrate(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if err := db.Exec(schemaSQL).Error; err != nil {
		return fmt.Errorf("fees: migrate: %w", err)
	}
	return nil
}

// slabsJSON is the nullable jsonb slabs column.
type slabsJSON []Slab

func (s slabsJSON) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal([]Slab(s))
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (s *slabsJSON) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = nil
		return nil
	case []byte:
		return json.Unmarshal(v, (*[]Slab)(s))
	case string:
		return json.Unmarshal([]byte(v), (*[]Slab)(s))
	default:
		return fmt.Errorf("fees: cannot scan %T into slabs", src)
	}
}

type ruleRow struct {
	ID            uint   `gorm:"primarykey"`
	LineageID     string `gorm:"type:uuid"`
	Version       int    `gorm:"not null"`
	Method        string `gorm:"not null"`
	Connector     *string
	CardType      *string
	Region        *string
	Currency      string `gorm:"not null"`
	MinorUnits    int32
	Percent       decimal.Decimal     `gorm:"type:numeric(9,6)"`
	Flat          decimal.Decimal     `gorm:"type:numeric(38,18)"`
	Slabs         slabsJSON           `gorm:"type:jsonb"`
	MinFee        decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	MaxFee        decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	Taxable       bool
	TaxPercent    decimal.Decimal `gorm:"type:numeric(9,6)"`
	FeeBearer     string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	CreatedBy     string
	CreatedAt     time.Time
}

func (ruleRow) TableName() string { return "fee_rules" }

func nullDec(v *decimal.Decimal) decimal.NullDecimal {
	if v == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{Decimal: *v, Valid: true}
}

func decPtr(v decimal.NullDecimal) *decimal.Decimal {
	if !v.Valid {
		return nil
	}
	return &v.Decimal
}

func newRuleRow(lineage string, version int, s Scope, minorUnits int32, p Pricing, actor string) ruleRow {
	var card *string
	if s.CardType != nil {
		c := string(*s.CardType)
		card = &c
	}
	return ruleRow{
		LineageID:     lineage,
		Version:       version,
		Method:        string(s.Method),
		Connector:     s.Connector,
		CardType:      card,
		Region:        s.Region,
		Currency:      s.Currency,
		MinorUnits:    minorUnits,
		Percent:       p.Percent,
		Flat:          p.Flat,
		Slabs:         slabsJSON(p.Slabs),
		MinFee:        nullDec(p.MinFee),
		MaxFee:        nullDec(p.MaxFee),
		Taxable:       p.Taxable,
		TaxPercent:    p.TaxPercent,
		FeeBearer:     string(p.FeeBearer),
		EffectiveFrom: *p.EffectiveFrom,
		EffectiveTo:   p.EffectiveTo,
		CreatedBy:     actor,
	}
}

func (r ruleRow) rule() Rule {
	var card *CardType
	if r.CardType != nil {
		c := CardType(*r.CardType)
		card = &c
	}
	var to *time.Time
	if r.EffectiveTo != nil {
		t := r.EffectiveTo.UTC()
		to = &t
	}
	return Rule{
		ID:        r.ID,
		LineageID: r.LineageID,
		Version:   r.Version,
		Scope: Scope{
			Method:    Method(r.Method),
			Connector: r.Connector,
			CardType:  card,
			Region:    r.Region,
			Currency:  r.Currency,
		},
		MinorUnits:    r.MinorUnits,
		Percent:       r.Percent,
		Flat:          r.Flat,
		Slabs:         []Slab(r.Slabs),
		MinFee:        decPtr(r.MinFee),
		MaxFee:        decPtr(r.MaxFee),
		Taxable:       r.Taxable,
		TaxPercent:    r.TaxPercent,
		FeeBearer:     FeeBearer(r.FeeBearer),
		EffectiveFrom: r.EffectiveFrom.UTC(),
		EffectiveTo:   to,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt.UTC(),
	}
}

type snapshotRow struct {
	ID               uint `gorm:"primarykey"`
	AttemptID        string
	PaymentRequestID uint
	MerchantID       uint
	FeeRuleID        uint
	FeeRuleVersion   int
	Currency         string
	LedgerAsset      string
	FeeBearer        string
	CreatedAt        time.Time
}

func (snapshotRow) TableName() string { return "fee_snapshots" }

func (r snapshotRow) snapshot() Snapshot {
	return Snapshot{
		ID: r.ID, AttemptID: r.AttemptID, PaymentRequestID: r.PaymentRequestID, MerchantID: r.MerchantID,
		RuleID: r.FeeRuleID, RuleVersion: r.FeeRuleVersion, Currency: r.Currency, LedgerAsset: r.LedgerAsset,
		FeeBearer: FeeBearer(r.FeeBearer), CreatedAt: r.CreatedAt.UTC(),
	}
}
