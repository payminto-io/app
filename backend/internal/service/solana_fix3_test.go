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
// R3-L2: and the failed polls do not spend the balance-hold budget.
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
	// R3-L2: an outage longer than the hold budget spends none of it and writes no anomaly.
	for i := 0; i <= f.svc.cfg.MaxHeldAttempts; i++ {
		f.svc.PollOnce(ctx)
	}
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.LastBalanceRaw == "25000000" || got.BalanceHoldAttempts != 0 {
		t.Fatalf("failed polls explained the movement or spent the budget: last_balance_raw=%q hold=%d", got.LastBalanceRaw, got.BalanceHoldAttempts)
	}
	if missed, _ := f.missed.ListUnresolved(); len(missed) != 0 {
		t.Fatalf("anomaly during an outage: %+v", missed)
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

// processingSweepRows writes the rows a crash before signing leaves behind, aged past the validity window.
func (f *sweepFixture) processingSweepRows(ata string, dep models.Deposit) models.Sweep {
	f.t.Helper()
	sweep := models.Sweep{Status: SweepStatusProcessing, BlockchainID: f.chain.ID}
	must(f.t, f.db.Create(&sweep).Error)
	must(f.t, f.db.Create(&models.SweepTransaction{Amount: dep.Amount, FromAddress: ata, ToAddress: "hot", Status: SweepTxStatusPending, SweepID: sweep.ID, BlockchainCurrencyID: f.usdc.ID}).Error)
	must(f.t, f.db.Create(&models.SolanaSweepDeposit{SweepID: sweep.ID, DepositID: dep.ID}).Error)
	must(f.t, f.db.Create(&models.SolanaSweepLock{TokenAccount: ata, SweepID: sweep.ID}).Error)
	f.db.Model(&models.Sweep{}).Where("id = ?", sweep.ID).Update("created_at", f.now.Add(-time.Hour))
	return sweep
}

func (f *sweepFixture) history(sigs ...string) {
	f.rpc.On("getSignaturesForAddress", func([]any) (any, error) {
		out := make([]map[string]any, len(sigs))
		for i, s := range sigs {
			out[i] = map[string]any{"signature": s, "slot": 700, "err": nil, "confirmationStatus": "finalized"}
		}
		return out, nil
	})
}

func (f *sweepFixture) finalizedEverywhere() {
	f.statuses(map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
}

// NEW2-I2, balance moved: a processing sweep with no attempt whose account was drained by a transaction
// our fee payer signed records that transaction as its attempt, with a real height, and books it.
func TestSolanaSweep_ProcessingWithoutAttemptRecoversOurTransactionFromHistory(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("0")
	sweep := f.processingSweepRows(ata, deps[0])
	f.history("OURS")
	f.rpc.Result("getBlockHeight", 1000)
	f.unknownEverywhere()
	f.finalizedTx("OURS", ata, "25000000") // unknownEverywhere reset getTransaction
	ctx := context.Background()
	f.svc.TrackConfirmations(ctx)
	att := f.attempts(sweep.ID)
	if len(att) != 1 || att[0].Signature != "OURS" || att[0].LastValidBlockHeight == 0 || f.sweepStatus(sweep.ID) != SweepStatusPending {
		t.Fatalf("not recovered: sweep=%s attempts=%+v", f.sweepStatus(sweep.ID), att)
	}
	if att[0].LastValidBlockHeight != 1000+solana.BlockhashValidityBlocks {
		t.Fatalf("recovered height = %d", att[0].LastValidBlockHeight)
	}
	f.finalizedEverywhere()
	f.rpc.Result("getAccountInfo", solana.ContextValue(1, nil))
	if done, err := f.svc.TrackConfirmations(ctx); err != nil || done != 1 {
		t.Fatalf("track: %d %v", done, err)
	}
	if f.sweepStatus(sweep.ID) != SweepStatusCompleted || f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatalf("sweep=%s deposit=%s", f.sweepStatus(sweep.ID), f.depositStatus(deps[0].ID))
	}
	var locks int64
	f.db.Model(&models.SolanaSweepLock{}).Count(&locks)
	if locks != 0 {
		t.Fatal("lock kept after booking")
	}
}

// NEW2-I2, balance moved by someone else: the sweep fails, the account is set aside with an anomaly.
func TestSolanaSweep_ProcessingWithoutAttemptAndForeignDrainFailsWithAnomaly(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("0")
	sweep := f.processingSweepRows(ata, deps[0])
	f.history("THEIRS")
	f.rpc.Result("getTransaction", map[string]any{"slot": 700, "transaction": map[string]any{"signatures": []string{"THEIRS"}, "message": map[string]any{
		"accountKeys": []any{map[string]any{"pubkey": fxPayer, "signer": true}}, "instructions": []any{}}}, "meta": map[string]any{"err": nil}})
	f.svc.TrackConfirmations(context.Background())
	acct, _ := f.accounts.GetByTokenAccount(ata)
	missed, _ := f.missed.ListUnresolved()
	if f.sweepStatus(sweep.ID) != SweepStatusFailed || acct.Status != models.SolanaDepositAccountDrained || len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, anomalyUnexplainedDrain) {
		t.Fatalf("sweep=%s account=%s missed=%+v", f.sweepStatus(sweep.ID), acct.Status, missed)
	}
	if n, _ := f.svc.SweepConfirmed(context.Background()); n != 0 {
		t.Fatal("a drained account was swept again")
	}
}

