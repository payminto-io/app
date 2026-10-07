package fees

import (
	"errors"
	"testing"
	"time"
)

func validInput() RuleInput {
	return RuleInput{
		Scope:   Scope{Method: MethodCard, Currency: "USD"},
		Pricing: Pricing{Percent: d("2.9"), Flat: d("0.30"), FeeBearer: BearerMerchant},
	}
}

func wantField(t *testing.T, err error, field string) {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != field {
		t.Fatalf("err = %v, want ValidationError on %q", err, field)
	}
}

func TestValidateInputNormalisesAndDefaults(t *testing.T) {
	in := validInput()
	in.Connector, in.CardType, in.Region, in.Currency = sp(" Stripe "), ct(CardCredit), sp("in"), "usd"
	got, err := validateInput(in, DefaultPolicy(), t0)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Connector != "stripe" || *got.Region != "IN" || got.Currency != "USD" {
		t.Errorf("not normalised: %+v", got.Scope)
	}
	if got.EffectiveFrom == nil || !got.EffectiveFrom.Equal(t0) {
		t.Errorf("effective_from = %v, want now", got.EffectiveFrom)
	}
}

func TestValidateInputRejects(t *testing.T) {
	past := t0.Add(-time.Minute)
	cases := []struct {
		name  string
		mut   func(*RuleInput)
		field string
	}{
		{"unknown method", func(in *RuleInput) { in.Method = "cash" }, "method"},
		{"unknown currency", func(in *RuleInput) { in.Currency = "DOGECOIN" }, "currency"},
		{"bad connector", func(in *RuleInput) { in.Connector = sp("no spaces allowed") }, "connector"},
		{"card type without connector", func(in *RuleInput) { in.CardType = ct(CardDebit) }, "card_type"},
		{"card type on a non-card method", func(in *RuleInput) { in.Method, in.Connector, in.CardType = MethodBank, sp("x"), ct(CardDebit) }, "card_type"},
		{"unknown card type", func(in *RuleInput) { in.Connector, in.CardType = sp("x"), ct("gold") }, "card_type"},
		{"region without card type", func(in *RuleInput) { in.Connector, in.Region = sp("x"), sp("IN") }, "region"},
		{"bad region", func(in *RuleInput) { in.Connector, in.CardType, in.Region = sp("x"), ct(CardDebit), sp("I-N") }, "region"},
		{"negative percent", func(in *RuleInput) { in.Percent = d("-1") }, "percent"},
		{"percent above 100", func(in *RuleInput) { in.Percent = d("100.01") }, "percent"},
		{"negative flat", func(in *RuleInput) { in.Flat = d("-0.01") }, "flat"},
		{"min above max", func(in *RuleInput) { in.MinFee, in.MaxFee = dp("5"), dp("4") }, "min_fee"},
		{"negative min", func(in *RuleInput) { in.MinFee = dp("-1") }, "min_fee"},
		{"max finer than the currency", func(in *RuleInput) { in.MaxFee = dp("1.001") }, "max_fee"},
		{"taxable without a rate", func(in *RuleInput) { in.Taxable = true }, "tax_percent"},
		{"rate without taxable", func(in *RuleInput) { in.TaxPercent = d("18") }, "tax_percent"},
		{"unknown bearer", func(in *RuleInput) { in.FeeBearer = "nobody" }, "fee_bearer"},
		{"backdated", func(in *RuleInput) { in.EffectiveFrom = &past }, "effective_from"},
		{"empty window", func(in *RuleInput) { in.EffectiveFrom, in.EffectiveTo = ptrTime(t0), ptrTime(t0) }, "effective_to"},
		{"slabs with rule-level percent", func(in *RuleInput) { in.Slabs = []Slab{{UpTo: nil, Percent: d("1")}} }, "slabs"},
		{"slabs not increasing", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{UpTo: dp("100"), Percent: d("1")}, {UpTo: dp("100"), Percent: d("1")}, {Percent: d("1")}}
		}, "slabs"},
		{"slabs without an open last slab", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{UpTo: dp("100"), Percent: d("1")}}
		}, "slabs"},
		{"open slab before the last", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{Percent: d("1")}, {Percent: d("1")}}
		}, "slabs"},
		{"slab percent above 100", func(in *RuleInput) {
			in.Percent, in.Flat = d("0"), d("0")
			in.Slabs = []Slab{{Percent: d("101")}}
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
}

func TestValidateInputForbidsSurchargeWhereMethodDisallows(t *testing.T) {
	in := validInput()
	in.Method, in.Currency, in.FeeBearer = MethodUPI, "INR", BearerCustomer
	if _, err := validateInput(in, DefaultPolicy(), t0); !errors.Is(err, ErrSurchargeForbidden) {
		t.Fatalf("err = %v, want ErrSurchargeForbidden", err)
	}
	in.FeeBearer = BearerMerchant
	if _, err := validateInput(in, DefaultPolicy(), t0); err != nil {
		t.Fatalf("merchant-borne UPI fee rejected: %v", err)
	}
	in.Method, in.FeeBearer = MethodCard, BearerCustomer
	if _, err := validateInput(in, DefaultPolicy(), t0); err != nil {
		t.Fatalf("card surcharge rejected under the default policy: %v", err)
	}
}

func TestParsePolicy(t *testing.T) {
	p, err := ParsePolicy("card, upi", "")
	if err != nil || !p.SurchargeForbidden[MethodCard] || !p.SurchargeForbidden[MethodUPI] || p.SurchargeForbidden[MethodBank] {
		t.Fatalf("policy = %+v err %v", p, err)
	}
	if _, err := ParsePolicy("card,cash", ""); err == nil {
		t.Fatal("unknown method accepted")
	}
	if p, err := ParsePolicy("", ""); err != nil || !p.SurchargeForbidden[MethodUPI] {
		t.Fatalf("empty value should give the default policy: %+v %v", p, err)
	}
	if p, err := ParsePolicy("none", ""); err != nil || len(p.SurchargeForbidden) != 0 {
		t.Fatalf("none should allow every surcharge: %+v %v", p, err)
	}
}

func TestParsePolicyReadsPrecision(t *testing.T) {
	p, err := ParsePolicy("", "XRP:6")
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := p.Precision.MinorUnits("XRP"); !ok || n != 6 {
		t.Fatalf("XRP = %d,%v", n, ok)
	}
	if _, err := ParsePolicy("", "USD:3"); err == nil {
		t.Fatal("fiat override accepted")
	}
}
