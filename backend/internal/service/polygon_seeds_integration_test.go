//go:build integration

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func applySeeds(t *testing.T, db *gorm.DB, network string) {
	t.Helper()
	dir := filepath.Join("..", "..", "migrations", "seeds", network)
	for _, name := range []string{"9001_seed_blockchains.sql", "9002_seed_blockchain_currencies.sql"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(raw)).Error; err != nil {
			t.Fatalf("%s/%s: %v", network, name, err)
		}
	}
}

// Every chain that carries a token row must have exactly one native row, or gas cannot be booked.
func TestIntegration_SeedsGiveEveryTokenChainOneNativeRow(t *testing.T) {
	for _, network := range []string{"testnet", "mainnet"} {
		t.Run(network, func(t *testing.T) {
			db, cleanup := database.NewTestDB(t)
			defer cleanup()
			applySeeds(t, db, network)
			var bad []string
			err := db.Raw(`
SELECT blockchain_code FROM blockchain_currencies
GROUP BY blockchain_code
HAVING count(*) FILTER (WHERE lower(standard) = 'native') <> 1`).Scan(&bad).Error
			if err != nil {
				t.Fatal(err)
			}
			if len(bad) != 0 {
				t.Fatalf("chains without exactly one native row: %v", bad)
			}
		})
	}
}

func TestIntegration_PolygonSweepAndWithdrawalBookGasInPOL(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	applySeeds(t, db, "testnet")
	ctx := context.Background()
	journal := ledger.New(db)
	ledgerSvc := NewLedgerService(repository.NewAccountRepository(db), WithJournal(journal, blockchainCurrencyAssetResolver()))
	usdcPoly, err := repository.NewBlockchainCurrencyRepository(db).GetByBlockchainCodeAndCurrencyCode("POLYGON", "USDC")
	if err != nil {
		t.Fatal(err)
	}
	var poly models.Blockchain
	if err := db.Where("code = ?", "POLYGON").First(&poly).Error; err != nil {
		t.Fatal(err)
	}

	sweepSvc := NewSweepService(db, repository.NewSweepRepository(db), repository.NewSweepTransactionRepository(db), repository.NewBlockchainRepository(db), ledgerSvc)
	sweep, err := sweepSvc.CreateSweep(poly.ID)
	if err != nil {
		t.Fatal(err)
	}
	amount, gas := decimal.NewFromInt(250), decimal.RequireFromString("0.03")
	if err := sweepSvc.MarkCompleted(ctx, sweep.ID, amount, gas, usdcPoly.ID); err != nil {
		t.Fatalf("Polygon sweep: %v", err)
	}
	_, lines := journalLines(t, db, "payminto:sweep:"+idStr(sweep.ID))
	expectLines(t, lines, []postedLine{
		{"cold_wallet_assets", ledger.KindAsset, "USDC.POLYGON", amount},
		{"crypto_assets", ledger.KindAsset, "USDC.POLYGON", amount.Neg()},
		{"sweep_gas", ledger.KindExpense, "POL.POLYGON", gas},
		{"crypto_assets", ledger.KindAsset, "POL.POLYGON", gas.Neg()},
	})

	wdSvc := NewWithdrawalProcessingService(repository.NewWithdrawalRepository(db), repository.NewWithdrawRepository(db), ledgerSvc, nil, nil, nil)
	member := models.Member{Name: "polygon merchant"}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	platform := models.ExternalPlatform{Name: "p", SuccessEndpoint: "https://example.com"}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	w := &models.Withdrawal{
		ReferenceID: "wd_poly", State: models.WithdrawalStatePending, BlockchainCode: "POLYGON", CurrencyCode: "USDC",
		Amount: decimal.NewFromInt(40), ToAddress: "0xdead", MemberID: member.ID, ExternalPlatformID: platform.ID, BlockchainCurrencyID: usdcPoly.ID,
	}
	if err := db.Create(w).Error; err != nil {
		t.Fatal(err)
	}
	if err := wdSvc.Execute(ctx, w); err != nil {
		t.Fatalf("Polygon withdrawal: %v", err)
	}
	_, lines = journalLines(t, db, "payminto:withdrawal:"+idStr(w.ID))
	expectLines(t, lines, []postedLine{
		{"merchant_balance", ledger.KindLiability, "USDC.POLYGON", w.Amount},
		{"crypto_assets", ledger.KindAsset, "USDC.POLYGON", w.Amount.Neg()},
	})
	after, _ := repository.NewWithdrawalRepository(db).GetByID(w.ID)
	if after.State != models.WithdrawalStateProcessed {
		t.Fatalf("state = %s", after.State)
	}
}
