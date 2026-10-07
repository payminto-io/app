//go:build integration

package ledger_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func acct(owner ledger.OwnerType, id, asset string, kind ledger.AccountKind) ledger.AccountKey {
	return ledger.AccountKey{OwnerType: owner, OwnerID: id, Asset: asset, Kind: kind}
}

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func paymentJournal(key, amount string) ledger.Journal {
	return ledger.Journal{
		Kind:           ledger.KindPayment,
		Reference:      ledger.Reference{Type: "payment", ID: key},
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{Account: acct(ledger.OwnerPlatform, "hot", "USDC", ledger.KindAsset), Amount: d(amount)},
			{Account: acct(ledger.OwnerMember, "m1", "USDC", ledger.KindLiability), Amount: d(amount).Neg()},
		},
	}
}

func TestIntegration_SchemaConvergesFromAutoMigrateAndChecksummedMigration(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()

	if _, err := database.ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("ApplyMigrations after dev-style migrate: %v", err)
	}
	for _, trigger := range []string{
		"ledger_lines_append_only", "ledger_journals_append_only", "ledger_accounts_append_only",
		"ledger_lines_no_truncate", "ledger_journals_no_truncate", "ledger_accounts_no_truncate",
		"ledger_journals_stamp_txid", "ledger_lines_same_transaction",
		"ledger_lines_balance", "ledger_journals_has_lines",
	} {
		var n int64
		if err := db.Raw(`SELECT count(*) FROM pg_trigger WHERE tgname = ?`, trigger).Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("trigger %s count = %d, want 1", trigger, n)
		}
	}
	var fk int64
	if err := db.Raw(`SELECT count(*) FROM pg_constraint WHERE conname = 'ledger_lines_account_asset_fkey'`).Scan(&fk).Error; err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("composite account/asset FK count = %d, want 1", fk)
	}
	var indexes []string
	if err := db.Raw(`SELECT indexname FROM pg_indexes WHERE tablename = 'ledger_accounts' AND indexname LIKE 'ledger_accounts_%owner_asset_kind_key'`).Scan(&indexes).Error; err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 1 || indexes[0] != "ledger_accounts_env_owner_asset_kind_key" {
		t.Fatalf("ledger_accounts uniqueness indexes = %v, want only the environment-scoped one", indexes)
	}
	for _, table := range []string{"ledger_accounts", "api_keys"} {
		if !db.Migrator().HasColumn(table, "environment") {
			t.Fatalf("%s.environment missing after migrations", table)
		}
	}
}

