package links

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
)

// renderKeys is the public contract with checkout; adding a key is a deliberate decision, not a side effect.
var renderKeys = []string{
	"short_code", "url", "available", "unavailable_reason", "merchant_name", "title", "description",
	"amount_mode", "amount", "amount_min", "amount_max", "currency", "line_items", "subtotal", "tax_total",
	"customer_fields", "billing_required", "shipping_required", "questions", "methods", "fee_bearer",
	"chain_tolerance_bps", "quote_expiry_seconds", "success_mode", "success_message", "failure_retry",
	"failure_message", "receipt_email", "expires_at", "branding",
}

func secretiveInput() Input {
	in := validInput()
	in.Description = "Single origin"
	in.ReferenceID = "order-SECRET-REF"
	in.Metadata = map[string]string{"crm_id": "SECRET-META"}
	in.Category = "SECRET-CATEGORY"
	in.WebhookID = uintp(11)
	in.ReceiptNote = "SECRET-NOTE"
	in.SettlementOverride = &SettlementOverride{Kind: SettleFiat, DestinationID: "dest_ok"}
	in.CustomerFields.Email = FieldRule{Mode: FieldHidden, Prefill: "vip@secret.test"}
	in.CustomerFields.Name = FieldRule{Mode: FieldRequired, Prefill: "Ada"}
	in.SuccessMode, in.SuccessURL = SuccessRedirect, "https://shop.test/thanks?SECRET-URL=1"
	in.Methods = []MethodSpec{card, usdcSol}
	in.Questions = []Question{{Key: "size", Label: "Size", Type: QuestionSelect, Options: []string{"S", "M"}, Required: true, PerOrder: true}}
	in.LogoURL, in.AccentColor = "https://cdn.test/logo.png", "#112233"
	return in
}

func TestRenderModelExposesOnlyWhatCheckoutNeeds(t *testing.T) {
	f := newFixture(t)
	l := f.published(secretiveInput())
	m, err := f.svc.Render(context.Background(), l.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(m)
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range top {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	want := slices.Clone(renderKeys)
	slices.Sort(want)
	if !slices.Equal(keys, want) {
		t.Fatalf("render keys\n got %v\nwant %v", keys, want)
	}
	body := string(raw)
	for _, secret := range []string{l.ID, "SECRET", "vip@secret.test", "dest_ok", `"id"`, "member", "platform", "webhook", "revision", "uses_count", "environment", "connector", "reference"} {
		if strings.Contains(body, secret) {
			t.Errorf("render model leaks %q: %s", secret, body)
		}
	}
	if !m.Available || m.UnavailableReason != nil || *m.MerchantName != "Acme Coffee" || m.URL != "https://checkout.test/l/"+l.ShortCode {
		t.Fatalf("render %+v", m)
	}
	if m.CustomerFields.Email.Prefill != nil || m.CustomerFields.Email.Mode != FieldHidden {
		t.Fatalf("hidden prefill shown: %+v", m.CustomerFields.Email)
	}
	if m.CustomerFields.Name.Prefill == nil || *m.CustomerFields.Name.Prefill != "Ada" {
		t.Fatalf("visible prefill missing: %+v", m.CustomerFields.Name)
	}
	if len(m.Methods) != 2 || *m.Methods[1].Chain != "SOL" || m.Methods[0].Chain != nil || m.Methods[0].Fee != nil {
		t.Fatalf("methods %+v", m.Methods)
	}
	if m.Questions[0].Key != "size" || len(m.Questions[0].Options) != 2 {
		t.Fatalf("questions %+v", m.Questions)
	}
}

func TestRenderShowsSurchargeTotalsAndLineItems(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.FeeBearer = fees.BearerCustomer
	in.AmountMode, in.Amount = AmountLineItems, nil
	in.LineItems = []LineItem{{Name: "Beans", Quantity: 2, UnitPrice: dec("50"), TaxRate: dec("10")}}
	l := f.published(in)
	m, err := f.svc.Render(context.Background(), l.ShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Amount.Equal(dec("110")) || !m.Subtotal.Equal(dec("100")) || !m.TaxTotal.Equal(dec("10")) || !m.LineItems[0].Total.Equal(dec("110")) {
		t.Fatalf("amounts %v %v %v %+v", m.Amount, m.Subtotal, m.TaxTotal, m.LineItems)
	}
	// 2.9% of 110 = 3.19, customer pays 113.19.
	if m.Methods[0].Fee == nil || !m.Methods[0].Fee.Equal(dec("3.19")) || !m.Methods[0].CustomerTotal.Equal(dec("113.19")) {
		t.Fatalf("surcharge %+v", m.Methods[0])
	}
}

func TestRenderOmitsMethodsThatLostTheirRuleOrConnector(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.Methods = []MethodSpec{card, upi}
	l := f.published(in)
	f.creator.offer(upi)
	m, _ := f.svc.Render(context.Background(), l.ShortCode)
	if len(m.Methods) != 1 || m.Methods[0].Method != fees.MethodCard {
		t.Fatalf("methods %+v", m.Methods)
	}
	f.creator.offer(card)
	m, _ = f.svc.Render(context.Background(), l.ShortCode)
	if m.Available || *m.UnavailableReason != ReasonNoMethods {
		t.Fatalf("no methods but available: %+v", m)
	}
}

func TestRenderStates(t *testing.T) {
	f := newFixture(t)
	ctx, pid := context.Background(), merchant.PlatformID

	draft := f.create(validInput())
	if _, err := f.svc.Render(ctx, "AAAAAAAAAAAA"); CodeOf(err) != CodeNotFound {
		t.Fatalf("unknown code: %v", err)
	}
	if _, err := f.svc.Render(ctx, "../../etc"); CodeOf(err) != CodeNotFound {
		t.Fatalf("malformed code: %v", err)
	}
	_ = draft

	paused := f.linkIn(StatusPaused)
	m, err := f.svc.Render(ctx, paused.ShortCode)
	if err != nil || m.Available || *m.UnavailableReason != ReasonPaused {
		t.Fatalf("paused: %+v %v", m, err)
	}

	archived := f.linkIn(StatusActive)
	f.svc.Archive(ctx, pid, archived.ID)
	if _, err := f.svc.Render(ctx, archived.ShortCode); CodeOf(err) != CodeArchived {
		t.Fatalf("archived: %v", err)
	}

	in := validInput()
	exp := f.now.Add(time.Hour)
	in.ExpiresAt = &exp
	expiring := f.published(in)
	f.now = exp
	m, _ = f.svc.Render(ctx, expiring.ShortCode)
	if m.Available || *m.UnavailableReason != ReasonExpired {
		t.Fatalf("expired: %+v", m)
	}
}
