package service

import (
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// WithSolanaDepositAccounts enables the owner-to-ATA mapping for SOLANA assignments.
func (s *DepositAddressService) WithSolanaDepositAccounts(repo repository.SolanaDepositAccountRepository) *DepositAddressService {
	s.solanaAccounts = repo
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

// createSolanaDepositAddress writes the deposit address (the ATA) and its owner record together.
func (s *DepositAddressService) createSolanaDepositAddress(bc *models.BlockchainCurrency, ownerAddress string, memberID uint, paymentID *uint) (*models.DepositAddress, error) {
	if s.solanaAccounts == nil {
		return nil, errors.New("solana deposit accounts repository not wired")
	}
	acct, err := solanaTokenAccountFor(bc, ownerAddress)
	if err != nil {
		return nil, err
	}
	da := &models.DepositAddress{
		Address:              acct.TokenAccount,
		BlockchainCurrencyID: bc.ID,
		MemberID:             memberID,
		PaymentRequestID:     paymentID,
	}
	if err := s.depositAddressRepo.Create(da); err != nil {
		return nil, fmt.Errorf("create deposit_address: %w", err)
	}
	acct.DepositAddressID = da.ID
	acct.PaymentRequestID = paymentID
	if err := s.solanaAccounts.Create(acct); err != nil {
		return nil, fmt.Errorf("create solana deposit account: %w", err)
	}
	return da, nil
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
