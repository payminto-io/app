package links

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
)

func payReq(key string) PayRequest {
	return PayRequest{IdempotencyKey: key, Method: card, Customer: CustomerInput{Email: strp("ada@example.test")}}
}

func (f *fixture) pay(code string, req PayRequest) (PayResult, error) {
	return f.svc.Pay(context.Background(), code, req)
}

func TestPayCreatesAPaymentWithTheServerAmount(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.ReferenceID = "order-42"
	in.Metadata = map[string]string{"crm": "x"}
	in.SuccessMode, in.SuccessURL = SuccessRedirect, "https://shop.test/thanks?src=link"
	l := f.published(in)
	res, err := f.pay(l.ShortCode, payReq("k1"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Amount.Equal(dec("25")) || !res.CustomerTotal.Equal(dec("25")) || res.Fee == nil || !res.Fee.Equal(dec("0.73")) {
		t.Fatalf("result %+v", res)
	}
	if res.SuccessRedirectURL != "https://shop.test/thanks?reference_id=order-42&src=link" {
		t.Fatalf("redirect %s", res.SuccessRedirectURL)
	}
	got := f.creator.created[0]
	if got.MemberID != merchant.MemberID || got.PlatformID != merchant.PlatformID || got.Connector != "mockcard" ||
		got.ReferenceID != "order-42" || got.Metadata["crm"] != "x" || got.CustomerEmail != "ada@example.test" ||
		got.FeeRuleID == 0 || got.Environment != EnvTest || got.LinkPaymentID == "" {
		t.Fatalf("payment request %+v", got)
	}
	if after, _ := f.svc.Get(context.Background(), merchant.PlatformID, l.ID); after.UsesCount != 1 {
		t.Fatalf("uses %d", after.UsesCount)
	}
}

func TestPayChargesTheSurchargedTotal(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.FeeBearer = fees.BearerCustomer
	l := f.published(in)
	res, err := f.pay(l.ShortCode, payReq("k1"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.CustomerTotal.Equal(dec("25.73")) || f.creator.created[0].FeeBearer != fees.BearerCustomer || !f.creator.created[0].CustomerTotal.Equal(dec("25.73")) {
		t.Fatalf("surcharge %+v", res)
	}
}

func TestPayCrossCurrencyMethodCarriesTheRuleButNoInventedFee(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.Methods = []MethodSpec{usdcSol}
	l := f.published(in)
	req := payReq("k1")
	req.Method = MethodSpec{Method: "CRYPTO", Chain: "sol", Asset: "usdc"}
	res, err := f.pay(l.ShortCode, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fee != nil || res.Tax != nil || !res.CustomerTotal.Equal(dec("25")) || f.creator.created[0].FeeRuleID == 0 {
		t.Fatalf("cross-currency %+v", res)
	}
}

func TestPayCustomerAmountRespectsBounds(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.AmountMode, in.Amount, in.AmountMin, in.AmountMax = AmountCustomer, nil, decp("5"), decp("50")
	l := f.published(in)
	req := payReq("k1")
	req.Amount = decp("50.01")
	if _, err := f.pay(l.ShortCode, req); CodeOf(err) != CodeAmountOutOfRange {
		t.Fatalf("over max: %v", err)
	}
	req.Amount = decp("12.34")
	res, err := f.pay(l.ShortCode, req)
	if err != nil || !res.Amount.Equal(dec("12.34")) {
		t.Fatalf("in range: %+v %v", res, err)
	}
}

func TestPayRefusesBadPayerInput(t *testing.T) {
	cases := []struct {
		name  string
		code  Code
		field string
		link  func(*Input)
		req   func(*PayRequest)
	}{
		{"no idempotency key", CodeIdempotencyKeyRequired, "Idempotency-Key", nil, func(r *PayRequest) { r.IdempotencyKey = " " }},
		{"method not on the link", CodeMethodNotEnabled, "method", nil, func(r *PayRequest) { r.Method = upi }},
		{"absurd amount", CodeAmountInvalid, "amount", nil, func(r *PayRequest) { r.Amount = decp("1e999999999") }},
		{"client amount on a fixed link", CodeAmountNotAllowed, "amount", nil, func(r *PayRequest) { r.Amount = decp("0.01") }},
		{"hidden field sent", CodeCustomerFieldHidden, "customer.phone", nil, func(r *PayRequest) { r.Customer.Phone = strp("+14155550123") }},
		{"required field missing", CodeCustomerFieldRequired, "customer.name", func(in *Input) { in.CustomerFields.Name.Mode = FieldRequired }, nil},
		{"bad email", CodeCustomerEmailInvalid, "customer.email", nil, func(r *PayRequest) { r.Customer.Email = strp("ada at example") }},
		{"bad phone", CodeCustomerPhoneInvalid, "customer.phone", func(in *Input) { in.CustomerFields.Phone.Mode = FieldOptional },
			func(r *PayRequest) { r.Customer.Phone = strp("555-0123") }},
		{"billing required", CodeAddressRequired, "billing_address", func(in *Input) { in.BillingRequired = true }, nil},
		{"shipping invalid", CodeAddressInvalid, "shipping_address.country", nil, func(r *PayRequest) {
			r.ShippingAddress = &Address{Line1: "1 Main", City: "Pune", PostalCode: "411001", Country: "India"}
		}},
		{"required question unanswered", CodeAnswerRequired, "answers.size", nil, func(r *PayRequest) { r.Answers = map[string]string{"size": " "} }},
		{"unknown question", CodeAnswerUnknownQuestion, "answers.colour", nil, func(r *PayRequest) { r.Answers = map[string]string{"size": "S", "colour": "red"} }},
		{"option not offered", CodeAnswerInvalid, "answers.size", nil, func(r *PayRequest) { r.Answers = map[string]string{"size": "XXL"} }},
		{"checkbox not boolean", CodeAnswerInvalid, "answers.terms", nil, func(r *PayRequest) { r.Answers = map[string]string{"size": "S", "terms": "yes"} }},
		{"required checkbox unchecked", CodeAnswerRequired, "answers.agree", func(in *Input) {
			in.Questions = append(in.Questions, Question{Key: "agree", Label: "I agree", Type: QuestionCheckbox, Required: true, PerOrder: true})
		}, func(r *PayRequest) { r.Answers = map[string]string{"size": "S", "agree": "false"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			in := validInput()
			in.Questions = []Question{
				{Key: "size", Label: "Size", Type: QuestionSelect, Options: []string{"S", "M"}, Required: true, PerOrder: true},
				{Key: "terms", Label: "Terms", Type: QuestionCheckbox},
			}
			if tc.link != nil {
				tc.link(&in)
			}
			l := f.published(in)
			req := payReq("k1")
			req.Answers = map[string]string{"size": "S"}
			if tc.req != nil {
				tc.req(&req)
			}
			_, err := f.pay(l.ShortCode, req)
			found := false
			for _, e := range allErrs(err) {
				found = found || (e.Code == tc.code && e.Field == tc.field)
			}
			if !found {
				t.Fatalf("err %v (%v), want %s on %s", err, Codes(err), tc.code, tc.field)
			}
			if f.creator.calls.Load() != 0 || len(f.store.Payments(l.ID)) != 0 {
				t.Fatal("a refused payment reached the creator or the store")
			}
		})
	}
}

func TestPayStoresAnswersAndPrefills(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.CustomerFields.Name = FieldRule{Mode: FieldHidden, Prefill: "Ada Lovelace"}
	in.Questions = []Question{{Key: "note", Label: "Note", Type: QuestionText}}
	l := f.published(in)
	req := payReq("k1")
	req.Answers = map[string]string{"note": "  ring twice "}
	if _, err := f.pay(l.ShortCode, req); err != nil {
		t.Fatal(err)
	}
	p := f.store.Payments(l.ID)[0]
	if p.CustomerName != "Ada Lovelace" || p.Answers[0].Value != "ring twice" || p.Answers[0].QuestionLabel != "Note" {
		t.Fatalf("stored %+v", p)
	}
}

func TestRequiredQuestionsAreAskedEveryTimeAndRevealNoPastPayer(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.MultiUse = true
	in.Questions = []Question{{Key: "company", Label: "Company", Type: QuestionText, Required: true, PerOrder: false}}
	l := f.published(in)
	first := payReq("k1")
	first.Answers = map[string]string{"company": "Acme"}
	if _, err := f.pay(l.ShortCode, first); err != nil {
		t.Fatal(err)
	}
	probe := func(email string) []Code {
		req := payReq("probe-" + email)
		req.Customer.Email = strp(email)
		req.Method = upi
		_, err := f.pay(l.ShortCode, req)
		return Codes(err)
	}
	past, stranger := probe("ada@example.test"), probe("bob@example.test")
	if fmt.Sprint(past) != fmt.Sprint(stranger) || !slices.Contains(past, CodeAnswerRequired) {
		t.Fatalf("a past payer's email changes the refusal: %v vs %v", past, stranger)
	}
	if _, err := f.pay(l.ShortCode, payReq("k2")); CodeOf(err) != CodeAnswerRequired {
		t.Fatalf("a claimed email skipped a required question: %v", err)
	}
}

func TestPayIsIdempotentPerKey(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.MultiUse = true
	l := f.published(in)
	first, err := f.pay(l.ShortCode, payReq("same"))
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.pay(l.ShortCode, payReq("same"))
	if err != nil || !again.Replayed || again.PaymentReference != first.PaymentReference {
		t.Fatalf("replay %+v %v", again, err)
	}
	if f.creator.calls.Load() != 1 {
		t.Fatalf("creator called %d times", f.creator.calls.Load())
	}
	other := payReq("same")
	other.Customer.Email = strp("eve@example.test")
	if _, err := f.pay(l.ShortCode, other); CodeOf(err) != CodeIdempotencyKeyReused {
		t.Fatalf("key reused with another body: %v", err)
	}
	if after, _ := f.svc.Get(context.Background(), merchant.PlatformID, l.ID); after.UsesCount != 1 {
		t.Fatalf("replay took a use: %d", after.UsesCount)
	}
}

func TestReplayStillAnswersAfterTheLinkCloses(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	first, _ := f.pay(l.ShortCode, payReq("k1"))
	f.svc.Pause(context.Background(), merchant.PlatformID, l.ID)
	again, err := f.pay(l.ShortCode, payReq("k1"))
	if err != nil || again.PaymentReference != first.PaymentReference {
		t.Fatalf("replay after pause: %v", err)
	}
}

func TestPayRefusesClosedLinks(t *testing.T) {
	f := newFixture(t)
	ctx, pid := context.Background(), merchant.PlatformID

	paused := f.linkIn(StatusPaused)
	if _, err := f.pay(paused.ShortCode, payReq("k")); CodeOf(err) != CodePaused {
		t.Fatalf("paused: %v", err)
	}
	archived := f.linkIn(StatusActive)
	f.svc.Archive(ctx, pid, archived.ID)
	if _, err := f.pay(archived.ShortCode, payReq("k")); CodeOf(err) != CodeArchived {
		t.Fatalf("archived: %v", err)
	}
	in := validInput()
	exp := f.now.Add(time.Minute)
	in.ExpiresAt = &exp
	expiring := f.published(in)
	f.now = exp.Add(time.Second)
	if _, err := f.pay(expiring.ShortCode, payReq("k")); CodeOf(err) != CodeExpired {
		t.Fatalf("expired: %v", err)
	}
	if _, err := f.pay("Zzzzzzzzzzzz", payReq("k")); CodeOf(err) != CodeNotFound {
		t.Fatalf("unknown: %v", err)
	}
	if f.creator.calls.Load() != 0 {
		t.Fatal("closed link reached the creator")
	}
}

func TestSingleUseAndUseLimits(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*Input)
		takes int
	}{
		{"single use", func(*Input) {}, 1},
		{"use limit", func(in *Input) { in.MultiUse, in.UseLimit = true, intp(3) }, 3},
		{"expires after payments", func(in *Input) { in.MultiUse, in.ExpiresAfterPayments = true, intp(2) }, 2},
		{"lower of both", func(in *Input) { in.MultiUse, in.UseLimit, in.ExpiresAfterPayments = true, intp(4), intp(2) }, 2},
		{"unlimited", func(in *Input) { in.MultiUse = true }, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, WithLimits(ReserveLimits{}))
			in := validInput()
			tc.edit(&in)
			l := f.published(in)
			ok := 0
			for i := range 10 {
				if _, err := f.pay(l.ShortCode, payReq(fmt.Sprint("k", i))); err == nil {
					ok++
				} else if CodeOf(err) != CodeUseLimitReached {
					t.Fatalf("payment %d: %v", i, err)
				}
			}
			if ok != tc.takes {
				t.Fatalf("took %d payments, want %d", ok, tc.takes)
			}
		})
	}
}

func TestExactlyOnePayerWinsTheLastUse(t *testing.T) {
	f := newFixture(t)
	f.creator.delay = 2 * time.Millisecond
	in := validInput()
	in.MultiUse, in.UseLimit = true, intp(3)
	l := f.published(in)
	for i := range 2 {
		if _, err := f.pay(l.ShortCode, payReq(fmt.Sprint("warm", i))); err != nil {
			t.Fatal(err)
		}
	}
	const racers = 32
	var wg sync.WaitGroup
	results := make([]error, racers)
	start := make(chan struct{})
	for i := range racers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, results[i] = f.pay(l.ShortCode, payReq(fmt.Sprint("race", i)))
		}(i)
	}
	close(start)
	wg.Wait()
	wins := 0
	for _, err := range results {
		switch CodeOf(err) {
		case "":
			if err == nil {
				wins++
			} else {
				t.Fatalf("unexpected error %v", err)
			}
		case CodeUseLimitReached:
		default:
			t.Fatalf("unexpected error %v", err)
		}
	}
	if wins != 1 || f.creator.calls.Load() != 3 {
		t.Fatalf("%d winners, %d payments created; want 1 and 3", wins, f.creator.calls.Load())
	}
}

func TestConcurrentRetriesOfOneKeyCreateOnePayment(t *testing.T) {
	f := newFixture(t)
	f.creator.delay = 5 * time.Millisecond
	in := validInput()
	in.MultiUse = true
	l := f.published(in)
	var wg sync.WaitGroup
	errs := make([]error, 16)
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, errs[i] = f.pay(l.ShortCode, payReq("one")) }(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil && CodeOf(err) != CodePaymentInProgress {
			t.Fatalf("retry: %v", err)
		}
	}
	if f.creator.calls.Load() != 1 {
		t.Fatalf("creator called %d times for one key", f.creator.calls.Load())
	}
}

func TestCreatorRefusalPassesThrough(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	f.creator.err = newErr(CodeMethodUnavailable, "method", "processor refuses")
	if _, err := f.pay(l.ShortCode, payReq("k1")); CodeOf(err) != CodeMethodUnavailable {
		t.Fatalf("err %v", err)
	}
}

func TestPayRefusesAMethodThatLostItsRule(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	f.creator.offer(card)
	if _, err := f.pay(l.ShortCode, payReq("k1")); CodeOf(err) != CodeMethodUnavailable {
		t.Fatalf("err %v", err)
	}
}
