package chaindeposit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// MemoryBackend mirrors Payminto's behaviour for tests: it picks the reference itself (the caller cannot), stores
// the attempt id the way the real backend stores it in invoice_id, and moves state the way the finalizer does.
type MemoryBackend struct {
	mu        sync.Mutex
	seq       int
	payments  map[string]*PaymentStatus
	byAttempt map[string]string
	now       func() time.Time
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{payments: map[string]*PaymentStatus{}, byAttempt: map[string]string{}, now: time.Now}
}

func (m *MemoryBackend) OpenPayment(_ context.Context, req OpenRequest) (OpenResult, error) {
	if req.ChainCode == "" || req.CurrencyCode == "" || !req.AmountInUSD.IsPositive() || req.AttemptID == "" {
		return OpenResult{}, fmt.Errorf("invalid open request")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	ref := fmt.Sprintf("ref_%d", m.seq)
	expires := m.now().Add(30 * time.Minute)
	m.payments[ref] = &PaymentStatus{
		Reference: ref, State: "OPEN", AmountInUSD: req.AmountInUSD,
		ChainCode: req.ChainCode, CurrencyCode: req.CurrencyCode, Address: fmt.Sprintf("0xdeposit%04d", m.seq), ExpiresAt: &expires,
	}
	m.byAttempt[req.AttemptID] = ref
	return OpenResult{Reference: ref, Address: m.payments[ref].Address, ExpiresAt: &expires}, nil
}

func (m *MemoryBackend) PaymentStatus(_ context.Context, reference string) (PaymentStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[reference]
	if !ok {
		return PaymentStatus{}, ErrBackendNotFound
	}
	return *p, nil
}

func (m *MemoryBackend) PaymentStatusByAttempt(ctx context.Context, attemptID string) (PaymentStatus, error) {
	m.mu.Lock()
	ref, ok := m.byAttempt[attemptID]
	m.mu.Unlock()
	if !ok {
		return PaymentStatus{}, ErrBackendNotFound
	}
	return m.PaymentStatus(ctx, ref)
}

func (m *MemoryBackend) CancelPayment(_ context.Context, reference string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[reference]
	if !ok {
		return ErrBackendNotFound
	}
	if !strings.EqualFold(p.State, "OPEN") && !strings.EqualFold(p.State, "PARTIALLY_FILLED") {
		return fmt.Errorf("payment %s is %s", reference, p.State)
	}
	p.State = "CANCELLED"
	return nil
}

// Deposit confirms an amount against the payment, moving the state the way Payminto's finalizer does.
func (m *MemoryBackend) Deposit(reference string, amount decimal.Decimal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[reference]
	if !ok {
		return ErrBackendNotFound
	}
	if p.State != "OPEN" && p.State != "PARTIALLY_FILLED" {
		return fmt.Errorf("payment %s is %s", reference, p.State)
	}
	p.Received = p.Received.Add(amount)
	switch diff := p.Received.Sub(p.AmountInUSD); {
	case diff.IsZero():
		p.State = "FILLED"
	case diff.IsPositive():
		p.State = "OVER_FILLED"
	default:
		p.State = "PARTIALLY_FILLED"
	}
	return nil
}
