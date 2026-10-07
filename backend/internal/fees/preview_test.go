package fees

import (
	"errors"
	"testing"
)

func previewReq(amount string) PreviewRequest {
	return PreviewRequest{Query: Query{Method: MethodCard, Currency: "USD", At: t0}, Amount: d(amount)}
}

func TestPreviewAppliesTheResolvedRule(t *testing.T) {
	r := rule(4, nil, nil, nil)
	r.Percent = d("2")
	b, err := preview([]Rule{r}, previewReq("50"), DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if b.RuleID != 4 || !b.Fee.Equal(d("1")) || !b.MerchantNet.Equal(d("49")) {
		t.Fatalf("breakdown = %+v", b)
	}
}

func TestPreviewFeeBearerOverride(t *testing.T) {
	r := rule(4, nil, nil, nil)
	r.Percent = d("2")
	req := previewReq("50")
	bearer := BearerCustomer
	req.FeeBearer = &bearer
	b, err := preview([]Rule{r}, req, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if !b.CustomerTotal.Equal(d("51")) || !b.MerchantNet.Equal(d("50")) || b.FeeBearer != BearerCustomer {
		t.Fatalf("breakdown = %+v", b)
	}
}

func TestPreviewForbidsSurchargeForMethod(t *testing.T) {
	r := rule(4, nil, nil, nil)
	r.Method, r.Currency = MethodUPI, "INR"
	req := PreviewRequest{Query: Query{Method: MethodUPI, Currency: "INR", At: t0}, Amount: d("100")}
	bearer := BearerCustomer
	req.FeeBearer = &bearer
	if _, err := preview([]Rule{r}, req, DefaultPolicy()); !errors.Is(err, ErrSurchargeForbidden) {
		t.Fatalf("err = %v, want ErrSurchargeForbidden", err)
	}
	// A stored customer-borne rule is also refused once policy forbids it.
	r.FeeBearer = BearerCustomer
	req.FeeBearer = nil
	if _, err := preview([]Rule{r}, req, DefaultPolicy()); !errors.Is(err, ErrSurchargeForbidden) {
		t.Fatalf("err = %v, want ErrSurchargeForbidden for a stored rule", err)
	}
}

func TestPreviewValidatesRequest(t *testing.T) {
	r := rule(4, nil, nil, nil)
	cases := []struct {
		name  string
		mut   func(*PreviewRequest)
		field string
	}{
		{"zero amount", func(p *PreviewRequest) { p.Amount = d("0") }, "amount"},
		{"negative amount", func(p *PreviewRequest) { p.Amount = d("-5") }, "amount"},
		{"amount finer than the currency", func(p *PreviewRequest) { p.Amount = d("1.001") }, "amount"},
		{"unknown currency", func(p *PreviewRequest) { p.Currency = "NOPE1" }, "currency"},
		{"unknown method", func(p *PreviewRequest) { p.Method = "cash" }, "method"},
		{"unknown bearer", func(p *PreviewRequest) { b := FeeBearer("x"); p.FeeBearer = &b }, "fee_bearer"},
		{"unknown card type", func(p *PreviewRequest) { p.CardType = "gold" }, "card_type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := previewReq("10")
			tc.mut(&req)
			_, err := preview([]Rule{r}, req, DefaultPolicy())
			wantField(t, err, tc.field)
		})
	}
}
