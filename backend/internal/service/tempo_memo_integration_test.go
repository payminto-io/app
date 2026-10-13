//go:build integration

package service

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func tempoPostgresFixture(t *testing.T) (*TempoMemoService, *gorm.DB, *models.PaymentRequest) {
	t.Helper()
	// NewTestDB drops its isolated schema at cleanup. Never accept an application target here.
	if dsn := os.Getenv("PAYMINTO_INTEGRATION_DATABASE_URL"); dsn != "" {
		u, err := url.Parse(dsn)
		if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !strings.HasSuffix(strings.TrimPrefix(u.Path, "/"), "_test") {
			t.Fatal("Tempo integration override must identify a disposable *_test PostgreSQL database")
		}
	}
	db, cleanup := database.NewTestDB(t)
	t.Cleanup(cleanup)
	family := &models.BlockchainFamily{Name: "Tempo", Code: "tempo", Family: "TEMPO_Family"}
	member := &models.Member{Name: "Tempo merchant", MemberType: "merchant", State: "active"}
	platform := &models.ExternalPlatform{Name: "Tempo test shop"}
	cur := &models.Currency{Code: "ALPHAUSD", Name: "Alpha USD", Type: "token", DepositEnabled: true, WalletPrecision: 6}
	for _, row := range []any{family, member, platform, cur} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	chainID := int64(tempo.ModeratoChainID)
	chain := &models.Blockchain{Code: tempo.ChainCode, Name: "Tempo test", Status: "active", MinConfirmations: 1, ChainID: &chainID, BlockchainFamilyID: family.ID}
	if err := db.Create(chain).Error; err != nil {
		t.Fatal(err)
	}
	bc := models.BlockchainCurrency{BlockchainID: chain.ID, CurrencyID: cur.ID, Address: tempo.AlphaUSD.Hex(), Standard: "TIP20", BlockchainCode: tempo.ChainCode, CurrencyCode: "ALPHAUSD", DepositEnabled: true, WalletPrecision: 6}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatal(err)
	}
	p := &models.PaymentRequest{ReferenceID: "tempo-postgres-invoice", MemberID: member.ID, ExternalPlatformID: platform.ID, State: models.PaymentStateOpen, AmountInUSD: decimal.NewFromInt(2)}
	if err := db.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	addressSvc := NewDepositAddressService(repository.NewDepositAddressRepository(db), nil, repository.NewBlockchainCurrencyRepository(db), nil, nil).WithTempoMerchant(tempoMerchant.Hex(), member.ID)
	if _, err := addressSvc.AssignForPayment(p, tempo.ChainCode, "ALPHAUSD"); err != nil {
		t.Fatal(err)
	}
	depSvc := NewDepositService(repository.NewDepositRepository(db), repository.NewDepositAddressRepository(db), repository.NewPaymentRepository(db), repository.NewBlockchainCurrencyRepository(db), repository.NewBlockchainRepository(db))
	ledgerSvc := NewLedgerService(repository.NewAccountRepository(db), WithJournal(ledger.New(db), blockchainCurrencyAssetResolver()))
	svc := NewTempoMemoService(db, &fakeTempoReader{head: 120}, chain, TempoTokensFromCurrencies([]models.BlockchainCurrency{bc}), depSvc, ledgerSvc, nil, TempoMemoConfig{Merchant: tempoMerchant, MemberID: member.ID, StartBlock: 100})
	return svc, db, p
}

func TestTempoPostgresConcurrentReplayBooksOneJournal(t *testing.T) {
	svc, db, p := tempoPostgresFixture(t)
	transfer := memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xf2", 0, 110)
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			won, err := svc.handle(context.Background(), transfer)
			results <- won
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	winners := 0
	for won := range results {
		if won {
			winners++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("credited %d times", winners)
	}
	assertTempoPostgresMoney(t, db, p.ID, models.PaymentStateFilled, 1, "2")
}

func TestTempoPostgresPaymentFailureRollsBackJournalThenRetries(t *testing.T) {
	svc, db, p := tempoPostgresFixture(t)
	reader := svc.client.(*fakeTempoReader)
	reader.transfers = []tempo.MemoTransfer{memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xf3", 0, 110)}
	if err := db.Callback().Update().Before("gorm:update").Register("test:tempo-payment-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "payment_requests" {
			_ = tx.AddError(errors.New("temporary payment failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertTempoPostgresMoney(t, db, p.ID, models.PaymentStateOpen, 0, "0")
	var d models.Deposit
	if err := db.First(&d).Error; err != nil || d.Status != models.DepositStatusPending {
		t.Fatalf("not retryable: %s, %v", d.Status, err)
	}
	if err := db.Callback().Update().Remove("test:tempo-payment-failure"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertTempoPostgresMoney(t, db, p.ID, models.PaymentStateFilled, 1, "2")
}

func TestTempoPostgresPartialFillAndAdditionalLogOverpayment(t *testing.T) {
	svc, db, p := tempoPostgresFixture(t)
	for i, amount := range []int64{750_000, 1_250_000, 250_000} {
		tr := memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, amount, "0xf4", uint(i), 110)
		if won, err := svc.handle(context.Background(), tr); err != nil || !won {
			t.Fatalf("credit=%t, %v", won, err)
		}
		if i == 0 {
			assertTempoPostgresMoney(t, db, p.ID, models.PaymentStatePartiallyFilled, 1, "0.75")
		}
	}
	assertTempoPostgresMoney(t, db, p.ID, models.PaymentStateOverFilled, 3, "2.25")
}

func assertTempoPostgresMoney(t *testing.T, db *gorm.DB, paymentID uint, state string, journals int64, amount string) {
	t.Helper()
	var p models.PaymentRequest
	if err := db.First(&p, paymentID).Error; err != nil || p.State != state {
		t.Fatalf("state=%s, want %s; err=%v", p.State, state, err)
	}
	var count int64
	if err := db.Table("ledger_journals").Count(&count).Error; err != nil || count != journals {
		t.Fatalf("journals=%d, want %d; err=%v", count, journals, err)
	}
	var total decimal.Decimal
	if err := db.Table("ledger_lines").Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").Where("ledger_accounts.owner_id = ? AND ledger_accounts.asset = ?", "merchant_balance", "ALPHAUSD.TEMPO").Select("COALESCE(SUM(amount), 0)").Scan(&total).Error; err != nil {
		t.Fatal(err)
	}
	if !total.Neg().Equal(decimal.RequireFromString(amount)) {
		t.Fatalf("merchant journal balance=%s, want %s", total.Neg(), amount)
	}
}
