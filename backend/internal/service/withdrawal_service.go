package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OTPPrompt is returned when a withdrawal requires OTP verification before
// it can proceed to the approval queue.
type OTPPrompt struct {
	OTPRequired bool   `json:"otpRequired"`
	Message     string `json:"message"`
}

// CreateWithdrawalInput is the request body for creating a new withdrawal.
type CreateWithdrawalInput struct {
	BlockchainCode       string          `json:"blockchainCode"`
	CurrencyCode         string          `json:"currencyCode"`
	Amount               decimal.Decimal `json:"amount"`
	ToAddress            string          `json:"toAddress"`
	Memo                 string          `json:"memo,omitempty"`
	CustomerID           string          `json:"customerID,omitempty"`
	Email                string          `json:"email,omitempty"`
	MemberID             uint
	ExternalPlatformID   uint
	BlockchainCurrencyID uint
}

// WithdrawalService is the customer-facing payout service. It validates
// withdrawal requests against per-platform per-currency limits, generates
// OTPs for high-value withdrawals, and manages the approval workflow.
//
// State machine:
//
//	pending-otp → pending-approval → pending → (handoff to WithdrawalProcessingService)
//	                                          → initiated → sent → processed
//	                                                              ↘ failed
type WithdrawalService struct {
	withdrawalRepo    repository.WithdrawalRepository
	blockchainCcyRepo repository.BlockchainCurrencyRepository // resolve blockchain_currency_id from codes
	otpSvc            *OTPService
	eventEmitter      *EventEmitterService
	epbcSvc           *ExternalPlatformBlockchainCurrencyService // set via Pass 2 setter
	blacklistSvc      *AddressBlacklistService                   // set via Pass 2 setter
}

// ErrAddressBlacklisted is returned when the caller attempts to withdraw
// to an address that operators have blocked.
var ErrAddressBlacklisted = errors.New("destination address is blacklisted")

// ErrWithdrawalNotFound deliberately covers both an absent withdrawal and a
// withdrawal owned by another Tenant so the HTTP interface does not disclose
// cross-tenant identifiers.
var ErrWithdrawalNotFound = errors.New("withdrawal not found")

// SetAddressBlacklistService wires the blacklist checker used during Create.
func (s *WithdrawalService) SetAddressBlacklistService(bl *AddressBlacklistService) {
	s.blacklistSvc = bl
}

// fallbackAutoApproveThresholdUSD is used only when no EPBC row exists for
// the (platform, currency) pair AND the operator has chosen to allow this
// path during early bootstrap. The Phase J wiring rejects unconfigured pairs
// outright via ErrCurrencyNotEnabledForPlatform; this fallback exists only
// for tests that don't seed an EPBC row.
const fallbackAutoApproveThresholdUSD = 100

// NewWithdrawalService constructs a WithdrawalService. The EPBC service is
// wired in via SetExternalPlatformBlockchainCurrencyService during Pass 2
// of ServiceRegistry initialization to break the circular dependency.
func NewWithdrawalService(
	withdrawalRepo repository.WithdrawalRepository,
	blockchainCcyRepo repository.BlockchainCurrencyRepository,
	otpSvc *OTPService,
	eventEmitter *EventEmitterService,
) *WithdrawalService {
	return &WithdrawalService{
		withdrawalRepo:    withdrawalRepo,
		blockchainCcyRepo: blockchainCcyRepo,
		otpSvc:            otpSvc,
		eventEmitter:      eventEmitter,
	}
}

// SetExternalPlatformBlockchainCurrencyService wires the EPBC limit service.
// Called by ServiceRegistry Pass 2 to break the circular dependency between
// the two services.
func (s *WithdrawalService) SetExternalPlatformBlockchainCurrencyService(epbc *ExternalPlatformBlockchainCurrencyService) {
	s.epbcSvc = epbc
}

