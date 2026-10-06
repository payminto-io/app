# dto

Owns the request and response shapes for Payminto's public HTTP API. This is the contract boundary between handlers and external callers: every struct here is what a merchant, dashboard, or widget actually sees on the wire. Exposes types like `CreatePaymentRequest`, `CreatePaymentResponse`, and `PaymentResponse`, plus mapping helpers such as `ToPaymentResponse` that project domain models into DTOs. Depends on `internal/models` for source types and `shopspring/decimal` for monetary fields; has no other dependencies. Handlers import this package to decode input, call services, and encode output — keeping models out of the JSON layer.

## Files

- `payment_dto.go` — payment request/response DTOs and `ToPaymentResponse` mapper.
- `payment_dto_test.go` — round-trip JSON and mapper tests.

## See also

- `internal/api/handler` — handlers that consume these DTOs
- `internal/models` — domain types converted into DTOs
