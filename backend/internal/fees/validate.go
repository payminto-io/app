package fees

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Policy is configuration that constrains rules and previews (FEES_* keys, README "Configuration").
type Policy struct {
	SurchargeForbidden map[Method]bool
}

// DefaultPolicy forbids surcharging UPI, which NPCI does not permit.
func DefaultPolicy() Policy {
	return Policy{SurchargeForbidden: map[Method]bool{MethodUPI: true}}
}

// ParsePolicy reads FEES_SURCHARGE_FORBIDDEN_METHODS: a comma list of methods, "none", or empty for the default.
func ParsePolicy(forbidden string) (Policy, error) {
	forbidden = strings.TrimSpace(forbidden)
	if forbidden == "" {
		return DefaultPolicy(), nil
	}
	p := Policy{SurchargeForbidden: map[Method]bool{}}
	if forbidden == "none" {
		return p, nil
	}
	for _, raw := range strings.Split(forbidden, ",") {
		m := Method(strings.ToLower(strings.TrimSpace(raw)))
		if !slices.Contains(Methods, m) {
			return Policy{}, fmt.Errorf("fees: FEES_SURCHARGE_FORBIDDEN_METHODS: unknown method %q", raw)
		}
		p.SurchargeForbidden[m] = true
	}
	return p, nil
}

func (p Policy) checkBearer(m Method, b FeeBearer) error {
	if b == BearerCustomer && p.SurchargeForbidden[m] {
		return fmt.Errorf("%w: %s", ErrSurchargeForbidden, m)
	}
	return nil
}

var (
	connectorPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	regionPattern    = regexp.MustCompile(`^[A-Z]{2,8}$`)
	hundred          = decimal.NewFromInt(100)
)

func normalized(s *string, f func(string) string) *string {
	if s == nil {
		return nil
	}
	v := f(strings.TrimSpace(*s))
	return &v
}

func (q Query) normalize() Query {
	q.Method = Method(strings.ToLower(strings.TrimSpace(string(q.Method))))
	q.Connector = strings.ToLower(strings.TrimSpace(q.Connector))
	q.CardType = CardType(strings.ToLower(strings.TrimSpace(string(q.CardType))))
	q.Region = strings.ToUpper(strings.TrimSpace(q.Region))
	q.Currency = strings.ToUpper(strings.TrimSpace(q.Currency))
	return q
}

func validateScope(s Scope) (Scope, error) {
	s.Method = Method(strings.ToLower(strings.TrimSpace(string(s.Method))))
	s.Currency = strings.ToUpper(strings.TrimSpace(s.Currency))
	s.Connector = normalized(s.Connector, strings.ToLower)
	s.Region = normalized(s.Region, strings.ToUpper)
	if s.CardType != nil {
		c := CardType(strings.ToLower(strings.TrimSpace(string(*s.CardType))))
		s.CardType = &c
	}
	if !slices.Contains(Methods, s.Method) {
		return s, invalid("method", "must be one of %v", Methods)
	}
	if _, ok := MinorUnits(s.Currency); !ok {
		return s, invalid("currency", "unknown currency %q", s.Currency)
	}
	if s.Connector != nil && !connectorPattern.MatchString(*s.Connector) {
		return s, invalid("connector", "must match %s", connectorPattern)
	}
	if s.CardType != nil {
		switch {
		case s.Connector == nil:
			return s, invalid("card_type", "requires connector")
		case s.Method != MethodCard:
			return s, invalid("card_type", "only applies to method card")
		case !slices.Contains(CardTypes, *s.CardType):
			return s, invalid("card_type", "must be one of %v", CardTypes)
		}
	}
	if s.Region != nil {
		if s.CardType == nil {
			return s, invalid("region", "requires connector and card_type")
		}
		if !regionPattern.MatchString(*s.Region) {
			return s, invalid("region", "must match %s", regionPattern)
		}
	}
	return s, nil
}

func checkPercent(field string, v decimal.Decimal) error {
	if v.IsNegative() || v.GreaterThan(hundred) {
		return invalid(field, "must be between 0 and 100")
	}
	return nil
}

func validateSlabs(slabs []Slab) error {
	for i, s := range slabs {
		if err := checkPercent("slabs", s.Percent); err != nil {
			return invalid("slabs", "slab %d percent must be between 0 and 100", i)
		}
		if s.Flat.IsNegative() {
			return invalid("slabs", "slab %d flat must not be negative", i)
		}
		last := i == len(slabs)-1
		if s.UpTo == nil {
			if !last {
				return invalid("slabs", "only the last slab may be open-ended")
			}
			continue
		}
		if last {
			return invalid("slabs", "the last slab must be open-ended (up_to null)")
		}
		if !s.UpTo.IsPositive() {
			return invalid("slabs", "slab %d up_to must be positive", i)
		}
		if i > 0 && !s.UpTo.GreaterThan(*slabs[i-1].UpTo) {
			return invalid("slabs", "up_to must strictly increase")
		}
	}
	return nil
}

