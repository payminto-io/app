package chaindeposit

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentOpener is the one PaymentService method the backend calls; *service.PaymentService satisfies it.
type PaymentOpener interface {
	CreatePayment(input service.CreatePaymentInput, memberID, platformID uint) (*service.CreatePaymentResult, error)
}

// PaymintoBackend drives the inherited deposit flow through its existing services; nothing in Payminto changes.
// The attempt id travels as CreatePaymentInput.InvoiceID, which CreatePayment persists in the same insert, so a
// crash after the request committed still leaves it findable by attempt.
type PaymintoBackend struct {
	payments PaymentOpener
	repo     repository.PaymentRepository
	deposits repository.DepositRepository
	db       *gorm.DB
}

func NewPaymintoBackend(payments PaymentOpener, repo repository.PaymentRepository, deposits repository.DepositRepository, db *gorm.DB) *PaymintoBackend {
	return &PaymintoBackend{payments: payments, repo: repo, deposits: deposits, db: db}
}

func (b *PaymintoBackend) OpenPayment(_ context.Context, req OpenRequest) (OpenResult, error) {
	if req.AttemptID == "" {
		return OpenResult{}, fmt.Errorf("attempt id required")
	}
	invoice := req.AttemptID
	result, err := b.payments.CreatePayment(service.CreatePaymentInput{
		AmountInUSD:    req.AmountInUSD,
		InvoiceID:      &invoice,
		BlockchainCode: req.ChainCode,
		CurrencyCode:   req.CurrencyCode,
	}, req.MerchantMemberID, req.PlatformID)
	if err != nil {
		return OpenResult{}, err
	}
	if result.DepositAddress == nil {
		return OpenResult{}, fmt.Errorf("payment %s opened without a deposit address", result.Payment.ReferenceID)
	}
	return OpenResult{Reference: result.Payment.ReferenceID, Address: result.DepositAddress.Address, ExpiresAt: result.Payment.ExpiresAt}, nil
}

// PaymentStatus reports the state Payminto's finalizer wrote and the sum of confirmed deposits.
func (b *PaymintoBackend) PaymentStatus(ctx context.Context, reference string) (PaymentStatus, error) {
	p, err := b.repo.GetByReferenceID(reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PaymentStatus{}, ErrBackendNotFound
		}
		return PaymentStatus{}, err
	}
	return b.status(ctx, p)
}

func (b *PaymintoBackend) PaymentStatusByAttempt(ctx context.Context, attemptID string) (PaymentStatus, error) {
	var p models.PaymentRequest
	err := b.db.WithContext(ctx).Where("invoice_id = ?", attemptID).Order("id").First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PaymentStatus{}, ErrBackendNotFound
		}
		return PaymentStatus{}, err
	}
	return b.status(ctx, &p)
}

func (b *PaymintoBackend) status(ctx context.Context, p *models.PaymentRequest) (PaymentStatus, error) {
	deposits, err := b.deposits.ListByPaymentRequestID(p.ID)
	if err != nil {
		return PaymentStatus{}, err
	}
	received := decimal.Zero
	for _, d := range deposits {
		if d.Status == models.DepositStatusConfirmed || d.Status == models.DepositStatusSwept {
			received = received.Add(d.Amount)
		}
	}
	st := PaymentStatus{Reference: p.ReferenceID, State: p.State, AmountInUSD: p.AmountInUSD, Received: received, ExpiresAt: p.ExpiresAt}
	var addr models.DepositAddress
	err = b.db.WithContext(ctx).Preload("BlockchainCurrency").Where("payment_request_id = ?", p.ID).Order("id").First(&addr).Error
	if err == nil {
		st.Address = addr.Address
		if addr.BlockchainCurrency != nil {
			st.CurrencyCode = addr.BlockchainCurrency.CurrencyCode
			st.ChainCode = addr.BlockchainCurrency.BlockchainCode
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return PaymentStatus{}, err
	}
	return st, nil
}

func (b *PaymintoBackend) CancelPayment(_ context.Context, reference string) error {
	p, err := b.repo.GetByReferenceID(reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBackendNotFound
		}
		return err
	}
	if p.State != models.PaymentStateOpen && p.State != models.PaymentStatePartiallyFilled {
		return fmt.Errorf("payment %s is %s", reference, p.State)
	}
	return b.repo.UpdateState(p.ID, models.PaymentStateCancelled)
}
