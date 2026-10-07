//go:build integration

package modules

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/config"
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

	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto_prod", databaseHost: cfg.Host, testDatabase: "payminto_test", networkType: "mainnet"}
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

// nonLoopbackHost returns an address for the container that is not loopback: the machine's own LAN
// address when the published port answers there, else the FQDN form of localhost, which resolves to
// 127.0.0.1 but is not literally "localhost" or an IP, so the policy treats it as remote.
func nonLoopbackHost(t *testing.T, cfg config.DatabaseConfig) string {
	t.Helper()
	candidates := []string{}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLoopback() {
				candidates = append(candidates, ipn.IP.String())
			}
		}
	}
	candidates = append(candidates, "localhost.")
	for _, host := range candidates {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(cfg.Port)), 2*time.Second)
		if err != nil {
			continue
		}
		conn.Close()
		return host
	}
	t.Fatal("no non-loopback route to the test container")
	return ""
}

func TestIntegration_RemotePreTicketDatabaseIsRefusedUntilAdoptedLive(t *testing.T) {
	ctx := context.Background()
	cfg, stop := database.NewTestDBConfig(t)
	defer stop()
	admin, err := database.Connect(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec(`CREATE DATABASE payminto`).Error; err != nil {
		t.Fatal(err)
	}
	remote := cfg
	remote.Host = nonLoopbackHost(t, cfg)
	remote.Database = "payminto"
	if environment.NormalizeDatabaseName(remote.Host) == "localhost" {
		t.Fatal("host alias is loopback by name")
	}
	db, err := database.Connect(remote)
	if err != nil {
		t.Fatalf("connect through %s: %v", remote.Host, err)
	}
	// The pre-ticket deployment: schema, money rows and a legacy key, then the upgrade's migration.
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.MigrateExpandSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	preTicket := ledger.New(db, ledger.WithEnvironment(environment.Test))
	amount := decimal.RequireFromString("40")
	if _, err := preTicket.Post(ctx, ledger.Journal{
		Kind: ledger.KindPayment, Reference: ledger.Reference{Type: "payment", ID: "p1"}, IdempotencyKey: "p1",
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: "USDC", Kind: ledger.KindAsset}, Amount: amount},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "m1", Asset: "USDC", Kind: ledger.KindLiability}, Amount: amount.Neg()},
		},
	}); err != nil {
		t.Fatal(err)
	}
	platform := models.ExternalPlatform{Name: "merchant"}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.APIKey{Key: "sha256-of-pm_legacy", Status: "active", ExternalPlatformID: platform.ID}).Error; err != nil {
		t.Fatal(err)
	}

	testProc := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: remote.Host, testDatabase: "payminto_test"}
	if err := testProc.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("a test process opened the remote production database: %v", err)
	}
	liveProc := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: remote.Host, testDatabase: "payminto_test", networkType: "mainnet"}
	if err := liveProc.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("live VerifyDatabase on the unstamped database: %v", err)
	}
	if err := liveProc.VerifySchema(ctx, db); err != nil {
		t.Fatalf("live VerifySchema: %v", err)
	}
	err = liveProc.Stamp(ctx, db)
	if !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "adopt-live --confirm-adopt-live=payminto") {
		t.Fatalf("live boot stamped or wrongly refused a populated unstamped database: %v", err)
	}
	var keyEnv string
	db.Raw(`SELECT environment FROM api_keys WHERE key = 'sha256-of-pm_legacy'`).Scan(&keyEnv)
	if keyEnv != "test" {
		t.Fatalf("legacy key relabelled before adoption: %q", keyEnv)
	}
	// N5: the function must belong to the ledger owner, or adoption refuses early with the manual step.
	if err := db.Exec(`CREATE ROLE adopt_other NOLOGIN`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`ALTER FUNCTION ledger_adopt_environment(text) OWNER TO adopt_other`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := liveProc.AdoptLive(ctx, db, "payminto"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "ALTER FUNCTION ledger_adopt_environment(text) OWNER TO ledger_owner") {
		t.Fatalf("adoption with a misowned function = %v, want refusal with the manual step", err)
	}
	if err := db.Exec(`ALTER FUNCTION ledger_adopt_environment(text) OWNER TO ledger_owner`).Error; err != nil {
		t.Fatal(err)
	}

	result, err := liveProc.AdoptLive(ctx, db, "payminto")
	if err != nil {
		t.Fatalf("adopt-live: %v", err)
	}
	if result.LedgerAccounts != 2 || result.LedgerJournals != 1 || result.APIKeys != 1 {
		t.Fatalf("result = %+v", result)
	}
	if err := liveProc.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("live boot after adoption refused: %v", err)
	}
	if err := liveProc.Stamp(ctx, db); err != nil {
		t.Fatalf("live stamp after adoption: %v", err)
	}
	liveLedger := ledger.New(db, ledger.WithEnvironment(environment.Live))
	totals, err := liveLedger.Balances(ctx, ledger.OwnerMember, "m1")
	if err != nil || !totals["USDC"].Equal(amount) {
		t.Fatalf("live balances after adoption = %v, %v", totals, err)
	}
	if got, err := preTicket.Balances(ctx, ledger.OwnerMember, "m1"); err != nil || len(got) != 0 {
		t.Fatalf("test environment still sees adopted balances: %v, %v", got, err)
	}
	if err := testProc.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("test process accepted the adopted live database: %v", err)
	}
}

