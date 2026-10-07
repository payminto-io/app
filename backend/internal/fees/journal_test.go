package fees

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
)

func paymentFee(fee, tax string, bearer FeeBearer) PaymentFee {
	return PaymentFee{
		PaymentRequestID: 42,
		MerchantID:       "17",
		Breakdown: Breakdown{
			RuleID: 7, Version: 2, Currency: "USDC", FeeBearer: bearer,
			Amount: d("100"), Fee: d(fee), Tax: d(tax),
		},
	}
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

func TestFeeJournalBalancesAndNamesAccounts(t *testing.T) {
	for _, bearer := range []FeeBearer{BearerMerchant, BearerCustomer} {
		j, ok, err := feeJournal(paymentFee("2.5", "0.45", bearer))
		if err != nil || !ok {
			t.Fatalf("%s: ok=%v err=%v", bearer, ok, err)
		}
		if err := j.Validate(); err != nil {
			t.Fatalf("%s: journal invalid: %v", bearer, err)
		}
		if j.Kind != ledger.KindFee || j.IdempotencyKey != "fee:payment_request:42" || j.Reference.ID != "42" {
			t.Errorf("%s: header = %+v", bearer, j)
		}
		merchant := ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "17", Asset: "USDC", Kind: ledger.KindLiability}
		income := ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: FeeIncomeOwnerID, Asset: "USDC", Kind: ledger.KindIncome}
		tax := ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: TaxOwnerID, Asset: "USDC", Kind: ledger.KindLiability}
		assertDec(t, "merchant debit", lineAmount(t, j, merchant), "2.95")
		assertDec(t, "fee income credit", lineAmount(t, j, income), "-2.5")
		assertDec(t, "tax liability credit", lineAmount(t, j, tax), "-0.45")
	}
}

func TestFeeJournalOmitsZeroLines(t *testing.T) {
	j, ok, err := feeJournal(paymentFee("1", "0", BearerMerchant))
	if err != nil || !ok || len(j.Lines) != 2 {
		t.Fatalf("untaxed fee: ok=%v err=%v lines=%d", ok, err, len(j.Lines))
	}
	if _, ok, err := feeJournal(paymentFee("0", "0", BearerMerchant)); err != nil || ok {
		t.Fatalf("zero fee should post nothing: ok=%v err=%v", ok, err)
	}
}

func TestFeeJournalRejectsIncompleteInput(t *testing.T) {
	cases := map[string]func(*PaymentFee){
		"no payment":  func(p *PaymentFee) { p.PaymentRequestID = 0 },
		"no merchant": func(p *PaymentFee) { p.MerchantID = "" },
		"no rule":     func(p *PaymentFee) { p.Breakdown.RuleID = 0 },
		"negative":    func(p *PaymentFee) { p.Breakdown.Fee = d("-1") },
	}
	for name, mut := range cases {
		pf := paymentFee("1", "0", BearerMerchant)
		mut(&pf)
		var ve *ValidationError
		if _, _, err := feeJournal(pf); !errors.As(err, &ve) {
			t.Errorf("%s: err = %v, want ValidationError", name, err)
		}
	}
}
