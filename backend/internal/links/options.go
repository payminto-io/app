package links

import (
	"context"
	"slices"
	"strings"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// Offering is one method a PaymentCreator can take in a currency.
type Offering struct {
	Currency string
	Method   MethodSpec
}

// Catalog is implemented by a PaymentCreator that can list what it takes; without it Options is empty.
type Catalog interface {
	Offerings(ctx context.Context, env Environment) ([]Offering, error)
}

// OptionMethod is one enabled method and the link currencies it can be offered in.
type OptionMethod struct {
	Method     fees.Method `json:"method"`
	Chain      *string     `json:"chain"`
	Asset      *string     `json:"asset"`
	Currencies []string    `json:"currencies"`
}

// Options is what the form may offer in the process environment: each offering with a connector and an active fee rule.
type Options struct {
	Environment Environment    `json:"environment"`
	Currencies  []string       `json:"currencies"`
	Methods     []OptionMethod `json:"methods"`
}

// Options lists the methods and currencies a merchant can publish with here, using the publish path's own checks.
func (s *Service) Options(ctx context.Context) (Options, error) {
	out := Options{Environment: s.env, Currencies: []string{}, Methods: []OptionMethod{}}
	cat, ok := s.creator.(Catalog)
	if !ok {
		return out, nil
	}
	offers, err := cat.Offerings(ctx, s.env)
	if err != nil {
		return out, err
	}
	byKey := map[string]int{}
	for _, o := range offers {
		cur := strings.ToUpper(strings.TrimSpace(o.Currency))
		l := Link{Input: Input{Currency: cur, FeeBearer: fees.BearerMerchant}, Environment: s.env}
		_, e, err := s.price(ctx, l, o.Method, decimal.New(1, 0), false)
		if err != nil {
			return out, err
		}
		if e != nil {
			continue
		}
		if !slices.Contains(out.Currencies, cur) {
			out.Currencies = append(out.Currencies, cur)
		}
		key := o.Method.String()
		i, seen := byKey[key]
		if !seen {
			i = len(out.Methods)
			byKey[key] = i
			out.Methods = append(out.Methods, OptionMethod{Method: o.Method.Method, Chain: strOrNil(o.Method.Chain), Asset: strOrNil(o.Method.Asset), Currencies: []string{}})
		}
		if !slices.Contains(out.Methods[i].Currencies, cur) {
			out.Methods[i].Currencies = append(out.Methods[i].Currencies, cur)
		}
	}
	slices.Sort(out.Currencies)
	return out, nil
}
