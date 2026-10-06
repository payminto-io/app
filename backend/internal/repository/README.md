# repository

Owns the persistence layer: one repository interface and GORM-backed implementation per domain aggregate in `internal/models`. Every repository takes a `*gorm.DB` in its constructor, exposes typed CRUD plus domain-specific queries (e.g. `FindActiveByCode`, `LockForUpdate`, `BulkUpsert`), and accepts variadic `QueryOption` values (`WithLimit`, `WithOffset`, `WithAscendingOrder`, preloads) for composable read queries. `BaseRepository` provides the shared `DB` handle with a `WithDB(tx)` method so services can run multi-repo work inside a single GORM transaction. Depends on `internal/models` and GORM; deliberately has no business logic — that belongs in `internal/service`. Shared test helpers (`openTestDB`, `openFreshDB`, `newTestDB`, `setupBlockchainDB`, `setupRPCNodeDB`, `setupWalletExtrasDB`, `openWebhookTestDB`) back the per-repo tests.

## Files

- `base_repo.go` — `BaseRepository` with `WithDB` transaction binder.
- `option.go` — `QueryOption` type and `WithLimit`/`WithOffset`/`WithAscendingOrder` combinators.
- `utils_repo.go` — shared query utilities.
- `account_repo_impl.go`, `activity_log_repo_impl.go`, `address_deployment_repo_impl.go`, `address_pool_repo_impl.go`, `address_repo_impl.go`, `analytics_repo_impl.go`, `api_key_repo_impl.go`, `auth_refresh_token_repo_impl.go`, `blockchain_currency_repo_impl.go`, `blockchain_family_repo_impl.go`, `blockchain_repo_impl.go`, `configuration_repo_impl.go`, `currency_repo_impl.go`, `deposit_addresses_repo_impl.go`, `deposit_repo_impl.go`, `ee_event_repo_impl.go`, `external_platform_blockchain_currency_repo_impl.go`, `external_platform_repo_impl.go`, `generic_data_store_repo_impl.go`, `internal_blockchain_tx_repo_impl.go`, `member_external_platform_role_repo_impl.go`, `member_repo_impl.go`, `missed_deposit_repo_impl.go`, `onramper_payments_repo_impl.go`, `otp_repo_impl.go`, `payment_repo_impl.go`, `permission_repo_impl.go`, `recipient_repo_impl.go`, `referral_repo_impl.go`, `role_repo_impl.go`, `rpc_node_repo_impl.go`, `secrets_vault_activity_repo_impl.go`, `secrets_vault_repo_impl.go`, `sweep_repo_impl.go`, `sweep_transaction_repo_impl.go`, `utxo_repo_impl.go`, `wallet_function_repo_impl.go`, `wallet_repo_impl.go`, `wallet_scw_repo_impl.go`, `wallet_xpub_repo_impl.go`, `webhook_delivery_log_repo_impl.go`, `webhook_repo_impl.go`, `withdrawal_repo_impl.go` — per-aggregate repositories.
- `identity_repo_test.go` — defines `openTestDB` / `openFreshDB` test helpers reused across identity tests.
- `wallet_repo_test.go` — defines `newTestDB` helper.
- `blockchain_repo_test.go`, `rpc_node_repo_test.go`, `wallet_extras_repo_test.go`, `webhook_repo_test.go` — per-repo setup helpers and tests.
- `account_repo_ledger_test.go`, `base_repo_test.go`, `configuration_repo_test.go`, `option_test.go`, `payment_repo_test.go`, `secrets_vault_repo_test.go`, `utils_repo_test.go`, `utxo_repo_test.go` — additional per-repo tests.

## See also

- `internal/models` — aggregates persisted here
- `internal/service` — business logic that orchestrates these repositories
- `internal/database` — `Connect`/`AutoMigrate` and the integration `NewTestDB`
