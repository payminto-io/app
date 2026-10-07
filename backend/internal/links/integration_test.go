//go:build integration

package links_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func d(s string) *decimal.Decimal {
	v := decimal.RequireFromString(s)
	return &v
}

type creator struct {
	calls atomic.Int64
	fail  atomic.Bool
	delay time.Duration
}

func (c *creator) Connectors(_ context.Context, _ links.Environment, _ string, m links.MethodSpec) ([]string, error) {
	if m.Method == fees.MethodBank {
		return nil, nil
	}
	return []string{""}, nil
}

func (c *creator) CreatePayment(_ context.Context, req links.PaymentRequest) (links.CreatedPayment, error) {
	c.calls.Add(1)
	time.Sleep(c.delay)
	if c.fail.Load() {
		return links.CreatedPayment{}, errors.New("processor down")
	}
	return links.CreatedPayment{Reference: "ref-" + req.LinkPaymentID, CheckoutURL: "https://checkout.test/pay/ref-" + req.LinkPaymentID}, nil
}

type stack struct {
	db      *gorm.DB
	svc     *links.Service
	store   *links.PGStore
	creator *creator
	actor   links.Actor
	webhook uint
	ruleID  uint
}

func newStack(t *testing.T, opts ...links.Option) *stack {
	t.Helper()
	db, cleanup := database.NewTestDB(t)
	t.Cleanup(cleanup)
	platform := models.ExternalPlatform{Name: "Acme Coffee", SuccessEndpoint: "https://example.com"}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	member := models.Member{Name: "M", MemberType: "merchant"}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	hook := models.Webhook{URL: "https://hooks.test", Secret: "s", ExternalPlatformID: platform.ID}
	if err := db.Create(&hook).Error; err != nil {
		t.Fatal(err)
	}
	feeSvc := fees.NewService(db, nil, fees.DefaultPolicy())
	rule, err := feeSvc.CreateRule(context.Background(), fees.RuleInput{
		Scope:   fees.Scope{Method: fees.MethodCard, Currency: "USD"},
		Pricing: fees.Pricing{Percent: decimal.RequireFromString("2.9"), FeeBearer: fees.BearerMerchant},
	}, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	c := &creator{}
	store := links.NewPGStore(db)
	base := []links.Option{links.WithCheckoutBaseURL("https://checkout.test")}
	return &stack{
		db: db, store: store, creator: c, actor: links.Actor{MemberID: member.ID, PlatformID: platform.ID},
		svc: links.NewService(store, feeSvc, c, append(base, opts...)...), webhook: hook.ID, ruleID: rule.ID,
	}
}

func fullInput(webhook uint) links.Input {
	in := links.DefaultInput()
	in.Title = "Roastery box"
	in.Description = "Monthly beans"
	in.AmountMode = links.AmountLineItems
	in.Currency = "USD"
	in.ReferenceID = "order-1"
	in.Metadata = map[string]string{"crm": "42", "plan": "gold"}
	in.Category = "subscriptions"
	in.CustomerFields = links.CustomerFieldPolicy{
		Name:  links.FieldRule{Mode: links.FieldRequired, Prefill: "Ada"},
		Email: links.FieldRule{Mode: links.FieldRequired},
		Phone: links.FieldRule{Mode: links.FieldOptional, Prefill: "+14155550123"},
	}
	in.BillingRequired, in.ShippingRequired = true, true
	in.MultiUse, in.UseLimit, in.ExpiresAfterPayments = true, intp(10), intp(8)
	in.Methods = []links.MethodSpec{{Method: fees.MethodCard}}
	in.CaptureMode, in.ThreeDSPolicy = links.CaptureManual, links.ThreeDSForce
	in.ChainToleranceBps, in.QuoteExpirySeconds = 50, 600
	in.SuccessMode, in.SuccessURL, in.SuccessMessage = links.SuccessRedirect, "https://shop.test/thanks", "Thanks"
	in.ReceiptEmail, in.ReceiptNote = true, "See you soon"
	in.WebhookID = &webhook
	in.FailureRetry, in.FailureMessage = false, "Card declined"
	in.SettlementTiming = links.TimingImmediate
	exp := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Microsecond)
	in.ExpiresAt = &exp
	in.LogoURL, in.AccentColor, in.Language = "https://cdn.test/logo.png", "#AABBCC", "pt-BR"
	in.LineItems = []links.LineItem{
		{Name: "Beans", Quantity: 2, UnitPrice: *d("12.50"), TaxRate: *d("8.875")},
		{Name: "Grinder", Quantity: 1, UnitPrice: *d("80"), TaxRate: *d("0")},
	}
	in.Questions = []links.Question{
		{Key: "grind", Label: "Grind", Type: links.QuestionSelect, Options: []string{"whole", "fine"}, Required: true, PerOrder: true},
		{Key: "company", Label: "Company", Type: links.QuestionText, Required: true},
		{Key: "gift", Label: "Gift wrap", Type: links.QuestionCheckbox},
	}
	return in
}