func TestIntegration_EnvironmentsNeverShareAccountsOrBalances(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()
	// Two processes, one per environment, each guarded; one database stands in for two here.
	testSvc := ledger.New(db, ledger.WithEnvironment(environment.Test))
	liveSvc := ledger.New(db, ledger.WithEnvironment(environment.Live))

	if _, err := testSvc.Post(environment.WithContext(ctx, environment.Test), paymentJournal("t1", "10.000000000000000001")); err != nil {
		t.Fatal(err)
	}
	if _, err := liveSvc.Post(environment.WithContext(ctx, environment.Live), paymentJournal("l1", "70")); err != nil {
		t.Fatal(err)
	}
	if _, err := testSvc.Post(ctx, paymentJournal("l2", "1")); err != nil {
		t.Fatal(err)
	}

	member := acct(ledger.OwnerMember, "m1", "USDC", ledger.KindLiability)
	testID, err := testSvc.AccountID(ctx, member)
	if err != nil {
		t.Fatal(err)
	}
	liveID, err := liveSvc.AccountID(ctx, member)
	if err != nil {
		t.Fatal(err)
	}
	if testID == liveID {
		t.Fatal("same account id across environments")
	}
	testBal, _ := testSvc.Balance(ctx, testID)
	liveBal, _ := liveSvc.Balance(ctx, liveID)
	if !testBal.Equal(d("-11.000000000000000001")) || !liveBal.Equal(d("-70")) {
		t.Fatalf("balances mixed: test=%s live=%s", testBal, liveBal)
	}
	testTotals, _ := testSvc.Balances(ctx, ledger.OwnerMember, "m1")
	liveTotals, _ := liveSvc.Balances(ctx, ledger.OwnerMember, "m1")
	if !testTotals["USDC"].Equal(d("11.000000000000000001")) || !liveTotals["USDC"].Equal(d("70")) {
		t.Fatalf("owner totals mixed: test=%v live=%v", testTotals, liveTotals)
	}

	// A live row cannot be written through the test process, and vice versa.
	if _, err := testSvc.Post(environment.WithContext(ctx, environment.Live), paymentJournal("x1", "1")); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("test process accepted a live post: %v", err)
	}
	liveKeyed := paymentJournal("x2", "1")
	for i := range liveKeyed.Lines {
		liveKeyed.Lines[i].Account.Environment = environment.Live
	}
	if _, err := testSvc.Post(ctx, liveKeyed); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("test process accepted live-keyed lines: %v", err)
	}
	if _, err := liveSvc.Balances(environment.WithContext(ctx, environment.Test), ledger.OwnerMember, "m1"); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live process served a test read: %v", err)
	}
	if err := db.Exec(`INSERT INTO ledger_accounts (environment, owner_type, owner_id, asset, kind, created_at) VALUES ('prod', 'member', 'm9', 'USDC', 'asset', now())`).Error; err == nil || !strings.Contains(err.Error(), "ledger_accounts_environment_check") {
		t.Fatalf("database accepted an unknown environment: %v", err)
	}
}

func TestIntegration_UpdateAndDeleteAreRejectedByTriggers(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	id, err := s.Post(context.Background(), paymentJournal("p1", "10"))
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"update line":     `UPDATE ledger_lines SET amount = amount + 1 WHERE journal_id = ?`,
		"delete line":     `DELETE FROM ledger_lines WHERE journal_id = ?`,
		"update journal":  `UPDATE ledger_journals SET kind = 'refund' WHERE id = ?`,
		"delete journal":  `DELETE FROM ledger_journals WHERE id = ?`,
		"update account":  `UPDATE ledger_accounts SET owner_id = 'x' WHERE id IN (SELECT account_id FROM ledger_lines WHERE journal_id = ?)`,
		"delete accounts": `DELETE FROM ledger_accounts WHERE id IN (SELECT account_id FROM ledger_lines WHERE journal_id = ?)`,
	}
	for name, stmt := range cases {
		t.Run(name, func(t *testing.T) {
			err := db.Exec(stmt, id).Error
			if err == nil || !strings.Contains(err.Error(), "append-only") {
				t.Fatalf("got %v, want append-only rejection", err)
			}
		})
	}
	hot, _ := s.AccountID(context.Background(), acct(ledger.OwnerPlatform, "hot", "USDC", ledger.KindAsset))
	bal, err := s.Balance(context.Background(), hot)
	if err != nil || !bal.Equal(d("10")) {
		t.Fatalf("balance after rejected mutations = %s, %v; want 10", bal, err)
	}
}

