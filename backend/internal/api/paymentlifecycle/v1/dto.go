package paymentlifecyclev1

// OpenPaymentRequest is the version-one wire contract for opening an Invoice.
// Tenant identity and the Idempotency Key are deliberately absent: the former
// comes from authentication context and the latter from Idempotency-Key.
type OpenPaymentRequest struct {
	MerchantReference string               `json:"merchantReference"`
	InvoiceAmount     InvoiceAmountRequest `json:"invoiceAmount"`
	PaymentMethod     PaymentMethodRequest `json:"paymentMethod"`
	ExpiresAt         string               `json:"expiresAt"`
}

type InvoiceAmountRequest struct {
	Currency   string `json:"currency"`
	MinorUnits string `json:"minorUnits"`
}

type PaymentMethodRequest struct {
	ChainID string `json:"chainId"`
	AssetID string `json:"assetId"`
}

type OpenPaymentEnvelope struct {
	Payment PaymentResponse `json:"payment"`
}

type PaymentResponse struct {
	InvoiceID         string                 `json:"invoiceId"`
	MerchantReference string                 `json:"merchantReference"`
	InvoiceAmount     InvoiceAmountResponse  `json:"invoiceAmount"`
	PaymentMethod     PaymentMethodResponse  `json:"paymentMethod"`
	Quote             QuoteResponse          `json:"quote"`
	DepositAddress    DepositAddressResponse `json:"depositAddress"`
	State             string                 `json:"state"`
	Revision          uint64                 `json:"revision"`
	OpenedAt          string                 `json:"openedAt"`
	ExpiresAt         string                 `json:"expiresAt"`
}

type InvoiceAmountResponse struct {
	Currency   string `json:"currency"`
	MinorUnits string `json:"minorUnits"`
}

type PaymentMethodResponse struct {
	ChainID string `json:"chainId"`
	AssetID string `json:"assetId"`
}

type QuoteResponse struct {
	ID                  string `json:"id"`
	InvoiceCurrency     string `json:"invoiceCurrency"`
	InvoiceMinorUnits   string `json:"invoiceMinorUnits"`
	ChainID             string `json:"chainId"`
	AssetID             string `json:"assetId"`
	RequiredAtomicUnits string `json:"requiredAtomicUnits"`
	AssetDecimals       uint8  `json:"assetDecimals"`
	RateNumerator       string `json:"rateNumerator"`
	RateDenominator     string `json:"rateDenominator"`
	Source              string `json:"source"`
	QuotedAt            string `json:"quotedAt"`
	ExpiresAt           string `json:"expiresAt"`
	Rounding            string `json:"rounding"`
}

type DepositAddressResponse struct {
	AssignmentID string `json:"assignmentId"`
	Address      string `json:"address"`
}

type ErrorCode string

const (
	ErrorCodeInvalidRequest            ErrorCode = "invalid_request"
	ErrorCodeUnauthenticated           ErrorCode = "unauthenticated"
	ErrorCodeUnsupportedPaymentMethod  ErrorCode = "unsupported_payment_method"
	ErrorCodeIdempotencyConflict       ErrorCode = "idempotency_conflict"
	ErrorCodeMerchantReferenceConflict ErrorCode = "merchant_reference_conflict"
	ErrorCodeQuoteUnavailable          ErrorCode = "quote_unavailable"
	ErrorCodeQuoteExpired              ErrorCode = "quote_expired"
	ErrorCodeQuoteMismatched           ErrorCode = "quote_mismatched"
	ErrorCodeDepositAddressUnavailable ErrorCode = "deposit_address_unavailable"
	ErrorCodeStorageUnavailable        ErrorCode = "storage_unavailable"
)

type ErrorEnvelope struct {
	Error ErrorResponse `json:"error"`
}

type ErrorResponse struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}
