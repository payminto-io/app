package modules

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// FeesAdapter maps the switch's fee port onto internal/fees.
type FeesAdapter struct {
	Port fees.Port
}

func (a FeesAdapter) Snapshot(ctx context.Context, tx *gorm.DB, ref paymentswitch.FeeRef, q paymentswitch.FeeQuery) error {
	_, err := a.Port.Snapshot(ctx, tx, fees.AttemptRef{PaymentRequestID: ref.PaymentRecordID, AttemptID: ref.AttemptID},
		fees.Query{Method: fees.Method(q.Method), Connector: q.Connector, Currency: q.Currency, Chain: q.Chain})
	if errors.Is(err, fees.ErrNoRule) {
		return fmt.Errorf("%w: %v", paymentswitch.ErrFeeRuleMissing, err)
	}
	return err
}

func (a FeesAdapter) PostFee(ctx context.Context, tx *gorm.DB, ref paymentswitch.FeeRef, captured decimal.Decimal) error {
	_, err := a.Port.PostFee(ctx, tx, fees.AttemptRef{PaymentRequestID: ref.PaymentRecordID, AttemptID: ref.AttemptID}, captured)
	if errors.Is(err, fees.ErrFeeAlreadyPosted) {
		return fmt.Errorf("%w: %v", paymentswitch.ErrFeeAlreadyPosted, err)
	}
	return err
}

// PaymintoPaymentRecords opens the payment_requests row fees prices an intent against. Payminto's record prices
// in USD; an intent in any other asset has no honest amount_in_usd and is refused rather than guessed.
type PaymintoPaymentRecords struct {
	DB *gorm.DB
}

func (p PaymintoPaymentRecords) Open(_ context.Context, tx *gorm.DB, rec paymentswitch.PaymentRecord) (uint, error) {
	if rec.Money.Asset != "USD" {
		return 0, fmt.Errorf("payment records price in USD; intent %s is in %s", rec.IntentID, rec.Money.Asset)
	}
	member, platform, err := parseIDs(rec.MerchantID, rec.PlatformID)
	if err != nil {
		return 0, err
	}
	invoice := rec.IntentID
	expires := rec.ExpiresAt
	if expires.IsZero() {
		return 0, fmt.Errorf("payment record for intent %s has no expiry", rec.IntentID)
	}
	// expires_at lets Payminto's expiry worker close the OPEN anchor row with the intent instead of leaving it forever.
	row := models.PaymentRequest{ReferenceID: rec.IntentID, AmountInUSD: rec.Money.Amount, State: models.PaymentStateOpen, InvoiceID: &invoice, ExpiresAt: &expires, MemberID: member, ExternalPlatformID: platform}
	if err := tx.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func parseIDs(merchant, platform string) (uint, uint, error) {
	var m, p uint
	if _, err := fmt.Sscanf(merchant, "%d", &m); err != nil || m == 0 {
		return 0, 0, fmt.Errorf("merchant id %q is not a Payminto member id", merchant)
	}
	if _, err := fmt.Sscanf(platform, "%d", &p); err != nil || p == 0 {
		return 0, 0, fmt.Errorf("platform id %q is not a Payminto platform id", platform)
	}
	return m, p, nil
}
