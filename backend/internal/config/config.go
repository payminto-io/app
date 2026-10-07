package config

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"

	environmentpkg "github.com/payminto/payminto/backend/internal/environment"
)

// Config is the top-level configuration struct loaded from environment variables at startup.
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Blockchain BlockchainConfig
	Solana     SolanaConfig
	Security   SecurityConfig
	Email      EmailConfig
	Telemetry  TelemetryConfig
	Gateway    GatewayConfig
	Modules    ModulesConfig
	Fees       FeesConfig
}

// FeesConfig holds FEES_* keys; internal/fees/README.md "Configuration" documents them.
type FeesConfig struct {
	// SurchargeForbiddenMethods is a comma list of payment methods, "none", or empty for the default (upi).
	SurchargeForbiddenMethods string
	// AssetPrecision adds on-chain assets as "CODE:decimals,...".
	AssetPrecision string
	// OperatorPlatformID is the only platform allowed to manage fee rules; 0 means unset.
	OperatorPlatformID uint
}

// GatewayConfig is the money mode this process serves; see internal/environment.
type GatewayConfig struct {
	// Environment is "test" or "live" (GATEWAY_ENVIRONMENT, default test).
	Environment string
	// TestDatabaseName (GATEWAY_TEST_DATABASE_NAME) names the one remote database a test process may
	// open although its name does not end in _test; it must equal current_database() exactly.
	TestDatabaseName string
}

// ModulesConfig holds the provider chosen for each slot module (docs/architecture/MODULES.md).
type ModulesConfig struct {
	// Providers maps a slot name ("custody") to its provider ("mock", "bitgo"); unset slots are absent.
	Providers map[string]string
}

// EmailConfig holds SMTP delivery settings. When Host/From are empty, email
// delivery falls back to a no-op logger (safe for dev/test).
type EmailConfig struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
}

// TelemetryConfig controls observability features.
type TelemetryConfig struct {
	// MetricsEnabled toggles the Prometheus /metrics endpoint.
	MetricsEnabled bool
	// SentryDSN, when set, enables Sentry error reporting.
	SentryDSN string
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int
	Environment     string
	CheckoutBaseURL string
	// AllowedOrigins is the comma-separated list of origins permitted to make
	// credentialed cross-origin requests. Empty means "any origin, no
	// credentials" (safe public-API default).
	AllowedOrigins []string
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
	// TestDatabase is the name reserved for test money; a live process refuses to open it.
	TestDatabase       string
	Username           string
	Password           string
	SSLMode            string
	SchemaMode         string
	AllowInsecureLocal bool
	// LedgerAppRole, when set, is narrowed to SELECT, INSERT on the ledger by cmd/migrate.
	LedgerAppRole string
}

const (
	SchemaModeValidate    = "validate"
	SchemaModeAutoMigrate = "auto-migrate"

	EnvironmentDevelopment = "DEVELOPMENT"
	EnvironmentTest        = "TEST"
	EnvironmentStaging     = "STAGING"
	EnvironmentProduction  = "PRODUCTION"
)

// RedisConfig holds the Redis connection URL.
type RedisConfig struct {
	URL string
}

// BlockchainConfig holds RPC endpoint URLs, the network type (testnet|mainnet),
// and the cold-storage sweep destinations.
type BlockchainConfig struct {
	NetworkType   string
	EthRPCURL     string
	BaseRPCURL    string
	BtcRPCURL     string
	TronAPIURL    string
	ColdWalletETH string // EVM cold-storage destination for native sweeps
	ColdWalletBTC string // Bitcoin cold-storage destination
	ColdWalletTRX string // Tron cold-storage destination
}

