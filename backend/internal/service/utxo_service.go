package service

import (
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// UTXOService provides CRUD operations for UTXO records used by the Bitcoin
// sweep pipeline.
type UTXOService struct {
	utxoRepo repository.UTXORepository
}

// NewUTXOService constructs a UTXOService backed by the given repository.
func NewUTXOService(utxoRepo repository.UTXORepository) *UTXOService {
	return &UTXOService{utxoRepo: utxoRepo}
}

// RecordUTXO idempotently creates a UTXO record. If the (txID, vout) pair
// already exists, it returns the existing record without error.
func (s *UTXOService) RecordUTXO(txID string, vout uint, amount decimal.Decimal, address string, depositID *uint) (*models.UTXO, error) {
	existing, err := s.utxoRepo.GetByTxIDAndVout(txID, vout)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("record utxo: duplicate check: %w", err)
	}

	u := &models.UTXO{
		TxID:      txID,
		Vout:      vout,
		Amount:    amount,
		Address:   address,
		Spent:     false,
		DepositID: depositID,
	}
	if err := s.utxoRepo.Create(u); err != nil {
		return nil, fmt.Errorf("record utxo: create: %w", err)
	}
	return u, nil
}

// GetByID fetches a UTXO by primary key.
func (s *UTXOService) GetByID(id uint) (*models.UTXO, error) {
	return s.utxoRepo.GetByID(id)
}

// ListUnspentByAddress returns all unspent UTXOs for a Bitcoin address.
func (s *UTXOService) ListUnspentByAddress(address string) ([]models.UTXO, error) {
	return s.utxoRepo.ListUnspentByAddress(address)
}

// ListUnspent returns all unspent UTXOs across all addresses.
func (s *UTXOService) ListUnspent() ([]models.UTXO, error) {
	return s.utxoRepo.ListUnspent()
}

// MarkSpent marks a UTXO as consumed by a sweep transaction.
func (s *UTXOService) MarkSpent(id uint, spentTxID string) error {
	return s.utxoRepo.MarkSpent(id, spentTxID)
}
