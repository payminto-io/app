package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// WithSolanaDepositAccounts enables the owner-to-ATA mapping for SOLANA assignments. db opens the
// transaction both rows are written in; lateWindow is how long after payment expiry the account is watched.
func (s *DepositAddressService) WithSolanaDepositAccounts(repo repository.SolanaDepositAccountRepository, db *gorm.DB, lateWindow time.Duration) *DepositAddressService {
	s.solanaAccounts = repo
	s.solanaDB = db
	if lateWindow <= 0 {
		lateWindow = 7 * 24 * time.Hour
	}
	s.solanaLateWindow = lateWindow
	return s
}

// isSolanaToken reports whether assignments for bc need an associated token account.
func isSolanaToken(bc *models.BlockchainCurrency) bool {
	return bc != nil && bc.BlockchainCode == solana.ChainCode && bc.Address != ""
}

// solanaTokenAccountFor derives the ATA that becomes the payment's deposit address.
func solanaTokenAccountFor(bc *models.BlockchainCurrency, ownerAddress string) (*models.SolanaDepositAccount, error) {
	owner, err := solana.ParsePublicKey(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("solana owner address: %w", err)
	}
	mint, err := solana.ParsePublicKey(bc.Address)
	if err != nil {
		return nil, fmt.Errorf("solana mint for %s: %w", bc.CurrencyCode, err)
	}
	program, err := solana.TokenProgramFor(bc.Standard)
	if err != nil {
		return nil, err
	}
	ata, err := solana.AssociatedTokenAddress(owner, mint, program)
	if err != nil {
		return nil, err
	}
	return &models.SolanaDepositAccount{
		BlockchainCurrencyID: bc.ID,
		OwnerAddress:         ownerAddress,
		TokenAccount:         ata.String(),
		Mint:                 bc.Address,
		TokenProgram:         program.String(),
		Decimals:             uint8(bc.WalletPrecision),
		Status:               models.SolanaDepositAccountWatching,
	}, nil
}

// createSolanaDepositAddress writes the deposit address (the ATA) and its owner record in one
// transaction; on failure the pool row is returned so no payment carries an unwatched address.
func (s *DepositAddressService) createSolanaDepositAddress(bc *models.BlockchainCurrency, pool *models.AddressPool, payment *models.PaymentRequest) (*models.DepositAddress, error) {
	if s.solanaAccounts == nil || s.solanaDB == nil {
		return nil, errors.New("solana deposit accounts repository not wired")
	}
	acct, err := solanaTokenAccountFor(bc, pool.Address)
	if err != nil {
		s.releasePool(pool)
		return nil, err
	}
	paymentID := payment.ID
	now := time.Now()
	expires := now.Add(s.solanaLateWindow)
	if payment.ExpiresAt != nil {
		acct.PaymentExpiresAt = payment.ExpiresAt
		expires = payment.ExpiresAt.Add(s.solanaLateWindow)
	}
	acct.WatchUntil = &expires
	acct.PaymentRequestID = &paymentID
	da := &models.DepositAddress{Address: acct.TokenAccount, BlockchainCurrencyID: bc.ID, MemberID: payment.MemberID, PaymentRequestID: &paymentID}
	err = s.solanaDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(da).Error; err != nil {
			return fmt.Errorf("create deposit_address: %w", err)
		}
		acct.DepositAddressID = da.ID
		return s.solanaAccounts.WithTx(tx).Create(acct)
	})
	if err != nil {
		s.releasePool(pool)
		return nil, err
	}
	return da, nil
}

// releasePool gives a claimed pool row back after a failed assignment.
func (s *DepositAddressService) releasePool(pool *models.AddressPool) {
	if s.solanaDB == nil || pool == nil {
		return
	}
	s.solanaDB.Model(&models.AddressPool{}).Where("id = ?", pool.ID).Update("status", "available")
}

// SolanaOwnerAddress returns the owner keypair address behind a Solana deposit address, for the
// checkout to show next to the token account (payers often send to the owner).
func (s *DepositAddressService) SolanaOwnerAddress(depositAddressID uint) (string, bool) {
	if s.solanaAccounts == nil {
		return "", false
	}
	acct, err := s.solanaAccounts.GetByDepositAddressID(depositAddressID)
	if err != nil || errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false
	}
	return acct.OwnerAddress, true
}
