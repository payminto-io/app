package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

// scriptAccounts answers getMultipleAccounts with the given raw balances per token account (nil = missing).
func (f *solanaFixture) scriptAccounts(balances map[string]string) {
	f.rpc.On("getMultipleAccounts", func(p []any) (any, error) {
		addrs, _ := p[0].([]string)
		out := make([]any, len(addrs))
		for i, a := range addrs {
			raw, ok := balances[a]
			if !ok {
				continue
			}
			out[i] = map[string]any{"lamports": 2039280, "owner": solana.TokenProgram.String(), "data": map[string]any{"parsed": map[string]any{"type": "account", "info": map[string]any{"tokenAmount": map[string]any{"amount": raw, "decimals": 6}}}}}
		}
		return solana.ContextValue(1, out), nil
	})
}

func (f *solanaFixture) finalizedSlot(slot uint64) {
	f.rpc.On("getSlot", func(p []any) (any, error) { return slot, nil })
}

// C1 (reviewer's probe): the node answering getTransaction lags the one that listed the signature.
func TestSolanaDeposit_C1_NullTransactionHoldsTheCursorAndCreditsLater(t *testing.T) {
	f := newSolanaFixture(t)
	pr, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	fetch := f.rpc.Handler("getTransaction")
	lagging := true
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		if lagging {
			return nil, nil
		}
		return fetch(p)
	})
	ctx := context.Background()
	if n := f.mustPoll(ctx); n != 0 {
		t.Fatalf("first poll recorded %d", n)
	}
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.TokenAccountCursor != "" {
		t.Fatalf("DEPOSIT LOST: cursor advanced past an unreadable signature: %q", got.TokenAccountCursor)
	}
	if got.HeldSignature == "" || got.HeldAttempts != 1 {
		t.Fatalf("held signature not tracked: %+v", got)
	}
	lagging = false
	if n := f.mustPoll(ctx); n != 1 {
		t.Fatalf("second poll recorded %d, want the deposit", n)
	}
	if deps := f.depositsFor(pr); len(deps) != 1 || !deps[0].Amount.Equal(decimal.RequireFromString("25")) {
		t.Fatalf("deposits = %+v", deps)
	}
	got, _ = f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.HeldSignature != "" || got.TokenAccountCursor == "" {
		t.Fatalf("hold not released: %+v", got)
	}
}

// C1: after the retry budget the signature is recorded as unresolved, tracked by itself, and
// later signatures proceed; when it resolves it is credited.
func TestSolanaDeposit_C1_UnresolvedSignatureIsAnomalyThenCredited(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.cfg.MaxHeldAttempts = 2
	pr, acct := f.newPayment("40", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json", "cpi_inner_transfer.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "37500000"})
	fetch := f.rpc.Handler("getTransaction")
	first, _ := solana.LoadFixtureTransaction(filepath.Join(solanaFixtureDir, "usdc_transfer_checked.json"))
	unreadable := true
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		if unreadable && solana.FirstParamString(p) == first.Signature() {
			return nil, nil
		}
		return fetch(p)
	})
	ctx := context.Background()
	if f.mustPoll(ctx) != 0 {
		t.Fatal("first tick must hold, not credit")
	}
	if f.mustPoll(ctx) != 1 {
		t.Fatal("later signature did not proceed once the budget was spent")
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_unresolved_signature") || missed[0].TxHash != first.Signature() {
		t.Fatalf("missed = %+v", missed)
	}
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if !strings.Contains(got.UnresolvedSignatures, first.Signature()) || got.HeldSignature != "" {
		t.Fatalf("unresolved not tracked individually: %+v", got)
	}
	if len(f.depositsFor(pr)) != 1 {
		t.Fatal("the readable signature should be a deposit")
	}
	unreadable = false
	if f.mustPoll(ctx) != 1 {
		t.Fatal("resolved signature not credited")
	}
	got, _ = f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if strings.Contains(got.UnresolvedSignatures, first.Signature()) {
		t.Fatal("resolved signature still listed")
	}
	if deps := f.depositsFor(pr); len(deps) != 2 {
		t.Fatalf("deposits = %+v", deps)
	}
	if f.mustPoll(ctx) != 0 {
		t.Fatal("re-credited")
	}
}

