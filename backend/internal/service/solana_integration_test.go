//go:build integration

package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TestSolanaEndToEnd runs a local validator: creates USDC-like and USDT-like mints, assigns
// deposit addresses through the HD pool, pays (to the ATA, to the owner, the wrong mint, in two
// parts), detects, finalizes with ledger journals, sweeps with a sponsored fee payer and closes
// the deposit accounts. Skips only when solana-test-validator is not installed.
func TestSolanaEndToEnd(t *testing.T) {
	bin, err := exec.LookPath("solana-test-validator")
	if err != nil {
		t.Skip("solana-test-validator is not installed; install the Solana CLI (https://docs.solanalabs.com/cli/install) to run this test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	rpcURL := startValidator(t, bin)
	client := solana.NewClient(&solana.StaticCaller{URL: rpcURL})
	db, cleanup := database.NewTestDB(t)
	defer cleanup()

	feePayer := newSigner(t)
	payer := newSigner(t)
	hot := newSigner(t)
	airdrop(t, ctx, client, feePayer.PublicKey(), 10_000_000_000)
	airdrop(t, ctx, client, payer.PublicKey(), 5_000_000_000)

	usdcMint := createMint(t, ctx, client, feePayer, 6)
	usdtMint := createMint(t, ctx, client, feePayer, 6)
	payerUSDC := fundTokenAccount(t, ctx, client, feePayer, payer.PublicKey(), usdcMint, 1_000_000_000)
	payerUSDT := fundTokenAccount(t, ctx, client, feePayer, payer.PublicKey(), usdtMint, 1_000_000_000)

	// Catalogue rows: the mints are data, as in the seeds.
	family := models.BlockchainFamily{Name: "Solana", Code: "sol", Family: "SOL_Family"}
	must(t, db.Create(&family).Error)
	chain := &models.Blockchain{Code: solana.ChainCode, Name: "Solana Local", Family: "SOL_Family", BlockchainFamilyID: family.ID, MinConfirmations: 32, Status: "active"}
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

	// Owner keys from the HD path; the key provider replays the same derivation (what KeyResolver does from the vault).
	seed, _ := crypto.SeedFromMnemonic("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
	keys := solanaKeys{}
	for i := uint32(0); i < 6; i++ {
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
	addrSvc := NewDepositAddressService(repository.NewDepositAddressRepository(db), repository.NewWalletRepository(db), bcRepo, nil, poolSvc).WithSolanaDepositAccounts(accounts)
	depositSvc := NewDepositService(depositRepo, repository.NewDepositAddressRepository(db), repository.NewPaymentRepository(db), bcRepo, repository.NewBlockchainRepository(db))
	journal := ledger.New(db)
	ledgerSvc := NewLedgerService(repository.NewAccountRepository(db), WithJournal(journal, blockchainCurrencyAssetResolver()))
	watcher := NewSolanaDepositService(db, client, chain, accounts, depositRepo, depositSvc, missed, bcRepo, ledgerSvc, nil, SolanaDepositConfig{})
	sweepRepo := repository.NewSweepRepository(db)
	sweepTxRepo := repository.NewSweepTransactionRepository(db)
	sweepSvc := NewSweepService(db, sweepRepo, sweepTxRepo, repository.NewBlockchainRepository(db), ledgerSvc)
	sweeper := NewSolanaSweepService(db, client, chain, depositRepo, accounts, bcRepo, sweepRepo, sweepTxRepo, sweepSvc,
		NewSweepTransactionService(sweepTxRepo, sweepRepo, ledgerSvc), keys, feePayer, hot.PublicKey(), journal,
		SolanaSweepConfig{CloseAccounts: true, Send: solana.SendOptions{Poll: 500 * time.Millisecond, Wait: 60 * time.Second}})

	newPayment := func(ref, amount, currency string) (*models.PaymentRequest, *models.SolanaDepositAccount) {
		pr := &models.PaymentRequest{ReferenceID: ref, AmountInUSD: decimal.RequireFromString(amount), State: models.PaymentStateOpen, MemberID: member.ID, ExternalPlatformID: platform.ID}
		must(t, db.Create(pr).Error)
		da, err := addrSvc.AssignForPayment(pr, solana.ChainCode, currency)
		must(t, err)
		acct, err := accounts.GetByDepositAddressID(da.ID)
		must(t, err)
		return pr, acct
	}
	payUSDC, acctA := newPayment("usdc-exact", "25", "USDC")
	payUSDT, acctB := newPayment("usdt-owner", "10", "USDT")
	payWrong, acctC := newPayment("usdc-wrong-mint", "30", "USDC")
	payPartial, acctD := newPayment("usdc-partial", "50", "USDC")

	// A: 25 USDC to the token account (the wallet creates it on the payer's rent).
	sendToken(t, ctx, client, payer, payerUSDC, acctA, usdcMint, 25_000_000)
	// B: 10 USDT "to the owner": on chain this is the same thing, the owner's ATA gets created and credited.
	sendToken(t, ctx, client, payer, payerUSDT, acctB, usdtMint, 10_000_000)
	// C: USDT sent to a USDC payment's owner: lands in the owner's USDT ATA, must become an anomaly.
	wrongATA := sendToken(t, ctx, client, payer, payerUSDT, acctC, usdtMint, 30_000_000)
	// D: 20 then 30 USDC.
	sendToken(t, ctx, client, payer, payerUSDC, acctD, usdcMint, 20_000_000)
	sendToken(t, ctx, client, payer, payerUSDC, acctD, usdcMint, 30_000_000)

	waitFor(t, ctx, "deposits detected", func() bool {
		n, err := watcher.PollOnce(ctx)
		must(t, err)
		_ = n
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
	anomalies, err := missed.ListUnresolved()
	must(t, err)
	if len(anomalies) != 1 || anomalies[0].ToAddress != wrongATA.String() || anomalies[0].BlockchainCurrencyID == nil || *anomalies[0].BlockchainCurrencyID != rows["USDT"].ID {
		t.Fatalf("anomalies = %+v", anomalies)
	}
	assertJournalAsset(t, db, "deposit", "USDC.SOLANA", 3)
	assertJournalAsset(t, db, "deposit", "USDT.SOLANA", 1)

	// Sweep: one USDC transaction (A and D) and one USDT transaction (B), fees paid by the fee payer.
	feeBefore, err := client.GetBalance(ctx, feePayer.PublicKey().String(), solana.CommitmentConfirmed)
	must(t, err)
	sent, err := sweeper.SweepConfirmed(ctx)
	must(t, err)
	if sent != 2 {
		t.Fatalf("sweeps sent = %d, want 2", sent)
	}
	waitFor(t, ctx, "sweeps completed", func() bool {
		_, err := sweeper.TrackConfirmations(ctx)
		must(t, err)
		var count int64
		db.Model(&models.Sweep{}).Where("status = ?", SweepStatusCompleted).Count(&count)
		return count == 2
	})
	hotUSDC, _ := solana.AssociatedTokenAddress(hot.PublicKey(), usdcMint, solana.TokenProgram)
	hotUSDT, _ := solana.AssociatedTokenAddress(hot.PublicKey(), usdtMint, solana.TokenProgram)
	if bal, _ := client.GetTokenAccountBalance(ctx, hotUSDC.String(), solana.CommitmentConfirmed); bal.Amount != "75000000" {
		t.Fatalf("hot USDC = %s, want 75000000", bal.Amount)
	}
	if bal, _ := client.GetTokenAccountBalance(ctx, hotUSDT.String(), solana.CommitmentConfirmed); bal.Amount != "10000000" {
		t.Fatalf("hot USDT = %s, want 10000000", bal.Amount)
	}
	for _, acct := range []*models.SolanaDepositAccount{acctA, acctB, acctD} {
		if info, _ := client.GetAccountInfo(ctx, acct.TokenAccount, solana.CommitmentConfirmed); info != nil {
			t.Fatalf("deposit account %s not closed", acct.TokenAccount)
		}
	}
	feeAfter, _ := client.GetBalance(ctx, feePayer.PublicKey().String(), solana.CommitmentConfirmed)
	var sweeps []models.Sweep
	must(t, db.Find(&sweeps).Error)
	gas := decimal.Zero
	for _, s := range sweeps {
		gas = gas.Add(s.TotalGasFee)
	}
	// Fee payer paid the fees and hot ATA rent, got the three deposit ATA rents back.
	if gas.IsZero() || feeAfter >= feeBefore+3*2_039_280 {
		t.Fatalf("fee payer balance %d -> %d, booked gas %s", feeBefore, feeAfter, gas)
	}
	assertJournalAsset(t, db, "sweep", "SOL.SOLANA", 2)
	assertJournalAsset(t, db, "solana_rent", "SOL.SOLANA", 2)
	var swept int64
	db.Model(&models.Deposit{}).Where("status = ?", models.DepositStatusSwept).Count(&swept)
	if swept != 4 {
		t.Fatalf("swept deposits = %d", swept)
	}
}

func assertJournalAsset(t *testing.T, db *gorm.DB, referenceType, asset string, journals int64) {
	t.Helper()
	var n int64
	must(t, db.Table("ledger_journals").
		Joins("JOIN ledger_lines ON ledger_lines.journal_id = ledger_journals.id").
		Where("ledger_journals.reference_type = ? AND ledger_lines.asset = ?", referenceType, asset).
		Distinct("ledger_journals.id").Count(&n).Error)
	if n != journals {
		t.Fatalf("%s journals in %s = %d, want %d", referenceType, asset, n, journals)
	}
}

func newSigner(t *testing.T) solana.Ed25519Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	must(t, err)
	s, err := solana.NewEd25519Signer(priv)
	must(t, err)
	return s
}

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	must(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func startValidator(t *testing.T, bin string) string {
	t.Helper()
	rpc := freePort(t)
	faucet := freePort(t)
	gossip := freePort(t)
	ledgerDir := filepath.Join(t.TempDir(), "ledger")
	cmd := exec.Command(bin, "--reset", "--quiet", "--ledger", ledgerDir, "--bind-address", "127.0.0.1",
		"--rpc-port", strconv.Itoa(rpc), "--faucet-port", strconv.Itoa(faucet), "--gossip-port", strconv.Itoa(gossip),
		"--dynamic-port-range", fmt.Sprintf("%d-%d", gossip+1, gossip+30), "--ticks-per-slot", "8")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	must(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	url := fmt.Sprintf("http://127.0.0.1:%d", rpc)
	client := solana.NewClient(&solana.StaticCaller{URL: url})
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if h, err := client.GetHealth(context.Background()); err == nil && h == "ok" {
			if slot, err := client.GetSlot(context.Background(), solana.CommitmentConfirmed); err == nil && slot > 2 {
				return url
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("solana-test-validator did not become healthy in 90s")
	return ""
}

func airdrop(t *testing.T, ctx context.Context, c *solana.Client, to solana.PublicKey, lamports uint64) {
	t.Helper()
	sig, err := c.RequestAirdrop(ctx, to.String(), lamports)
	must(t, err)
	_, err = c.WaitForSignature(ctx, sig, 0, solana.SendOptions{Commitment: solana.CommitmentConfirmed, Poll: 300 * time.Millisecond, Wait: 60 * time.Second})
	must(t, err)
}

func send(t *testing.T, ctx context.Context, c *solana.Client, feePayer solana.Signer, ixs []solana.Instruction, signers ...solana.Signer) string {
	t.Helper()
	build := func(bh string) (solana.Message, error) {
		return solana.CompileMessage(feePayer.PublicKey(), solana.MustPublicKey(bh), ixs)
	}
	res, err := solana.SendAndConfirm(ctx, c, build, append([]solana.Signer{feePayer}, signers...), solana.SendOptions{Poll: 300 * time.Millisecond, Wait: 60 * time.Second})
	must(t, err)
	return res.Signature
}

func createMint(t *testing.T, ctx context.Context, c *solana.Client, authority solana.Ed25519Signer, decimals uint8) solana.PublicKey {
	t.Helper()
	mint := newSigner(t)
	rent, err := c.GetMinimumBalanceForRentExemption(ctx, solana.MintAccountSize)
	must(t, err)
	send(t, ctx, c, authority, []solana.Instruction{
		solana.SystemCreateAccount(authority.PublicKey(), mint.PublicKey(), rent, solana.MintAccountSize, solana.TokenProgram),
		solana.TokenInitializeMint2(solana.TokenProgram, mint.PublicKey(), decimals, authority.PublicKey()),
	}, mint)
	info, err := c.GetMint(ctx, mint.PublicKey().String())
	must(t, err)
	if info.Decimals != decimals || info.TokenProgram != solana.TokenProgram {
		t.Fatalf("mint info = %+v", info)
	}
	return mint.PublicKey()
}

func fundTokenAccount(t *testing.T, ctx context.Context, c *solana.Client, authority solana.Ed25519Signer, owner, mint solana.PublicKey, amount uint64) solana.PublicKey {
	t.Helper()
	create, ata, err := solana.CreateAssociatedTokenAccountIdempotent(authority.PublicKey(), owner, mint, solana.TokenProgram)
	must(t, err)
	send(t, ctx, c, authority, []solana.Instruction{create, solana.TokenMintTo(solana.TokenProgram, mint, ata, authority.PublicKey(), amount)})
	return ata
}

// sendToken pays like a wallet does: create the recipient owner's ATA for the mint if missing, then transferChecked.
func sendToken(t *testing.T, ctx context.Context, c *solana.Client, payer solana.Ed25519Signer, source solana.PublicKey, acct *models.SolanaDepositAccount, mint solana.PublicKey, amount uint64) solana.PublicKey {
	t.Helper()
	owner := solana.MustPublicKey(acct.OwnerAddress)
	create, ata, err := solana.CreateAssociatedTokenAccountIdempotent(payer.PublicKey(), owner, mint, solana.TokenProgram)
	must(t, err)
	send(t, ctx, c, payer, []solana.Instruction{create, solana.TokenTransferChecked(solana.TokenProgram, source, mint, ata, payer.PublicKey(), amount, 6)})
	return ata
}

func waitFor(t *testing.T, ctx context.Context, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Second):
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

var _ = errors.New
