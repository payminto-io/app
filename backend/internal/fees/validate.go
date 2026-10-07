package fees

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Policy is configuration that constrains rules and previews (config.FeesConfig, README "Configuration").
type Policy struct {
	SurchargeForbidden map[Method]bool
	Precision          Precision
}

// DefaultPolicy forbids surcharging UPI, which NPCI does not permit.
func DefaultPolicy() Policy {
	return Policy{SurchargeForbidden: map[Method]bool{MethodUPI: true}, Precision: DefaultPrecision()}
}

// ParsePolicy reads FEES_SURCHARGE_FORBIDDEN_METHODS (a comma list, "none", or empty for the default) and FEES_ASSET_PRECISION.
func ParsePolicy(forbidden, precision string) (Policy, error) {
	p := DefaultPolicy()
	var err error
	if p.Precision, err = ParsePrecision(precision); err != nil {
		return Policy{}, err
	}
	forbidden = strings.TrimSpace(forbidden)
	if forbidden == "" {
		return p, nil
	}
	p.SurchargeForbidden = map[Method]bool{}
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

const (
	// percentPlaces matches numeric(9,6) on percent, tax_percent and slab percent.
	percentPlaces = 6
	// maxExponent bounds a decimal's exponent before any arithmetic; numeric(38,18) never needs more.
	maxExponent = 40
	maxDigits   = 40
)

var (
	connectorPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	regionPattern    = regexp.MustCompile(`^[A-Z]{2,8}$`)
	hundred          = decimal.NewFromInt(100)
	// maxMagnitude is the numeric(38,18) integer range.
	maxMagnitude = decimal.New(1, 20)
)

// checkDecimal rejects values outside numeric(38,18) or finer than places; the cheap exponent test runs first
// because rescaling a value like 1e100000000 costs seconds and gigabytes.
func checkDecimal(field string, v decimal.Decimal, places int32) error {
	if exp := v.Exponent(); exp < -maxExponent || exp > maxExponent {
		return invalid(field, "out of range")
	}
	if v.NumDigits() > maxDigits {
		return invalid(field, "out of range")
	}
	if v.Abs().GreaterThanOrEqual(maxMagnitude) {
		return invalid(field, "must be below 1e20")
	}
	if !v.Equal(v.Truncate(places)) {
		return invalid(field, "has more than %d decimal places", places)
	}
	return nil
}

func checkPercent(field string, v decimal.Decimal) error {
	if err := checkDecimal(field, v, percentPlaces); err != nil {
		return err
	}
	if v.IsNegative() || v.GreaterThan(hundred) {
		return invalid(field, "must be between 0 and 100")
	}
	return nil
}

func checkMoney(field string, v decimal.Decimal, places int32) error {
	if err := checkDecimal(field, v, places); err != nil {
		return err
	}
	if v.IsNegative() {
		return invalid(field, "must not be negative")
	}
	return nil
}

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
	q.Chain = strings.ToUpper(strings.TrimSpace(q.Chain))
	return q
}

// checkCurrency returns the minor units and refuses a currency whose class does not fit the method.
func checkCurrency(p Precision, m Method, currency string) (int32, error) {
	places, ok := p.MinorUnits(currency)
	if !ok {
		return 0, invalid("currency", "unknown currency %q (ISO 4217, or an asset in FEES_ASSET_PRECISION)", currency)
	}
	if fiat := p.IsFiat(currency); (m == MethodCrypto) == fiat {
		if fiat {
			return 0, invalid("currency", "method crypto needs an on-chain asset, not %s", currency)
		}
		return 0, invalid("currency", "method %s needs a fiat currency, not %s", m, currency)
	}
	return places, nil
}

func validateScope(s Scope, p Precision) (Scope, int32, error) {
	s.Method = Method(strings.ToLower(strings.TrimSpace(string(s.Method))))
	s.Currency = strings.ToUpper(strings.TrimSpace(s.Currency))
	s.Connector = normalized(s.Connector, strings.ToLower)
	s.Region = normalized(s.Region, strings.ToUpper)
	if s.CardType != nil {
		c := CardType(strings.ToLower(strings.TrimSpace(string(*s.CardType))))
		s.CardType = &c
	}
	if !slices.Contains(Methods, s.Method) {
		return s, 0, invalid("method", "must be one of %v", Methods)
	}
	places, err := checkCurrency(p, s.Method, s.Currency)
	if err != nil {
		return s, 0, err
	}
	if s.Connector != nil && !connectorPattern.MatchString(*s.Connector) {
		return s, 0, invalid("connector", "must match %s", connectorPattern)
	}
	if s.CardType != nil {
		switch {
		case s.Connector == nil:
			return s, 0, invalid("card_type", "requires connector")
		case s.Method != MethodCard:
			return s, 0, invalid("card_type", "only applies to method card")
		case !slices.Contains(CardTypes, *s.CardType):
			return s, 0, invalid("card_type", "must be one of %v", CardTypes)
		}
	}
	if s.Region != nil {
		if s.CardType == nil {
			return s, 0, invalid("region", "requires connector and card_type")
		}
		if !regionPattern.MatchString(*s.Region) {
			return s, 0, invalid("region", "must match %s", regionPattern)
		}
	}
	return s, places, nil
}

