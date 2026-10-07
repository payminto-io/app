package cre

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeLiabilities struct {
	totals []LedgerTotal
	max    uint64
	at     time.Time
}

func (f fakeLiabilities) LiabilityTotals(context.Context) (LedgerSnapshot, error) {
	return LedgerSnapshot{Totals: f.totals, Head: f.max, TakenAt: f.at}, nil
}

func TestBuildCheckpointOmitsUnknownAssetsAndIsDeterministic(t *testing.T) {
	dec, err := ParseDecimals("XRP:6")
	if err != nil {
		t.Fatal(err)
	}
	src := fakeLiabilities{totals: []LedgerTotal{{"USDC.SOLANA", "1250.5"}, {"MYSTERY", "1"}, {"XRP", "2"}, {"SOL", "-3"}}, max: 77}
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	cp, skipped, err := BuildCheckpoint(context.Background(), src, dec, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 2 || skipped[0].Asset != "MYSTERY" || skipped[1].Asset != "SOL" || !strings.Contains(skipped[1].Reason, "negative") {
		t.Fatalf("skipped = %v", skipped)
	}
	// A negative total is an anomaly, never rewritten as zero and attested.
	if len(cp.Assets) != 2 || cp.Assets[0].Asset != "USDC.SOLANA" || cp.Assets[0].Liabilities.String() != "1250500000" || cp.Assets[0].Decimals != 6 || cp.Assets[1].Asset != "XRP" {
		t.Fatalf("assets = %+v", cp.Assets)
	}
	again, _, _ := BuildCheckpoint(context.Background(), src, dec, now)
	if again.Hash != cp.Hash || cp.MaxJournalID != 77 {
		t.Fatal("checkpoint hash must be a function of its facts")
	}
	src.max = 78
	changed, _, _ := BuildCheckpoint(context.Background(), src, dec, now)
	if changed.Hash == cp.Hash {
		t.Fatal("journal head must be part of the hash")
	}
}

func TestParseDecimalsRejectsBadEntries(t *testing.T) {
	for _, bad := range []string{"usdc", "XRP:19", "XRP:-1", "X:1", "XRP:abc"} {
		if _, err := ParseDecimals(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	d, _ := ParseDecimals("")
	if _, _, ok := d.Minor("USDC.BASE", "0.0000001"); ok {
		t.Error("amount finer than the grid accepted")
	}
}

// The checkpoint's taken_at is the ledger snapshot's own time, and the stored facts carry it as hashed.
func TestBuildCheckpointTakesTheSnapshotTime(t *testing.T) {
	at := time.Date(2026, 10, 7, 11, 59, 58, 123456000, time.UTC)
	src := fakeLiabilities{totals: []LedgerTotal{{"USDC.SOLANA", "1"}}, max: 5, at: at}
	cp, _, err := BuildCheckpoint(context.Background(), src, Decimals{"USDC": 6}, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !cp.TakenAt.Equal(at) || CheckpointSubject(cp).Facts["taken_at"] != "2026-10-07T11:59:58Z" || CheckpointSubject(cp).Facts["max_journal_id"] != uint64(5) {
		t.Fatalf("taken_at = %s facts %v", cp.TakenAt, CheckpointSubject(cp).Facts)
	}
}
