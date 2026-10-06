package paymentlifecyclev1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

type fakePaymentLifecycle struct {
	view  paymentlifecycle.PaymentView
	err   error
	got   paymentlifecycle.OpenPayment
	calls int
}

func (f *fakePaymentLifecycle) Open(_ context.Context, cmd paymentlifecycle.OpenPayment) (paymentlifecycle.PaymentView, error) {
	f.calls++
	f.got = cmd
	return f.view, f.err
}

func TestOpenReturnsExactServerQuoteAndUsesAuthenticatedTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 27, 10, 15, 0, 123456000, time.UTC)
	view := validPaymentView(now)
	lifecycle := &fakePaymentLifecycle{view: view}
	router := newTestRouter(lifecycle, uint(41))

	requestBody := `{
		"merchantReference":"order-1042",
		"invoiceAmount":{"currency":"USD","minorUnits":"900719925474099312345"},
		"paymentMethod":{"chainId":"eip155:1","assetId":"eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7"},
		"expiresAt":"2026-08-27T10:45:00.123456Z"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBufferString(requestBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(IdempotencyKeyHeader, "checkout-20260827-0001")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if lifecycle.got.TenantID != paymentlifecycle.TenantID("41") {
		t.Fatalf("TenantID = %q, want authenticated tenant 41", lifecycle.got.TenantID)
	}
	if lifecycle.got.IdempotencyKey != paymentlifecycle.IdempotencyKey("checkout-20260827-0001") {
		t.Fatalf("IdempotencyKey = %q", lifecycle.got.IdempotencyKey)
	}
	if lifecycle.got.InvoiceAmount.MinorUnits != "900719925474099312345" {
		t.Fatalf("MinorUnits = %q; exact wire string was not preserved", lifecycle.got.InvoiceAmount.MinorUnits)
	}
	if !lifecycle.got.ExpiresAt.Equal(time.Date(2026, 8, 27, 10, 45, 0, 123456000, time.UTC)) {
		t.Fatalf("ExpiresAt = %s", lifecycle.got.ExpiresAt)
	}

	var envelope struct {
		Payment struct {
			InvoiceID         string `json:"invoiceId"`
			MerchantReference string `json:"merchantReference"`
			InvoiceAmount     struct {
				MinorUnits string `json:"minorUnits"`
			} `json:"invoiceAmount"`
			State    string `json:"state"`
			Revision uint64 `json:"revision"`
			Quote    struct {
				RequiredAtomicUnits string `json:"requiredAtomicUnits"`
				RateNumerator       string `json:"rateNumerator"`
				RateDenominator     string `json:"rateDenominator"`
			} `json:"quote"`
			DepositAddress struct {
				Address string `json:"address"`
			} `json:"depositAddress"`
		} `json:"payment"`
	}
	decodeJSON(t, response, &envelope)
	if envelope.Payment.InvoiceID != "invoice-000001" || envelope.Payment.MerchantReference != "order-1042" {
		t.Fatalf("payment identity = %#v", envelope.Payment)
	}
	if envelope.Payment.InvoiceAmount.MinorUnits != "900719925474099312345" {
		t.Fatalf("response minor units = %q", envelope.Payment.InvoiceAmount.MinorUnits)
	}
	if envelope.Payment.Quote.RequiredAtomicUnits != "123456789012345678901234567890" ||
		envelope.Payment.Quote.RateNumerator != "123456789012345678901" ||
		envelope.Payment.Quote.RateDenominator != "100000000000000000000" {
		t.Fatalf("quote strings lost precision: %#v", envelope.Payment.Quote)
	}
	if envelope.Payment.DepositAddress.Address != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("address = %q", envelope.Payment.DepositAddress.Address)
	}
	if envelope.Payment.State != "open" {
		t.Fatalf("state = %q", envelope.Payment.State)
	}
	if envelope.Payment.Revision != 7 {
		t.Fatalf("revision = %d, want 7", envelope.Payment.Revision)
	}
}

func TestOpenReturnsIdenticalEnvelopeForIdenticalDomainView(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lifecycle := &fakePaymentLifecycle{view: validPaymentView(time.Date(2026, 8, 27, 10, 15, 0, 0, time.UTC))}
	router := newTestRouter(lifecycle, uint(41))

	first := httptest.NewRecorder()
	router.ServeHTTP(first, validRequest())
	second := httptest.NewRecorder()
	router.ServeHTTP(second, validRequest())

	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("statuses = %d, %d; want two identical 201 responses", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("identical PaymentView produced different envelopes:\nfirst=%s\nsecond=%s", first.Body.String(), second.Body.String())
	}
	if lifecycle.calls != 2 {
		t.Fatalf("Open calls = %d, want 2", lifecycle.calls)
	}
}

func TestOpenRejectsTenantOverrideBeforeCallingLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lifecycle := &fakePaymentLifecycle{}
	router := newTestRouter(lifecycle, uint(41))
	body := `{
		"tenantId":"999",
		"merchantReference":"order-1042",
		"invoiceAmount":{"currency":"USD","minorUnits":"1250"},
		"paymentMethod":{"chainId":"eip155:1","assetId":"eip155:1/slip44:60"},
		"expiresAt":"2026-08-27T10:45:00Z"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(IdempotencyKeyHeader, "checkout-20260827-0001")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertError(t, response, http.StatusBadRequest, ErrorCodeInvalidRequest)
	if lifecycle.calls != 0 {
		t.Fatalf("lifecycle calls = %d, body tenant override must be rejected at HTTP seam", lifecycle.calls)
	}
}

func TestOpenRejectsMissingAuthenticatedTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lifecycle := &fakePaymentLifecycle{}
	router := gin.New()
	router.POST("/api/v1/payments", NewHandler(lifecycle).Open)
	request := validRequest()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertError(t, response, http.StatusUnauthorized, ErrorCodeUnauthenticated)
}

func TestOpenRejectsMalformedContractAndIdempotencyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name           string
		body           string
		idempotencyKey string
	}{
		{name: "missing header", body: validBody()},
		{name: "short header", body: validBody(), idempotencyKey: "too-short"},
		{name: "non-visible header", body: validBody(), idempotencyKey: "checkout-20260827-\n0001"},
		{name: "unknown field", body: `{"merchantReference":"order-1042","invoiceAmount":{"currency":"USD","minorUnits":"1250"},"paymentMethod":{"chainId":"eip155:1","assetId":"eip155:1/slip44:60"},"expiresAt":"2026-08-27T10:45:00Z","extra":true}`, idempotencyKey: "checkout-20260827-0001"},
		{name: "trailing json", body: validBody() + `{}`, idempotencyKey: "checkout-20260827-0001"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lifecycle := &fakePaymentLifecycle{}
			router := newTestRouter(lifecycle, uint(41))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBufferString(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.idempotencyKey != "" {
				request.Header.Set(IdempotencyKeyHeader, tc.idempotencyKey)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assertError(t, response, http.StatusBadRequest, ErrorCodeInvalidRequest)
			if lifecycle.calls != 0 {
				t.Fatal("invalid transport request reached lifecycle")
			}
		})
	}
}

