package service

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/shopspring/decimal"
)

// secondWorker is another sweeper process on the same database and RPC, with its own clock.
func (f *sweepFixture) secondWorker(offset time.Duration) *SolanaSweepService {
	w := *f.svc
	w.now = func() time.Time { return f.now.Add(offset) }
	return &w
}

func (f *sweepFixture) sweepJournals() int64 {
	var n int64
	f.db.Table("ledger_journals").Where("reference_type = ?", "sweep").Count(&n)
	return n
}

// R3-I2: worker A is signing slowly while worker B, whose clock is past the validity window, resolves
// the attempt-less sweep and fails it. A's persist then loses its compare-and-set and sends nothing; the
// next round sweeps the deposit once, it books once, and no lock is left behind.
func TestSolanaSweep_TwoWorkersSlowSignerSendExactlyOnce(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	b := f.secondWorker(time.Hour)
	ctx := context.Background()
	slow := true
	f.rpc.On("getLatestBlockhash", func([]any) (any, error) {
		if slow {
			slow = false
			if _, err := b.TrackConfirmations(ctx); err != nil {
				t.Fatal(err)
			}
		}
		return solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}), nil
	})
	var sent []string
	f.rpc.On("sendTransaction", func(p []any) (any, error) {
		tx, _ := solana.DecodeTransactionBase64(solana.FirstParamString(p))
		sent = append(sent, tx.Signature())
		return tx.Signature(), nil
	})
	if n, _ := f.svc.SweepConfirmed(ctx); n != 0 || len(sent) != 0 {
		t.Fatalf("the late signer sent: started=%d sends=%d", n, len(sent))
	}
	if f.sweepStatus(1) != SweepStatusFailed || len(f.attempts(1)) != 0 || f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("sweep 1=%s attempts=%d deposit=%s", f.sweepStatus(1), len(f.attempts(1)), f.depositStatus(deps[0].ID))
	}
	// Next round: one sweep, one transfer.
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 || len(sent) != 1 {
		t.Fatalf("re-sweep: started=%d sends=%d", n, len(sent))
	}
	f.finalizedEverywhere()
	f.finalizedTx(sent[0], ata, "25000000")
	f.rpc.Result("getAccountInfo", solana.ContextValue(1, nil))
	if done, err := f.svc.TrackConfirmations(ctx); err != nil || done != 1 {
		t.Fatalf("track: %d %v", done, err)
	}
	b.TrackConfirmations(ctx)
	var locks int64
	f.db.Model(&models.SolanaSweepLock{}).Count(&locks)
	if f.sweepStatus(2) != SweepStatusCompleted || f.depositStatus(deps[0].ID) != models.DepositStatusSwept || f.sweepJournals() != 1 || locks != 0 || len(sent) != 1 {
		t.Fatalf("sweep 2=%s deposit=%s journals=%d locks=%d sends=%d", f.sweepStatus(2), f.depositStatus(deps[0].ID), f.sweepJournals(), locks, len(sent))
	}
}

// R3-M1: expiry needs two endpoints whose own finalized height is past the attempt's validity; a lagging
// endpoint's null is not evidence, so nothing is rebuilt until a second endpoint has caught up.
func TestSolanaSweep_ExpiryEvidenceIsPerEndpoint(t *testing.T) {
	f := newSweepFixture(t)
	f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	f.rpc.Result("sendTransaction", "SIGE")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	f.unknownEverywhere()
	f.rpc.Result("getBlockHeight", 600)
	lagging := true
	f.rpc.OnNode = func(node uint, method string, _ []any) (any, error, bool) {
		if node == 2 && method == "getBlockHeight" && lagging {
			return 450, nil, true
		}
		return nil, nil, false
	}
	f.svc.TrackConfirmations(ctx)
	if att := f.attempts(1); len(att) != 1 || att[0].Status == models.SolanaSweepAttemptExpired {
		t.Fatalf("expired on one endpoint's evidence: %+v", att)
	}
	lagging = false
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "7pWqF1vXjQ2nD4sT8kL6mB3cR5yH9wE2aG7uN1xP4zV8", "lastValidBlockHeight": 750}))
	f.rpc.Result("sendTransaction", "SIGE2")
	f.svc.TrackConfirmations(ctx)
	if att := f.attempts(1); len(att) != 2 || att[0].Status != models.SolanaSweepAttemptExpired {
		t.Fatalf("not rebuilt once both endpoints evidence it: %+v", att)
	}
}

