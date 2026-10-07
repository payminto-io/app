package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Environment != "DEVELOPMENT" {
		t.Errorf("expected DEVELOPMENT, got %s", cfg.Server.Environment)
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	os.Setenv("API_PORT", "9090")
	os.Setenv("POSTGRES_HOST", "db.example.com")
	defer os.Unsetenv("API_PORT")
	defer os.Unsetenv("POSTGRES_HOST")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("expected db.example.com, got %s", cfg.Database.Host)
	}
}

func TestLoad_ProductionRejectsMissingCriticalSecrets(t *testing.T) {
	t.Setenv("SERVER", "production")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want production configuration rejection")
	}
	for _, name := range []string{"POSTGRES_PASSWORD", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("Load() error = %q, want missing variable %s", err, name)
		}
	}
	for _, unused := range []string{"AES_KEY", "VAULT_PASSPHRASE"} {
		if strings.Contains(err.Error(), unused) {
			t.Errorf("Load() error = %q, must not require disabled custody secret %s", err, unused)
		}
	}
}

func TestLoad_EnvironmentIsClosedAndCanonical(t *testing.T) {
	for input, want := range map[string]string{
		"development": EnvironmentDevelopment,
		"TEST":        EnvironmentTest,
		"StAgInG":     EnvironmentStaging,
		"PRODUCTION":  EnvironmentProduction,
	} {
		t.Run(input, func(t *testing.T) {
			t.Setenv("SERVER", input)
			if want == EnvironmentStaging || want == EnvironmentProduction {
				setDeploymentSecrets(t)
				t.Setenv("SERVER", input)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Server.Environment != want {
				t.Fatalf("Environment = %q, want %q", cfg.Server.Environment, want)
			}
		})
	}
}

func TestLoad_RejectsUnknownEnvironment(t *testing.T) {
	t.Setenv("SERVER", "prodution")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SERVER") {
		t.Fatalf("Load() error = %v, want closed environment enum rejection", err)
	}
}

func TestLoad_DatabaseSSLModeIsConfigurable(t *testing.T) {
	t.Setenv("SERVER", "DEVELOPMENT")
	t.Setenv("POSTGRES_SSL_MODE", "verify-full")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.SSLMode != "verify-full" {
		t.Errorf("Database.SSLMode = %q, want verify-full", cfg.Database.SSLMode)
	}
	if !strings.Contains(cfg.Database.DSN(), "sslmode=verify-full") {
		t.Errorf("Database.DSN() = %q, want configured SSL mode", cfg.Database.DSN())
	}
}

func TestLoad_ProductionRejectsUnsafeDatabaseSSLMode(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("POSTGRES_SSL_MODE", "disable")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_SSL_MODE") {
		t.Fatalf("Load() error = %v, want unsafe SSL mode rejection", err)
	}
}

func TestLoad_StagingUsesProductionGradeStartupPolicy(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("SERVER", "staging")
	t.Setenv("POSTGRES_SCHEMA_MODE", SchemaModeAutoMigrate)

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_SCHEMA_MODE") {
		t.Fatalf("Load() error = %v, want staging auto-migrate rejection", err)
	}
}

func TestLoad_NonLocalDeploymentDefaultsToVerifyFull(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("SERVER", "staging")
	t.Setenv("POSTGRES_HOST", "database.internal")
	t.Setenv("POSTGRES_SSL_MODE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.SSLMode != "verify-full" {
		t.Fatalf("Database.SSLMode = %q, want verify-full", cfg.Database.SSLMode)
	}
}

func TestLoad_NonLocalDeploymentRequiresVerifyFull(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("POSTGRES_HOST", "database.internal")
	t.Setenv("POSTGRES_SSL_MODE", "require")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "verify-full") {
		t.Fatalf("Load() error = %v, want hostname-verifying TLS rejection", err)
	}
}

