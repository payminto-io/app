package service

import (
	"context"
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// flakyResolver fails the first n calls, then resolves like the fixture.
type flakyResolver struct {
	failures int
	inner    AssetResolver
}

func (r *flakyResolver) resolve(tx *gorm.DB, id uint) (Assets, error) {
	if r.failures > 0 {
		r.failures--
		return Assets{}, errors.New("transient resolver failure")
	}
	return r.inner(tx, id)
}

func setupJournalingConfirmer(t *testing.T, failures int) (*EVMSweepConfirmer, *callerFixture, *SweepService) {
	t.Helper()
	f := newCallerFixture(t)
	flaky := &flakyResolver{failures: failures, inner: blockchainCurrencyAssetResolver()}
	ledgerSvc := NewLedgerService(repository.NewAccountRepository(f.db), WithJournal(f.journal, flaky.resolve))
	stRepo := repository.NewSweepTransactionRepository(f.db)
	sweepSvc := NewSweepService(f.db, repository.NewSweepRepository(f.db), stRepo, repository.NewBlockchainRepository(f.db), ledgerSvc)
	c := NewEVMSweepConfirmer(stRepo, repository.NewBlockchainCurrencyRepository(f.db), sweepSvc, fakeChecker{confs: 200})
	return c, f, sweepSvc
}

func seedBaseSweepTx(t *testing.T, f *callerFixture, status string) (sweepTxID, sweepID uint) {
	t.Helper()
	var base models.Blockchain
	if err := f.db.Where("code = ?", "BASE").First(&base).Error; err != nil {
		t.Fatal(err)
	}
	sweep := models.Sweep{Status: SweepStatusPending, BlockchainID: base.ID}
	if err := f.db.Create(&sweep).Error; err != nil {
		t.Fatal(err)
	}
	st := models.SweepTransaction{
		TxHash: "0xsweep", Amount: decimal.NewFromInt(100), GasFee: decimal.RequireFromString("0.002"),
		FromAddress: "0xfrom", ToAddress: "0xcold", Status: status, SweepID: sweep.ID, BlockchainCurrencyID: f.usdcBase,
	}
	if err := f.db.Create(&st).Error; err != nil {
		t.Fatal(err)
	}
	return st.ID, sweep.ID
}

func TestSweepConfirmer_FailedPostIsRetriedNextRound(t *testing.T) {
	c, f, _ := setupJournalingConfirmer(t, 1)
	stID, sweepID := seedBaseSweepTx(t, f, SweepTxStatusBroadcast)
	ctx := context.Background()

	if n, _ := c.TrackConfirmations(ctx); n != 0 {
		t.Fatalf("round 1 confirmed = %d, want 0 (post failed)", n)
	}
	var st models.SweepTransaction
	f.db.First(&st, stID)
	var sweep models.Sweep
	f.db.First(&sweep, sweepID)
	if st.Status == SweepTxStatusConfirmed || sweep.Status == SweepStatusCompleted {
		t.Fatalf("after a failed post: tx=%s sweep=%s; nothing may be marked", st.Status, sweep.Status)
	}
	if n := count(t, f.db, &ledger.JournalRow{}); n != 0 {
		t.Fatalf("journals = %d, want 0", n)
	}

	if n, _ := c.TrackConfirmations(ctx); n != 1 {
		t.Fatalf("round 2 confirmed = %d, want 1", n)
	}
	f.db.First(&st, stID)
	f.db.First(&sweep, sweepID)
	if st.Status != SweepTxStatusConfirmed || sweep.Status != SweepStatusCompleted {
		t.Fatalf("after recovery: tx=%s sweep=%s", st.Status, sweep.Status)
	}
	_, lines := journalLines(t, f.db, "payminto:sweep:"+idStr(sweepID))
	if len(lines) != 4 {
		t.Fatalf("journal lines = %d, want 4", len(lines))
	}
}

func TestSweepConfirmer_RepollsConfirmedTxsWhoseSweepIsNotCompleted(t *testing.T) {
	c, f, _ := setupJournalingConfirmer(t, 0)
	_, sweepID := seedBaseSweepTx(t, f, SweepTxStatusConfirmed)

	if n, err := c.TrackConfirmations(context.Background()); err != nil || n != 1 {
		t.Fatalf("TrackConfirmations = (%d, %v), want (1, nil)", n, err)
	}
	var sweep models.Sweep
	f.db.First(&sweep, sweepID)
	if sweep.Status != SweepStatusCompleted {
		t.Fatalf("sweep status = %s, want completed", sweep.Status)
	}
	if n := count(t, f.db, &ledger.JournalRow{}); n != 1 {
		t.Fatalf("journals = %d, want 1", n)
	}
	if n, _ := c.TrackConfirmations(context.Background()); n != 0 {
		t.Fatalf("second round = %d, want 0", n)
	}
}