func TestIntegration_UnbalancedCommitIsRejectedAtTheDatabase(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	ctx := context.Background()
	hot, err := s.EnsureAccount(ctx, acct(ledger.OwnerPlatform, "hot", "USDC", ledger.KindAsset))
	if err != nil {
		t.Fatal(err)
	}

	// Bypass Validate and write a one-sided journal straight to the tables.
	err = db.Transaction(func(tx *gorm.DB) error {
		j := ledger.JournalRow{Kind: ledger.KindPayment, ReferenceType: "t", ReferenceID: "1", IdempotencyKey: "raw:1", RequestHash: strings.Repeat("a", 64), PostedAt: time.Now(), Metadata: ledger.Metadata{}}
		if err := tx.Create(&j).Error; err != nil {
			return err
		}
		return tx.Create(&ledger.LineRow{JournalID: j.ID, AccountID: hot, Asset: "USDC", Amount: d("5")}).Error
	})
	if err == nil || !strings.Contains(err.Error(), "does not balance") {
		t.Fatalf("commit of unbalanced journal = %v, want balance trigger error", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		j := ledger.JournalRow{Kind: ledger.KindPayment, ReferenceType: "t", ReferenceID: "2", IdempotencyKey: "raw:2", RequestHash: strings.Repeat("b", 64), PostedAt: time.Now(), Metadata: ledger.Metadata{}}
		return tx.Create(&j).Error
	})
	if err == nil || !strings.Contains(err.Error(), "has no lines") {
		t.Fatalf("commit of empty journal = %v, want has-no-lines trigger error", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		j := ledger.JournalRow{Kind: ledger.KindPayment, ReferenceType: "t", ReferenceID: "3", IdempotencyKey: "raw:3", RequestHash: strings.Repeat("c", 64), PostedAt: time.Now(), Metadata: ledger.Metadata{}}
		if err := tx.Create(&j).Error; err != nil {
			return err
		}
		return tx.Create(&ledger.LineRow{JournalID: j.ID, AccountID: hot, Asset: "SOL", Amount: d("1")}).Error
	})
	if err == nil || !strings.Contains(err.Error(), "ledger_lines_account_asset_fkey") {
		t.Fatalf("line on wrong asset = %v, want composite FK violation", err)
	}

	var journals, lines int64
	db.Model(&ledger.JournalRow{}).Count(&journals)
	db.Model(&ledger.LineRow{}).Count(&lines)
	if journals != 0 || lines != 0 {
		t.Fatalf("journals=%d lines=%d after rejected commits, want 0/0", journals, lines)
	}
}

func TestIntegration_ConcurrentPostsWithOneKeyProduceOneJournal(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	const workers = 12

	start := make(chan struct{})
	ids := make(chan ledger.JournalID, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			id, err := s.Post(context.Background(), paymentJournal("race", "7"))
			if err != nil {
				errs <- err
				return
			}
			ids <- id
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Post: %v", err)
	}
	seen := map[ledger.JournalID]bool{}
	for id := range ids {
		seen[id] = true
	}
	if len(seen) != 1 {
		t.Fatalf("distinct journal ids = %v, want exactly one", seen)
	}
	var journals, lines, accounts int64
	db.Model(&ledger.JournalRow{}).Count(&journals)
	db.Model(&ledger.LineRow{}).Count(&lines)
	db.Model(&ledger.AccountRow{}).Count(&accounts)
	if journals != 1 || lines != 2 || accounts != 2 {
		t.Fatalf("journals=%d lines=%d accounts=%d, want 1/2/2", journals, lines, accounts)
	}

	if _, err := s.Post(context.Background(), paymentJournal("race", "8")); !errors.Is(err, ledger.ErrIdempotencyConflict) {
		t.Fatalf("different payload on used key = %v, want ErrIdempotencyConflict", err)
	}
}

func TestIntegration_EighteenDecimalsAreExact(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	ctx := context.Background()
	for i, amt := range []string{"0.000000000000000001", "0.000000000000000002", "99999999999999999999.999999999999999997"} {
		if _, err := s.Post(ctx, paymentJournal("x"+string(rune('a'+i)), amt)); err != nil {
			t.Fatal(err)
		}
	}
	hot, _ := s.AccountID(ctx, acct(ledger.OwnerPlatform, "hot", "USDC", ledger.KindAsset))
	bal, err := s.Balance(ctx, hot)
	if err != nil {
		t.Fatal(err)
	}
	if want := d("100000000000000000000"); !bal.Equal(want) {
		t.Fatalf("balance = %s, want %s", bal, want)
	}
	lines, err := s.Statement(ctx, ledger.OwnerMember, "m1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil || len(lines) != 3 {
		t.Fatalf("statement = %d lines, %v", len(lines), err)
	}
	if !lines[2].Running.Equal(d("-100000000000000000000")) {
		t.Fatalf("running = %s", lines[2].Running)
	}
}

