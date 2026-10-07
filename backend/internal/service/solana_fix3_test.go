package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
)

// NEW2-C1, unheld case: a moved balance whose signature poll fails is not persisted, so the next tick polls again.
func TestSolanaDeposit_TransportErrorKeepsMovedBalanceUnexplained(t *testing.T) {
	f := newSolanaFixture(t)
	pr, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	list := f.rpc.Handler("getSignaturesForAddress")
	failing := true
	f.rpc.On("getSignaturesForAddress", func(p []any) (any, error) {
		if failing && solana.FirstParamString(p) == fxUSDCATA {
			return nil, errors.New("solana rpc http 429")
		}
		return list(p)
	})
	ctx := context.Background()
	f.svc.PollOnce(ctx)
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.LastBalanceRaw == "25000000" || got.BalanceHoldAttempts != 1 {
		t.Fatalf("a failed poll explained the movement: last_balance_raw=%q hold=%d", got.LastBalanceRaw, got.BalanceHoldAttempts)
	}
	failing = false
	if f.mustPoll(ctx) != 1 || len(f.depositsFor(pr)) != 1 {
		t.Fatal("deposit not detected after the transport error")
	}
	got, _ = f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.LastBalanceRaw != "25000000" || got.BalanceHoldAttempts != 0 {
		t.Fatalf("after the good poll: %+v", got)
	}
}

// NEW2-L3: unresolved signatures keep an account from expiring only within the budget; past it the
// account expires with an anomaly and the expired scan still resolves and credits the signature.
func TestSolanaDeposit_UnresolvedSignatureDoesNotBlockExpiryForever(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.cfg.MaxUnresolvedPolls = 3
	f.svc.cfg.ExpiredScan = time.Hour
	pr, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	first, _ := solana.LoadFixtureTransaction(filepath.Join(solanaFixtureDir, "usdc_transfer_checked.json"))
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}, fxOwner: {}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	fetch := f.rpc.Handler("getTransaction")
	unreadable := true
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		if unreadable && solana.FirstParamString(p) == first.Signature() {
			return nil, nil
		}
		return fetch(p)
	})
	past := f.now.Add(-time.Minute)
	must(t, f.accounts.Update(acct.ID, map[string]any{"watch_until": past, "payment_expires_at": past.Add(-time.Hour),
		"unresolved_signatures": encodeSignatures([]string{first.Signature()}), "last_balance_raw": "25000000"}))
	ctx := context.Background()
	at := f.now
	for i := 1; i <= 3; i++ {
		f.svc.PollOnce(ctx)
		got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
		t.Logf("poll %d: status=%s unresolved_attempts=%d", i, got.Status, got.UnresolvedAttempts)
		if got.Status != models.SolanaDepositAccountWatching {
			t.Fatalf("expired after %d polls with an unresolved signature inside the budget", i)
		}
		at = at.Add(f.svc.cfg.LateCadence + time.Minute)
		now := at
		f.svc.now = func() time.Time { return now }
	}
	f.svc.PollOnce(ctx)
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.Status != models.SolanaDepositAccountExpired || !strings.Contains(got.UnresolvedSignatures, first.Signature()) {
		t.Fatalf("past the budget: status=%s unresolved=%q", got.Status, got.UnresolvedSignatures)
	}
	missed, _ := f.missed.ListUnresolved()
	found := false
	for _, m := range missed {
		found = found || strings.HasPrefix(m.Reason, anomalyUnresolvedAtExpiry)
	}
	if !found {
		t.Fatalf("no %s anomaly: %+v", anomalyUnresolvedAtExpiry, missed)
	}
	// The expired scan keeps retrying it; once readable it is credited and leaves the list.
	unreadable = false
	later := at.Add(2 * time.Hour)
	f.svc.now = func() time.Time { return later }
	f.svc.PollOnce(ctx)
	got, _ = f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.UnresolvedSignatures != "" || len(f.depositsFor(pr)) != 1 {
		t.Fatalf("expired scan did not resolve it: unresolved=%q deposits=%d", got.UnresolvedSignatures, len(f.depositsFor(pr)))
	}
}
