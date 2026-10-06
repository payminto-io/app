# PayRam Database Schema - Complete Documentation

**Source:** Extracted from running PayRam Docker container (`payramapp/payram:latest`)
**Database:** PostgreSQL 14
**Extraction Date:** April 6, 2026
**Total Tables:** 74 | **Total Foreign Keys:** 125+ | **Total Migrations:** 86

---

## Table of Contents

1. [Schema Overview](#1-schema-overview)
2. [Entity Relationship Summary](#2-entity-relationship-summary)
3. [Domain Groups](#3-domain-groups)
4. [Table Definitions](#4-table-definitions)
   - [Core Identity & Auth](#41-core-identity--auth)
   - [Blockchain Infrastructure](#42-blockchain-infrastructure)
   - [Wallet & Key Management](#43-wallet--key-management)
   - [Payment Processing](#44-payment-processing)
   - [Sweep & Settlement](#45-sweep--settlement)
   - [Withdrawal & Payout](#46-withdrawal--payout)
   - [Accounting (Double-Entry Ledger)](#47-accounting-double-entry-ledger)
   - [Webhooks & Events](#48-webhooks--events)
   - [Analytics & Dashboard](#49-analytics--dashboard)
   - [Referral System](#410-referral-system)
   - [System & Configuration](#411-system--configuration)
5. [Foreign Key Map](#5-foreign-key-map)
6. [Index Summary](#6-index-summary)
7. [Key Design Patterns](#7-key-design-patterns)
8. [SQL Recreation Script](#8-sql-recreation-script)

---

## 1. Schema Overview

PayRam uses a **74-table PostgreSQL schema** built with GORM (Go ORM) and managed through 86 sequential migrations. The schema uses:

- **bigint auto-increment IDs** (not UUIDs) as primary keys
- **Soft deletes** via `deleted_at` timestamp (GORM convention)
- **`created_at` / `updated_at` / `deleted_at`** on nearly every table
- **`numeric(38,18)`** for all monetary amounts (18 decimal places for crypto precision)
- **`character varying`** with explicit max lengths for all string fields
- **Foreign keys** with GORM naming convention (`fk_<parent>_<child>`)
- **Indexes** on `deleted_at`, `created_at`, and all FK columns

---

## 2. Entity Relationship Summary

```
members (Users/Merchants)
 ├── accounts (Balance per currency)
 ├── account_addresses (Balance per blockchain address)
 ├── account_rewards (Reward balances)
 ├── deposit_addresses (Assigned deposit addresses)
 ├── wallets (HD/SC wallets)
 │    ├── address_pools (Pre-generated addresses)
 │    └── wallet_functions (Wallet capabilities)
 ├── secrets_vaults (Encrypted keys)
 ├── api_keys (API authentication)
 ├── member_roles (Role assignments)
 ├── member_external_platforms (Project memberships)
 │    └── member_external_platform_roles
 ├── web_socket_tokens (WebSocket auth)
 └── withdrawals (Payout requests)

external_platforms (Projects/Merchants)
 ├── api_keys
 ├── webhooks → webhook_delivery_logs
 ├── payment_requests
 ├── withdrawals
 └── external_platform_blockchain_currencies

blockchains
 ├── blockchain_currencies (Token per chain)
 ├── blockchain_contracts (Smart contracts)
 │    └── contract_addresses (Deployed instances)
 │         └── address_contract_signatures
 └── blockchain_families (EVM, BTC, TRX groups)

deposits (Incoming crypto)
 ├── internal_blockchain_transactions
 ├── utxos (Bitcoin UTXOs)
 └── payment_requests (links deposit to payment)

sweeps / sweep_transactions (Cold storage transfers)
 ├── account_addresses (locked during sweep)
 └── utxos (consumed UTXOs)
```

---

## 3. Domain Groups

| Group | Tables | Purpose |
|-------|--------|---------|
| **Core Identity & Auth** | `members`, `roles`, `permissions`, `role_permissions`, `member_roles`, `api_keys`, `otps`, `web_socket_tokens` | User management, RBAC, authentication |
| **Blockchain Infrastructure** | `blockchain_families`, `blockchains`, `currencies`, `blockchain_currencies`, `blockchain_contracts`, `contract_addresses` | Chain/token/contract configuration |
| **Wallet & Key Management** | `wallets`, `wallet_functions`, `secrets_vaults`, `address_pools`, `address_contract_signatures`, `deposit_addresses` | HD wallets, key storage, address generation |
| **Payment Processing** | `payment_requests`, `deposits`, `internal_blockchain_transactions` | Payment lifecycle |
| **Sweep & Settlement** | `sweeps`, `sweep_transactions`, `utxos`, `withdraw_deposits_btcs` | Fund consolidation to cold storage |
| **Withdrawal & Payout** | `withdrawals`, `withdraws` | Outbound crypto transfers |
| **Accounting** | `accounts`, `account_addresses`, `account_rewards`, `assets`, `liabilities`, `revenues`, `expenses` | Double-entry bookkeeping |
| **Webhooks & Events** | `webhooks`, `webhook_delivery_logs`, `ee_events` | Event notification system |
| **Multi-tenant / Projects** | `external_platforms`, `member_external_platforms`, `member_external_platform_roles`, `external_platform_blockchain_currencies` | Multi-project support |
| **Analytics** | `analytics_groups`, `analytics_graphs`, `analytics_filters`, `analytics_custom_filters`, `analytics_group_filters`, `analytics_user_groups` | Dashboard analytics engine |
| **Referral System** | `referral_campaigns`, `referral_events`, `referral_members`, `referral_rewards`, `referral_event_logs`, `referral_campaign_events`, `referral_campaign_event_logs`, `referral_member_campaigns`, `processed_rewards` | Referral & rewards program |
| **Payment Channels** | `payment_channels`, `disabled_payment_channel_projects`, `payments_apps` | Onramp/fiat payment channels |
| **RPC & Infrastructure** | `rpc_nodes`, `entrypoint_sc_addresses`, `address_deployments` | Node management, contract deployment |
| **Recipients** | `recipients` | Payout recipient address book |
| **Auth** | `auth_refresh_tokens` | JWT refresh token management |
| **Wallet Extensions** | `wallet_scws`, `wallet_xpubs`, `external_platform_wallet_blockchain_families` | Smart contract wallets, BIP32 xpubs |
| **Activity** | `activity_logs` | API request audit trail |
| **Missed Deposits** | `missed_deposits` | Deposits detected but not matched |
| **System** | `configurations`, `migrations`, `seeder_logs`, `generic_data_stores`, `tags` | Config, migrations, misc storage |

---

## 4. Table Definitions

### 4.1 Core Identity & Auth

#### `members`
Central user/merchant table. All major entities reference this.

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | bigint | NO | auto | Primary key |
| `created_at` | timestamptz | YES | | |
| `updated_at` | timestamptz | YES | | |
| `deleted_at` | timestamptz | YES | | Soft delete |
| `name` | varchar(100) | NO | | Display name |
| `email` | text | YES | | Email address |
| `customer_id` | text | YES | | External customer ID |
| `level` | bigint | NO | 0 | Member level |
| `group` | varchar(20) | NO | 'vip-0' | Member group tier |
| `state` | varchar(20) | NO | 'active' | active/inactive/banned |
| `username` | varchar(20) | YES | | Login username |
| `password` | text | YES | | Hashed password |
| `reset_password_required` | boolean | NO | false | Force password reset |
| `reset_password_token` | text | YES | | Password reset token |
| `reset_password_expiry` | timestamptz | YES | | Token expiry |
| `member_type` | varchar(20) | NO | 'customer' | customer/admin/system |
| `referred_by_member_id` | bigint | YES | | FK → members.id |

**FK:** `referred_by_member_id` → `members(id)` (self-referencing)

#### `roles`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `name` | varchar(200) | NO | |
| `display_name` | varchar(255) | NO | |
| `description` | text | YES | |

#### `permissions`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `name` | varchar(200) | NO | |
| `description` | text | YES | |
| `display_name` | varchar(255) | NO | |

#### `role_permissions` (join table)
| Column | Type | Nullable |
|--------|------|----------|
| `role_id` | bigint | NO |
| `permission_id` | bigint | NO |

**PK:** (role_id, permission_id)

#### `member_roles` (join table)
| Column | Type | Nullable |
|--------|------|----------|
| `role_id` | bigint | NO |
| `member_id` | bigint | NO |

**PK:** (member_id, role_id)

#### `api_keys`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `key` | varchar(255) | NO | |
| `status` | varchar(20) | NO | 'active' |
| `member_id` | bigint | YES | |
| `external_platform_id` | bigint | NO | |
| `role_id` | bigint | YES | |
| `expire_at` | timestamptz | YES | |
| `description` | text | YES | |

**FK:** `member_id` → `members(id)`, `external_platform_id` → `external_platforms(id)`, `role_id` → `roles(id)`

#### `otps`
One-time passwords for 2FA/email verification.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `entity_id` | bigint | NO | |
| `purpose` | varchar(50) | NO | |
| `code` | varchar(20) | NO | |
| `expires_at` | timestamptz | NO | |
| `verified` | boolean | YES | false |
| `verified_at` | timestamptz | YES | |
| `attempts` | bigint | YES | 0 |
| `last_attempt_at` | timestamptz | YES | |
| `event_metadata` | json | YES | |

#### `web_socket_tokens`
| Column | Type | Nullable |
|--------|------|----------|
| `id` | bigint | NO |
| `token` | varchar(255) | NO |
| `server_url` | varchar(255) | NO |
| `expiry` | timestamptz | NO |
| `client_id` | varchar(255) | NO |
| `member_id` | bigint | NO |

**FK:** `member_id` → `members(id)`

---

### 4.2 Blockchain Infrastructure

#### `blockchain_families`
Groups blockchains by derivation type (EVM, Bitcoin, Tron).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `family` | varchar(30) | NO | |
| `path` | varchar(255) | YES | |
| `gas_path` | varchar(255) | YES | |
| `supports_hd_wallet` | boolean | NO | false |
| `supports_sc_wallet` | boolean | NO | false |

#### `blockchains`
Individual blockchain networks (Ethereum, Base, Polygon, Bitcoin, Tron).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `code` | varchar(20) | NO | |
| `name` | varchar(150) | NO | |
| `family` | varchar(30) | NO | |
| `client` | varchar(200) | YES | |
| `height` | bigint | YES | |
| `height_timestamp` | timestamptz | YES | |
| `explorer_address` | varchar(200) | YES | |
| `explorer_transaction` | varchar(200) | YES | |
| `min_confirmations` | bigint | NO | 1 |
| `chain_id` | bigint | YES | |
| `status` | varchar(20) | NO | 'active' |
| `is_scw` | boolean | NO | false |
| `blockchain_family_id` | bigint | NO | |
| `is_create2_supported` | boolean | YES | false |

**FK:** `blockchain_family_id` → `blockchain_families(id)`

#### `currencies`
Tokens/coins supported by the system.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `name` | varchar(10) | NO | |
| `code` | varchar(10) | NO | |
| `description` | text | YES | |
| `homepage` | text | YES | |
| `type` | varchar(30) | NO | 'coin' |
| `visible` | boolean | NO | true |
| `deposit_enabled` | boolean | NO | true |
| `withdrawal_enabled` | boolean | NO | true |
| `wallet_precision` | bigint | NO | 6 |
| `base_precision` | bigint | NO | 8 |
| `icon_url` | text | YES | |
| `price` | numeric(38,18) | YES | |

#### `blockchain_currencies`
Maps currencies to specific blockchains (e.g., USDC on Ethereum, USDC on Base).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `address` | varchar(150) | NO | |
| `standard` | varchar(30) | NO | |
| `currency_code` | varchar(20) | NO | |
| `blockchain_code` | varchar(20) | NO | |
| `deposit_fee` | numeric(38,18) | YES | |
| `min_deposit_amount` | numeric(38,18) | YES | |
| `min_collection_amount` | numeric(38,18) | YES | |
| `withdraw_fee` | numeric(38,18) | YES | |
| `min_withdraw_amount` | numeric(38,18) | YES | |
| `withdraw_limit24hr` | numeric(38,18) | YES | |
| `withdraw_limit72hr` | numeric(38,18) | YES | |
| `visible` | boolean | NO | true |
| `deposit_enabled` | boolean | NO | true |
| `withdrawal_enabled` | boolean | NO | false |
| `wallet_precision` | bigint | NO | 6 |
| `abi` | text | YES | |
| `approval_fee_amount` | numeric(38,18) | YES | |
| `min_balance_for_fees_transfer` | numeric(38,18) | YES | |
| `min_balance_for_sweep` | numeric(38,18) | YES | |
| `sweep_address_locked_duration_in_minutes` | bigint | YES | |
| `sweep_batch_size` | bigint | YES | |
| `sweep_approval_batch_size` | bigint | YES | |
| `sweep_max_wait_time_in_minutes` | bigint | YES | |
| `min_amount_in_sweep_tx` | numeric(38,18) | YES | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |

**FK:** `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`

#### `blockchain_contracts`
Smart contract definitions (ABI + bytecode).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `blockchain_code` | varchar(20) | NO | |
| `contract_type` | varchar(100) | NO | |
| `abi` | text | YES | |
| `bytecode` | text | YES | |
| `address` | varchar(200) | YES | |
| `description` | text | YES | |
| `status` | varchar(20) | NO | 'active' |
| `blockchain_id` | bigint | NO | |
| `currency_standard` | varchar(30) | NO | |

**FK:** `blockchain_id` → `blockchains(id)`

#### `contract_addresses`
Deployed contract instances.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `blockchain_code` | varchar(20) | NO | |
| `name` | varchar(100) | YES | |
| `address` | varchar(200) | YES | |
| `status` | varchar(20) | NO | 'active' |
| `wallet_address` | varchar(200) | YES | |
| `transaction_hash` | text | YES | |
| `currency_standard` | varchar(30) | NO | |
| `blockchain_contract_type` | varchar(100) | NO | |
| `blockchain_contract_id` | bigint | NO | |
| `attributes` | text | YES | |
| `abi` | text | YES | |

**FK:** `blockchain_contract_id` → `blockchain_contracts(id)`

---

### 4.3 Wallet & Key Management

#### `wallets`
HD wallets and smart contract wallets per member.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `name` | varchar(255) | NO | |
| `family` | varchar(30) | YES | |
| `blockchain_code` | varchar(20) | YES | |
| `currency_code` | varchar(20) | YES | |
| `secret_type` | varchar(255) | NO | |
| `wallet_type` | varchar(255) | NO | |
| `wallet_sub_type` | varchar(255) | NO | 'eoa' |
| `private_key` | text | YES | |
| `public_key` | text | YES | |
| `address` | text | YES | |
| `member_id` | bigint | NO | |
| `status` | varchar(255) | NO | 'active' |
| `blockchain_family_id` | bigint | YES | |
| `blockchain_id` | bigint | YES | |
| `currency_id` | bigint | YES | |
| `default` | boolean | NO | false |
| `abi` | text | YES | |
| `bytecode` | text | YES | |
| `fund_sweeper_address` | text | YES | |
| `salt` | text | YES | |
| `master_address` | text | YES | |
| `private_key_hash` | char(64) | YES | |
| `address_hash` | char(64) | YES | |
| `fund_sweeper_address_hash` | char(64) | YES | |

**FK:** `member_id` → `members(id)`, `blockchain_family_id` → `blockchain_families(id)`, `blockchain_id` → `blockchains(id)`, `currency_id` → `currencies(id)`

#### `wallet_functions`
Capabilities/activities associated with a wallet.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `activity_type` | varchar(255) | NO | |
| `display_name` | varchar(255) | NO | |
| `description` | text | YES | |
| `secret_vault_id` | bigint | YES | |
| `wallet_id` | bigint | YES | |

**FK:** `secret_vault_id` → `secrets_vaults(id)`, `wallet_id` → `wallets(id)`

#### `secrets_vaults`
Encrypted key storage (AES-256 encrypted private keys).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `name` | varchar(255) | NO | |
| `blockchain_code` | varchar(20) | YES | |
| `currency_code` | varchar(20) | YES | |
| `secret_type` | varchar(255) | NO | |
| `secret_data` | text | NO | |
| `public_key` | text | YES | |
| `member_id` | bigint | NO | |
| `encryption_scheme` | varchar(255) | YES | |
| `status` | varchar(255) | YES | 'active' |

**FK:** `member_id` → `members(id)`

#### `address_pools`
Pre-generated blockchain addresses for fast assignment.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `address` | varchar(200) | NO | **UNIQUE** |
| `private_key` | text | YES | |
| `family` | varchar(30) | NO | |
| `blockchain_family_id` | bigint | NO | |
| `path` | varchar(100) | YES | |
| `path_index` | bigint | NO | 0 |
| `status` | varchar(20) | NO | 'open' |
| `wallet_id` | bigint | YES | |
| `salt` | text | YES | |
| `type` | varchar(20) | NO | 'eoa' |
| `salt_counter` | bigint | YES | |

**FK:** `blockchain_family_id` → `blockchain_families(id)`, `wallet_id` → `wallets(id)`

#### `address_contract_signatures`
Smart contract signatures for deposit addresses (SmartSweep approvals).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `address` | varchar(200) | NO | |
| `signature` | text | NO | **UNIQUE** |
| `address_path_index` | bigint | NO | |
| `family` | varchar(30) | NO | |
| `contract_addr` | varchar(200) | YES | |
| `blockchain_code` | varchar(20) | NO | |
| `currency_code` | varchar(20) | NO | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |
| `status` | varchar(20) | NO | 'pending' |
| `blockchain_contract_type` | varchar(100) | NO | |
| `blockchain_contract_id` | bigint | NO | |
| `contract_address_id` | bigint | NO | |

**PK:** (id, address, contract_address_id, blockchain_contract_id)
**FK:** `blockchain_contract_id` → `blockchain_contracts(id)`, `contract_address_id` → `contract_addresses(id)`

#### `deposit_addresses`
Addresses assigned to specific members for receiving payments.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `address` | varchar(200) | NO | |
| `address_lower` | varchar(200) | NO | '' |
| `family` | varchar(30) | NO | |
| `status` | varchar(30) | NO | 'active' |
| `member_id` | bigint | NO | |
| `wallet_id` | bigint | YES | |
| `blockchain_family_id` | bigint | NO | |

**FK:** `member_id` → `members(id)`, `wallet_id` → `wallets(id)`, `blockchain_family_id` → `blockchain_families(id)`

---

### 4.4 Payment Processing

#### `payment_requests`
Core payment table — each payment link/invoice creates one record.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `amount` | numeric(38,18) | YES | |
| `amount_in_usd` | numeric(38,18) | NO | |
| `filled_amount` | numeric(38,18) | YES | |
| `filled_amount_in_usd` | numeric(38,18) | YES | |
| `sponsored_amount` | numeric(38,18) | NO | 0 |
| `sponsored_amount_in_usd` | numeric(38,18) | NO | 0 |
| `currency_code` | varchar(20) | YES | |
| `blockchain_code` | varchar(20) | YES | |
| `invoice_id` | varchar(100) | NO | |
| `reference_id` | varchar(100) | NO | |
| `status` | varchar(20) | NO | 'open' |
| `expire_at` | timestamptz | YES | |
| `member_id` | bigint | NO | |
| `currency_id` | bigint | YES | |
| `blockchain_id` | bigint | YES | |
| `deposit_id` | bigint | YES | |
| `created_by` | varchar(20) | NO | 'user' |
| `external_platform_id` | bigint | NO | |

**Status values:** open → confirming → confirmed / expired
**FK:** `member_id` → `members(id)`, `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`, `deposit_id` → `deposits(id)`, `external_platform_id` → `external_platforms(id)`

#### `deposits`
On-chain incoming transactions detected by block monitors.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `member_id` | bigint | NO | |
| `currency_code` | varchar(20) | NO | |
| `blockchain_code` | varchar(20) | NO | |
| `amount` | numeric(38,18) | NO | |
| `price_in_usd` | numeric(38,18) | YES | |
| `amount_in_usd` | numeric(38,18) | YES | |
| `fee` | numeric(38,18) | YES | |
| `address` | varchar(200) | YES | |
| `from_addresses` | varchar(200) | YES | |
| `tx_hash` | varchar(200) | NO | |
| `unique_tx_hash` | varchar(250) | NO | |
| `block_hash` | varchar(200) | YES | |
| `status` | varchar(20) | NO | 'pending' |
| `block_number` | bigint | NO | |
| `type` | varchar(20) | NO | 'coin' |
| `timestamp` | timestamptz | YES | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |
| `metadata` | jsonb | YES | |
| `origin` | varchar(200) | YES | |

**Status values:** pending → accepted / skipped / rejected
**FK:** `member_id` → `members(id)`, `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`

#### `internal_blockchain_transactions`
Internal record of all blockchain transactions (gas transfers, sweeps, etc.).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `member_id` | bigint | YES | |
| `blockchain_code` | varchar(20) | NO | |
| `currency_code` | varchar(20) | YES | |
| `amount` | numeric(38,18) | YES | |
| `price_in_usd` | numeric(38,18) | YES | |
| `amount_in_usd` | numeric(38,18) | YES | |
| `fee` | numeric(38,18) | YES | |
| `from_address` | varchar(200) | YES | |
| `to_address` | varchar(200) | YES | |
| `tx_hash` | varchar(200) | NO | |
| `unique_tx_hash` | varchar(250) | NO | |
| `block_hash` | varchar(200) | YES | |
| `status` | varchar(20) | NO | 'sent' |
| `block_number` | bigint | YES | |
| `type` | varchar(20) | NO | 'coin' |
| `transaction_type` | varchar(20) | NO | 'coin' |
| `timestamp` | timestamptz | YES | |
| `blockchain_id` | bigint | NO | |
| `currency_id` | bigint | YES | |
| `deposit_id` | bigint | YES | |

**FK:** `member_id` → `members(id)`, `blockchain_id` → `blockchains(id)`, `currency_id` → `currencies(id)`, `deposit_id` → `deposits(id)`

---

### 4.5 Sweep & Settlement

#### `sweep_transactions`
Batched sweep operations from deposit addresses to cold storage.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `blockchain_code` | varchar(20) | NO | |
| `currency_code` | varchar(20) | NO | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |
| `status` | varchar(20) | NO | 'pending' |
| `tx_hash` | varchar(200) | YES | |
| `signed_payload` | text | YES | |
| `raw_payload` | text | YES | |
| `payload_type` | varchar(20) | NO | |
| `wallet_id` | bigint | NO | |
| `fee_percentage` | numeric(10,6) | YES | |

**FK:** `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`, `wallet_id` → `wallets(id)`

#### `sweeps`
Individual on-chain sweep transactions (confirmed on blockchain).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `currency_code` | varchar(20) | NO | |
| `blockchain_code` | varchar(20) | NO | |
| `amount` | numeric(38,18) | YES | |
| `price_in_usd` | numeric(38,18) | YES | |
| `amount_in_usd` | numeric(38,18) | YES | |
| `fee` | numeric(38,18) | YES | |
| `from_address` | varchar(200) | YES | |
| `to_address` | varchar(200) | YES | |
| `contract_address` | varchar(200) | YES | |
| `token_address` | varchar(200) | YES | |
| `tx_hash` | varchar(200) | NO | |
| `unique_tx_hash` | varchar(250) | NO | |
| `tx_index` | bigint | YES | |
| `block_hash` | varchar(200) | YES | |
| `status` | varchar(200) | NO | 'pending' |
| `block_number` | bigint | NO | |
| `type` | varchar(20) | YES | 'coin' |
| `timestamp` | timestamptz | YES | |
| `attributes` | text | YES | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |

**FK:** `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`

#### `utxos`
Bitcoin UTXO tracking for BTC sweeps.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `deposit_id` | bigint | NO | |
| `withdraw_id` | bigint | YES | |
| `amount` | numeric(38,18) | NO | |
| `address` | varchar(200) | YES | |
| `tx_hash` | varchar(200) | NO | |
| `unique_tx_hash` | varchar(250) | NO | |
| `index` | bigint | NO | |
| `script_pub_key_hex` | varchar(300) | NO | |
| `raw_transaction_hex` | text | NO | |
| `status` | varchar(20) | NO | 'unspent' |
| `currency_code` | varchar(20) | NO | |
| `blockchain_code` | varchar(20) | NO | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |
| `sweeping_transaction_id` | bigint | YES | |
| `sweep_id` | bigint | YES | |

**FK:** `deposit_id` → `deposits(id)`, `address` → `address_pools(address)`, `sweeping_transaction_id` → `sweep_transactions(id)`, `sweep_id` → `sweeps(id)`

#### `withdraw_deposits_btcs`
Bitcoin-specific withdrawal UTXO tracking.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `deposit_id` | bigint | NO | |
| `withdraw_id` | bigint | YES | |
| `amount` | numeric(38,18) | NO | |
| `address` | varchar(200) | YES | |
| `tx_hash` | varchar(200) | NO | |
| `unique_tx_hash` | varchar(250) | NO | |
| `index` | bigint | NO | |
| `script_pub_key_hex` | varchar(300) | NO | |
| `raw_transaction_hex` | text | NO | |
| `status` | varchar(20) | NO | 'unspent' |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |

---

### 4.6 Withdrawal & Payout

#### `withdrawals`
Merchant-initiated outbound crypto transfers (payouts).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `currency_code` | varchar(200) | NO | |
| `blockchain_code` | varchar(200) | NO | |
| `amount` | numeric(38,18) | NO | |
| `price_in_usd` | numeric(38,18) | YES | |
| `amount_in_usd` | numeric(38,18) | YES | |
| `fee` | numeric(38,18) | YES | |
| `from_address` | varchar(200) | YES | |
| `to_address` | varchar(200) | NO | |
| `recipient_email` | varchar(255) | YES | |
| `token_address` | varchar(200) | YES | |
| `tx_hash` | varchar(200) | YES | |
| `unique_tx_hash` | varchar(250) | YES | |
| `tx_index` | bigint | YES | |
| `block_hash` | varchar(200) | YES | |
| `status` | varchar(200) | NO | 'pending' |
| `block_number` | bigint | YES | |
| `type` | varchar(20) | NO | |
| `currency_type` | varchar(200) | NO | 'coin' |
| `timestamp` | timestamptz | YES | |
| `attributes` | text | YES | |
| `signed_tx` | text | YES | |
| `failure_reason` | text | YES | |
| `webhook_status` | text | YES | |
| `retry_count` | bigint | NO | 0 |
| `last_retry_at` | timestamptz | YES | |
| `member_id` | bigint | NO | |
| `created_by_member_id` | bigint | YES | |
| `created_by_project_id` | bigint | YES | |
| `rejected_by_member_id` | bigint | YES | |
| `approved_by_member_id` | bigint | YES | |
| `external_platform_id` | bigint | YES | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |
| `created_by` | varchar(50) | NO | 'system' |

**Status values:** pending → approved → processing → completed / failed / rejected
**FK:** `member_id` → `members(id)`, many other FKs for approval workflow

#### `withdraws`
Bitcoin-specific withdrawal transactions (BTC has special UTXO handling).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `tx_hash` | varchar(200) | NO | |
| `currency_code` | varchar(20) | NO | |
| `block_number` | bigint | YES | |
| `status` | varchar(20) | NO | 'initiated' |
| `amount` | numeric(38,18) | NO | |
| `price_in_usd` | numeric(38,18) | YES | |
| `amount_in_usd` | numeric(38,18) | YES | |
| `amount_in_change_address` | numeric(38,18) | YES | |
| `fee` | numeric(38,18) | YES | |
| `address` | varchar(200) | YES | |
| `amount_to_payram` | numeric(38,18) | YES | |
| `payram_address` | varchar(200) | YES | |
| `block_hash` | varchar(200) | YES | |
| `timestamp` | timestamptz | YES | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |

**FK:** `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`

---

### 4.7 Accounting (Double-Entry Ledger)

PayRam uses a **double-entry accounting system** with four ledger tables that share identical structure.

#### `accounts`
Member balance per currency.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `member_id` | bigint | YES | |
| `currency_id` | bigint | YES | |
| `currency_code` | varchar(20) | NO | |
| `balance` | numeric(38,18) | NO | 0.0 |
| `locked` | numeric(38,18) | NO | 0.0 |
| `status` | varchar(20) | NO | 'active' |

**FK:** `member_id` → `members(id)`, `currency_id` → `currencies(id)`

#### `account_addresses`
Balance tracking per blockchain deposit address.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `address` | varchar(200) | NO | |
| `balance` | numeric(38,18) | NO | 0.0 |
| `locked` | numeric(38,18) | NO | 0.0 |
| `deposit_fee` | numeric(38,18) | NO | 0 |
| `approval_fee` | numeric(38,18) | NO | 0 |
| `member_id` | bigint | NO | |
| `status` | varchar(20) | NO | 'active' |
| `blockchain_code` | varchar(20) | NO | |
| `currency_code` | varchar(20) | NO | |
| `currency_id` | bigint | NO | |
| `blockchain_id` | bigint | NO | |
| `sweeping_transaction_id` | bigint | YES | |

**FK:** `member_id` → `members(id)`, `currency_id` → `currencies(id)`, `blockchain_id` → `blockchains(id)`, `address` → `address_pools(address)`, `sweeping_transaction_id` → `sweep_transactions(id)`

#### `account_rewards`
Reward balances per member per currency.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `member_id` | bigint | NO | |
| `currency_id` | bigint | NO | |
| `currency_code` | varchar(20) | NO | |
| `balance` | numeric(38,18) | NO | 0.0 |
| `locked` | numeric(38,18) | NO | 0.0 |
| `status` | varchar(20) | NO | 'active' |

**PK:** (id, member_id, currency_id)

#### Ledger tables: `assets`, `liabilities`, `revenues`, `expenses`
All four share identical structure (double-entry bookkeeping):

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `member_id` | bigint | YES | |
| `account_id` | bigint | YES | |
| `currency_id` | bigint | NO | |
| `debit` | numeric(38,18) | NO | 0.0 |
| `credit` | numeric(38,18) | NO | 0.0 |
| `currency_code` | varchar(20) | NO | |
| `reference_type` | text | YES | |
| `reference_id` | bigint | YES | |
| `code` | bigint | NO | |

**FK (all four):** `member_id` → `members(id)`, `account_id` → `accounts(id)`, `currency_id` → `currencies(id)`

---

### 4.8 Webhooks & Events

#### `webhooks`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `external_platform_id` | bigint | NO | |
| `name` | varchar(200) | NO | |
| `url` | varchar(500) | NO | |
| `access_key` | varchar(255) | NO | |
| `status` | varchar(20) | NO | 'active' |
| `tested` | boolean | NO | false |

**FK:** `external_platform_id` → `external_platforms(id)`

#### `webhook_delivery_logs`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `payment_request_id` | bigint | NO | |
| `webhook_id` | bigint | NO | |
| `status` | varchar(30) | NO | 'pending' |
| `attempt_count` | bigint | NO | 0 |
| `next_retry_at` | timestamptz | YES | |
| `last_attempt_at` | timestamptz | YES | |
| `last_response_code` | bigint | YES | |
| `last_error` | text | YES | |

**FK:** `payment_request_id` → `payment_requests(id)`, `webhook_id` → `webhooks(id)`

#### `ee_events`
Event emitter events (internal event bus).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `event_name` | varchar(200) | NO | |
| `profile_id` | varchar(200) | YES | |
| `attribute` | text | YES | |
| `valid_until` | timestamptz | YES | |
| `info` | text | YES | |

---

### 4.9 Analytics & Dashboard

#### `analytics_groups`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `name` | varchar(100) | NO | |
| `description` | text | YES | |
| `is_default` | boolean | YES | false |
| `category` | varchar(50) | YES | |
| `sequence` | bigint | NO | 0 |

#### `analytics_graphs`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `group_id` | bigint | NO | |
| `name` | varchar(100) | NO | |
| `description` | text | YES | |
| `graph_type` | varchar(50) | NO | |
| `query_template` | text | NO | |
| `filter_mappings` | text | YES | |
| `color_mappings` | text | YES | |
| `attributes` | text | YES | |
| `active` | boolean | YES | true |
| `sequence` | bigint | NO | 0 |

#### `analytics_filters`
| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `name` | varchar(100) | NO | |
| `type` | varchar(50) | NO | |
| `value` | text | NO | |
| `options` | text | YES | |
| `value_mappings` | text | YES | |
| `icon_mappings` | text | YES | |

#### `analytics_group_filters` (join)
PK: (analytics_group_id, analytics_filter_id)

#### `analytics_custom_filters`
Per-user custom filter values.

#### `analytics_user_groups`
Maps members to analytics groups with visibility toggle.

---

### 4.10 Referral System

#### `referral_campaigns`
Full-featured referral campaign engine with budgets, caps, and time-limited rewards.

Key columns: `name`, `reward_type`, `reward_value`, `currency_code`, `budget`, `start_date`, `end_date`, `status`, `campaign_type_per_customer`, `max_occurrences_per_customer`

#### `referral_events`
Trackable events: `key`, `name`, `event_type`

#### `referral_members`
Referral program members with `code` (referral code) and `referred_by_member_id`

#### `referral_rewards`
Reward ledger: `rewarded_member_id`, `related_member_id`, `amount`, `currency_code`, `member_type` (referrer/referee)

#### `referral_event_logs`
Event trigger log with `event_key`, `amount`, `triggered_at`, `data`

#### `referral_campaign_events` (join)
Links campaigns to events.

#### `referral_campaign_event_logs`
Per-campaign event processing with `referee_reward_id` and `referred_reward_id`

#### `referral_member_campaigns` (join)
Links members to campaigns.

#### `processed_rewards`
Tracks reward distribution: `reward_id`, `member_id`, `reference_id`, `status`, `failure_reason`

---

### 4.11 System & Configuration

#### `external_platforms`
Multi-tenant project/merchant entities (PayRam calls these "projects").

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `created_at` | timestamptz | YES | |
| `updated_at` | timestamptz | YES | |
| `deleted_at` | timestamptz | YES | |
| `name` | varchar(200) | NO | |
| `logo_path` | varchar(200) | YES | |
| `brand_color` | varchar(20) | YES | '#000000' |
| `website` | varchar(255) | YES | |
| `success_endpoint` | varchar(500) | YES | |
| `frontend_endpoint` | varchar(500) | YES | |
| `cancel_endpoint` | varchar(500) | YES | |
| `support_email` | varchar(255) | YES | |
| `linked_in_url` | varchar(255) | YES | |
| `twitter_url` | varchar(255) | YES | |
| `discord_url` | varchar(255) | YES | |
| `telegram_url` | varchar(255) | YES | |
| `default_payment_blockchain_id` | bigint | YES | |
| `default_payment_blockchain_currency_id` | bigint | YES | |
| `email_send_request_from` | varchar(255) | YES | |
| `email_send_request_reply_to` | varchar(255) | YES | |

#### `external_platform_blockchain_currencies` (join)
Per-project blockchain currency enablement.

| Column | Type | Nullable |
|--------|------|----------|
| `external_platform_id` | bigint | NO |
| `blockchain_currency_id` | bigint | NO |
| `blockchain_code` | varchar(20) | YES |
| `currency_code` | varchar(20) | YES |
| `blockchain_family` | varchar(20) | YES |

**UNIQUE:** (external_platform_id, blockchain_currency_id)

#### `member_external_platforms`
Links members to projects.

#### `member_external_platform_roles`
Role assignment per member per project.

**PK:** (member_id, external_platform_id)

#### `configurations`
System-wide key-value config store (JWT secrets, system settings).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `config_key` | varchar(200) | NO | |
| `config_value` | text | NO | |
| `override_config_value` | text | YES | |
| `description` | text | YES | |
| `encrypt` | boolean | YES | false |

#### `generic_data_stores`
Temporary/misc data storage with TTL.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `type` | varchar(200) | NO | |
| `valid_till` | timestamptz | YES | |
| `data` | text | YES | |
| `status` | varchar(20) | YES | 'active' |

#### `tags`
Polymorphic tagging system.

| Column | Type | Nullable |
|--------|------|----------|
| `id` | bigint | NO |
| `table_name` | varchar(100) | NO |
| `table_id` | bigint | NO |
| `name` | varchar(100) | NO |

#### `migrations`
| Column | Type |
|--------|------|
| `id` | varchar(255) |

#### `seeder_logs`
| Column | Type |
|--------|------|
| `name` | text (PK) |
| `hash` | varchar(64) |

---

### 4.12 Additional Tables (Discovered During Runtime)

#### `activity_logs`
Full API audit trail with geolocation.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | bigint | NO | Primary key |
| `member_id` | bigint | YES | Acting member |
| `session_id` | varchar(100) | YES | Session identifier |
| `project_ids` | jsonb | YES | Affected projects |
| `method` | varchar(10) | NO | HTTP method |
| `api_part` | varchar(255) | NO | API endpoint |
| `api_status` | varchar(50) | NO | success/error |
| `status_code` | bigint | YES | HTTP status code |
| `description` | varchar(255) | YES | |
| `ip_address` | varchar(50) | YES | |
| `user_agent` | varchar(500) | YES | |
| `referer` | varchar(500) | YES | |
| `api_action` | varchar(100) | NO | Action type |
| `api_error_msg` | varchar(1000) | YES | |
| `request_body` | text | YES | |
| `response_body` | text | YES | |
| `metadata` | json | YES | |
| `role` | varchar(100) | YES | |
| `event_category` | varchar(100) | YES | |
| `event_name` | varchar(100) | YES | |
| `country` | varchar(100) | YES | |
| `country_code` | varchar(10) | YES | |
| `region` | varchar(100) | YES | |
| `city` | varchar(100) | YES | |
| `timezone` | varchar(100) | YES | |
| `latitude` | numeric(10,7) | YES | |
| `longitude` | numeric(10,7) | YES | |

#### `auth_refresh_tokens`
JWT refresh token management with revocation support.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `member_id` | bigint | NO | |
| `token` | varchar(255) | NO | **UNIQUE** |
| `expires_at` | timestamptz | NO | |
| `last_used_at` | timestamptz | YES | |
| `revoked_at` | timestamptz | YES | |

**FK:** `member_id` → `members(id)`

#### `rpc_nodes`
Blockchain RPC node configuration (Alchemy, Infura, custom).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `blockchain_id` | bigint | NO | |
| `url` | varchar(500) | NO | |
| `node_type` | varchar(20) | NO | 'free' |
| `is_preferred` | boolean | NO | false |
| `is_active` | boolean | NO | true |
| `max_daily_calls` | bigint | YES | |
| `api_key` | varchar(500) | YES | |
| `auth_username` | varchar(200) | YES | |
| `auth_password` | varchar(200) | YES | |
| `chain_identifier` | varchar(100) | NO | |
| `credential_hash` | varchar(64) | NO | |

**UNIQUE:** (url, chain_identifier, credential_hash)
**FK:** `blockchain_id` → `blockchains(id)`

#### `recipients`
Payout recipient address book per merchant.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `name` | varchar(100) | YES | |
| `email` | varchar(100) | YES | |
| `mobile_number` | varchar(100) | YES | |
| `residential_address` | varchar(200) | YES | |
| `blockchain_code` | varchar(200) | NO | |
| `address` | varchar(200) | YES | |
| `member_id` | bigint | NO | |
| `status` | varchar(70) | NO | 'active' |
| `operated_by_member_id` | bigint | NO | |
| `last_operation` | varchar(200) | NO | |
| `blockchain_id` | bigint | NO | |

**UNIQUE:** (blockchain_code, address, member_id)
**FK:** `member_id` → `members(id)`, `operated_by_member_id` → `members(id)`, `blockchain_id` → `blockchains(id)`

#### `payment_channels`
Fiat onramp payment channel definitions.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `name` | varchar(200) | NO | |
| `type` | varchar(50) | NO | |
| `logo_url` | varchar(500) | YES | |
| `status` | varchar(20) | NO | 'active' |
| `fees_percentage` | numeric(10,6) | YES | |
| `min_amount` | numeric(38,18) | YES | |
| `max_amount` | numeric(38,18) | YES | |

#### `disabled_payment_channel_projects`
Per-project disabled payment channels.

| Column | Type | Nullable |
|--------|------|----------|
| `id` | bigint | NO |
| `external_platform_id` | bigint | NO |
| `payment_channel_id` | bigint | NO |
| `reason` | text | YES |
| `disabled_by_member_id` | bigint | YES |

**UNIQUE:** (external_platform_id, payment_channel_id)
**FK:** `external_platform_id` → `external_platforms(id)`, `payment_channel_id` → `payment_channels(id)`, `disabled_by_member_id` → `members(id)`

#### `payments_apps`
Per-project payment app settings (sponsorship configuration).

| Column | Type | Nullable |
|--------|------|----------|
| `id` | bigint | NO |
| `project_id` | bigint | NO |
| `sponsorship_percentage` | numeric(5,4) | NO |
| `sponsorship_cut_off` | numeric(38,18) | NO |

**UNIQUE:** (project_id)
**FK:** `project_id` → `external_platforms(id)`

#### `entrypoint_sc_addresses`
Smart contract entrypoint addresses (factory/router contracts).

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `name` | varchar(100) | NO | |
| `description` | text | YES | |
| `address` | varchar(200) | NO | **UNIQUE** |
| `status` | varchar(20) | NO | 'active' |

#### `address_deployments`
Tracks smart contract wallet deployments per blockchain.

| Column | Type | Nullable | Default |
|--------|------|----------|---------|
| `id` | bigint | NO | auto |
| `address` | varchar(200) | NO | |
| `blockchain_code` | varchar(20) | NO | |
| `transaction_hash` | varchar(200) | YES | |
| `transaction_fee` | varchar(50) | YES | |
| `blockchain_id` | bigint | NO | |
| `status` | varchar(20) | NO | 'pending' |
| `broadcasted_at` | timestamptz | YES | |

**UNIQUE:** (address, blockchain_id)
**FK:** `blockchain_id` → `blockchains(id)`

#### `wallet_scws`
Smart contract wallet (SCW) instances per blockchain.

| Column | Type | Nullable |
|--------|------|----------|
| `id` | bigint | NO |
| `family` | varchar(30) | NO |
| `blockchain_code` | varchar(20) | NO |
| `fund_collector_address` | text | YES |
| `transaction_hash` | text | YES |
| `wallet_id` | bigint | NO |
| `blockchain_family_id` | bigint | NO |
| `blockchain_id` | bigint | NO |

**UNIQUE:** (blockchain_code, wallet_id), (transaction_hash)
**FK:** `wallet_id` → `wallets(id)`, `blockchain_family_id` → `blockchain_families(id)`, `blockchain_id` → `blockchains(id)`

#### `wallet_xpubs`
BIP32 extended public keys for HD wallet derivation.

| Column | Type | Nullable |
|--------|------|----------|
| `id` | bigint | NO |
| `xpub` | varchar(500) | NO |
| `family` | varchar(30) | YES |
| `path` | text | YES |
| `wallet_id` | bigint | NO |
| `blockchain_family_id` | bigint | NO |
| `fund_collector_address` | text | YES |
| `xpub_hash` | char(64) | YES |

**UNIQUE:** (wallet_id, blockchain_family_id), (xpub_hash)
**FK:** `wallet_id` → `wallets(id)`, `blockchain_family_id` → `blockchain_families(id)`

#### `external_platform_wallet_blockchain_families`
Links projects to specific wallets per blockchain family.

| Column | Type | Nullable |
|--------|------|----------|
| `external_platform_id` | bigint | NO |
| `wallet_name` | varchar(255) | YES |
| `family` | varchar(20) | YES |
| `wallet_id` | bigint | NO |
| `blockchain_family_id` | bigint | NO |

**UNIQUE:** (external_platform_id, blockchain_family_id)
**FK:** `external_platform_id` → `external_platforms(id)`, `wallet_id` → `wallets(id)`, `blockchain_family_id` → `blockchain_families(id)`

#### `missed_deposits`
Deposits detected on-chain but not matched to any known address.

*(Schema follows standard GORM pattern with blockchain/currency references)*

---

## 5. Foreign Key Map

**110 foreign keys** connecting 63 tables. Core relationships:

| From Table | Column | → To Table | Column |
|-----------|--------|------------|--------|
| account_addresses | member_id | members | id |
| account_addresses | currency_id | currencies | id |
| account_addresses | blockchain_id | blockchains | id |
| account_addresses | address | address_pools | address |
| account_addresses | sweeping_transaction_id | sweep_transactions | id |
| account_rewards | member_id | members | id |
| account_rewards | currency_id | currencies | id |
| accounts | member_id | members | id |
| accounts | currency_id | currencies | id |
| address_contract_signatures | blockchain_contract_id | blockchain_contracts | id |
| address_contract_signatures | contract_address_id | contract_addresses | id |
| address_pools | blockchain_family_id | blockchain_families | id |
| address_pools | wallet_id | wallets | id |
| api_keys | member_id | members | id |
| api_keys | external_platform_id | external_platforms | id |
| api_keys | role_id | roles | id |
| assets | member_id | members | id |
| assets | account_id | accounts | id |
| assets | currency_id | currencies | id |
| blockchain_contracts | blockchain_id | blockchains | id |
| blockchain_currencies | currency_id | currencies | id |
| blockchain_currencies | blockchain_id | blockchains | id |
| blockchains | blockchain_family_id | blockchain_families | id |
| contract_addresses | blockchain_contract_id | blockchain_contracts | id |
| deposit_addresses | member_id | members | id |
| deposit_addresses | wallet_id | wallets | id |
| deposit_addresses | blockchain_family_id | blockchain_families | id |
| deposits | member_id | members | id |
| deposits | currency_id | currencies | id |
| deposits | blockchain_id | blockchains | id |
| expenses | member_id | members | id |
| expenses | account_id | accounts | id |
| expenses | currency_id | currencies | id |
| external_platforms | default_payment_blockchain_id | blockchains | id |
| external_platforms | default_payment_blockchain_currency_id | blockchain_currencies | id |
| internal_blockchain_transactions | member_id | members | id |
| internal_blockchain_transactions | blockchain_id | blockchains | id |
| internal_blockchain_transactions | currency_id | currencies | id |
| internal_blockchain_transactions | deposit_id | deposits | id |
| liabilities | member_id | members | id |
| liabilities | account_id | accounts | id |
| liabilities | currency_id | currencies | id |
| member_external_platform_roles | member_id | members | id |
| member_external_platform_roles | external_platform_id | external_platforms | id |
| member_external_platform_roles | role_id | roles | id |
| member_external_platforms | member_id | members | id |
| member_external_platforms | external_platform_id | external_platforms | id |
| member_roles | member_id | members | id |
| member_roles | role_id | roles | id |
| members | referred_by_member_id | members | id |
| payment_requests | member_id | members | id |
| payment_requests | currency_id | currencies | id |
| payment_requests | blockchain_id | blockchains | id |
| payment_requests | deposit_id | deposits | id |
| payment_requests | external_platform_id | external_platforms | id |
| revenues | member_id | members | id |
| revenues | account_id | accounts | id |
| revenues | currency_id | currencies | id |
| secrets_vaults | member_id | members | id |
| sweep_transactions | currency_id | currencies | id |
| sweep_transactions | blockchain_id | blockchains | id |
| sweep_transactions | wallet_id | wallets | id |
| sweeps | currency_id | currencies | id |
| sweeps | blockchain_id | blockchains | id |
| utxos | deposit_id | deposits | id |
| utxos | address | address_pools | address |
| utxos | sweeping_transaction_id | sweep_transactions | id |
| utxos | sweep_id | sweeps | id |
| wallet_functions | secret_vault_id | secrets_vaults | id |
| wallet_functions | wallet_id | wallets | id |
| wallets | member_id | members | id |
| wallets | blockchain_family_id | blockchain_families | id |
| wallets | blockchain_id | blockchains | id |
| wallets | currency_id | currencies | id |
| web_socket_tokens | member_id | members | id |
| webhook_delivery_logs | payment_request_id | payment_requests | id |
| webhook_delivery_logs | webhook_id | webhooks | id |
| webhooks | external_platform_id | external_platforms | id |
| withdrawals | member_id | members | id |
| withdrawals | currency_id | currencies | id |
| withdrawals | blockchain_id | blockchains | id |
| withdrawals | external_platform_id | external_platforms | id |
| withdrawals | created_by_member_id | members | id |
| withdrawals | created_by_project_id | external_platforms | id |
| withdrawals | approved_by_member_id | members | id |
| withdrawals | rejected_by_member_id | members | id |
| withdraws | currency_id | currencies | id |
| withdraws | blockchain_id | blockchains | id |

---

## 6. Index Summary

Every table has indexes on:
- `deleted_at` (soft delete filtering)
- `created_at` (time-based queries)
- All foreign key columns
- `status` columns

Notable unique indexes:
- `address_pools.address` — globally unique addresses
- `address_contract_signatures.signature` — unique signatures
- `external_platform_blockchain_currencies.(external_platform_id, blockchain_currency_id)` — unique per-project currency enablement

---

## 7. Key Design Patterns

### 7.1 GORM Conventions
- All tables use `id bigint` auto-increment primary keys
- Soft delete via `deleted_at` timestamp (queries filter `WHERE deleted_at IS NULL`)
- `created_at` / `updated_at` managed by GORM

### 7.2 Multi-Chain Architecture
```
blockchain_families (EVM, BTC, TRX)
  └── blockchains (ethereum, base, polygon, bitcoin, tron)
       └── blockchain_currencies (USDC-on-ethereum, USDC-on-base, etc.)
            └── blockchain_contracts (SmartSweep contracts per chain)
```

### 7.3 Address Pool Pattern
Pre-generated addresses in `address_pools` with status `open` → assigned to deposits. HD wallet derivation paths tracked via `path` and `path_index`.

### 7.4 Double-Entry Accounting
Four ledger tables (`assets`, `liabilities`, `revenues`, `expenses`) with identical structure. Each entry has `debit`/`credit` amounts and `reference_type`/`reference_id` for linking to source transactions.

### 7.5 Multi-Tenant via External Platforms
`external_platforms` = merchant projects. Members can belong to multiple platforms. API keys, webhooks, and payment requests are all scoped to a platform.

### 7.6 SmartSweep Flow
1. Deposits land in `address_pools` addresses → tracked in `account_addresses`
2. `sweep_transactions` batch multiple addresses into one sweep
3. On-chain result stored in `sweeps` table
4. `account_addresses.locked` tracks funds during sweep processing

### 7.7 Bitcoin UTXO Model
Separate `utxos` and `withdraw_deposits_btcs` tables for Bitcoin's UTXO model, tracking `script_pub_key_hex` and `raw_transaction_hex`.

---

## 8. SQL Recreation Script

The complete SQL schema is available at:
- **Raw SQL dump:** `payram_schema_raw.sql` (7,853 lines)

To recreate in a new database:
```bash
# Create database
psql -U postgres -c "CREATE USER payram WITH PASSWORD 'payram123';"
psql -U postgres -c "CREATE DATABASE payram OWNER payram;"

# Apply schema
psql -U payram -d payram < payram_schema_raw.sql
```

---

## Running PayRam Locally

### Prerequisites
- Docker (or Colima on macOS)
- PostgreSQL 14+

### Quick Start
```bash
# 1. Start Docker runtime (macOS with Colima)
colima start --cpu 4 --memory 8

# 2. Create database
psql -U postgres -c "CREATE USER payram WITH PASSWORD 'payram123';"
psql -U postgres -c "CREATE DATABASE payram OWNER payram;"

# 3. Generate AES key
AES_KEY=$(openssl rand -hex 32)

# 4. Get host IP (for Docker → host PostgreSQL)
HOST_IP=$(colima ssh -- ip route show default | awk '{print $3}')

# 5. Run PayRam (testnet)
docker run -d \
  --name payram \
  --platform linux/amd64 \
  --restart unless-stopped \
  --publish 8080:8080 \
  --publish 8443:8443 \
  --publish 8880:80 \
  --publish 8844:443 \
  -e AES_KEY="$AES_KEY" \
  -e BLOCKCHAIN_NETWORK_TYPE="testnet" \
  -e SERVER="DEVELOPMENT" \
  -e POSTGRES_SSLMODE="disable" \
  -e POSTGRES_HOST="$HOST_IP" \
  -e POSTGRES_PORT="5432" \
  -e POSTGRES_DATABASE="payram" \
  -e POSTGRES_USERNAME="payram" \
  -e POSTGRES_PASSWORD="payram123" \
  -e SSL_CERT_PATH="" \
  payramapp/payram:latest

# 6. Verify tables created
psql -U payram -d payram -c "\dt+"
```

### Access Points
| Service | URL |
|---------|-----|
| Backend API (HTTP) | http://localhost:8080 |
| Backend API (HTTPS) | https://localhost:8443 |
| Frontend (HTTP) | http://localhost:8880 |
| Frontend (HTTPS) | https://localhost:8844 |

### Environment Variables
| Variable | Value | Description |
|----------|-------|-------------|
| `AES_KEY` | 64-char hex | Hot wallet encryption key |
| `BLOCKCHAIN_NETWORK_TYPE` | testnet/mainnet | Network mode |
| `SERVER` | DEVELOPMENT/PRODUCTION | Server mode |
| `POSTGRES_HOST` | Host IP | DB host (use Docker gateway IP) |
| `POSTGRES_PORT` | 5432 | DB port |
| `POSTGRES_DATABASE` | payram | DB name |
| `POSTGRES_USERNAME` | payram | DB user |
| `POSTGRES_PASSWORD` | payram123 | DB password |
| `SSL_CERT_PATH` | (empty) | SSL cert path (empty = no SSL) |
