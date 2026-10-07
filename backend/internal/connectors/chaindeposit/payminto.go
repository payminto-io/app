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
type PaymintoBackend struct {
	payments PaymentOpener
	repo     repository.PaymentRepository
	deposits repository.DepositRepository
}

func NewPaymintoBackend(payments PaymentOpener, repo repository.PaymentRepository, deposits repository.DepositRepository) *PaymintoBackend {
	return &PaymintoBackend{payments: payments, repo: repo, deposits: deposits}
}

// OpenPayment creates the payment request and assigns a deposit address; Payminto picks the reference id.
func (b *PaymintoBackend) OpenPayment(_ context.Context, req OpenRequest) (OpenResult, error) {
	result, err := b.payments.CreatePayment(service.CreatePaymentInput{
		AmountInUSD:    req.AmountInUSD,
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
func (b *PaymintoBackend) PaymentStatus(_ context.Context, reference string) (PaymentStatus, error) {
	p, err := b.repo.GetByReferenceID(reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PaymentStatus{}, ErrBackendNotFound
		}
		return PaymentStatus{}, err
	}
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
	return PaymentStatus{State: p.State, AmountInUSD: p.AmountInUSD, Received: received}, nil
}

func (b *PaymintoBackend) CancelPayment(_ context.Context, reference string) error {
	p, err := b.repo.GetByReferenceID(reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBackendNotFound
		}
		return err
	}
	if p.State != models.PaymentStateOpen {
		return fmt.Errorf("payment %s is %s", reference, p.State)
	}
	return b.repo.UpdateState(p.ID, models.PaymentStateCancelled)
}
