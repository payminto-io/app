package links

import (
	"context"
	"regexp"
	"testing"

	"github.com/shopspring/decimal"
)

func TestLineItemsAmountIsComputedServerSide(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.AmountMode, in.Amount = AmountLineItems, nil
	in.LineItems = []LineItem{
		{Name: "Beans", Quantity: 3, UnitPrice: dec("9.99"), TaxRate: dec("8.875")},
		{Name: "Mug", Quantity: 1, UnitPrice: dec("12.50"), TaxRate: dec("0")},
		{Name: "Sticker", Quantity: 2, UnitPrice: dec("0.05"), TaxRate: dec("5")},
	}
	l := f.create(in)
	// 29.97 + round(2.65983750) 2.66 = 32.63; 12.50; 0.10 + round(0.005) 0.01 = 0.11; total 45.24.
	if l.Total == nil || !l.Total.Equal(dec("45.24")) {
		t.Fatalf("total %v, want 45.24", l.Total)
	}
	if l.Amount != nil {
		t.Fatalf("line-items link kept a client amount: %v", l.Amount)
	}
	tot := itemsTotals(l.LineItems, 2)
	if !tot.Subtotal.Equal(dec("42.57")) || !tot.Tax.Equal(dec("2.67")) || !tot.Lines[2].Tax.Equal(dec("0.01")) {
		t.Fatalf("totals %+v", tot)
	}
}

func TestLineItemTaxRoundsToTheCurrencyGrid(t *testing.T) {
	got := lineTotals(LineItem{Name: "x", Quantity: 1, UnitPrice: dec("1001"), TaxRate: dec("10.05")}, 0)
	if !got.Tax.Equal(dec("101")) || !got.Total.Equal(dec("1102")) {
		t.Fatalf("JPY line %+v", got)
	}
}

func TestPayAmountPerMode(t *testing.T) {
	fixed := Link{Input: Input{AmountMode: AmountFixed, Amount: decp("25")}}
	if a, err := payAmount(fixed, nil, 2); err != nil || !a.Equal(dec("25")) {
		t.Fatalf("fixed %v %v", a, err)
	}
	if _, err := payAmount(fixed, decp("1"), 2); CodeOf(err) != CodeAmountNotAllowed {
		t.Fatalf("fixed took a client amount: %v", err)
	}

	items := Link{Input: Input{AmountMode: AmountLineItems, LineItems: []LineItem{{Name: "a", Quantity: 2, UnitPrice: dec("5"), TaxRate: dec("10")}}}}
	if a, err := payAmount(items, nil, 2); err != nil || !a.Equal(dec("11")) {
		t.Fatalf("line items %v %v", a, err)
	}
	if _, err := payAmount(items, decp("0.01"), 2); CodeOf(err) != CodeAmountNotAllowed {
		t.Fatalf("line items trusted a client total: %v", err)
	}

	cust := Link{Input: Input{AmountMode: AmountCustomer, AmountMin: decp("5"), AmountMax: decp("100")}}
	for _, tc := range []struct {
		in   *string
		code Code
	}{
		{nil, CodeAmountRequired}, {strp("4.99"), CodeAmountOutOfRange}, {strp("100.01"), CodeAmountOutOfRange},
		{strp("0"), CodeAmountInvalid}, {strp("-5"), CodeAmountInvalid}, {strp("10.001"), CodeAmountInvalid},
		{strp("5"), ""}, {strp("100"), ""}, {strp("42.50"), ""},
	} {
		var amt *decimal.Decimal
		if tc.in != nil {
			amt = decp(*tc.in)
		}
		_, got := payAmount(cust, amt, 2)
		if CodeOf(got) != tc.code {
			t.Errorf("customer amount %v: %v, want %q", deref(tc.in), got, tc.code)
		}
	}
	open := Link{Input: Input{AmountMode: AmountCustomer}}
	if a, err := payAmount(open, decp("1000000"), 2); err != nil || !a.Equal(dec("1000000")) {
		t.Fatalf("unbounded customer amount %v %v", a, err)
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestShortCodesAreLongUnguessableAndUnique(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Za-z0-9]{12}$`)
	seen := map[string]bool{}
	chars := map[rune]bool{}
	for range 5000 {
		c, err := NewShortCode()
		if err != nil {
			t.Fatal(err)
		}
		if !pattern.MatchString(c) || !ValidShortCode(c) {
			t.Fatalf("bad short code %q", c)
		}
		if seen[c] {
			t.Fatalf("duplicate short code %q in 5000 draws", c)
		}
		seen[c] = true
		for _, r := range c {
			chars[r] = true
		}
	}
	if len(chars) != len(shortCodeAlphabet) {
		t.Fatalf("only %d of %d alphabet characters drawn", len(chars), len(shortCodeAlphabet))
	}
	for _, bad := range []string{"", "short", "has space here", "abc/def/ghi", "abcdefgh-ijk", "x' OR 1=1 --"} {
		if ValidShortCode(bad) {
			t.Fatalf("%q accepted as a short code", bad)
		}
	}
}

func TestEveryPublishedLinkGetsADistinctCode(t *testing.T) {
	f := newFixture(t)
	seen := map[string]bool{}
	for range 50 {
		l := f.published(validInput())
		if seen[l.ShortCode] {
			t.Fatalf("short code %s reused", l.ShortCode)
		}
		seen[l.ShortCode] = true
		got, err := f.svc.store.GetByShortCode(context.Background(), l.ShortCode)
		if err != nil || got.ID != l.ID {
			t.Fatalf("lookup by code %v", err)
		}
	}
}
