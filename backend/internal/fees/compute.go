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

// baseFromGross finds the base amount whose customer total under r is gross: the inverse of a customer-borne
// surcharge. Each pricing regime (every slab's or the rule's percent+flat, the min clamp, the max clamp) gives an
// estimate; grid points around it are checked exactly with Compute. Exactly one base must match.
func baseFromGross(r Rule, gross decimal.Decimal) (decimal.Decimal, error) {
	unit := decimal.New(1, -r.MinorUnits)
	taxRate := decimal.Zero
	if r.Taxable {
		taxRate = r.TaxPercent.Shift(-2)
	}
	one := decimal.NewFromInt(1)
	withTax := one.Add(taxRate)
	var estimates []decimal.Decimal
	linear := func(percent, flat decimal.Decimal) {
		// gross = A + (A*p + flat) * (1 + t)  =>  A = (gross - flat*(1+t)) / (1 + p*(1+t))
		den := one.Add(percent.Shift(-2).Mul(withTax))
		estimates = append(estimates, gross.Sub(flat.Mul(withTax)).DivRound(den, r.MinorUnits+4))
	}
	constant := func(fee decimal.Decimal) {
		f := roundHalfUp(fee, r.MinorUnits)
		estimates = append(estimates, gross.Sub(f).Sub(roundHalfUp(f.Mul(taxRate), r.MinorUnits)))
	}
	if len(r.Slabs) > 0 {
		for _, s := range r.Slabs {
			linear(s.Percent, s.Flat)
		}
	} else {
		linear(r.Percent, r.Flat)
	}
	if r.MinFee != nil {
		constant(*r.MinFee)
	}
	if r.MaxFee != nil {
		constant(*r.MaxFee)
	}
	var found []decimal.Decimal
	for _, e := range estimates {
		center := e.Round(r.MinorUnits)
		for k := int64(-3); k <= 3; k++ {
			a := center.Add(unit.Mul(decimal.NewFromInt(k)))
			if !a.IsPositive() || Compute(r, a).CustomerTotal.Cmp(gross) != 0 {
				continue
			}
			dup := false
			for _, f := range found {
				dup = dup || f.Equal(a)
			}
			if !dup {
				found = append(found, a)
			}
		}
	}
	if len(found) != 1 {
		return decimal.Zero, fmt.Errorf("%w: %d base amounts give %s %s", ErrGrossMismatch, len(found), gross, r.Currency)
	}
	return found[0], nil
}