// validatePricing checks p for scope s and defaults EffectiveFrom to now; backdating is refused.
func validatePricing(p Pricing, s Scope, policy Policy, now time.Time) (Pricing, error) {
	places, _ := MinorUnits(s.Currency)
	if err := checkPercent("percent", p.Percent); err != nil {
		return p, err
	}
	if p.Flat.IsNegative() {
		return p, invalid("flat", "must not be negative")
	}
	if len(p.Slabs) > 0 {
		if !p.Percent.IsZero() || !p.Flat.IsZero() {
			return p, invalid("slabs", "percent and flat must be zero when slabs are set")
		}
		if err := validateSlabs(p.Slabs); err != nil {
			return p, err
		}
	}
	for _, f := range []struct {
		name string
		v    *decimal.Decimal
	}{{"min_fee", p.MinFee}, {"max_fee", p.MaxFee}} {
		if f.v == nil {
			continue
		}
		if f.v.IsNegative() {
			return p, invalid(f.name, "must not be negative")
		}
		if !f.v.Equal(f.v.Round(places)) {
			return p, invalid(f.name, "has more than %d decimal places for %s", places, s.Currency)
		}
	}
	if p.MinFee != nil && p.MaxFee != nil && p.MinFee.GreaterThan(*p.MaxFee) {
		return p, invalid("min_fee", "must not exceed max_fee")
	}
	if p.Taxable {
		if !p.TaxPercent.IsPositive() || p.TaxPercent.GreaterThan(hundred) {
			return p, invalid("tax_percent", "must be above 0 and at most 100 when taxable")
		}
	} else if !p.TaxPercent.IsZero() {
		return p, invalid("tax_percent", "must be 0 when not taxable")
	}
	if p.FeeBearer != BearerMerchant && p.FeeBearer != BearerCustomer {
		return p, invalid("fee_bearer", "must be merchant or customer")
	}
	if err := policy.checkBearer(s.Method, p.FeeBearer); err != nil {
		return p, err
	}
	now = now.UTC().Truncate(time.Microsecond)
	if p.EffectiveFrom == nil {
		p.EffectiveFrom = &now
	} else {
		from := p.EffectiveFrom.UTC().Truncate(time.Microsecond)
		if from.Before(now) {
			return p, invalid("effective_from", "must not be in the past")
		}
		p.EffectiveFrom = &from
	}
	if p.EffectiveTo != nil {
		to := p.EffectiveTo.UTC().Truncate(time.Microsecond)
		if !to.After(*p.EffectiveFrom) {
			return p, invalid("effective_to", "must be after effective_from")
		}
		p.EffectiveTo = &to
	}
	return p, nil
}

func validateInput(in RuleInput, policy Policy, now time.Time) (RuleInput, error) {
	scope, err := validateScope(in.Scope)
	if err != nil {
		return in, err
	}
	pricing, err := validatePricing(in.Pricing, scope, policy, now)
	if err != nil {
		return in, err
	}
	return RuleInput{Scope: scope, Pricing: pricing}, nil
}

// preview validates req, resolves among candidates and computes; the bearer override is policy-checked.
func preview(candidates []Rule, req PreviewRequest, policy Policy) (Breakdown, error) {
	req.Query = req.Query.normalize()
	if !slices.Contains(Methods, req.Method) {
		return Breakdown{}, invalid("method", "must be one of %v", Methods)
	}
	if req.CardType != "" && !slices.Contains(CardTypes, req.CardType) {
		return Breakdown{}, invalid("card_type", "must be one of %v", CardTypes)
	}
	places, ok := MinorUnits(req.Currency)
	if !ok {
		return Breakdown{}, invalid("currency", "unknown currency %q", req.Currency)
	}
	if !req.Amount.IsPositive() {
		return Breakdown{}, invalid("amount", "must be positive")
	}
	if !req.Amount.Equal(req.Amount.Round(places)) {
		return Breakdown{}, invalid("amount", "has more than %d decimal places for %s", places, req.Currency)
	}
	if req.FeeBearer != nil && *req.FeeBearer != BearerMerchant && *req.FeeBearer != BearerCustomer {
		return Breakdown{}, invalid("fee_bearer", "must be merchant or customer")
	}
	r, err := resolve(candidates, req.Query)
	if err != nil {
		return Breakdown{}, err
	}
	if req.FeeBearer != nil {
		r.FeeBearer = *req.FeeBearer
	}
	if err := policy.checkBearer(r.Method, r.FeeBearer); err != nil {
		return Breakdown{}, err
	}
	return Compute(r, req.Amount), nil
}
