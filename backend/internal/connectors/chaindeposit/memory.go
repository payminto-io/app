package chaindeposit

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/shopspring/decimal"
)

// MemoryBackend mirrors Payminto's state machine for tests: deposits are credited with Deposit and the
// state follows FinalizeFromConfirmedDeposits (filled, partially_filled, over_filled).
type MemoryBackend struct {
	mu       sync.Mutex
	seq      int
	payments map[string]*PaymentStatus
}

func NewMemoryBackend() *MemoryBackend { return &MemoryBackend{payments: map[string]*PaymentStatus{}} }

func (m *MemoryBackend) OpenPayment(_ context.Context, req OpenRequest) (OpenResult, error) {
	if req.ChainCode == "" || req.CurrencyCode == "" || !req.AmountInUSD.IsPositive() {
		return OpenResult{}, fmt.Errorf("invalid open request")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	ref := req.Reference
	if ref == "" {
		ref = fmt.Sprintf("ref_%d", m.seq)
	}
	m.payments[ref] = &PaymentStatus{State: "OPEN", AmountInUSD: req.AmountInUSD}
	return OpenResult{Reference: ref, Address: fmt.Sprintf("0xdeposit%04d", m.seq)}, nil
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

func (m *MemoryBackend) CancelPayment(_ context.Context, reference string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[reference]
	if !ok {
		return ErrBackendNotFound
	}
	if !strings.EqualFold(p.State, "OPEN") {
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
