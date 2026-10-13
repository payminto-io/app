package service

import (
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/models"
)

// WithTempoMerchant enables the Tempo leg: every TEMPO payment is paid to this one merchant
// receiving address, told apart by its memo (tempo.MemoForReference of the payment reference).
func (s *DepositAddressService) WithTempoMerchant(address string) *DepositAddressService {
	s.tempoMerchant = address
	return s
}

// TempoEnabled reports whether TEMPO payments can be assigned.
func (s *DepositAddressService) TempoEnabled() bool { return s != nil && s.tempoMerchant != "" }

func isTempo(bc *models.BlockchainCurrency) bool {
	return strings.EqualFold(bc.BlockchainCode, tempo.ChainCode)
}

// assignTempo attaches the Tempo leg to a payment: a deposit_addresses row holding the shared merchant
// address, so the payment reads like any other chain leg. The watcher matches by memo, not address.
func (s *DepositAddressService) assignTempo(bc *models.BlockchainCurrency, payment *models.PaymentRequest) (*models.DepositAddress, error) {
	if s.tempoMerchant == "" {
		return nil, fmt.Errorf("tempo is not configured (TEMPO_MERCHANT_ADDRESS)")
	}
	if _, err := tempo.MemoForReference(payment.ReferenceID); err != nil {
		return nil, err
	}
	paymentID := payment.ID
	da := &models.DepositAddress{Address: s.tempoMerchant, BlockchainCurrencyID: bc.ID, MemberID: payment.MemberID, PaymentRequestID: &paymentID}
	if err := s.depositAddressRepo.Create(da); err != nil {
		return nil, fmt.Errorf("create tempo deposit_address: %w", err)
	}
	rows, err := s.depositAddressRepo.GetByPaymentRequestID(payment.ID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == da.ID {
			return &rows[i], nil
		}
	}
	return da, nil
}
