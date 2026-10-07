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

func scopeInput(method fees.Method, currency, percent string) fees.RuleInput {
	return fees.RuleInput{
		Scope:   fees.Scope{Method: method, Currency: currency},
		Pricing: fees.Pricing{Percent: d(percent), FeeBearer: fees.BearerMerchant},
	}
}

func cardDefault(percent string) fees.RuleInput { return scopeInput(fees.MethodCard, "USD", percent) }

func pricing(percent string, from time.Time) fees.Pricing {
	return fees.Pricing{Percent: d(percent), FeeBearer: fees.BearerMerchant, EffectiveFrom: &from}
}

func activeAt(t *testing.T, svc *fees.Service, lineage string, at time.Time) []int {
	t.Helper()
	rules, err := svc.ListRules(context.Background(), fees.RuleFilter{LineageID: lineage})
	if err != nil {
		t.Fatal(err)
	}
	var versions []int
	for _, r := range rules {
		if r.ActiveAt(at) {
			versions = append(versions, r.Version)
		}
	}
	return versions
}

func TestIntegration_NewVersionClosesPreviousAtomically(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
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
	if closed.EffectiveTo == nil || !closed.EffectiveTo.Equal(v2.EffectiveFrom) || !closed.Percent.Equal(d("2")) {
		t.Fatalf("v1 after versioning = %+v", closed)
	}
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

// C1: v1 open, v2 scheduled, v3 now must leave exactly one active version at every instant.
func TestIntegration_SupersedingAScheduledVersionClosesTheWholeLineage(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	scheduled := time.Now().Add(10 * 24 * time.Hour)
	v2, err := svc.NewVersion(ctx, v1.ID, pricing("3", scheduled), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	v3, err := svc.NewVersion(ctx, v2.ID, fees.Pricing{Percent: d("2.5"), FeeBearer: fees.BearerMerchant}, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{
		v3.EffectiveFrom, v3.EffectiveFrom.Add(time.Hour), scheduled.Add(-time.Second), scheduled, scheduled.Add(time.Hour),
	} {
		if got := activeAt(t, svc, v1.LineageID, at); len(got) != 1 || got[0] != 3 {
			t.Errorf("active versions at %s = %v, want [3]", at, got)
		}
		r, err := svc.Resolve(ctx, fees.Query{Method: "card", Currency: "USD", At: at})
		if err != nil || r.ID != v3.ID {
			t.Errorf("resolve at %s = %d %v, want v3", at, r.ID, err)
		}
	}
	old, _ := svc.GetRule(ctx, v1.ID)
	if old.EffectiveTo == nil || !old.EffectiveTo.Equal(v3.EffectiveFrom) {
		t.Errorf("v1 effective_to = %v, want v3 start %v", old.EffectiveTo, v3.EffectiveFrom)
	}
	sch, _ := svc.GetRule(ctx, v2.ID)
	if sch.EffectiveTo == nil || !sch.EffectiveTo.Equal(sch.EffectiveFrom) {
		t.Errorf("superseded scheduled version should be an empty window: %+v", sch)
	}
}

func TestIntegration_NewVersionRollsBackTheCloseWhenTheInsertFails(t *testing.T) {
	svc, _, db := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil || after.EffectiveTo != nil {
		t.Fatalf("v1 was closed although v2 was never written: %+v %v", after, err)
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

// I2: two active rules for one scope can never coexist, whichever path writes them.
func TestIntegration_OverlappingRulesForOneScopeAreRefused(t *testing.T) {
	svc, _, db := newService(t)
	ctx := context.Background()
	in := cardDefault("2")
	in.Connector = sp("stripe")
	a, err := svc.CreateRule(ctx, in, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateRule(ctx, in, "member:1")
	var overlap *fees.OverlapError
	if !errors.As(err, &overlap) || len(overlap.RuleIDs) != 1 || overlap.RuleIDs[0] != a.ID {
		t.Fatalf("err = %v, want OverlapError naming %d", err, a.ID)
	}
	// A future rule in another lineage still overlaps the open one.
	future := time.Now().Add(24 * time.Hour)
	in.EffectiveFrom = &future
	if _, err := svc.CreateRule(ctx, in, "member:1"); !errors.As(err, &overlap) {
		t.Fatalf("future overlap: err = %v, want OverlapError", err)
	}
	// Other scopes are independent.
	other := in
	other.Connector, other.EffectiveFrom = sp("adyen"), nil
	if _, err := svc.CreateRule(ctx, other, "member:1"); err != nil {
		t.Fatalf("other connector: %v", err)
	}
	// The database refuses it even past the service.
	err = db.Exec(`INSERT INTO fee_rules (lineage_id, version, method, connector, currency, minor_units, fee_bearer, effective_from, created_by)
		VALUES (gen_random_uuid(), 1, 'card', 'stripe', 'USD', 2, 'merchant', now(), 'sql')`).Error
	if err == nil || !strings.Contains(err.Error(), "fee_rules_no_overlap") {
		t.Fatalf("direct insert: err = %v, want exclusion violation", err)
	}

	// Concurrent creates of one new scope: exactly one wins.
	var wg sync.WaitGroup
	errs := make([]error, 6)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = svc.CreateRule(ctx, scopeInput(fees.MethodBank, "EUR", "1"), "member:"+strconv.Itoa(i))
		}()
	}
	wg.Wait()
	won := 0
	for _, err := range errs {
		if err == nil {
			won++
		} else if !errors.As(err, &overlap) {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if won != 1 {
		t.Fatalf("%d concurrent creates won, want 1", won)
	}
	var ext int64
	db.Raw(`SELECT count(*) FROM pg_extension WHERE extname = 'btree_gist'`).Scan(&ext)
	if ext != 1 {
		t.Fatal("btree_gist extension not installed by the migration")
	}
}

func TestIntegration_RulesAreAppendOnlyInTheDatabase(t *testing.T) {
	svc, l, db := newService(t)
	ctx := context.Background()
	v1, err := svc.CreateRule(ctx, cardDefault("2"), "member:1")
	if err != nil {
		t.Fatal(err)
	}
	pr := seedPayment(t, db, "ref-ao")
	if err := l.Transaction(ctx, func(tx *gorm.DB) error {
		_, err := svc.Snapshot(ctx, tx, fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_ao"}, fees.Query{Method: "card", Currency: "USD"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE fee_rules SET effective_to = now() + interval '1 day' WHERE id = ?`, v1.ID).Error; err != nil {
		t.Fatalf("closing a rule in the future must be allowed: %v", err)
	}
	cases := map[string]string{
		"change percent":     `UPDATE fee_rules SET percent = 9 WHERE id = ?`,
		"reopen":             `UPDATE fee_rules SET effective_to = NULL WHERE id = ?`,
		"extend":             `UPDATE fee_rules SET effective_to = effective_to + interval '1 day' WHERE id = ?`,
		"close in the past":  `UPDATE fee_rules SET effective_to = now() - interval '30 days' WHERE id = ?`,
		"delete":             `DELETE FROM fee_rules WHERE id = ?`,
		"truncate":           `TRUNCATE fee_rules CASCADE`,
		"rewrite a snapshot": `UPDATE fee_snapshots SET fee_rule_version = 9`,
		"delete a snapshot":  `DELETE FROM fee_snapshots`,
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

func TestIntegration_DecimalsRoundTripExactly(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	upTo := d("100")
	minFee, maxFee := d("0.50"), d("25")
	in := fees.RuleInput{
		Scope: fees.Scope{Method: fees.MethodCrypto, Currency: "USDC"},
		Pricing: fees.Pricing{
			Slabs:  []fees.Slab{{UpTo: &upTo, Percent: d("1.5")}, {Percent: d("1.123456"), Flat: d("0.25")}},
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
	if got.MinorUnits != 6 || len(got.Slabs) != 2 || !got.Slabs[1].Percent.Equal(d("1.123456")) || got.Slabs[1].UpTo != nil ||
		!got.MinFee.Equal(minFee) || !got.MaxFee.Equal(maxFee) || !got.TaxPercent.Equal(d("18")) {
		t.Fatalf("round trip = %+v", got)
	}
	if !created.Percent.Equal(got.Percent) || created.MinorUnits != got.MinorUnits {
		t.Fatalf("create response %+v differs from the stored row %+v", created, got)
	}
	b, err := svc.Preview(ctx, fees.PreviewRequest{Query: fees.Query{Method: "crypto", Currency: "USDC"}, Amount: d("100")})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Fee.Equal(d("1.5")) || !b.Tax.Equal(d("0.27")) || !b.CustomerTotal.Equal(d("101.77")) || !b.MerchantNet.Equal(d("100")) {
		t.Fatalf("preview = %+v", b)
	}
}

func seedPayment(t *testing.T, db *gorm.DB, ref string) models.PaymentRequest {
	t.Helper()
	platform := models.ExternalPlatform{Name: "P", SuccessEndpoint: "https://example.com"}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	member := models.Member{Name: "M", MemberType: "merchant"}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	pr := models.PaymentRequest{ReferenceID: ref, AmountInUSD: d("100"), MemberID: member.ID, ExternalPlatformID: platform.ID}
	if err := db.Create(&pr).Error; err != nil {
		t.Fatal(err)
	}
	return pr
}

func inTx(t *testing.T, l *ledger.Service, fn func(tx *gorm.DB) error) error {
	t.Helper()
	return l.Transaction(context.Background(), fn)
}

func legacySnapshot(t *testing.T, db *gorm.DB, id uint) (ruleID *uint, version *int) {
	t.Helper()
	var row struct {
		FeeRuleID      *uint
		FeeRuleVersion *int
	}
	if err := db.Raw(`SELECT fee_rule_id, fee_rule_version FROM payment_requests WHERE id = ?`, id).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row.FeeRuleID, row.FeeRuleVersion
}

// I3 + I4: snapshot per attempt at creation; fee posted on capture from the snapshotted rule and stored rows.
func TestIntegration_SnapshotPerAttemptAndPostFeeOnCapture(t *testing.T) {
	svc, l, db := newService(t)
	ctx := context.Background()
	stripeIn := cardDefault("2")
	stripeIn.Connector, stripeIn.Taxable, stripeIn.TaxPercent = sp("stripe"), true, d("10")
	stripe, err := svc.CreateRule(ctx, stripeIn, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	adyenIn := cardDefault("3")
	adyenIn.Connector = sp("adyen")
	adyen, err := svc.CreateRule(ctx, adyenIn, "member:1")
	if err != nil {
		t.Fatal(err)
	}
	pr := seedPayment(t, db, "ref-1")
	merchant := strconv.FormatUint(uint64(pr.MemberID), 10)

	first := fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_1"}
	var s1 fees.Snapshot
	if err := inTx(t, l, func(tx *gorm.DB) error {
		s1, err = svc.Snapshot(ctx, tx, first, fees.Query{Method: "card", Connector: "stripe", Currency: "USD"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if s1.RuleID != stripe.ID || s1.MerchantID != pr.MemberID || s1.LedgerAsset != "USD" {
		t.Fatalf("snapshot 1 = %+v", s1)
	}
	if id, _ := legacySnapshot(t, db, pr.ID); id != nil {
		t.Fatal("legacy columns written before any attempt succeeded")
	}

	// Attempt 1 fails; the retry on another connector takes its own snapshot.
	second := fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_2"}
	var s2 fees.Snapshot
	if err := inTx(t, l, func(tx *gorm.DB) error {
		s2, err = svc.Snapshot(ctx, tx, second, fees.Query{Method: "card", Connector: "adyen", Currency: "USD"})
		return err
	}); err != nil {
		t.Fatalf("retry on another connector: %v", err)
	}
	if s2.RuleID != adyen.ID {
		t.Fatalf("snapshot 2 = %+v", s2)
	}

	// The rule is edited after the snapshot; the capture still prices with the snapshotted version.
	if _, err := svc.NewVersion(ctx, adyen.ID, fees.Pricing{Percent: d("9"), FeeBearer: fees.BearerMerchant}, "member:1"); err != nil {
		t.Fatal(err)
	}
	var b fees.Breakdown
	for range 2 { // the replay is idempotent
		if err := inTx(t, l, func(tx *gorm.DB) error {
			b, err = svc.PostFee(ctx, tx, second, d("80"))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if b.RuleID != adyen.ID || b.Version != 1 || !b.Amount.Equal(d("80")) || !b.Fee.Equal(d("2.4")) {
		t.Fatalf("breakdown = %+v, want 3%% of the captured 80 under adyen v1", b)
	}
	if id, v := legacySnapshot(t, db, pr.ID); id == nil || *id != adyen.ID || *v != 1 {
		t.Fatalf("legacy snapshot = %v/%v, want the successful attempt's rule", id, v)
	}
	var journals int64
	db.Model(&ledger.JournalRow{}).Where("kind = ? AND reference_type = ? AND reference_id = ?", ledger.KindFee, "payment_attempt", "att_2").Count(&journals)
	if journals != 1 {
		t.Fatalf("fee journals = %d, want 1", journals)
	}
	accounts, err := l.AccountBalances(ctx, ledger.OwnerMember, merchant)
	if err != nil || len(accounts) != 1 || accounts[0].Account.Asset != "USD" || !accounts[0].Signed.Equal(d("2.4")) {
		t.Fatalf("merchant accounts = %+v err %v, want a 2.4 debit in USD", accounts, err)
	}

	// A replay with another captured amount is refused by the ledger, not silently re-priced.
	err = inTx(t, l, func(tx *gorm.DB) error { _, err := svc.PostFee(ctx, tx, second, d("81")); return err })
	if !errors.Is(err, ledger.ErrIdempotencyConflict) {
		t.Fatalf("different captured amount on replay: err = %v", err)
	}
	// Same attempt id on another payment is a conflict, not a second snapshot.
	other := seedPayment(t, db, "ref-2")
	err = inTx(t, l, func(tx *gorm.DB) error {
		_, err := svc.Snapshot(ctx, tx, fees.AttemptRef{PaymentRequestID: other.ID, AttemptID: "att_1"}, fees.Query{Method: "card", Currency: "USD", Connector: "stripe"})
		return err
	})
	if !errors.Is(err, fees.ErrSnapshotConflict) {
		t.Fatalf("attempt reused on another payment: err = %v", err)
	}
	err = inTx(t, l, func(tx *gorm.DB) error {
		_, err := svc.PostFee(ctx, tx, fees.AttemptRef{PaymentRequestID: other.ID, AttemptID: "att_1"}, d("10"))
		return err
	})
	if !errors.Is(err, fees.ErrSnapshotConflict) {
		t.Fatalf("post with a mismatched payment: err = %v", err)
	}
	err = inTx(t, l, func(tx *gorm.DB) error {
		_, err := svc.PostFee(ctx, tx, fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_none"}, d("10"))
		return err
	})
	if !errors.Is(err, fees.ErrSnapshotNotFound) {
		t.Fatalf("unknown attempt: err = %v", err)
	}
	// A fee above the captured amount is refused (stripe: 2% of 0.01 is 0, so use a min fee rule).
	err = inTx(t, l, func(tx *gorm.DB) error { _, err := svc.PostFee(ctx, tx, first, d("0.01")); return err })
	if err != nil {
		t.Fatalf("tiny capture with no min fee: %v", err)
	}
}

func TestIntegration_CryptoFeePostsInTheChainQualifiedAsset(t *testing.T) {
	svc, l, db := newService(t)
	ctx := context.Background()
	in := scopeInput(fees.MethodCrypto, "USDC", "1")
	minFee := d("0.5")
	in.MinFee = &minFee
	if _, err := svc.CreateRule(ctx, in, "member:1"); err != nil {
		t.Fatal(err)
	}
	pr := seedPayment(t, db, "ref-c")
	ref := fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_c"}
	err := inTx(t, l, func(tx *gorm.DB) error {
		_, err := svc.Snapshot(ctx, tx, ref, fees.Query{Method: "crypto", Currency: "USDC"})
		return err
	})
	var ve *fees.ValidationError
	if !errors.As(err, &ve) || ve.Field != "chain" {
		t.Fatalf("crypto snapshot without a chain: err = %v", err)
	}
	var s fees.Snapshot
	if err := inTx(t, l, func(tx *gorm.DB) error {
		s, err = svc.Snapshot(ctx, tx, ref, fees.Query{Method: "crypto", Currency: "USDC", Chain: "base"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if s.LedgerAsset != "USDC.BASE" {
		t.Fatalf("ledger asset = %s", s.LedgerAsset)
	}
	err = inTx(t, l, func(tx *gorm.DB) error { _, err := svc.PostFee(ctx, tx, ref, d("0.4")); return err })
	if !errors.Is(err, fees.ErrFeeExceedsAmount) {
		t.Fatalf("min fee above the capture: err = %v", err)
	}
	if err := inTx(t, l, func(tx *gorm.DB) error { _, err := svc.PostFee(ctx, tx, ref, d("100.123456")); return err }); err != nil {
		t.Fatal(err)
	}
	income, err := l.AccountBalances(ctx, ledger.OwnerFees, fees.FeeIncomeOwnerID)
	if err != nil || len(income) != 1 || income[0].Account.Asset != "USDC.BASE" || !income[0].Natural.Equal(d("1.001235")) {
		t.Fatalf("fee income = %+v err %v", income, err)
	}
}

func TestIntegration_SoftDeletedPaymentsAreRefused(t *testing.T) {
	svc, l, db := newService(t)
	ctx := context.Background()
	if _, err := svc.CreateRule(ctx, cardDefault("2"), "member:1"); err != nil {
		t.Fatal(err)
	}
	pr := seedPayment(t, db, "ref-d")
	ref := fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_d"}
	if err := inTx(t, l, func(tx *gorm.DB) error {
		_, err := svc.Snapshot(ctx, tx, ref, fees.Query{Method: "card", Currency: "USD"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&pr).Error; err != nil {
		t.Fatal(err)
	}
	err := inTx(t, l, func(tx *gorm.DB) error { _, err := svc.PostFee(ctx, tx, ref, d("10")); return err })
	if !errors.Is(err, fees.ErrPaymentNotFound) {
		t.Fatalf("post on a deleted payment: err = %v", err)
	}
	err = inTx(t, l, func(tx *gorm.DB) error {
		_, err := svc.Snapshot(ctx, tx, fees.AttemptRef{PaymentRequestID: pr.ID, AttemptID: "att_d2"}, fees.Query{Method: "card", Currency: "USD"})
		return err
	})
	if !errors.Is(err, fees.ErrPaymentNotFound) {
		t.Fatalf("snapshot on a deleted payment: err = %v", err)
	}
}
