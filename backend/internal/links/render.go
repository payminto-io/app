package links

import (
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// RenderModel is everything the hosted checkout needs and nothing else: no ids beyond the short code,
// no metadata, reference, webhook, settlement or hidden prefills.
type RenderModel struct {
	ShortCode          string           `json:"short_code"`
	URL                string           `json:"url"`
	Available          bool             `json:"available"`
	UnavailableReason  *string          `json:"unavailable_reason"`
	MerchantName       *string          `json:"merchant_name"`
	Title              string           `json:"title"`
	Description        *string          `json:"description"`
	AmountMode         AmountMode       `json:"amount_mode"`
	Amount             *decimal.Decimal `json:"amount"`
	AmountMin          *decimal.Decimal `json:"amount_min"`
	AmountMax          *decimal.Decimal `json:"amount_max"`
	Currency           string           `json:"currency"`
	LineItems          []RenderLineItem `json:"line_items"`
	Subtotal           *decimal.Decimal `json:"subtotal"`
	TaxTotal           *decimal.Decimal `json:"tax_total"`
	CustomerFields     RenderCustomer   `json:"customer_fields"`
	BillingRequired    bool             `json:"billing_required"`
	ShippingRequired   bool             `json:"shipping_required"`
	Questions          []RenderQuestion `json:"questions"`
	Methods            []RenderMethod   `json:"methods"`
	FeeBearer          fees.FeeBearer   `json:"fee_bearer"`
	ChainToleranceBps  int              `json:"chain_tolerance_bps"`
	QuoteExpirySeconds int              `json:"quote_expiry_seconds"`
	SuccessMode        string           `json:"success_mode"`
	SuccessMessage     *string          `json:"success_message"`
	FailureRetry       bool             `json:"failure_retry"`
	FailureMessage     *string          `json:"failure_message"`
	ReceiptEmail       bool             `json:"receipt_email"`
	ExpiresAt          *time.Time       `json:"expires_at"`
	Branding           RenderBranding   `json:"branding"`
}

type RenderLineItem struct {
	Name      string          `json:"name"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
	TaxRate   decimal.Decimal `json:"tax_rate"`
	Subtotal  decimal.Decimal `json:"subtotal"`
	Tax       decimal.Decimal `json:"tax"`
	Total     decimal.Decimal `json:"total"`
}

// RenderField omits the prefill of a hidden field: it is the merchant's data, not the payer's to see.
type RenderField struct {
	Mode    FieldMode `json:"mode"`
	Prefill *string   `json:"prefill"`
}

type RenderCustomer struct {
	Name  RenderField `json:"name"`
	Email RenderField `json:"email"`
	Phone RenderField `json:"phone"`
}

type RenderQuestion struct {
	Key      string       `json:"key"`
	Label    string       `json:"label"`
	Type     QuestionType `json:"type"`
	Options  []string     `json:"options"`
	Required bool         `json:"required"`
	PerOrder bool         `json:"per_order"`
}

// RenderMethod carries a surcharge breakdown only when it is computable now (known amount, same currency); else nulls.
type RenderMethod struct {
	Method        fees.Method      `json:"method"`
	Chain         *string          `json:"chain"`
	Asset         *string          `json:"asset"`
	Fee           *decimal.Decimal `json:"fee"`
	Tax           *decimal.Decimal `json:"tax"`
	CustomerTotal *decimal.Decimal `json:"customer_total"`
}

type RenderBranding struct {
	LogoURL     *string `json:"logo_url"`
	AccentColor *string `json:"accent_color"`
	Language    string  `json:"language"`
}

const (
	ReasonPaused          = "paused"
	ReasonExpired         = "expired"
	ReasonUseLimitReached = "use_limit_reached"
	ReasonNoMethods       = "no_methods_available"
)

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func renderField(r FieldRule) RenderField {
	f := RenderField{Mode: r.Mode}
	if r.Mode != FieldHidden {
		f.Prefill = strOrNil(r.Prefill)
	}
	return f
}

func unavailableReason(err error) *string {
	reason := ""
	switch CodeOf(err) {
	case CodePaused:
		reason = ReasonPaused
	case CodeExpired:
		reason = ReasonExpired
	case CodeUseLimitReached:
		reason = ReasonUseLimitReached
	default:
		return nil
	}
	return &reason
}

func (s *Service) renderModel(l Link, merchant string, methods []RenderMethod, availErr error) RenderModel {
	places, _ := s.precision.MinorUnits(l.Currency)
	m := RenderModel{
		ShortCode: l.ShortCode, URL: s.URL(l.ShortCode),
		MerchantName: strOrNil(merchant), Title: l.Title, Description: strOrNil(l.Description),
		AmountMode: l.AmountMode, Amount: l.Total, AmountMin: l.AmountMin, AmountMax: l.AmountMax, Currency: l.Currency,
		LineItems: []RenderLineItem{},
		CustomerFields: RenderCustomer{
			Name: renderField(l.CustomerFields.Name), Email: renderField(l.CustomerFields.Email), Phone: renderField(l.CustomerFields.Phone),
		},
		BillingRequired: l.BillingRequired, ShippingRequired: l.ShippingRequired,
		Questions: make([]RenderQuestion, len(l.Questions)), Methods: methods, FeeBearer: l.FeeBearer,
		ChainToleranceBps: l.ChainToleranceBps, QuoteExpirySeconds: l.QuoteExpirySeconds,
		SuccessMode: l.SuccessMode, SuccessMessage: strOrNil(l.SuccessMessage),
		FailureRetry: l.FailureRetry, FailureMessage: strOrNil(l.FailureMessage), ReceiptEmail: l.ReceiptEmail,
		ExpiresAt: l.ExpiresAt,
		Branding:  RenderBranding{LogoURL: strOrNil(l.LogoURL), AccentColor: strOrNil(l.AccentColor), Language: l.Language},
	}
	if l.AmountMode == AmountLineItems {
		t := itemsTotals(l.LineItems, places)
		for i, li := range l.LineItems {
			m.LineItems = append(m.LineItems, RenderLineItem{
				Name: li.Name, Quantity: li.Quantity, UnitPrice: li.UnitPrice, TaxRate: li.TaxRate,
				Subtotal: t.Lines[i].Subtotal, Tax: t.Lines[i].Tax, Total: t.Lines[i].Total,
			})
		}
		m.Subtotal, m.TaxTotal = &t.Subtotal, &t.Tax
	}
	for i, q := range l.Questions {
		opts := q.Options
		if opts == nil {
			opts = []string{}
		}
		m.Questions[i] = RenderQuestion{Key: q.Key, Label: q.Label, Type: q.Type, Options: opts, Required: q.Required, PerOrder: q.PerOrder}
	}
	m.UnavailableReason = unavailableReason(availErr)
	if m.UnavailableReason == nil && len(methods) == 0 {
		r := ReasonNoMethods
		m.UnavailableReason = &r
	}
	m.Available = m.UnavailableReason == nil
	return m
}
