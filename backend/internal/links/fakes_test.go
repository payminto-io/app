package links

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func decp(s string) *decimal.Decimal {
	d := dec(s)
	return &d
}
func intp(n int) *int       { return &n }
func strp(s string) *string { return &s }
func uintp(n uint) *uint    { return &n }

type ruleKey struct {
	method    fees.Method
	currency  string
	connector string
}

// fakeFees resolves connector-scoped rules first, then method defaults, and forbids surcharging UPI like the default policy.
type fakeFees struct {
	mu    sync.Mutex
	rules map[ruleKey]fees.Rule
	err   error
}

func newFakeFees() *fakeFees { return &fakeFees{rules: map[ruleKey]fees.Rule{}} }

func (f *fakeFees) add(m fees.Method, currency, connector string, percent string, bearer fees.FeeBearer) fees.Rule {
	f.mu.Lock()
	defer f.mu.Unlock()
	places, _ := fees.DefaultPrecision().MinorUnits(currency)
	r := fees.Rule{
		ID: uint(len(f.rules) + 1), Version: 1, Scope: fees.Scope{Method: m, Currency: currency},
		MinorUnits: places, Percent: dec(percent), FeeBearer: bearer, EffectiveFrom: time.Unix(0, 0),
	}
	if connector != "" {
		r.Connector = &connector
	}
	f.rules[ruleKey{m, currency, connector}] = r
	return r
}

func (f *fakeFees) Resolve(_ context.Context, q fees.Query) (fees.Rule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return fees.Rule{}, f.err
	}
	if r, ok := f.rules[ruleKey{q.Method, q.Currency, q.Connector}]; ok {
		return r, nil
	}
	if r, ok := f.rules[ruleKey{q.Method, q.Currency, ""}]; ok {
		return r, nil
	}
	return fees.Rule{}, fees.ErrNoRule
}

func (f *fakeFees) Preview(ctx context.Context, req fees.PreviewRequest) (fees.Breakdown, error) {
	r, err := f.Resolve(ctx, req.Query)
	if err != nil {
		return fees.Breakdown{}, err
	}
	if req.FeeBearer != nil {
		r.FeeBearer = *req.FeeBearer
	}
	if r.FeeBearer == fees.BearerCustomer && r.Method == fees.MethodUPI {
		return fees.Breakdown{}, fees.ErrSurchargeForbidden
	}
	b := fees.Compute(r, req.Amount)
	if b.Fee.Add(b.Tax).GreaterThan(req.Amount) {
		return fees.Breakdown{}, fees.ErrFeeExceedsAmount
	}
	return b, nil
}

// fakeCreator offers connectors per method and is idempotent on LinkPaymentID, like the contract requires.
// err fails before creating; errAfter creates and then fails (the ambiguous case); findErr fails lookups.
type fakeCreator struct {
	mu         sync.Mutex
	connectors map[string][]string
	created    []PaymentRequest
	byID       map[string]CreatedPayment
	err        error
	errAfter   error
	findErr    error
	calls      atomic.Int64
	delay      time.Duration
	cancelled  map[string]bool
	fenced     map[string]bool
	paid       map[string]bool
	// leaky makes the fence a no-op, to exercise the cancel path for a creator that cannot fence.
	leaky bool
}

func newFakeCreator() *fakeCreator {
	return &fakeCreator{connectors: map[string][]string{}, byID: map[string]CreatedPayment{},
		cancelled: map[string]bool{}, fenced: map[string]bool{}, paid: map[string]bool{}}
}

func (c *fakeCreator) offer(m MethodSpec, conns ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connectors[m.String()] = conns
}

func (c *fakeCreator) Connectors(_ context.Context, _ Environment, _ string, m MethodSpec) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectors[m.String()], nil
}

func (c *fakeCreator) CreatePayment(ctx context.Context, req PaymentRequest) (CreatedPayment, error) {
	c.calls.Add(1)
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return CreatedPayment{}, c.err
	}
	if got, ok := c.byID[req.LinkPaymentID]; ok {
		return got, nil
	}
	if c.fenced[req.LinkPaymentID] {
		return CreatedPayment{}, fmt.Errorf("%w: reference is fenced", ErrNotCreated)
	}
	if err := ctx.Err(); err != nil {
		return CreatedPayment{}, err
	}
	created := CreatedPayment{Reference: "pay_" + req.LinkPaymentID, CheckoutURL: "https://checkout.test/pay/pay_" + req.LinkPaymentID}
	c.byID[req.LinkPaymentID] = created
	c.created = append(c.created, req)
	if c.errAfter != nil {
		return CreatedPayment{}, c.errAfter
	}
	return created, nil
}

