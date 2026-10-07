package links

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed schema.sql
var schemaSQL string

// SchemaSQL is the DDL migration 2026100710 must repeat verbatim.
func SchemaSQL() string { return schemaSQL }

// Migrate creates the link tables in dev/test; Postgres only.
func Migrate(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if err := db.Exec(schemaSQL).Error; err != nil {
		return fmt.Errorf("links: migrate: %w", err)
	}
	return nil
}

// PGStore is the Postgres Store.
type PGStore struct {
	db *gorm.DB
}

var _ Store = (*PGStore)(nil)

func NewPGStore(db *gorm.DB) *PGStore { return &PGStore{db: db} }

type linkRow struct {
	ID                   string `gorm:"primaryKey"`
	MemberID             uint
	ExternalPlatformID   uint
	Environment          string
	Title                string
	Description          string
	AmountMode           string
	Amount               decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	AmountMin            decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	AmountMax            decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	Currency             string
	ReferenceID          string
	Metadata             string `gorm:"type:jsonb"`
	Category             string
	CustomerFieldPolicy  string `gorm:"column:customer_field_policy;type:jsonb"`
	BillingRequired      bool
	ShippingRequired     bool
	MultiUse             bool
	UseLimit             *int
	Methods              string `gorm:"type:jsonb"`
	CaptureMode          string
	ThreeDSPolicy        string `gorm:"column:three_ds_policy"`
	ChainToleranceBps    int
	QuoteExpirySeconds   int
	FeeBearer            string
	SuccessMode          string
	SuccessURL           string `gorm:"column:success_url"`
	SuccessMessage       string
	ReceiptEmail         bool
	ReceiptNote          string
	WebhookID            *uint
	FailureRetry         bool
	FailureMessage       string
	SettlementOverride   *string `gorm:"type:jsonb"`
	HoldInAsset          bool
	SettlementTiming     string
	ExpiresAt            *time.Time
	ExpiresAfterPayments *int
	Status               string
	LogoURL              string `gorm:"column:logo_url"`
	AccentColor          string
	Language             string
	ShortCode            *string
	UsesCount            int
	Revision             int
	PublishedAt          *time.Time
	CreatedAt            time.Time `gorm:"autoCreateTime:false"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime:false"`
}

func (linkRow) TableName() string { return "payment_links" }

type lineItemRow struct {
	ID        uint `gorm:"primaryKey"`
	LinkID    string
	Position  int
	Name      string
	Quantity  int
	UnitPrice decimal.Decimal `gorm:"type:numeric(38,18)"`
	TaxRate   decimal.Decimal `gorm:"type:numeric(9,6)"`
}

func (lineItemRow) TableName() string { return "payment_link_line_items" }

type questionRow struct {
	ID       uint `gorm:"primaryKey"`
	LinkID   string
	Position int
	Key      string
	Label    string
	Type     string
	Options  string `gorm:"type:jsonb"`
	Required bool
	PerOrder bool
}

func (questionRow) TableName() string { return "payment_link_questions" }

type paymentRow struct {
	ID                string `gorm:"primaryKey"`
	LinkID            string
	IdempotencyKey    string
	RequestHash       string
	Status            string
	Environment       string
	Method            string
	Chain             string
	Asset             string
	Connector         string
	Amount            decimal.Decimal `gorm:"type:numeric(38,18)"`
	Currency          string
	Fee               decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	Tax               decimal.NullDecimal `gorm:"type:numeric(38,18)"`
	CustomerTotal     decimal.Decimal     `gorm:"type:numeric(38,18)"`
	FeeBearer         string
	FeeRuleID         *uint
	FeeRuleVersion    *int
	CustomerName      string
	CustomerEmail     string
	CustomerPhone     string
	BillingAddress    *string `gorm:"type:jsonb"`
	ShippingAddress   *string `gorm:"type:jsonb"`
	PaymentReference  *string
	ProcessorResponse *string `gorm:"type:jsonb"`
	ClientKey         string
	ReservedUntil     time.Time
	OpenUntil         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (paymentRow) TableName() string { return "payment_link_payments" }

type answerRow struct {
	ID            uint `gorm:"primaryKey"`
	LinkPaymentID string
	LinkID        string
	QuestionKey   string
	QuestionLabel string
	Value         string
}

func (answerRow) TableName() string { return "payment_link_answers" }

func toJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	return string(raw), err
}