// SolanaConfig is the SOLANA_* section; mints live in the blockchain_currencies seeds, not here.
// Reasoning: internal/blockchain/solana/README.md.
type SolanaConfig struct {
	// Cluster is mainnet-beta, devnet, testnet or localnet; defaults from BLOCKCHAIN_NETWORK_TYPE.
	Cluster string
	// HotWalletAddress is the owner whose ATAs receive sweeps; empty disables sweeping.
	HotWalletAddress string
	// FeePayerKey funds sweep fees and ATA rent: base58 secret, CLI JSON array, or a path to one.
	FeePayerKey string
	// DevnetUSDTMint fills the devnet USDT row, which has no official mint; ignored on mainnet.
	DevnetUSDTMint           string
	PriorityFeeMicroLamports uint64
	ComputeUnitLimit         uint32
	SweepBatchSize           int
	CloseDepositAccounts     bool
	PollIntervalSeconds      int
	SweepIntervalSeconds     int
	// LateWindowDays keeps a deposit account watched after payment expiry (late money).
	LateWindowDays int
	// RequestsPerSecond is the per-RPC-node budget; the public endpoints allow about 10.
	RequestsPerSecond int
	// PostDepositJournals posts the deposit's payment journal from the watcher at finalization.
	// The switch posts payment journals on attempt success; set false once it is wired (see
	// .superpowers/solana-fix-1-report.md).
	PostDepositJournals bool
}

// SecurityConfig controls authentication and the optional custody capability.
// AESKey is retained for legacy configuration compatibility; current custody
// encrypts wallet material through the passphrase-derived SecretsVault.
type SecurityConfig struct {
	AESKey          string
	JWTSecret       string
	VaultPassphrase string
	CustodyEnabled  bool
	// DevKeystore enables the development keystore (DEV_KEYSTORE); live refuses to boot with it.
	DevKeystore bool
}

