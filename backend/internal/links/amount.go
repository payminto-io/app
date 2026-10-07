package links

import (
	"github.com/shopspring/decimal"
)

var hundred = decimal.NewFromInt(100)

// LineTotals is one line's server-computed money: quantity x unit price, plus tax rounded half-up per line.
type LineTotals struct {
	Subtotal decimal.Decimal
	Tax      decimal.Decimal
	Total    decimal.Decimal
}

func lineTotals(li LineItem, places int32) LineTotals {
	sub := li.UnitPrice.Mul(decimal.NewFromInt(int64(li.Quantity)))
	tax := sub.Mul(li.TaxRate).Div(hundred).Round(places)
	return LineTotals{Subtotal: sub, Tax: tax, Total: sub.Add(tax)}
}

// ItemsTotals sums the lines; the link's amount in line-items mode is Total, never a client figure.
type ItemsTotals struct {
	Lines    []LineTotals
	Subtotal decimal.Decimal
	Tax      decimal.Decimal
	Total    decimal.Decimal
}

func itemsTotals(items []LineItem, places int32) ItemsTotals {
	out := ItemsTotals{Lines: make([]LineTotals, len(items))}
	for i, li := range items {
		t := lineTotals(li, places)
		out.Lines[i] = t
		out.Subtotal = out.Subtotal.Add(t.Subtotal)
		out.Tax = out.Tax.Add(t.Tax)
		out.Total = out.Total.Add(t.Total)
	}
	return out
}

// storedAmount is what the amount column holds: the fixed amount, the line sum, or nothing for customer-entered.
func storedAmount(in Input, places int32) *decimal.Decimal {
	switch in.AmountMode {
	case AmountFixed:
		return in.Amount
	case AmountLineItems:
		if len(in.LineItems) == 0 {
			return nil
		}
		t := itemsTotals(in.LineItems, places).Total
		return &t
	}
	return nil
}

// payAmount is the base amount one payment is for. Only customer mode reads the payer's figure, inside [min, max].
func payAmount(l Link, requested *decimal.Decimal, places int32) (decimal.Decimal, error) {
	switch l.AmountMode {
	case AmountFixed, AmountLineItems:
		if requested != nil {
			return decimal.Zero, newErr(CodeAmountNotAllowed, "amount", "this link's amount is set by the merchant")
		}
		if l.AmountMode == AmountLineItems {
			return itemsTotals(l.LineItems, places).Total, nil
		}
		if l.Amount == nil {
			return decimal.Zero, newErr(CodeAmountRequired, "amount", "link has no amount")
		}
		return *l.Amount, nil
	case AmountCustomer:
		if requested == nil {
			return decimal.Zero, newErr(CodeAmountRequired, "amount", "enter an amount")
		}
		if err := checkMoney("amount", *requested, places); err != nil {
			return decimal.Zero, err
		}
		if l.AmountMin != nil && requested.LessThan(*l.AmountMin) {
			return decimal.Zero, newErr(CodeAmountOutOfRange, "amount", "must be at least %s", l.AmountMin.String())
		}
		if l.AmountMax != nil && requested.GreaterThan(*l.AmountMax) {
			return decimal.Zero, newErr(CodeAmountOutOfRange, "amount", "must be at most %s", l.AmountMax.String())
		}
		return *requested, nil
	}
	return decimal.Zero, newErr(CodeAmountModeInvalid, "amount_mode", "unknown amount mode %q", l.AmountMode)
}

// maxMoney keeps values inside numeric(38,18) and refuses absurd exponents before any arithmetic.
var maxMoney = decimal.New(1, 20)

// checkMoney requires a positive value no finer than the currency's minor unit.
func checkMoney(field string, v decimal.Decimal, places int32) error {
	if v.Exponent() < -40 || v.Exponent() > 40 || !v.Abs().LessThan(maxMoney) {
		return newErr(CodeAmountInvalid, field, "out of range")
	}
	if !v.IsPositive() {
		return newErr(CodeAmountInvalid, field, "must be positive")
	}
	if !v.Equal(v.Truncate(places)) {
		return newErr(CodeAmountInvalid, field, "has more than %d decimal places", places)
	}
	return nil
}
