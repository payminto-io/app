package service

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Fixture addresses shared with internal/blockchain/solana/testdata.
const (
	fxOwner   = "HAgk14JpMQLgt6rVgv7cBQFJWFto5Dqxi472uT3DKpqk"
	fxPayer   = "9h1cLBiraaUqM1CdJTaVaew1oQtgQUW24FZ8YdnLLgJY"
	fxUSDC    = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	fxUSDT    = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYb"
	fxUSDCATA = "5N3f1tj9v1vc5TUZ8S7mCAnVmjVKrfnzXWhxLaxyZAgt"
	fxUSDTATA = "4fz24twEFEWmsAKeeD7hgGZtVBHiGjRLEFuciSgRcgdw"
)

var solanaFixtureDir = filepath.Join("..", "blockchain", "solana", "testdata", "tx")

type solanaFixture struct {
	t        *testing.T
	db       *gorm.DB
	rpc      *solana.ScriptedCaller
	chain    *models.Blockchain
	usdc     *models.BlockchainCurrency
	usdt     *models.BlockchainCurrency
	sol      *models.BlockchainCurrency
	member   *models.Member
	platform *models.ExternalPlatform
	ledger   *LedgerService
	journal  *ledger.Service
	deposits repository.DepositRepository
	accounts repository.SolanaDepositAccountRepository
	missed   repository.MissedDepositRepository
	svc      *SolanaDepositService
	now      time.Time
}

func newSolanaFixture(t *testing.T) *solanaFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Member{}, &models.ExternalPlatform{}, &models.PaymentRequest{}, &models.Deposit{}, &models.DepositAddress{},
		&models.SolanaDepositAccount{}, &models.BlockchainFamily{}, &models.Blockchain{}, &models.Currency{}, &models.BlockchainCurrency{},
		&models.MissedDeposit{}, &models.Account{}, &models.Asset{}, &models.Liability{}, &models.Revenue{}, &models.Expense{},
		&models.Sweep{}, &models.SweepTransaction{}, &models.Wallet{}, &models.AddressPool{}, &models.WalletXpub{},
	); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Migrate(db); err != nil {
		t.Fatal(err)
	}
	f := &solanaFixture{t: t, db: db, rpc: solana.NewScriptedCaller(), now: time.Now()}

	family := models.BlockchainFamily{Name: "Solana", Code: "sol", Family: "SOL_Family"}
	must(t, db.Create(&family).Error)
	f.chain = &models.Blockchain{Code: solana.ChainCode, Name: "Solana Devnet", Family: "SOL_Family", BlockchainFamilyID: family.ID, MinConfirmations: 32, Status: "active"}
	must(t, db.Create(f.chain).Error)
	for _, c := range []models.Currency{{Code: "SOL", Name: "Solana", Type: "native", WalletPrecision: 9}, {Code: "USDC", Name: "USD Coin", Type: "token", WalletPrecision: 6}, {Code: "USDT", Name: "Tether", Type: "token", WalletPrecision: 6}} {
		c := c
		must(t, db.Create(&c).Error)
		row := models.BlockchainCurrency{BlockchainID: f.chain.ID, CurrencyID: c.ID, CurrencyCode: c.Code, BlockchainCode: solana.ChainCode, WalletPrecision: c.WalletPrecision, DepositEnabled: true, Visible: true}
		switch c.Code {
		case "SOL":
			row.Standard = "native"
		case "USDC":
			row.Standard, row.Address = "SPL", fxUSDC
		case "USDT":
			row.Standard, row.Address = "SPL", fxUSDT
		}
		must(t, db.Create(&row).Error)
		switch c.Code {
		case "SOL":
			f.sol = &row
		case "USDC":
			f.usdc = &row
		case "USDT":
			f.usdt = &row
		}
	}
	f.member = &models.Member{Name: "merchant", MemberType: "merchant", State: "active"}
	must(t, db.Create(f.member).Error)
	f.platform = &models.ExternalPlatform{Name: "shop"}
	must(t, db.Create(f.platform).Error)

	f.deposits = repository.NewDepositRepository(db)
	f.accounts = repository.NewSolanaDepositAccountRepository(db)
	f.missed = repository.NewMissedDepositRepository(db)
	depositAddrRepo := repository.NewDepositAddressRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	bcRepo := repository.NewBlockchainCurrencyRepository(db)
	depositSvc := NewDepositService(f.deposits, depositAddrRepo, paymentRepo, bcRepo, repository.NewBlockchainRepository(db))
	f.journal = ledger.New(db)
	f.ledger = NewLedgerService(repository.NewAccountRepository(db), WithJournal(f.journal, blockchainCurrencyAssetResolver()))
	f.svc = NewSolanaDepositService(db, solana.NewClient(f.rpc), f.chain, f.accounts, f.deposits, depositSvc, f.missed, bcRepo, f.ledger, nil, SolanaDepositConfig{DropGrace: time.Minute})
	f.svc.now = func() time.Time { return f.now }
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// newPayment creates an open payment whose deposit address is the owner's ATA for bc.
func (f *solanaFixture) newPayment(amountUSD string, bc *models.BlockchainCurrency, owner, tokenAccount string) (*models.PaymentRequest, *models.SolanaDepositAccount) {
	f.t.Helper()
	pr := &models.PaymentRequest{ReferenceID: "ref-" + tokenAccount[:8], AmountInUSD: decimal.RequireFromString(amountUSD), State: models.PaymentStateOpen, MemberID: f.member.ID, ExternalPlatformID: f.platform.ID}
	must(f.t, f.db.Create(pr).Error)
	da := &models.DepositAddress{Address: tokenAccount, BlockchainCurrencyID: bc.ID, MemberID: f.member.ID, PaymentRequestID: &pr.ID}
	must(f.t, f.db.Create(da).Error)
	acct := &models.SolanaDepositAccount{DepositAddressID: da.ID, PaymentRequestID: &pr.ID, BlockchainCurrencyID: bc.ID, OwnerAddress: owner, TokenAccount: tokenAccount, Mint: bc.Address, TokenProgram: solana.TokenProgram.String(), Decimals: 6, Status: models.SolanaDepositAccountWatching}
	must(f.t, f.accounts.Create(acct))
	return pr, acct
}