// R3-I2, revive versus lock: a failed sweep whose transaction the chain executed takes the lock over from
// the successor (which has no live attempt), the successor fails, and the landed sweep books once.
func TestSolanaSweep_RevivedSweepTakesOverTheLock(t *testing.T) {
	f := newSweepFixture(t)
	f.svc.cfg.MaxAttempts = 1
	_, ata, deps := f.newOwner("25")
	f.balanceAlways("25000000")
	f.blockhash(500)
	f.rpc.Result("sendTransaction", "SIG1")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	// Sweep 1 is failed with evidence (as if misjudged) and sweep 2 takes the account.
	f.unknownEverywhere()
	f.rpc.Result("getBlockHeight", 600)
	f.svc.TrackConfirmations(ctx)
	if f.sweepStatus(1) != SweepStatusFailed {
		t.Fatalf("sweep 1 = %s", f.sweepStatus(1))
	}
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "7pWqF1vXjQ2nD4sT8kL6mB3cR5yH9wE2aG7uN1xP4zV8", "lastValidBlockHeight": 700}))
	f.rpc.Result("sendTransaction", "SIG2")
	if n, _ := f.svc.SweepConfirmed(ctx); n != 1 {
		t.Fatal("sweep 2 not started")
	}
	// SIG1 lands after all; SIG2 fails on chain; the account is empty.
	sig1, sig2 := f.sig("SIG1"), f.sig("SIG2")
	f.balanceAlways("0")
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		out := make([]any, len(sigs))
		for i, s := range sigs {
			switch s {
			case "SIG2":
				out[i] = map[string]any{"slot": 701, "confirmations": nil, "err": map[string]any{"InstructionError": []any{2, map[string]any{"Custom": 1}}}, "confirmationStatus": "finalized"}
			case "SIG1":
				out[i] = map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"}
			}
		}
		return solana.ContextValue(1, out), nil
	})
	f.history("SIG1")
	f.finalizedTx("SIG1", ata, "25000000")
	f.rpc.Result("getAccountInfo", solana.ContextValue(1, nil))
	f.svc.now = func() time.Time { return f.now.Add(2 * time.Hour) }
	for i := 0; i < 3; i++ {
		f.svc.TrackConfirmations(ctx)
		f.svc.SweepConfirmed(ctx)
	}
	var locks int64
	f.db.Model(&models.SolanaSweepLock{}).Count(&locks)
	if f.sweepStatus(1) != SweepStatusCompleted || f.sweepStatus(2) != SweepStatusFailed || f.depositStatus(deps[0].ID) != models.DepositStatusSwept || f.sweepJournals() != 1 || locks != 0 {
		t.Fatalf("sweep1=%s sweep2=%s deposit=%s journals=%d locks=%d (%s %s)", f.sweepStatus(1), f.sweepStatus(2), f.depositStatus(deps[0].ID), f.sweepJournals(), locks, sig1[:6], sig2[:6])
	}
}

// R3-L1: a drain explained only by an untracked transaction of ours sets the account aside once, instead
// of re-reading its history every tick.
func TestSolanaSweep_UntrackedDrainSetsTheAccountAside(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, _ := f.newOwner("25")
	f.balanceAlways("0")
	f.history("OURS")
	f.finalizedTx("OURS", ata, "25000000")
	ctx := context.Background()
	f.svc.SweepConfirmed(ctx)
	acct, _ := f.accounts.GetByTokenAccount(ata)
	missed, _ := f.missed.ListUnresolved()
	if acct.Status != models.SolanaDepositAccountDrained || len(missed) != 1 || !strings.HasPrefix(missed[0].Reason, anomalyUntrackedSweep) {
		t.Fatalf("account=%s missed=%+v", acct.Status, missed)
	}
	before := f.rpc.Count("getSignaturesForAddress")
	for i := 0; i < 3; i++ {
		f.svc.SweepConfirmed(ctx)
	}
	if f.rpc.Count("getSignaturesForAddress") != before {
		t.Fatal("history re-read for an account set aside")
	}
}

// R3-L3: a legacy payment whose merchant invoice id names a switch attempt it does not belong to still
// gets its one watcher journal.
func TestSolanaDeposit_InvoiceIDMatchingAnotherAttemptIsNotSwitchOwned(t *testing.T) {
	f := newSolanaFixture(t)
	must(t, paymentswitch.Migrate(f.db))
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	other := "the-other-payment-reference"
	now := time.Now()
	must(t, f.db.Create(&paymentswitch.AttemptRow{ID: "pa_copied", IntentID: "pi_1", MerchantID: strconv.FormatUint(uint64(f.member.ID), 10), ConnectorCode: "chaindeposit",
		Status: paymentswitch.AttemptPending, Amount: decimal.NewFromInt(25), Asset: "USD", ConnectorTransactionID: &other, StatusChangedAt: now, CreatedAt: now, UpdatedAt: now}).Error)
	must(t, f.db.Model(&models.PaymentRequest{}).Where("id = ?", pr.ID).Update("invoice_id", "pa_copied").Error)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	ctx := context.Background()
	f.mustPoll(ctx)
	f.statuses(map[string]any{"slot": 250000123, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	if f.mustConfirm(ctx) != 1 {
		t.Fatal("not finalized")
	}
	var journals int64
	f.db.Table("ledger_journals").Where("kind = ?", "payment").Count(&journals)
	if journals != 1 {
		t.Fatalf("payment journals = %d, want one from the watcher", journals)
	}
}
