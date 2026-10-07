package quote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

var (
	canonicalIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)
	currencyPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
	positiveInteger    = regexp.MustCompile(`^[1-9][0-9]*$`)
)

type Adapter struct {
	catalog AssetCatalog
	prices  PriceSource
	config  Config
}

var _ paymentlifecycle.QuoteAdapter = (*Adapter)(nil)

func New(catalog AssetCatalog, prices PriceSource, config Config) (*Adapter, error) {
	if catalog == nil || prices == nil || config.Now == nil || config.QuoteTTL <= 0 || config.MaxPriceAge <= 0 {
		return nil, errors.New("quote: catalog, price source, clock, positive TTL, and positive freshness are required")
	}
	return &Adapter{catalog: catalog, prices: prices, config: config}, nil
}

func (a *Adapter) Quote(ctx context.Context, request paymentlifecycle.QuoteRequest) (paymentlifecycle.Quote, error) {
	now := a.config.Now().UTC()
	if err := validateRequest(request, now); err != nil {
		return paymentlifecycle.Quote{}, err
	}
	asset, err := a.catalog.Resolve(ctx, request.TenantID, request.PaymentMethod.ChainID, request.PaymentMethod.AssetID)
	if err != nil {
		if errors.Is(err, ErrAssetUnknown) || errors.Is(err, ErrAssetDisabled) {
			return paymentlifecycle.Quote{}, paymentlifecycle.ErrUnsupportedPaymentMethod
		}
		return paymentlifecycle.Quote{}, quoteUnavailable(err)
	}
	if !asset.Enabled {
		return paymentlifecycle.Quote{}, paymentlifecycle.ErrUnsupportedPaymentMethod
	}
	if asset.TenantID != request.TenantID || asset.ChainID != request.PaymentMethod.ChainID || asset.AssetID != request.PaymentMethod.AssetID || asset.Decimals > 77 {
		return paymentlifecycle.Quote{}, quoteUnavailable(errors.New("catalog returned malformed asset"))
	}
	price, err := a.prices.Price(ctx, request.InvoiceAmount.Currency, asset.ChainID, asset.AssetID)
	if err != nil {
		return paymentlifecycle.Quote{}, quoteUnavailable(err)
	}
	if err := validatePrice(price, request, now, a.config.MaxPriceAge); err != nil {
		return paymentlifecycle.Quote{}, err
	}
	required, err := ceilingAtomicUnits(request.InvoiceAmount.MinorUnits, price.MinorUnitsPerAtomicNumerator, price.MinorUnitsPerAtomicDenominator)
	if err != nil || len(required) > 78 {
		return paymentlifecycle.Quote{}, quoteUnavailable(errors.New("price conversion exceeds supported exact amount"))
	}
	expiresAt := minimumTime(request.InvoiceExpiry, now.Add(a.config.QuoteTTL), price.ObservedAt.Add(a.config.MaxPriceAge)).UTC()
	if !expiresAt.After(now) {
		return paymentlifecycle.Quote{}, quoteUnavailable(errors.New("price freshness cannot cover quote"))
	}
	result := paymentlifecycle.Quote{
		InvoiceCurrency: request.InvoiceAmount.Currency, InvoiceMinorUnits: request.InvoiceAmount.MinorUnits,
		ChainID: asset.ChainID, AssetID: asset.AssetID, RequiredAtomicUnits: required,
		AssetDecimals: asset.Decimals, RateNumerator: price.MinorUnitsPerAtomicNumerator,
		RateDenominator: price.MinorUnitsPerAtomicDenominator, Source: price.Source,
		QuotedAt: price.ObservedAt.UTC(), ExpiresAt: expiresAt, Rounding: paymentlifecycle.RoundingCeil,
	}
	result.ID = quoteID(request.TenantID, result)
	return result, nil
}