func nullDec(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{Decimal: *d, Valid: true}
}

func fromNull(d decimal.NullDecimal) *decimal.Decimal {
	if !d.Valid {
		return nil
	}
	v := d.Decimal
	return &v
}

func optJSON(v any, present bool) (*string, error) {
	if !present {
		return nil, nil
	}
	s, err := toJSON(v)
	return &s, err
}

func toRow(l Link) (linkRow, error) {
	r := linkRow{
		ID: l.ID, MemberID: l.MemberID, ExternalPlatformID: l.ExternalPlatformID, Environment: string(l.Environment),
		Title: l.Title, Description: l.Description, AmountMode: string(l.AmountMode),
		Amount: nullDec(l.Total), AmountMin: nullDec(l.AmountMin), AmountMax: nullDec(l.AmountMax),
		Currency: l.Currency, ReferenceID: l.ReferenceID, Category: l.Category,
		BillingRequired: l.BillingRequired, ShippingRequired: l.ShippingRequired, MultiUse: l.MultiUse, UseLimit: l.UseLimit,
		CaptureMode: l.CaptureMode, ThreeDSPolicy: l.ThreeDSPolicy, ChainToleranceBps: l.ChainToleranceBps,
		QuoteExpirySeconds: l.QuoteExpirySeconds, FeeBearer: string(l.FeeBearer), SuccessMode: l.SuccessMode,
		SuccessURL: l.SuccessURL, SuccessMessage: l.SuccessMessage, ReceiptEmail: l.ReceiptEmail, ReceiptNote: l.ReceiptNote,
		WebhookID: l.WebhookID, FailureRetry: l.FailureRetry, FailureMessage: l.FailureMessage, HoldInAsset: l.HoldInAsset,
		SettlementTiming: l.SettlementTiming, ExpiresAt: l.ExpiresAt, ExpiresAfterPayments: l.ExpiresAfterPayments,
		Status: string(l.Status), LogoURL: l.LogoURL, AccentColor: l.AccentColor, Language: l.Language,
		UsesCount: l.UsesCount, Revision: l.Revision, PublishedAt: l.PublishedAt,
	}
	if l.ShortCode != "" {
		code := l.ShortCode
		r.ShortCode = &code
	}
	var err error
	if r.Metadata, err = toJSON(l.Metadata); err != nil {
		return r, err
	}
	if r.CustomerFieldPolicy, err = toJSON(l.CustomerFields); err != nil {
		return r, err
	}
	if r.Methods, err = toJSON(l.Methods); err != nil {
		return r, err
	}
	r.SettlementOverride, err = optJSON(l.SettlementOverride, l.SettlementOverride != nil)
	return r, err
}

