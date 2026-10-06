package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

const paymentPayloadSchemaV1 uint16 = 1

// paymentEnvelopeV1 is the stable persisted receipt/event schema. It is kept
// private to the adapter so Go field renames cannot silently rewrite durable
// JSON contracts.
type paymentEnvelopeV1 struct {
	SchemaVersion uint16        `json:"schema_version"`
	Payment       paymentViewV1 `json:"payment"`
}

type paymentViewV1 struct {
	InvoiceID         string           `json:"invoice_id"`
	TenantID          string           `json:"tenant_id"`
	MerchantReference string           `json:"merchant_reference"`
	InvoiceAmount     fiatAmountV1     `json:"invoice_amount"`
	PaymentMethod     paymentMethodV1  `json:"payment_method"`
	Quote             quoteV1          `json:"quote"`
	DepositAddress    depositAddressV1 `json:"deposit_address"`
	State             string           `json:"state"`
	Revision          uint64           `json:"revision"`
	OpenedAt          time.Time        `json:"opened_at"`
	ExpiresAt         time.Time        `json:"expires_at"`
}

type fiatAmountV1 struct {
	Currency   string `json:"currency"`
	MinorUnits string `json:"minor_units"`
}

type paymentMethodV1 struct {
	ChainID string `json:"chain_id"`
	AssetID string `json:"asset_id"`
}

type quoteV1 struct {
	ID                  string    `json:"id"`
	InvoiceCurrency     string    `json:"invoice_currency"`
	InvoiceMinorUnits   string    `json:"invoice_minor_units"`
	ChainID             string    `json:"chain_id"`
	AssetID             string    `json:"asset_id"`
	RequiredAtomicUnits string    `json:"required_atomic_units"`
	AssetDecimals       uint8     `json:"asset_decimals"`
	RateNumerator       string    `json:"rate_numerator"`
	RateDenominator     string    `json:"rate_denominator"`
	Source              string    `json:"source"`
	QuotedAt            time.Time `json:"quoted_at"`
	ExpiresAt           time.Time `json:"expires_at"`
	Rounding            string    `json:"rounding"`
}

type depositAddressV1 struct {
	AssignmentID string `json:"assignment_id"`
	Address      string `json:"address"`
}

func encodePaymentViewV1(view paymentlifecycle.PaymentView) ([]byte, error) {
	return json.Marshal(paymentEnvelopeV1{SchemaVersion: paymentPayloadSchemaV1, Payment: toPaymentViewV1(view)})
}

func decodePaymentViewV1(raw []byte) (paymentlifecycle.PaymentView, error) {
	var envelope paymentEnvelopeV1
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return paymentlifecycle.PaymentView{}, err
	}
	if envelope.SchemaVersion != paymentPayloadSchemaV1 {
		return paymentlifecycle.PaymentView{}, fmt.Errorf("unsupported payment payload schema %d", envelope.SchemaVersion)
	}
	return fromPaymentViewV1(envelope.Payment), nil
}

func toPaymentViewV1(view paymentlifecycle.PaymentView) paymentViewV1 {
	return paymentViewV1{
		InvoiceID: string(view.InvoiceID), TenantID: string(view.TenantID),
		MerchantReference: string(view.MerchantReference),
		InvoiceAmount:     fiatAmountV1{Currency: view.InvoiceAmount.Currency, MinorUnits: view.InvoiceAmount.MinorUnits},
		PaymentMethod:     paymentMethodV1{ChainID: string(view.PaymentMethod.ChainID), AssetID: string(view.PaymentMethod.AssetID)},
		Quote: quoteV1{
			ID: string(view.Quote.ID), InvoiceCurrency: view.Quote.InvoiceCurrency,
			InvoiceMinorUnits: view.Quote.InvoiceMinorUnits, ChainID: string(view.Quote.ChainID), AssetID: string(view.Quote.AssetID),
			RequiredAtomicUnits: view.Quote.RequiredAtomicUnits, AssetDecimals: view.Quote.AssetDecimals,
			RateNumerator: view.Quote.RateNumerator, RateDenominator: view.Quote.RateDenominator,
			Source: view.Quote.Source, QuotedAt: view.Quote.QuotedAt.UTC(), ExpiresAt: view.Quote.ExpiresAt.UTC(),
			Rounding: view.Quote.Rounding,
		},
		DepositAddress: depositAddressV1{AssignmentID: string(view.DepositAddress.AssignmentID), Address: view.DepositAddress.Address},
		State:          string(view.State), Revision: view.Revision, OpenedAt: view.OpenedAt.UTC(), ExpiresAt: view.ExpiresAt.UTC(),
	}
}

func fromPaymentViewV1(view paymentViewV1) paymentlifecycle.PaymentView {
	return paymentlifecycle.PaymentView{
		InvoiceID: paymentlifecycle.InvoiceID(view.InvoiceID), TenantID: paymentlifecycle.TenantID(view.TenantID),
		MerchantReference: paymentlifecycle.MerchantReference(view.MerchantReference),
		InvoiceAmount:     paymentlifecycle.FiatAmount{Currency: view.InvoiceAmount.Currency, MinorUnits: view.InvoiceAmount.MinorUnits},
		PaymentMethod:     paymentlifecycle.PaymentMethod{ChainID: paymentlifecycle.ChainID(view.PaymentMethod.ChainID), AssetID: paymentlifecycle.AssetID(view.PaymentMethod.AssetID)},
		Quote: paymentlifecycle.Quote{
			ID: paymentlifecycle.QuoteID(view.Quote.ID), InvoiceCurrency: view.Quote.InvoiceCurrency,
			InvoiceMinorUnits: view.Quote.InvoiceMinorUnits, ChainID: paymentlifecycle.ChainID(view.Quote.ChainID),
			AssetID: paymentlifecycle.AssetID(view.Quote.AssetID), RequiredAtomicUnits: view.Quote.RequiredAtomicUnits,
			AssetDecimals: view.Quote.AssetDecimals, RateNumerator: view.Quote.RateNumerator,
			RateDenominator: view.Quote.RateDenominator, Source: view.Quote.Source,
			QuotedAt: view.Quote.QuotedAt, ExpiresAt: view.Quote.ExpiresAt, Rounding: view.Quote.Rounding,
		},
		DepositAddress: paymentlifecycle.DepositAddress{AssignmentID: paymentlifecycle.AddressAssignmentID(view.DepositAddress.AssignmentID), Address: view.DepositAddress.Address},
		State:          paymentlifecycle.InvoiceState(view.State), Revision: view.Revision,
		OpenedAt: view.OpenedAt, ExpiresAt: view.ExpiresAt,
	}
}
