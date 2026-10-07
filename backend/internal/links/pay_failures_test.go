package links

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
)

func (f *fixture) uses(id string) int {
	f.t.Helper()
	l, err := f.svc.Get(context.Background(), merchant.PlatformID, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return l.UsesCount
}

func TestDefinitiveRefusalReleasesTheUse(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	f.creator.err = fmt.Errorf("%w: no wallet for this chain", ErrNotCreated)
	if _, err := f.pay(l.ShortCode, payReq("k1")); CodeOf(err) != CodePaymentCreationFailed {
		t.Fatalf("err %v", err)
	}
	if f.uses(l.ID) != 0 || len(f.store.Payments(l.ID)) != 0 {
		t.Fatal("a definitive refusal kept the use")
	}
	f.creator.err = nil
	if _, err := f.pay(l.ShortCode, payReq("k1")); err != nil {
		t.Fatalf("retry after a definitive refusal: %v", err)
	}
}

func TestAmbiguousFailureKeepsTheUseAndTheRetryReturnsThePayment(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	f.creator.errAfter = errBoom
	_, err := f.pay(l.ShortCode, payReq("k1"))
	var e *Error
	if !errors.As(err, &e) || e.Code != CodePaymentInProgress || e.RetryAfter <= 0 || e.RetryAfter > DefaultLease {
		t.Fatalf("ambiguous failure: %v", err)
	}
	if f.uses(l.ID) != 1 {
		t.Fatal("an ambiguous failure gave the use back")
	}
	f.creator.errAfter = nil
	if _, err := f.pay(l.ShortCode, payReq("k1")); CodeOf(err) != CodePaymentInProgress {
		t.Fatalf("retry inside the lease: %v", err)
	}
	if _, err := f.pay(l.ShortCode, payReq("k2")); CodeOf(err) != CodeUseLimitReached {
		t.Fatalf("another payer took the held use: %v", err)
	}
	f.now = f.now.Add(DefaultLease + time.Second)
	res, err := f.pay(l.ShortCode, payReq("k1"))
	if err != nil || !res.Replayed || res.PaymentReference == "" {
		t.Fatalf("retry after the lease: %+v %v", res, err)
	}
	if f.creator.payments() != 1 {
		t.Fatalf("%d payments created for one key", f.creator.payments())
	}
}

func TestAmbiguousFailureWithNothingCreatedIsRetriedAfterTheLease(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	f.creator.err = errBoom
	if _, err := f.pay(l.ShortCode, payReq("k1")); CodeOf(err) != CodePaymentInProgress {
		t.Fatalf("err %v", err)
	}
	f.creator.err = nil
	f.now = f.now.Add(DefaultLease + time.Second)
	res, err := f.pay(l.ShortCode, payReq("k1"))
	if err != nil || res.Replayed {
		t.Fatalf("fresh attempt after release: %+v %v", res, err)
	}
	if f.creator.payments() != 1 || f.uses(l.ID) != 1 {
		t.Fatalf("payments %d uses %d", f.creator.payments(), f.uses(l.ID))
	}
}

func TestFailedCompleteIsResolvedFromThePayment(t *testing.T) {
	f := newFixture(t)
	l := f.published(validInput())
	f.store.FailNextComplete(errBoom)
	if _, err := f.pay(l.ShortCode, payReq("k1")); CodeOf(err) != CodePaymentInProgress {
		t.Fatalf("err %v", err)
	}
	f.now = f.now.Add(DefaultLease + time.Second)
	completed, released, err := f.svc.ResolveExpired(context.Background(), 10)
	if err != nil || completed != 1 || released != 0 {
		t.Fatalf("resolver %d %d %v", completed, released, err)
	}
	res, err := f.pay(l.ShortCode, payReq("k1"))
	if err != nil || !res.Replayed || f.creator.payments() != 1 {
		t.Fatalf("retry after resolution: %+v %v", res, err)
	}
}

// reserveOnly leaves a pending use as a process that died between Reserve and Complete would.
func (f *fixture) reserveOnly(l Link, key string) LinkPayment {
	f.t.Helper()
	p, err := f.svc.quote(context.Background(), l, payReq(key))
	if err != nil {
		f.t.Fatal(err)
	}
	p.IdempotencyKey, p.RequestHash = key, requestHash(payReq(key))
	p.ReservedUntil, p.OpenUntil = f.now.Add(DefaultLease), f.now.Add(DefaultLease)
	r, _, err := f.store.Reserve(context.Background(), p, f.now, DefaultLimits)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}

func TestResolverSettlesReservationsLeftByACrash(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.MultiUse = true
	l := f.published(in)
	made := f.reserveOnly(l, "made")
	f.reserveOnly(l, "never")
	unknown := f.reserveOnly(l, "unknown")
	if _, err := f.creator.CreatePayment(context.Background(), PaymentRequest{LinkPaymentID: made.ID}); err != nil {
		t.Fatal(err)
	}
	if c, r, err := f.svc.ResolveExpired(context.Background(), 10); c+r != 0 || err != nil {
		t.Fatalf("resolver touched live leases: %d %d %v", c, r, err)
	}
	f.now = f.now.Add(DefaultLease + time.Second)
	f.creator.findErr = errBoom
	if c, r, err := f.svc.ResolveExpired(context.Background(), 10); c+r != 0 || err == nil {
		t.Fatalf("lookup failure settled something: %d %d %v", c, r, err)
	}
	f.creator.findErr = nil
	completed, released, err := f.svc.ResolveExpired(context.Background(), 10)
	if err != nil || completed != 1 || released != 2 {
		t.Fatalf("resolver %d %d %v", completed, released, err)
	}
	if f.uses(l.ID) != 1 {
		t.Fatalf("uses %d, want only the created payment", f.uses(l.ID))
	}
	_ = unknown
}

func TestCreatorCallIsBoundedInsideTheLease(t *testing.T) {
	f := newFixture(t, WithLease(40*time.Millisecond))
	f.creator.delay = 50 * time.Millisecond
	l := f.published(validInput())
	f.creator.err = nil
	start := time.Now()
	_, _ = f.pay(l.ShortCode, payReq("k1"))
	if f.creator.payments() != 0 {
		t.Fatal("the creator wrote after its deadline")
	}
	if time.Since(start) > time.Second {
		t.Fatal("pay did not return")
	}
}

func TestOpenPaymentCaps(t *testing.T) {
	f := newFixture(t, WithLimits(ReserveLimits{MaxOpen: 4, MaxOpenPerClient: 2}))
	in := validInput()
	in.MultiUse = true
	l := f.published(in)
	pay := func(key, ip string) error {
		req := payReq(key)
		req.ClientIP = ip
		_, err := f.pay(l.ShortCode, req)
		return err
	}
	for i, tc := range []struct {
		ip   string
		want Code
	}{{"1.1.1.1", ""}, {"1.1.1.1", ""}, {"1.1.1.1", CodeOpenPaymentsLimit}, {"2.2.2.2", ""}, {"3.3.3.3", ""}, {"4.4.4.4", CodeOpenPaymentsLimit}} {
		if err := pay(fmt.Sprint("k", i), tc.ip); CodeOf(err) != tc.want {
			t.Fatalf("payment %d from %s: %v, want %q", i, tc.ip, err, tc.want)
		}
	}
	f.now = f.now.Add(time.Duration(in.QuoteExpirySeconds+1) * time.Second)
	if err := pay("later", "1.1.1.1"); err != nil {
		t.Fatalf("caps did not free up after the payments expired: %v", err)
	}
}

func TestPayRefusesALinkOfTheOtherEnvironment(t *testing.T) {
	live, _ := environment.NewGuard(environment.Live)
	f := newFixture(t, WithGuard(live))
	l := f.published(validInput())
	if l.Environment != EnvLive {
		t.Fatalf("link tagged %s under a live guard", l.Environment)
	}
	testSvc := NewService(f.store, f.fees, f.creator, WithClock(func() time.Time { return f.now }))
	if _, err := testSvc.Pay(context.Background(), l.ShortCode, payReq("k1")); CodeOf(err) != CodeEnvironmentMismatch {
		t.Fatalf("a test process paid a live link: %v", err)
	}
	if len(f.store.Payments(l.ID)) != 0 {
		t.Fatal("a refused environment reserved a use")
	}
}

func TestUseLimitCannotDropBelowUsesTaken(t *testing.T) {
	f := newFixture(t, WithLimits(ReserveLimits{}))
	in := validInput()
	in.MultiUse, in.UseLimit = true, intp(5)
	l := f.published(in)
	for i := range 3 {
		if _, err := f.pay(l.ShortCode, payReq(fmt.Sprint("k", i))); err != nil {
			t.Fatal(err)
		}
	}
	l, _ = f.svc.Get(context.Background(), merchant.PlatformID, l.ID)
	for _, tc := range []struct {
		field string
		edit  func(*Input)
	}{
		{"use_limit", func(in *Input) { in.UseLimit = intp(2) }},
		{"expires_after_payments", func(in *Input) { in.ExpiresAfterPayments = intp(1) }},
		{"multi_use", func(in *Input) { in.MultiUse, in.UseLimit = false, nil }},
	} {
		next := l.Input
		tc.edit(&next)
		_, err := f.svc.Update(context.Background(), merchant.PlatformID, l.ID, l.Revision, next)
		if e, ok := err.(*Error); !ok || e.Code != CodeUseLimitBelowUses || e.Field != tc.field {
			t.Fatalf("%s: %v", tc.field, err)
		}
	}
	next := l.Input
	next.UseLimit = intp(3)
	if _, err := f.svc.Update(context.Background(), merchant.PlatformID, l.ID, l.Revision, next); err != nil {
		t.Fatalf("limit equal to uses: %v", err)
	}
}

func TestTotalsBeyondTheColumnAreRefused(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.AmountMode, in.Amount = AmountLineItems, nil
	in.LineItems = []LineItem{{Name: "a", Quantity: 100000, UnitPrice: dec("99999999999999999999")}}
	if _, err := f.svc.Create(context.Background(), merchant, in); !hasCode(err, CodeAmountInvalid) {
		t.Fatalf("overflowing line total: %v", err)
	}
	in = validInput()
	in.AmountMode, in.Amount, in.FeeBearer = AmountCustomer, nil, fees.BearerCustomer
	l := f.published(in)
	req := payReq("k1")
	req.Amount = decp("99999999999999999999")
	if _, err := f.pay(l.ShortCode, req); CodeOf(err) != CodeAmountInvalid {
		t.Fatalf("surcharged total above the column: %v", err)
	}
}

func TestMerchantFeePreviewPerMethod(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.Methods = []MethodSpec{card, usdcSol, {Method: fees.MethodBank}}
	in.FeeBearer = fees.BearerMerchant
	l := f.create(in)
	got := f.svc.FeePreview(context.Background(), l)
	if len(got) != 3 {
		t.Fatalf("previews %+v", got)
	}
	c := got[0]
	if *c.Connector != "mockcard" || *c.RuleID == 0 || *c.RuleVersion != 1 || !c.Fee.Equal(dec("0.73")) || !c.MerchantNet.Equal(dec("24.27")) || c.Unavailable != nil {
		t.Fatalf("card %+v", c)
	}
	if u := got[1]; u.RuleID == nil || u.Fee != nil || u.FeeCurrency != "USDC" {
		t.Fatalf("cross-currency crypto %+v", u)
	}
	if b := got[2]; b.Unavailable == nil || *b.Unavailable != CodeMethodNoConnector {
		t.Fatalf("bank %+v", b)
	}
}

func TestOpenCapsIgnorePaidPayments(t *testing.T) {
	f := newFixture(t, WithLimits(ReserveLimits{MaxOpen: 100, MaxOpenPerClient: 3}))
	in := validInput()
	in.MultiUse = true
	l := f.published(in)
	for i := range 3 {
		req := payReq(fmt.Sprint("k", i))
		req.ClientIP = "198.51.100.4"
		res, err := f.pay(l.ShortCode, req)
		if err != nil {
			t.Fatal(err)
		}
		f.creator.markPaid(res.PaymentReference)
	}
	req := payReq("k4")
	req.ClientIP = "198.51.100.4"
	if _, err := f.pay(l.ShortCode, req); err != nil {
		t.Fatalf("a fourth payer after three paid: %v", err)
	}
}

func TestIPv6ClientsAreGroupedBySlash64(t *testing.T) {
	if clientKey("2001:db8:1:2::1") != clientKey("2001:db8:1:2:ffff::9") {
		t.Fatal("two addresses in one /64 are different clients")
	}
	if clientKey("2001:db8:1:2::1") == clientKey("2001:db8:1:3::1") {
		t.Fatal("two /64s are one client")
	}
	if clientKey("::ffff:198.51.100.4") != clientKey("198.51.100.4") || clientKey("198.51.100.4") == clientKey("198.51.100.5") {
		t.Fatal("IPv4 keys are not per address")
	}
	f := newFixture(t, WithLimits(ReserveLimits{MaxOpenPerClient: 2}))
	in := validInput()
	in.MultiUse = true
	l := f.published(in)
	for i, ip := range []string{"2001:db8::1", "2001:db8::2", "2001:db8::3"} {
		req := payReq(fmt.Sprint("v6-", i))
		req.ClientIP = ip
		_, err := f.pay(l.ShortCode, req)
		if want := i == 2; (CodeOf(err) == CodeOpenPaymentsLimit) != want {
			t.Fatalf("payer %d from %s: %v", i, ip, err)
		}
	}
}
