package paymentswitch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxKeyLen   = 128
	maxAssetLen = 16
	maxIDLen    = 128
	maxScale    = 18
)

var maxMagnitude = decimal.New(1, 20)

// Service is the only writer of switch rows. Connector calls happen outside transactions; every state change
// is a short transaction with a version check. Lock order everywhere: intent, then attempt, then refund.
type Service struct {
	db         *gorm.DB
	connectors connectors.Lookup
	selector   ConnectorSelector
	ledger     Ledger
	events     Events
	now        func() time.Time
	newID      func(prefix string) string
	logf       func(format string, args ...any)
}

type Option func(*Service)

func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }
func WithIDs(gen func(prefix string) string) Option {
	return func(s *Service) { s.newID = gen }
}
func WithEvents(e Events) Option { return func(s *Service) { s.events = e } }
func WithLogger(logf func(format string, args ...any)) Option {
	return func(s *Service) { s.logf = logf }
}

func New(db *gorm.DB, lookup connectors.Lookup, selector ConnectorSelector, ledger Ledger, opts ...Option) *Service {
	s := &Service{
		db:         db,
		connectors: lookup,
		selector:   selector,
		ledger:     ledger,
		events:     NoEvents{},
		now:        func() time.Time { return time.Now().UTC() },
		newID:      func(prefix string) string { return prefix + "_" + strings.ReplaceAll(uuid.NewString(), "-", "") },
		logf:       log.Printf,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Migrate creates the switch tables and constraints for development and test; production uses the
// checksummed migration 2026100703_switch_intents_attempts.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("paymentswitch: automigrate: %w", err)
	}
	return InstallConstraints(db)
}

// CreateCommand opens an intent. An empty IdempotencyKey gets a generated one, so every intent has a key.
type CreateCommand struct {
	MerchantID     string
	PlatformID     string
	IdempotencyKey string
	Money          Money
	CaptureMethod  connectors.CaptureMethod
	PaymentMethod  *connectors.PaymentMethod
	Confirm        bool
	Description    string
	ReturnURL      string
	Metadata       map[string]string
}

type ConfirmCommand struct {
	PaymentMethod *connectors.PaymentMethod
}

type CaptureCommand struct {
	Amount *decimal.Decimal
}

type CancelCommand struct {
	Reason string
}

type RefundCommand struct {
	IdempotencyKey string
	Amount         *decimal.Decimal
	Reason         string
}

func (c CreateCommand) validate() error {
	if strings.TrimSpace(c.MerchantID) == "" || len(c.MerchantID) > maxIDLen {
		return fmt.Errorf("%w: merchant id", ErrInvalid)
	}
	if len(c.IdempotencyKey) > maxKeyLen {
		return fmt.Errorf("%w: idempotency key longer than %d", ErrInvalid, maxKeyLen)
	}
	if err := validateMoney(c.Money); err != nil {
		return err
	}
	switch c.CaptureMethod {
	case connectors.CaptureAutomatic, connectors.CaptureManual:
	default:
		return fmt.Errorf("%w: capture method %q", ErrInvalid, c.CaptureMethod)
	}
	if c.PaymentMethod != nil && c.PaymentMethod.Type == "" {
		return fmt.Errorf("%w: payment method type required", ErrInvalid)
	}
	if c.Confirm && c.PaymentMethod == nil {
		return fmt.Errorf("%w: confirm requires a payment method", ErrInvalid)
	}
	return nil
}

func validateMoney(m Money) error {
	if err := validateAmount(m.Amount); err != nil {
		return err
	}
	if n := len(strings.TrimSpace(m.Asset)); n == 0 || n > maxAssetLen || n != len(m.Asset) {
		return fmt.Errorf("%w: asset %q", ErrInvalid, m.Asset)
	}
	return nil
}

// validateAmount mirrors the ledger's numeric(38,18) rules so nothing is rounded silently on the way in (M1).
func validateAmount(a decimal.Decimal) error {
	if !a.IsPositive() {
		return fmt.Errorf("%w: amount must be positive", ErrInvalid)
	}
	if !a.Equal(a.Truncate(maxScale)) {
		return fmt.Errorf("%w: amount has more than %d decimal places", ErrInvalid, maxScale)
	}
	if a.GreaterThanOrEqual(maxMagnitude) {
		return fmt.Errorf("%w: amount magnitude must be below 1e20", ErrInvalid)
	}
	return nil
}

