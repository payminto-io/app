package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

func (f *sweepFixture) balanceAlways(raw string) {
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": raw, "decimals": 6}), nil
	})
}

func (f *sweepFixture) blockhash(lastValid uint64) {
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": lastValid}))
}

func (f *sweepFixture) unknownEverywhere() {
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		return solana.ContextValue(1, make([]any, len(sigs))), nil
	})
	f.rpc.Result("getTransaction", nil)
}

func (f *sweepFixture) finalizedTx(sig, ata string, received string) {
	hotATA, _ := solana.AssociatedTokenAddress(f.hot, solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		if solana.FirstParamString(p) != sig {
			return nil, nil
		}
		return map[string]any{
			"slot": 700, "transaction": map[string]any{"signatures": []string{sig}, "message": map[string]any{"accountKeys": []any{
				map[string]any{"pubkey": f.feePayer.PublicKey().String(), "signer": true, "writable": true}, map[string]any{"pubkey": ata}, map[string]any{"pubkey": hotATA.String()},
			}, "instructions": []any{}}},
			"meta": map[string]any{"err": nil, "fee": 5000, "preBalances": []uint64{1, 2039280, 0}, "postBalances": []uint64{1, 0, 2039280},
				"preTokenBalances":  []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": "0", "decimals": 6}}},
				"postTokenBalances": []any{map[string]any{"accountIndex": 2, "mint": fxUSDC, "owner": f.hot.String(), "uiTokenAmount": map[string]any{"amount": received, "decimals": 6}}}},
		}, nil
	})
}

// NEW2-C1: one transport error while a signature is held must not clear the hold.
func TestProbe2_HeldSignatureSurvivesATransportError(t *testing.T) {
	f := newSolanaFixture(t)
	pr, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.scriptAccounts(map[string]string{fxUSDCATA: "25000000"})
	fetch := f.rpc.Handler("getTransaction")
	list := f.rpc.Handler("getSignaturesForAddress")
	lagging, failing := true, false
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		if lagging {
			return nil, nil
		}
		return fetch(p)
	})
	f.rpc.On("getSignaturesForAddress", func(p []any) (any, error) {
		if failing && solana.FirstParamString(p) == fxUSDCATA {
			return nil, errors.New("solana rpc http 503")
		}
		return list(p)
	})
	ctx := context.Background()
	f.mustPoll(ctx)
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	t.Logf("tick 1: held=%q attempts=%d last_balance_raw=%q", got.HeldSignature, got.HeldAttempts, got.LastBalanceRaw)
	failing = true
	f.mustPoll(ctx)
	got, _ = f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	t.Logf("tick 2 (503): held=%q attempts=%d", got.HeldSignature, got.HeldAttempts)
	if got.HeldSignature == "" {
		t.Fatal("the held signature was dropped by one transport error")
	}
	failing, lagging = false, false
	before := f.rpc.Count("getSignaturesForAddress")
	if f.mustPoll(ctx) != 1 || len(f.depositsFor(pr)) != 1 {
		t.Fatalf("DEPOSIT NOT DETECTED after the error (ata signature polls after the error: %d)", f.rpc.Count("getSignaturesForAddress")-before)
	}
}

