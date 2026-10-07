package ledger

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/shopspring/decimal"
)

var propertyAssets = []string{"USDC", "SOL", "BTC", "EUR"}

var propertyOwners = []AccountKey{
	{OwnerType: OwnerPlatform, OwnerID: "hot", Kind: KindAsset},
	{OwnerType: OwnerPlatform, OwnerID: "cold", Kind: KindAsset},
	{OwnerType: OwnerMember, OwnerID: "m1", Kind: KindLiability},
	{OwnerType: OwnerMember, OwnerID: "m2", Kind: KindLiability},
	{OwnerType: OwnerFees, OwnerID: "platform", Kind: KindIncome},
	{OwnerType: OwnerConnector, OwnerID: "stripe", Kind: KindAsset},
	{OwnerType: OwnerReserve, OwnerID: "chargeback", Kind: KindLiability},
}

// randomAmount has up to maxScale decimal places and fewer than 20 integer digits.
func randomAmount(rng *rand.Rand, maxScale int) decimal.Decimal {
	scale := rng.IntN(maxScale + 1)
	digits := int64(1 + rng.IntN(1_000_000_000))
	amt := decimal.New(digits, -int32(scale))
	if rng.IntN(2) == 0 {
		amt = amt.Neg()
	}
	return amt
}

// randomJournal builds a balanced journal over 1-3 assets.
func randomJournal(rng *rand.Rand, n int, maxScale int) Journal {
	j := Journal{
		Kind:           journalKinds[rng.IntN(len(journalKinds))],
		Reference:      Reference{Type: "prop", ID: fmt.Sprint(n)},
		IdempotencyKey: fmt.Sprintf("prop:%d", n),
	}
	for range 1 + rng.IntN(3) {
		asset := propertyAssets[rng.IntN(len(propertyAssets))]
		var sum decimal.Decimal
		for range 1 + rng.IntN(3) {
			amt := randomAmount(rng, maxScale)
			acct := propertyOwners[rng.IntN(len(propertyOwners))]
			acct.Asset = asset
			j.Lines = append(j.Lines, Line{Account: acct, Amount: amt})
			sum = sum.Add(amt)
		}
		if !sum.IsZero() {
			acct := propertyOwners[rng.IntN(len(propertyOwners))]
			acct.Asset = asset
			j.Lines = append(j.Lines, Line{Account: acct, Amount: sum.Neg()})
		}
	}
	return j
}

// corrupt turns a balanced journal into one the ledger must refuse, returning the expected error.
func corrupt(rng *rand.Rand, j Journal) (Journal, error) {
	i := rng.IntN(len(j.Lines))
	switch rng.IntN(3) {
	case 0:
		j.Lines[i].Amount = j.Lines[i].Amount.Add(decimal.New(1, -18))
		return j, ErrUnbalanced
	case 1:
		extra := decimal.New(1, -19)
		j.Lines[i].Amount = j.Lines[i].Amount.Add(extra)
		j.Lines = append(j.Lines, Line{Account: j.Lines[i].Account, Amount: extra.Neg()})
		return j, ErrScale
	default:
		j.Lines[i].Amount = decimal.Zero
		return j, ErrZeroAmount
	}
}

// RunPropertyTest is shared by the SQLite unit run (integer amounts, see newTestService) and
// the Postgres integration run (full 18-decimal amounts).
func RunPropertyTest(t *testing.T, s *Service, maxScale int) {
	t.Helper()
	ctx := context.Background()
	const seed = 20261007
	rng := rand.New(rand.NewPCG(seed, uint64(maxScale)))

	expected := map[AccountKey]decimal.Decimal{}
	ids := map[string]JournalID{}
	rejected := 0
	for n := range 300 {
		j := randomJournal(rng, n, maxScale)
		if rng.IntN(4) == 0 {
			bad, want := corrupt(rng, j)
			if _, err := s.Post(ctx, bad); !errors.Is(err, want) {
				t.Fatalf("seed %d journal %d: corrupt post = %v, want %v", seed, n, err, want)
			}
			rejected++
			continue
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
	if rejected == 0 {
		t.Fatal("generator produced no invalid journals")
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
		t.Fatalf("journals = %d, want %d (replays and rejections must not create journals)", journals, len(ids))
	}
}

func TestProperty_RandomJournalsKeepEveryAssetAtZeroAndBalancesReplayable(t *testing.T) {
	RunPropertyTest(t, newTestService(t), 0)
}