func validateRequest(request paymentlifecycle.QuoteRequest, now time.Time) error {
	chain := string(request.PaymentMethod.ChainID)
	asset := string(request.PaymentMethod.AssetID)
	if !canonicalIDPattern.MatchString(string(request.TenantID)) || len(request.TenantID) > 128 || !currencyPattern.MatchString(request.InvoiceAmount.Currency) ||
		!validPositive(request.InvoiceAmount.MinorUnits) || chain == "" || len(chain) > 128 ||
		!canonicalIDPattern.MatchString(chain) || len(asset) > 255 || !canonicalIDPattern.MatchString(asset) ||
		!strings.HasPrefix(asset, chain+"/") || len(asset) == len(chain)+1 {
		return quoteUnavailable(errors.New("malformed quote request"))
	}
	if request.InvoiceExpiry.IsZero() || !request.InvoiceExpiry.After(now) {
		return paymentlifecycle.ErrQuoteExpired
	}
	return nil
}

func validatePrice(price Price, request paymentlifecycle.QuoteRequest, now time.Time, maxAge time.Duration) error {
	if price.Currency != request.InvoiceAmount.Currency || price.ChainID != request.PaymentMethod.ChainID || price.AssetID != request.PaymentMethod.AssetID ||
		!validPositive(price.MinorUnitsPerAtomicNumerator) || !validPositive(price.MinorUnitsPerAtomicDenominator) ||
		!visibleASCII(price.Source, 1, 128) || price.ObservedAt.IsZero() || price.ObservedAt.After(now) {
		return quoteUnavailable(errors.New("price source returned malformed price"))
	}
	if now.Sub(price.ObservedAt) > maxAge {
		return quoteUnavailable(errors.New("price source returned stale price"))
	}
	return nil
}

func validPositive(value string) bool {
	return len(value) <= 78 && positiveInteger.MatchString(value)
}

func ceilingAtomicUnits(invoiceMinorUnits, numerator, denominator string) (string, error) {
	invoice, ok := new(big.Int).SetString(invoiceMinorUnits, 10)
	if !ok {
		return "", errors.New("invalid invoice minor units")
	}
	n, ok := new(big.Int).SetString(numerator, 10)
	if !ok || n.Sign() <= 0 {
		return "", errors.New("invalid price numerator")
	}
	d, ok := new(big.Int).SetString(denominator, 10)
	if !ok || d.Sign() <= 0 {
		return "", errors.New("invalid price denominator")
	}
	product := new(big.Int).Mul(invoice, d)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(product, n, remainder)
	if remainder.Sign() != 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if quotient.Sign() <= 0 {
		return "", errors.New("non-positive required atomic units")
	}
	return quotient.String(), nil
}

func quoteID(tenantID paymentlifecycle.TenantID, quote paymentlifecycle.Quote) paymentlifecycle.QuoteID {
	h := sha256.New()
	write := func(value string) {
		_, _ = h.Write([]byte(strconv.Itoa(len(value))))
		_, _ = h.Write([]byte{':'})
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{'|'})
	}
	for _, value := range []string{
		"paymentlifecycle.quote.v1", string(tenantID), quote.InvoiceCurrency, quote.InvoiceMinorUnits,
		string(quote.ChainID), string(quote.AssetID), quote.RequiredAtomicUnits,
		strconv.Itoa(int(quote.AssetDecimals)), quote.RateNumerator, quote.RateDenominator,
		quote.Source, quote.QuotedAt.UTC().Format(time.RFC3339Nano), quote.ExpiresAt.UTC().Format(time.RFC3339Nano), quote.Rounding,
	} {
		write(value)
	}
	return paymentlifecycle.QuoteID("quote-" + hex.EncodeToString(h.Sum(nil)))
}

func minimumTime(values ...time.Time) time.Time {
	minimum := values[0]
	for _, value := range values[1:] {
		if value.Before(minimum) {
			minimum = value
		}
	}
	return minimum
}

func visibleASCII(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for i := range len(value) {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func quoteUnavailable(cause error) error {
	return &paymentlifecycle.Error{Code: paymentlifecycle.CodeQuoteUnavailable, Cause: fmt.Errorf("quote adapter: %w", cause)}
}
