package chaindeposit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Opened is what Payminto's CreatePayment produced for the switch: its reference, the deposit address, expiry.
type Opened struct {
	Reference string
	Address   string
	ExpiresAt *time.Time
}

// PaymentOpener is PaymentService.CreatePayment behind a function, so this package never imports
// internal/service (service/registry.go imports modules, which imports this package). main.go adapts it;
// invoiceID must be persisted in the same insert as the request.
type PaymentOpener func(amountInUSD decimal.Decimal, invoiceID, chainCode, currencyCode string, memberID, platformID uint) (Opened, error)

// PaymintoBackend drives the inherited deposit flow through its existing services; nothing in Payminto changes.
// The attempt id travels as the request's invoice_id, written in the same insert, so a crash after the request
// committed still leaves it findable by attempt.
type PaymintoBackend struct {
	open     PaymentOpener
	repo     repository.PaymentRepository
	deposits repository.DepositRepository
	db       *gorm.DB
}

func NewPaymintoBackend(open PaymentOpener, repo repository.PaymentRepository, deposits repository.DepositRepository, db *gorm.DB) *PaymintoBackend {
	return &PaymintoBackend{open: open, repo: repo, deposits: deposits, db: db}
}

func (b *PaymintoBackend) OpenPayment(_ context.Context, req OpenRequest) (OpenResult, error) {
	if req.AttemptID == "" {
		return OpenResult{}, fmt.Errorf("attempt id required")
	}
	opened, err := b.open(req.AmountInUSD, req.AttemptID, req.ChainCode, req.CurrencyCode, req.MerchantMemberID, req.PlatformID)
	if err != nil {
		return OpenResult{}, err
	}
	if opened.Address == "" || opened.Reference == "" {
		return OpenResult{}, fmt.Errorf("payment %q opened without a deposit address", opened.Reference)
	}
	return OpenResult{Reference: opened.Reference, Address: opened.Address, ExpiresAt: opened.ExpiresAt}, nil
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

// CancelPayment is one conditional UPDATE: the finalizer locks the row to fill it, and a fill that commits between
// a read and an unconditional update must win, so the state is part of the WHERE clause.
func (b *PaymintoBackend) CancelPayment(ctx context.Context, reference string) error {
	res := b.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("reference_id = ? AND deleted_at IS NULL AND state IN ?", reference, []string{models.PaymentStateOpen, models.PaymentStatePartiallyFilled}).
		Update("state", models.PaymentStateCancelled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 1 {
		return nil
	}
	p, err := b.repo.GetByReferenceID(reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBackendNotFound
		}
		return err
	}
	return fmt.Errorf("%w: payment %s is %s", ErrNotCancellable, reference, p.State)
}
