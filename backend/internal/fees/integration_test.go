//go:build integration

package fees_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func sp(s string) *string { return &s }

func newService(t *testing.T) (*fees.Service, *ledger.Service, *gorm.DB) {
	t.Helper()
	db, cleanup := database.NewTestDB(t)
	t.Cleanup(cleanup)
	l := ledger.New(db)
	return fees.NewService(db, l, fees.DefaultPolicy()), l, db
}

func cardDefault(percent string) fees.RuleInput {
	return fees.RuleInput{
		Scope:   fees.Scope{Method: fees.MethodCard, Currency: "USD"},
		Pricing: fees.Pricing{Percent: d(percent), FeeBearer: fees.BearerMerchant},
	}
}

func pricing(percent string, from time.Time) fees.Pricing {
	return fees.Pricing{Percent: d(percent), FeeBearer: fees.BearerMerchant, EffectiveFrom: &from}
}

func TestIntegration_NewVersionClosesPreviousAtomically(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	if v1.Version != 1 || v1.EffectiveTo != nil || v1.LineageID == "" {
		t.Fatalf("v1 = %+v", v1)
	}
	from := time.Now().Add(time.Hour)
	v2, err := svc.NewVersion(ctx, v1.ID, pricing("3", from), "member:2")
	if err != nil {
		t.Fatal(err)
	}
	if v2.Version != 2 || v2.LineageID != v1.LineageID || v2.CreatedBy != "member:2" || v2.Method != fees.MethodCard {
		t.Fatalf("v2 = %+v", v2)
	}
	closed, err := svc.GetRule(ctx, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.EffectiveTo == nil || !closed.EffectiveTo.Equal(v2.EffectiveFrom) {
		t.Fatalf("v1 effective_to = %v, want v2 effective_from %v", closed.EffectiveTo, v2.EffectiveFrom)
	}
	if !closed.Percent.Equal(d("2")) || closed.CreatedBy != "member:1" {
		t.Fatalf("v1 was mutated beyond effective_to: %+v", closed)
	}

	// v1 still prices now; v2 prices from its start; a future rule is not active early.
	got, err := svc.Resolve(ctx, fees.Query{Method: "card", Currency: "usd"})
	if err != nil || got.ID != v1.ID {
		t.Fatalf("resolve now = %d %v, want v1", got.ID, err)
	}
	got, err = svc.Resolve(ctx, fees.Query{Method: "card", Currency: "USD", At: from})
	if err != nil || got.ID != v2.ID {
		t.Fatalf("resolve at v2 start = %d %v, want v2", got.ID, err)
	}

	if _, err := svc.NewVersion(ctx, v1.ID, pricing("4", from.Add(time.Hour)), "member:3"); !errors.Is(err, fees.ErrStaleVersion) {
		t.Fatalf("editing a superseded version: err = %v, want ErrStaleVersion", err)
	}
	if _, err := svc.NewVersion(ctx, 999999, pricing("4", from), "member:3"); !errors.Is(err, fees.ErrNotFound) {
		t.Fatalf("unknown rule: err = %v, want ErrNotFound", err)
	}
}

func TestIntegration_NewVersionRollsBackTheCloseWhenTheInsertFails(t *testing.T) {
	svc, _, db := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	// Fail the second statement of the transaction (the insert) after the close has run.
	if err := db.Exec(`
CREATE FUNCTION test_reject_v2() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'injected insert failure'; END $$;
CREATE TRIGGER test_reject_v2 BEFORE INSERT ON fee_rules FOR EACH ROW WHEN (NEW.version = 2)
    EXECUTE FUNCTION test_reject_v2();`).Error; err != nil {
		t.Fatal(err)
	}
	_, err = svc.NewVersion(ctx, v1.ID, pricing("3", time.Now().Add(time.Minute)), "member:2")
	if err == nil || !strings.Contains(err.Error(), "injected insert failure") {
		t.Fatalf("err = %v, want the injected failure", err)
	}
	after, err := svc.GetRule(ctx, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.EffectiveTo != nil {
		t.Fatalf("v1 was closed although v2 was never written: effective_to = %v", after.EffectiveTo)
	}
	rules, err := svc.ListRules(ctx, fees.RuleFilter{LineageID: v1.LineageID})
	if err != nil || len(rules) != 1 {
		t.Fatalf("lineage rows = %d err %v, want 1", len(rules), err)
	}
}

func TestIntegration_ConcurrentEditsProduceExactlyOneNextVersion(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	const editors = 8
	var wg sync.WaitGroup
	errs := make([]error, editors)
	from := time.Now().Add(time.Hour)
	for i := range editors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = svc.NewVersion(ctx, v1.ID, pricing(strconv.Itoa(3+i), from), "member:"+strconv.Itoa(i))
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, fees.ErrStaleVersion):
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d editors succeeded, want exactly 1", ok)
	}
	rules, err := svc.ListRules(ctx, fees.RuleFilter{LineageID: v1.LineageID})
	if err != nil || len(rules) != 2 || rules[1].Version != 2 {
		t.Fatalf("lineage = %+v err %v, want versions 1 and 2", rules, err)
	}
}