func intp(n int) *int { return &n }

func TestIntegration_LinkRoundTripsEveryField(t *testing.T) {
	s := newStack(t)
	ctx := context.Background()
	in := fullInput(s.webhook)
	created, err := s.svc.Create(ctx, s.actor, in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.svc.Get(ctx, s.actor.PlatformID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Input, created.Input) {
		t.Fatalf("round trip differs\n got %+v\nwant %+v", got.Input, created.Input)
	}
	if got.Total == nil || !got.Total.Equal(*d("107.22")) || got.Environment != links.EnvTest || got.Revision != 1 {
		t.Fatalf("stored link %+v total %v", got, got.Total)
	}
	if !got.LineItems[0].TaxRate.Equal(*d("8.875")) || got.Questions[0].Options[1] != "fine" || got.Questions[1].PerOrder {
		t.Fatalf("children %+v %+v", got.LineItems, got.Questions)
	}
	var stored decimal.Decimal
	s.db.Raw(`SELECT amount FROM payment_links WHERE id = ?`, got.ID).Scan(&stored)
	if !stored.Equal(*d("107.22")) {
		t.Fatalf("amount column %s", stored)
	}

	next := got.Input
	next.Title = "Roastery box v2"
	next.Questions = next.Questions[:1]
	updated, err := s.svc.Update(ctx, s.actor.PlatformID, got.ID, next)
	if err != nil || updated.Revision != 2 || len(updated.Questions) != 1 || updated.Title != "Roastery box v2" {
		t.Fatalf("update %+v %v", updated, err)
	}
	if _, err := s.store.Save(ctx, got); err == nil {
		t.Fatal("save on a stale revision succeeded")
	}

	published, err := s.svc.Publish(ctx, s.actor.PlatformID, got.ID)
	if err != nil || !links.ValidShortCode(published.ShortCode) || published.PublishedAt == nil {
		t.Fatalf("publish %+v %v", published, err)
	}
	byCode, err := s.store.GetByShortCode(ctx, published.ShortCode)
	if err != nil || byCode.ID != got.ID {
		t.Fatalf("by code %v", err)
	}
	list, total, err := s.svc.List(ctx, s.actor.PlatformID, links.ListFilter{Status: links.StatusActive})
	if err != nil || total != 1 || list[0].ID != got.ID || len(list[0].LineItems) != 2 {
		t.Fatalf("list %d %v", total, err)
	}
	if _, err := s.svc.Get(ctx, s.actor.PlatformID+1, got.ID); links.CodeOf(err) != links.CodeNotFound {
		t.Fatalf("other platform: %v", err)
	}
	if _, err := s.svc.Get(ctx, s.actor.PlatformID, "not-a-uuid"); links.CodeOf(err) != links.CodeNotFound {
		t.Fatalf("bad id: %v", err)
	}
}

func TestIntegration_LifecycleAndDuplicate(t *testing.T) {
	s := newStack(t)
	ctx, pid := context.Background(), s.actor.PlatformID
	in := fullInput(s.webhook)
	l, _ := s.svc.Create(ctx, s.actor, in)
	pub, err := s.svc.Publish(ctx, pid, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	paused, err := s.svc.Pause(ctx, pid, l.ID)
	if err != nil || paused.Status != links.StatusPaused {
		t.Fatalf("pause %v", err)
	}
	resumed, err := s.svc.Publish(ctx, pid, l.ID)
	if err != nil || resumed.ShortCode != pub.ShortCode || !resumed.PublishedAt.Equal(*pub.PublishedAt) {
		t.Fatalf("resume %+v %v", resumed, err)
	}
	moved := resumed.Input
	moved.Currency = "EUR"
	if _, err := s.svc.Update(ctx, pid, l.ID, moved); links.CodeOf(err) != links.CodePublishedImmutable {
		t.Fatalf("currency change on a live link: %v", err)
	}
	dup, err := s.svc.Duplicate(ctx, s.actor, l.ID)
	if err != nil || dup.Status != links.StatusDraft || dup.ShortCode != "" || len(dup.Questions) != 3 {
		t.Fatalf("duplicate %+v %v", dup, err)
	}
	if _, err := s.svc.Archive(ctx, pid, l.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.svc.Delete(ctx, pid, l.ID); links.CodeOf(err) != links.CodeNotDeletable {
		t.Fatalf("delete archived: %v", err)
	}
	if err := s.svc.Delete(ctx, pid, dup.ID); err != nil {
		t.Fatal(err)
	}
	var n int64
	s.db.Raw(`SELECT count(*) FROM payment_link_questions WHERE link_id = ?`, dup.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("deleted draft left %d questions", n)
	}
}

func TestIntegration_ShortCodeCollisionIsRetried(t *testing.T) {
	codes := []string{"SAMECODE0001", "SAMECODE0001", "OTHERCODE002"}
	var i atomic.Int64
	s := newStack(t, links.WithShortCodes(func() (string, error) { return codes[i.Add(1)-1], nil }))
	ctx := context.Background()
	var got []string
	for range 2 {
		l, _ := s.svc.Create(ctx, s.actor, fullInput(s.webhook))
		p, err := s.svc.Publish(ctx, s.actor.PlatformID, l.ID)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, p.ShortCode)
	}
	if got[0] != "SAMECODE0001" || got[1] != "OTHERCODE002" {
		t.Fatalf("codes %v", got)
	}
}

func cardPay(key, email string) links.PayRequest {
	return links.PayRequest{
		IdempotencyKey: key, Method: links.MethodSpec{Method: fees.MethodCard},
		Customer: links.CustomerInput{Email: &email},
	}
}

func simpleLink(t *testing.T, s *stack, edit func(*links.Input)) links.Link {
	t.Helper()
	in := links.DefaultInput()
	in.Title, in.Currency, in.Amount, in.SuccessMessage = "Beans", "USD", d("25.00"), "Thanks"
	in.Methods = []links.MethodSpec{{Method: fees.MethodCard}}
	if edit != nil {
		edit(&in)
	}
	ctx := context.Background()
	l, err := s.svc.Create(ctx, s.actor, in)
	if err != nil {
		t.Fatal(err)
	}
	if l, err = s.svc.Publish(ctx, s.actor.PlatformID, l.ID); err != nil {
		t.Fatal(err)
	}
	return l
}

func TestIntegration_PayPersistsTheUseWithFeeRuleAndAnswers(t *testing.T) {
	s := newStack(t)
	ctx := context.Background()
	l := simpleLink(t, s, func(in *links.Input) {
		in.MultiUse = true
		in.Questions = []links.Question{{Key: "company", Label: "Company", Type: links.QuestionText, Required: true}}
	})
	req := cardPay("k1", "Ada@Example.test")
	req.Answers = map[string]string{"company": "Acme"}
	res, err := s.svc.Pay(ctx, l.ShortCode, req)
	if err != nil {
		t.Fatal(err)
	}
	var row struct {
		Status           string
		PaymentReference string
		FeeRuleID        uint
		FeeRuleVersion   int
		Fee              decimal.Decimal
		CustomerEmail    string
	}
	s.db.Raw(`SELECT status, payment_reference, fee_rule_id, fee_rule_version, fee, customer_email FROM payment_link_payments WHERE link_id = ?`, l.ID).Scan(&row)
	if row.Status != "created" || row.PaymentReference != res.PaymentReference || row.FeeRuleID != s.ruleID ||
		row.FeeRuleVersion != 1 || !row.Fee.Equal(*d("0.73")) || row.CustomerEmail != "ada@example.test" {
		t.Fatalf("stored use %+v", row)
	}
	if _, err := s.svc.Pay(ctx, l.ShortCode, cardPay("k2", "ada@example.test")); err != nil {
		t.Fatalf("returning customer asked again: %v", err)
	}
	if _, err := s.svc.Pay(ctx, l.ShortCode, cardPay("k3", "bob@example.test")); links.CodeOf(err) != links.CodeAnswerRequired {
		t.Fatalf("new customer: %v", err)
	}
	replay, err := s.svc.Pay(ctx, l.ShortCode, req)
	if err != nil || !replay.Replayed || replay.PaymentReference != res.PaymentReference || !replay.Fee.Equal(*d("0.73")) {
		t.Fatalf("replay %+v %v", replay, err)
	}
}

func TestIntegration_ExactlyOnePayerWinsTheLastUse(t *testing.T) {
	s := newStack(t)
	s.creator.delay = 5 * time.Millisecond
	ctx := context.Background()
	l := simpleLink(t, s, func(in *links.Input) { in.MultiUse, in.UseLimit = true, intp(3) })
	for i := range 2 {
		if _, err := s.svc.Pay(ctx, l.ShortCode, cardPay(fmt.Sprint("warm", i), "a@example.test")); err != nil {
			t.Fatal(err)
		}
	}
	const racers = 24
	var wg sync.WaitGroup
	errs := make([]error, racers)
	start := make(chan struct{})
	for i := range racers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = s.svc.Pay(ctx, l.ShortCode, cardPay(fmt.Sprint("race", i), "a@example.test"))
		}(i)
	}
	close(start)
	wg.Wait()
	wins := 0
	for _, err := range errs {
		switch {
		case err == nil:
			wins++
		case links.CodeOf(err) != links.CodeUseLimitReached:
			t.Fatalf("racer: %v", err)
		}
	}
	var uses, rows int64
	s.db.Raw(`SELECT uses_count FROM payment_links WHERE id = ?`, l.ID).Scan(&uses)
	s.db.Raw(`SELECT count(*) FROM payment_link_payments WHERE link_id = ?`, l.ID).Scan(&rows)
	if wins != 1 || uses != 3 || rows != 3 || s.creator.calls.Load() != 3 {
		t.Fatalf("wins %d uses %d rows %d calls %d; want 1, 3, 3, 3", wins, uses, rows, s.creator.calls.Load())
	}
}

