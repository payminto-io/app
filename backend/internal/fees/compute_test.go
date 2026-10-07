package fees

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func dp(s string) *decimal.Decimal { v := d(s); return &v }

func usdRule() Rule {
	return Rule{ID: 7, Version: 3, Scope: Scope{Method: MethodCard, Currency: "USD"}, FeeBearer: BearerMerchant}
}

func assertDec(t *testing.T, name string, got decimal.Decimal, want string) {
	t.Helper()
	if !got.Equal(d(want)) {
		t.Errorf("%s = %s, want %s", name, got, want)
	}
}

func TestComputePercentPlusFlat(t *testing.T) {
	r := usdRule()
	r.Percent, r.Flat = d("2.9"), d("0.30")
	b := Compute(r, d("100"))
	assertDec(t, "fee", b.Fee, "3.20")
	assertDec(t, "tax", b.Tax, "0")
	if b.RuleID != 7 || b.Version != 3 || b.Currency != "USD" {
		t.Errorf("snapshot fields = %+v", b)
	}
}

func TestComputeSlabBoundaries(t *testing.T) {
	r := usdRule()
	r.Slabs = []Slab{
		{UpTo: dp("100"), Percent: d("3"), Flat: d("0")},
		{UpTo: dp("1000"), Percent: d("2"), Flat: d("0")},
		{UpTo: nil, Percent: d("1"), Flat: d("5")},
	}
	cases := []struct{ amount, fee string }{
		{"0.01", "0.00"}, // 0.0003 rounds to zero
		{"99.99", "3.00"},
		{"100", "3.00"},    // upper bound is inclusive: still the first slab
		{"100.01", "2.00"}, // just past the bound: the second slab (2.0002 rounds to 2.00)
		{"1000", "20.00"},
		{"1000.01", "15.00"}, // open-ended slab: 10.0001 + 5
		{"50000", "505.00"},
	}
	for _, tc := range cases {
		b := Compute(r, d(tc.amount))
		assertDec(t, "fee for "+tc.amount, b.Fee, tc.fee)
	}
}

func TestComputeMinAndMaxClamp(t *testing.T) {
	r := usdRule()
	r.Percent = d("1")
	r.MinFee, r.MaxFee = dp("0.50"), dp("10")
	assertDec(t, "below min", Compute(r, d("10")).Fee, "0.50")
	assertDec(t, "between", Compute(r, d("500")).Fee, "5.00")
	assertDec(t, "at max", Compute(r, d("1000")).Fee, "10")
	assertDec(t, "above max", Compute(r, d("5000")).Fee, "10")
}

func TestComputeTax(t *testing.T) {
	r := usdRule()
	r.Percent = d("2")
	r.Taxable, r.TaxPercent = true, d("18")
	b := Compute(r, d("123.45"))
	assertDec(t, "fee", b.Fee, "2.47") // 2.469 half-up
	assertDec(t, "tax", b.Tax, "0.44") // 18% of the rounded fee: 0.4446
	assertDec(t, "net", b.MerchantNet, "120.54")

	r.Taxable, r.TaxPercent = false, d("0")
	assertDec(t, "untaxed", Compute(r, d("123.45")).Tax, "0")
}

func TestComputeFeeBearer(t *testing.T) {
	r := usdRule()
	r.Percent, r.Taxable, r.TaxPercent = d("2"), true, d("10")

	r.FeeBearer = BearerCustomer
	b := Compute(r, d("100"))
	assertDec(t, "customer total (customer bears)", b.CustomerTotal, "102.20")
	assertDec(t, "merchant net (customer bears)", b.MerchantNet, "100")

	r.FeeBearer = BearerMerchant
	b = Compute(r, d("100"))
	assertDec(t, "customer total (merchant bears)", b.CustomerTotal, "100")
	assertDec(t, "merchant net (merchant bears)", b.MerchantNet, "97.80")
	if b.FeeBearer != BearerMerchant {
		t.Errorf("bearer = %s", b.FeeBearer)
	}
}

func TestComputeRoundingHalfUpPerCurrency(t *testing.T) {
	cases := []struct {
		currency, amount, percent, fee string
	}{
		{"USD", "0.50", "1", "0.01"},             // 0.005 rounds half up
		{"USD", "0.49", "1", "0.00"},             // 0.0049 rounds down
		{"JPY", "150", "1", "2"},                 // zero-decimal fiat: 1.5 -> 2
		{"KWD", "1.2345", "10", "0.123"},         // three-decimal fiat: 0.12345 -> 0.123
		{"KWD", "1.2355", "10", "0.124"},         // 0.12355 -> 0.124
		{"USDC", "1.0000005", "100", "1.000001"}, // six-decimal asset
		{"USDC", "0.1234565", "10", "0.012346"},
		{"BTC", "0.123456785", "100", "0.12345679"},
	}
	for _, tc := range cases {
		r := usdRule()
		r.Currency, r.Percent = tc.currency, d(tc.percent)
		assertDec(t, tc.currency+" "+tc.amount, Compute(r, d(tc.amount)).Fee, tc.fee)
	}
}

func TestMinorUnits(t *testing.T) {
	cases := map[string]int32{"USD": 2, "EUR": 2, "INR": 2, "JPY": 0, "KWD": 3, "USDC": 6, "USDT": 6, "BTC": 8, "ETH": 18, "SOL": 9}
	for code, want := range cases {
		got, ok := MinorUnits(code)
		if !ok || got != want {
			t.Errorf("MinorUnits(%s) = %d,%v want %d", code, got, ok, want)
		}
	}
	for _, code := range []string{"", "usd", "DOGECOIN", "US"} {
		if _, ok := MinorUnits(code); ok {
			t.Errorf("MinorUnits(%q) accepted an unknown code", code)
		}
	}
}