// Load reads all Payminto configuration from environment variables, falling back
// to sensible development defaults.
func Load() (*Config, error) {
	environment, err := normalizeEnvironment(envStr("SERVER", EnvironmentDevelopment))
	if err != nil {
		return nil, err
	}
	defaultSSLMode := "disable"
	if isDeploymentEnvironment(environment) {
		defaultSSLMode = "verify-full"
	}
	feesOperator, err := envUint("FEES_OPERATOR_PLATFORM_ID")
	if err != nil {
		return nil, err
	}
	custodyEnabled, err := envBoolStrict("CUSTODY_ENABLED", false)
	if err != nil {
		return nil, err
	}
	devKeystore, err := envBoolStrict("DEV_KEYSTORE", false)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Server: ServerConfig{
			Port:            envInt("API_PORT", 8080),
			Environment:     environment,
			CheckoutBaseURL: strings.TrimRight(envStr("CHECKOUT_BASE_URL", "http://localhost:3002"), "/"),
			AllowedOrigins:  envCSV("CORS_ALLOWED_ORIGINS"),
		},
		Database: DatabaseConfig{
			Host:               envStr("POSTGRES_HOST", "localhost"),
			Port:               envInt("POSTGRES_PORT", 5432),
			Database:           envStr("POSTGRES_DATABASE", "payminto"),
			TestDatabase:       envStr("POSTGRES_TEST_DATABASE", "payminto_test"),
			Username:           envStr("POSTGRES_USERNAME", "payminto"),
			Password:           envStr("POSTGRES_PASSWORD", ""),
			SSLMode:            envStr("POSTGRES_SSL_MODE", defaultSSLMode),
			SchemaMode:         envStr("POSTGRES_SCHEMA_MODE", SchemaModeValidate),
			AllowInsecureLocal: envBool("POSTGRES_ALLOW_INSECURE_LOCAL", false),
			LedgerAppRole:      envStr("POSTGRES_LEDGER_APP_ROLE", ""),
		},
		Redis: RedisConfig{
			URL: envStr("REDIS_URL", "redis://localhost:6379"),
		},
		Blockchain: BlockchainConfig{
			NetworkType:   envStr("BLOCKCHAIN_NETWORK_TYPE", "testnet"),
			EthRPCURL:     envStr("ETH_RPC_URL", ""),
			BaseRPCURL:    envStr("BASE_RPC_URL", ""),
			BtcRPCURL:     envStr("BTC_RPC_URL", ""),
			TronAPIURL:    envStr("TRON_API_URL", ""),
			ColdWalletETH: envStr("COLD_WALLET_ETH", ""),
			ColdWalletBTC: envStr("COLD_WALLET_BTC", ""),
			ColdWalletTRX: envStr("COLD_WALLET_TRX", ""),
		},
		Solana: SolanaConfig{
			Cluster:                  envStr("SOLANA_CLUSTER", ""),
			HotWalletAddress:         envStr("SOLANA_HOT_WALLET_ADDRESS", ""),
			FeePayerKey:              envStr("SOLANA_FEE_PAYER_KEY", ""),
			DevnetUSDTMint:           envStr("SOLANA_DEVNET_USDT_MINT", ""),
			PriorityFeeMicroLamports: uint64(envInt("SOLANA_PRIORITY_FEE_MICROLAMPORTS", 0)),
			ComputeUnitLimit:         uint32(envInt("SOLANA_COMPUTE_UNIT_LIMIT", 120000)),
			SweepBatchSize:           envInt("SOLANA_SWEEP_BATCH_SIZE", 5),
			CloseDepositAccounts:     envBool("SOLANA_CLOSE_DEPOSIT_ACCOUNTS", true),
			PollIntervalSeconds:      envInt("SOLANA_POLL_INTERVAL_SECONDS", 5),
			SweepIntervalSeconds:     envInt("SOLANA_SWEEP_INTERVAL_SECONDS", 30),
			LateWindowDays:           envInt("SOLANA_LATE_WINDOW_DAYS", 7),
			RequestsPerSecond:        envInt("SOLANA_RPC_REQUESTS_PER_SECOND", 10),
			PostDepositJournals:      envBool("SOLANA_POST_DEPOSIT_JOURNALS", true),
		},
		Security: SecurityConfig{
			AESKey:          envStr("AES_KEY", ""),
			JWTSecret:       envStr("JWT_SECRET", ""),
			VaultPassphrase: envStr("VAULT_PASSPHRASE", ""),
			CustodyEnabled:  custodyEnabled,
			DevKeystore:     devKeystore,
		},
		Email: EmailConfig{
			SMTPHost: envStr("SMTP_HOST", ""),
			SMTPPort: envInt("SMTP_PORT", 587),
			Username: envStr("SMTP_USERNAME", ""),
			Password: envStr("SMTP_PASSWORD", ""),
			From:     envStr("SMTP_FROM", ""),
		},
		Telemetry: TelemetryConfig{
			MetricsEnabled: envBool("METRICS_ENABLED", true),
			SentryDSN:      envStr("SENTRY_DSN", ""),
		},
		Gateway: GatewayConfig{
			Environment:      envStr("GATEWAY_ENVIRONMENT", string(environmentpkg.Test)),
			TestDatabaseName: strings.TrimSpace(envStr("GATEWAY_TEST_DATABASE_NAME", "")),
		},
		Modules: ModulesConfig{
			Providers: envSlotProviders(),
		},
		Fees: FeesConfig{
			SurchargeForbiddenMethods: envStr("FEES_SURCHARGE_FORBIDDEN_METHODS", ""),
			AssetPrecision:            envStr("FEES_ASSET_PRECISION", ""),
			OperatorPlatformID:        feesOperator,
		},
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	environment, err := normalizeEnvironment(c.Server.Environment)
	if err != nil {
		return err
	}
	c.Server.Environment = environment

	gatewayEnv, err := environmentpkg.Parse(c.Gateway.Environment)
	if err != nil {
		return fmt.Errorf("GATEWAY_ENVIRONMENT must be test or live; got %q", c.Gateway.Environment)
	}
	c.Gateway.Environment = string(gatewayEnv)

	c.Database.Database = strings.TrimSpace(c.Database.Database)
	c.Database.TestDatabase = strings.TrimSpace(c.Database.TestDatabase)
	if strings.ContainsAny(c.Gateway.TestDatabaseName, dsnUnsafe) {
		return fmt.Errorf("GATEWAY_TEST_DATABASE_NAME must be a plain database name; got %q", c.Gateway.TestDatabaseName)
	}
	for name, value := range map[string]string{"POSTGRES_DATABASE": c.Database.Database, "POSTGRES_TEST_DATABASE": c.Database.TestDatabase, "POSTGRES_HOST": c.Database.Host, "POSTGRES_USERNAME": c.Database.Username} {
		if value == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
		if strings.ContainsAny(value, dsnUnsafe) {
			return fmt.Errorf("%s must not contain whitespace, quotes, backslashes or '='; got %q", name, value)
		}
	}

	schemaMode := strings.ToLower(strings.TrimSpace(c.Database.SchemaMode))
	if schemaMode != SchemaModeValidate && schemaMode != SchemaModeAutoMigrate {
		return fmt.Errorf("POSTGRES_SCHEMA_MODE must be %q or %q; got %q", SchemaModeValidate, SchemaModeAutoMigrate, c.Database.SchemaMode)
	}
	c.Database.SchemaMode = schemaMode

	sslMode := strings.ToLower(strings.TrimSpace(c.Database.SSLMode))
	if !slices.Contains([]string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}, sslMode) {
		return fmt.Errorf("POSTGRES_SSL_MODE is invalid: %q", c.Database.SSLMode)
	}
	c.Database.SSLMode = sslMode

	if c.Security.CustodyEnabled && !isStrongSecret(c.Security.VaultPassphrase, 24) {
		return fmt.Errorf("VAULT_PASSPHRASE is required when CUSTODY_ENABLED=true and must contain at least 24 non-placeholder characters")
	}
	if !isDeploymentEnvironment(environment) {
		return nil
	}
	if schemaMode == SchemaModeAutoMigrate {
		return fmt.Errorf("%s POSTGRES_SCHEMA_MODE must be %q; auto-migrate is limited to development and test", strings.ToLower(environment), SchemaModeValidate)
	}
	if sslMode != "verify-full" {
		if !(c.Database.AllowInsecureLocal && isLocalDatabaseHost(c.Database.Host) && sslMode == "disable") {
			if isLocalDatabaseHost(c.Database.Host) {
				return fmt.Errorf("%s POSTGRES_SSL_MODE must be verify-full; set POSTGRES_ALLOW_INSECURE_LOCAL=true with sslmode=disable only for an explicit local database exception", strings.ToLower(environment))
			}
			return fmt.Errorf("%s POSTGRES_SSL_MODE must be verify-full for non-local databases; got %q", strings.ToLower(environment), c.Database.SSLMode)
		}
	}

	invalid := make([]string, 0, 3)
	if !isStrongSecret(c.Database.Password, 16) {
		invalid = append(invalid, "POSTGRES_PASSWORD (minimum 16 non-placeholder characters)")
	}
	if !isStrongSecret(c.Security.JWTSecret, 32) {
		invalid = append(invalid, "JWT_SECRET (minimum 32 non-placeholder characters)")
	}
	if len(invalid) != 0 {
		slices.Sort(invalid)
		return fmt.Errorf("%s configuration rejects weak or missing %s", strings.ToLower(environment), strings.Join(invalid, ", "))
	}
	return nil
}