func fromRow(r linkRow, items []lineItemRow, qs []questionRow) (Link, error) {
	l := Link{
		ID: r.ID, MemberID: r.MemberID, ExternalPlatformID: r.ExternalPlatformID, Environment: Environment(r.Environment),
		Status: Status(r.Status), UsesCount: r.UsesCount, Revision: r.Revision, PublishedAt: r.PublishedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Total: fromNull(r.Amount),
		Input: Input{
			Title: r.Title, Description: r.Description, AmountMode: AmountMode(r.AmountMode),
			AmountMin: fromNull(r.AmountMin), AmountMax: fromNull(r.AmountMax), Currency: r.Currency,
			ReferenceID: r.ReferenceID, Category: r.Category, BillingRequired: r.BillingRequired,
			ShippingRequired: r.ShippingRequired, MultiUse: r.MultiUse, UseLimit: r.UseLimit,
			CaptureMode: r.CaptureMode, ThreeDSPolicy: r.ThreeDSPolicy, ChainToleranceBps: r.ChainToleranceBps,
			QuoteExpirySeconds: r.QuoteExpirySeconds, FeeBearer: fees.FeeBearer(r.FeeBearer), SuccessMode: r.SuccessMode,
			SuccessURL: r.SuccessURL, SuccessMessage: r.SuccessMessage, ReceiptEmail: r.ReceiptEmail, ReceiptNote: r.ReceiptNote,
			WebhookID: r.WebhookID, FailureRetry: r.FailureRetry, FailureMessage: r.FailureMessage, HoldInAsset: r.HoldInAsset,
			SettlementTiming: r.SettlementTiming, ExpiresAt: r.ExpiresAt, ExpiresAfterPayments: r.ExpiresAfterPayments,
			LogoURL: r.LogoURL, AccentColor: r.AccentColor, Language: r.Language,
			LineItems: make([]LineItem, len(items)), Questions: make([]Question, len(qs)),
		},
	}
	if r.ShortCode != nil {
		l.ShortCode = *r.ShortCode
	}
	if l.AmountMode == AmountFixed {
		l.Amount = fromNull(r.Amount)
	}
	for _, j := range []struct {
		raw string
		dst any
	}{{r.Metadata, &l.Metadata}, {r.CustomerFieldPolicy, &l.CustomerFields}, {r.Methods, &l.Methods}} {
		if err := json.Unmarshal([]byte(j.raw), j.dst); err != nil {
			return Link{}, fmt.Errorf("links: decode link %s: %w", r.ID, err)
		}
	}
	if r.SettlementOverride != nil {
		l.SettlementOverride = &SettlementOverride{}
		if err := json.Unmarshal([]byte(*r.SettlementOverride), l.SettlementOverride); err != nil {
			return Link{}, fmt.Errorf("links: decode settlement override %s: %w", r.ID, err)
		}
	}
	for i, it := range items {
		l.LineItems[i] = LineItem{Name: it.Name, Quantity: it.Quantity, UnitPrice: it.UnitPrice, TaxRate: it.TaxRate}
	}
	for i, q := range qs {
		l.Questions[i] = Question{Key: q.Key, Label: q.Label, Type: QuestionType(q.Type), Required: q.Required, PerOrder: q.PerOrder}
		if err := json.Unmarshal([]byte(q.Options), &l.Questions[i].Options); err != nil {
			return Link{}, fmt.Errorf("links: decode question options %s: %w", r.ID, err)
		}
		if len(l.Questions[i].Options) == 0 {
			l.Questions[i].Options = nil
		}
	}
	return l, nil
}

func (s *PGStore) children(tx *gorm.DB, ids []string) (map[string][]lineItemRow, map[string][]questionRow, error) {
	items := map[string][]lineItemRow{}
	qs := map[string][]questionRow{}
	if len(ids) == 0 {
		return items, qs, nil
	}
	var ir []lineItemRow
	if err := tx.Where("link_id IN ?", ids).Order("link_id, position").Find(&ir).Error; err != nil {
		return nil, nil, fmt.Errorf("links: load line items: %w", err)
	}
	for _, r := range ir {
		items[r.LinkID] = append(items[r.LinkID], r)
	}
	var qr []questionRow
	if err := tx.Where("link_id IN ?", ids).Order("link_id, position").Find(&qr).Error; err != nil {
		return nil, nil, fmt.Errorf("links: load questions: %w", err)
	}
	for _, r := range qr {
		qs[r.LinkID] = append(qs[r.LinkID], r)
	}
	return items, qs, nil
}