func TestLoad_LocalDeploymentAllowsExplicitInsecureDatabaseException(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_SSL_MODE", "disable")
	t.Setenv("POSTGRES_ALLOW_INSECURE_LOCAL", "true")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v, want explicit local exception", err)
	}
}

func TestLoad_LocalDeploymentRejectsImplicitInsecureDatabase(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("POSTGRES_HOST", "127.0.0.1")
	t.Setenv("POSTGRES_SSL_MODE", "disable")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_ALLOW_INSECURE_LOCAL") {
		t.Fatalf("Load() error = %v, want explicit local exception requirement", err)
	}
}

func TestLoad_RejectsUnknownDatabaseSSLMode(t *testing.T) {
	t.Setenv("SERVER", "DEVELOPMENT")
	t.Setenv("POSTGRES_SSL_MODE", "plaintext-ish")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_SSL_MODE") {
		t.Fatalf("Load() error = %v, want unknown SSL mode rejection", err)
	}
}

func TestLoad_ProductionRejectsAutoMigrateAtStartup(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("POSTGRES_SCHEMA_MODE", "auto-migrate")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_SCHEMA_MODE") {
		t.Fatalf("Load() error = %v, want production auto-migrate rejection", err)
	}
}

func TestLoad_DeploymentRejectsWeakOrPlaceholderSecrets(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "short JWT secret", key: "JWT_SECRET", value: "too-short"},
		{name: "placeholder JWT secret", key: "JWT_SECRET", value: "generate_a_strong_random_secret"},
		{name: "placeholder database password", key: "POSTGRES_PASSWORD", value: "change_me_in_production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDeploymentSecrets(t)
			t.Setenv(tt.key, tt.value)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("Load() error = %v, want %s rejection", err, tt.key)
			}
		})
	}
}

func TestLoad_CustodySecretsAreRequiredOnlyWhenCustodyIsEnabled(t *testing.T) {
	setDeploymentSecrets(t)
	t.Setenv("CUSTODY_ENABLED", "false")
	t.Setenv("AES_KEY", "")
	t.Setenv("VAULT_PASSPHRASE", "")
	if _, err := Load(); err != nil {
		t.Fatalf("Load() disabled custody error = %v", err)
	}

	t.Setenv("CUSTODY_ENABLED", "true")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "VAULT_PASSPHRASE") {
		t.Fatalf("Load() enabled custody error = %v, want vault requirement", err)
	}
	if strings.Contains(err.Error(), "AES_KEY") {
		t.Fatalf("Load() enabled custody error = %v, AES_KEY is not consumed by custody implementation", err)
	}
}

func TestLoad_RejectsUnknownCustodyCapabilityValue(t *testing.T) {
	t.Setenv("CUSTODY_ENABLED", "enabled-ish")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CUSTODY_ENABLED") {
		t.Fatalf("Load() error = %v, want invalid capability value rejection", err)
	}
}

func setDeploymentSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("SERVER", "production")
	t.Setenv("POSTGRES_HOST", "database.internal")
	t.Setenv("POSTGRES_PASSWORD", "database-secret-with-32-random-chars")
	t.Setenv("JWT_SECRET", "jwt-secret-with-at-least-32-random-chars")
	t.Setenv("CUSTODY_ENABLED", "false")
}

type fakeConfigRepo struct {
	values map[string]string
}

func (f *fakeConfigRepo) Get(key string) (string, error) {
	v, ok := f.values[key]
	if !ok {
		return "", errNotFound
	}
	return v, nil
}

