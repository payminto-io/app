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

func (f *sweepFixture) attempts(sweepID uint) []models.SolanaSweepAttempt {
	var out []models.SolanaSweepAttempt
	must(f.t, f.db.Where("sweep_id = ?", sweepID).Order("id").Find(&out).Error)
	return out
}

func (f *sweepFixture) sweepStatus(id uint) string {
	var s models.Sweep
	must(f.t, f.db.First(&s, id).Error)
	return s.Status
}

// L5: SweepConfirmed broadcasts and returns; it never waits on signature statuses.
func TestSolanaSweep_L5_SendDoesNotBlockOnConfirmation(t *testing.T) {
	f := newSweepFixture(t)
	f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.Result("sendTransaction", "SIG1")
	start := time.Now()
	if n, err := f.svc.SweepConfirmed(context.Background()); err != nil || n != 1 {
		t.Fatalf("sweep: %d %v", n, err)
	}
	if time.Since(start) > 500*time.Millisecond || f.rpc.Count("getSignatureStatuses") != 0 {
		t.Fatalf("SweepConfirmed waited: %s, status calls %d", time.Since(start), f.rpc.Count("getSignatureStatuses"))
	}
	att := f.attempts(1)
	if len(att) != 1 || att[0].Signature != "SIG1" || att[0].LastValidBlockHeight != 500 || att[0].Status != models.SolanaSweepAttemptSent {
		t.Fatalf("attempts = %+v", att)
	}
}

// I2 scenario A (reviewer): attempt 1 lands but a lagging node reports it unknown past its blockhash
// validity; attempt 2 is rebuilt and fails on chain. Attempt 1 must be found and booked exactly once.
func TestSolanaSweep_I2_LandedRebuildIsBookedOnce(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	blockhashes := []string{"GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "7pWqF1vXjQ2nD4sT8kL6mB3cR5yH9wE2aG7uN1xP4zV8"}
	f.rpc.On("getLatestBlockhash", func([]any) (any, error) {
		i := min(f.rpc.Count("getLatestBlockhash")-1, 1)
		return solana.ContextValue(1, map[string]any{"blockhash": blockhashes[i], "lastValidBlockHeight": 500 + 100*i}), nil
	})
	sent := 0
	f.rpc.On("sendTransaction", func([]any) (any, error) { sent++; return "SIG" + string(rune('0'+sent)), nil })
	ctx := context.Background()
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 {
		t.Fatal("not sent")
	}
	// Lagging node: SIG1 unknown everywhere, finalized height past attempt 1's validity.
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		out := make([]any, len(sigs))
		return solana.ContextValue(1, out), nil
	})
	f.rpc.Result("getBlockHeight", 600)
	f.rpc.Result("getTransaction", nil)
	if _, err := f.svc.TrackConfirmations(ctx); err != nil {
		t.Fatal(err)
	}
	att := f.attempts(1)
	if len(att) != 2 || att[0].Status != models.SolanaSweepAttemptExpired || att[1].Signature != "SIG2" {
		t.Fatalf("attempts after rebuild = %+v", att)
	}
	// The cluster catches up: SIG1 finalized, SIG2 failed (account already drained).
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		out := make([]any, len(sigs))
		for i, s := range sigs {
			switch s {
			case "SIG1":
				out[i] = map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"}
			case "SIG2":
				out[i] = map[string]any{"slot": 710, "confirmations": nil, "err": map[string]any{"InstructionError": []any{3, map[string]any{"Custom": 1}}}, "confirmationStatus": "finalized"}
			}
		}
		return solana.ContextValue(1, out), nil
	})
	hotATA, _ := solana.AssociatedTokenAddress(f.hot, solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		if solana.FirstParamString(p) != "SIG1" {
			return nil, nil
		}
		return map[string]any{
			"slot": 700, "transaction": map[string]any{"signatures": []string{"SIG1"}, "message": map[string]any{"accountKeys": []any{
				map[string]any{"pubkey": f.feePayer.PublicKey().String(), "signer": true, "writable": true},
				map[string]any{"pubkey": ata, "signer": false, "writable": true},
				map[string]any{"pubkey": hotATA.String(), "signer": false, "writable": true},
			}, "instructions": []any{}}},
			"meta": map[string]any{"err": nil, "fee": 5000, "preBalances": []uint64{1_000_000_000, 2039280, 0}, "postBalances": []uint64{1_000_000_000 - 5000, 0, 2039280},
				"preTokenBalances":  []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "0", "decimals": 6}}},
				"postTokenBalances": []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "25000000", "decimals": 6}}}},
		}, nil
	})
	done, err := f.svc.TrackConfirmations(ctx)
	must(t, err)
	if done != 1 || f.sweepStatus(1) != SweepStatusCompleted {
		t.Fatalf("done=%d status=%s", done, f.sweepStatus(1))
	}
	att = f.attempts(1)
	if att[0].Status != models.SolanaSweepAttemptLanded || att[1].Status != models.SolanaSweepAttemptFailed {
		t.Fatalf("attempts = %+v", att)
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatal("deposit not swept through the booked sweep")
	}
	if lines := f.journalLines("sweep", "1"); len(lines) != 4 {
		t.Fatalf("sweep journal = %+v", lines)
	}
	if done, _ := f.svc.TrackConfirmations(ctx); done != 0 {
		t.Fatal("booked twice")
	}
	if lines := f.journalLines("sweep", "1"); len(lines) != 4 {
		t.Fatal("journal duplicated")
	}
	if missed, _ := f.missed.ListUnresolved(); len(missed) != 0 {
		t.Fatalf("reconciliation raised anomalies on a matching sweep: %+v", missed)
	}
}

