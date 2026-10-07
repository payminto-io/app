package ledger

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/shopspring/decimal"
)

// randomJournal builds a balanced journal over 1-3 assets with integer amounts (exact in SQLite, see newTestService).
func randomJournal(rng *rand.Rand, n int) Journal {
	assets := []string{"USDC", "SOL", "BTC", "EUR"}
	owners := []AccountKey{
		{OwnerType: OwnerPlatform, OwnerID: "hot", Kind: KindAsset},
		{OwnerType: OwnerPlatform, OwnerID: "cold", Kind: KindAsset},
		{OwnerType: OwnerMember, OwnerID: "m1", Kind: KindLiability},
		{OwnerType: OwnerMember, OwnerID: "m2", Kind: KindLiability},
		{OwnerType: OwnerFees, OwnerID: "platform", Kind: KindIncome},
		{OwnerType: OwnerConnector, OwnerID: "stripe", Kind: KindAsset},
		{OwnerType: OwnerReserve, OwnerID: "chargeback", Kind: KindLiability},
	}
	j := Journal{
		Kind:           journalKinds[rng.IntN(len(journalKinds))],
		Reference:      Reference{Type: "prop", ID: fmt.Sprint(n)},
		IdempotencyKey: fmt.Sprintf("prop:%d", n),
	}
	for range 1 + rng.IntN(3) {
		asset := assets[rng.IntN(len(assets))]
		var sum decimal.Decimal
		for range 1 + rng.IntN(3) {
			amt := decimal.NewFromInt(int64(1 + rng.IntN(1_000_000)))
			if rng.IntN(2) == 0 {
				amt = amt.Neg()
			}
			acct := owners[rng.IntN(len(owners))]
			acct.Asset = asset
			j.Lines = append(j.Lines, Line{Account: acct, Amount: amt})
			sum = sum.Add(amt)
		}
		if !sum.IsZero() {
			acct := owners[rng.IntN(len(owners))]
			acct.Asset = asset
			j.Lines = append(j.Lines, Line{Account: acct, Amount: sum.Neg()})
		}
	}
	return j
}

func TestProperty_RandomJournalsKeepEveryAssetAtZeroAndBalancesReplayable(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	const seed = 20261007
	rng := rand.New(rand.NewPCG(seed, 1))

	expected := map[AccountKey]decimal.Decimal{}
	ids := map[string]JournalID{}
	for n := range 300 {
		j := randomJournal(rng, n)
		if len(j.Lines) < 2 {
			// A single self-cancelling line cannot exist; the generator always adds a balancing line.
			t.Fatalf("generator produced %d lines", len(j.Lines))
		}
		id, err := s.Post(ctx, j)
		if err != nil {
			t.Fatalf("seed %d journal %d: %v", seed, n, err)
		}
		ids[j.IdempotencyKey] = id
		for _, l := range j.Lines {
			expected[l.Account] = expected[l.Account].Add(l.Amount)
		}
		if rng.IntN(5) == 0 {
			again, err := s.Post(ctx, j)
			if err != nil || again != id {
				t.Fatalf("seed %d journal %d replay = (%d, %v), want (%d, nil)", seed, n, again, err, id)
			}
		}
	}

	var totals []sumRow
	if err := s.db.Model(&LineRow{}).Select("asset, COALESCE(SUM(amount), 0) AS total").Group("asset").Scan(&totals).Error; err != nil {
		t.Fatal(err)
	}
	if len(totals) == 0 {
		t.Fatal("no lines posted")
	}
	for _, r := range totals {
		if !r.Total.IsZero() {
			t.Errorf("asset %s sums to %s globally, want 0", r.Asset, r.Total)
		}
	}

	for acct, want := range expected {
		id, err := s.AccountID(ctx, acct)
		if err != nil {
			t.Fatalf("account %+v: %v", acct, err)
		}
		got, err := s.Balance(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Equal(want) {
			t.Errorf("account %+v derived %s, replayed %s", acct, got, want)
		}
	}

	var journals int64
	s.db.Model(&JournalRow{}).Count(&journals)
	if journals != int64(len(ids)) {
		t.Fatalf("journals = %d, want %d (replays must not create journals)", journals, len(ids))
	}
}
