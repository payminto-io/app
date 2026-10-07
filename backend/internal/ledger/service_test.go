package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestService uses in-memory SQLite like the rest of the repo's unit tests.
// SQLite stores numeric as float, so unit tests use integer and dyadic amounts;
// 18-decimal exactness is proven on Postgres in integration_test.go.
func newTestService(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return New(db)
}

func mustPost(t *testing.T, s *Service, j Journal) JournalID {
	t.Helper()
	id, err := s.Post(context.Background(), j)
	if err != nil {
		t.Fatalf("Post(%s) = %v", j.IdempotencyKey, err)
	}
	return id
}

func TestPost_RejectsInvalidJournalBeforeTouchingTheDatabase(t *testing.T) {
	s := newTestService(t)
	j := balancedJournal()
	j.Lines[0].Amount = dec("1")
	if _, err := s.Post(context.Background(), j); !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("Post() = %v, want ErrUnbalanced", err)
	}
	var n int64
	s.db.Model(&JournalRow{}).Count(&n)
	if n != 0 {
		t.Fatalf("journals written = %d, want 0", n)
	}
}

func TestPost_SameKeyReturnsOriginalJournal(t *testing.T) {
	s := newTestService(t)
	j := balancedJournal()
	first := mustPost(t, s, j)
	again := balancedJournal()
	again.Lines[0], again.Lines[1] = again.Lines[1], again.Lines[0]
	second := mustPost(t, s, again)
	if first != second {
		t.Fatalf("replay returned %d, want original %d", second, first)
	}
	var lines int64
	s.db.Model(&LineRow{}).Count(&lines)
	if lines != 2 {
		t.Fatalf("lines = %d, want 2 (replay must not post again)", lines)
	}
}

func TestPost_SameKeyDifferentPayloadIsAnError(t *testing.T) {
	s := newTestService(t)
	mustPost(t, s, balancedJournal())
	changed := balancedJournal()
	changed.Lines[0].Amount = dec("11")
	changed.Lines[1].Amount = dec("-11")
	if _, err := s.Post(context.Background(), changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("Post() = %v, want ErrIdempotencyConflict", err)
	}
}

func TestPost_ReusesAccountsAcrossJournals(t *testing.T) {
	s := newTestService(t)
	mustPost(t, s, balancedJournal())
	j := balancedJournal()
	j.IdempotencyKey = "payment:p2"
	mustPost(t, s, j)
	var accounts int64
	s.db.Model(&AccountRow{}).Count(&accounts)
	if accounts != 2 {
		t.Fatalf("accounts = %d, want 2", accounts)
	}
}

func TestBalance_IsTheSignedSumOfLines(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	mustPost(t, s, balancedJournal())
	j := balancedJournal()
	j.IdempotencyKey = "payment:p2"
	j.Lines[0].Amount = dec("2.5")
	j.Lines[1].Amount = dec("-2.5")
	mustPost(t, s, j)

	hot, err := s.AccountID(ctx, key(OwnerPlatform, "hot", "USDC", KindAsset))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Balance(ctx, hot)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(dec("12.5")) {
		t.Fatalf("Balance(hot) = %s, want 12.5", got)
	}
	member, err := s.AccountID(ctx, key(OwnerMember, "m1", "USDC", KindLiability))
	if err != nil {
		t.Fatal(err)
	}
	got, err = s.Balance(ctx, member)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(dec("-12.5")) {
		t.Fatalf("Balance(member) = %s, want -12.5", got)
	}
	if _, err := s.Balance(ctx, 9999); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("Balance(unknown) = %v, want ErrAccountNotFound", err)
	}
}

func TestBalances_GroupsByAssetForOneOwner(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	mustPost(t, s, balancedJournal())
	mustPost(t, s, Journal{
		Kind:           KindPayment,
		IdempotencyKey: "payment:sol",
		Lines: []Line{
			{Account: key(OwnerPlatform, "hot", "SOL", KindAsset), Amount: dec("3")},
			{Account: key(OwnerMember, "m1", "SOL", KindLiability), Amount: dec("-3")},
		},
	})
	got, err := s.Balances(ctx, OwnerMember, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got["USDC"].Equal(dec("-10")) || !got["SOL"].Equal(dec("-3")) {
		t.Fatalf("Balances(m1) = %v", got)
	}
	none, err := s.Balances(ctx, OwnerMember, "nobody")
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("Balances(nobody) = %v, want empty", none)
	}
}

func TestStatement_RunningBalanceStartsFromOpeningBalance(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	post := func(k string, day int, amt string) {
		mustPost(t, s, Journal{
			Kind:           KindPayment,
			IdempotencyKey: k,
			PostedAt:       base.AddDate(0, 0, day),
			Lines: []Line{
				{Account: key(OwnerPlatform, "hot", "USDC", KindAsset), Amount: dec(amt)},
				{Account: key(OwnerMember, "m1", "USDC", KindLiability), Amount: dec(amt).Neg()},
			},
		})
	}
	post("a", 0, "10")
	post("b", 1, "5")
	post("c", 2, "-4")
	post("d", 3, "1")

	from, to := base.AddDate(0, 0, 1), base.AddDate(0, 0, 3)
	lines, err := s.Statement(ctx, OwnerMember, "m1", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("Statement returned %d lines, want 2 (window is [from, to))", len(lines))
	}
	if !lines[0].Amount.Equal(dec("-5")) || !lines[0].Running.Equal(dec("-15")) {
		t.Fatalf("line 0 = %+v, want amount -5 running -15", lines[0])
	}
	if !lines[1].Amount.Equal(dec("4")) || !lines[1].Running.Equal(dec("-11")) {
		t.Fatalf("line 1 = %+v, want amount 4 running -11", lines[1])
	}
	if lines[0].Asset != "USDC" || lines[0].AccountKind != KindLiability || lines[0].JournalID == 0 {
		t.Fatalf("line 0 missing identity: %+v", lines[0])
	}
}

func TestPost_StoresReferenceMetadataAndPostedAt(t *testing.T) {
	s := newTestService(t)
	j := balancedJournal()
	j.PostedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	j.Metadata = map[string]any{"tx_hash": "0xabc"}
	id := mustPost(t, s, j)
	var row JournalRow
	if err := s.db.First(&row, id).Error; err != nil {
		t.Fatal(err)
	}
	if row.ReferenceType != "payment" || row.ReferenceID != "p1" || !row.PostedAt.Equal(j.PostedAt) || row.Metadata["tx_hash"] != "0xabc" {
		t.Fatalf("journal row = %+v", row)
	}
	if row.RequestHash != j.requestHash() {
		t.Fatal("request hash not stored")
	}
}

func TestDecimalRoundTrip(t *testing.T) {
	s := newTestService(t)
	j := balancedJournal()
	j.Lines[0].Amount = decimal.NewFromInt(1 << 40)
	j.Lines[1].Amount = decimal.NewFromInt(1 << 40).Neg()
	id := mustPost(t, s, j)
	var row LineRow
	if err := s.db.Where("journal_id = ?", id).Order("id").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if !row.Amount.Equal(j.Lines[0].Amount) {
		t.Fatalf("amount = %s, want %s", row.Amount, j.Lines[0].Amount)
	}
}
