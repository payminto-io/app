package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newEnvService(t *testing.T, env environment.Environment) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return New(db, WithEnvironment(env)), db
}

func envJournal(env environment.Environment, key, amount string) Journal {
	j := balancedJournal()
	j.IdempotencyKey, j.Reference.ID = key, key
	j.Lines[0].Amount, j.Lines[1].Amount = dec(amount), dec(amount).Neg()
	for i := range j.Lines {
		j.Lines[i].Account.Environment = env
	}
	return j
}

func TestEnvironment_SameOwnerAndAssetAreDifferentAccountsPerEnvironment(t *testing.T) {
	// One sqlite file stands in for two databases here; Postgres separation is proven in integration_test.go.
	s, _ := newEnvService(t, environment.Test)
	s.guard = nil
	ctx := context.Background()
	mustPost(t, s, envJournal(environment.Test, "t1", "10"))
	mustPost(t, s, envJournal(environment.Live, "l1", "25"))

	testKey := key(OwnerMember, "m1", "USDC", KindLiability)
	testKey.Environment = environment.Test
	liveKey := testKey
	liveKey.Environment = environment.Live
	testID, err := s.AccountID(ctx, testKey)
	if err != nil {
		t.Fatal(err)
	}
	liveID, err := s.AccountID(ctx, liveKey)
	if err != nil {
		t.Fatal(err)
	}
	if testID == liveID {
		t.Fatal("test and live resolved to the same ledger account")
	}
	testBal, _ := s.Balance(ctx, testID)
	liveBal, _ := s.Balance(environment.WithContext(ctx, environment.Live), liveID)
	if !testBal.Equal(dec("-10")) || !liveBal.Equal(dec("-25")) {
		t.Fatalf("balances mixed: test=%s live=%s", testBal, liveBal)
	}
	if _, err := s.Balance(ctx, liveID); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("Balance by id must not cross environments: %v", err)
	}

	testTotals, err := s.Balances(environment.WithContext(ctx, environment.Test), OwnerMember, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if !testTotals["USDC"].Equal(dec("10")) {
		t.Fatalf("test owner totals include live lines: %v", testTotals)
	}
	liveTotals, _ := s.Balances(environment.WithContext(ctx, environment.Live), OwnerMember, "m1")
	if !liveTotals["USDC"].Equal(dec("25")) {
		t.Fatalf("live owner totals wrong: %v", liveTotals)
	}
	var n int64
	s.db.Model(&AccountRow{}).Where("owner_type = ? AND owner_id = ?", OwnerMember, "m1").Count(&n)
	if n != 2 {
		t.Fatalf("accounts for m1 = %d, want one per environment", n)
	}
}

func TestEnvironment_ZeroValueResolvesFromContextThenServiceDefault(t *testing.T) {
	s, _ := newEnvService(t, environment.Test)
	ctx := context.Background()
	mustPost(t, s, balancedJournal())
	var row AccountRow
	if err := s.db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Environment != environment.Test {
		t.Fatalf("default environment = %q, want test", row.Environment)
	}
	id, err := s.AccountID(environment.WithContext(ctx, environment.Test), key(OwnerMember, "m1", "USDC", KindLiability))
	if err != nil || id == 0 {
		t.Fatalf("AccountID from context = %d, %v", id, err)
	}
	if _, err := s.AccountID(environment.WithContext(ctx, environment.Live), key(OwnerMember, "m1", "USDC", KindLiability)); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live context on a test process must be refused by the guard: %v", err)
	}
}

func TestEnvironment_GuardRejectsCrossEnvironmentPosts(t *testing.T) {
	s, db := newEnvService(t, environment.Test)
	ctx := context.Background()
	if _, err := s.Post(ctx, envJournal(environment.Live, "l1", "5")); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live journal on test process = %v, want ErrMismatch", err)
	}
	if _, err := s.Post(environment.WithContext(ctx, environment.Live), balancedJournal()); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live context on test process = %v, want ErrMismatch", err)
	}
	var n int64
	db.Model(&JournalRow{}).Count(&n)
	if n != 0 {
		t.Fatalf("journals written = %d, want 0", n)
	}
	if _, err := s.Balances(environment.WithContext(ctx, environment.Live), OwnerMember, "m1"); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live read on test process = %v, want ErrMismatch", err)
	}
}

func TestValidate_MixedEnvironmentsRejected(t *testing.T) {
	j := balancedJournal()
	j.Lines[0].Account.Environment = environment.Live
	j.Lines[1].Account.Environment = environment.Test
	if err := j.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate() = %v, want ErrInvalid for mixed environments", err)
	}
	j.Lines[0].Account.Environment = "prod"
	if err := j.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate() = %v, want ErrInvalid for unknown environment", err)
	}
}

func TestRequestHash_LegacyHashIsPinnedAndNewHashesCarryTheEnvironment(t *testing.T) {
	// Journals posted before this column existed must still replay cleanly: pinned from the pre-environment code.
	const pinned = "d854d561d5a494db1a56afdb5a84ea5341695da7268e2a7aba912bad02b67da5"
	if got := balancedJournal().legacyRequestHash(); got != pinned {
		t.Fatalf("legacy hash = %s, want %s", got, pinned)
	}
	j := balancedJournal()
	if j.requestHashFor(environment.Test) == j.requestHashFor(environment.Live) || j.requestHashFor(environment.Test) == pinned {
		t.Fatal("the stored hash must differ per environment and from the legacy form")
	}
}

func TestIdempotency_IsScopedToTheEnvironment(t *testing.T) {
	s, db := newEnvService(t, environment.Test)
	s.guard = nil
	ctx := context.Background()
	first := mustPost(t, s, balancedJournal())
	// Same key and lines in the other environment: a conflict, never a silent replay of the test journal.
	if _, err := s.Post(environment.WithContext(ctx, environment.Live), balancedJournal()); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("live post with a test key = %v, want ErrIdempotencyConflict", err)
	}
	var journals int64
	db.Model(&JournalRow{}).Count(&journals)
	if journals != 1 {
		t.Fatalf("journals = %d, want 1", journals)
	}
	receipt, err := s.PostIn(ctx, db, balancedJournal())
	if err != nil || !receipt.Replayed || receipt.ID != first {
		t.Fatalf("same-environment replay = %+v, %v", receipt, err)
	}
}

func TestIdempotency_RowsWithTheLegacyHashStillReplay(t *testing.T) {
	s, db := newEnvService(t, environment.Test)
	j := balancedJournal()
	row := JournalRow{Kind: j.Kind, ReferenceType: j.Reference.Type, ReferenceID: j.Reference.ID, IdempotencyKey: j.IdempotencyKey, Environment: environment.Test, RequestHash: j.legacyRequestHash(), PostedAt: time.Now()}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	receipt, err := s.PostIn(context.Background(), db, j)
	if err != nil || !receipt.Replayed || receipt.ID != row.ID {
		t.Fatalf("legacy replay = %+v, %v", receipt, err)
	}
	changed := j
	changed.Lines[0].Amount, changed.Lines[1].Amount = dec("11"), dec("-11")
	if _, err := s.PostIn(context.Background(), db, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed payload against a legacy row = %v, want conflict", err)
	}
}