// NEW2-I1: a send that times out after the node forwarded it keeps its signature and is booked.
func TestProbe2_SendTransportErrorButLanded(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	f.rpc.On("sendTransaction", func([]any) (any, error) { return nil, errors.New("Post: context deadline exceeded") })
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	att := f.attempts(1)
	if f.sweepStatus(1) == SweepStatusFailed || len(att) != 1 || att[0].Signature == "" || att[0].LastValidBlockHeight != 500 {
		t.Fatalf("LANDED SWEEP LOST: sweep=%s attempts=%+v deposit=%s", f.sweepStatus(1), att, f.depositStatus(deps[0].ID))
	}
	// It landed: the signature we persisted is finalized.
	sig := att[0].Signature
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		out := make([]any, len(sigs))
		for i, s := range sigs {
			if s == sig {
				out[i] = map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"}
			}
		}
		return solana.ContextValue(1, out), nil
	})
	f.finalizedTx(sig, ata, "25000000")
	f.rpc.Result("getAccountInfo", solana.ContextValue(1, nil))
	if done, err := f.svc.TrackConfirmations(ctx); err != nil || done != 1 {
		t.Fatalf("track: %d %v", done, err)
	}
	if f.sweepStatus(1) != SweepStatusCompleted || f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatalf("sweep=%s deposit=%s", f.sweepStatus(1), f.depositStatus(deps[0].ID))
	}
	// A JSON-RPC rejection, by contrast, fails the sweep and releases the deposit.
	g := newSweepFixture(t)
	_, _, deps2 := g.newOwner("25")
	g.balanceAlways("25000000")
	g.blockhash(500)
	g.rpc.On("sendTransaction", func([]any) (any, error) {
		return nil, &solana.RPCError{Code: -32002, Message: "Transaction simulation failed"}
	})
	g.svc.SweepConfirmed(ctx)
	if g.sweepStatus(1) != SweepStatusFailed || g.depositStatus(deps2[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("rejected send: sweep=%s deposit=%s", g.sweepStatus(1), g.depositStatus(deps2[0].ID))
	}
}

// NEW2-I2: a processing sweep with no attempt (crash before send) is resolved from the chain.
func TestProbe2_ProcessingSweepWithoutAttemptIsStuck(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	// The rows exist; the process died before signing.
	sweep := models.Sweep{Status: SweepStatusProcessing, BlockchainID: f.chain.ID}
	must(t, f.db.Create(&sweep).Error)
	must(t, f.db.Create(&models.SweepTransaction{Amount: deps[0].Amount, FromAddress: ata, ToAddress: "hot", Status: SweepTxStatusPending, SweepID: sweep.ID, BlockchainCurrencyID: f.usdc.ID}).Error)
	must(t, f.db.Create(&models.SolanaSweepDeposit{SweepID: sweep.ID, DepositID: deps[0].ID}).Error)
	must(t, f.db.Create(&models.SolanaSweepLock{TokenAccount: ata, SweepID: sweep.ID}).Error)
	ctx := context.Background()
	f.svc.TrackConfirmations(ctx)
	if f.sweepStatus(sweep.ID) != SweepStatusProcessing {
		t.Fatal("resolved before the validity window")
	}
	f.svc.now = func() time.Time { return f.now.Add(12 * time.Hour) }
	f.db.Model(&models.Sweep{}).Where("id = ?", sweep.ID).Update("created_at", f.now.Add(-12*time.Hour))
	for i := 0; i < 5; i++ {
		f.svc.TrackConfirmations(ctx)
	}
	missed, _ := f.missed.ListUnresolved()
	if f.sweepStatus(sweep.ID) != SweepStatusFailed || f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed || len(missed) != 1 {
		t.Fatalf("STUCK FOREVER: sweep=%s deposit=%s anomalies=%d", f.sweepStatus(sweep.ID), f.depositStatus(deps[0].ID), len(missed))
	}
	var locks int64
	f.db.Model(&models.SolanaSweepLock{}).Count(&locks)
	if locks != 0 {
		t.Fatal("lock not released")
	}
	// Next round sweeps it normally.
	f.blockhash(500)
	f.rpc.Result("sendTransaction", "SIGN")
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 {
		t.Fatal("not re-swept after release")
	}
}