// Create validates a withdrawal request, creates the Withdrawal row, and returns
// an OTPPrompt if OTP verification is required before the request enters the
// approval queue.
func (s *WithdrawalService) Create(input CreateWithdrawalInput) (*models.Withdrawal, *OTPPrompt, error) {
	if input.ToAddress == "" {
		return nil, nil, errors.New("to_address is required")
	}
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, errors.New("amount must be positive")
	}
	if input.BlockchainCode == "" || input.CurrencyCode == "" {
		return nil, nil, errors.New("blockchain_code and currency_code are required")
	}

	if s.blacklistSvc != nil && s.blacklistSvc.IsBlacklisted(input.BlockchainCode, input.ToAddress) {
		return nil, nil, ErrAddressBlacklisted
	}

	// Resolve BlockchainCurrencyID from codes if not already set.
	if input.BlockchainCurrencyID == 0 && s.blockchainCcyRepo != nil {
		bc, err := s.blockchainCcyRepo.GetByBlockchainCodeAndCurrencyCode(input.BlockchainCode, input.CurrencyCode)
		if err != nil {
			return nil, nil, fmt.Errorf("unknown blockchain/currency pair %s/%s", input.BlockchainCode, input.CurrencyCode)
		}
		input.BlockchainCurrencyID = bc.ID
	}

	// Phase J: enforce per-platform per-currency limits via EPBC. When the
	// EPBC service is wired, look up the row for the (platform, currency)
	// pair and reject the withdrawal if the currency is not enabled for the
	// platform. The configured AutoApproveThreshold/Min/Max/HourlyCap/DailyCap
	// override the fallback constants.
	needsOTP := false
	threshold := decimal.NewFromInt(fallbackAutoApproveThresholdUSD)
	if s.epbcSvc != nil {
		epbc, err := s.epbcSvc.GetByPair(input.ExternalPlatformID, input.BlockchainCurrencyID)
		if err != nil {
			return nil, nil, err
		}
		if !epbc.Enabled {
			return nil, nil, ErrCurrencyNotEnabledForPlatform
		}
		if epbc.MinAmount != nil && input.Amount.LessThan(*epbc.MinAmount) {
			return nil, nil, fmt.Errorf("amount below platform minimum %s", epbc.MinAmount.String())
		}
		if epbc.MaxAmount != nil && input.Amount.GreaterThan(*epbc.MaxAmount) {
			return nil, nil, fmt.Errorf("amount above platform maximum %s", epbc.MaxAmount.String())
		}
		if epbc.AutoApproveThreshold != nil {
			threshold = *epbc.AutoApproveThreshold
		}
		if epbc.HourlyCap != nil {
			used, err := s.withdrawalRepo.SumByPlatformAndCurrencyInWindow(
				input.ExternalPlatformID, input.BlockchainCurrencyID,
				time.Now().Add(-1*time.Hour),
			)
			if err != nil {
				return nil, nil, fmt.Errorf("hourly cap query: %w", err)
			}
			if used.Add(input.Amount).GreaterThan(*epbc.HourlyCap) {
				return nil, nil, fmt.Errorf("hourly cap %s exceeded (used %s)", epbc.HourlyCap.String(), used.String())
			}
		}
		if epbc.DailyCap != nil {
			used, err := s.withdrawalRepo.SumByPlatformAndCurrencyInWindow(
				input.ExternalPlatformID, input.BlockchainCurrencyID,
				time.Now().Add(-24*time.Hour),
			)
			if err != nil {
				return nil, nil, fmt.Errorf("daily cap query: %w", err)
			}
			if used.Add(input.Amount).GreaterThan(*epbc.DailyCap) {
				return nil, nil, fmt.Errorf("daily cap %s exceeded (used %s)", epbc.DailyCap.String(), used.String())
			}
		}
	}
	if input.Amount.GreaterThan(threshold) {
		needsOTP = true
	}

	initialState := models.WithdrawalStatePendingApproval
	if needsOTP {
		initialState = models.WithdrawalStatePendingOTP
	}

	refID := generateWithdrawalReference()
	var memoPtr *string
	if input.Memo != "" {
		m := input.Memo
		memoPtr = &m
	}
	var customerIDPtr *string
	if input.CustomerID != "" {
		c := input.CustomerID
		customerIDPtr = &c
	}
	var emailPtr *string
	if input.Email != "" {
		e := input.Email
		emailPtr = &e
	}

	withdrawal := &models.Withdrawal{
		ReferenceID:          refID,
		State:                initialState,
		Email:                emailPtr,
		BlockchainCode:       input.BlockchainCode,
		CurrencyCode:         input.CurrencyCode,
		Amount:               input.Amount,
		ToAddress:            input.ToAddress,
		Memo:                 memoPtr,
		CustomerID:           customerIDPtr,
		MemberID:             input.MemberID,
		ExternalPlatformID:   input.ExternalPlatformID,
		BlockchainCurrencyID: input.BlockchainCurrencyID,
	}

	if err := s.withdrawalRepo.Create(withdrawal); err != nil {
		return nil, nil, fmt.Errorf("create withdrawal: %w", err)
	}

	if needsOTP {
		// Generate OTP for the member.
		if s.otpSvc != nil {
			code, err := s.otpSvc.Generate(input.MemberID, models.OTPPurposeWithdrawalApproval)
			if err != nil {
				return withdrawal, nil, fmt.Errorf("generate OTP: %w", err)
			}
			// Emit OTP email event.
			if s.eventEmitter != nil && input.Email != "" {
				_ = s.eventEmitter.EmitEmail(EmailPayload{
					To:       input.Email,
					Template: "otp",
					Subject:  "Withdrawal OTP",
					Data:     map[string]any{"Code": code},
				})
			}
		}
		return withdrawal, &OTPPrompt{
			OTPRequired: true,
			Message:     "OTP sent to your registered email. Please verify to proceed.",
		}, nil
	}

	return withdrawal, &OTPPrompt{OTPRequired: false}, nil
}

