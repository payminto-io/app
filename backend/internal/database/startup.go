package database

import (
	"fmt"
	"slices"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"gorm.io/gorm"
)

// schemaTable is the startup compatibility manifest. AutoMigrate remains the
// canonical model registration used by development/test; this manifest names
// every table it produces and the discriminator columns needed to distinguish
// the current model from the incompatible legacy PayRam layout.
type schemaTable struct {
	name            string
	requiredColumns []string
}

var currentSchemaManifest = []schemaTable{
	{name: "account_addresses"},
	{name: "account_rewards"},
	{name: "accounts", requiredColumns: []string{"currency_id", "member_id", "balance", "locked"}},
	{name: "activity_logs"},
	{name: "address_contract_signatures"},
	{name: "address_deployments"},
	{name: "address_pools", requiredColumns: []string{"encrypted_key", "path_index", "wallet_id"}},
	{name: "analytics_custom_filters"},
	{name: "analytics_filters"},
	{name: "analytics_graphs"},
	{name: "analytics_group_filters"},
	{name: "analytics_groups"},
	{name: "analytics_user_groups"},
	{name: "api_keys", requiredColumns: []string{"key", "external_platform_id"}},
	{name: "assets"},
	{name: "auth_refresh_tokens", requiredColumns: []string{"token_hash"}},
	{name: "blockchain_contracts"},
	{name: "blockchain_currencies", requiredColumns: []string{"blockchain_id", "currency_id"}},
	{name: "blockchain_families"},
	{name: "blockchains", requiredColumns: []string{"code", "family", "min_confirmations"}},
	{name: "configurations", requiredColumns: []string{"key", "value"}},
	{name: "contract_addresses"},
	{name: "currencies", requiredColumns: []string{"code", "wallet_precision"}},
	{name: "deposit_addresses", requiredColumns: []string{"blockchain_currency_id", "payment_request_id"}},
	{name: "deposits", requiredColumns: []string{"tx_id", "blockchain_currency_id", "payment_request_id", "confirmations"}},
	{name: "disabled_payment_channel_projects"},
	{name: "ee_events"},
	{name: "entrypoint_sc_addresses"},
	{name: "expenses"},
	{name: "external_platform_blockchain_currencies"},
	{name: "external_platform_wallet_blockchain_families"},
	{name: "external_platforms", requiredColumns: []string{"name", "success_endpoint"}},
	{name: "generic_data_stores"},
	{name: "internal_blockchain_transactions"},
	{name: "liabilities"},
	{name: "member_external_platform_roles"},
	{name: "member_roles"},
	{name: "members", requiredColumns: []string{"member_type", "password", "state"}},
	{name: "missed_deposits"},
	{name: "onramper_payments"},
	{name: "otps"},
	{name: "payment_channels"},
	{name: "payment_requests", requiredColumns: []string{"reference_id", "amount_in_usd", "state", "expires_at", "external_platform_id"}},
	{name: "payments_apps"},
	{name: "permissions"},
	{name: "processed_rewards"},
	{name: "recipients"},
	{name: "referral_campaign_event_logs"},
	{name: "referral_campaign_events"},
	{name: "referral_campaigns"},
	{name: "referral_event_logs"},
	{name: "referral_events"},
	{name: "referral_member_campaigns"},
	{name: "referral_members"},
	{name: "referral_rewards"},
	{name: "revenues"},
	{name: "role_permissions"},
	{name: "roles"},
	{name: "rpc_nodes", requiredColumns: []string{"blockchain_id", "url", "priority"}},
	{name: "secrets_vault_activities"},
	{name: "secrets_vaults", requiredColumns: []string{"label", "ciphertext", "secret_type"}},
	{name: "seeder_logs"},
	{name: "solana_deposit_accounts", requiredColumns: []string{"deposit_address_id", "owner_address", "token_account", "watch_until"}},
	{name: "solana_sweep_attempts", requiredColumns: []string{"sweep_id", "signature"}},
	{name: "solana_sweep_deposits", requiredColumns: []string{"sweep_id", "deposit_id"}},
	{name: "solana_sweep_locks", requiredColumns: []string{"token_account", "sweep_id"}},
	{name: "sweep_transactions"},
	{name: "sweeps"},
	{name: "tags"},
	{name: "utxos"},
	{name: "wallet_functions"},
	{name: "wallet_scws"},
	{name: "wallet_xpubs"},
	{name: "wallets", requiredColumns: []string{"kind", "blockchain_family_id", "member_id"}},
	{name: "web_socket_tokens"},
	{name: "webhook_delivery_logs"},
	{name: "webhooks", requiredColumns: []string{"url", "secret", "events"}},
	{name: "withdrawals", requiredColumns: []string{"external_platform_id", "to_address", "state"}},
	{name: "withdraws"},
}

