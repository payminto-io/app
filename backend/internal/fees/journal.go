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

// feeJournal debits the merchant fee+tax and credits fee income and tax payable in the snapshot's ledger asset;
// ok is false when there is nothing to post.
func feeJournal(s Snapshot, b Breakdown) (ledger.Journal, bool, error) {
	switch {
	case s.AttemptID == "" || s.PaymentRequestID == 0 || s.MerchantID == 0 || s.LedgerAsset == "":
		return ledger.Journal{}, false, invalid("snapshot", "incomplete snapshot")
	case b.RuleID != s.RuleID || b.Version != s.RuleVersion:
		return ledger.Journal{}, false, invalid("rule_id", "breakdown is from rule %d v%d, snapshot is %d v%d", b.RuleID, b.Version, s.RuleID, s.RuleVersion)
	case b.Fee.IsNegative() || b.Tax.IsNegative():
		return ledger.Journal{}, false, invalid("fee", "must not be negative")
	}
	total := b.Fee.Add(b.Tax)
	if total.IsZero() {
		return ledger.Journal{}, false, nil
	}
	j := ledger.Journal{
		Kind:           ledger.KindFee,
		Reference:      ledger.Reference{Type: "payment_attempt", ID: s.AttemptID},
		IdempotencyKey: "fee:attempt:" + s.AttemptID,
		Metadata: map[string]any{
			"payment_request_id": strconv.FormatUint(uint64(s.PaymentRequestID), 10),
			"fee_rule_id":        strconv.FormatUint(uint64(b.RuleID), 10),
			"fee_rule_version":   strconv.Itoa(b.Version),
			"fee_bearer":         string(b.FeeBearer),
			"amount":             b.Amount.String(),
			"fee":                b.Fee.String(),
			"tax":                b.Tax.String(),
		},
		Lines: []ledger.Line{{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: strconv.FormatUint(uint64(s.MerchantID), 10), Asset: s.LedgerAsset, Kind: ledger.KindLiability},
			Amount:  total,
		}},
	}
	if !b.Fee.IsZero() {
		j.Lines = append(j.Lines, ledger.Line{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: FeeIncomeOwnerID, Asset: s.LedgerAsset, Kind: ledger.KindIncome},
			Amount:  b.Fee.Neg(),
		})
	}
	if !b.Tax.IsZero() {
		j.Lines = append(j.Lines, ledger.Line{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerFees, OwnerID: TaxOwnerID, Asset: s.LedgerAsset, Kind: ledger.KindLiability},
			Amount:  b.Tax.Neg(),
		})
	}
	return j, true, nil
}