// script wires getSignaturesForAddress and getTransaction from fixture files keyed by address.
func (f *solanaFixture) script(byAddress map[string][]string) {
	f.t.Helper()
	txs := map[string]any{}
	slots := map[string]uint64{}
	for _, names := range byAddress {
		for _, name := range names {
			tx, err := solana.LoadFixtureTransaction(filepath.Join(solanaFixtureDir, name))
			must(f.t, err)
			raw, err := solana.LoadFixtureJSON(filepath.Join(solanaFixtureDir, name))
			must(f.t, err)
			txs[tx.Signature()] = raw
			slots[tx.Signature()] = tx.Slot
		}
	}
	sigOf := func(name string) string {
		tx, _ := solana.LoadFixtureTransaction(filepath.Join(solanaFixtureDir, name))
		return tx.Signature()
	}
	f.rpc.On("getSignaturesForAddress", func(p []any) (any, error) {
		names := byAddress[solana.FirstParamString(p)]
		opts, _ := p[1].(map[string]any)
		until, _ := opts["until"].(string)
		var out []map[string]any
		// Fixtures are listed oldest first; the RPC answers newest first and stops at the cursor.
		for i := len(names) - 1; i >= 0; i-- {
			sig := sigOf(names[i])
			if sig == until {
				break
			}
			out = append(out, map[string]any{"signature": sig, "slot": slots[sig], "err": nil, "confirmationStatus": "confirmed"})
		}
		return out, nil
	})
	f.rpc.On("getTransaction", func(p []any) (any, error) {
		return txs[solana.FirstParamString(p)], nil
	})
	// Every tick reports a different balance so the token account is always polled; tests of the
	// cadence override this with scriptAccounts.
	f.rpc.On("getMultipleAccounts", func(p []any) (any, error) {
		addrs, _ := p[0].([]string)
		out := make([]any, len(addrs))
		for i := range addrs {
			out[i] = map[string]any{"lamports": 1, "owner": solana.TokenProgram.String(), "data": map[string]any{"parsed": map[string]any{"type": "account", "info": map[string]any{"tokenAmount": map[string]any{"amount": strconv.Itoa(f.rpc.Count("getMultipleAccounts")), "decimals": 6}}}}}
		}
		return solana.ContextValue(1, out), nil
	})
}

func (f *solanaFixture) statuses(status map[string]any) {
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) {
		n := 0
		switch sigs := p[0].(type) {
		case []string:
			n = len(sigs)
		case []any:
			n = len(sigs)
		}
		out := make([]any, n)
		for i := range out {
			out[i] = status
		}
		return solana.ContextValue(1, out), nil
	})
}

