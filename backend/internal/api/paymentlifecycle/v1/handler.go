package paymentlifecyclev1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

const (
	IdempotencyKeyHeader          = "Idempotency-Key"
	AuthenticatedTenantContextKey = "externalPlatformID"
	maxOpenRequestBytes           = 64 << 10
)

type Handler struct {
	lifecycle paymentlifecycle.PaymentLifecycle
}

func NewHandler(lifecycle paymentlifecycle.PaymentLifecycle) *Handler {
	return &Handler{lifecycle: lifecycle}
}

// Open handles the version-one POST contract. It is intentionally not wired
// into the production router until the domain and PostgreSQL adapters pass the
// integration review described in cloud.agent.md.
func (h *Handler) Open(c *gin.Context) {
	tenantID, ok := authenticatedTenant(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, ErrorCodeUnauthenticated, "authentication is required")
		return
	}

	idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
	if !validIdempotencyKey(idempotencyKey) {
		writeError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "request is invalid")
		return
	}

	var request OpenPaymentRequest
	if err := decodeRequest(c, &request); err != nil {
		writeError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "request is invalid")
		return
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, request.ExpiresAt)
	if err != nil {
		writeError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "request is invalid")
		return
	}

	view, err := h.lifecycle.Open(c.Request.Context(), paymentlifecycle.OpenPayment{
		TenantID:          tenantID,
		IdempotencyKey:    paymentlifecycle.IdempotencyKey(idempotencyKey),
		MerchantReference: paymentlifecycle.MerchantReference(request.MerchantReference),
		InvoiceAmount: paymentlifecycle.FiatAmount{
			Currency:   request.InvoiceAmount.Currency,
			MinorUnits: request.InvoiceAmount.MinorUnits,
		},
		PaymentMethod: paymentlifecycle.PaymentMethod{
			ChainID: paymentlifecycle.ChainID(request.PaymentMethod.ChainID),
			AssetID: paymentlifecycle.AssetID(request.PaymentMethod.AssetID),
		},
		ExpiresAt: expiresAt,
	})
	if err != nil {
		writeLifecycleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, OpenPaymentEnvelope{Payment: paymentResponse(view)})
}

func decodeRequest(c *gin.Context, target *OpenPaymentRequest) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxOpenRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}

func authenticatedTenant(c *gin.Context) (paymentlifecycle.TenantID, bool) {
	value, exists := c.Get(AuthenticatedTenantContextKey)
	if !exists {
		return "", false
	}
	var tenantID string
	switch typed := value.(type) {
	case uint:
		if typed == 0 {
			return "", false
		}
		tenantID = strconv.FormatUint(uint64(typed), 10)
	case uint64:
		if typed == 0 {
			return "", false
		}
		tenantID = strconv.FormatUint(typed, 10)
	case paymentlifecycle.TenantID:
		tenantID = string(typed)
	default:
		return "", false
	}
	if tenantID == "" {
		return "", false
	}
	return paymentlifecycle.TenantID(tenantID), true
}

func validIdempotencyKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] < 0x21 || key[i] > 0x7e {
			return false
		}
	}
	return true
}

func paymentResponse(view paymentlifecycle.PaymentView) PaymentResponse {
	return PaymentResponse{
		InvoiceID:         string(view.InvoiceID),
		MerchantReference: string(view.MerchantReference),
		InvoiceAmount: InvoiceAmountResponse{
			Currency:   view.InvoiceAmount.Currency,
			MinorUnits: view.InvoiceAmount.MinorUnits,
		},
		PaymentMethod: PaymentMethodResponse{
			ChainID: string(view.PaymentMethod.ChainID),
			AssetID: string(view.PaymentMethod.AssetID),
		},
		Quote: QuoteResponse{
			ID:                  string(view.Quote.ID),
			InvoiceCurrency:     view.Quote.InvoiceCurrency,
			InvoiceMinorUnits:   view.Quote.InvoiceMinorUnits,
			ChainID:             string(view.Quote.ChainID),
			AssetID:             string(view.Quote.AssetID),
			RequiredAtomicUnits: view.Quote.RequiredAtomicUnits,
			AssetDecimals:       view.Quote.AssetDecimals,
			RateNumerator:       view.Quote.RateNumerator,
			RateDenominator:     view.Quote.RateDenominator,
			Source:              view.Quote.Source,
			QuotedAt:            view.Quote.QuotedAt.UTC().Format(time.RFC3339Nano),
			ExpiresAt:           view.Quote.ExpiresAt.UTC().Format(time.RFC3339Nano),
			Rounding:            view.Quote.Rounding,
		},
		DepositAddress: DepositAddressResponse{
			AssignmentID: string(view.DepositAddress.AssignmentID),
			Address:      view.DepositAddress.Address,
		},
		State:     string(view.State),
		Revision:  view.Revision,
		OpenedAt:  view.OpenedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt: view.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}
}

func writeLifecycleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, paymentlifecycle.ErrInvalidCommand):
		writeError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "request is invalid")
	case errors.Is(err, paymentlifecycle.ErrUnsupportedPaymentMethod):
		writeError(c, http.StatusUnprocessableEntity, ErrorCodeUnsupportedPaymentMethod, "payment method is unsupported or disabled")
	case errors.Is(err, paymentlifecycle.ErrIdempotencyConflict):
		writeError(c, http.StatusConflict, ErrorCodeIdempotencyConflict, "idempotency key was already used for a different request")
	case errors.Is(err, paymentlifecycle.ErrMerchantReferenceConflict):
		writeError(c, http.StatusConflict, ErrorCodeMerchantReferenceConflict, "merchant reference already exists")
	case errors.Is(err, paymentlifecycle.ErrQuoteExpired):
		writeError(c, http.StatusUnprocessableEntity, ErrorCodeQuoteExpired, "quote expired before the payment could be opened")
	case errors.Is(err, paymentlifecycle.ErrQuoteMismatched):
		writeError(c, http.StatusUnprocessableEntity, ErrorCodeQuoteMismatched, "quote does not match the requested payment")
	case errors.Is(err, paymentlifecycle.ErrQuoteUnavailable):
		writeError(c, http.StatusServiceUnavailable, ErrorCodeQuoteUnavailable, "quote is temporarily unavailable")
	case errors.Is(err, paymentlifecycle.ErrDepositAddressUnavailable):
		writeError(c, http.StatusServiceUnavailable, ErrorCodeDepositAddressUnavailable, "deposit address is temporarily unavailable")
	case errors.Is(err, paymentlifecycle.ErrStorageUnavailable):
		writeError(c, http.StatusServiceUnavailable, ErrorCodeStorageUnavailable, "payment storage is temporarily unavailable")
	default:
		writeError(c, http.StatusServiceUnavailable, ErrorCodeStorageUnavailable, "payment storage is temporarily unavailable")
	}
}

func writeError(c *gin.Context, status int, code ErrorCode, message string) {
	c.JSON(status, ErrorEnvelope{Error: ErrorResponse{Code: code, Message: message}})
}
