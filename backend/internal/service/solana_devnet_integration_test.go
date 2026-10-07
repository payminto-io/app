//go:build integration

package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// devnetUSDC is Circle's devnet USDC mint, as seeded.
const devnetUSDC = "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"

// TestSolanaDevnet runs the deposit and sweep flow against a real cluster (lagging backends, rate
// limits, real finality), which the local validator cannot reproduce. Needs:
//
//	SOLANA_DEVNET_PAYER_KEY   keypair (base58, JSON array or file:<path>) holding SOL, devnet USDC and the test USDT
//	SOLANA_DEVNET_USDT_MINT   a 6-decimal test mint the payer holds (spl-token create-token --decimals 6)
//	SOLANA_DEVNET_RPC_URL     optional, default https://api.devnet.solana.com
//
// It skips with a message when the SOLANA_DEVNET_* variables are absent.
func TestSolanaDevnet(t *testing.T) {
	payerKey := os.Getenv("SOLANA_DEVNET_PAYER_KEY")
	usdtMintStr := os.Getenv("SOLANA_DEVNET_USDT_MINT")
	if payerKey == "" || usdtMintStr == "" {
		t.Skip("set SOLANA_DEVNET_PAYER_KEY and SOLANA_DEVNET_USDT_MINT to run the devnet test")
	}
	rpcURL := os.Getenv("SOLANA_DEVNET_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://api.devnet.solana.com"
	}
	payer, err := solana.ParseKeypair(payerKey)
	must(t, err)
	usdtMint, err := solana.ParsePublicKey(usdtMintStr)
	must(t, err)
	usdcMint := solana.MustPublicKey(devnetUSDC)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	client := solana.NewClient(&solana.StaticCaller{URL: rpcURL})
	db, cleanup := database.NewTestDB(t)
	defer cleanup()

	// The payer pays, sponsors sweeps and receives them; it must hold both tokens.
	payerUSDC, _ := solana.AssociatedTokenAddress(payer.PublicKey(), usdcMint, solana.TokenProgram)
	payerUSDT, _ := solana.AssociatedTokenAddress(payer.PublicKey(), usdtMint, solana.TokenProgram)
	for _, a := range []string{payerUSDC.String(), payerUSDT.String()} {
		bal, err := client.GetTokenAccountBalance(ctx, a, solana.CommitmentConfirmed)
		if err != nil || bal.Amount == "0" {
			t.Skipf("payer token account %s is empty or missing (%v); fund it first", a, err)
		}
	}

	family := models.BlockchainFamily{Name: "Solana", Code: "sol", Family: "SOL_Family"}
	must(t, db.Create(&family).Error)
	chain := &models.Blockchain{Code: solana.ChainCode, Name: "Solana Devnet", Family: "SOL_Family", BlockchainFamilyID: family.ID, MinConfirmations: 32, Status: "active"}
	must(t, db.Create(chain).Error)
	rows := map[string]*models.BlockchainCurrency{}
	for _, c := range []struct {
		code, standard, mint string
		precision            uint
	}{{"SOL", "native", "", 9}, {"USDC", "SPL", usdcMint.String(), 6}, {"USDT", "SPL", usdtMint.String(), 6}} {
		cur := models.Currency{Code: c.code, Name: c.code, Type: "token", WalletPrecision: c.precision}
		must(t, db.Create(&cur).Error)
		row := &models.BlockchainCurrency{BlockchainID: chain.ID, CurrencyID: cur.ID, CurrencyCode: c.code, BlockchainCode: solana.ChainCode, Standard: c.standard, Address: c.mint, WalletPrecision: c.precision, DepositEnabled: true, Visible: true}
		must(t, db.Create(row).Error)
		rows[c.code] = row
	}
	member := &models.Member{Name: "merchant", MemberType: "merchant", State: "active"}
	must(t, db.Create(member).Error)
	platform := &models.ExternalPlatform{Name: "shop"}
	must(t, db.Create(platform).Error)
	wallet := models.Wallet{Name: "HD Wallet (sol)", Kind: "hd", Status: "active", BlockchainFamilyID: family.ID, MemberID: member.ID}
	must(t, db.Create(&wallet).Error)
	mnemonic, err := crypto.NewMnemonic(128)
	must(t, err)
	seed, _ := crypto.SeedFromMnemonic(mnemonic, "")
	keys := solanaKeys{}
	for i := uint32(0); i < 4; i++ {
		addr, priv, err := crypto.DeriveSolanaAddress(seed, i)
		must(t, err)
		keys[addr] = priv
		must(t, db.Create(&models.AddressPool{Address: addr, PathIndex: uint(i), Status: "available", WalletID: wallet.ID, BlockchainFamilyID: family.ID}).Error)
	}

	accounts := repository.NewSolanaDepositAccountRepository(db)
	depositRepo := repository.NewDepositRepository(db)
	bcRepo := repository.NewBlockchainCurrencyRepository(db)
	missed := repository.NewMissedDepositRepository(db)
	poolSvc := NewAddressPoolService(repository.NewAddressPoolRepository(db), repository.NewWalletRepository(db), repository.NewBlockchainFamilyRepository(db), nil)
	addrSvc := NewDepositAddressService(repository.NewDepositAddressRepository(db), repository.NewWalletRepository(db), bcRepo, nil, poolSvc).WithSolanaDepositAccounts(accounts, db, 0)
	depositSvc := NewDepositService(depositRepo, repository.NewDepositAddressRepository(db), repository.NewPaymentRepository(db), bcRepo, repository.NewBlockchainRepository(db))
	journal := ledger.New(db)
	ledgerSvc := NewLedgerService(repository.NewAccountRepository(db), WithJournal(journal, blockchainCurrencyAssetResolver()))
	watcher := NewSolanaDepositService(db, client, chain, accounts, depositRepo, depositSvc, missed, bcRepo, ledgerSvc, nil, SolanaDepositConfig{})
	sweepRepo := repository.NewSweepRepository(db)
	sweepTxRepo := repository.NewSweepTransactionRepository(db)
	sweepSvc := NewSweepService(db, sweepRepo, sweepTxRepo, repository.NewBlockchainRepository(db), ledgerSvc)
	sweeper := NewSolanaSweepService(db, client, chain, depositRepo, accounts, bcRepo, missed, sweepRepo, sweepTxRepo, sweepSvc,
		NewSweepTransactionService(sweepTxRepo, sweepRepo, ledgerSvc), keys, payer, payer.PublicKey(), journal, SolanaSweepConfig{CloseAccounts: true})

	newPayment := func(ref, amount, currency string) (*models.PaymentRequest, *models.SolanaDepositAccount) {
		pr := &models.PaymentRequest{ReferenceID: ref, AmountInUSD: decimal.RequireFromString(amount), State: models.PaymentStateOpen, MemberID: member.ID, ExternalPlatformID: platform.ID}
		must(t, db.Create(pr).Error)
		da, err := addrSvc.AssignForPayment(pr, solana.ChainCode, currency)
		must(t, err)
		acct, err := accounts.GetByDepositAddressID(da.ID)
		must(t, err)
		return pr, acct
	}
	// Small amounts: 0.25 USDC exact, 0.1 USDT to the owner, 0.3 USDT to a USDC payment (wrong mint), 0.2 + 0.3 USDC in two parts.
	payUSDC, acctA := newPayment("devnet-usdc", "0.25", "USDC")
	payUSDT, acctB := newPayment("devnet-usdt", "0.1", "USDT")
	payWrong, acctC := newPayment("devnet-wrong", "0.3", "USDC")
	payPartial, acctD := newPayment("devnet-partial", "0.5", "USDC")
	sendToken(t, ctx, client, payer, payerUSDC, acctA, usdcMint, 250_000)
	sendToken(t, ctx, client, payer, payerUSDT, acctB, usdtMint, 100_000)
	wrongATA := sendToken(t, ctx, client, payer, payerUSDT, acctC, usdtMint, 300_000)
	sendToken(t, ctx, client, payer, payerUSDC, acctD, usdcMint, 200_000)
	sendToken(t, ctx, client, payer, payerUSDC, acctD, usdcMint, 300_000)

	waitFor(t, ctx, "deposits detected", func() bool {
		_, err := watcher.PollOnce(ctx)
		must(t, err)
		var count int64
		db.Model(&models.Deposit{}).Count(&count)
		return count == 4
	})
	waitFor(t, ctx, "deposits finalized", func() bool {
		_, err := watcher.ConfirmOnce(ctx)
		must(t, err)
		var count int64
		db.Model(&models.Deposit{}).Where("status = ?", models.DepositStatusConfirmed).Count(&count)
		return count == 4
	})
	for _, c := range []struct {
		pr    *models.PaymentRequest
		state string
	}{{payUSDC, models.PaymentStateFilled}, {payUSDT, models.PaymentStateFilled}, {payWrong, models.PaymentStateOpen}, {payPartial, models.PaymentStateFilled}} {
		var p models.PaymentRequest
		must(t, db.First(&p, c.pr.ID).Error)
		if p.State != c.state {
			t.Fatalf("payment %s state = %s, want %s", c.pr.ReferenceID, p.State, c.state)
		}
	}
	anomalies, _ := missed.ListUnresolved()
	if len(anomalies) != 1 || anomalies[0].ToAddress != wrongATA.String() {
		t.Fatalf("anomalies = %+v", anomalies)
	}
	assertJournalAsset(t, db, "deposit", "USDC.SOLANA", 3)
	assertJournalAsset(t, db, "deposit", "USDT.SOLANA", 1)

	if sent, err := sweeper.SweepConfirmed(ctx); err != nil || sent != 2 {
		t.Fatalf("sweeps sent = %d (%v)", sent, err)
	}
	waitFor(t, ctx, "sweeps completed", func() bool {
		_, err := sweeper.TrackConfirmations(ctx)
		must(t, err)
		var count int64
		db.Model(&models.Sweep{}).Where("status = ?", SweepStatusCompleted).Count(&count)
		return count == 2
	})
	assertJournalAsset(t, db, "sweep", "SOL.SOLANA", 2)
	var swept int64
	db.Model(&models.Deposit{}).Where("status = ?", models.DepositStatusSwept).Count(&swept)
	if swept != 4 {
		t.Fatalf("swept deposits = %d", swept)
	}
	if later, _ := missed.ListUnresolved(); len(later) != 1 {
		t.Fatalf("sweep reconciliation raised anomalies: %+v", later)
	}
}
