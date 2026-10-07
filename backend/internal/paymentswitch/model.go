package paymentswitch

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

// JSONMap is a jsonb column; a map so GORM binds it as text rather than bytea (same as ledger.Metadata).
type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (m *JSONMap) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*m = nil
		return nil
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("paymentswitch: cannot scan %T into JSONMap", src)
	}
}

// Text ids with a type prefix (pi_, pa_, re_) like the payment_lifecycle tables; they are safe to expose.
type IntentRow struct {
	ID                string                   `gorm:"type:varchar(64);primarykey"`
	MerchantID        string                   `gorm:"type:varchar(128);not null;index;uniqueIndex:switch_intents_merchant_idempotency_key,priority:1"`
	PlatformID        string                   `gorm:"type:varchar(128);not null;default:''"`
	IdempotencyKey    string                   `gorm:"type:varchar(128);not null;uniqueIndex:switch_intents_merchant_idempotency_key,priority:2"`
	RequestHash       string                   `gorm:"type:char(64);not null"`
	Status            IntentStatus             `gorm:"type:varchar(32);not null;index"`
	Amount            decimal.Decimal          `gorm:"type:numeric(38,18);not null"`
	Asset             string                   `gorm:"type:varchar(16);not null"`
	AmountCaptured    decimal.Decimal          `gorm:"type:numeric(38,18);not null;default:0"`
	AmountRefunded    decimal.Decimal          `gorm:"type:numeric(38,18);not null;default:0"`
	CaptureMethod     connectors.CaptureMethod `gorm:"type:varchar(16);not null"`
	PaymentMethodType connectors.Method        `gorm:"type:varchar(16);not null;default:''"`
	PaymentMethod     JSONMap                  `gorm:"type:jsonb;not null"`
	ConnectorCode     connectors.Code          `gorm:"type:varchar(32);not null;default:''"`
	ActiveAttemptID   string                   `gorm:"type:varchar(64);not null;default:''"`
	Description       string                   `gorm:"type:text;not null;default:''"`
	ReturnURL         string                   `gorm:"type:text;not null;default:''"`
	Metadata          JSONMap                  `gorm:"type:jsonb;not null"`
	NextAction        JSONMap                  `gorm:"type:jsonb"`
	LastErrorCode     string                   `gorm:"type:varchar(64);not null;default:''"`
	LastErrorMessage  string                   `gorm:"type:text;not null;default:''"`
	ConfirmRequested  bool                     `gorm:"not null;default:false"`
	PaymentRecordID   uint                     `gorm:"not null;default:0"`
	Version           int64                    `gorm:"not null;default:0"`
	CreatedAt         time.Time                `gorm:"not null"`
	UpdatedAt         time.Time                `gorm:"not null"`
}

func (IntentRow) TableName() string { return "switch_payment_intents" }