func validateSlabs(slabs []Slab, places int32) error {
	for i, s := range slabs {
		if err := checkPercent("slabs", s.Percent); err != nil {
			return invalid("slabs", "slab %d percent: %s", i, err.(*ValidationError).Reason)
		}
		if err := checkMoney("slabs", s.Flat, places); err != nil {
			return invalid("slabs", "slab %d flat: %s", i, err.(*ValidationError).Reason)
		}
		last := i == len(slabs)-1
		if s.UpTo == nil {
			if !last {
				return invalid("slabs", "only the last slab may be open-ended")
			}
			continue
		}
		if err := checkMoney("slabs", *s.UpTo, places); err != nil {
			return invalid("slabs", "slab %d up_to: %s", i, err.(*ValidationError).Reason)
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

// validatePricing checks p for scope s at the scope's minor units and defaults EffectiveFrom to now; backdating is refused.
func validatePricing(p Pricing, s Scope, places int32, policy Policy, now time.Time) (Pricing, error) {
	if err := checkPercent("percent", p.Percent); err != nil {
		return p, err
	}
	if err := checkMoney("flat", p.Flat, places); err != nil {
		return p, err
	}
	if len(p.Slabs) > 0 {
		if !p.Percent.IsZero() || !p.Flat.IsZero() {
			return p, invalid("slabs", "percent and flat must be zero when slabs are set")
		}
		if err := validateSlabs(p.Slabs, places); err != nil {
			return p, err
		}
	}
	if p.MinFee != nil {
		if err := checkMoney("min_fee", *p.MinFee, places); err != nil {
			return p, err
		}
	}
	if p.MaxFee != nil {
		if err := checkMoney("max_fee", *p.MaxFee, places); err != nil {
			return p, err
		}
	}
	if p.MinFee != nil && p.MaxFee != nil && p.MinFee.GreaterThan(*p.MaxFee) {
		return p, invalid("min_fee", "must not exceed max_fee")
	}
	if err := checkPercent("tax_percent", p.TaxPercent); err != nil {
		return p, err
	}
	if p.Taxable && !p.TaxPercent.IsPositive() {
		return p, invalid("tax_percent", "must be above 0 when taxable")
	}
	if !p.Taxable && !p.TaxPercent.IsZero() {
		return p, invalid("tax_percent", "must be 0 when not taxable")
	}
	if p.FeeBearer != BearerMerchant && p.FeeBearer != BearerCustomer {
		return p, invalid("fee_bearer", "must be merchant or customer")
	}
	if err := policy.checkBearer(s.Method, p.FeeBearer); err != nil {
		return p, err
	}
	if err := checkInvertible(p, places); err != nil {
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

// validated is a rule input that passed validation, with the precision it will be stored at.
type validated struct {
	RuleInput
	minorUnits int32
}

func validateInput(in RuleInput, policy Policy, now time.Time) (validated, error) {
	scope, places, err := validateScope(in.Scope, policy.Precision)
	if err != nil {
		return validated{}, err
	}
	pricing, err := validatePricing(in.Pricing, scope, places, policy, now)
	if err != nil {
		return validated{}, err
	}
	return validated{RuleInput: RuleInput{Scope: scope, Pricing: pricing}, minorUnits: places}, nil
}

// checkAmount bounds an amount before any arithmetic, then requires it positive and on the currency's grid.
func checkAmount(v decimal.Decimal, places int32, currency string) error {
	if err := checkDecimal("amount", v, places); err != nil {
		return invalid("amount", "%s for %s", err.(*ValidationError).Reason, currency)
	}
	if !v.IsPositive() {
		return invalid("amount", "must be positive")
	}
	return nil
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
	places, err := checkCurrency(policy.Precision, req.Method, req.Currency)
	if err != nil {
		return Breakdown{}, err
	}
	if err := checkAmount(req.Amount, places, req.Currency); err != nil {
		return Breakdown{}, err
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
	return computeChecked(r, req.Amount)
}

// checkInvertible keeps a customer-borne total strictly increasing across slab bounds, so PostFee can recover
// the base from the captured gross (baseFromGross) without ambiguity.
func checkInvertible(p Pricing, places int32) error {
	if p.FeeBearer != BearerCustomer || len(p.Slabs) < 2 {
		return nil
	}
	r := Rule{MinorUnits: places, Slabs: p.Slabs, MinFee: p.MinFee, MaxFee: p.MaxFee, Taxable: p.Taxable, TaxPercent: p.TaxPercent, FeeBearer: BearerCustomer}
	unit := decimal.New(1, -places)
	for _, s := range p.Slabs[:len(p.Slabs)-1] {
		if !Compute(r, s.UpTo.Add(unit)).CustomerTotal.GreaterThan(Compute(r, *s.UpTo).CustomerTotal) {
			return invalid("slabs", "a customer-borne rule must not lower the customer total above up_to %s", s.UpTo)
		}
	}
	return nil
}
