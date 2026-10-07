//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/service"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func databaseEnv(cfg config.DatabaseConfig) map[string]string {
	return map[string]string{
		"POSTGRES_HOST":          cfg.Host,
		"POSTGRES_PORT":          strconv.Itoa(cfg.Port),
		"POSTGRES_DATABASE":      cfg.Database,
		"POSTGRES_TEST_DATABASE": cfg.TestDatabase,
		"POSTGRES_USERNAME":      cfg.Username,
		"POSTGRES_PASSWORD":      cfg.Password,
		"POSTGRES_SSL_MODE":      cfg.SSLMode,
		"METRICS_ENABLED":        "false",
	}
}

// awaitHealthy polls /healthz until the child answers or exits.
func awaitHealthy(t *testing.T, base string, exited <-chan struct{}, output func() string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			t.Fatalf("process exited before becoming healthy:\n%s", output())
		default:
		}
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatal("process never became healthy")
}

func getEnvironment(t *testing.T, base, key string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest("GET", base+"/v2/environment", nil)
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

// runServer boots main in a child with env and returns the base URL and a stop function.
func runServer(t *testing.T, env map[string]string) (string, func() string) {
	t.Helper()
	port := freePort(t)
	env["API_PORT"] = strconv.Itoa(port)
	cmd, wait := bootProcess(t, env)
	exited := make(chan struct{})
	var exitCode int
	var output string
	go func() {
		exitCode, output = wait()
		close(exited)
	}()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	stop := func() string {
		select {
		case <-exited:
			return output
		default:
		}
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-exited:
		case <-time.After(40 * time.Second):
			_ = cmd.Process.Kill()
			<-exited
		}
		_ = exitCode
		return output
	}
	t.Cleanup(func() { stop() })
	awaitHealthy(t, base, exited, func() string { return output })
	return base, stop
}

func TestIntegration_TestProcessBootsAgainstTheTestDatabase(t *testing.T) {
	dbCfg, stop := database.NewTestDBConfig(t)
	defer stop()
	env := databaseEnv(dbCfg)
	env["GATEWAY_ENVIRONMENT"] = "test"
	env["SERVER"] = "development"
	env["POSTGRES_SCHEMA_MODE"] = "auto-migrate"
	env["JWT_SECRET"] = "dev-jwt-secret"
	base, _ := runServer(t, env)
	if code, _ := getEnvironment(t, base, ""); code != http.StatusUnauthorized {
		t.Fatalf("/v2/environment without auth = %d, want 401", code)
	}
}