func TestIntegration_ConcurrentRetriesOfOneKeyCreateOnePayment(t *testing.T) {
	s := newStack(t)
	s.creator.delay = 10 * time.Millisecond
	ctx := context.Background()
	l := simpleLink(t, s, func(in *links.Input) { in.MultiUse = true })
	var wg sync.WaitGroup
	errs := make([]error, 12)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = s.svc.Pay(ctx, l.ShortCode, cardPay("one", "a@example.test"))
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil && links.CodeOf(err) != links.CodePaymentInProgress {
			t.Fatalf("retry: %v", err)
		}
	}
	if s.creator.calls.Load() != 1 {
		t.Fatalf("creator called %d times", s.creator.calls.Load())
	}
}

func TestIntegration_FailedCreationReleasesTheUse(t *testing.T) {
	s := newStack(t)
	ctx := context.Background()
	l := simpleLink(t, s, nil)
	s.creator.fail.Store(true)
	if _, err := s.svc.Pay(ctx, l.ShortCode, cardPay("k", "a@example.test")); links.CodeOf(err) != links.CodePaymentCreationFailed {
		t.Fatalf("err %v", err)
	}
	var uses, rows int64
	s.db.Raw(`SELECT uses_count FROM payment_links WHERE id = ?`, l.ID).Scan(&uses)
	s.db.Raw(`SELECT count(*) FROM payment_link_payments WHERE link_id = ?`, l.ID).Scan(&rows)
	if uses != 0 || rows != 0 {
		t.Fatalf("uses %d rows %d after a failed creation", uses, rows)
	}
	s.creator.fail.Store(false)
	if _, err := s.svc.Pay(ctx, l.ShortCode, cardPay("k", "a@example.test")); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if _, err := s.svc.Pay(ctx, l.ShortCode, cardPay("k2", "a@example.test")); links.CodeOf(err) != links.CodeUseLimitReached {
		t.Fatalf("single use took a second payment: %v", err)
	}
}

func TestIntegration_DatabaseRefusesUsesBeyondTheLimit(t *testing.T) {
	s := newStack(t)
	single := simpleLink(t, s, nil)
	capped := simpleLink(t, s, func(in *links.Input) { in.MultiUse, in.UseLimit = true, intp(2) })
	for _, tc := range []struct {
		id   string
		uses int
	}{{single.ID, 2}, {capped.ID, 3}} {
		if err := s.db.Exec(`UPDATE payment_links SET uses_count = ? WHERE id = ?`, tc.uses, tc.id).Error; err == nil {
			t.Fatalf("uses_count %d accepted on %s", tc.uses, tc.id)
		}
	}
	if err := s.db.Exec(`UPDATE payment_links SET short_code = NULL WHERE id = ?`, single.ID).Error; err == nil {
		t.Fatal("an active link lost its short code")
	}
}