func TestIntegration_RulesAreAppendOnlyInTheDatabase(t *testing.T) {
	svc, _, db := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE fee_rules SET effective_to = effective_from + interval '1 day' WHERE id = ?`, v1.ID).Error; err != nil {
		t.Fatalf("closing a rule must be allowed: %v", err)
	}
	cases := map[string]string{
		"change percent":       `UPDATE fee_rules SET percent = 9 WHERE id = ?`,
		"reopen":               `UPDATE fee_rules SET effective_to = NULL WHERE id = ?`,
		"extend":               `UPDATE fee_rules SET effective_to = effective_to + interval '1 day' WHERE id = ?`,
		"delete":               `DELETE FROM fee_rules WHERE id = ?`,
		"truncate (ignores ?)": `TRUNCATE fee_rules CASCADE`,
	}
	for name, stmt := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			if strings.Contains(stmt, "?") {
				err = db.Exec(stmt, v1.ID).Error
			} else {
				err = db.Exec(stmt).Error
			}
			if err == nil || !strings.Contains(err.Error(), "append-only") {
				t.Fatalf("got %v, want append-only rejection", err)
			}
		})
	}
}

func TestIntegration_ResolveTieIsReportedNotPicked(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	in := cardDefault("2")
	in.Connector = sp("stripe")
	a, err := svc.CreateRule(ctx, in, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateRule(ctx, in, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Resolve(ctx, fees.Query{Method: fees.MethodCard, Connector: "stripe", Currency: "USD"})
	var amb *fees.AmbiguousRuleError
	if !errors.As(err, &amb) || len(amb.RuleIDs) != 2 || amb.RuleIDs[0] != a.ID || amb.RuleIDs[1] != b.ID {
		t.Fatalf("err = %v, want ambiguity over %d and %d", err, a.ID, b.ID)
	}
}

func TestIntegration_SlabsAndDecimalsRoundTrip(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	upTo := d("100")
	minFee, maxFee := d("0.50"), d("25")
	in := fees.RuleInput{
		Scope: fees.Scope{Method: fees.MethodCrypto, Currency: "USDC"},
		Pricing: fees.Pricing{
			Slabs:  []fees.Slab{{UpTo: &upTo, Percent: d("1.5")}, {Percent: d("1"), Flat: d("0.25")}},
			MinFee: &minFee, MaxFee: &maxFee,
			Taxable: true, TaxPercent: d("18"),
			FeeBearer: fees.BearerCustomer,
		},
	}
	created, err := svc.CreateRule(ctx, in, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetRule(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Slabs) != 2 || got.Slabs[0].UpTo == nil || !got.Slabs[0].UpTo.Equal(upTo) || got.Slabs[1].UpTo != nil ||
		!got.MinFee.Equal(minFee) || !got.MaxFee.Equal(maxFee) || !got.TaxPercent.Equal(d("18")) {
		t.Fatalf("round trip = %+v", got)
	}
	b, err := svc.Preview(ctx, fees.PreviewRequest{Query: fees.Query{Method: "crypto", Currency: "USDC"}, Amount: d("100")})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Fee.Equal(d("1.5")) || !b.Tax.Equal(d("0.27")) || !b.CustomerTotal.Equal(d("101.77")) || !b.MerchantNet.Equal(d("100")) {
		t.Fatalf("preview = %+v", b)
	}
}

func seedPayment(t *testing.T, db *gorm.DB) models.PaymentRequest {
	t.Helper()
	platform := models.ExternalPlatform{Name: "P", SuccessEndpoint: "https://example.com"}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	member := models.Member{Name: "M", MemberType: "merchant"}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	pr := models.PaymentRequest{ReferenceID: "ref-1", AmountInUSD: d("100"), MemberID: member.ID, ExternalPlatformID: platform.ID}
	if err := db.Create(&pr).Error; err != nil {
		t.Fatal(err)
	}
	return pr
}

func TestIntegration_ApplyToPaymentSnapshotsAndPostsFeeJournal(t *testing.T) {
	svc, l, db := newService(t)
	ctx := context.Background()
	in := cardDefault("2")
	in.Taxable, in.TaxPercent = true, d("10")
	rule, err := svc.CreateRule(ctx, in, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	pr := seedPayment(t, db)
	merchant := strconv.FormatUint(uint64(pr.MemberID), 10)
	pf := fees.PaymentFee{PaymentRequestID: pr.ID, MerchantID: merchant, Breakdown: fees.Compute(rule, d("100"))}

	for range 2 { // the second apply is an idempotent replay
		if err := l.Transaction(ctx, func(tx *gorm.DB) error { return svc.ApplyToPayment(ctx, tx, pf) }); err != nil {
			t.Fatal(err)
		}
	}
	var snap struct {
		FeeRuleID      uint
		FeeRuleVersion int
	}
	if err := db.Raw(`SELECT fee_rule_id, fee_rule_version FROM payment_requests WHERE id = ?`, pr.ID).Scan(&snap).Error; err != nil {
		t.Fatal(err)
	}
	if snap.FeeRuleID != rule.ID || snap.FeeRuleVersion != 1 {
		t.Fatalf("snapshot = %+v, want rule %d v1", snap, rule.ID)
	}
	var journals int64
	if err := db.Model(&ledger.JournalRow{}).Where("kind = ? AND reference_id = ?", ledger.KindFee, strconv.FormatUint(uint64(pr.ID), 10)).Count(&journals).Error; err != nil {
		t.Fatal(err)
	}
	if journals != 1 {
		t.Fatalf("fee journals = %d, want 1", journals)
	}
	bal, err := l.Balances(ctx, ledger.OwnerFees, fees.FeeIncomeOwnerID)
	if err != nil || !bal["USD"].Equal(d("-2")) {
		t.Fatalf("fee income = %v err %v, want -2 (credit)", bal, err)
	}
	tax, err := l.Balances(ctx, ledger.OwnerFees, fees.TaxOwnerID)
	if err != nil || !tax["USD"].Equal(d("-0.2")) {
		t.Fatalf("tax payable = %v err %v, want -0.2 (credit)", tax, err)
	}

	// A different rule cannot overwrite the snapshot, through the service or directly.
	other, err := svc.CreateRule(ctx, fees.RuleInput{Scope: fees.Scope{Method: fees.MethodBank, Currency: "USD"}, Pricing: fees.Pricing{Flat: d("1"), FeeBearer: fees.BearerMerchant}}, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	pf.Breakdown = fees.Compute(other, d("100"))
	err = l.Transaction(ctx, func(tx *gorm.DB) error { return svc.ApplyToPayment(ctx, tx, pf) })
	if !errors.Is(err, fees.ErrSnapshotConflict) {
		t.Fatalf("err = %v, want ErrSnapshotConflict", err)
	}
	err = db.Exec(`UPDATE payment_requests SET fee_rule_id = ?, fee_rule_version = 1 WHERE id = ?`, other.ID, pr.ID).Error
	if err == nil || !strings.Contains(err.Error(), "fixed once written") {
		t.Fatalf("direct overwrite: err = %v, want trigger rejection", err)
	}

	pf.PaymentRequestID = 999999
	err = l.Transaction(ctx, func(tx *gorm.DB) error { return svc.ApplyToPayment(ctx, tx, pf) })
	if !errors.Is(err, fees.ErrPaymentNotFound) {
		t.Fatalf("err = %v, want ErrPaymentNotFound", err)
	}
}
