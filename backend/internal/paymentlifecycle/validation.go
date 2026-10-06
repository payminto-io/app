package paymentlifecycle

import (
	"regexp"
	"strings"
	"time"
)

var (
	canonicalIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,254}$`)
	currencyPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
	positiveInteger    = regexp.MustCompile(`^[1-9][0-9]*$`)
	requestHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func validateCommandStructure(cmd OpenPayment) error {
	if !canonicalIDPattern.MatchString(string(cmd.TenantID)) || len(cmd.TenantID) > 128 {
		return failure(CodeInvalidCommand, "tenant_id", nil)
	}
	if !visibleASCII(string(cmd.IdempotencyKey), 16, 128) {
		return failure(CodeInvalidCommand, "idempotency_key", nil)
	}
	if !visibleASCII(string(cmd.MerchantReference), 1, 128) {
		return failure(CodeInvalidCommand, "merchant_reference", nil)
	}
	if !currencyPattern.MatchString(cmd.InvoiceAmount.Currency) {
		return failure(CodeInvalidCommand, "invoice_amount.currency", nil)
	}
	if !positiveInteger.MatchString(cmd.InvoiceAmount.MinorUnits) || len(cmd.InvoiceAmount.MinorUnits) > 78 {
		return failure(CodeInvalidCommand, "invoice_amount.minor_units", nil)
	}
	if !canonicalIDPattern.MatchString(string(cmd.PaymentMethod.ChainID)) || len(cmd.PaymentMethod.ChainID) > 128 {
		return failure(CodeInvalidCommand, "payment_method.chain_id", nil)
	}
	if !canonicalIDPattern.MatchString(string(cmd.PaymentMethod.AssetID)) {
		return failure(CodeInvalidCommand, "payment_method.asset_id", nil)
	}
	assetPrefix := string(cmd.PaymentMethod.ChainID) + "/"
	if !strings.HasPrefix(string(cmd.PaymentMethod.AssetID), assetPrefix) || len(cmd.PaymentMethod.AssetID) == len(assetPrefix) {
		return failure(CodeInvalidCommand, "payment_method.asset_id", nil)
	}
	if cmd.ExpiresAt.IsZero() {
		return failure(CodeInvalidCommand, "expires_at", nil)
	}
	return nil
}

func validateCreateCommand(cmd OpenPayment, now time.Time) error {
	if !cmd.ExpiresAt.After(now) {
		return failure(CodeInvalidCommand, "expires_at", nil)
	}
	return nil
}

func validateQuote(quote Quote, cmd OpenPayment, now time.Time) error {
	if quote.ExpiresAt.IsZero() || !quote.ExpiresAt.After(now) {
		return failure(CodeQuoteExpired, "quote.expires_at", nil)
	}
	if !canonicalIDPattern.MatchString(string(quote.ID)) || len(quote.ID) > 128 ||
		quote.InvoiceCurrency != cmd.InvoiceAmount.Currency ||
		quote.InvoiceMinorUnits != cmd.InvoiceAmount.MinorUnits ||
		quote.ChainID != cmd.PaymentMethod.ChainID ||
		quote.AssetID != cmd.PaymentMethod.AssetID {
		return failure(CodeQuoteMismatched, "quote.identity", nil)
	}
	if !positiveInteger.MatchString(quote.RequiredAtomicUnits) || len(quote.RequiredAtomicUnits) > 78 {
		return failure(CodeQuoteMismatched, "quote.required_atomic_units", nil)
	}
	if quote.AssetDecimals > 77 {
		return failure(CodeQuoteMismatched, "quote.asset_decimals", nil)
	}
	if !positiveInteger.MatchString(quote.RateNumerator) || len(quote.RateNumerator) > 78 {
		return failure(CodeQuoteMismatched, "quote.rate_numerator", nil)
	}
	if !positiveInteger.MatchString(quote.RateDenominator) || len(quote.RateDenominator) > 78 {
		return failure(CodeQuoteMismatched, "quote.rate_denominator", nil)
	}
	if !visibleASCII(quote.Source, 1, 128) {
		return failure(CodeQuoteMismatched, "quote.source", nil)
	}
	if quote.QuotedAt.IsZero() || quote.QuotedAt.After(now) {
		return failure(CodeQuoteMismatched, "quote.quoted_at", nil)
	}
	if !validRounding(quote.Rounding) {
		return failure(CodeQuoteMismatched, "quote.rounding", nil)
	}
	if !quote.ExpiresAt.After(quote.QuotedAt) || quote.ExpiresAt.After(cmd.ExpiresAt) {
		return failure(CodeQuoteMismatched, "quote.expires_at", nil)
	}
	return nil
}

func validateStoreResult(result StoreResult, cmd OpenPayment, quote Quote, openedAt time.Time) error {
	view := result.View
	if result.Disposition != StoreCreated && result.Disposition != StoreReplayed {
		return failure(CodeStorageUnavailable, "store.disposition", nil)
	}
	if !canonicalIDPattern.MatchString(string(view.InvoiceID)) ||
		!canonicalIDPattern.MatchString(string(view.DepositAddress.AssignmentID)) ||
		!visibleASCII(view.DepositAddress.Address, 1, 512) {
		return failure(CodeStorageUnavailable, "store.identity", nil)
	}
	if view.TenantID != cmd.TenantID ||
		view.MerchantReference != cmd.MerchantReference ||
		view.InvoiceAmount != cmd.InvoiceAmount ||
		view.PaymentMethod != cmd.PaymentMethod {
		return failure(CodeStorageUnavailable, "store.projection", nil)
	}
	if result.Disposition == StoreCreated && !equalQuote(view.Quote, quote) {
		return failure(CodeStorageUnavailable, "store.quote", nil)
	}
	if result.Disposition == StoreReplayed && !validPersistedQuote(view.Quote, cmd) {
		return failure(CodeStorageUnavailable, "store.quote", nil)
	}
	if result.Disposition == StoreReplayed &&
		(view.Quote.QuotedAt.After(view.OpenedAt) || !view.Quote.ExpiresAt.After(view.OpenedAt)) {
		return failure(CodeStorageUnavailable, "store.quote_timeline", nil)
	}
	if view.State != InvoiceOpen || view.Revision != 1 || !view.ExpiresAt.Equal(cmd.ExpiresAt) {
		return failure(CodeStorageUnavailable, "store.version", nil)
	}
	if view.OpenedAt.IsZero() || !view.ExpiresAt.After(view.OpenedAt) {
		return failure(CodeStorageUnavailable, "store.timeline", nil)
	}
	if result.Disposition == StoreCreated && !view.OpenedAt.Equal(openedAt) {
		return failure(CodeStorageUnavailable, "store.opened_at", nil)
	}
	return nil
}

func validPersistedQuote(quote Quote, cmd OpenPayment) bool {
	return canonicalIDPattern.MatchString(string(quote.ID)) && len(quote.ID) <= 128 &&
		quote.InvoiceCurrency == cmd.InvoiceAmount.Currency &&
		quote.InvoiceMinorUnits == cmd.InvoiceAmount.MinorUnits &&
		quote.ChainID == cmd.PaymentMethod.ChainID && quote.AssetID == cmd.PaymentMethod.AssetID &&
		positiveInteger.MatchString(quote.RequiredAtomicUnits) && len(quote.RequiredAtomicUnits) <= 78 &&
		quote.AssetDecimals <= 77 &&
		positiveInteger.MatchString(quote.RateNumerator) && len(quote.RateNumerator) <= 78 &&
		positiveInteger.MatchString(quote.RateDenominator) && len(quote.RateDenominator) <= 78 &&
		visibleASCII(quote.Source, 1, 128) && !quote.QuotedAt.IsZero() &&
		quote.ExpiresAt.After(quote.QuotedAt) && !quote.ExpiresAt.After(cmd.ExpiresAt) &&
		validRounding(quote.Rounding)
}

func validRounding(rounding string) bool {
	switch rounding {
	case RoundingCeil, RoundingFloor, RoundingHalfUp, RoundingHalfEven:
		return true
	default:
		return false
	}
}

func equalQuote(a, b Quote) bool {
	return a.ID == b.ID &&
		a.InvoiceCurrency == b.InvoiceCurrency && a.InvoiceMinorUnits == b.InvoiceMinorUnits &&
		a.ChainID == b.ChainID && a.AssetID == b.AssetID &&
		a.RequiredAtomicUnits == b.RequiredAtomicUnits && a.AssetDecimals == b.AssetDecimals &&
		a.RateNumerator == b.RateNumerator && a.RateDenominator == b.RateDenominator &&
		a.Source == b.Source && a.QuotedAt.Equal(b.QuotedAt) && a.ExpiresAt.Equal(b.ExpiresAt) &&
		a.Rounding == b.Rounding
}

func visibleASCII(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}
