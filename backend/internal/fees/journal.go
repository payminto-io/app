package fees

import (
	"strconv"

	"github.com/payminto/payminto/backend/internal/ledger"
)

// Ledger owner ids under ledger.OwnerFees; see README "Ledger".
const (
	FeeIncomeOwnerID = "platform"
	TaxOwnerID       = "tax"
)

// feeJournal debits the merchant fee+tax and credits fee income and tax payable; ok is false when there is nothing to post.
func feeJournal(pf PaymentFee) (ledger.Journal, bool, error) {
	b := pf.Breakdown
	switch {
	case pf.PaymentRequestID == 0:
		return ledger.Journal{}, false, invalid("payment_request_id", "required")
	case pf.MerchantID == "":
		return ledger.Journal{}, false, invalid("merchant_id", "required")
	case b.RuleID == 0 || b.Version < 1:
		return ledger.Journal{}, false, invalid("rule_id", "breakdown carries no rule")
	case b.Fee.IsNegative() || b.Tax.IsNegative():
		return ledger.Journal{}, false, invalid("fee", "must not be negative")
	}
	total := b.Fee.Add(b.Tax)
	if total.IsZero() {
		return ledger.Journal{}, false, nil
	}
	ref := strconv.FormatUint(uint64(pf.PaymentRequestID), 10)
	j := ledger.Journal{
		Kind:           ledger.KindFee,
		Reference:      ledger.Reference{Type: "payment_request", ID: ref},
		IdempotencyKey: "fee:payment_request:" + ref,
		Metadata: map[string]any{
			"fee_rule_id":      strconv.FormatUint(uint64(b.RuleID), 10),
			"fee_rule_version": strconv.Itoa(b.Version),
			"fee_bearer":       string(b.FeeBearer),
			"fee":              b.Fee.String(),
			"tax":              b.Tax.String(),
		},
		Lines: []ledger.Line{{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: pf.MerchantID, Asset: b.Currency, Kind: ledger.KindLiability},
			Amount:  total,
		}},
	}
	if !b.Fee.IsZero() {
		j.Lines = append(j.Lines, ledger.Line{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: FeeIncomeOwnerID, Asset: b.Currency, Kind: ledger.KindIncome},
			Amount:  b.Fee.Neg(),
		})
	}
	if !b.Tax.IsZero() {
		j.Lines = append(j.Lines, ledger.Line{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: TaxOwnerID, Asset: b.Currency, Kind: ledger.KindLiability},
			Amount:  b.Tax.Neg(),
		})
	}
	return j, true, nil
}
