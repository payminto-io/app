package fees

import (
	"errors"
	"testing"
	"time"
)

func TestPrecisionKnowsISOFiatAndConfiguredAssetsOnly(t *testing.T) {
	p := DefaultPrecision()
	cases := map[string]int32{"USD": 2, "EUR": 2, "INR": 2, "JPY": 0, "KWD": 3, "CLF": 4, "USDC": 6, "USDT": 6, "BTC": 8, "ETH": 18, "SOL": 9}
	for code, want := range cases {
		if got, ok := p.MinorUnits(code); !ok || got != want {
			t.Errorf("MinorUnits(%s) = %d,%v want %d", code, got, ok, want)
		}
	}
	// M2: three-letter codes that are not ISO 4217 are not fiat.
	for _, code := range []string{"", "usd", "XRP", "TON", "ADA", "USF", "DOGECOIN", "US"} {
		if _, ok := p.MinorUnits(code); ok {
			t.Errorf("MinorUnits(%q) accepted an unknown code", code)
		}
	}
	if !p.IsFiat("USD") || p.IsFiat("USDC") || p.IsFiat("XRP") {
		t.Error("IsFiat misclassifies")
	}
}

func TestParsePrecisionExtendsAssets(t *testing.T) {
	p, err := ParsePrecision("XRP:6, ton:9")
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := p.MinorUnits("XRP"); !ok || n != 6 || p.IsFiat("XRP") {
		t.Errorf("XRP = %d,%v", n, ok)
	}
	if n, ok := p.MinorUnits("TON"); !ok || n != 9 {
		t.Errorf("TON = %d,%v", n, ok)
	}
	for _, bad := range []string{"XRP", "XRP:x", "XRP:19", "XRP:-1", "USD:3", "x y:2"} {
		if _, err := ParsePrecision(bad); err == nil {
			t.Errorf("ParsePrecision(%q) accepted", bad)
		}
	}
	if p, err := ParsePrecision(""); err != nil || len(p.assets) != len(DefaultPrecision().assets) {
		t.Errorf("empty value should give the defaults: %v", err)
	}
}

func TestValidateRejectsUnconfiguredCurrency(t *testing.T) {
	in := validInput()
	in.Method, in.Currency = MethodCrypto, "XRP"
	_, err := validateInput(in, DefaultPolicy(), t0)
	wantField(t, err, "currency")

	policy := DefaultPolicy()
	policy.Precision, _ = ParsePrecision("XRP:6")
	got, err := validateInput(in, policy, t0)
	if err != nil || got.minorUnits != 6 {
		t.Fatalf("configured XRP: %v minor units %d", err, got.minorUnits)
	}
}

func TestValidateMatchesCurrencyClassToMethod(t *testing.T) {
	in := validInput()
	in.Currency = "USDC" // card priced in a crypto asset
	wantField(t, func() error { _, err := validateInput(in, DefaultPolicy(), t0); return err }(), "currency")
	in.Method, in.Currency = MethodCrypto, "USD" // crypto rail priced in fiat
	wantField(t, func() error { _, err := validateInput(in, DefaultPolicy(), t0); return err }(), "currency")
}

func TestValidateRejectsPrecisionBeyondTheColumn(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*RuleInput)
		field string
	}{
		{"percent beyond 6 places", func(in *RuleInput) { in.Percent = d("2.9999999") }, "percent"},
		{"tax percent beyond 6 places", func(in *RuleInput) { in.Taxable, in.TaxPercent = true, d("18.0000001") }, "tax_percent"},
		{"flat finer than the currency", func(in *RuleInput) { in.Flat = d("0.305") }, "flat"},
		{"flat too large", func(in *RuleInput) { in.Flat = d("100000000000000000000") }, "flat"},
		{"slab up_to finer than the currency", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{UpTo: dp("100.001"), Percent: d("1")}, {Percent: d("1")}}
		}, "slabs"},
		{"slab percent beyond 6 places", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{Percent: d("1.0000001")}}
		}, "slabs"},
		{"slab flat finer than the currency", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{Percent: d("1"), Flat: d("0.001")}}
		}, "slabs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mut(&in)
			_, err := validateInput(in, DefaultPolicy(), t0)
			wantField(t, err, tc.field)
		})
	}
	// Trailing zeros are not precision.
	in := validInput()
	in.Percent, in.Flat = d("2.90000000"), d("0.3000")
	if _, err := validateInput(in, DefaultPolicy(), t0); err != nil {
		t.Fatalf("trailing zeros rejected: %v", err)
	}
}

func TestHugeExponentsAreRejectedBeforeArithmetic(t *testing.T) {
	for _, raw := range []string{"1e100000000", "1e-100000000", "-1e2000000000"} {
		start := time.Now()
		req := previewReq("1")
		req.Amount = d(raw)
		_, err := preview([]Rule{rule(4, nil, nil, nil)}, req, DefaultPolicy())
		wantField(t, err, "amount")
		if el := time.Since(start); el > 10*time.Millisecond {
			t.Errorf("preview amount %s took %s", raw, el)
		}

		start = time.Now()
		in := validInput()
		in.Percent, in.Flat = d("0"), d("0")
		upTo := d(raw)
		in.Slabs = []Slab{{UpTo: &upTo, Percent: d("1")}, {Percent: d("1")}}
		_, err = validateInput(in, DefaultPolicy(), t0)
		wantField(t, err, "slabs")
		if el := time.Since(start); el > 10*time.Millisecond {
			t.Errorf("slab up_to %s took %s", raw, el)
		}
	}
}

func TestPreviewRefusesAFeeLargerThanTheAmount(t *testing.T) {
	r := rule(4, nil, nil, nil)
	r.Percent, r.MinFee = d("1"), dp("5")
	for _, bearer := range []FeeBearer{BearerMerchant, BearerCustomer} {
		r.FeeBearer = bearer
		if _, err := preview([]Rule{r}, previewReq("4.99"), DefaultPolicy()); !errors.Is(err, ErrFeeExceedsAmount) {
			t.Errorf("%s: err = %v, want ErrFeeExceedsAmount", bearer, err)
		}
	}
	r.FeeBearer = BearerMerchant
	b, err := preview([]Rule{r}, previewReq("5"), DefaultPolicy())
	if err != nil || !b.MerchantNet.IsZero() {
		t.Fatalf("fee equal to amount: %+v %v", b, err)
	}
	r.Taxable, r.TaxPercent = true, d("10")
	if _, err := preview([]Rule{r}, previewReq("5"), DefaultPolicy()); !errors.Is(err, ErrFeeExceedsAmount) {
		t.Fatalf("fee plus tax above amount: err = %v", err)
	}
}
