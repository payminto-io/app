package cre

import (
	"context"
	"testing"
	"time"
)

type fakeLiabilities struct {
	totals []LedgerTotal
	max    uint64
}

func (f fakeLiabilities) LiabilityTotals(context.Context) ([]LedgerTotal, uint64, error) {
	return f.totals, f.max, nil
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
	if len(skipped) != 1 || skipped[0] != "MYSTERY" {
		t.Fatalf("skipped = %v", skipped)
	}
	if len(cp.Assets) != 3 || cp.Assets[0].Asset != "SOL" || cp.Assets[1].Asset != "USDC.SOLANA" || cp.Assets[1].Liabilities.String() != "1250500000" || cp.Assets[1].Decimals != 6 {
		t.Fatalf("assets = %+v", cp.Assets)
	}
	if cp.Assets[0].Liabilities.Sign() != 0 {
		t.Fatalf("negative liability must clamp to zero, got %s", cp.Assets[0].Liabilities)
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
