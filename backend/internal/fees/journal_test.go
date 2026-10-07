package fees

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
)

func snap() Snapshot {
	return Snapshot{
		AttemptID: "att_1", PaymentRequestID: 42, MerchantID: 17,
		RuleID: 7, RuleVersion: 2, Currency: "USDC", LedgerAsset: "USDC.SOLANA", FeeBearer: BearerMerchant,
	}
}

func breakdown(fee, tax string) Breakdown {
	return Breakdown{RuleID: 7, Version: 2, Currency: "USDC", FeeBearer: BearerMerchant, Amount: d("100"), Fee: d(fee), Tax: d(tax)}
}

func lineAmount(t *testing.T, j ledger.Journal, key ledger.AccountKey) decimal.Decimal {
	t.Helper()
	for _, l := range j.Lines {
		if l.Account == key {
			return l.Amount
		}
	}
	t.Fatalf("no line for %+v in %+v", key, j.Lines)
	return decimal.Zero
}

func TestFeeJournalIsKeyedByAttemptAndPostsInTheLedgerAsset(t *testing.T) {
	j, ok, err := feeJournal(snap(), breakdown("2.5", "0.45"))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if err := j.Validate(); err != nil {
		t.Fatalf("journal invalid: %v", err)
	}
	if j.Kind != ledger.KindFee || j.IdempotencyKey != "fee:attempt:att_1" || j.Reference != (ledger.Reference{Type: "payment_attempt", ID: "att_1"}) {
		t.Errorf("header = %+v", j)
	}
	if j.Metadata["payment_request_id"] != "42" || j.Metadata["amount"] != "100" {
		t.Errorf("metadata = %+v", j.Metadata)
	}
	merchant := ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "17", Asset: "USDC.SOLANA", Kind: ledger.KindLiability}
	income := ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: FeeIncomeOwnerID, Asset: "USDC.SOLANA", Kind: ledger.KindIncome}
	tax := ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: TaxOwnerID, Asset: "USDC.SOLANA", Kind: ledger.KindLiability}
	assertDec(t, "merchant debit", lineAmount(t, j, merchant), "2.95")
	assertDec(t, "fee income credit", lineAmount(t, j, income), "-2.5")
	assertDec(t, "tax liability credit", lineAmount(t, j, tax), "-0.45")
}

func TestFeeJournalOmitsZeroLines(t *testing.T) {
	j, ok, err := feeJournal(snap(), breakdown("1", "0"))
	if err != nil || !ok || len(j.Lines) != 2 {
		t.Fatalf("untaxed fee: ok=%v err=%v lines=%d", ok, err, len(j.Lines))
	}
	if _, ok, err := feeJournal(snap(), breakdown("0", "0")); err != nil || ok {
		t.Fatalf("zero fee should post nothing: ok=%v err=%v", ok, err)
	}
}

func TestFeeJournalRefusesABreakdownFromAnotherRule(t *testing.T) {
	b := breakdown("1", "0")
	b.RuleID = 8
	var ve *ValidationError
	if _, _, err := feeJournal(snap(), b); !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestFiatAssetIsTheBareCodeAndTakesNoChain(t *testing.T) {
	if a, err := fiatAsset("USD", ""); err != nil || a != "USD" {
		t.Fatalf("USD = %q %v", a, err)
	}
	if _, err := fiatAsset("USD", "BASE"); err == nil {
		t.Fatal("a chain on fiat accepted")
	}
}