func TestIntegration_LiveProcessBootsAndRefusesTestKeys(t *testing.T) {
	dbCfg, stop := database.NewTestDBConfig(t)
	defer stop()
	admin, err := database.Connect(dbCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec(`CREATE DATABASE payminto_live`).Error; err != nil {
		t.Fatal(err)
	}
	liveCfg := dbCfg
	liveCfg.Database = "payminto_live"
	live, err := database.Connect(liveCfg)
	if err != nil {
		t.Fatal(err)
	}
	// Live runs in validate mode, so the schema is prepared the way an operator would: the privileged
	// role migrates, then the server connects as a narrowed application role.
	if err := database.AutoMigrate(live); err != nil {
		t.Fatal(err)
	}
	if err := database.MigrateExpandSchema(live); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ApplyMigrations(context.Background(), live); err != nil {
		t.Fatal(err)
	}
	const appRole, appPassword = "payminto_app_live", "payminto_app_live_password_x"
	for _, stmt := range []string{
		`CREATE ROLE ` + appRole + ` LOGIN PASSWORD '` + appPassword + `'`,
		`GRANT USAGE ON SCHEMA public TO ` + appRole,
		`GRANT ALL ON ALL TABLES IN SCHEMA public TO ` + appRole,
		`GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO ` + appRole,
	} {
		if err := live.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := ledger.GrantAppRole(live, appRole); err != nil {
		t.Fatal(err)
	}
	member := models.Member{Name: "m", MemberType: "root", State: "active"}
	if err := live.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	platform := models.ExternalPlatform{Name: "p"}
	if err := live.Create(&platform).Error; err != nil {
		t.Fatal(err)
	}
	keys := map[environment.Environment]string{}
	for _, keyEnv := range environment.All() {
		raw, err := service.GenerateAPIKeyFor(keyEnv)
		if err != nil {
			t.Fatal(err)
		}
		row := service.NewAPIKeyRow(raw, keyEnv, platform.ID)
		row.MemberID = &member.ID
		if err := live.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		keys[keyEnv] = raw
	}
	legacy := "pm_legacy_key_issued_before_environments"
	if err := live.Create(&models.APIKey{Key: service.HashAPIKey(legacy), Status: "active", MemberID: &member.ID, ExternalPlatformID: platform.ID}).Error; err != nil {
		t.Fatal(err)
	}

	appCfg := liveCfg
	appCfg.Username, appCfg.Password = appRole, appPassword
	base, _ := runServer(t, liveProcessEnv(appCfg))

	code, body := getEnvironment(t, base, keys[environment.Live])
	if code != http.StatusOK || body["environment"] != "live" {
		t.Fatalf("live key on live process = %d %v", code, body)
	}
	for name, key := range map[string]string{"test key": keys[environment.Test], "legacy key": legacy} {
		code, body := getEnvironment(t, base, key)
		if code != http.StatusUnauthorized || body["code"] != "api_key_environment_mismatch" {
			t.Fatalf("%s on live process = %d %v, want 401 api_key_environment_mismatch", name, code, body)
		}
	}
}

// stampedDatabase creates and migrates a database named name in the container and stamps it env.
func stampedDatabase(t *testing.T, dbCfg config.DatabaseConfig, name string, env environment.Environment) config.DatabaseConfig {
	t.Helper()
	admin, err := database.Connect(dbCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec(`CREATE DATABASE ` + name).Error; err != nil {
		t.Fatal(err)
	}
	cfg := dbCfg
	cfg.Database = name
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.MigrateExpandSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&environment.StampRow{ID: environment.StampID, Environment: env, StampedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	return cfg
}

func liveProcessEnv(cfg config.DatabaseConfig) map[string]string {
	env := databaseEnv(cfg)
	env["GATEWAY_ENVIRONMENT"] = "live"
	env["SERVER"] = "production"
	env["POSTGRES_ALLOW_INSECURE_LOCAL"] = "true"
	env["JWT_SECRET"] = "a-strong-jwt-secret-value-with-32-plus-chars"
	env["BLOCKCHAIN_NETWORK_TYPE"] = "mainnet"
	return env
}

func TestIntegration_LiveRefusesAReachableDatabaseStampedTest(t *testing.T) {
	dbCfg, stop := database.NewTestDBConfig(t)
	defer stop()
	// The name passes every string check; only what the database itself says can refuse it.
	liveCfg := stampedDatabase(t, dbCfg, "payminto_prod", environment.Test)
	requireRefusal(t, liveProcessEnv(liveCfg), "is stamped test; a live process must not open it")
}

func TestIntegration_TestProcessRefusesALoopbackDatabaseStampedLive(t *testing.T) {
	dbCfg, stop := database.NewTestDBConfig(t)
	defer stop()
	liveCfg := stampedDatabase(t, dbCfg, "payminto", environment.Live)
	env := databaseEnv(liveCfg)
	env["GATEWAY_ENVIRONMENT"] = "test"
	env["SERVER"] = "development"
	env["POSTGRES_SCHEMA_MODE"] = "auto-migrate"
	env["JWT_SECRET"] = "dev-jwt-secret"
	requireRefusal(t, env, "stamped live")
}

func TestIntegration_FirstBootStampsTheDatabase(t *testing.T) {
	dbCfg, stop := database.NewTestDBConfig(t)
	defer stop()
	env := databaseEnv(dbCfg)
	env["GATEWAY_ENVIRONMENT"] = "test"
	env["SERVER"] = "development"
	env["POSTGRES_SCHEMA_MODE"] = "auto-migrate"
	env["JWT_SECRET"] = "dev-jwt-secret"
	_, stopServer := runServer(t, env)
	stopServer()
	db, err := database.Connect(dbCfg)
	if err != nil {
		t.Fatal(err)
	}
	var row environment.StampRow
	if err := db.First(&row, environment.StampID).Error; err != nil || row.Environment != environment.Test {
		t.Fatalf("stamp after first boot = %+v, %v", row, err)
	}
}
