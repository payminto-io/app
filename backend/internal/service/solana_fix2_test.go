package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

// NEW-C2: a balance movement no node lists is held under its own budget and ends in an anomaly.
func TestSolanaDeposit_C2_UnexplainedBalanceMovementIsHeldThenFlagged(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.cfg.MaxHeldAttempts = 3
	_, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {}, fxOwner: {}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		f.mustPoll(ctx)
		got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
		if got.LastBalanceRaw != "" || got.BalanceHoldAttempts != i+1 {
			t.Fatalf("tick %d: balance persisted early or hold not counted: %+v", i+1, got)
		}
	}
	f.mustPoll(ctx)
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.LastBalanceRaw != "25000000" || got.BalanceHoldAttempts != 0 {
		t.Fatalf("after the budget: %+v", got)
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_unexplained_balance") {
		t.Fatalf("missed = %+v", missed)
	}
	if f.rpc.Count("getSignaturesForAddress") < 3 {
		t.Fatal("the token account was not re-polled while the movement was unexplained")
	}
}

// NEW-M1: expiry keeps held signatures tracked, reads the balance once and flags late money.
func TestSolanaDeposit_M1_ExpiryKeepsHeldAndFlagsLateBalance(t *testing.T) {
	f := newSolanaFixture(t)
	f.svc.cfg.ExpiredScan = time.Hour
	_, held := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	_, late := f.newPayment("10", f.usdt, fxOwner, fxUSDTATA)
	past := f.now.Add(-time.Minute)
	must(t, f.accounts.Update(held.ID, map[string]any{"watch_until": past, "held_signature": "stuck", "held_attempts": 1}))
	must(t, f.accounts.Update(late.ID, map[string]any{"watch_until": past}))
	f.script(map[string][]string{fxUSDCATA: {}, fxUSDTATA: {}, fxOwner: {}})
	f.scriptAccounts(map[string]string{fxUSDTATA: "10000000"})
	f.mustPoll(context.Background())
	h, _ := f.accounts.GetByDepositAddressID(held.DepositAddressID)
	if h.Status != models.SolanaDepositAccountWatching {
		t.Fatal("an account holding a signature must not expire")
	}
	l, _ := f.accounts.GetByDepositAddressID(late.DepositAddressID)
	if l.Status != models.SolanaDepositAccountExpired || l.LastBalanceRaw != "10000000" {
		t.Fatalf("late account = %+v", l)
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_late_balance") || !missed[0].Amount.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("missed = %+v", missed)
	}
	// The expired scan runs on its cadence and flags a further rise once.
	f.svc.now = func() time.Time { return f.now.Add(2 * time.Hour) }
	f.scriptAccounts(map[string]string{fxUSDTATA: "15000000"})
	f.mustPoll(context.Background())
	if missed, _ = f.missed.ListUnresolved(); len(missed) != 2 {
		t.Fatalf("expired scan did not flag the rise: %+v", missed)
	}
}

// NEW-M3: with one endpoint a drop cannot be evidenced; the deposit stays pending with an anomaly.
func TestSolanaDeposit_M3_SingleEndpointNeverFailsADeposit(t *testing.T) {
	f := newSolanaFixture(t)
	f.rpc.Nodes = 1
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	ctx := context.Background()
	f.mustPoll(ctx)
	f.svc.now = func() time.Time { return f.now.Add(time.Hour) }
	f.db.Model(&models.Deposit{}).Where("payment_request_id = ?", pr.ID).Update("created_at", f.now.Add(-time.Hour))
	f.rpc.On("getSignatureStatuses", func([]any) (any, error) { return solana.ContextValue(1, []any{nil}), nil })
	f.rpc.Result("getTransaction", nil)
	f.finalizedSlot(250000200)
	f.mustConfirm(ctx)
	if got := f.depositsFor(pr)[0]; got.Status != models.DepositStatusPending {
		t.Fatalf("deposit = %s, want pending", got.Status)
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_evidence_unavailable") {
		t.Fatalf("missed = %+v", missed)
	}
}