func (f *solanaFixture) depositsFor(pr *models.PaymentRequest) []models.Deposit {
	var out []models.Deposit
	must(f.t, f.db.Where("payment_request_id = ?", pr.ID).Order("id").Find(&out).Error)
	return out
}

func (f *solanaFixture) paymentState(pr *models.PaymentRequest) string {
	var p models.PaymentRequest
	must(f.t, f.db.First(&p, pr.ID).Error)
	return p.State
}

type ledgerLine struct {
	OwnerID string
	Asset   string
	Kind    string
	Amount  decimal.Decimal
}

func (f *solanaFixture) journalLines(referenceType, referenceID string) []ledgerLine {
	var rows []ledgerLine
	must(f.t, f.db.Table("ledger_lines").
		Select("ledger_accounts.owner_id, ledger_accounts.asset, ledger_accounts.kind, ledger_lines.amount").
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Joins("JOIN ledger_journals ON ledger_journals.id = ledger_lines.journal_id").
		Where("ledger_journals.reference_type = ? AND ledger_journals.reference_id = ?", referenceType, referenceID).
		Order("ledger_lines.id").Scan(&rows).Error)
	return rows
}

func TestSolanaDeposit_USDCDetectedConfirmedThenFinalizedWithJournal(t *testing.T) {
	f := newSolanaFixture(t)
	pr, acct := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	ctx := context.Background()

	n, err := f.svc.PollOnce(ctx)
	must(t, err)
	if n != 1 {
		t.Fatalf("recorded = %d, want 1", n)
	}
	deps := f.depositsFor(pr)
	if len(deps) != 1 || deps[0].Status != models.DepositStatusPending || !deps[0].Amount.Equal(decimal.RequireFromString("25")) || deps[0].RequiredConfirmations != 32 {
		t.Fatalf("deposit = %+v", deps)
	}
	if deps[0].ToAddress != fxUSDCATA || deps[0].FromAddress != fxPayer || deps[0].BlockNumber != 250000123 {
		t.Fatalf("deposit routing = %+v", deps[0])
	}
	// A second poll with the cursor advanced records nothing new.
	if n, _ := f.svc.PollOnce(ctx); n != 0 {
		t.Fatalf("second poll recorded %d", n)
	}
	got, _ := f.accounts.GetByDepositAddressID(acct.DepositAddressID)
	if got.TokenAccountCursor != deps[0].TxID || got.LastSeenSlot != 250000123 {
		t.Fatalf("cursor not persisted: %+v", got)
	}

	// confirmed commitment: seen, not credited.
	f.statuses(map[string]any{"slot": 250000123, "confirmations": 9, "err": nil, "confirmationStatus": "confirmed"})
	done, err := f.svc.ConfirmOnce(ctx)
	must(t, err)
	deps = f.depositsFor(pr)
	if done != 0 || deps[0].Status != models.DepositStatusConfirming || deps[0].Confirmations != 9 {
		t.Fatalf("after confirmed: done=%d deposit=%+v", done, deps[0])
	}
	if f.paymentState(pr) != models.PaymentStateOpen {
		t.Fatal("payment must stay open until finalized")
	}
	if lines := f.journalLines("deposit", "1"); len(lines) != 0 {
		t.Fatalf("journal posted before finalization: %+v", lines)
	}

	// finalized: credited, payment filled, journal in USDC.SOLANA.
	f.statuses(map[string]any{"slot": 250000123, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	done, err = f.svc.ConfirmOnce(ctx)
	must(t, err)
	deps = f.depositsFor(pr)
	if done != 1 || deps[0].Status != models.DepositStatusConfirmed || deps[0].Confirmations != 32 {
		t.Fatalf("after finalized: done=%d deposit=%+v", done, deps[0])
	}
	if f.paymentState(pr) != models.PaymentStateFilled {
		t.Fatalf("payment state = %s, want FILLED", f.paymentState(pr))
	}
	lines := f.journalLines("deposit", decimal.NewFromInt(int64(deps[0].ID)).String())
	if len(lines) != 2 {
		t.Fatalf("journal lines = %+v", lines)
	}
	for _, l := range lines {
		if l.Asset != "USDC.SOLANA" {
			t.Fatalf("line asset = %s, want USDC.SOLANA", l.Asset)
		}
	}
	if lines[0].OwnerID != "crypto_assets" || !lines[0].Amount.Equal(decimal.RequireFromString("25")) || lines[1].OwnerID != "merchant_balance" || !lines[1].Amount.Equal(decimal.RequireFromString("-25")) {
		t.Fatalf("journal lines = %+v", lines)
	}
	// Idempotent: a repeated finalized status does nothing.
	if done, _ := f.svc.ConfirmOnce(ctx); done != 0 {
		t.Fatal("finalized twice")
	}
	if lines := f.journalLines("deposit", decimal.NewFromInt(int64(deps[0].ID)).String()); len(lines) != 2 {
		t.Fatal("journal duplicated")
	}
}

func TestSolanaDeposit_USDTIsItsOwnAsset(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("10", f.usdt, fxOwner, fxUSDTATA)
	f.script(map[string][]string{fxUSDTATA: {"usdt_transfer.json"}})
	ctx := context.Background()
	if n, err := f.svc.PollOnce(ctx); err != nil || n != 1 {
		t.Fatalf("poll: %d %v", n, err)
	}
	f.statuses(map[string]any{"slot": 250000456, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	if done, err := f.svc.ConfirmOnce(ctx); err != nil || done != 1 {
		t.Fatalf("confirm: %d %v", done, err)
	}
	deps := f.depositsFor(pr)
	if deps[0].BlockchainCurrencyID != f.usdt.ID || !deps[0].Amount.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("deposit = %+v", deps[0])
	}
	lines := f.journalLines("deposit", decimal.NewFromInt(int64(deps[0].ID)).String())
	if len(lines) != 2 || lines[0].Asset != "USDT.SOLANA" || lines[1].Asset != "USDT.SOLANA" {
		t.Fatalf("journal lines = %+v", lines)
	}
	if f.paymentState(pr) != models.PaymentStateFilled {
		t.Fatalf("state = %s", f.paymentState(pr))
	}
}

func TestSolanaDeposit_USDTToUSDCPaymentIsAnomalyNeverCredited(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	// The payer sent USDT to the owner address; the owner's signatures carry the transfer.
	f.script(map[string][]string{fxOwner: {"wrong_mint_usdt_to_usdc_payment.json"}})
	ctx := context.Background()
	if n, err := f.svc.PollOnce(ctx); err != nil || n != 0 {
		t.Fatalf("poll: %d %v", n, err)
	}
	if deps := f.depositsFor(pr); len(deps) != 0 {
		t.Fatalf("wrong-mint transfer became a deposit: %+v", deps)
	}
	missed, err := f.missed.ListUnresolved()
	must(t, err)
	if len(missed) != 1 {
		t.Fatalf("missed deposits = %+v", missed)
	}
	m := missed[0]
	if m.Reason[:len("wrong_mint")] != "wrong_mint" || m.ToAddress != fxUSDTATA || m.BlockchainCurrencyID == nil || *m.BlockchainCurrencyID != f.usdt.ID || !m.Amount.Equal(decimal.RequireFromString("25")) {
		t.Fatalf("anomaly = %+v", m)
	}
	// Re-polling the same signature does not duplicate the anomaly.
	f.svc.now = func() time.Time { return f.now.Add(time.Minute) }
	acct, _ := f.accounts.GetByTokenAccount(fxUSDCATA)
	must(t, f.accounts.Update(acct.ID, map[string]any{"owner_cursor": "", "owner_poll_after": nil}))
	if _, err := f.svc.PollOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if missed, _ := f.missed.ListUnresolved(); len(missed) != 1 {
		t.Fatalf("anomaly duplicated: %d rows", len(missed))
	}
	if f.paymentState(pr) != models.PaymentStateOpen {
		t.Fatal("payment must stay open")
	}
}

func TestSolanaDeposit_PaymentToOwnerAddressIsCredited(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("40", f.usdc, fxOwner, fxUSDCATA)
	// The wallet created the ATA in the same transaction; both addresses see the signature.
	f.script(map[string][]string{fxOwner: {"owner_payment_creates_ata.json"}, fxUSDCATA: {"owner_payment_creates_ata.json"}})
	if n, err := f.svc.PollOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("poll: %d %v", n, err)
	}
	deps := f.depositsFor(pr)
	if len(deps) != 1 || !deps[0].Amount.Equal(decimal.RequireFromString("40")) || deps[0].ToAddress != fxUSDCATA {
		t.Fatalf("deposits = %+v", deps)
	}
	if missed, _ := f.missed.ListUnresolved(); len(missed) != 0 {
		t.Fatalf("unexpected anomalies: %+v", missed)
	}
}

func TestSolanaDeposit_NativeSOLToOwnerIsAnomaly(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxOwner: {"native_sol_to_owner.json"}})
	if n, err := f.svc.PollOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("poll: %d %v", n, err)
	}
	if len(f.depositsFor(pr)) != 0 {
		t.Fatal("SOL credited as USDC")
	}
	missed, _ := f.missed.ListUnresolved()
	if len(missed) != 1 || missed[0].Reason[:len("native_sol_to_owner")] != "native_sol_to_owner" || missed[0].BlockchainCurrencyID != nil {
		t.Fatalf("missed = %+v", missed)
	}
}

