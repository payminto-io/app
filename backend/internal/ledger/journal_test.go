package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func key(owner OwnerType, id, asset string, kind AccountKind) AccountKey {
	return AccountKey{OwnerType: owner, OwnerID: id, Asset: asset, Kind: kind}
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func balancedJournal() Journal {
	return Journal{
		Kind:           KindPayment,
		Reference:      Reference{Type: "payment", ID: "p1"},
		IdempotencyKey: "payment:p1",
		Lines: []Line{
			{Account: key(OwnerPlatform, "hot", "USDC", KindAsset), Amount: dec("10")},
			{Account: key(OwnerMember, "m1", "USDC", KindLiability), Amount: dec("-10")},
		},
	}
}

func TestValidate_BalancedJournalPasses(t *testing.T) {
	if err := balancedJournal().Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidate_UnbalancedJournalRejected(t *testing.T) {
	j := balancedJournal()
	j.Lines[1].Amount = dec("-9.5")
	err := j.Validate()
	if !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("Validate() = %v, want ErrUnbalanced", err)
	}
}

func TestValidate_MixedAssetsBalancePerAsset(t *testing.T) {
	j := Journal{
		Kind:           KindConversion,
		IdempotencyKey: "conv:1",
		Lines: []Line{
			{Account: key(OwnerMember, "m1", "USDC", KindLiability), Amount: dec("100")},
			{Account: key(OwnerPlatform, "trade", "USDC", KindAsset), Amount: dec("-100")},
			{Account: key(OwnerPlatform, "trade", "SOL", KindAsset), Amount: dec("0.5")},
			{Account: key(OwnerMember, "m1", "SOL", KindLiability), Amount: dec("-0.5")},
		},
	}
	if err := j.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	// Totals cancel across assets but not within one: must be rejected.
	cross := Journal{
		Kind:           KindConversion,
		IdempotencyKey: "conv:2",
		Lines: []Line{
			{Account: key(OwnerMember, "m1", "USDC", KindLiability), Amount: dec("1")},
			{Account: key(OwnerMember, "m1", "SOL", KindLiability), Amount: dec("-1")},
		},
	}
	if err := cross.Validate(); !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("Validate() = %v, want ErrUnbalanced", err)
	}
}

func TestValidate_ZeroAmountLineRejected(t *testing.T) {
	j := balancedJournal()
	j.Lines = append(j.Lines, Line{Account: key(OwnerFees, "platform", "USDC", KindIncome), Amount: decimal.Zero})
	if err := j.Validate(); !errors.Is(err, ErrZeroAmount) {
		t.Fatalf("Validate() = %v, want ErrZeroAmount", err)
	}
}

func TestValidate_EmptyJournalRejected(t *testing.T) {
	j := balancedJournal()
	j.Lines = nil
	if err := j.Validate(); !errors.Is(err, ErrEmptyJournal) {
		t.Fatalf("Validate() = %v, want ErrEmptyJournal", err)
	}
}

func TestValidate_RejectsBadEnumsAndKeys(t *testing.T) {
	cases := map[string]func(*Journal){
		"unknown kind":         func(j *Journal) { j.Kind = "bogus" },
		"missing key":          func(j *Journal) { j.IdempotencyKey = "" },
		"unknown owner type":   func(j *Journal) { j.Lines[0].Account.OwnerType = "alien" },
		"unknown account kind": func(j *Journal) { j.Lines[0].Account.Kind = "equity" },
		"empty asset":          func(j *Journal) { j.Lines[0].Account.Asset = ""; j.Lines[1].Account.Asset = "" },
		"empty owner id":       func(j *Journal) { j.Lines[0].Account.OwnerID = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			j := balancedJournal()
			mutate(&j)
			if err := j.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate() = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestRequestHash_IsCanonical(t *testing.T) {
	a := balancedJournal()
	b := balancedJournal()
	b.Lines[0], b.Lines[1] = b.Lines[1], b.Lines[0]
	b.Lines[0].Amount = dec("-10.000")
	if a.legacyRequestHash() != b.legacyRequestHash() {
		t.Fatal("line order and decimal formatting must not change the request hash")
	}
	c := balancedJournal()
	c.Lines[0].Amount = dec("11")
	c.Lines[1].Amount = dec("-11")
	if a.legacyRequestHash() == c.legacyRequestHash() {
		t.Fatal("different amounts must produce a different request hash")
	}
}

func TestValidate_RejectsMoreThanEighteenDecimals(t *testing.T) {
	j := balancedJournal()
	j.Lines[0].Amount = dec("0.0000000000000000015")
	j.Lines[1].Amount = dec("-0.0000000000000000015")
	if err := j.Validate(); !errors.Is(err, ErrScale) {
		t.Fatalf("Validate() = %v, want ErrScale", err)
	}
	ok := balancedJournal()
	ok.Lines[0].Amount = dec("1.000000000000000000000")
	ok.Lines[1].Amount = dec("-1.000000000000000000000")
	if err := ok.Validate(); err != nil {
		t.Fatalf("trailing zeros beyond 18 places are not precision: %v", err)
	}
}

func TestValidate_RejectsMagnitudeAtOrAbove1e20(t *testing.T) {
	j := balancedJournal()
	j.Lines[0].Amount = dec("100000000000000000000")
	j.Lines[1].Amount = dec("-100000000000000000000")
	if err := j.Validate(); !errors.Is(err, ErrMagnitude) {
		t.Fatalf("Validate() = %v, want ErrMagnitude", err)
	}
	j.Lines[0].Amount = dec("99999999999999999999.999999999999999999")
	j.Lines[1].Amount = j.Lines[0].Amount.Neg()
	if err := j.Validate(); err != nil {
		t.Fatalf("largest representable amount rejected: %v", err)
	}
}

func TestRequestHash_IncludesExplicitPostedAt(t *testing.T) {
	a := balancedJournal()
	b := balancedJournal()
	if a.legacyRequestHash() != b.legacyRequestHash() {
		t.Fatal("zero PostedAt must hash the same")
	}
	b.PostedAt = time.Now()
	if a.legacyRequestHash() == b.legacyRequestHash() {
		t.Fatal("an explicit PostedAt must change the hash")
	}
	c := balancedJournal()
	c.PostedAt = b.PostedAt.In(time.FixedZone("x", 3600)).Add(500 * time.Nanosecond)
	if b.legacyRequestHash() != c.legacyRequestHash() {
		t.Fatal("zone and sub-microsecond differences must not change the hash")
	}
}

func TestValidate_AssetCodeFormatAndLength(t *testing.T) {
	ok := []string{"USDC", "USDC.BASE", "USDC.E.AVALANCHE", "WSTETH.ARBITRUM", "USDT.AVALANCHE_C", "USD"}
	for _, asset := range ok {
		j := balancedJournal()
		j.Lines[0].Account.Asset, j.Lines[1].Account.Asset = asset, asset
		if err := j.Validate(); err != nil {
			t.Errorf("asset %q rejected: %v", asset, err)
		}
	}
	bad := []string{"usdc", ".USDC", "USDC.", "USDC..BASE", "US DC", "USDC/BASE", "A2345678901234567890123456789012X"}
	for _, asset := range bad {
		j := balancedJournal()
		j.Lines[0].Account.Asset, j.Lines[1].Account.Asset = asset, asset
		if err := j.Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("asset %q accepted", asset)
		}
	}
}
