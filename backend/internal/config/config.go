package config

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Config is the top-level configuration struct loaded from environment variables at startup.
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Blockchain BlockchainConfig
	Security   SecurityConfig
	Email      EmailConfig
	Telemetry  TelemetryConfig
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
	Host               string
	Port               int
	Database           string
	Username           string
	Password           string
	SSLMode            string
	SchemaMode         string
	AllowInsecureLocal bool
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

// SecurityConfig controls authentication and the optional custody capability.
// AESKey is retained for legacy configuration compatibility; current custody
// encrypts wallet material through the passphrase-derived SecretsVault.
type SecurityConfig struct {
	AESKey          string
	JWTSecret       string
	VaultPassphrase string
	CustodyEnabled  bool
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
	custodyEnabled, err := envBoolStrict("CUSTODY_ENABLED", false)
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
			Username:           envStr("POSTGRES_USERNAME", "payminto"),
			Password:           envStr("POSTGRES_PASSWORD", ""),
			SSLMode:            envStr("POSTGRES_SSL_MODE", defaultSSLMode),
			SchemaMode:         envStr("POSTGRES_SCHEMA_MODE", SchemaModeValidate),
			AllowInsecureLocal: envBool("POSTGRES_ALLOW_INSECURE_LOCAL", false),
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
		Security: SecurityConfig{
			AESKey:          envStr("AES_KEY", ""),
			JWTSecret:       envStr("JWT_SECRET", ""),
			VaultPassphrase: envStr("VAULT_PASSPHRASE", ""),
			CustodyEnabled:  custodyEnabled,
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

// DSN constructs a PostgreSQL DSN string from the DatabaseConfig fields.
func (d DatabaseConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + strconv.Itoa(d.Port) +
		" user=" + d.Username +
		" password=" + d.Password +
		" dbname=" + d.Database +
		" sslmode=" + d.SSLMode
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