func (c *fakeCreator) FencePayment(_ context.Context, req PaymentRequest) (CreatedPayment, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.findErr != nil {
		return CreatedPayment{}, false, c.findErr
	}
	if got, ok := c.byID[req.LinkPaymentID]; ok && !c.cancelled[req.LinkPaymentID] {
		return got, true, nil
	}
	if !c.leaky {
		c.fenced[req.LinkPaymentID] = true
	}
	return CreatedPayment{}, false, nil
}

func (c *fakeCreator) CancelPayment(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancelled[id] = true
	return nil
}

func (c *fakeCreator) OpenPayments(_ context.Context, ids []string) (map[string]bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]bool{}
	for _, id := range ids {
		_, made := c.byID[id]
		out[id] = made && !c.paid[id] && !c.cancelled[id]
	}
	return out, nil
}

func (c *fakeCreator) markPaid(ref string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, got := range c.byID {
		if got.Reference == ref {
			c.paid[id] = true
		}
	}
}

// livePayments counts payments created and not cancelled.
func (c *fakeCreator) livePayments() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, r := range c.created {
		if !c.cancelled[r.LinkPaymentID] {
			n++
		}
	}
	return n
}

func (c *fakeCreator) payments() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.created)
}

type verifier map[string]bool

func (v verifier) Verified(_ context.Context, _ uint, o SettlementOverride) (bool, error) {
	return v[o.DestinationID], nil
}

var (
	card     = MethodSpec{Method: fees.MethodCard}
	upi      = MethodSpec{Method: fees.MethodUPI}
	usdcSol  = MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDC"}
	merchant = Actor{MemberID: 7, PlatformID: 3}
	errBoom  = errors.New("boom")
)

type fixture struct {
	t       *testing.T
	store   *MemStore
	fees    *fakeFees
	creator *fakeCreator
	svc     *Service
	now     time.Time
}

func newFixture(t *testing.T, opts ...Option) *fixture {
	t.Helper()
	f := &fixture{t: t, store: NewMemStore(), fees: newFakeFees(), creator: newFakeCreator(), now: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)}
	f.fees.add(fees.MethodCard, "USD", "", "2.9", fees.BearerMerchant)
	f.fees.add(fees.MethodUPI, "USD", "", "1", fees.BearerMerchant)
	f.fees.add(fees.MethodCrypto, "USDC", "", "1", fees.BearerMerchant)
	f.creator.offer(card, "mockcard")
	f.creator.offer(upi, "payvang")
	f.creator.offer(usdcSol, "")
	f.store.AddWebhook(merchant.PlatformID, 11)
	f.store.SetMerchantName(merchant.PlatformID, "Acme Coffee")
	base := []Option{WithCheckoutBaseURL("https://checkout.test/"), WithClock(func() time.Time { return f.now }),
		WithDestinations(verifier{"dest_ok": true})}
	f.svc = NewService(f.store, f.fees, f.creator, append(base, opts...)...)
	return f
}

// validInput is a publishable fixed-amount card link.
func validInput() Input {
	in := DefaultInput()
	in.Title = "Coffee beans"
	in.Amount = decp("25.00")
	in.Currency = "USD"
	in.Methods = []MethodSpec{card}
	in.SuccessMessage = "Thanks!"
	return in
}

func (f *fixture) create(in Input) Link {
	f.t.Helper()
	l, err := f.svc.Create(context.Background(), merchant, in)
	if err != nil {
		f.t.Fatalf("create: %v", err)
	}
	return l
}

func (f *fixture) published(in Input) Link {
	f.t.Helper()
	l := f.create(in)
	p, err := f.svc.Publish(context.Background(), merchant.PlatformID, l.ID)
	if err != nil {
		f.t.Fatalf("publish: %v", err)
	}
	return p
}

func hasCode(err error, code Code) bool {
	for _, c := range Codes(err) {
		if c == code {
			return true
		}
	}
	return false
}
