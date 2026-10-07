package fees

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// roundHalfUp relies on decimal.Round rounding half away from zero, which is half-up for the non-negative values here.
func roundHalfUp(v decimal.Decimal, places int32) decimal.Decimal {
	return v.Round(places)
}

func slabFor(slabs []Slab, amount decimal.Decimal) Slab {
	for _, s := range slabs {
		if s.UpTo == nil || amount.LessThanOrEqual(*s.UpTo) {
			return s
		}
	}
	return slabs[len(slabs)-1]
}

// Compute applies r to amount: percent of the amount plus flat (or the slab's pair), clamped to
// [min, max], rounded half-up to the currency's minor unit, then tax on the rounded fee.
// Rounding uses r.MinorUnits, fixed when the rule was written.
func Compute(r Rule, amount decimal.Decimal) Breakdown {
	places := r.MinorUnits
	percent, flat := r.Percent, r.Flat
	if len(r.Slabs) > 0 {
		s := slabFor(r.Slabs, amount)
		percent, flat = s.Percent, s.Flat
	}
	fee := amount.Mul(percent).Shift(-2).Add(flat)
	if r.MinFee != nil && fee.LessThan(*r.MinFee) {
		fee = *r.MinFee
	}
	if r.MaxFee != nil && fee.GreaterThan(*r.MaxFee) {
		fee = *r.MaxFee
	}
	fee = roundHalfUp(fee, places)
	tax := decimal.Zero
	if r.Taxable {
		tax = roundHalfUp(fee.Mul(r.TaxPercent).Shift(-2), places)
	}
	b := Breakdown{
		RuleID:    r.ID,
		Version:   r.Version,
		Currency:  r.Currency,
		FeeBearer: r.FeeBearer,
		Amount:    amount,
		Fee:       fee,
		Tax:       tax,
	}
	if r.FeeBearer == BearerCustomer {
		b.CustomerTotal = amount.Add(fee).Add(tax)
		b.MerchantNet = amount
	} else {
		b.CustomerTotal = amount
		b.MerchantNet = amount.Sub(fee).Sub(tax)
	}
	return b
}

// computeChecked refuses a fee plus tax above the amount rather than reporting a negative net.
func computeChecked(r Rule, amount decimal.Decimal) (Breakdown, error) {
	b := Compute(r, amount)
	if b.Fee.Add(b.Tax).GreaterThan(amount) {
		return Breakdown{}, fmt.Errorf("%w: fee %s plus tax %s on %s %s", ErrFeeExceedsAmount, b.Fee, b.Tax, amount, r.Currency)
	}
	return b, nil
}