func TestOpenMapsDomainFailuresToSafeStableErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   ErrorCode
	}{
		{name: "validation", err: &paymentlifecycle.Error{Code: paymentlifecycle.CodeInvalidCommand, Field: "invoiceAmount.minorUnits", Cause: errors.New("sensitive cause")}, wantStatus: http.StatusBadRequest, wantCode: ErrorCodeInvalidRequest},
		{name: "unsupported method", err: paymentlifecycle.ErrUnsupportedPaymentMethod, wantStatus: http.StatusUnprocessableEntity, wantCode: ErrorCodeUnsupportedPaymentMethod},
		{name: "idempotency conflict", err: paymentlifecycle.ErrIdempotencyConflict, wantStatus: http.StatusConflict, wantCode: ErrorCodeIdempotencyConflict},
		{name: "reference conflict", err: paymentlifecycle.ErrMerchantReferenceConflict, wantStatus: http.StatusConflict, wantCode: ErrorCodeMerchantReferenceConflict},
		{name: "quote unavailable", err: paymentlifecycle.ErrQuoteUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: ErrorCodeQuoteUnavailable},
		{name: "quote expired", err: paymentlifecycle.ErrQuoteExpired, wantStatus: http.StatusUnprocessableEntity, wantCode: ErrorCodeQuoteExpired},
		{name: "quote mismatched", err: paymentlifecycle.ErrQuoteMismatched, wantStatus: http.StatusUnprocessableEntity, wantCode: ErrorCodeQuoteMismatched},
		{name: "address unavailable", err: paymentlifecycle.ErrDepositAddressUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: ErrorCodeDepositAddressUnavailable},
		{name: "storage unavailable", err: paymentlifecycle.ErrStorageUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: ErrorCodeStorageUnavailable},
		{name: "unknown is storage", err: errors.New("database host and password must not leak"), wantStatus: http.StatusServiceUnavailable, wantCode: ErrorCodeStorageUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lifecycle := &fakePaymentLifecycle{err: tc.err}
			response := httptest.NewRecorder()
			newTestRouter(lifecycle, uint(41)).ServeHTTP(response, validRequest())
			assertError(t, response, tc.wantStatus, tc.wantCode)
			if bytes.Contains(response.Body.Bytes(), []byte("sensitive")) || bytes.Contains(response.Body.Bytes(), []byte("password")) {
				t.Fatalf("unsafe internal detail leaked: %s", response.Body.String())
			}
		})
	}
}

func newTestRouter(lifecycle paymentlifecycle.PaymentLifecycle, tenant any) http.Handler {
	router := gin.New()
	router.POST("/api/v1/payments", func(c *gin.Context) {
		c.Set(AuthenticatedTenantContextKey, tenant)
		c.Next()
	}, NewHandler(lifecycle).Open)
	return router
}

func validRequest() *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBufferString(validBody()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(IdempotencyKeyHeader, "checkout-20260827-0001")
	return request
}

func validBody() string {
	return `{"merchantReference":"order-1042","invoiceAmount":{"currency":"USD","minorUnits":"1250"},"paymentMethod":{"chainId":"eip155:1","assetId":"eip155:1/slip44:60"},"expiresAt":"2026-08-27T10:45:00Z"}`
}

func validPaymentView(now time.Time) paymentlifecycle.PaymentView {
	return paymentlifecycle.PaymentView{
		InvoiceID:         "invoice-000001",
		TenantID:          "41",
		MerchantReference: "order-1042",
		InvoiceAmount: paymentlifecycle.FiatAmount{
			Currency:   "USD",
			MinorUnits: "900719925474099312345",
		},
		PaymentMethod: paymentlifecycle.PaymentMethod{
			ChainID: "eip155:1",
			AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		Quote: paymentlifecycle.Quote{
			ID:                  "quote-000001",
			InvoiceCurrency:     "USD",
			InvoiceMinorUnits:   "900719925474099312345",
			ChainID:             "eip155:1",
			AssetID:             "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
			RequiredAtomicUnits: "123456789012345678901234567890",
			AssetDecimals:       18,
			RateNumerator:       "123456789012345678901",
			RateDenominator:     "100000000000000000000",
			Source:              "merchant-price-oracle",
			QuotedAt:            now,
			ExpiresAt:           now.Add(5 * time.Minute),
			Rounding:            "ceil",
		},
		DepositAddress: paymentlifecycle.DepositAddress{
			AssignmentID: "assignment-000001",
			Address:      "0x1111111111111111111111111111111111111111",
		},
		State:     paymentlifecycle.InvoiceOpen,
		Revision:  7,
		OpenedAt:  now,
		ExpiresAt: now.Add(30 * time.Minute),
	}
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode ErrorCode) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}
	var envelope struct {
		Error struct {
			Code    ErrorCode `json:"code"`
			Message string    `json:"message"`
		} `json:"error"`
	}
	decodeJSON(t, response, &envelope)
	if envelope.Error.Code != wantCode {
		t.Fatalf("code = %q, want %q; body=%s", envelope.Error.Code, wantCode, response.Body.String())
	}
	if envelope.Error.Message == "" {
		t.Fatal("safe error message is empty")
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body.String())
	}
}
