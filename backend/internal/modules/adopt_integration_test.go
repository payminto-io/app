//go:build integration

package modules

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

func TestIntegration_AdoptLiveRelabelsAPopulatedTestDatabaseOnce(t *testing.T) {
	ctx := context.Background()
	cfg, stop := database.NewTestDBConfig(t)
	defer stop()
	admin, err := database.Connect(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec(`CREATE DATABASE payminto_prod`).Error; err != nil {
		t.Fatal(err)
	}
	prodCfg := cfg
	prodCfg.Database = "payminto_prod"
	prod, err := database.Connect(prodCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(prod); err != nil {
		t.Fatal(err)
	}
	if err := database.MigrateExpandSchema(prod); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ApplyMigrations(ctx, prod); err != nil {
		t.Fatal(err)
	}
	// A deployment that ran before environments existed: test journals, a legacy key, and the stamp
	// the upgraded binary wrote on its first boot.
	testLedger := ledger.New(prod, ledger.WithEnvironment(environment.Test))
	amount := decimal.RequireFromString("12.5")
	for _, key := range []string{"p1", "p2"} {
		_, err := testLedger.Post(ctx, ledger.Journal{
			Kind: ledger.KindPayment, Reference: ledger.Reference{Type: "payment", ID: key}, IdempotencyKey: key,
			Lines: []ledger.Line{
				{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: "USDC", Kind: ledger.KindAsset}, Amount: amount},
				{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "m1", Asset: "USDC", Kind: ledger.KindLiability}, Amount: amount.Neg()},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	platform := models.ExternalPlatform{Name: "merchant"}
	if err := prod.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	if err := prod.Create(&models.APIKey{Key: "sha256-of-pm_legacy_key", Status: "active", ExternalPlatformID: platform.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := prod.Create(&environment.StampRow{ID: environment.StampID, Environment: environment.Test, StampedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}

	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto_prod", databaseHost: cfg.Host, testDatabase: "payminto_test"}
	if _, err := live.AdoptLive(ctx, prod, "payminto_test"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "confirm-adopt-live") {
		t.Fatalf("confirmation naming another database accepted: %v", err)
	}
	result, err := live.AdoptLive(ctx, prod, "payminto_prod")
	if err != nil {
		t.Fatalf("adopt-live: %v", err)
	}
	if result.LedgerAccounts != 2 || result.LedgerJournals != 2 || result.APIKeys != 1 {
		t.Fatalf("result = %+v", result)
	}
	liveLedger := ledger.New(prod, ledger.WithEnvironment(environment.Live))
	totals, err := liveLedger.Balances(ctx, ledger.OwnerMember, "m1")
	if err != nil || !totals["USDC"].Equal(decimal.RequireFromString("25")) {
		t.Fatalf("live balances after adoption = %v, %v", totals, err)
	}
	if _, err := testLedger.Balances(ctx, ledger.OwnerMember, "m1"); err != nil {
		t.Fatal(err)
	} else if got, _ := testLedger.Balances(ctx, ledger.OwnerMember, "m1"); len(got) != 0 {
		t.Fatalf("test environment still sees adopted balances: %v", got)
	}
	if err := live.VerifyDatabase(ctx, prod); err != nil {
		t.Fatalf("live refused the adopted database: %v", err)
	}
	if err := live.VerifySchema(ctx, prod); err != nil {
		t.Fatalf("adopted legacy key failed the prefix check: %v", err)
	}
	if _, err := live.AdoptLive(ctx, prod, "payminto_prod"); !environment.IsBootRefusal(err) {
		t.Fatalf("second adoption accepted: %v", err)
	}
	// The append-only trigger is back on.
	if err := prod.Exec(`UPDATE ledger_accounts SET environment = 'test'`).Error; err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("ledger_accounts writable after adoption: %v", err)
	}
	// Only the migrator may call the relabel function.
	for _, stmt := range []string{
		`CREATE ROLE adopt_app LOGIN PASSWORD 'adopt_app_pw'`,
		`GRANT USAGE ON SCHEMA public TO adopt_app`,
		`GRANT SELECT, INSERT ON ledger_accounts, ledger_journals, ledger_lines TO adopt_app`,
	} {
		if err := prod.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	app := database.ConnectTestDBAs(t, prod, "adopt_app", "adopt_app_pw")
	if err := app.Exec(`SELECT ledger_adopt_environment('live')`).Error; err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("application role could call the relabel function: %v", err)
	}
}
