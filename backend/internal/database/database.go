package database

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection using the given DatabaseConfig and configures
// connection pool limits.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn, err := VerifiedDSN(cfg)
	if err != nil {
		return nil, err
	}
	logLevel := logger.Info
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	return db, nil
}

// VerifiedDSN renders the keyword/value DSN and parses it back with pgx: every connection field
// must come out exactly as configured, so no value can smuggle a second dbname, options or sslmode.
func VerifiedDSN(cfg config.DatabaseConfig) (string, error) {
	dsn := cfg.DSN()
	parsed, err := pgx.ParseConfig(dsn)
	if err != nil {
		return "", fmt.Errorf("database: connection parameters do not form a valid DSN: %w", err)
	}
	if parsed.Host != cfg.Host || int(parsed.Port) != cfg.Port || parsed.User != cfg.Username || parsed.Password != cfg.Password || parsed.Database != cfg.Database {
		return "", fmt.Errorf("database: connection parameters changed when parsed back (host, port, user, password or dbname); refusing to connect")
	}
	if len(parsed.RuntimeParams) != 0 {
		return "", fmt.Errorf("database: connection carries runtime settings %v; they come from PGAPPNAME, PGOPTIONS or a PGSERVICE file in this process environment, which the gateway refuses (unset them; docs/OPERATIONS.md, Environments)", parsed.RuntimeParams)
	}
	return dsn, nil
}

// AutoMigrate runs GORM AutoMigrate for all 73 Payminto models, creating or
// updating tables to match the current struct definitions.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// --- Core identity / auth (original) ---
		&models.Member{},
		&models.Role{},
		&models.Permission{},
		&models.ExternalPlatform{},
		&models.MemberExternalPlatformRole{},
		&models.APIKey{},

		// --- Blockchain / chain config (original) ---
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.RPCNode{},
		&models.BlockchainContract{},
		&models.ContractAddress{},
		&models.Currency{},
		&models.BlockchainCurrency{},

		// --- Wallets / addresses (original) ---
		&models.Wallet{},
		&models.AddressPool{},
		&models.DepositAddress{},

		// --- Payments / deposits / sweeps (original) ---
		&models.PaymentRequest{},
		&models.Deposit{},
		&models.Sweep{},
		&models.SweepTransaction{},
		&models.UTXO{},

		// --- Webhooks (original) ---
		&models.Webhook{},
		&models.WebhookDeliveryLog{},

		// --- Double-entry ledger (original) ---
		&models.Account{},
		&models.AccountAddress{},
		&models.Asset{},
		&models.Liability{},
		&models.Revenue{},
		&models.Expense{},

		// --- System / infra — Phase A/B ---
		&models.Configuration{},
		&models.SecretsVault{},
		&models.SecretsVaultActivity{},

		// --- Auth / identity extras — Phase C ---
		&models.AuthRefreshToken{},
		&models.OTP{},
		&models.WebSocketToken{},

		// --- Wallet / address extras — Phase C ---
		&models.WalletXpub{},
		&models.WalletSCW{},
		&models.WalletFunction{},
		&models.AddressContractSignature{},
		&models.AddressDeployment{},
		&models.EntrypointSCAddress{},
		&models.InternalBlockchainTransaction{},
		&models.MissedDeposit{},
		&models.SolanaDepositAccount{},

		// --- Withdrawals / onramp / recipient — Phase C ---
		&models.Withdrawal{},
		&models.Withdraw{},
		&models.OnramperPayments{},
		&models.Recipient{},

		// --- Referral system — Phase C ---
		&models.ReferralCampaign{},
		&models.ReferralEvent{},
		&models.ReferralMember{},
		&models.ReferralReward{},
		&models.ReferralEventLog{},
		&models.ReferralCampaignEvent{},
		&models.ReferralCampaignEventLog{},
		&models.ReferralMemberCampaign{},
		&models.ProcessedReward{},

		// --- Analytics — Phase C ---
		&models.AnalyticsGroup{},
		&models.AnalyticsGraph{},
		&models.AnalyticsFilter{},
		&models.AnalyticsCustomFilter{},
		&models.AnalyticsGroupFilter{},
		&models.AnalyticsUserGroup{},

		// --- Payment channels + EP extras + account reward + system + activity + EEEvent — Phase C ---
		&models.PaymentChannel{},
		&models.DisabledPaymentChannelProject{},
		&models.PaymentsApp{},
		&models.ExternalPlatformBlockchainCurrency{},
		&models.ExternalPlatformWalletBlockchainFamily{},
		&models.AccountReward{},
		&models.SeederLog{},
		&models.GenericDataStore{},
		&models.Tag{},
		&models.ActivityLog{},
		&models.EEEvent{},
	)
}
