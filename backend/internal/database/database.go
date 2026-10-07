package database

import (
	"fmt"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection using the given DatabaseConfig and configures
// connection pool limits.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	logLevel := logger.Info
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
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
