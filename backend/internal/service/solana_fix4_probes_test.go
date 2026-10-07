package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

func (f *sweepFixture) probeBroadcastDefaults(balance *string) {
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": *balance, "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		return solana.ContextValue(1, make([]any, len(sigs))), nil
	})
	f.rpc.Result("getTransaction", nil)
	f.rpc.Result("getBlockHeight", 100000)
	f.rpc.On("getSignaturesForAddress", func([]any) (any, error) { return []any{}, nil })
}

// Reviewer's round-3 probes (re-review 3), ported unchanged.

// R3-1: ClaimForSweep commits deposits as swept before the rows transaction; a crash in between
// (the window spans the account read and a finalized balance RPC per account) leaves no sweep.
func TestProbe3_CrashAfterClaimBeforeRows(t *testing.T) {
	f := newSweepFixture(t)
	_, _, deps := f.newOwner("25")
	balance := "25000000"
	f.probeBroadcastDefaults(&balance)
	f.rpc.Result("sendTransaction", "SIGX")
	if n, err := f.deposits.ClaimForSweep(deps[0].ID); err != nil || n != 1 {
		t.Fatalf("claim: %d %v", n, err)
	}
	// process dies here; restart and run for two days
	ctx := context.Background()
	f.svc.now = func() time.Time { return f.now.Add(48 * time.Hour) }
	for i := 0; i < 5; i++ {
		f.svc.SweepConfirmed(ctx)
		f.svc.TrackConfirmations(ctx)
	}
	var sweeps, links int64
	f.db.Model(&models.Sweep{}).Count(&sweeps)
	f.db.Model(&models.SolanaSweepDeposit{}).Count(&links)
	missed, _ := f.missed.ListUnresolved()
	t.Logf("after restart and 48h: deposit=%s sweeps=%d links=%d anomalies=%d balance still %s", f.depositStatus(deps[0].ID), sweeps, links, len(missed), balance)
	if f.depositStatus(deps[0].ID) == models.DepositStatusSwept && sweeps == 0 {
		t.Fatalf("ORPHANED CLAIM: deposit %d is swept with no sweep, no link, no anomaly; 25 USDC stays in the ATA and nothing ever picks it up", deps[0].ID)
	}
}

// R3-2: resolveNoAttempt decides "no attempt" from a read, then failSweep unconditionally; a slow
// signer that persists its attempt in between has its signed attempt marked expired without evidence
// and then sends it. The landed transaction belongs to a failed sweep while a second sweep holds the lock.
func TestProbe3_NoAttemptFailRacesALateSigner(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	sweep := f.processingSweepRows(ata, deps[0]) // aged past ValidityWindow (worker A is a slow signer)
	persisted := false
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		if !persisted {
			persisted = true
			// Worker A's signAndSend persist transaction commits here, between B's attempt read and B's failSweep.
			must(t, f.db.Transaction(func(tx *gorm.DB) error {
				// Round 4: the signer's persist is a compare-and-set on (status, version) and bumps the version.
				res := tx.Model(&models.Sweep{}).Where("id = ? AND status = ? AND version = ?", sweep.ID, SweepStatusProcessing, 0).
					Updates(map[string]any{"status": SweepStatusPending, "version": gorm.Expr("version + 1")})
				if res.Error != nil || res.RowsAffected != 1 {
					return errors.New("not in flight")
				}
				return tx.Create(&models.SolanaSweepAttempt{SweepID: sweep.ID, AttemptNo: 1, Signature: "SIGA", Blockhash: "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", LastValidBlockHeight: 500, Status: models.SolanaSweepAttemptSigned}).Error
			}))
		}
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	ctx := context.Background()
	f.svc.TrackConfirmations(ctx) // worker B
	a := f.attempts(sweep.ID)
	t.Logf("after B's pass (A now sends SIGA, which it persisted): sweep1=%s attempt=%s deposit=%s", f.sweepStatus(sweep.ID), a[0].Status, f.depositStatus(deps[0].ID))
	// SIGA is in flight on the chain. Next round re-sweeps the same claim.
	f.balanceAlways("25000000")
	f.blockhash(600)
	f.rpc.Result("sendTransaction", "SIGB")
	n, _ := f.svc.SweepConfirmed(ctx)
	t.Logf("next round: second sweep started=%d (two live transactions for one 25 USDC claim)", n)
	// SIGA lands (account drained to 0); SIGB fails on chain (insufficient funds).
	f.balanceAlways("0")
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		sigs, _ := p[0].([]string)
		out := make([]any, len(sigs))
		for i, s := range sigs {
			switch s {
			case "SIGB":
				out[i] = map[string]any{"slot": 701, "confirmations": nil, "err": map[string]any{"InstructionError": []any{2, map[string]any{"Custom": 1}}}, "confirmationStatus": "finalized"}
			case "SIGA":
				out[i] = map[string]any{"slot": 700, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"}
			}
		}
		return solana.ContextValue(1, out), nil
	})
	f.history("SIGA")
	f.rpc.Result("getBlockHeight", 1000)
	f.finalizedTx("SIGA", ata, "25000000")
	f.svc.TrackConfirmations(ctx)
	f.svc.now = func() time.Time { return f.now.Add(3 * time.Hour) }
	for i := 0; i < 5; i++ {
		f.svc.TrackConfirmations(ctx)
		f.svc.SweepConfirmed(ctx)
	}
	missed, _ := f.missed.ListUnresolved()
	reasons := []string{}
	for _, m := range missed {
		reasons = append(reasons, m.Reason)
	}
	var journals int64
	f.db.Table("ledger_journals").Where("reference_type = ?", "sweep").Count(&journals)
	var locks []models.SolanaSweepLock
	f.db.Find(&locks)
	status := func(id uint) string {
		var sw models.Sweep
		if f.db.First(&sw, id).Error != nil {
			return "none"
		}
		return sw.Status
	}
	t.Logf("3h later: sweep1=%s sweep2=%s deposit=%s sweep journals=%d locks=%+v anomalies=%v", status(sweep.ID), status(sweep.ID+1), f.depositStatus(deps[0].ID), journals, locks, reasons)
	if status(sweep.ID) != SweepStatusCompleted && status(sweep.ID+1) != SweepStatusCompleted {
		t.Fatalf("LANDED SWEEP NEVER BOOKED: SIGA moved 25 USDC to the hot wallet, sweep 1 is %s, sweep 2 is %s holding the lock, deposit %s", status(sweep.ID), status(sweep.ID+1), f.depositStatus(deps[0].ID))
	}
}