func (s *PGStore) loadRows(tx *gorm.DB, rows []linkRow) ([]Link, error) {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	items, qs, err := s.children(tx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Link, len(rows))
	for i, r := range rows {
		if out[i], err = fromRow(r, items[r.ID], qs[r.ID]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *PGStore) loadOne(tx *gorm.DB, query string, args ...any) (Link, error) {
	var rows []linkRow
	if err := tx.Where(query, args...).Limit(1).Find(&rows).Error; err != nil {
		return Link{}, fmt.Errorf("links: load link: %w", err)
	}
	if len(rows) == 0 {
		return Link{}, ErrStoreNotFound
	}
	out, err := s.loadRows(tx, rows)
	if err != nil {
		return Link{}, err
	}
	return out[0], nil
}

func writeChildren(tx *gorm.DB, l Link) error {
	if err := tx.Where("link_id = ?", l.ID).Delete(&lineItemRow{}).Error; err != nil {
		return fmt.Errorf("links: clear line items: %w", err)
	}
	if err := tx.Where("link_id = ?", l.ID).Delete(&questionRow{}).Error; err != nil {
		return fmt.Errorf("links: clear questions: %w", err)
	}
	for i, li := range l.LineItems {
		row := lineItemRow{LinkID: l.ID, Position: i, Name: li.Name, Quantity: li.Quantity, UnitPrice: li.UnitPrice, TaxRate: li.TaxRate}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("links: insert line item: %w", err)
		}
	}
	for i, q := range l.Questions {
		opts := q.Options
		if opts == nil {
			opts = []string{}
		}
		raw, err := toJSON(opts)
		if err != nil {
			return err
		}
		row := questionRow{LinkID: l.ID, Position: i, Key: q.Key, Label: q.Label, Type: string(q.Type), Options: raw, Required: q.Required, PerOrder: q.PerOrder}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("links: insert question: %w", err)
		}
	}
	return nil
}

func (s *PGStore) Insert(ctx context.Context, l Link) (Link, error) {
	l.ID, l.Revision = uuid.NewString(), 1
	row, err := toRow(l)
	if err != nil {
		return Link{}, err
	}
	var out Link
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now, err := txNow(tx)
		if err != nil {
			return err
		}
		row.CreatedAt, row.UpdatedAt = now, now
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("links: insert link: %w", err)
		}
		if err := writeChildren(tx, l); err != nil {
			return err
		}
		out, err = s.loadOne(tx, "id = ?", l.ID)
		return err
	})
	return out, err
}

func txNow(tx *gorm.DB) (time.Time, error) {
	var now time.Time
	if err := tx.Raw(`SELECT now()`).Scan(&now).Error; err != nil {
		return time.Time{}, fmt.Errorf("links: database time: %w", err)
	}
	return now.UTC(), nil
}

func (s *PGStore) Get(ctx context.Context, platformID uint, id string) (Link, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Link{}, ErrStoreNotFound
	}
	return s.loadOne(s.db.WithContext(ctx), "id = ? AND external_platform_id = ?", id, platformID)
}

func (s *PGStore) GetByShortCode(ctx context.Context, code string) (Link, error) {
	return s.loadOne(s.db.WithContext(ctx), "short_code = ?", code)
}

func (s *PGStore) List(ctx context.Context, platformID uint, f ListFilter) ([]Link, int64, error) {
	db := s.db.WithContext(ctx)
	q := db.Model(&linkRow{}).Where("external_platform_id = ?", platformID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("links: count: %w", err)
	}
	var rows []linkRow
	if err := q.Order("created_at DESC, id DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("links: list: %w", err)
	}
	out, err := s.loadRows(db, rows)
	return out, total, err
}

// classifyMiss tells a missing link from one whose revision moved on, after a conditional write touched no row.
func classifyMiss(tx *gorm.DB, platformID uint, id string) error {
	var n int64
	if err := tx.Model(&linkRow{}).Where("id = ? AND external_platform_id = ?", id, platformID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrStoreNotFound
	}
	return ErrStale
}

