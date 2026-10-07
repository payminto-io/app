package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

func TestDepositAddressService_SolanaAssignsOwnerATAAndRecordsOwner(t *testing.T) {
	f := newSolanaFixture(t)
	var family models.BlockchainFamily
	must(t, f.db.Where("code = ?", "sol").First(&family).Error)
	wallet := models.Wallet{Name: "HD Wallet (sol)", Kind: "hd", Status: "active", BlockchainFamilyID: family.ID, MemberID: f.member.ID}
	must(t, f.db.Create(&wallet).Error)
	must(t, f.db.Create(&models.AddressPool{Address: fxOwner, PathIndex: 0, Status: "available", WalletID: wallet.ID, BlockchainFamilyID: family.ID}).Error)

	poolSvc := NewAddressPoolService(repository.NewAddressPoolRepository(f.db), repository.NewWalletRepository(f.db), repository.NewBlockchainFamilyRepository(f.db), nil)
	svc := NewDepositAddressService(repository.NewDepositAddressRepository(f.db), repository.NewWalletRepository(f.db), repository.NewBlockchainCurrencyRepository(f.db), nil, poolSvc).
		WithSolanaDepositAccounts(f.accounts)

	pr := &models.PaymentRequest{ReferenceID: "ref-ata", AmountInUSD: decimal.RequireFromString("5"), State: models.PaymentStateOpen, MemberID: f.member.ID, ExternalPlatformID: f.platform.ID}
	must(t, f.db.Create(pr).Error)
	da, err := svc.AssignForPayment(pr, solana.ChainCode, "USDC")
	must(t, err)
	if da.Address != fxUSDCATA {
		t.Fatalf("deposit address = %s, want the owner's USDC ATA %s", da.Address, fxUSDCATA)
	}
	if da.BlockchainCurrency == nil || da.BlockchainCurrency.CurrencyCode != "USDC" {
		t.Fatalf("currency not preloaded: %+v", da)
	}
	acct, err := f.accounts.GetByDepositAddressID(da.ID)
	must(t, err)
	if acct.OwnerAddress != fxOwner || acct.Mint != fxUSDC || acct.TokenProgram != solana.TokenProgram.String() || acct.Decimals != 6 || acct.PaymentRequestID == nil || *acct.PaymentRequestID != pr.ID {
		t.Fatalf("account = %+v", acct)
	}
	if owner, ok := svc.SolanaOwnerAddress(da.ID); !ok || owner != fxOwner {
		t.Fatalf("owner lookup = %q %v", owner, ok)
	}
	// The pool row is consumed: a second assignment has nothing to claim and does not reuse the owner.
	pr2 := &models.PaymentRequest{ReferenceID: "ref-ata-2", AmountInUSD: decimal.RequireFromString("5"), State: models.PaymentStateOpen, MemberID: f.member.ID, ExternalPlatformID: f.platform.ID}
	must(t, f.db.Create(pr2).Error)
	if _, err := svc.AssignForPayment(pr2, solana.ChainCode, "USDT"); err == nil {
		t.Fatal("an empty pool with no wallet service must fail, never reuse an owner")
	}
}

func TestSolanaTokenAccountFor_RejectsUnknownStandard(t *testing.T) {
	bc := &models.BlockchainCurrency{Address: fxUSDC, Standard: "ERC20", CurrencyCode: "USDC", BlockchainCode: solana.ChainCode}
	if _, err := solanaTokenAccountFor(bc, fxOwner); err == nil {
		t.Fatal("ERC20 is not a Solana standard")
	}
	bc.Standard = "SPL-2022"
	acct, err := solanaTokenAccountFor(bc, fxOwner)
	must(t, err)
	if acct.TokenProgram != solana.Token2022Program.String() || acct.TokenAccount == fxUSDCATA {
		t.Fatalf("token-2022 ATA must differ and carry its program: %+v", acct)
	}
}