// A pending sweep whose account sits below its claim with nothing finalized waits, and after DrainWait
// with nothing of ours in the history it fails with an anomaly instead of waiting forever.
func TestSolanaSweep_ShortBalanceWaitsThenNeedsAnExplanation(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	f.rpc.Result("sendTransaction", "SIGS")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	f.balanceAlways("0")
	f.unknownEverywhere()
	f.rpc.Result("getBlockHeight", 600)
	f.history()
	f.svc.TrackConfirmations(ctx)
	if f.sweepStatus(1) != SweepStatusPending {
		t.Fatalf("did not wait: %s", f.sweepStatus(1))
	}
	f.svc.now = func() time.Time { return f.now.Add(2 * time.Hour) }
	f.db.Model(&models.SolanaSweepAttempt{}).Where("sweep_id = ?", 1).Update("created_at", f.now.Add(-2*time.Hour))
	f.svc.TrackConfirmations(ctx)
	acct, _ := f.accounts.GetByTokenAccount(ata)
	if f.sweepStatus(1) != SweepStatusFailed || acct.Status != models.SolanaDepositAccountDrained || f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("sweep=%s account=%s deposit=%s", f.sweepStatus(1), acct.Status, f.depositStatus(deps[0].ID))
	}
}

// Every signature is in the database before any node sees it.
func TestSolanaSweep_SignatureIsPersistedBeforeSend(t *testing.T) {
	f := newSweepFixture(t)
	f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	var atSend []models.SolanaSweepAttempt
	f.rpc.On("sendTransaction", func(p []any) (any, error) {
		tx, err := solana.DecodeTransactionBase64(solana.FirstParamString(p))
		must(t, err)
		must(t, f.db.Where("signature = ?", tx.Signature()).Find(&atSend).Error)
		return tx.Signature(), nil
	})
	if n, _ := f.svc.SweepConfirmed(context.Background()); n != 1 {
		t.Fatal("not sent")
	}
	if len(atSend) != 1 || atSend[0].Status != models.SolanaSweepAttemptSigned || atSend[0].LastValidBlockHeight != 500 {
		t.Fatalf("attempt at send time = %+v", atSend)
	}
	if att := f.attempts(1); att[0].Status != models.SolanaSweepAttemptSent {
		t.Fatalf("after ack: %+v", att)
	}
}

// A signed attempt whose send failed in transport and never landed expires on evidence and is rebuilt.
func TestSolanaSweep_SignedAttemptThatNeverLandedIsRebuilt(t *testing.T) {
	f := newSweepFixture(t)
	f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	f.rpc.On("sendTransaction", func([]any) (any, error) { return nil, errors.New("connection reset by peer") })
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	if att := f.attempts(1); len(att) != 1 || att[0].Status != models.SolanaSweepAttemptSigned {
		t.Fatalf("attempts = %+v", att)
	}
	f.unknownEverywhere()
	f.rpc.Result("getBlockHeight", 600)
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "7pWqF1vXjQ2nD4sT8kL6mB3cR5yH9wE2aG7uN1xP4zV8", "lastValidBlockHeight": 750}))
	f.rpc.Result("sendTransaction", "SIGR2")
	f.svc.TrackConfirmations(ctx)
	att := f.attempts(1)
	if len(att) != 2 || att[0].Status != models.SolanaSweepAttemptExpired || att[1].Status != models.SolanaSweepAttemptSent || att[1].LastValidBlockHeight != 750 {
		t.Fatalf("attempts after rebuild = %+v", att)
	}
}

// NEW2-I4 at the database: the lock's unique index refuses a second sweep on one account.
func TestSolanaSweep_LockIsUniquePerAccount(t *testing.T) {
	f := newSweepFixture(t)
	must(t, f.db.Create(&models.SolanaSweepLock{TokenAccount: "ATA1", SweepID: 1}).Error)
	if err := f.db.Create(&models.SolanaSweepLock{TokenAccount: "ATA1", SweepID: 2}).Error; err == nil {
		t.Fatal("second lock on one account accepted")
	}
}

// Two workers rebuilding one sweep: the (sweep_id, attempt_no) index lets one attempt through and the
// other is never sent; a sweep failed meanwhile takes no attempt at all.
func TestSolanaSweep_ConcurrentRebuildAndFailedSweepSendNothing(t *testing.T) {
	f := newSweepFixture(t)
	f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	sent := 0
	f.rpc.On("sendTransaction", func(p []any) (any, error) {
		sent++
		tx, _ := solana.DecodeTransactionBase64(solana.FirstParamString(p))
		return tx.Signature(), nil
	})
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	// Another worker already persisted attempt 2 (its persist bumps the sweep's version).
	must(t, f.db.Create(&models.SolanaSweepAttempt{SweepID: 1, AttemptNo: 2, Signature: "OTHER", Blockhash: "x", LastValidBlockHeight: 900, Status: models.SolanaSweepAttemptSigned}).Error)
	sweep := models.Sweep{}
	must(t, f.db.First(&sweep, 1).Error)
	must(t, f.db.Model(&models.Sweep{}).Where("id = ?", 1).Update("version", sweep.Version+1).Error)
	f.unknownEverywhere()
	f.rpc.Result("getBlockHeight", 600)
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "7pWqF1vXjQ2nD4sT8kL6mB3cR5yH9wE2aG7uN1xP4zV8", "lastValidBlockHeight": 750}))
	att := f.attempts(1)
	if err := f.svc.rebuildOrFail(ctx, &sweep, att[:1], 0); err == nil || sent != 1 {
		t.Fatalf("duplicate attempt number sent: err=%v sends=%d", err, sent)
	}
	must(t, f.sweepSvc.MarkFailed(1))
	if err := f.svc.rebuildOrFail(ctx, &sweep, f.attempts(1)[:1], 0); err == nil || sent != 1 || f.sweepStatus(1) != SweepStatusFailed {
		t.Fatalf("failed sweep revived by a rebuild: err=%v sends=%d status=%s", err, sent, f.sweepStatus(1))
	}
}