func (s *PGStore) Save(ctx context.Context, l Link) (Link, error) {
	row, err := toRow(l)
	if err != nil {
		return Link{}, err
	}
	var out Link
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&linkRow{}).Where("id = ? AND external_platform_id = ? AND revision = ?", l.ID, l.ExternalPlatformID, l.Revision).
			Updates(map[string]any{
				"title": row.Title, "description": row.Description, "amount_mode": row.AmountMode, "amount": row.Amount,
				"amount_min": row.AmountMin, "amount_max": row.AmountMax, "currency": row.Currency, "reference_id": row.ReferenceID,
				"metadata": row.Metadata, "category": row.Category, "customer_field_policy": row.CustomerFieldPolicy,
				"billing_required": row.BillingRequired, "shipping_required": row.ShippingRequired, "multi_use": row.MultiUse,
				"use_limit": row.UseLimit, "methods": row.Methods, "capture_mode": row.CaptureMode, "three_ds_policy": row.ThreeDSPolicy,
				"chain_tolerance_bps": row.ChainToleranceBps, "quote_expiry_seconds": row.QuoteExpirySeconds,
				"fee_bearer": row.FeeBearer, "success_mode": row.SuccessMode, "success_url": row.SuccessURL,
				"success_message": row.SuccessMessage, "receipt_email": row.ReceiptEmail, "receipt_note": row.ReceiptNote,
				"webhook_id": row.WebhookID, "failure_retry": row.FailureRetry, "failure_message": row.FailureMessage,
				"settlement_override": row.SettlementOverride, "hold_in_asset": row.HoldInAsset,
				"settlement_timing": row.SettlementTiming, "expires_at": row.ExpiresAt,
				"expires_after_payments": row.ExpiresAfterPayments, "logo_url": row.LogoURL, "accent_color": row.AccentColor,
				"language": row.Language, "revision": gorm.Expr("revision + 1"), "updated_at": gorm.Expr("now()"),
			})
		if res.Error != nil {
			return fmt.Errorf("links: save link: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return classifyMiss(tx, l.ExternalPlatformID, l.ID)
		}
		if err := writeChildren(tx, l); err != nil {
			return err
		}
		out, err = s.loadOne(tx, "id = ?", l.ID)
		return err
	})
	return out, err
}

func (s *PGStore) SetStatus(ctx context.Context, platformID uint, id string, revision int, to Status, code string, now time.Time) (Link, error) {
	var out Link
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`UPDATE payment_links SET status = ?,
				short_code = COALESCE(short_code, NULLIF(?, '')),
				published_at = CASE WHEN ? = 'active' AND published_at IS NULL THEN ? ELSE published_at END,
				revision = revision + 1, updated_at = now()
			WHERE id = ? AND external_platform_id = ? AND revision = ?`,
			string(to), code, string(to), now, id, platformID, revision)
		if res.Error != nil {
			var pg *pgconn.PgError
			if errors.As(res.Error, &pg) && pg.Code == "23505" && pg.ConstraintName == "payment_links_short_code_key" {
				return ErrShortCodeTaken
			}
			return fmt.Errorf("links: set status: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return classifyMiss(tx, platformID, id)
		}
		var err error
		out, err = s.loadOne(tx, "id = ?", id)
		return err
	})
	return out, err
}

func (s *PGStore) DeleteDraft(ctx context.Context, platformID uint, id string, revision int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND external_platform_id = ? AND revision = ? AND status = 'draft'", id, platformID, revision).Delete(&linkRow{})
		if res.Error != nil {
			return fmt.Errorf("links: delete draft: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return classifyMiss(tx, platformID, id)
		}
		return nil
	})
}

