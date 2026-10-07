//go:build integration

package links_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

// stalled wraps the real Payminto creator and holds the first CreatePayment until released.
type stalled struct {
	*service.LinkPaymentCreator
	entered, release chan struct{}
	once             sync.Once
}

func (s *stalled) CreatePayment(ctx context.Context, req links.PaymentRequest) (links.CreatedPayment, error) {
	first := false
	s.once.Do(func() { first = true })
	if first {
		close(s.entered)
		<-s.release
	}
	return s.LinkPaymentCreator.CreatePayment(context.WithoutCancel(ctx), req)
}

func TestIntegration_StalledPaymintoCreationAfterReleaseLeavesOneLivePayment(t *testing.T) {
	base := newStack(t)
	ctx := context.Background()
	family := models.BlockchainFamily{Name: "SOL family", Code: "sol-fam"}
	if err := base.db.Create(&family).Error; err != nil {
		t.Fatal(err)
	}
	chain := models.Blockchain{Code: "SOL", Name: "Solana", BlockchainFamilyID: family.ID, Status: "active"}
	if err := base.db.Create(&chain).Error; err != nil {
		t.Fatal(err)
	}
	cur := models.Currency{Code: "USDC", Name: "USD Coin", Type: "token"}
	if err := base.db.Create(&cur).Error; err != nil {
		t.Fatal(err)
	}
	if err := base.db.Create(&models.BlockchainCurrency{CurrencyCode: "USDC", BlockchainCode: "SOL", BlockchainID: chain.ID, CurrencyID: cur.ID, DepositEnabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	feeSvc := fees.NewService(base.db, nil, fees.DefaultPolicy())
	if _, err := feeSvc.CreateRule(ctx, fees.RuleInput{
		Scope:   fees.Scope{Method: fees.MethodCrypto, Currency: "USDC"},
		Pricing: fees.Pricing{Percent: decimal.RequireFromString("1"), FeeBearer: fees.BearerMerchant},
	}, "member:1"); err != nil {
		t.Fatal(err)
	}
	real := service.NewLinkPaymentCreator(service.NewPaymentService(repository.NewPaymentRepository(base.db)), base.db, "https://checkout.test")
	sc := &stalled{LinkPaymentCreator: real, entered: make(chan struct{}), release: make(chan struct{})}
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	svc := links.NewService(base.store, feeSvc, sc, links.WithClock(now), links.WithLimits(links.ReserveLimits{}))

	in := links.DefaultInput()
	in.Title, in.Currency, in.Amount, in.SuccessMessage = "Beans", "USD", d("25.00"), "Thanks"
	usdc := links.MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDC"}
	in.Methods = []links.MethodSpec{usdc}
	l, err := svc.Create(ctx, base.actor, in)
	if err != nil {
		t.Fatal(err)
	}
	if l, err = svc.Publish(ctx, base.actor.PlatformID, l.ID); err != nil {
		t.Fatal(err)
	}
	req := cardPay("k1", "a@example.test")
	req.Method = usdc

	firstErr := make(chan error, 1)
	go func() { _, err := svc.Pay(ctx, l.ShortCode, req); firstErr <- err }()
	<-sc.entered
	clock.Add(int64(links.DefaultLease + time.Second))
	if _, released, err := svc.ResolveExpired(ctx, 10); err != nil || released != 1 {
		t.Fatalf("resolver released %d: %v", released, err)
	}
	close(sc.release)
	if err := <-firstErr; err == nil {
		t.Fatal("the stalled call succeeded after its use was released")
	}
	res, err := svc.Pay(ctx, l.ShortCode, req)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	var live []string
	base.db.Model(&models.PaymentRequest{}).Where("reference_id LIKE 'pl_%' AND state <> ?", models.PaymentStateCancelled).Pluck("reference_id", &live)
	if len(live) != 1 || live[0] != res.PaymentReference || !strings.HasPrefix(res.PaymentReference, "pl_") {
		t.Fatalf("live payments %v, retry %s", live, res.PaymentReference)
	}
	var uses int
	base.db.Raw(`SELECT uses_count FROM payment_links WHERE id = ?`, l.ID).Scan(&uses)
	if uses != 1 {
		t.Fatalf("uses %d", uses)
	}
}