type canonicalCreate struct {
	MerchantID    string            `json:"merchant_id"`
	Amount        string            `json:"amount"`
	Asset         string            `json:"asset"`
	CaptureMethod string            `json:"capture_method"`
	PaymentMethod map[string]any    `json:"payment_method,omitempty"`
	Confirm       bool              `json:"confirm"`
	Description   string            `json:"description"`
	ReturnURL     string            `json:"return_url"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// requestHash is the canonical fingerprint stored with the row so a replayed key is checked against its original.
func (c CreateCommand) requestHash() string {
	payload := canonicalCreate{
		MerchantID:    c.MerchantID,
		Amount:        c.Money.Amount.String(),
		Asset:         c.Money.Asset,
		CaptureMethod: string(c.CaptureMethod),
		Confirm:       c.Confirm,
		Description:   c.Description,
		ReturnURL:     c.ReturnURL,
		Metadata:      c.Metadata,
	}
	if c.PaymentMethod != nil {
		payload.PaymentMethod = paymentMethodJSON(*c.PaymentMethod)
	}
	return hashJSON(payload)
}

func hashJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		raw = []byte("unmarshalable:" + err.Error())
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Create opens an intent. Same key and same body returns the existing intent; same key and a different body
// is ErrIdempotencyConflict. With Confirm it also runs the first attempt; if that fails before a claim, the
// created intent is returned alongside the error, and a replay re-runs the confirm (M4).
func (s *Service) Create(ctx context.Context, cmd CreateCommand) (Intent, error) {
	if cmd.CaptureMethod == "" {
		cmd.CaptureMethod = connectors.CaptureAutomatic
	}
	if err := cmd.validate(); err != nil {
		return Intent{}, err
	}
	if cmd.IdempotencyKey == "" {
		cmd.IdempotencyKey = s.newID("idem")
	}
	hash := cmd.requestHash()
	now := s.now()
	row := IntentRow{
		ID:               s.newID("pi"),
		MerchantID:       cmd.MerchantID,
		PlatformID:       cmd.PlatformID,
		IdempotencyKey:   cmd.IdempotencyKey,
		RequestHash:      hash,
		Status:           IntentRequiresPaymentMethod,
		Amount:           cmd.Money.Amount,
		Asset:            cmd.Money.Asset,
		CaptureMethod:    cmd.CaptureMethod,
		PaymentMethod:    JSONMap{},
		Description:      cmd.Description,
		ReturnURL:        cmd.ReturnURL,
		Metadata:         jsonMapOfStrings(cmd.Metadata),
		ConfirmRequested: cmd.Confirm,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if cmd.PaymentMethod != nil {
		row.Status = IntentRequiresConfirmation
		row.PaymentMethodType = cmd.PaymentMethod.Type
		row.PaymentMethod = paymentMethodJSON(*cmd.PaymentMethod)
	}

	var replayed bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "merchant_id"}, {Name: "idempotency_key"}},
			DoNothing: true,
		}).Create(&row)
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: insert intent: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			var existing IntentRow
			if err := tx.Where("merchant_id = ? AND idempotency_key = ?", cmd.MerchantID, cmd.IdempotencyKey).First(&existing).Error; err != nil {
				return fmt.Errorf("paymentswitch: load intent for key: %w", err)
			}
			if existing.RequestHash != hash {
				return fmt.Errorf("%w: key %q", ErrIdempotencyConflict, cmd.IdempotencyKey)
			}
			row = existing
			replayed = true
			return nil
		}
		return s.recordTransition(tx, "intent", row.ID, "", string(row.Status), "create")
	})
	if err != nil {
		return Intent{}, err
	}
	needsConfirm := row.ConfirmRequested && row.ActiveAttemptID == "" && confirmable(row.Status)
	if !replayed && cmd.Confirm || replayed && needsConfirm {
		confirmed, err := s.Confirm(ctx, cmd.MerchantID, row.ID, ConfirmCommand{})
		if err != nil {
			return row.toIntent(), err
		}
		return confirmed, nil
	}
	return row.toIntent(), nil
}

// Get returns the intent with its attempts and refunds, scoped to the merchant.
func (s *Service) Get(ctx context.Context, merchantID, intentID string) (View, error) {
	db := s.db.WithContext(ctx)
	row, err := loadIntent(db, merchantID, intentID)
	if err != nil {
		return View{}, err
	}
	var attempts []AttemptRow
	if err := db.Where("intent_id = ?", intentID).Order("created_at, id").Find(&attempts).Error; err != nil {
		return View{}, fmt.Errorf("paymentswitch: load attempts: %w", err)
	}
	var refunds []RefundRow
	if err := db.Where("intent_id = ?", intentID).Order("created_at, id").Find(&refunds).Error; err != nil {
		return View{}, fmt.Errorf("paymentswitch: load refunds: %w", err)
	}
	view := View{Intent: row.toIntent()}
	for _, a := range attempts {
		view.Attempts = append(view.Attempts, a.toAttempt())
	}
	for _, r := range refunds {
		view.Refunds = append(view.Refunds, r.toRefund())
	}
	return view, nil
}

// Anomalies lists what the switch recorded rather than acted on for one intent.
func (s *Service) Anomalies(ctx context.Context, merchantID, intentID string) ([]Anomaly, error) {
	if _, err := loadIntent(s.db.WithContext(ctx), merchantID, intentID); err != nil {
		return nil, err
	}
	var rows []AnomalyRow
	if err := s.db.WithContext(ctx).Where("intent_id = ?", intentID).Order("id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("paymentswitch: load anomalies: %w", err)
	}
	out := make([]Anomaly, 0, len(rows))
	for _, r := range rows {
		out = append(out, Anomaly{ID: r.ID, Entity: r.Entity, EntityID: r.EntityID, IntentID: r.IntentID, Kind: r.Kind, Detail: r.Detail, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func loadIntent(db *gorm.DB, merchantID, intentID string) (IntentRow, error) {
	var row IntentRow
	err := db.Where("id = ? AND merchant_id = ?", intentID, merchantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return IntentRow{}, fmt.Errorf("%w: intent %s", ErrNotFound, intentID)
	}
	if err != nil {
		return IntentRow{}, fmt.Errorf("paymentswitch: load intent: %w", err)
	}
	return row, nil
}

func loadAttempt(db *gorm.DB, attemptID string) (AttemptRow, error) {
	var row AttemptRow
	err := db.Where("id = ?", attemptID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AttemptRow{}, fmt.Errorf("%w: attempt %s", ErrNotFound, attemptID)
	}
	if err != nil {
		return AttemptRow{}, fmt.Errorf("paymentswitch: load attempt: %w", err)
	}
	return row, nil
}

func loadRefund(db *gorm.DB, refundID string) (RefundRow, error) {
	var row RefundRow
	err := db.Where("id = ?", refundID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return RefundRow{}, fmt.Errorf("%w: refund %s", ErrNotFound, refundID)
	}
	if err != nil {
		return RefundRow{}, fmt.Errorf("paymentswitch: load refund: %w", err)
	}
	return row, nil
}

// lockIfPostgres takes a row lock inside a transaction; SQLite (unit tests) serialises writers itself.
func lockIfPostgres(tx *gorm.DB) *gorm.DB {
	if tx.Dialector.Name() == "postgres" {
		return tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return tx
}

func (s *Service) recordTransition(tx *gorm.DB, entity, id, from, to, reason string) error {
	if from == to {
		return nil
	}
	row := TransitionRow{Entity: entity, EntityID: id, FromStatus: from, ToStatus: to, Reason: reason, CreatedAt: s.now()}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("paymentswitch: record transition: %w", err)
	}
	return nil
}

// recordNote writes an audit row for something that happened without a status change (an unknown outcome).
func (s *Service) recordNote(tx *gorm.DB, entity, id, status, reason string) error {
	row := TransitionRow{Entity: entity, EntityID: id, FromStatus: status, ToStatus: status, Reason: reason, CreatedAt: s.now()}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("paymentswitch: record note: %w", err)
	}
	return nil
}

// recordAnomaly writes the contradiction and logs it at error level; once per (entity, kind) unless detail differs.
func (s *Service) recordAnomaly(tx *gorm.DB, entity, id, intentID, kind, detail string) error {
	var n int64
	if err := tx.Model(&AnomalyRow{}).Where("entity_id = ? AND kind = ? AND detail = ?", id, kind, detail).Count(&n).Error; err != nil {
		return fmt.Errorf("paymentswitch: check anomaly: %w", err)
	}
	if n > 0 {
		return nil
	}
	s.logf("[paymentswitch] ERROR anomaly %s on %s %s (intent %s): %s", kind, entity, id, intentID, detail)
	row := AnomalyRow{Entity: entity, EntityID: id, IntentID: intentID, Kind: kind, Detail: detail, CreatedAt: s.now()}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("paymentswitch: record anomaly: %w", err)
	}
	return nil
}

// saveIntent writes the row only if nobody else changed it since it was read.
func (s *Service) saveIntent(tx *gorm.DB, row *IntentRow) error {
	expected := row.Version
	row.Version++
	row.UpdatedAt = s.now()
	res := tx.Model(&IntentRow{}).Where("id = ? AND version = ?", row.ID, expected).Select("*").Omit("id", "created_at").Updates(row)
	if res.Error != nil {
		return fmt.Errorf("paymentswitch: save intent: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: intent %s", ErrConcurrentUpdate, row.ID)
	}
	return nil
}

func (s *Service) saveAttempt(tx *gorm.DB, row *AttemptRow) error {
	expected := row.Version
	row.Version++
	row.UpdatedAt = s.now()
	res := tx.Model(&AttemptRow{}).Where("id = ? AND version = ?", row.ID, expected).Select("*").Omit("id", "created_at").Updates(row)
	if res.Error != nil {
		return fmt.Errorf("paymentswitch: save attempt: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: attempt %s", ErrConcurrentUpdate, row.ID)
	}
	return nil
}

func (s *Service) saveRefund(tx *gorm.DB, row *RefundRow) error {
	expected := row.Version
	row.Version++
	row.UpdatedAt = s.now()
	res := tx.Model(&RefundRow{}).Where("id = ? AND version = ?", row.ID, expected).Select("*").Omit("id", "created_at").Updates(row)
	if res.Error != nil {
		return fmt.Errorf("paymentswitch: save refund: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: refund %s", ErrConcurrentUpdate, row.ID)
	}
	return nil
}

func (s *Service) connector(code connectors.Code) (connectors.Connector, error) {
	c, ok := s.connectors.Get(code)
	if !ok {
		return nil, fmt.Errorf("%w: %q", connectors.ErrUnknownConnector, code)
	}
	return c, nil
}

// emit publishes events after commit; a failure to enqueue is logged, never fails the money path.
func (s *Service) emit(ctx context.Context, events []Event) {
	for _, ev := range events {
		if err := s.events.Emit(ctx, ev); err != nil {
			s.logf("[paymentswitch] ERROR emit %s for intent %s: %v", ev.Type, ev.IntentID, err)
		}
	}
}

// redact classifies a connector error for the merchant; the raw text goes to the transition reason and the log.
func redact(err error) (code, message string) {
	switch {
	case errors.Is(err, connectors.ErrDeclined):
		return ErrorCodeDeclined, "the provider refused the operation"
	case errors.Is(err, connectors.ErrTimeout), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return ErrorCodeTimeout, "the provider did not answer in time; the switch will sync the outcome"
	case errors.Is(err, connectors.ErrNotFound):
		return ErrorCodeNotFound, "the provider does not know this transaction yet; the switch will sync"
	default:
		return ErrorCodeConnector, "the provider returned an error; the switch will sync the outcome"
	}
}

// backoff is the reconciler's schedule: base doubling per sync, capped.
func backoff(count int, base time.Duration) time.Duration {
	if count > 8 {
		count = 8
	}
	return base * time.Duration(1<<count)
}
