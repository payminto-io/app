package fees

import (
	"regexp"

	"github.com/shopspring/decimal"
)

// assetMinorUnits lists non-fiat assets; their precision is the token's on-chain decimals.
var assetMinorUnits = map[string]int32{
	"USDC": 6, "USDT": 6, "PYUSD": 6, "EURC": 6, "DAI": 18,
	"BTC": 8, "ETH": 18, "SOL": 9, "TRX": 6, "POL": 18, "MATIC": 18, "BNB": 18,
}

// fiatMinorUnits lists ISO 4217 codes whose minor unit is not 2.
var fiatMinorUnits = map[string]int32{
	"JPY": 0, "KRW": 0, "VND": 0, "CLP": 0, "ISK": 0, "UGX": 0, "XAF": 0, "XOF": 0,
	"BHD": 3, "KWD": 3, "OMR": 3, "JOD": 3, "TND": 3, "IQD": 3, "LYD": 3,
}

var fiatCode = regexp.MustCompile(`^[A-Z]{3}$`)

// MinorUnits is the number of decimal places a currency settles in; see README "Rounding".
func MinorUnits(currency string) (int32, bool) {
	if n, ok := assetMinorUnits[currency]; ok {
		return n, true
	}
	if n, ok := fiatMinorUnits[currency]; ok {
		return n, true
	}
	if fiatCode.MatchString(currency) {
		return 2, true
	}
	return 0, false
}

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
// r must have passed validation; the currency is therefore known.
func Compute(r Rule, amount decimal.Decimal) Breakdown {
	places, _ := MinorUnits(r.Currency)
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
