# service

Owns all business logic for the Payminto backend. Each file is one cohesive service (`AuthService`, `PaymentService`, `DepositService`, `WithdrawalService`, `SweepService`, `LedgerService`, `WebhookService`, `MemberService`, `RoleService`, `ReferralService`, `SecretsVaultService`, `ConfigurationService`, `AnalyticsService`, `TickerService`, `OnrampService`, etc.) plus the central `ServiceRegistry` (in `registry.go`) that acts as the DI container: it owns `*gorm.DB`, Redis, `*config.Config`, the `AdapterRegistry`, and constructs every repository and service in dependency order for `cmd/server` and the tests. Services depend on `internal/repository`, `internal/blockchain`, `internal/crypto`, `internal/email`, and `internal/models`; they never touch HTTP or GORM directly in callers. `config_adapter.go` adapts external config shapes into service-friendly values, and `registry_test.go` exposes the shared `newTestRegistry` helper used by every service test to boot an in-memory registry.

## Files

- `registry.go` — `ServiceRegistry` DI container wiring every repo and service.
- `config_adapter.go` — adapter layer between `internal/config` and service constructors.
- `auth_service.go`, `jwt_token_service.go`, `otp_service.go`, `nonce_service.go` — authentication, JWT issuance, OTP, and nonce generation.
- `member_service.go`, `role_service.go`, `permission_service.go`, `mep_role_service.go` (`member_external_platform_role_service.go`) — RBAC and tenant membership.
- `external_platform_service.go`, `external_platform_blockchain_currency_service.go`, `configuration_service.go`, `system_service.go` — tenant and system configuration.
- `payment_service.go`, `deposit_service.go`, `deposit_address_service.go`, `address_service.go`, `address_pool_service.go`, `address_blacklist_service.go` — payment and deposit lifecycle.
- `sweep_service.go`, `sweep_transaction_service.go`, `sweep_utxo_service.go`, `utxo_service.go`, `internal_blockchain_tx_service.go` — sweep engine and on-chain settlement.
- `withdrawal_service.go`, `withdrawal_processing_service.go`, `recipient_service.go` — withdrawal flows and recipients.
- `ledger_service.go`, `account_reward_service.go` — double-entry ledger.
- `webhook_service.go`, `webhook_management_service.go`, `event_emitter_service.go` — webhook delivery and event emission.
- `referral_service.go`, `referral_campaign_service.go`, `analytics_referral_service.go`, `analytics_service.go`, `ticker_service.go` — referrals, analytics, and price tickers.
- `onramp_service.go`, `onramper_payments_service.go` — fiat-to-crypto onramp.
- `email_service.go` — template rendering and email dispatch.
- `missed_deposit_service.go` — missed-deposit reconciliation.
- `secrets_vault_service.go`, `wallet_service.go`, `generic_data_store_service.go` — secrets vault, wallet ops, and generic key/value storage.
- `registry_test.go` — `newTestRegistry(t)` shared test helper.
- `*_test.go` — per-service unit tests; `onramper_fuzz_test.go` is a fuzz target; `ledger_referral_payout_test.go` is a ledger integration test.

## See also

- `internal/repository` — persistence layer consumed here
- `internal/blockchain` — chain adapter registry
- `internal/api/handler` — HTTP layer that calls these services
- `internal/worker` — background processors that also use them
