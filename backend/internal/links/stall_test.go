package links

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stallingCreator ignores ctx and blocks inside CreatePayment until released, like Payminto's payment service.
type stallingCreator struct {
	*fakeCreator
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *stallingCreator) CreatePayment(ctx context.Context, req PaymentRequest) (CreatedPayment, error) {
	stall := false
	c.once.Do(func() { stall = true })
	if stall {
		close(c.entered)
		<-c.release
		return c.fakeCreator.CreatePayment(context.WithoutCancel(ctx), req)
	}
	return c.fakeCreator.CreatePayment(ctx, req)
}

func TestStalledCreatorAfterReleaseNeverLeavesASecondLivePayment(t *testing.T) {
	f := newFixture(t)
	var clock atomic.Int64
	clock.Store(f.now.UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	sc := &stallingCreator{fakeCreator: f.creator, entered: make(chan struct{}), release: make(chan struct{})}
	svc := NewService(f.store, f.fees, sc, WithClock(now), WithLimits(ReserveLimits{}))
	l := f.published(validInput())

	firstErr := make(chan error, 1)
	go func() {
		_, err := svc.Pay(context.Background(), l.ShortCode, payReq("k1"))
		firstErr <- err
	}()
	<-sc.entered
	clock.Add(int64(DefaultLease + time.Second))
	if _, released, err := svc.ResolveExpired(context.Background(), 10); err != nil || released != 1 {
		t.Fatalf("resolver released %d: %v", released, err)
	}
	close(sc.release)
	if err := <-firstErr; err == nil {
		t.Fatal("the stalled call reported success for a use that was released")
	}
	res, err := svc.Pay(context.Background(), l.ShortCode, payReq("k1"))
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if live := f.creator.livePayments(); live != 1 {
		t.Fatalf("%d live payments for one key on a single-use link (retry got %s)", live, res.PaymentReference)
	}
	if f.uses(l.ID) != 1 {
		t.Fatalf("uses %d", f.uses(l.ID))
	}
}

func TestCreatorThatCannotFenceHasItsLatePaymentCancelled(t *testing.T) {
	f := newFixture(t)
	f.creator.leaky = true
	var clock atomic.Int64
	clock.Store(f.now.UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	sc := &stallingCreator{fakeCreator: f.creator, entered: make(chan struct{}), release: make(chan struct{})}
	svc := NewService(f.store, f.fees, sc, WithClock(now), WithLimits(ReserveLimits{}))
	l := f.published(validInput())
	firstErr := make(chan error, 1)
	go func() {
		_, err := svc.Pay(context.Background(), l.ShortCode, payReq("k1"))
		firstErr <- err
	}()
	<-sc.entered
	clock.Add(int64(DefaultLease + time.Second))
	if _, released, err := svc.ResolveExpired(context.Background(), 10); err != nil || released != 1 {
		t.Fatalf("resolver released %d: %v", released, err)
	}
	close(sc.release)
	if err := <-firstErr; CodeOf(err) != CodePaymentCreationFailed {
		t.Fatalf("late creation after release: %v", err)
	}
	if _, err := svc.Pay(context.Background(), l.ShortCode, payReq("k1")); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if live := f.creator.livePayments(); live != 1 {
		t.Fatalf("%d live payments for one key", live)
	}
}