func (s *PGStore) WebhookExists(ctx context.Context, platformID, webhookID uint) (bool, error) {
	var n int64
	err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM webhooks WHERE id = ? AND external_platform_id = ? AND deleted_at IS NULL`,
		webhookID, platformID).Scan(&n).Error
	return n == 1, err
}

func (s *PGStore) MerchantName(ctx context.Context, platformID uint) (string, error) {
	var names []string
	err := s.db.WithContext(ctx).Raw(`SELECT name FROM external_platforms WHERE id = ? AND deleted_at IS NULL`, platformID).Scan(&names).Error
	if err != nil || len(names) == 0 {
		return "", err
	}
	return names[0], nil
}

func toPayment(r paymentRow, answers []answerRow) (LinkPayment, error) {
	p := LinkPayment{
		ID: r.ID, LinkID: r.LinkID, IdempotencyKey: r.IdempotencyKey, RequestHash: r.RequestHash, Status: r.Status,
		Environment: Environment(r.Environment), Method: MethodSpec{Method: fees.Method(r.Method), Chain: r.Chain, Asset: r.Asset},
		Connector: r.Connector, Amount: r.Amount, Currency: r.Currency, Fee: fromNull(r.Fee), Tax: fromNull(r.Tax),
		CustomerTotal: r.CustomerTotal, FeeBearer: fees.FeeBearer(r.FeeBearer), FeeRuleID: r.FeeRuleID, FeeRuleVersion: r.FeeRuleVersion,
		CustomerName: r.CustomerName, CustomerEmail: r.CustomerEmail, CustomerPhone: r.CustomerPhone, CreatedAt: r.CreatedAt,
		ClientKey: r.ClientKey, ReservedUntil: r.ReservedUntil.UTC(), OpenUntil: r.OpenUntil.UTC(),
	}
	if r.PaymentReference != nil {
		p.PaymentReference = *r.PaymentReference
	}
	for _, j := range []struct {
		raw *string
		dst any
	}{{r.BillingAddress, &p.BillingAddress}, {r.ShippingAddress, &p.ShippingAddress}, {r.ProcessorResponse, &p.Processor}} {
		if j.raw == nil {
			continue
		}
		if err := json.Unmarshal([]byte(*j.raw), j.dst); err != nil {
			return LinkPayment{}, fmt.Errorf("links: decode payment %s: %w", r.ID, err)
		}
	}
	for _, a := range answers {
		p.Answers = append(p.Answers, Answer{QuestionKey: a.QuestionKey, QuestionLabel: a.QuestionLabel, Value: a.Value})
	}
	return p, nil
}

func findPayment(tx *gorm.DB, linkID, key string) (*LinkPayment, error) {
	var rows []paymentRow
	if err := tx.Where("link_id = ? AND idempotency_key = ? AND status <> 'released'", linkID, key).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("links: find payment: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	var answers []answerRow
	if err := tx.Where("link_payment_id = ?", rows[0].ID).Order("id").Find(&answers).Error; err != nil {
		return nil, fmt.Errorf("links: load answers: %w", err)
	}
	p, err := toPayment(rows[0], answers)
	return &p, err
}

func (s *PGStore) FindPayment(ctx context.Context, linkID, key string) (*LinkPayment, error) {
	return findPayment(s.db.WithContext(ctx), linkID, key)
}

// Reserve locks the link row, so concurrent payers on one link serialise and the use limit holds exactly.
func (s *PGStore) Reserve(ctx context.Context, p LinkPayment, now time.Time, limits ReserveLimits) (LinkPayment, *LinkPayment, error) {
	var existing *LinkPayment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []linkRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", p.LinkID).Find(&rows).Error; err != nil {
			return fmt.Errorf("links: lock link: %w", err)
		}
		if len(rows) == 0 {
			return ErrStoreNotFound
		}
		var err error
		if existing, err = findPayment(tx, p.LinkID, p.IdempotencyKey); err != nil || existing != nil {
			return err
		}
		l, err := fromRow(rows[0], nil, nil)
		if err != nil {
			return err
		}
		if err := availability(l, now); err != nil {
			return err
		}
		if l.FeeBearer != p.FeeBearer {
			return ErrStale
		}
		if l.MultiUse {
			var counts struct{ Open, Mine int }
			if err := tx.Raw(`SELECT count(*) AS open, count(*) FILTER (WHERE client_key = ?) AS mine
				FROM payment_link_payments WHERE link_id = ? AND status <> 'released' AND open_until > ?`, p.ClientKey, p.LinkID, now).Scan(&counts).Error; err != nil {
				return fmt.Errorf("links: count open payments: %w", err)
			}
			if err := openPaymentsError(l, limits, counts.Open, counts.Mine); err != nil {
				return err
			}
		}
		p.ID, p.Status = uuid.NewString(), paymentPending
		row := paymentRow{
			ID: p.ID, LinkID: p.LinkID, IdempotencyKey: p.IdempotencyKey, RequestHash: p.RequestHash, Status: p.Status,
			Environment: string(p.Environment), Method: string(p.Method.Method), Chain: p.Method.Chain, Asset: p.Method.Asset,
			Connector: p.Connector, Amount: p.Amount, Currency: p.Currency, Fee: nullDec(p.Fee), Tax: nullDec(p.Tax),
			CustomerTotal: p.CustomerTotal, FeeBearer: string(p.FeeBearer), FeeRuleID: p.FeeRuleID, FeeRuleVersion: p.FeeRuleVersion,
			CustomerName: p.CustomerName, CustomerEmail: p.CustomerEmail, CustomerPhone: p.CustomerPhone,
			ClientKey: p.ClientKey, ReservedUntil: p.ReservedUntil, OpenUntil: p.OpenUntil,
			CreatedAt: now, UpdatedAt: now,
		}
		if row.BillingAddress, err = optJSON(p.BillingAddress, p.BillingAddress != nil); err != nil {
			return err
		}
		if row.ShippingAddress, err = optJSON(p.ShippingAddress, p.ShippingAddress != nil); err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("links: insert reservation: %w", err)
		}
		for _, a := range p.Answers {
			ar := answerRow{LinkPaymentID: p.ID, LinkID: p.LinkID, QuestionKey: a.QuestionKey, QuestionLabel: a.QuestionLabel, Value: a.Value}
			if err := tx.Create(&ar).Error; err != nil {
				return fmt.Errorf("links: insert answer: %w", err)
			}
		}
		if err := tx.Exec(`UPDATE payment_links SET uses_count = uses_count + 1 WHERE id = ?`, p.LinkID).Error; err != nil {
			return fmt.Errorf("links: take a use: %w", err)
		}
		p.CreatedAt = now
		return nil
	})
	if err != nil || existing != nil {
		return LinkPayment{}, existing, err
	}
	return p, nil, nil
}

func (s *PGStore) Complete(ctx context.Context, id string, created CreatedPayment, openUntil time.Time) error {
	raw, err := toJSON(created)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Exec(`UPDATE payment_link_payments SET status = 'created', payment_reference = ?,
		processor_response = ?, open_until = ?, updated_at = now() WHERE id = ? AND status = 'pending'`, created.Reference, raw, openUntil, id)
	if res.Error != nil {
		return fmt.Errorf("links: complete reservation: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrStale
	}
	return nil
}

func (s *PGStore) Release(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var linkIDs []string
		if err := tx.Raw(`UPDATE payment_link_payments SET status = 'released', updated_at = now()
			WHERE id = ? AND status = 'pending' RETURNING link_id`, id).Scan(&linkIDs).Error; err != nil {
			return fmt.Errorf("links: release reservation: %w", err)
		}
		if len(linkIDs) == 0 {
			return ErrStale
		}
		return tx.Exec(`UPDATE payment_links SET uses_count = uses_count - 1 WHERE id = ?`, linkIDs[0]).Error
	})
}

func (s *PGStore) ExpiredPending(ctx context.Context, now time.Time, limit int) ([]LinkPayment, error) {
	var rows []paymentRow
	if err := s.db.WithContext(ctx).Where("status = 'pending' AND reserved_until < ?", now).
		Order("reserved_until").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("links: expired reservations: %w", err)
	}
	out := make([]LinkPayment, len(rows))
	for i, r := range rows {
		p, err := toPayment(r, nil)
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

func (s *PGStore) Link(ctx context.Context, id string) (Link, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Link{}, ErrStoreNotFound
	}
	return s.loadOne(s.db.WithContext(ctx), "id = ?", id)
}

func (s *PGStore) OpenCreated(ctx context.Context, linkID string, now time.Time, limit int) ([]LinkPayment, error) {
	var rows []paymentRow
	if err := s.db.WithContext(ctx).Where("link_id = ? AND status = 'created' AND open_until > ?", linkID, now).
		Order("created_at").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("links: open uses: %w", err)
	}
	out := make([]LinkPayment, len(rows))
	for i, r := range rows {
		p, err := toPayment(r, nil)
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

func (s *PGStore) CloseUses(ctx context.Context, ids []string, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Exec(`UPDATE payment_link_payments SET open_until = ?, updated_at = now()
		WHERE id IN ? AND status = 'created' AND open_until > ?`, now, ids, now).Error
}
