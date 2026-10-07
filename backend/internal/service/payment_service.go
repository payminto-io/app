package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/metrics"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentService manages the payment request lifecycle: creation, lookup,
// listing, and stale expiry. It optionally wires a DepositAddressService for
// automatic address assignment on payment creation.
type PaymentService struct {
	paymentRepo           repository.PaymentRepository
	depositAddressService *DepositAddressService      // set via SetDepositAddressService
	memberRepo            repository.MemberRepository // set via SetMemberRepo
}

// NewPaymentService constructs a PaymentService backed by the given repository.
func NewPaymentService(paymentRepo repository.PaymentRepository) *PaymentService {
	return &PaymentService{paymentRepo: paymentRepo}
}

// SetDepositAddressService wires the deposit address service for Pass 2
// circular-dep resolution. Called by ServiceRegistry after both services
// are constructed.
func (s *PaymentService) SetDepositAddressService(das *DepositAddressService) {
	s.depositAddressService = das
}

// SetMemberRepo wires the member repository for customer auto-creation.
// Called by ServiceRegistry during Pass 2.
func (s *PaymentService) SetMemberRepo(r repository.MemberRepository) {
	s.memberRepo = r
}

// CreatePaymentInput carries the caller-supplied fields for a new payment request.
type CreatePaymentInput struct {
	AmountInUSD    decimal.Decimal `json:"amountInUSD"`
	CustomerEmail  *string         `json:"customerEmail"`
	CustomerID     *string         `json:"customerID"`
	InvoiceID      *string         `json:"invoiceID"`
	BlockchainCode string          `json:"blockchainCode"`
	CurrencyCode   string          `json:"currencyCode"`
	// ReferenceID, when set, replaces the generated reference; the unique index makes a repeat fail rather than duplicate.
	ReferenceID string `json:"-"`
	// ExpiresIn, when set, replaces the default 30-minute expiry.
	ExpiresIn time.Duration `json:"-"`
}

// CreatePaymentResult carries both the created payment and any auto-assigned
// deposit address. The address is nil when the caller didn't request one.
type CreatePaymentResult struct {
	Payment        *models.PaymentRequest
	DepositAddress *models.DepositAddress
}

// Validate checks that the CreatePaymentInput fields satisfy basic business rules.
func (i *CreatePaymentInput) Validate() error {
	if i.AmountInUSD.LessThanOrEqual(decimal.Zero) {
		return errors.New("amountInUSD must be greater than 0")
	}
	return nil
}

// GenerateReferenceID returns a new UUID v4 string for use as a payment reference.
func GenerateReferenceID() string {
	return uuid.New().String()
}

// CreatePayment creates a new open payment request and optionally assigns a
// deposit address when the caller specifies a blockchain preference.
func (s *PaymentService) CreatePayment(input CreatePaymentInput, memberID, platformID uint) (*CreatePaymentResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	ttl, ref := 30*time.Minute, GenerateReferenceID()
	if input.ExpiresIn > 0 {
		ttl = input.ExpiresIn
	}
	if input.ReferenceID != "" {
		ref = input.ReferenceID
	}
	expiresAt := time.Now().Add(ttl)
	payment := &models.PaymentRequest{
		ReferenceID:        ref,
		AmountInUSD:        input.AmountInUSD,
		State:              models.PaymentStateOpen,
		CustomerEmail:      input.CustomerEmail,
		CustomerID:         input.CustomerID,
		InvoiceID:          input.InvoiceID,
		ExpiresAt:          &expiresAt,
		MemberID:           memberID,
		ExternalPlatformID: platformID,
	}

	if err := s.paymentRepo.Create(payment); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	result := &CreatePaymentResult{Payment: payment}
	metrics.PaymentCreated()

	// Auto-create a customer member record when customerEmail is provided.
	// This mirrors PayRam behaviour: every unique customer email gets a
	// member row with member_type = "customer" for the customer list.
	if input.CustomerEmail != nil && *input.CustomerEmail != "" && s.memberRepo != nil {
		email := *input.CustomerEmail
		_, err := s.memberRepo.GetByEmail(email)
		if err != nil {
			// Customer doesn't exist yet — create one.
			custName := email
			if input.CustomerID != nil && *input.CustomerID != "" {
				custName = *input.CustomerID
			}
			cust := &models.Member{
				Name:       custName,
				Email:      &email,
				MemberType: "customer",
				State:      "active",
			}
			if input.CustomerID != nil && *input.CustomerID != "" {
				cust.CustomerID = *input.CustomerID
			}
			_ = s.memberRepo.Create(cust)
		}
	}

	// Optionally assign a deposit address if caller specified a chain
	if input.BlockchainCode != "" && s.depositAddressService != nil {
		da, err := s.depositAddressService.AssignForPayment(payment, input.BlockchainCode, input.CurrencyCode)
		if err != nil {
			// Roll back: cancel the payment row we just created
			_ = s.paymentRepo.UpdateState(payment.ID, models.PaymentStateCancelled)
			return nil, fmt.Errorf("assign deposit address: %w", err)
		}
		result.DepositAddress = da
	}

	return result, nil
}

// GetPayment fetches a payment request by reference ID scoped to the given platform.
func (s *PaymentService) GetPayment(referenceID string, platformID uint) (*models.PaymentRequest, error) {
	payment, err := s.paymentRepo.GetByReferenceIDAndPlatform(referenceID, platformID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("payment not found: %w", err)
		}
		return nil, fmt.Errorf("payment not found: %w", err)
	}
	return payment, nil
}

// GetByReferenceIDPublic fetches a payment by reference ID without a tenant
// check. Intended for the unauthenticated public widget endpoint; handlers
// MUST only return narrowed projections (never the full row) to avoid
// leaking tenant-internal fields like webhook URLs or API key IDs.
func (s *PaymentService) GetByReferenceIDPublic(referenceID string) (*models.PaymentRequest, error) {
	payment, err := s.paymentRepo.GetByReferenceID(referenceID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}
	return payment, nil
}

// ListPaymentsFilter configures pagination and state filtering for ListPayments.
type ListPaymentsFilter struct {
	State  string
	Limit  int
	Offset int
}

// ListPayments returns a paginated list of payment requests for the given
// platform, together with the total count for pagination UI.
func (s *PaymentService) ListPayments(platformID uint, filter ListPaymentsFilter) ([]models.PaymentRequest, int64, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	total, err := s.paymentRepo.CountByPlatform(platformID, filter.State)
	if err != nil {
		return nil, 0, err
	}

	opts := []repository.QueryOption{
		repository.WithLimit(limit),
		repository.WithOffset(filter.Offset),
	}

	payments, err := s.paymentRepo.ListByPlatformAndState(platformID, filter.State, opts...)
	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// ExpireStalePayments cancels all open payment requests that have passed their
// expiry time. Returns the number of rows transitioned to CANCELLED.
func (s *PaymentService) ExpireStalePayments() (int64, error) {
	return s.paymentRepo.ExpireStale(time.Now())
}
