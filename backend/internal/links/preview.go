package links

import (
	"context"
	"log/slog"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// MethodPreview is what the merchant sees per method before publishing: the connector and rule version that
// will price it and, when the amount and currency allow, the breakdown. Unavailable carries the refusal code.
type MethodPreview struct {
	Method        fees.Method      `json:"method"`
	Chain         *string          `json:"chain"`
	Asset         *string          `json:"asset"`
	Connector     *string          `json:"connector"`
	RuleID        *uint            `json:"rule_id"`
	RuleVersion   *int             `json:"rule_version"`
	FeeBearer     fees.FeeBearer   `json:"fee_bearer"`
	FeeCurrency   string           `json:"fee_currency"`
	Amount        *decimal.Decimal `json:"amount"`
	Fee           *decimal.Decimal `json:"fee"`
	Tax           *decimal.Decimal `json:"tax"`
	CustomerTotal *decimal.Decimal `json:"customer_total"`
	MerchantNet   *decimal.Decimal `json:"merchant_net"`
	Unavailable   *Code            `json:"unavailable"`
}

// FeePreview prices each method on the link's real amount (fixed, line-item total or customer minimum).
func (s *Service) FeePreview(ctx context.Context, l Link) []MethodPreview {
	out := make([]MethodPreview, 0, len(l.Methods))
	if l.Currency == "" {
		return out
	}
	amount, real := s.representative(l)
	for _, m := range l.Methods {
		mp := MethodPreview{Method: m.Method, Chain: strOrNil(m.Chain), Asset: strOrNil(m.Asset), FeeBearer: l.FeeBearer, FeeCurrency: m.FeeCurrency(l.Currency)}
		p, e, err := s.price(ctx, l, m, amount, real)
		switch {
		case err != nil:
			slog.Error("links: fee preview", "link_id", l.ID, "method", m.String(), "error", err)
			code := CodeMethodUnavailable
			mp.Unavailable = &code
		case e != nil:
			code := e.Code
			mp.Unavailable = &code
		default:
			conn, id, ver := p.connector, p.rule.ID, p.rule.Version
			mp.Connector, mp.RuleID, mp.RuleVersion = &conn, &id, &ver
			if b := p.breakdown; b != nil {
				mp.Amount, mp.Fee, mp.Tax, mp.CustomerTotal, mp.MerchantNet = &b.Amount, &b.Fee, &b.Tax, &b.CustomerTotal, &b.MerchantNet
			}
		}
		out = append(out, mp)
	}
	return out
}

// DroppedMethod is a method the checkout would leave out, with the refusal that removed it.
type DroppedMethod struct {
	Method  fees.Method `json:"method"`
	Chain   *string     `json:"chain"`
	Asset   *string     `json:"asset"`
	Code    Code        `json:"code"`
	Message string      `json:"message"`
}

// PreviewResult is the checkout's render model for an unsaved form, plus the methods it dropped and why.
type PreviewResult struct {
	Model   RenderModel     `json:"model"`
	Dropped []DroppedMethod `json:"dropped_methods"`
}

// Preview renders in exactly as the public endpoint would once published, without storing anything.
// With linkID it takes status, uses and short code from that link, so a paused or exhausted link previews as such.
func (s *Service) Preview(ctx context.Context, platformID uint, in Input, linkID string) (PreviewResult, error) {
	in, total, err := s.prepare(ctx, platformID, in)
	if err != nil {
		return PreviewResult{}, err
	}
	l := Link{Status: StatusDraft, Environment: s.env, ExternalPlatformID: platformID}
	if linkID != "" {
		if l, err = s.Get(ctx, platformID, linkID); err != nil {
			return PreviewResult{}, err
		}
	}
	l.Input, l.Total = in, total
	m, dropped, err := s.render(ctx, l)
	if err != nil {
		return PreviewResult{}, err
	}
	return PreviewResult{Model: m, Dropped: dropped}, nil
}

// MerchantName is the platform's display name, as checkout shows it.
func (s *Service) MerchantName(ctx context.Context, platformID uint) (string, error) {
	return s.store.MerchantName(ctx, platformID)
}