func TestSolanaDeposit_DroppedBeforeFinalizationIsReversed(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	ctx := context.Background()
	f.mustPoll(ctx)
	f.statuses(map[string]any{"slot": 250000123, "confirmations": 3, "err": nil, "confirmationStatus": "confirmed"})
	f.mustConfirm(ctx)
	if f.depositsFor(pr)[0].Status != models.DepositStatusConfirming {
		t.Fatal("expected confirming")
	}
	// The cluster forgets the signature (fork dropped) and the node has no transaction either.
	f.rpc.On("getSignatureStatuses", func(p []any) (any, error) { return solana.ContextValue(1, []any{nil}), nil })
	f.rpc.Result("getTransaction", nil)
	f.mustConfirm(ctx)
	if f.depositsFor(pr)[0].Status != models.DepositStatusConfirming {
		t.Fatal("must wait out the drop grace before reversing")
	}
	f.svc.now = func() time.Time { return f.now.Add(time.Hour) }
	f.db.Model(&models.Deposit{}).Where("payment_request_id = ?", pr.ID).Update("created_at", f.now.Add(-time.Hour))
	f.finalizedSlot(250000200)
	f.mustConfirm(ctx)
	if got := f.depositsFor(pr)[0]; got.Status != models.DepositStatusFailed {
		t.Fatalf("deposit = %+v, want failed", got)
	}
	if f.paymentState(pr) != models.PaymentStateOpen {
		t.Fatal("payment must still be open")
	}
	if lines := f.journalLines("deposit", "1"); len(lines) != 0 {
		t.Fatalf("journal for a reversed deposit: %+v", lines)
	}
}