func TestIntegration_AdoptionRefusesTheWrongNetworkMode(t *testing.T) {
	ctx := context.Background()
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	if _, err := database.ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	testLedger := ledger.New(db, ledger.WithEnvironment(environment.Test))
	amount := decimal.RequireFromString("5")
	if _, err := testLedger.Post(ctx, ledger.Journal{
		Kind: ledger.KindPayment, Reference: ledger.Reference{Type: "payment", ID: "p1"}, IdempotencyKey: "p1",
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: "USDC", Kind: ledger.KindAsset}, Amount: amount},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "m1", Asset: "USDC", Kind: ledger.KindLiability}, Amount: amount.Neg()},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Configuration{Key: "mode", Value: "testnet"}).Error; err != nil {
		t.Fatal(err)
	}
	// The container database is named payminto_test; live adoption needs a live-acceptable name, so test
	// on the mode alone through a module whose policy sees a plain name while Postgres reports the real one.
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto_test", databaseHost: "localhost", testDatabase: "other_test", networkType: "mainnet"}
	if _, err := live.AdoptLive(ctx, db, "payminto_test"); !environment.IsBootRefusal(err) {
		t.Fatalf("adopt-live on a *_test name = %v, want refusal", err)
	}
	test := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto_test", databaseHost: "localhost", testDatabase: "payminto_test", networkType: "testnet"}
	db.Model(&models.Configuration{}).Where("key = ?", "mode").Update("value", "mainnet")
	if _, err := test.AdoptTest(ctx, db, "payminto_test"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "mainnet") {
		t.Fatalf("adopt-test on a mainnet database = %v, want refusal", err)
	}
	var stamps int64
	db.Model(&environment.StampRow{}).Count(&stamps)
	if stamps != 0 {
		t.Fatal("a refused adoption wrote a stamp")
	}
	db.Model(&models.Configuration{}).Where("key = ?", "mode").Update("value", "testnet")
	if _, err := test.AdoptTest(ctx, db, "payminto_test"); err != nil {
		t.Fatalf("adopt-test on a testnet database: %v", err)
	}
}

func TestIntegration_AdoptLiveRefusesATestnetDatabaseWithALiveName(t *testing.T) {
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
	if err := prod.Create(&models.Configuration{Key: "mode", Value: "testnet"}).Error; err != nil {
		t.Fatal(err)
	}
	platform := models.ExternalPlatform{Name: "merchant"}
	if err := prod.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto_prod", databaseHost: cfg.Host, testDatabase: "payminto_test", networkType: "mainnet"}
	if _, err := live.AdoptLive(ctx, prod, "payminto_prod"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "network mode") {
		t.Fatalf("adopt-live on a testnet database = %v, want refusal naming the network mode", err)
	}
	var stamps int64
	prod.Model(&environment.StampRow{}).Count(&stamps)
	if stamps != 0 {
		t.Fatal("a refused adoption wrote a stamp")
	}
	prod.Model(&models.Configuration{}).Where("key = ?", "mode").Update("value", "mainnet")
	if _, err := live.AdoptLive(ctx, prod, "payminto_prod"); err != nil {
		t.Fatalf("adopt-live on a mainnet database: %v", err)
	}
}
