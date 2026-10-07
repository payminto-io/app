package ledger

import (
	"context"

	"github.com/shopspring/decimal"
)

// Expected pairs a balance some other table stores with the ledger account that should back it.
type Expected struct {
	Account AccountKey
	Stored  decimal.Decimal
}

// Drift is one stored balance the ledger does not support. Derived is the natural balance.
type Drift struct {
	Account AccountKey
	Stored  decimal.Decimal
	Derived decimal.Decimal
}

func (d Drift) Delta() decimal.Decimal { return d.Stored.Sub(d.Derived) }

// NaturalBalance flips the signed debit-positive sum for credit-normal kinds so a liability reads as what is owed.
func NaturalBalance(kind AccountKind, signed decimal.Decimal) decimal.Decimal {
	if kind == KindLiability || kind == KindIncome {
		return signed.Neg()
	}
	return signed
}

// Reconcile reports every stored balance that differs from the ledger. It never writes.
// A missing account derives to zero, so an unposted stored balance is reported, not hidden.
func (s *Service) Reconcile(ctx context.Context, expected []Expected) ([]Drift, error) {
	var drifts []Drift
	for _, e := range expected {
		derived := decimal.Zero
		id, err := s.AccountID(ctx, e.Account)
		switch {
		case err == nil:
			signed, err := s.Balance(ctx, id)
			if err != nil {
				return nil, err
			}
			derived = NaturalBalance(e.Account.Kind, signed)
		case err != ErrAccountNotFound:
			return nil, err
		}
		if !derived.Equal(e.Stored) {
			drifts = append(drifts, Drift{Account: e.Account, Stored: e.Stored, Derived: derived})
		}
	}
	return drifts, nil
}
