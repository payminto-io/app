# models

Owns the GORM domain models for every Payminto table — ~73 structs that mirror the PayRam schema across identity/auth, blockchain/currency, wallets, payments, sweeps, withdrawals, webhooks, analytics, referrals, and the double-entry ledger. Exposes `PaymintoModel` and `BaseModel` (shared embedded base types providing `ID`, audit timestamps, and soft-delete) together with one file per domain aggregate (e.g. `Member`, `Role`, `Blockchain`, `Wallet`, `PaymentRequest`, `Sweep`, `Withdrawal`, `Webhook`, `Account`, `ReferralCampaign`, `SecretsVault`). Depends only on `gorm.io/gorm`, `shopspring/decimal`, and the Go standard library — it must stay free of service, repository, or HTTP imports so every other package can consume it without cycles.

## Files

- `base.go` — `PaymintoModel` and `BaseModel` embedded base types.
- `member.go`, `role.go`, `api_key.go`, `auth_token.go`, `otp.go`, `websocket_token.go` — identity, auth, and session tables.
- `blockchain.go`, `currency.go`, `payment_channel.go`, `rpc_node.go` — chain registry and supported currencies.
- `wallet.go`, `wallet_xpub.go`, `wallet_scw.go`, `wallet_function.go`, `address_deployment.go`, `address_contract_signature.go`, `entrypoint_sc_addr.go` — HD and smart-contract wallet tables.
- `payment.go`, `onramper_payments.go`, `recipient.go` — payment-request and onramp lifecycle.
- `sweep.go`, `internal_blockchain_tx.go`, `missed_deposit.go`, `withdrawal.go` — sweep, settlement, and withdrawal flows.
- `account.go`, `account_reward.go` — double-entry ledger accounts.
- `webhook.go`, `activity_log.go`, `ee_event.go` — webhooks, audit log, event emitter.
- `external_platform.go`, `external_platform_extras.go`, `system.go`, `system_extras.go` — tenant and system configuration.
- `referral.go` — referral campaigns, codes, and payouts.
- `secrets_vault.go` — at-rest encrypted secrets.
- `analytics.go` — analytics rollup tables.
- `*_test.go` — GORM tag and serialization tests (`base_test.go`, `payment_test.go`, `member_test.go`, etc.).

## See also

- `internal/repository` — persistence layer over these models
- `internal/database/database.go` — `AutoMigrate` list
- `payram_schema_raw.sql` — authoritative schema source