type AttemptRow struct {
	ID                     string               `gorm:"type:varchar(64);primarykey"`
	IntentID               string               `gorm:"type:varchar(64);not null;index"`
	MerchantID             string               `gorm:"type:varchar(128);not null;index"`
	ConnectorCode          connectors.Code      `gorm:"type:varchar(32);not null;uniqueIndex:switch_attempts_connector_tx_key,priority:1"`
	Status                 AttemptStatus        `gorm:"type:varchar(32);not null;index"`
	RawStatus              connectors.RawStatus `gorm:"type:varchar(64);not null;default:''"`
	Amount                 decimal.Decimal      `gorm:"type:numeric(38,18);not null"`
	Asset                  string               `gorm:"type:varchar(16);not null"`
	AmountToCapture        decimal.Decimal      `gorm:"type:numeric(38,18);not null;default:0"`
	AmountCaptured         decimal.Decimal      `gorm:"type:numeric(38,18);not null;default:0"`
	AmountReceived         *decimal.Decimal     `gorm:"type:numeric(38,18)"`
	ReceivedAsset          string               `gorm:"type:varchar(32);not null;default:''"`
	ConnectorTransactionID *string              `gorm:"type:varchar(128);uniqueIndex:switch_attempts_connector_tx_key,priority:2"`
	SelectionReason        string               `gorm:"type:text;not null;default:''"`
	ErrorCode              string               `gorm:"type:varchar(64);not null;default:''"`
	ErrorMessage           string               `gorm:"type:text;not null;default:''"`
	NextAction             JSONMap              `gorm:"type:jsonb"`
	SyncCount              int                  `gorm:"not null;default:0"`
	NextSyncAt             *time.Time           `gorm:"index"`
	LastSyncedAt           *time.Time
	// ClaimedUntil is the lease on an in-flight operation; no rollback edge is applied before it expires.
	ClaimedUntil *time.Time
	// StatusChangedAt is when the status last moved; stale age is measured from here, not from creation.
	StatusChangedAt time.Time `gorm:"not null"`
	Version         int64     `gorm:"not null;default:0"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (AttemptRow) TableName() string { return "switch_payment_attempts" }

type RefundRow struct {
	ID                string               `gorm:"type:varchar(64);primarykey"`
	IntentID          string               `gorm:"type:varchar(64);not null;index"`
	AttemptID         string               `gorm:"type:varchar(64);not null;index"`
	MerchantID        string               `gorm:"type:varchar(128);not null;uniqueIndex:switch_refunds_merchant_idempotency_key,priority:1"`
	ConnectorCode     connectors.Code      `gorm:"type:varchar(32);not null;uniqueIndex:switch_refunds_connector_refund_key,priority:1"`
	IdempotencyKey    string               `gorm:"type:varchar(128);not null;uniqueIndex:switch_refunds_merchant_idempotency_key,priority:2"`
	RequestHash       string               `gorm:"type:char(64);not null"`
	Status            RefundStatus         `gorm:"type:varchar(32);not null;index"`
	RawStatus         connectors.RawStatus `gorm:"type:varchar(64);not null;default:''"`
	Amount            decimal.Decimal      `gorm:"type:numeric(38,18);not null"`
	Asset             string               `gorm:"type:varchar(16);not null"`
	ConnectorRefundID *string              `gorm:"type:varchar(128);uniqueIndex:switch_refunds_connector_refund_key,priority:2"`
	Reason            string               `gorm:"type:text;not null;default:''"`
	ErrorCode         string               `gorm:"type:varchar(64);not null;default:''"`
	ErrorMessage      string               `gorm:"type:text;not null;default:''"`
	SyncCount         int                  `gorm:"not null;default:0"`
	NextSyncAt        *time.Time           `gorm:"index"`
	LastSyncedAt      *time.Time
	ClaimedUntil      *time.Time
	StatusChangedAt   time.Time `gorm:"not null"`
	Version           int64     `gorm:"not null;default:0"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

func (RefundRow) TableName() string { return "switch_refunds" }

// AnomalyRow is switch_anomalies: contradictions and stalls recorded for an operator, never turned into a status.
type AnomalyRow struct {
	ID        uint      `gorm:"primarykey"`
	Entity    string    `gorm:"type:varchar(16);not null"`
	EntityID  string    `gorm:"type:varchar(64);not null;index"`
	IntentID  string    `gorm:"type:varchar(64);not null;index"`
	Kind      string    `gorm:"type:varchar(64);not null"`
	Detail    string    `gorm:"type:text;not null;default:''"`
	CreatedAt time.Time `gorm:"not null"`
}

func (AnomalyRow) TableName() string { return "switch_anomalies" }

// WebhookEventRow is the replay guard: (connector_code, event_id) is unique.
type WebhookEventRow struct {
	ID            uint            `gorm:"primarykey"`
	ConnectorCode connectors.Code `gorm:"type:varchar(32);not null;uniqueIndex:switch_webhook_events_connector_event_key,priority:1"`
	EventID       string          `gorm:"type:varchar(255);not null;uniqueIndex:switch_webhook_events_connector_event_key,priority:2"`
	ReceivedAt    time.Time       `gorm:"not null"`
}

func (WebhookEventRow) TableName() string { return "switch_webhook_events" }

// TransitionRow is the audit trail of every status change with the reason it happened.
type TransitionRow struct {
	ID         uint      `gorm:"primarykey"`
	Entity     string    `gorm:"type:varchar(16);not null"`
	EntityID   string    `gorm:"type:varchar(64);not null;index"`
	FromStatus string    `gorm:"type:varchar(32);not null"`
	ToStatus   string    `gorm:"type:varchar(32);not null"`
	Reason     string    `gorm:"type:text;not null;default:''"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (TransitionRow) TableName() string { return "switch_status_transitions" }

// Models lists the GORM models for AutoMigrate (dev/test); production uses the checksummed migration.
func Models() []any {
	return []any{&IntentRow{}, &AttemptRow{}, &RefundRow{}, &WebhookEventRow{}, &TransitionRow{}, &AnomalyRow{}}
}

func (r IntentRow) toIntent() Intent {
	out := Intent{
		ID:                r.ID,
		MerchantID:        r.MerchantID,
		PlatformID:        r.PlatformID,
		IdempotencyKey:    r.IdempotencyKey,
		Status:            r.Status,
		Money:             Money{Amount: r.Amount, Asset: r.Asset},
		AmountCaptured:    r.AmountCaptured,
		AmountRefunded:    r.AmountRefunded,
		CaptureMethod:     r.CaptureMethod,
		PaymentMethodType: r.PaymentMethodType,
		ConnectorCode:     r.ConnectorCode,
		ActiveAttemptID:   r.ActiveAttemptID,
		Description:       r.Description,
		ReturnURL:         r.ReturnURL,
		Metadata:          stringMap(r.Metadata),
		NextAction:        nextActionFrom(r.NextAction),
		LastErrorCode:     r.LastErrorCode,
		LastErrorMessage:  r.LastErrorMessage,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
	return out
}

func (r AttemptRow) toAttempt() Attempt {
	out := Attempt{
		ID:              r.ID,
		IntentID:        r.IntentID,
		MerchantID:      r.MerchantID,
		ConnectorCode:   r.ConnectorCode,
		Status:          r.Status,
		RawStatus:       r.RawStatus,
		Money:           Money{Amount: r.Amount, Asset: r.Asset},
		AmountToCapture: r.AmountToCapture,
		AmountCaptured:  r.AmountCaptured,
		AmountReceived:  r.AmountReceived,
		ReceivedAsset:   r.ReceivedAsset,
		SelectionReason: r.SelectionReason,
		ErrorCode:       r.ErrorCode,
		ErrorMessage:    r.ErrorMessage,
		NextAction:      nextActionFrom(r.NextAction),
		ClaimedUntil:    r.ClaimedUntil,
		StatusChangedAt: r.StatusChangedAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
	if r.ConnectorTransactionID != nil {
		out.ConnectorTransactionID = *r.ConnectorTransactionID
	}
	return out
}

func (r RefundRow) toRefund() Refund {
	out := Refund{
		ID:             r.ID,
		IntentID:       r.IntentID,
		AttemptID:      r.AttemptID,
		MerchantID:     r.MerchantID,
		ConnectorCode:  r.ConnectorCode,
		IdempotencyKey: r.IdempotencyKey,
		Status:         r.Status,
		RawStatus:      r.RawStatus,
		Money:          Money{Amount: r.Amount, Asset: r.Asset},
		Reason:         r.Reason,
		ErrorCode:      r.ErrorCode,
		ErrorMessage:   r.ErrorMessage,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	if r.ConnectorRefundID != nil {
		out.ConnectorRefundID = *r.ConnectorRefundID
	}
	return out
}

func stringMap(m JSONMap) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func jsonMapOfStrings(m map[string]string) JSONMap {
	out := JSONMap{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func nextActionJSON(a *connectors.NextAction) JSONMap {
	if a == nil {
		return nil
	}
	raw, _ := json.Marshal(a)
	var m JSONMap
	_ = json.Unmarshal(raw, &m)
	return m
}

func nextActionFrom(m JSONMap) *connectors.NextAction {
	if len(m) == 0 {
		return nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var a connectors.NextAction
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil
	}
	return &a
}

func paymentMethodJSON(pm connectors.PaymentMethod) JSONMap {
	m := JSONMap{"type": string(pm.Type), "token": pm.Token}
	if len(pm.Details) > 0 {
		m["details"] = jsonMapOfStrings(pm.Details)
	}
	return m
}

func paymentMethodFrom(m JSONMap) (connectors.PaymentMethod, bool) {
	if len(m) == 0 {
		return connectors.PaymentMethod{}, false
	}
	pm := connectors.PaymentMethod{}
	if t, ok := m["type"].(string); ok {
		pm.Type = connectors.Method(t)
	}
	if t, ok := m["token"].(string); ok {
		pm.Token = t
	}
	if d, ok := m["details"].(map[string]any); ok {
		pm.Details = stringMap(JSONMap(d))
	}
	return pm, pm.Type != ""
}
