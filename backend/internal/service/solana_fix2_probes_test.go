package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
)

// NEW-C1 (reviewer's probe): a deposit released by a failed sweep is re-swept; the broadcast must
// have rows and be booked once, and the deposit ends swept through a sweep row.
func TestProbe_ResweepAfterFailedSweep(t *testing.T) {
	f := newSweepFixture(t)
	f.svc.cfg.MaxAttempts = 1
	_, ata, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	sent := 0
	f.rpc.On("sendTransaction", func([]any) (any, error) { sent++; return "SIG" + string(rune('0'+sent)), nil })
	ctx := context.Background()
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 {
		t.Fatal("first sweep not sent")
	}
	// Sweep 1 expires with evidence and the budget is spent: released.
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		return solana.ContextValue(1, make([]any, len(sigs))), nil
	})
	f.rpc.Result("getBlockHeight", 600)
	f.rpc.Result("getTransaction", nil)
	if _, err := f.svc.TrackConfirmations(ctx); err != nil {
		t.Fatal(err)
	}
	if f.sweepStatus(1) != SweepStatusFailed || f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("sweep 1 = %s deposit = %s", f.sweepStatus(1), f.depositStatus(deps[0].ID))
	}
	// Second round re-sweeps the same deposit.
	if n, err := f.svc.SweepConfirmed(ctx); err != nil || n != 1 {
		t.Fatalf("re-sweep: %d %v", n, err)
	}
	var sweeps, attempts, links int64
	f.db.Model(&models.Sweep{}).Count(&sweeps)
	f.db.Model(&models.SolanaSweepAttempt{}).Count(&attempts)
	f.db.Model(&models.SolanaSweepDeposit{}).Count(&links)
	if sweeps != 2 || attempts != 2 || links != 2 {
		t.Fatalf("BROADCAST WITHOUT ROWS: sweeps=%d attempts=%d links=%d deposit=%s", sweeps, attempts, links, f.depositStatus(deps[0].ID))
	}
	// SIG2 finalizes: booked once, deposit swept through sweep 2.
	f.statuses(map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	hotATA, _ := solana.AssociatedTokenAddress(f.hot, solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	f.rpc.Result("getTransaction", map[string]any{
		"slot": 700, "transaction": map[string]any{"signatures": []string{"SIG2"}, "message": map[string]any{"accountKeys": []any{
			map[string]any{"pubkey": f.feePayer.PublicKey().String()}, map[string]any{"pubkey": ata}, map[string]any{"pubkey": hotATA.String()},
		}, "instructions": []any{}}},
		"meta": map[string]any{"err": nil, "fee": 5000, "preBalances": []uint64{1, 2039280, 0}, "postBalances": []uint64{1, 0, 2039280},
			"preTokenBalances":  []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "0", "decimals": 6}}},
			"postTokenBalances": []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "25000000", "decimals": 6}}}},
	})
	if done, err := f.svc.TrackConfirmations(ctx); err != nil || done != 1 {
		t.Fatalf("track: %d %v", done, err)
	}
	if f.sweepStatus(2) != SweepStatusCompleted || f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatalf("sweep 2 = %s deposit = %s", f.sweepStatus(2), f.depositStatus(deps[0].ID))
	}
	if lines := f.journalLines("sweep", "2"); len(lines) != 4 {
		t.Fatalf("sweep 2 journal = %+v", lines)
	}
	if lines := f.journalLines("sweep", "1"); len(lines) != 0 {
		t.Fatalf("failed sweep 1 booked: %+v", lines)
	}
}

// NEW-C2 (reviewer's probe A): getMultipleAccounts shows the balance, the next node lags on
// getSignaturesForAddress; later ticks must still poll until the movement is explained.
func TestProbe_BalanceGate_LaggingSignatureNode(t *testing.T) {
	f := newSolanaFixture(t)
	pr, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	list := f.rpc.Handler("getSignaturesForAddress")
	lagging := true
	f.rpc.On("getSignaturesForAddress", func(p []any) (any, error) {
		if lagging && solana.FirstParamString(p) == fxUSDCATA {
			return []any{}, nil
		}
		return list(p)
	})
	ctx := context.Background()
	f.mustPoll(ctx)
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	t.Logf("after tick 1: last_balance_raw=%q held=%q cursor=%q", got.LastBalanceRaw, got.HeldSignature, got.TokenAccountCursor)
	lagging = false
	recorded := 0
	for i := 0; i < 4 && recorded == 0; i++ {
		recorded += f.mustPoll(ctx)
	}
	if recorded != 1 || len(f.depositsFor(pr)) != 1 {
		t.Fatalf("DEPOSIT NOT DETECTED after node B caught up (ata signature polls: %d)", f.rpc.Count("getSignaturesForAddress"))
	}
}

// NEW-C2 (reviewer's probe B): a transport error on the signature call must not park the account.
func TestProbe_BalanceGate_SignatureCallError(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	list := f.rpc.Handler("getSignaturesForAddress")
	failing := true
	f.rpc.On("getSignaturesForAddress", func(p []any) (any, error) {
		if failing && solana.FirstParamString(p) == fxUSDCATA {
			return nil, errors.New("solana rpc http 503")
		}
		return list(p)
	})
	ctx := context.Background()
	f.mustPoll(ctx)
	failing = false
	recorded := 0
	for i := 0; i < 4 && recorded == 0; i++ {
		recorded += f.mustPoll(ctx)
	}
	if recorded != 1 || len(f.depositsFor(pr)) != 1 {
		t.Fatalf("DEPOSIT NOT DETECTED after the error cleared (ata signature polls: %d)", f.rpc.Count("getSignaturesForAddress"))
	}
	if missed, _ := f.missed.ListUnresolved(); len(missed) != 0 && !strings.Contains(missed[0].Reason, "balance") {
		t.Fatalf("unexpected anomalies: %+v", missed)
	}
}
