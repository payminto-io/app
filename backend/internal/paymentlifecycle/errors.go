package paymentlifecycle

import "fmt"

type ErrorCode string

const (
	CodeInvalidCommand            ErrorCode = "invalid_command"
	CodeUnsupportedPaymentMethod  ErrorCode = "unsupported_payment_method"
	CodeQuoteUnavailable          ErrorCode = "quote_unavailable"
	CodeQuoteExpired              ErrorCode = "quote_expired"
	CodeQuoteMismatched           ErrorCode = "quote_mismatched"
	CodeIdempotencyConflict       ErrorCode = "idempotency_conflict"
	CodeMerchantReferenceConflict ErrorCode = "merchant_reference_conflict"
	CodeDepositAddressUnavailable ErrorCode = "deposit_address_unavailable"
	CodeStorageUnavailable        ErrorCode = "storage_unavailable"
)

type Error struct {
	Code  ErrorCode
	Field string
	Cause error
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("payment lifecycle: %s: %s", e.Code, e.Field)
	}
	return fmt.Sprintf("payment lifecycle: %s", e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

func (e *Error) Is(target error) bool {
	want, ok := target.(*Error)
	return ok && e.Code == want.Code
}

var (
	ErrInvalidCommand            = &Error{Code: CodeInvalidCommand}
	ErrUnsupportedPaymentMethod  = &Error{Code: CodeUnsupportedPaymentMethod}
	ErrQuoteUnavailable          = &Error{Code: CodeQuoteUnavailable}
	ErrQuoteExpired              = &Error{Code: CodeQuoteExpired}
	ErrQuoteMismatched           = &Error{Code: CodeQuoteMismatched}
	ErrIdempotencyConflict       = &Error{Code: CodeIdempotencyConflict}
	ErrMerchantReferenceConflict = &Error{Code: CodeMerchantReferenceConflict}
	ErrDepositAddressUnavailable = &Error{Code: CodeDepositAddressUnavailable}
	ErrStorageUnavailable        = &Error{Code: CodeStorageUnavailable}
)

func failure(code ErrorCode, field string, cause error) error {
	return &Error{Code: code, Field: field, Cause: cause}
}