func TestIntegration_PropertyWithEighteenDecimalAmounts(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	ledger.RunPropertyTest(t, ledger.New(db), 18)
}

func TestIntegration_TruncateIsRejected(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	if _, err := s.Post(context.Background(), paymentJournal("p1", "10")); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`TRUNCATE ledger_lines`,
		`TRUNCATE ledger_journals CASCADE`,
		`TRUNCATE ledger_accounts CASCADE`,
		`TRUNCATE ledger_lines, ledger_journals, ledger_accounts`,
	} {
		err := db.Exec(stmt).Error
		if err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s = %v, want append-only rejection", stmt, err)
		}
	}
	var lines int64
	db.Model(&ledger.LineRow{}).Count(&lines)
	if lines != 2 {
		t.Fatalf("lines after rejected truncates = %d, want 2", lines)
	}
}

func TestIntegration_LinesCannotBeAddedToAPostedJournal(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	ctx := context.Background()
	id, err := s.Post(ctx, paymentJournal("p1", "10"))
	if err != nil {
		t.Fatal(err)
	}
	hot, _ := s.AccountID(ctx, acct(ledger.OwnerPlatform, "hot", "USDC", ledger.KindAsset))
	member, _ := s.AccountID(ctx, acct(ledger.OwnerMember, "m1", "USDC", ledger.KindLiability))

	// A balanced pair appended later would rewrite history while passing the balance trigger.
	err = db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&[]ledger.LineRow{
			{JournalID: id, AccountID: hot, Asset: "USDC", Amount: d("1000")},
			{JournalID: id, AccountID: member, Asset: "USDC", Amount: d("-1000")},
		}).Error
	})
	if err == nil || !strings.Contains(err.Error(), "sealed") {
		t.Fatalf("late line insert = %v, want sealed-journal rejection", err)
	}
	bal, err := s.Balance(ctx, hot)
	if err != nil || !bal.Equal(d("10")) {
		t.Fatalf("balance = %s, %v; want 10", bal, err)
	}

	// The client cannot forge posting_txid: the stamp trigger overwrites whatever is sent.
	var stamped int64
	if err := db.Raw(`SELECT count(*) FROM ledger_journals WHERE posting_txid = 0`).Scan(&stamped).Error; err != nil {
		t.Fatal(err)
	}
	if stamped != 0 {
		t.Fatalf("%d journals carry posting_txid 0; the stamp trigger did not run", stamped)
	}
}

func TestIntegration_ConcurrentFirstUseOfAccountsInOppositeOrder(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	s := ledger.New(db)
	const rounds = 40
	errs := make(chan error, rounds*2)
	var wg sync.WaitGroup
	for n := range rounds {
		a := acct(ledger.OwnerMember, fmt.Sprintf("x%d", n), "USDC", ledger.KindLiability)
		b := acct(ledger.OwnerMember, fmt.Sprintf("y%d", n), "USDC", ledger.KindLiability)
		forward := ledger.Journal{Kind: ledger.KindAdjustment, IdempotencyKey: fmt.Sprintf("f%d", n), Lines: []ledger.Line{{Account: a, Amount: d("1")}, {Account: b, Amount: d("-1")}}}
		reverse := ledger.Journal{Kind: ledger.KindAdjustment, IdempotencyKey: fmt.Sprintf("r%d", n), Lines: []ledger.Line{{Account: b, Amount: d("1")}, {Account: a, Amount: d("-1")}}}
		for _, j := range []ledger.Journal{forward, reverse} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := s.Post(context.Background(), j)
				errs <- err
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent opposite-order post: %v", err)
		}
	}
}
