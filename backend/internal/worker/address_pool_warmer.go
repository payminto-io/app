package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
)

// AddressPoolWarmer is a background worker that keeps every active wallet's
// address pool topped up to a minimum size. Without it, payment creation
// would block on HD derivation the first time a merchant's pool drained.
//
// The worker runs on a ticker (default 60s). Each tick walks every wallet
// in the database and calls AddressPoolService.EnsurePoolSize. If a wallet
// is below the minimum, the service derives fresh addresses and persists
// them as available pool rows.
//
// Minimum pool size defaults to 50, matching DepositAddressService's
// auto-top-up fallback. The interval and size are configurable via
// WithInterval and WithMinSize for tests.
type AddressPoolWarmer struct {
	walletRepo repository.WalletRepository
	poolSvc    *service.AddressPoolService
	interval   time.Duration
	minSize    int
}

// NewAddressPoolWarmer creates an AddressPoolWarmer with default interval (60s)
// and minimum pool size (50). Use WithInterval and WithMinSize to override.
func NewAddressPoolWarmer(
	walletRepo repository.WalletRepository,
	poolSvc *service.AddressPoolService,
) *AddressPoolWarmer {
	return &AddressPoolWarmer{
		walletRepo: walletRepo,
		poolSvc:    poolSvc,
		interval:   60 * time.Second,
		minSize:    50,
	}
}

// WithInterval sets the tick interval (for tests).
func (w *AddressPoolWarmer) WithInterval(d time.Duration) *AddressPoolWarmer {
	w.interval = d
	return w
}

// WithMinSize sets the minimum pool size per wallet (for tests).
func (w *AddressPoolWarmer) WithMinSize(n int) *AddressPoolWarmer {
	w.minSize = n
	return w
}

// Name implements worker.Worker.
func (w *AddressPoolWarmer) Name() string { return "address_pool_warmer" }

// Start implements worker.Worker. Runs the ticker loop until ctx is
// cancelled. Each tick calls TickOnce to process every wallet.
func (w *AddressPoolWarmer) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately once so workers don't wait a full interval on startup
	w.TickOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.TickOnce(ctx)
		}
	}
}

// TickOnce runs a single pass: iterate every wallet and top up each one.
// Exposed for tests so they can trigger a single pass deterministically.
func (w *AddressPoolWarmer) TickOnce(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	wallets, err := w.walletRepo.ListAll()
	if err != nil {
		log.Printf("[address_pool_warmer] list wallets: %v", err)
		return
	}
	for _, wallet := range wallets {
		if ctx.Err() != nil {
			return
		}
		if wallet.Status != "" && wallet.Status != "active" {
			continue
		}
		added, err := w.poolSvc.EnsurePoolSize(wallet.ID, w.minSize)
		if err != nil {
			log.Printf("[address_pool_warmer] wallet %d top-up failed: %v", wallet.ID, err)
			continue
		}
		if added > 0 {
			log.Printf("[address_pool_warmer] wallet %d topped up (+%d addresses)", wallet.ID, added)
		}
	}
}

// Ensure AddressPoolWarmer satisfies the Worker interface.
var _ Worker = (*AddressPoolWarmer)(nil)