// NEW-L1: a deposit finalized for an expired payment is flagged, not just logged.
func TestSolanaDeposit_L1_LatePaymentIsAnAnomaly(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.db.Model(&models.PaymentRequest{}).Where("id = ?", pr.ID).Update("state", "EXPIRED")
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	ctx := context.Background()
	f.mustPoll(ctx)
	f.statuses(map[string]any{"slot": 1, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	f.mustConfirm(ctx)
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "late_payment") || !missed[0].Amount.Equal(decimal.RequireFromString("25")) {
		t.Fatalf("missed = %+v", missed)
	}
	if f.paymentState(pr) != "EXPIRED" {
		t.Fatal("payment state must not be rewritten")
	}
}

// NEW-I1: at the attempt budget an empty account means a possible landing, never a failure; a
// failed sweep whose signature explains a drain is revived and booked once.
func TestSolanaSweep_I1_LastAttemptLandingBehindLaggingNodesIsBooked(t *testing.T) {
	f := newSweepFixture(t)
	f.svc.cfg.MaxAttempts = 1
	_, ata, deps := f.newOwner("25")
	balance := "25000000"
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": balance, "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.Result("sendTransaction", "SIGL")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	// The attempt landed (the account is empty) but both nodes lag and the height passed validity.
	balance = "0"
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		return solana.ContextValue(1, make([]any, len(sigs))), nil
	})
	f.rpc.Result("getBlockHeight", 600)
	f.rpc.Result("getTransaction", nil)
	f.svc.TrackConfirmations(ctx)
	if f.sweepStatus(1) != SweepStatusPending {
		t.Fatalf("sweep failed with an empty account: %s", f.sweepStatus(1))
	}
	// Now simulate the pre-fix state: a sweep booked failed with evidence, then found to have landed.
	must(t, f.sweepSvc.MarkFailed(1))
	f.db.Model(&models.Deposit{}).Where("id = ?", deps[0].ID).Update("status", models.DepositStatusConfirmed)
	f.rpc.On("getSignaturesForAddress", func([]any) (any, error) {
		return []map[string]any{{"signature": "SIGL", "slot": 700, "err": nil, "confirmationStatus": "finalized"}}, nil
	})
	if n, _ := f.svc.SweepConfirmed(ctx); n != 0 {
		t.Fatal("re-broadcast for an account drained by our own sweep")
	}
	if f.sweepStatus(1) != SweepStatusPending {
		t.Fatalf("failed sweep not revived: %s", f.sweepStatus(1))
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_failed_sweep_landed") {
		t.Fatalf("missed = %+v", missed)
	}
	// The cluster catches up: booked once, deposit swept, no churn.
	f.statuses(map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	hotATA, _ := solana.AssociatedTokenAddress(f.hot, solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	f.rpc.Result("getTransaction", map[string]any{
		"slot": 700, "transaction": map[string]any{"signatures": []string{"SIGL"}, "message": map[string]any{"accountKeys": []any{
			map[string]any{"pubkey": f.feePayer.PublicKey().String()}, map[string]any{"pubkey": ata}, map[string]any{"pubkey": hotATA.String()},
		}, "instructions": []any{}}},
		"meta": map[string]any{"err": nil, "fee": 5000, "preBalances": []uint64{1, 2039280, 0}, "postBalances": []uint64{1, 0, 2039280},
			"preTokenBalances":  []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "0", "decimals": 6}}},
			"postTokenBalances": []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "25000000", "decimals": 6}}}},
	})
	f.rpc.Result("getAccountInfo", solana.ContextValue(1, nil))
	if done, err := f.svc.TrackConfirmations(ctx); err != nil || done != 1 {
		t.Fatalf("track: %d %v", done, err)
	}
	if f.sweepStatus(1) != SweepStatusCompleted || f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatalf("sweep = %s deposit = %s", f.sweepStatus(1), f.depositStatus(deps[0].ID))
	}
	if n, _ := f.svc.SweepConfirmed(ctx); n != 0 || f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatal("churn after booking")
	}
}

// NEW-M2: money beyond the claim stays for the next sweep: the transfer is the claim, no close.
func TestSolanaSweep_M2_MoneyBeyondTheClaimIsLeftForTheNextSweep(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "40000000", "decimals": 6}), nil // 15 more landed mid-sweep
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	var sent string
	f.rpc.On("sendTransaction", func(p []any) (any, error) { sent = solana.FirstParamString(p); return "SIGM2", nil })
	if n, _ := f.svc.SweepConfirmed(context.Background()); n != 1 {
		t.Fatal("not sent")
	}
	tx, err := solana.DecodeTransactionBase64(sent)
	must(t, err)
	transfers, closes := 0, 0
	for _, ci := range tx.Message.Instructions {
		ix, _ := tx.Message.Instruction(ci)
		if ix.ProgramID == solana.TokenProgram && ix.Data[0] == 12 {
			transfers++
			if string(ix.Data[1:9]) != string([]byte{0x40, 0x78, 0x7d, 0x01, 0, 0, 0, 0}) { // 25_000_000 LE
				t.Fatalf("transfer amount bytes = %x, want 25000000", ix.Data[1:9])
			}
		}
		if ix.ProgramID == solana.TokenProgram && ix.Data[0] == 9 {
			closes++
		}
	}
	if transfers != 1 || closes != 0 {
		t.Fatalf("transfers=%d closes=%d", transfers, closes)
	}
	var st models.SweepTransaction
	must(t, f.db.First(&st).Error)
	if !st.Amount.Equal(decimal.RequireFromString("25")) || st.FromAddress != ata {
		t.Fatalf("sweep tx = %+v", st)
	}
	_ = deps
}