func normalizeEnvironment(value string) (string, error) {
	environment := strings.ToUpper(strings.TrimSpace(value))
	if !slices.Contains([]string{EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction}, environment) {
		return "", fmt.Errorf("SERVER must be development, test, staging, or production; got %q", value)
	}
	return environment, nil
}

func isDeploymentEnvironment(environment string) bool {
	return environment == EnvironmentStaging || environment == EnvironmentProduction
}

func isLocalDatabaseHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isStrongSecret(value string, minimumLength int) bool {
	value = strings.TrimSpace(value)
	if len(value) < minimumLength {
		return false
	}
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(value))
	for _, marker := range []string{"changeme", "placeholder", "example", "generatestrongrandomsecret", "yourkey", "password"} {
		if strings.Contains(normalized, marker) {
			return false
		}
	}
	return true
}

// dsnUnsafe are the characters a connection parameter must never carry: each one would end or
// escape a keyword/value token and point the process at another database or option.
const dsnUnsafe = " \t\r\n=\\'\""

// DSN constructs a PostgreSQL keyword/value DSN; every value is quoted so it cannot inject parameters.
func (d DatabaseConfig) DSN() string {
	return "host=" + dsnValue(d.Host) +
		" port=" + strconv.Itoa(d.Port) +
		" user=" + dsnValue(d.Username) +
		" password=" + dsnValue(d.Password) +
		" dbname=" + dsnValue(d.Database) +
		" sslmode=" + dsnValue(d.SSLMode)
}

// dsnValue single-quotes a keyword/value parameter, escaping backslashes and quotes (libpq syntax).
func dsnValue(v string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}