func (f *fakeConfigRepo) Set(key, value string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

var errNotFound = errors.New("not found")

func TestEnforceModeMatch_EmptyEnv(t *testing.T) {
	err := EnforceModeMatch("", &fakeConfigRepo{})
	if err != nil {
		t.Errorf("empty env should be no-op, got %v", err)
	}
}

func TestEnforceModeMatch_FirstBootStamps(t *testing.T) {
	repo := &fakeConfigRepo{}
	if err := EnforceModeMatch("testnet", repo); err != nil {
		t.Fatal(err)
	}
	if repo.values["mode"] != "testnet" {
		t.Errorf("expected repo to be stamped with testnet, got %q", repo.values["mode"])
	}
}

func TestEnforceModeMatch_Matches(t *testing.T) {
	repo := &fakeConfigRepo{values: map[string]string{"mode": "mainnet"}}
	if err := EnforceModeMatch("mainnet", repo); err != nil {
		t.Errorf("matching modes should not error, got %v", err)
	}
}

func TestEnforceModeMatch_Mismatch(t *testing.T) {
	repo := &fakeConfigRepo{values: map[string]string{"mode": "mainnet"}}
	err := EnforceModeMatch("testnet", repo)
	if err == nil {
		t.Error("expected mismatch error")
	}
}

func TestLoad_GatewayEnvironmentDefaultsToTest(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Gateway.Environment != "test" {
		t.Errorf("GATEWAY_ENVIRONMENT default = %q, want test", cfg.Gateway.Environment)
	}
	if cfg.Database.TestDatabase != "payminto_test" {
		t.Errorf("POSTGRES_TEST_DATABASE default = %q", cfg.Database.TestDatabase)
	}
	if cfg.Security.DevKeystore {
		t.Error("DEV_KEYSTORE must default to false")
	}
	if len(cfg.Modules.Providers) != 0 {
		t.Errorf("no provider should be configured by default, got %v", cfg.Modules.Providers)
	}
}

func TestLoad_GatewayEnvironmentIsClosed(t *testing.T) {
	t.Setenv("GATEWAY_ENVIRONMENT", "LIVE")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Gateway.Environment != "live" {
		t.Errorf("GATEWAY_ENVIRONMENT = %q, want canonical live", cfg.Gateway.Environment)
	}
	t.Setenv("GATEWAY_ENVIRONMENT", "sandbox")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "GATEWAY_ENVIRONMENT") {
		t.Fatalf("Load() error = %v, want GATEWAY_ENVIRONMENT rejection", err)
	}
}

func TestLoad_SlotProvidersAndDevKeystore(t *testing.T) {
	t.Setenv("CUSTODY_PROVIDER", "mock")
	t.Setenv("CONNECTORS_PROVIDER", "stripe")
	t.Setenv("DEV_KEYSTORE", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Modules.Providers["custody"] != "mock" || cfg.Modules.Providers["connectors"] != "stripe" {
		t.Errorf("providers = %v", cfg.Modules.Providers)
	}
	if !cfg.Security.DevKeystore {
		t.Error("DEV_KEYSTORE=true not read")
	}
}

func TestBootFacts_ProjectsConfig(t *testing.T) {
	t.Setenv("GATEWAY_ENVIRONMENT", "live")
	t.Setenv("SERVER", "production")
	t.Setenv("POSTGRES_HOST", "db.internal")
	t.Setenv("POSTGRES_DATABASE", "gateway")
	t.Setenv("POSTGRES_PASSWORD", "a-strong-database-secret-value")
	t.Setenv("JWT_SECRET", "a-strong-jwt-secret-value-with-32-plus-chars")
	t.Setenv("AES_KEY", "legacy-local-master-key")
	t.Setenv("BLOCKCHAIN_NETWORK_TYPE", "mainnet")
	t.Setenv("CUSTODY_PROVIDER", "bitgo")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	facts := cfg.BootFacts()
	if facts.Environment != "live" || facts.DatabaseName != "gateway" || facts.DatabaseHost != "db.internal" {
		t.Errorf("facts = %+v", facts)
	}
	if !facts.DevKeystore {
		t.Error("AES_KEY must count as a local vault master key")
	}
	if !facts.DeploymentHardened || facts.DatabaseSSLMode != "verify-full" || facts.NetworkType != "mainnet" {
		t.Errorf("facts = %+v", facts)
	}
	if facts.SlotProviders["custody"] != "bitgo" || facts.TestDatabaseName != "payminto_test" {
		t.Errorf("facts = %+v", facts)
	}
}