// I1 (reviewer's scenario): a lagging node's null is not a drop; a drop needs the finalized slot
// past the deposit and the signature absent at finalized on two nodes; a later finalized
// status revives a failed deposit and credits it once.
func TestSolanaDeposit_I1_DropNeedsFinalizedEvidenceAndRevives(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	ctx := context.Background()
	f.mustPoll(ctx)
	f.statuses(map[string]any{"slot": 250000123, "confirmations": 3, "err": nil, "confirmationStatus": "confirmed"})
	f.mustConfirm(ctx)
	// Past the grace, node says unknown, but the finalized slot has not reached the deposit's slot.
	f.svc.now = func() time.Time { return f.now.Add(time.Hour) }
	f.db.Model(&models.Deposit{}).Where("payment_request_id = ?", pr.ID).Update("created_at", f.now.Add(-time.Hour))
	f.rpc.On("getSignatureStatuses", func([]any) (any, error) { return solana.ContextValue(1, []any{nil}), nil })
	f.rpc.Result("getTransaction", nil)
	f.finalizedSlot(250000100)
	f.mustConfirm(ctx)
	if got := f.depositsFor(pr)[0]; got.Status == models.DepositStatusFailed {
		t.Fatal("failed on a single node's unknown while the slot was not finalized")
	}
	// Finalized slot passes the deposit and the signature is absent at finalized: dropped.
	f.finalizedSlot(250000200)
	f.mustConfirm(ctx)
	if got := f.depositsFor(pr)[0]; got.Status != models.DepositStatusFailed {
		t.Fatalf("deposit = %s, want failed", got.Status)
	}
	if f.rpc.Count("getTransaction") < 2 {
		t.Fatal("absence must be checked on more than one node")
	}
	// The cluster now reports it finalized: revive and credit exactly once.
	f.statuses(map[string]any{"slot": 250000123, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	if f.mustConfirm(ctx) != 1 {
		t.Fatal("not revived")
	}
	deps := f.depositsFor(pr)
	if deps[0].Status != models.DepositStatusConfirmed || f.paymentState(pr) != models.PaymentStateFilled {
		t.Fatalf("deposit = %+v state = %s", deps[0], f.paymentState(pr))
	}
	if lines := f.journalLines("deposit", decimal.NewFromInt(int64(deps[0].ID)).String()); len(lines) != 2 {
		t.Fatalf("journal lines = %+v", lines)
	}
	f.mustConfirm(ctx)
	if lines := f.journalLines("deposit", decimal.NewFromInt(int64(deps[0].ID)).String()); len(lines) != 2 {
		t.Fatal("credited twice")
	}
}

// I5: accounts stop being polled after watch_until, poll slower after payment expiry, and an
// unchanged balance (getMultipleAccounts) costs no getSignaturesForAddress call.
func TestSolanaDeposit_I5_WatchWindowCadenceAndBatching(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.cfg.OwnerCadence = time.Hour
	f.svc.cfg.LateCadence = 10 * time.Minute
	_, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	expires := f.now.Add(30 * time.Minute)
	f.db.Model(&models.SolanaDepositAccount{}).Where("id = ?", acct.ID).Updates(map[string]any{"payment_expires_at": expires, "watch_until": expires.Add(24 * time.Hour)})
	f.script(map[string][]string{fxUSDCATA: {}, fxOwner: {}})
	f.scriptAccounts(map[string]string{})
	ctx := context.Background()
	f.mustPoll(ctx)
	if f.rpc.Count("getMultipleAccounts") != 1 || f.rpc.Count("getSignaturesForAddress") != 1 {
		t.Fatalf("first tick: accounts=%d signatures=%d (owner polled once, token account unchanged)", f.rpc.Count("getMultipleAccounts"), f.rpc.Count("getSignaturesForAddress"))
	}
	f.mustPoll(ctx)
	if f.rpc.Count("getSignaturesForAddress") != 1 {
		t.Fatal("unchanged balance and owner cadence not due, yet signatures were fetched")
	}
	// Balance changes: the token account is polled.
	f.scriptAccounts(map[string]string{fxUSDCATA: "1000000"})
	f.mustPoll(ctx)
	if f.rpc.Count("getSignaturesForAddress") != 2 {
		t.Fatalf("balance change did not trigger a signature poll: %d", f.rpc.Count("getSignaturesForAddress"))
	}
	// After payment expiry only the late cadence polls.
	f.svc.now = func() time.Time { return expires.Add(time.Minute) }
	before := f.rpc.Count("getMultipleAccounts")
	f.mustPoll(ctx)
	f.mustPoll(ctx)
	if f.rpc.Count("getMultipleAccounts") != before+1 {
		t.Fatalf("late cadence not applied: %d", f.rpc.Count("getMultipleAccounts")-before)
	}
	// After watch_until the account is expired and never polled again.
	f.svc.now = func() time.Time { return expires.Add(48 * time.Hour) }
	before = f.rpc.Count("getMultipleAccounts")
	f.mustPoll(ctx)
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.Status != models.SolanaDepositAccountExpired || f.rpc.Count("getMultipleAccounts") != before {
		t.Fatalf("status = %s calls = %d", got.Status, f.rpc.Count("getMultipleAccounts")-before)
	}
}

// L1: owner-address dust is bounded per account per day.
func TestSolanaDeposit_L1_DustAnomaliesAreCapped(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.cfg.MaxAnomaliesPerAccountPerDay = 2
	f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		name := "native_sol_to_owner.json"
		f.script(map[string][]string{fxOwner: {name}})
		sig := "dust" + string(rune('A'+i))
		raw, _ := solana.LoadFixtureJSON(filepath.Join(solanaFixtureDir, name))
		raw = []byte(strings.Replace(string(raw), "3aB5cD7eF9gH2jK4lM6nP8qR1sT3uV5wX7yZ9aB2cD4eF6gH8jK1lM3nP5qR7sT9uV2wX4yZ6aB8cD1eF3gH5j", sig, 1))
		f.rpc.On("getSignaturesForAddress", func(p []any) (any, error) {
			if solana.FirstParamString(p) != fxOwner {
				return []any{}, nil
			}
			return []map[string]any{{"signature": sig, "slot": 1, "err": nil, "confirmationStatus": "confirmed"}}, nil
		})
		f.rpc.On("getTransaction", func([]any) (any, error) { return raw, nil })
		f.svc.cfg.OwnerCadence = 0
		f.mustPoll(ctx)
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 2 {
		t.Fatalf("anomaly rows = %d, want the cap of 2", len(missed))
	}
}

// Switch merge: deposit journals can be turned off by wiring so the switch books the payment.
func TestSolanaDeposit_JournalPostingIsOptional(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.ledger = nil
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	ctx := context.Background()
	f.mustPoll(ctx)
	f.statuses(map[string]any{"slot": 1, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	f.mustConfirm(ctx)
	if f.paymentState(pr) != models.PaymentStateFilled {
		t.Fatal("payment not filled")
	}
	var n int64
	f.db.Table("ledger_journals").Count(&n)
	if n != 0 {
		t.Fatalf("journals posted with the flag off: %d", n)
	}
}
