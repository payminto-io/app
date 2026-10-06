package service

import (
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// ErrMissedDepositNotFound signals an unknown missed deposit row.
var ErrMissedDepositNotFound = errors.New("missed deposit not found")

// ErrMissedDepositAlreadyResolved signals a double-resolve attempt.
var ErrMissedDepositAlreadyResolved = errors.New("missed deposit already resolved")

// Resolve actions accepted by MissedDepositService.Resolve.
const (
	MissedDepositActionRefund            = "refunded"
	MissedDepositActionClaimAsRevenue    = "claimed_as_revenue"
	MissedDepositActionLinkToPayment     = "linked_to_payment"
	MissedDepositActionIgnored           = "ignored"
)

// MissedDepositService is the operator-facing facade for the missed_deposits
// table. The BlockchainProcessor writes rows when an on-chain transaction
// hits an address Payminto does not own; an admin then reconciles them via
// the dashboard.
type MissedDepositService struct {
	repo repository.MissedDepositRepository
}

// NewMissedDepositService wires the service.
func NewMissedDepositService(repo repository.MissedDepositRepository) *MissedDepositService {
	return &MissedDepositService{repo: repo}
}

// Create persists a new missed deposit row. Called by BlockchainProcessor.
func (s *MissedDepositService) Create(d *models.MissedDeposit) error {
	if d == nil {
		return errors.New("missed deposit is nil")
	}
	if d.TxHash == "" {
		return errors.New("tx hash is required")
	}
	if d.BlockchainID == 0 {
		return errors.New("blockchain id is required")
	}
	if d.Status == "" {
		d.Status = models.MissedDepositStatusPending
	}
	return s.repo.Create(d)
}

// GetByID returns a missed deposit by primary key.
func (s *MissedDepositService) GetByID(id uint) (*models.MissedDeposit, error) {
	d, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMissedDepositNotFound
		}
		return nil, err
	}
	return d, nil
}

// ListPending returns all unresolved missed deposits.
func (s *MissedDepositService) ListPending() ([]models.MissedDeposit, error) {
	return s.repo.ListUnresolved()
}

// ListByStatus returns missed deposits filtered by status.
func (s *MissedDepositService) ListByStatus(status string) ([]models.MissedDeposit, error) {
	return s.repo.ListByStatus(status)
}

// Resolve marks a missed deposit as handled with the given action and
// optional reason. Returns ErrMissedDepositAlreadyResolved when the row is
// not in the pending state.
func (s *MissedDepositService) Resolve(id uint, action, reason string, resolvedBy uint) error {
	var status string
	switch action {
	case MissedDepositActionRefund:
		status = models.MissedDepositStatusRefunded
	case MissedDepositActionClaimAsRevenue:
		status = models.MissedDepositStatusClaimed
	case MissedDepositActionLinkToPayment:
		status = models.MissedDepositStatusLinked
	case MissedDepositActionIgnored:
		status = models.MissedDepositStatusIgnored
	default:
		return fmt.Errorf("unknown resolve action %q", action)
	}

	// Existence check: distinguish "not found" from "already resolved".
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	resolution := action
	if reason != "" {
		resolution = action + ": " + reason
	}
	rows, err := s.repo.MarkResolved(id, status, resolution, resolvedBy)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMissedDepositAlreadyResolved
	}
	return nil
}