func TestSolanaDeposit_PartialThenOverPayment(t *testing.T) {
	f := newSolanaFixture(t)
	pr, _ := f.newPayment("30", f.usdc, fxOwner, fxUSDCATA)
	ctx := context.Background()
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	f.mustPoll(ctx)
	f.statuses(map[string]any{"slot": 1, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	f.mustConfirm(ctx)
	if f.paymentState(pr) != models.PaymentStatePartiallyFilled {
		t.Fatalf("after 25 of 30: %s", f.paymentState(pr))
	}
	// A second transfer of 12.5 lands; total 37.5 over-fills.
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json", "cpi_inner_transfer.json"}})
	n, err := f.svc.PollOnce(ctx)
	must(t, err)
	if n != 1 {
		t.Fatalf("second poll recorded %d", n)
	}
	f.mustConfirm(ctx)
	deps := f.depositsFor(pr)
	if len(deps) != 2 || deps[1].Status != models.DepositStatusConfirmed {
		t.Fatalf("deposits = %+v", deps)
	}
	if f.paymentState(pr) != models.PaymentStateOverFilled {
		t.Fatalf("after 37.5 of 30: %s", f.paymentState(pr))
	}
	// Each deposit has its own journal in USDC.SOLANA.
	for _, d := range deps {
		lines := f.journalLines("deposit", decimal.NewFromInt(int64(d.ID)).String())
		if len(lines) != 2 || !lines[0].Amount.Equal(d.Amount) {
			t.Fatalf("deposit %d journal = %+v", d.ID, lines)
		}
	}
}

func (f *solanaFixture) mustPoll(ctx context.Context) int {
	f.t.Helper()
	n, err := f.svc.PollOnce(ctx)
	must(f.t, err)
	return n
}

func (f *solanaFixture) mustConfirm(ctx context.Context) int {
	f.t.Helper()
	n, err := f.svc.ConfirmOnce(ctx)
	must(f.t, err)
	return n
}