// NEW2-I3: a recovered attempt without a height still expires by age plus the balance guard.
func TestProbe2_RecoveredAttemptThatDroppedNeverRebuildsOrFails(t *testing.T) {
	f := newSweepFixture(t)
	f.svc.cfg.MaxAttempts = 1
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	sweep := models.Sweep{Status: SweepStatusPending, BlockchainID: f.chain.ID}
	must(t, f.db.Create(&sweep).Error)
	must(t, f.db.Create(&models.SweepTransaction{TxHash: "SIGR", Amount: deps[0].Amount, FromAddress: ata, ToAddress: "hot", Status: SweepTxStatusBroadcast, SweepID: sweep.ID, BlockchainCurrencyID: f.usdc.ID}).Error)
	must(t, f.db.Create(&models.SolanaSweepDeposit{SweepID: sweep.ID, DepositID: deps[0].ID}).Error)
	must(t, f.db.Create(&models.SolanaSweepLock{TokenAccount: ata, SweepID: sweep.ID}).Error)
	must(t, f.db.Create(&models.SolanaSweepAttempt{SweepID: sweep.ID, Signature: "SIGR", Blockhash: "unknown", LastValidBlockHeight: 0, Status: models.SolanaSweepAttemptSent}).Error)
	f.unknownEverywhere()
	f.rpc.Result("getBlockHeight", 600)
	ctx := context.Background()
	f.svc.TrackConfirmations(ctx)
	if f.sweepStatus(sweep.ID) != SweepStatusPending {
		t.Fatal("expired a fresh recovered attempt")
	}
	f.svc.now = func() time.Time { return f.now.Add(12 * time.Hour) }
	f.db.Model(&models.SolanaSweepAttempt{}).Where("sweep_id = ?", sweep.ID).Update("created_at", f.now.Add(-12*time.Hour))
	f.svc.TrackConfirmations(ctx)
	if f.sweepStatus(sweep.ID) != SweepStatusFailed || f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("STUCK: sweep=%s deposit=%s attempts=%+v", f.sweepStatus(sweep.ID), f.depositStatus(deps[0].ID), f.attempts(sweep.ID))
	}
}

// NEW2-I4: a second deposit at an account whose sweep is in flight waits for the first to book.
func TestProbe2_SecondSweepBroadcastWhileFirstIsTracked(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	sent := 0
	f.rpc.On("sendTransaction", func([]any) (any, error) { sent++; return "SIGQ" + string(rune('0'+sent)), nil })
	ctx := context.Background()
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 {
		t.Fatal("first sweep not sent")
	}
	// D2 lands and is confirmed while attempt 1 is in flight.
	d2 := models.Deposit{TxID: "d2", Amount: decimalFromString("10"), Status: models.DepositStatusConfirmed, RequiredConfirmations: 32, ToAddress: ata, BlockchainCurrencyID: f.usdc.ID, MemberID: f.member.ID}
	must(t, f.db.Create(&d2).Error)
	f.balanceAlways("35000000")
	if n, _ := f.svc.SweepConfirmed(ctx); n != 0 {
		t.Fatalf("CONCURRENT SWEEPS on %s: sweep 2 broadcast while sweep 1 is still tracked", ata)
	}
	if f.depositStatus(d2.ID) != models.DepositStatusConfirmed {
		t.Fatal("the late deposit must stay confirmed until the first sweep books")
	}
	// Sweep 1 books (transfer 25, account closed); the lock is released and D2 is swept next.
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		out := make([]any, len(sigs))
		for i := range sigs {
			out[i] = map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"}
		}
		return solana.ContextValue(1, out), nil
	})
	f.finalizedTx("SIGQ1", ata, "25000000")
	f.rpc.Result("getAccountInfo", solana.ContextValue(1, map[string]any{"lamports": 2039280, "owner": solana.TokenProgram.String(), "data": map[string]any{}}))
	if done, _ := f.svc.TrackConfirmations(ctx); done != 1 {
		t.Fatal("sweep 1 not booked")
	}
	f.balanceAlways("10000000")
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 || f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatalf("second sweep after booking: sent=%d d1=%s", n, f.depositStatus(deps[0].ID))
	}
	if missed, _ := f.missed.ListUnresolved(); len(missed) != 0 {
		t.Fatalf("anomalies: %+v", missed)
	}
}

func decimalFromString(s string) decimal.Decimal { return decimal.RequireFromString(s) }