// VerifyOTP validates the OTP code for the given withdrawal and atomically
// advances state from pending-otp → pending-approval. Uses UpdateState which
// is a conditional UPDATE so concurrent callers cannot both succeed.
func (s *WithdrawalService) VerifyOTP(withdrawalID uint, memberID uint, code string) error {
	w, err := s.withdrawalRepo.GetByID(withdrawalID)
	if err != nil {
		return fmt.Errorf("fetch withdrawal: %w", err)
	}
	if w.MemberID != memberID {
		return errors.New("unauthorized")
	}
	if w.State != models.WithdrawalStatePendingOTP {
		return fmt.Errorf("withdrawal is in state %q, expected %q", w.State, models.WithdrawalStatePendingOTP)
	}

	if s.otpSvc == nil {
		return errors.New("OTP service not configured")
	}
	if err := s.otpSvc.Verify(memberID, models.OTPPurposeWithdrawalApproval, code); err != nil {
		return err
	}

	// Conditional UPDATE: only transitions from pending-otp → pending-approval
	// so a concurrent caller racing on the same withdrawal ID cannot win twice.
	return s.withdrawalRepo.UpdateState(withdrawalID,
		models.WithdrawalStatePendingOTP,
		models.WithdrawalStatePendingApproval)
}

// Approve atomically moves the withdrawal from pending-approval → pending so the
// WithdrawalProcessingService can pick it up. Uses a conditional UPDATE so
// concurrent callers cannot both succeed.
func (s *WithdrawalService) Approve(withdrawalID, approvedByMemberID uint) error {
	rowsAffected, err := s.withdrawalRepo.Approve(withdrawalID, approvedByMemberID)
	if err != nil {
		return fmt.Errorf("approve withdrawal: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("withdrawal %d not in pending-approval state (or concurrent modification)", withdrawalID)
	}
	return nil
}

// ApproveForPlatform approves a withdrawal only when it belongs to platformID.
func (s *WithdrawalService) ApproveForPlatform(withdrawalID, platformID, approvedByMemberID uint) error {
	if _, err := s.GetByIDForPlatform(withdrawalID, platformID); err != nil {
		return err
	}
	rowsAffected, err := s.withdrawalRepo.ApproveForPlatform(withdrawalID, platformID, approvedByMemberID)
	if err != nil {
		return fmt.Errorf("approve withdrawal: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("withdrawal %d not in pending-approval state (or concurrent modification)", withdrawalID)
	}
	return nil
}

// Cancel atomically cancels a withdrawal that has not yet been broadcast.
// Allowed from pending-otp, pending-approval, or pending states. Uses a
// conditional UPDATE so concurrent callers cannot both succeed.
func (s *WithdrawalService) Cancel(withdrawalID uint) error {
	rowsAffected, err := s.withdrawalRepo.AtomicCancel(withdrawalID)
	if err != nil {
		return fmt.Errorf("cancel withdrawal: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("withdrawal %d is not in a cancellable state (pending-otp/pending-approval/pending)", withdrawalID)
	}
	return nil
}

// CancelForPlatform cancels a withdrawal only when it belongs to platformID.
func (s *WithdrawalService) CancelForPlatform(withdrawalID, platformID uint) error {
	if _, err := s.GetByIDForPlatform(withdrawalID, platformID); err != nil {
		return err
	}
	rowsAffected, err := s.withdrawalRepo.AtomicCancelForPlatform(withdrawalID, platformID)
	if err != nil {
		return fmt.Errorf("cancel withdrawal: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("withdrawal %d is not in a cancellable state (pending-otp/pending-approval/pending)", withdrawalID)
	}
	return nil
}

// GetByID returns a withdrawal by its primary key.
func (s *WithdrawalService) GetByID(id uint) (*models.Withdrawal, error) {
	return s.withdrawalRepo.GetByID(id)
}

// GetByIDForPlatform is the handler-safe read seam and hides whether a row is
// absent or belongs to another Tenant.
func (s *WithdrawalService) GetByIDForPlatform(id, platformID uint) (*models.Withdrawal, error) {
	w, err := s.withdrawalRepo.GetByIDForPlatform(id, platformID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	return w, nil
}

// GetByReferenceID returns a withdrawal by its reference ID.
func (s *WithdrawalService) GetByReferenceID(referenceID string) (*models.Withdrawal, error) {
	return s.withdrawalRepo.GetByReferenceID(referenceID)
}

// ListByPlatform returns all withdrawals for the given platform.
func (s *WithdrawalService) ListByPlatform(platformID uint, opts ...repository.QueryOption) ([]models.Withdrawal, error) {
	return s.withdrawalRepo.ListByPlatform(platformID, opts...)
}

// generateWithdrawalReference returns a UUID-based unique reference ID.
func generateWithdrawalReference() string {
	return "wd_" + uuid.NewString()
}