// envSlotProviders reads <SLOT>_PROVIDER for every slot module; absent slots are left out.
func envSlotProviders() map[string]string {
	providers := map[string]string{}
	for _, slot := range environmentpkg.KnownSlots {
		if v := strings.TrimSpace(os.Getenv(strings.ToUpper(slot) + "_PROVIDER")); v != "" {
			providers[slot] = v
		}
	}
	return providers
}

// BootFacts projects the loaded configuration onto the environment module's boot gate.
func (c *Config) BootFacts() environmentpkg.BootFacts {
	return environmentpkg.BootFacts{
		Environment:                    environmentpkg.Environment(c.Gateway.Environment),
		DatabaseName:                   c.Database.Database,
		DatabaseHost:                   c.Database.Host,
		TestDatabaseName:               c.Database.TestDatabase,
		TestDatabaseAllowName:          c.Gateway.TestDatabaseName,
		DevKeystore:                    c.Security.DevKeystore || c.Security.AESKey != "",
		VaultDevMode:                   c.Security.CustodyEnabled && !isStrongSecret(c.Security.VaultPassphrase, 24),
		SlotProviders:                  c.Modules.Providers,
		DatabaseSSLMode:                c.Database.SSLMode,
		DatabaseInsecureLocalException: c.Database.AllowInsecureLocal,
		DeploymentHardened:             isDeploymentEnvironment(c.Server.Environment),
		NetworkType:                    c.Blockchain.NetworkType,
		JWTSecretWeak:                  isKnownDevSecret(c.Security.JWTSecret) || len(strings.TrimSpace(c.Security.JWTSecret)) < 32,
	}
}

// knownDevSecrets are the JWT secrets this repository ships for local stacks; live must never run on them.
var knownDevSecrets = []string{
	"payminto-development-jwt-secret-not-for-production",
	"dev-jwt-secret",
	"test-secret",
}

func isKnownDevSecret(value string) bool {
	value = strings.TrimSpace(value)
	return slices.Contains(knownDevSecrets, value) || !isStrongSecret(value, 32)
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envCSV reads a comma-separated env var into a trimmed, non-empty slice.
func envCSV(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// envBool reads a boolean env var. "1", "true", "yes" (case-insensitive) are
// true; "0", "false", "no" are false; anything else uses the fallback.
func envBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envBoolStrict(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean; got %q", key, value)
	}
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

// ConfigurationReader is a minimal interface EnforceModeMatch needs — keeps
// the config package from depending on the repository package directly.
type ConfigurationReader interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

// EnforceModeMatch reads the "mode" row from the configurations table and
// panics if it doesn't match the BLOCKCHAIN_NETWORK_TYPE env var. Catches the
// deployment mistake of pointing a mainnet-configured server at a testnet
// database (or vice versa). Safe to call before any writes.
//
// On first boot (no mode row present), it inserts the current env value as the
// stamp so subsequent boots have something to compare against.
//
// If envMode is empty, this is a no-op (development mode).
// CheckModeMatch is EnforceModeMatch without the first-boot write: it compares only when the row
// exists, for tools that must not stamp anything (cmd/migrate before the environment stamp).
func CheckModeMatch(envMode string, repo ConfigurationReader) error {
	if envMode == "" {
		return nil
	}
	c, err := repo.Get("mode")
	if err != nil {
		return nil
	}
	if c != "" && c != envMode {
		return fmt.Errorf("network mode mismatch: database is stamped as %q but BLOCKCHAIN_NETWORK_TYPE=%q; refusing", c, envMode)
	}
	return nil
}

func EnforceModeMatch(envMode string, repo ConfigurationReader) error {
	if envMode == "" {
		return nil
	}
	c, err := repo.Get("mode")
	if err != nil {
		// First boot — stamp the database with the env value and return.
		return repo.Set("mode", envMode)
	}
	if c != "" && c != envMode {
		return fmt.Errorf("network mode mismatch: database is stamped as %q but BLOCKCHAIN_NETWORK_TYPE=%q — refusing to start. To switch modes, redeploy with a fresh database", c, envMode)
	}
	return nil
}

// envUint reads an optional positive integer; unset is 0, anything else unparsable is an error.
func envUint(key string) (uint, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("%s must be a positive integer; got %q", key, raw)
	}
	return uint(n), nil
}
