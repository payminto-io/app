package service

import (
	"context"
	"testing"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// callerFixture seeds currencies and blockchain_currencies with ids that deliberately differ,
// so resolving a blockchain_currencies id through the currencies table would pick the wrong asset.
type callerFixture struct {
	db        *gorm.DB
	journal   *ledger.Service
	ledgerSvc *LedgerService
	usdcBase  uint // blockchain_currencies id of USDC on BASE (currency id 3)
	ethBase   uint // blockchain_currencies id of ETH on BASE (currency id 2)
	usdcPoly  uint // USDC on POLYGON, a chain with no native row
}

func newCallerFixture(t *testing.T) *callerFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Currency{}, &models.BlockchainFamily{}, &models.Blockchain{}, &models.BlockchainCurrency{}, &models.Member{}, &models.ExternalPlatform{},
		&models.Account{}, &models.Asset{}, &models.Liability{}, &models.Revenue{}, &models.Expense{},
		&models.Sweep{}, &models.SweepTransaction{}, &models.Withdrawal{}, &models.Withdraw{},
		&models.InternalBlockchainTransaction{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := ledger.Migrate(db); err != nil {
		t.Fatalf("ledger.Migrate: %v", err)
	}
	seedCurrencyCatalog(t, db)

	f := &callerFixture{db: db, journal: ledger.New(db)}
	bcRepo := repository.NewBlockchainCurrencyRepository(db)
	f.ledgerSvc = NewLedgerService(repository.NewAccountRepository(db), WithJournal(f.journal, blockchainCurrencyAssetResolver()))
	for _, bc := range []struct {
		code, chain string
		dst         *uint
	}{{"USDC", "BASE", &f.usdcBase}, {"ETH", "BASE", &f.ethBase}, {"USDC", "POLYGON", &f.usdcPoly}} {
		row, err := bcRepo.GetByBlockchainCodeAndCurrencyCode(bc.chain, bc.code)
		if err != nil {
			t.Fatalf("lookup %s on %s: %v", bc.code, bc.chain, err)
		}
		*bc.dst = row.ID
	}
	return f
}

func seedCurrencyCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()
	currencies := []models.Currency{{Code: "BTC", Name: "Bitcoin"}, {Code: "ETH", Name: "Ether"}, {Code: "USDC", Name: "USD Coin"}}
	for i := range currencies {
		if err := db.Create(&currencies[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	family := models.BlockchainFamily{Name: "EVM", Code: "evm"}
	if err := db.Create(&family).Error; err != nil {
		t.Fatal(err)
	}
	chains := []models.Blockchain{
		{Code: "BASE", Name: "Base", BlockchainFamilyID: family.ID},
		{Code: "ETH", Name: "Ethereum", BlockchainFamilyID: family.ID},
		{Code: "POLYGON", Name: "Polygon", BlockchainFamilyID: family.ID},
	}
	for i := range chains {
		if err := db.Create(&chains[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	eth, usdc := currencies[1], currencies[2]
	base, ethChain, poly := chains[0], chains[1], chains[2]
	rows := []models.BlockchainCurrency{
		{CurrencyID: eth.ID, BlockchainID: base.ID, CurrencyCode: "ETH", BlockchainCode: "BASE", Standard: "native"},
		{CurrencyID: usdc.ID, BlockchainID: base.ID, CurrencyCode: "USDC", BlockchainCode: "BASE", Standard: "ERC20"},
		{CurrencyID: usdc.ID, BlockchainID: ethChain.ID, CurrencyCode: "USDC", BlockchainCode: "ETH", Standard: "ERC20"},
		{CurrencyID: eth.ID, BlockchainID: ethChain.ID, CurrencyCode: "ETH", BlockchainCode: "ETH", Standard: "native"},
		{CurrencyID: usdc.ID, BlockchainID: poly.ID, CurrencyCode: "USDC", BlockchainCode: "POLYGON", Standard: "ERC20"},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
}

type postedLine struct {
	OwnerID string
	Kind    ledger.AccountKind
	Asset   string
	Amount  decimal.Decimal
}

func journalLines(t *testing.T, db *gorm.DB, key string) (ledger.JournalRow, []postedLine) {
	t.Helper()
	var row ledger.JournalRow
	if err := db.Where("idempotency_key = ?", key).First(&row).Error; err != nil {
		t.Fatalf("journal %q: %v", key, err)
	}
	var lines []postedLine
	err := db.Table("ledger_lines").
		Select("ledger_accounts.owner_id, ledger_accounts.kind, ledger_lines.asset, ledger_lines.amount").
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Where("ledger_lines.journal_id = ?", row.ID).
		Order("ledger_lines.id").
		Scan(&lines).Error
	if err != nil {
		t.Fatal(err)
	}
	return row, lines
}

func expectLines(t *testing.T, got []postedLine, want []postedLine) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("lines = %+v, want %+v", got, want)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.OwnerID != w.OwnerID || g.Kind != w.Kind || g.Asset != w.Asset || !g.Amount.Equal(w.Amount) {
			t.Fatalf("line %d = %+v, want %+v", i, g, w)
		}
	}
}

func count(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var n int64
	if err := db.Model(model).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBlockchainCurrencyAssetResolver_ResolvesFromTheTableTheIdRefersTo(t *testing.T) {
	f := newCallerFixture(t)
	resolveWith := blockchainCurrencyAssetResolver()
	resolve := func(id uint) (Assets, error) { return resolveWith(f.db, id) }

	got, err := resolve(f.usdcBase)
	if err != nil || got.Asset != "USDC.BASE" || got.Native != "ETH.BASE" {
		t.Fatalf("resolve(usdc on base) = %+v, %v", got, err)
	}
	// currencies row with the same id is ETH; the old resolver would have said so.
	var byCurrencyID models.Currency
	if err := f.db.First(&byCurrencyID, f.usdcBase).Error; err == nil && byCurrencyID.Code == "USDC" {
		t.Fatal("fixture ids collide; the test cannot tell the two lookups apart")
	}

	got, err = resolve(f.ethBase)
	if err != nil || got.Asset != "ETH.BASE" || got.Native != "ETH.BASE" {
		t.Fatalf("resolve(eth on base) = %+v, %v", got, err)
	}
	if _, err := resolve(f.usdcPoly); err == nil {
		t.Fatal("a chain without a native row must fail, not guess the gas asset")
	}
	if _, err := resolve(999); err == nil {
		t.Fatal("unknown id must fail")
	}
}

func TestSweepMarkCompleted_PostsJournalInTheSweepTransaction(t *testing.T) {
	f := newCallerFixture(t)
	ctx := context.Background()
	svc := NewSweepService(f.db, repository.NewSweepRepository(f.db), repository.NewSweepTransactionRepository(f.db), repository.NewBlockchainRepository(f.db), f.ledgerSvc)
	sweep, err := svc.CreateSweep(1)
	if err != nil {
		t.Fatal(err)
	}
	amount, gas := decimal.RequireFromString("1000"), decimal.RequireFromString("0.002")
	if err := svc.MarkCompleted(ctx, sweep.ID, amount, gas, f.usdcBase); err != nil {
		t.Fatal(err)
	}
	row, lines := journalLines(t, f.db, "payminto:sweep:"+idStr(sweep.ID))
	if row.Kind != ledger.KindTransfer {
		t.Fatalf("kind = %s", row.Kind)
	}
	expectLines(t, lines, []postedLine{
		{"cold_wallet_assets", ledger.KindAsset, "USDC.BASE", amount},
		{"crypto_assets", ledger.KindAsset, "USDC.BASE", amount.Neg()},
		{"sweep_gas", ledger.KindExpense, "ETH.BASE", gas},
		{"crypto_assets", ledger.KindAsset, "ETH.BASE", gas.Neg()},
	})

	// Resolution failure rolls the status change back: the sweep is not completed and nothing is written.
	second, _ := svc.CreateSweep(1)
	if err := svc.MarkCompleted(ctx, second.ID, amount, gas, 999); err == nil {
		t.Fatal("expected an error for an unknown blockchain currency")
	}
	got, _ := svc.GetByID(second.ID)
	if got.Status == SweepStatusCompleted {
		t.Fatal("sweep marked completed although its journal was never posted")
	}
	if n := count(t, f.db, &ledger.JournalRow{}); n != 1 {
		t.Fatalf("journals = %d, want 1", n)
	}
	if n := count(t, f.db, &models.Asset{}); n != 2 {
		t.Fatalf("legacy asset rows = %d, want 2 (one sweep)", n)
	}
}

func TestWithdrawalExecute_PostsJournalAndFailsClosed(t *testing.T) {
	f := newCallerFixture(t)
	ctx := context.Background()
	svc := NewWithdrawalProcessingService(repository.NewWithdrawalRepository(f.db), repository.NewWithdrawRepository(f.db), f.ledgerSvc, nil, nil, nil)

	w := createPendingWithdrawal(t, f.db)
	f.db.Model(w).Update("blockchain_currency_id", f.usdcBase)
	w.BlockchainCurrencyID = f.usdcBase
	if err := svc.Execute(ctx, w); err != nil {
		t.Fatal(err)
	}
	row, lines := journalLines(t, f.db, "payminto:withdrawal:"+idStr(w.ID))
	if row.Kind != ledger.KindSettlement {
		t.Fatalf("kind = %s", row.Kind)
	}
	expectLines(t, lines, []postedLine{
		{"merchant_balance", ledger.KindLiability, "USDC.BASE", w.Amount},
		{"crypto_assets", ledger.KindAsset, "USDC.BASE", w.Amount.Neg()},
	})
	after, _ := repository.NewWithdrawalRepository(f.db).GetByID(w.ID)
	if after.State != models.WithdrawalStateProcessed {
		t.Fatalf("state = %s", after.State)
	}

	bad := createPendingWithdrawal(t, f.db)
	f.db.Model(bad).Update("blockchain_currency_id", 999)
	bad.BlockchainCurrencyID = 999
	if err := svc.Execute(ctx, bad); err == nil {
		t.Fatal("expected an error when the ledger cannot post")
	}
	after, _ = repository.NewWithdrawalRepository(f.db).GetByID(bad.ID)
	if after.State != models.WithdrawalStateInitiated {
		t.Fatalf("state after failed ledger post = %s, want initiated (sent and processed must roll back)", after.State)
	}
	if n := count(t, f.db, &ledger.JournalRow{}); n != 1 {
		t.Fatalf("journals = %d, want 1", n)
	}
}

func TestRecordGasFunding_JoinsTheCallerTransaction(t *testing.T) {
	f := newCallerFixture(t)
	svc := NewInternalBlockchainTransactionService(f.db, repository.NewInternalBlockchainTransactionRepository(f.db), f.ledgerSvc)
	gas := decimal.RequireFromString("0.0004")
	ibt, err := svc.RecordGasFunding(f.ethBase, "0xfrom", "0xto", decimal.RequireFromString("0.1"), gas)
	if err != nil {
		t.Fatal(err)
	}
	row, lines := journalLines(t, f.db, "payminto:gas_fee:"+idStr(ibt.ID))
	if row.Kind != ledger.KindFee {
		t.Fatalf("kind = %s", row.Kind)
	}
	expectLines(t, lines, []postedLine{
		{"gas_fee", ledger.KindExpense, "ETH.BASE", gas},
		{"crypto_assets", ledger.KindAsset, "ETH.BASE", gas.Neg()},
	})

	if _, err := svc.RecordGasFunding(999, "0xfrom", "0xto", decimal.RequireFromString("0.1"), gas); err == nil {
		t.Fatal("expected an error for an unknown blockchain currency")
	}
	if n := count(t, f.db, &models.InternalBlockchainTransaction{}); n != 1 {
		t.Fatalf("ibt rows = %d, want 1: the IBT insert must roll back with the ledger", n)
	}
}

func TestLedgerService_RefusesARepositoryItCannotBindToTheTransaction(t *testing.T) {
	f := newCallerFixture(t)
	wrapped := NewLedgerService(unbindableRepo{repository.NewAccountRepository(f.db)}, WithJournal(f.journal, blockchainCurrencyAssetResolver()))
	if err := wrapped.RecordPaymentDeposit(1, f.usdcBase, decimal.NewFromInt(1)); err == nil {
		t.Fatal("a repository outside the journal transaction must be refused, not run unbound")
	}
	if n := count(t, f.db, &ledger.JournalRow{}); n != 0 {
		t.Fatalf("journals = %d, want 0", n)
	}
}

type unbindableRepo struct{ repository.AccountRepository }