// I2: an account that holds nothing is never marked swept; without a booked sweep it is an anomaly.
func TestSolanaSweep_I2_HoldsNothingIsAnomalyNotSwept(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return nil, &solana.RPCError{Code: -32602, Message: "Invalid param: could not find account"}
	})
	f.rpc.On("getSignaturesForAddress", func([]any) (any, error) {
		return []map[string]any{{"signature": "SOMEONE_ELSES", "slot": 1, "err": nil, "confirmationStatus": "finalized"}}, nil
	})
	if n, err := f.svc.SweepConfirmed(context.Background()); err != nil || n != 0 {
		t.Fatalf("sweep: %d %v", n, err)
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("deposit = %s, want confirmed (never swept without a booked row)", f.depositStatus(deps[0].ID))
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_unexplained_drain") || missed[0].ToAddress != ata {
		t.Fatalf("missed = %+v", missed)
	}
	acct, _ := f.accounts.GetByTokenAccount(ata)
	if acct.Status != models.SolanaDepositAccountDrained {
		t.Fatalf("account = %s", acct.Status)
	}
	// Drained accounts are not retried every round.
	if n, _ := f.svc.SweepConfirmed(context.Background()); n != 0 || f.rpc.Count("getTokenAccountBalance") != 1 {
		t.Fatal("drained account re-examined")
	}
}

// I2: a booked sweep whose hot ATA delta disagrees with the recorded amounts raises an anomaly.
func TestSolanaSweep_I2_ReconcileMismatchRaisesAnomaly(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, _ := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.Result("sendTransaction", "SIGM")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	f.statuses(map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	hotATA, _ := solana.AssociatedTokenAddress(f.hot, solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	f.rpc.Result("getTransaction", map[string]any{
		"slot": 700, "transaction": map[string]any{"signatures": []string{"SIGM"}, "message": map[string]any{"accountKeys": []any{
			map[string]any{"pubkey": f.feePayer.PublicKey().String()}, map[string]any{"pubkey": ata}, map[string]any{"pubkey": hotATA.String()},
		}, "instructions": []any{}}},
		"meta": map[string]any{"err": nil, "fee": 5000, "preBalances": []uint64{1, 2039280, 0}, "postBalances": []uint64{1, 0, 2039280},
			"preTokenBalances":  []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "0", "decimals": 6}}},
			"postTokenBalances": []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "24000000", "decimals": 6}}}},
	})
	if done, err := f.svc.TrackConfirmations(ctx); err != nil || done != 1 {
		t.Fatalf("track: %d %v", done, err)
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, "solana_sweep_mismatch") || !missed[0].Amount.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("missed = %+v", missed)
	}
}

// M5: failing a sweep releases only the deposits that sweep claimed.
func TestSolanaSweep_M5_FailReleasesOnlyItsDeposits(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	// An earlier, completed sweep already moved an older deposit at the same address.
	older := models.Deposit{TxID: "older", Amount: decimal.RequireFromString("5"), Status: models.DepositStatusSwept, RequiredConfirmations: 32, ToAddress: ata, BlockchainCurrencyID: f.usdc.ID, MemberID: f.member.ID}
	must(t, f.db.Create(&older).Error)
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.Result("sendTransaction", "SIGF")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	f.statuses(map[string]any{"slot": 700, "confirmations": nil, "err": map[string]any{"InstructionError": []any{0, map[string]any{"Custom": 1}}}, "confirmationStatus": "finalized"})
	f.svc.cfg.MaxAttempts = 1
	if _, err := f.svc.TrackConfirmations(ctx); err != nil {
		t.Fatal(err)
	}
	if f.sweepStatus(1) != SweepStatusFailed {
		t.Fatalf("sweep = %s", f.sweepStatus(1))
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatal("the batch's deposit must be released")
	}
	if f.depositStatus(older.ID) != models.DepositStatusSwept {
		t.Fatal("an older swept deposit at the same address was released")
	}
}