var legacySchemaMarkers = []struct {
	table  string
	column string
}{
	{table: "payment_requests", column: "status"},
	{table: "payment_requests", column: "expire_at"},
	{table: "deposits", column: "tx_hash"},
	{table: "deposits", column: "currency_id"},
	{table: "wallets", column: "private_key"},
}

var currentSchemaMarkers = []struct {
	table  string
	column string
}{
	{table: "payment_requests", column: "state"},
	{table: "payment_requests", column: "expires_at"},
	{table: "deposits", column: "tx_id"},
	{table: "wallets", column: "kind"},
}

// PrepareSchema is the runtime-safety seam for schema startup. It may mutate a
// schema only in the closed development/test environments and only when the
// explicit auto-migrate mode is selected. All other environments validate.
func PrepareSchema(db *gorm.DB, environment, mode string) error {
	if db == nil {
		return fmt.Errorf("schema readiness: database is nil")
	}

	environment = strings.ToUpper(strings.TrimSpace(environment))
	if !slices.Contains([]string{
		config.EnvironmentDevelopment,
		config.EnvironmentTest,
		config.EnvironmentStaging,
		config.EnvironmentProduction,
	}, environment) {
		return fmt.Errorf("schema startup: unsupported environment %q", environment)
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == config.SchemaModeAutoMigrate {
		if environment != config.EnvironmentDevelopment && environment != config.EnvironmentTest {
			return fmt.Errorf("schema startup: auto-migrate is limited to development and test")
		}
		if err := AutoMigrate(db); err != nil {
			return fmt.Errorf("schema auto-migrate: %w", err)
		}
		if err := MigrateExpandSchema(db); err != nil {
			return fmt.Errorf("schema auto-migrate: %w", err)
		}
	} else if mode != config.SchemaModeValidate {
		return fmt.Errorf("schema startup: unsupported mode %q", mode)
	}

	if err := validateCurrentSchema(db); err != nil {
		return err
	}
	return validateLedger(db, environment, mode)
}

// validateLedger refuses to boot without the ledger's tables and guarantees; validate mode also
// requires the migration record, and live environments require a role that cannot rewrite history.
func validateLedger(db *gorm.DB, environment, mode string) error {
	if err := ledger.ValidateSchema(db); err != nil {
		return fmt.Errorf("schema readiness: %w", err)
	}
	if mode == config.SchemaModeValidate {
		if err := ledger.ValidateMigrationRecorded(db); err != nil {
			return fmt.Errorf("schema readiness: %w", err)
		}
	}
	if environment == config.EnvironmentStaging || environment == config.EnvironmentProduction {
		if err := ledger.ValidatePrivileges(db); err != nil {
			return fmt.Errorf("schema readiness: %w", err)
		}
	}
	return nil
}

// MigrateExpandSchema creates the migration-managed tables that are deliberately outside the
// manifest (so ApplyMigrations can still run on a database that predates them) for dev/test.
func MigrateExpandSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(&environment.StampRow{}); err != nil {
		return fmt.Errorf("environment: automigrate: %w", err)
	}
	if err := ledger.Migrate(db); err != nil {
		return err
	}
	if err := fees.Migrate(db); err != nil {
		return err
	}
	if err := paymentswitch.Migrate(db); err != nil {
		return err
	}
	return links.Migrate(db)
}

func validateCurrentSchema(db *gorm.DB) error {
	migrator := db.Migrator()
	legacy := countSchemaMarkers(migrator, legacySchemaMarkers)
	current := countSchemaMarkers(migrator, currentSchemaMarkers)
	if legacy > 0 {
		if current > 0 {
			return fmt.Errorf("schema readiness: mixed legacy PayRam and current-model columns detected; refusing to start or mutate this database")
		}
		return fmt.Errorf("schema readiness: legacy PayRam schema detected; run the separately verified legacy-to-current migration before starting this server")
	}

	for _, table := range currentSchemaManifest {
		if !migrator.HasTable(table.name) {
			return fmt.Errorf("schema readiness: required table %q is missing; schema is unknown or incomplete and must be migrated explicitly", table.name)
		}
		for _, column := range table.requiredColumns {
			if !migrator.HasColumn(table.name, column) {
				return fmt.Errorf("schema readiness: required column %q.%q is missing; database is not compatible with the current server", table.name, column)
			}
		}
	}
	return nil
}

func countSchemaMarkers(migrator gorm.Migrator, markers []struct {
	table  string
	column string
}) int {
	count := 0
	for _, marker := range markers {
		if migrator.HasTable(marker.table) && migrator.HasColumn(marker.table, marker.column) {
			count++
		}
	}
	return count
}

func currentSchemaContainsTable(name string) bool {
	return slices.ContainsFunc(currentSchemaManifest, func(table schemaTable) bool {
		return table.name == name
	})
}
