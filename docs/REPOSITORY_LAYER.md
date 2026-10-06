# Payminto Repository Layer Reference

> Comprehensive documentation of the Go repository layer for the Payminto payment gateway backend. Derived from reverse-engineering of the PayRam reference implementation (50 repository implementations, 425+ functions, ~12,000 lines of repository code).

---

## Table of Contents

1. [Overview](#1-overview)
2. [Database Patterns](#2-database-patterns)
3. [Repository Catalog](#3-repository-catalog)
   - [Account Domain](#31-account-domain)
   - [Address & Pool Domain](#32-address--pool-domain)
   - [Blockchain Domain](#33-blockchain-domain)
   - [Configuration & System Domain](#34-configuration--system-domain)
   - [Currency Domain](#35-currency-domain)
   - [Deposit Domain](#36-deposit-domain)
   - [External Platform Domain](#37-external-platform-domain)
   - [Identity & Auth Domain](#38-identity--auth-domain)
   - [Payment Domain](#39-payment-domain)
   - [Sweep Domain](#310-sweep-domain)
   - [Wallet Domain](#311-wallet-domain)
   - [Webhook Domain](#312-webhook-domain)
   - [Withdrawal Domain](#313-withdrawal-domain)
   - [Miscellaneous Domain](#314-miscellaneous-domain)
4. [Accounting Integration Pattern](#4-accounting-integration-pattern)
5. [Encryption Patterns](#5-encryption-patterns)
6. [Locking and Concurrency](#6-locking-and-concurrency)
7. [Complex Query Patterns](#7-complex-query-patterns)

---

## 1. Overview

### Repository Pattern Architecture

The Payminto backend follows a **clean architecture** repository pattern where:

- Each domain entity has a **repository interface** (defined in a separate file or alongside the implementation) and a corresponding **implementation struct** (e.g., `AccountRepositoryImpl`).
- All implementations receive a `*gorm.DB` instance via their struct field, typically named `db`.
- Services depend on repository interfaces, enabling testability via mocks.
- Transaction management is handled by passing `*gorm.DB` transaction objects into closures or using GORM's `Transaction()` method.

### File Organization

```
internal/repository/
    account_address_repo_impl.go
    account_repo_impl.go
    account_reward_repo_impl.go
    address_contract_signature_repo_impl.go
    address_deployment_repo_impl.go
    address_pool_repo_impl.go
    address_repo_impl.go
    analytics_repo_impl.go
    api_key_repo_impl.go
    auth_refresh_token_repo_impl.go
    auth_repo_impl.go
    blockchain_contract_repo_impl.go
    blockchain_currency_repo_impl.go
    blockchain_family_repo_impl.go
    blockchain_repo_impl.go
    configuration_repo_impl.go
    contract_address_repo_impl.go
    currency_repo_impl.go
    deposit_addresses_repo_impl.go
    deposit_repo_impl.go
    entrypoint_sc_addresses_repo_impl.go
    external_platform_blockchain_currency_repo_impl.go
    external_platform_repo_impl.go
    external_platform_wallet_blockchain_family_repo_impl.go
    generic_data_store_repo_impl.go
    internal_blockchain_transaction_repo_impl.go
    member_external_platform_role_repo_impl.go
    member_repo_impl.go
    member_role_repo_impl.go
    missed_deposit_repo_impl.go
    onramper_payments_repo_impl.go
    option.go                              // Pagination and preload helpers
    otp_repo_impl.go
    payment_channel_project_repo_impl.go
    payment_channel_repo.go                // Filter helpers
    payment_channel_repo_impl.go
    payment_repo_impl.go
    payments_app_repo_impl.go
    permission_repo_impl.go
    recipient_repo_impl.go
    reward_accounting_repo_impl.go
    role_repo_impl.go
    rpc_node_repo_impl.go
    sweep_repo_impl.go
    sweep_transaction_repo_impl.go
    sweep_utxo_repo_impl.go
    utils_repo.go                          // sha256Sum helper
    utxo_repo_impl.go
    wallet_function_repo_impl.go
    wallet_repo_impl.go
    webhook_delivery_log_repo_impl.go
    webhook_repo_impl.go
    websocket_token_repo_impl.go
    withdrawal_processing_repo_impl.go
    withdrawal_repo_impl.go
```

### Implementation Struct Pattern

Every repository implementation follows this convention:

```go
type XxxRepositoryImpl struct {
    db *gorm.DB
}

func NewXxxRepository(db *gorm.DB) XxxRepository {
    return &XxxRepositoryImpl{db: db}
}
```

### Shared Utilities

- **`option.go`** -- Defines `AddPaginationParams` with functional options: `WithAscendingOrder`, `WithDescendingOrder`, `WithLimit`, `WithOffset`. Also defines `WithPreload` option functions for `APIKeyRepositoryImpl`.
- **`utils_repo.go`** -- Contains `sha256Sum` helper (2 lines) for hashing operations.
- **`payment_channel_repo.go`** -- Contains `ApplyPaymentChannelFilters` (54 lines) standalone filter function.

---

## 2. Database Patterns

### Base Models

Two base model types are used across all entities:

```go
// PayramModel -- used by most domain entities
type PayramModel struct {
    ID        uint           `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BaseModel -- used by activity logs, campaigns, events, rewards
type BaseModel struct {
    ID        uint           `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

Both provide:
- **Auto-increment bigint IDs** (not UUIDs)
- **Soft deletes** via `gorm.DeletedAt` (GORM automatically filters deleted records)
- **Timestamps** managed by GORM hooks

### Monetary Precision

All monetary amounts use `decimal.Decimal` (from `shopspring/decimal`) mapped to PostgreSQL `numeric(38,18)`:
- Balances, fees, amounts, prices
- Supports 18 decimal places (Ethereum wei precision)
- Never uses `float64` for money

### Preloading Strategies

Repositories use several GORM preloading patterns:

1. **Eager Preload** -- `db.Preload("Association")` for direct relationships
2. **Nested Preload** -- `db.Preload("Association.SubAssociation")` for deep graphs
3. **Conditional Preload** -- `db.Preload("Association", "status = ?", "active")`
4. **Joins Preload** -- `db.Joins("Association")` for inner join loads
5. **Functional Options** -- `WithPreload` pattern in APIKeyRepository

### Transaction Patterns

Two primary transaction strategies:

```go
// Pattern 1: GORM Transaction closure
func (r *Repo) DoWork() error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        // All operations use tx instead of r.db
        return nil
    })
}

// Pattern 2: Explicit DB parameter (WithDB suffix)
func (r *Repo) DoWorkWithDB(db *gorm.DB) error {
    return db.Model(&Model{}).Where(...).Update(...)
}
```

### Query Conventions

- **Soft delete awareness** -- All queries automatically exclude `deleted_at IS NOT NULL` records
- **Scoped queries** -- `db.Where("member_id = ?", memberID)` pattern
- **Batch operations** -- `db.CreateInBatches(records, batchSize)` for bulk inserts
- **Raw SQL** -- Used sparingly for complex aggregations in Analytics and Payment repos
- **Subqueries** -- Used in address eligibility checks, sweep calculations, and payment search

---

## 3. Repository Catalog

### 3.1 Account Domain

#### AccountAddressRepositoryImpl

**File:** `account_address_repo_impl.go`
**Models:** `AccountAddress`, `AddressPool`, `SweepTransaction`
**Total Lines:** ~36

| Method | Lines | Description |
|--------|-------|-------------|
| `GetByBlockchainCurrencyCodeAndAddress` | 15 | Finds an account address by blockchain code, currency code, and address string. Preloads `AddressPool`. |
| `GetByBlockchainCurrencyAndAddresses` | 11 | Batch lookup of account addresses for a list of address strings filtered by blockchain and currency codes. |
| `GetAllPendingAccountsForSweepTransaction` | 10 | Retrieves all account addresses linked to a specific sweep transaction ID with pending status. |

**Query Patterns:**
- Simple WHERE clauses with Preload("AddressPool")
- IN clause for batch address lookups
- Foreign key filtering on `sweeping_transaction_id`

---

#### AccountRepositoryImpl

**File:** `account_repo_impl.go`
**Models:** `Account`, `AccountAddress`, `Asset`, `Liability`, `Revenue`, `Expense`, `InternalBlockchainTransaction`, `Deposit`, `Sweep`, `SweepTransaction`
**Total Lines:** ~428

| Method | Lines | Description |
|--------|-------|-------------|
| `ProcessBlockchainInternalTransferEthereumGasFees` | 103 | Processes accounting entries for ETH gas fee transfers. Runs within a transaction. Creates Asset debit and Expense credit entries for gas costs. |
| `ProcessERC20Sweep` | 203 | **Most complex method in this repo.** Handles full accounting for ERC-20 token sweep operations. Within a single transaction: updates sweep status, creates Asset/Liability/Revenue/Expense entries, updates account address balances, and marks addresses as swept. |
| `ProcessDuplicateDepositsAsExpense` | 122 | Identifies duplicate deposit entries and records them as expenses in the double-entry ledger. Uses transaction to atomically update deposit status and create corresponding accounting entries. |

**Transaction Boundaries:**
- All three methods run inside `db.Transaction()` closures
- Inner functions (e.g., `ProcessERC20Sweep.func1` at 178 lines) contain the actual transaction logic
- Multiple model types are modified atomically

**Accounting Pattern:**
- Creates paired Asset/Liability entries for every financial event
- Revenue entries for platform fees
- Expense entries for operational costs (gas, duplicates)

---

#### AccountRewardRepositoryImpl

**File:** `account_reward_repo_impl.go`
**Models:** `AccountReward`, `Currency`
**Total Lines:** ~38

| Method | Lines | Description |
|--------|-------|-------------|
| `GetByMemberID` | 10 | Lists all reward accounts for a member. Preloads `Currency`. |
| `GetByMemberIDAndCurrencyID` | 11 | Finds a specific reward account by member and currency. |
| `GetTotalLockedAndBalance` | 17 | Aggregates total locked and balance amounts across all reward accounts for a member. Uses `Select("SUM(balance), SUM(locked)")` aggregation. |

**Query Patterns:**
- SUM aggregation query for balance totals
- Standard WHERE + Preload for lookups

---

### 3.2 Address & Pool Domain

#### AddressContractSignatureRepositoryImpl

**File:** `address_contract_signature_repo_impl.go`
**Models:** `AddressContractSignature`, `BlockchainContract`, `ContractAddress`
**Total Lines:** ~105

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Persists a new address contract signature record. |
| `UpdateStatusByAddressAndContractType` | 31 | Updates the status field for signatures matching a given address and contract type. Complex WHERE clause with multiple conditions. |
| `GetByBlockchainCurrencyCodesAddressContractTypeAndAddress` | 13 | Multi-field lookup by blockchain code, currency code, contract type, and address. |
| `GetByBlockchainCodeContractTypeAndStatus` | 9 | Filters signatures by blockchain code, contract type, and status. |
| `GetByIdWithDB` | 11 | Retrieves by ID using an externally provided `*gorm.DB` (for use within transactions). Preloads `BlockchainContract` and `ContractAddress`. |
| `UpdateWithDB` | 9 | Updates using externally provided `*gorm.DB`. |
| `GetByAddressPathIndex` | 13 | Finds signatures by address path index, blockchain code, and contract type. |
| `GetActiveSignatureByAddress` | 10 | Returns active signature for a given address. |

**Query Patterns:**
- `WithDB` suffix methods accept external transaction context
- Multi-column composite lookups
- Preloading nested blockchain contract data

---

#### AddressDeploymentRepositoryImpl

**File:** `address_deployment_repo_impl.go`
**Models:** `AddressDeployment`, `Blockchain`, `Asset`, `Expense`
**Total Lines:** ~243

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateAddressDeployment` | 8 | Inserts a single address deployment record. |
| `BulkCreateAddressDeployments` | 13 | Batch insert of address deployment records using `CreateInBatches`. |
| `BulkUpdateAddressDeployments` | 16 | Batch update of deployment records (status, transaction hash, broadcasted timestamp). |
| `GetAddressDeployments` | 33 | Paginated query with optional filters (status, blockchain code, address). Preloads `Blockchain`. |
| `DeleteAddressDeployment` | 8 | Soft-deletes a deployment record by ID. |
| `ProcessAddressDeploymentAccounting` | 132 | **Complex accounting method.** Within a transaction: validates the deployment, calculates fees, creates Asset debit entries for deployment costs, creates Expense credit entries, and updates the deployment status. |
| `UpdateTransactionFeeByTransactionHash` | 35 | Updates the transaction fee for a deployment identified by its transaction hash. Runs in a transaction. |

**Transaction Boundaries:**
- `ProcessAddressDeploymentAccounting` -- full transaction with inner closure (131 lines)
- `UpdateTransactionFeeByTransactionHash` -- transaction with inner closure (6 lines)

---

#### AddressPoolRepositoryImpl

**File:** `address_pool_repo_impl.go`
**Models:** `AddressPool`, `BlockchainFamily`, `Wallet`, `AccountAddress`, `UTXO`
**Total Lines:** ~366

| Method | Lines | Description |
|--------|-------|-------------|
| `GetByBlockchainFamilyAndAddressAndStatus` | 12 | Finds address pool entry by family, address, and status. |
| `GetAddressPools` | 18 | Lists address pools with optional family and status filters. Preloads `BlockchainFamily`. |
| `GetLastAddressPool` | 13 | Gets the most recently created address pool entry for a given family. Orders by `path_index DESC`. |
| `GetOpenedAddressPoolCount` | 11 | Counts address pool entries with "opened" status for a given wallet and family. |
| `BulkSaveAddresses` | 15 | Batch creates address pool entries. |
| `GetAddressesWithUnassignedPrivateKeys` | 15 | Finds pool addresses that have NULL private keys for a given family. |
| `AddPrivateKeys` | 28 | **Encryption method.** Encrypts private keys using RSA before storing them. Iterates addresses, calls `encryptPrivateKey`, and performs batch update. |
| `GetPrivateKeys` | 34 | **Decryption method.** Retrieves and decrypts private keys from the pool. Calls `decryptPrivateKey` for each record. |
| `GetCountForUnassignedAddresses` | 11 | Counts addresses without assigned private keys. |
| `GetByAddress` | 12 | Single address lookup with Preload. |
| `GetLastGeneratedAddress` | 29 | Complex query to find the last generated address for a wallet, ordered by `path_index DESC`. |
| `ProcessDepositWalletDeployment` | 48 | Updates address pool entries and related account addresses when a deposit wallet is deployed. |

**Encryption Helpers (package-level functions):**

| Function | Lines | Description |
|----------|-------|-------------|
| `encryptPrivateKey` | 27 | Encrypts a private key using AES-GCM with a key derived from RSA-encrypted password and nonce. |
| `getKeyFromRSAEncryptedPasswordWithNonce` | 25 | Derives an AES key from an RSA-encrypted password combined with a nonce. Uses SHA-256 for key derivation. Contains two inner closures. |
| `decryptPrivateKey` | 27 | Reverses the encryption process: derives key from RSA password + nonce, then decrypts AES-GCM. Contains two inner closures. |

**Query Patterns:**
- `ORDER BY path_index DESC LIMIT 1` for last-generated lookups
- Batch operations with `CreateInBatches`
- NULL checks for unassigned private keys: `WHERE private_key IS NULL`

---

#### AddressRepositoryImpl

**File:** `address_repo_impl.go`
**Models:** `AccountAddress`, `AddressPool`, `BlockchainCurrency`, `Blockchain`, `SweepTransaction`, `Wallet`, `WalletXpub`, `WalletSCW`
**Total Lines:** ~578

| Method | Lines | Description |
|--------|-------|-------------|
| `Balances` | 11 | Entry point that delegates to `getBalance`. |
| `getBalance` | 130 | **Complex aggregation query.** Fetches account address balances with joins across multiple tables (account_addresses, address_pools, blockchains, currencies). Builds dynamic WHERE conditions. Uses `defer` for error recovery (30-line deferwrap). |
| `PendingForApproval` | 72 | Finds addresses pending ERC-20 approval. Joins address_pools, blockchain_currencies, account_addresses with status filters and minimum balance checks. |
| `GetEligibleAddressesToTransferFees` | 29 | Identifies addresses eligible for fee transfers. Filters by balance thresholds from blockchain_currency configuration. |
| `GetEligibleSCWAddressPoolsToBroadcast` | 69 | Finds Smart Contract Wallet (SCW) address pools ready for broadcast. Complex join across address_pools, blockchains, blockchain_families with Create2 support checks. |
| `GetEligibleAddressesToSweep` | 158 | **Longest method in this repo.** Identifies addresses eligible for sweep operations. Implements complex business logic: checks minimum collection amounts, sweep batch sizes, locked duration, and balance thresholds. Uses subqueries for time-based filtering and aggregation. |
| `ProcessAddressBalanceAndUpdateStatus` | 78 | Updates address balance and status within a transaction. Handles status transitions (e.g., from "pending" to "active"). |

**Query Patterns:**
- Complex multi-table JOINs (5+ tables)
- Subqueries for threshold-based filtering
- Time-based conditions (`sweep_address_locked_duration`)
- Deferred error recovery with `deferwrap`
- Dynamic WHERE clause building based on optional parameters

---

### 3.3 Blockchain Domain

#### BlockchainContractRepositoryImpl

**File:** `blockchain_contract_repo_impl.go`
**Models:** `BlockchainContract`, `Blockchain`, `ContractAddress`
**Total Lines:** ~62

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new blockchain contract record. |
| `GetByID` | 13 | Retrieves contract by ID. Preloads `ContractAddresses` and `Blockchain`. |
| `GetByBlockchainCodeAndContractType` | 13 | Finds contract by blockchain code and type. Preloads associations. |
| `GetByContractType` | 13 | Filters by contract type with preloads. |
| `GetAllContractsByAddressContractType` | 14 | Retrieves all contracts for a given address contract type. Preloads `ContractAddresses`. |

**Query Patterns:**
- Consistent Preload("ContractAddresses") and Preload("Blockchain")
- Simple WHERE equality filters

---

#### BlockchainCurrencyRepositoryImpl

**File:** `blockchain_currency_repo_impl.go`
**Models:** `BlockchainCurrency`, `Blockchain`, `Currency`
**Total Lines:** ~138

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new blockchain-currency mapping. |
| `Update` | 9 | Updates an existing mapping. |
| `GetByBlockchainIDCurrencyID` | 10 | Composite key lookup. |
| `GetAllBlockchainCurrencies` | 10 | Lists all mappings. |
| `GetByBlockchainIDCurrencyIDWithBlockchainAndCurrencyPreload` | 10 | Same as above but with both Blockchain and Currency preloaded. |
| `CreateOrUpdateBlockchainCurrency` | 26 | Upsert pattern: attempts to find existing record, creates if not found, updates if found. |
| `GetBlockchainCurrencyByCurrencyAddressWithCurrencyAndBlockchainPreload` | 9 | Lookup by currency contract address with preloads. |
| `GetBlockchainCurrenciesByBlockchainIdWithBlockchainAndCurrencyPreload` | 9 | Lists all currencies for a blockchain. |
| `GetBlockchainCurrenciesByCurrencyIdWithBlockchainPreload` | 9 | Lists all blockchains for a currency. |
| `GetByCurrencyAddressAndBlockchainIDWithBlockchainAndCurrencyPreload` | 14 | Composite lookup by address + blockchain ID with full preloads. |

**Query Patterns:**
- Upsert via `FirstOrCreate` followed by `Updates`
- Heavy use of preloading for API response enrichment

---

#### BlockchainFamilyRepositoryImpl

**File:** `blockchain_family_repo_impl.go`
**Models:** `BlockchainFamily`
**Total Lines:** ~50

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new blockchain family. |
| `GetByFamily` | 20 | Finds a blockchain family by family code string. Preloads related data. |
| `GetBlockchainFamilyById` | 14 | Lookup by ID with preloads. |
| `GetAll` | 7 | Returns all blockchain families. |

---

#### BlockchainRepositoryImpl

**File:** `blockchain_repo_impl.go`
**Models:** `Blockchain`, `BlockchainFamily`, `Deposit`, `PaymentRequest`, `Withdraw`, `WithdrawDepositsBTC`, `UTXO`, `AddressPool`, `Account`, `Asset`, `Liability`, `Revenue`, `Expense`, `WebhookDeliveryLog`
**Total Lines:** ~1155+ (the most complex repository)

| Method | Lines | Description |
|--------|-------|-------------|
| `Update` | 12 | Updates a blockchain record. |
| `GetBlockchains` | 14 | Lists all blockchains with `BlockchainFamily` and `RPCNodes` preloads. |
| `GetBlockchainByCode` | 11 | Finds blockchain by code string. |
| `CreateDepositReceived` | 142 | **Critical path method.** Creates a deposit record when a blockchain transaction is detected. Within a transaction: validates the deposit, checks for duplicates, creates the deposit entry, links it to payment requests, creates accounting entries (Asset credit, Liability debit), and triggers webhook delivery logs. |
| `GetDeposits` | 32 | Paginated deposit query with filters (member_id, status, blockchain_code, currency_code). Preloads Member, Currency, Blockchain. |
| `UpdateDepositAfterConfirmation` | 45 | Transaction-wrapped method. Updates deposit status after reaching required confirmations. Updates block number, block hash, and status. |
| `UpdateDepositStatus` | 34 | Transaction-wrapped status update for deposits. |
| `linkDepositToOpenPaymentRequest` | 39 | **Internal method.** Finds open payment requests matching the deposit's member and currency, then links them. Updates payment request status and filled amounts. |
| `GetAllPendingPaymentRequestsForAccounting` | 12 | Retrieves payment requests in "pending_accounting" status. |
| `GetPaymentRequests` | 39 | Paginated payment request query with extensive filters. |
| `GetPaymentRequestsWithConfirmingDeposits` | 52 | Complex query joining payment requests with their associated deposits that are in "confirming" status. |
| `ProcessPaymentRequestAccounting` | 99 | **Accounting method.** Transaction-wrapped. Processes confirmed payment requests: creates Asset debit entries (merchant receives), Liability credit entries (platform obligation), Revenue entries for fees. Updates payment request status to "completed". |
| `ProcessWithdrawsAccounting` | 82 | **Accounting method.** Transaction-wrapped. Processes withdrawal confirmations: creates Liability debit (obligation reduced), Asset credit (funds sent), Expense entries for blockchain fees. |
| `GetAllUnspentDeposits` | 16 | Retrieves unspent deposits for a given blockchain and currency. Used by UTXO-based chains (Bitcoin). |
| `GetAllUnspentDepositsByTxIDs` | 15 | Batch lookup of unspent deposits by transaction IDs. |
| `WithdrawResponse` | 64 | **Transaction-wrapped.** Processes withdrawal response from blockchain. Updates withdrawal status, creates accounting entries, handles both success and failure cases. |
| `CreateUTXOEntries` | 8 | Batch creates UTXO records. |
| `UpdateWithdrawalDepositBTCEntriesToSpent` | 60 | Marks Bitcoin deposit entries as spent after they are consumed in a withdrawal. Updates UTXO statuses. |
| `GetDepositsByIDs` | 10 | Batch deposit lookup by IDs. |
| `GetAddressPoolByAddress` | 12 | Finds address pool entry by address string. |
| `GetWithdrawsByStatus` | 11 | Lists withdrawals (Withdraw model) filtered by status. |
| `UpdateWithdraw` | 9 | Updates a withdrawal record. |
| `GetBlockchainByFamilyAndClient` | 13 | Finds blockchain by family code and client identifier. |
| `UpdateBlockHeightForBlockchainId` | 19 | Updates the latest processed block height for a blockchain. Used by block monitor workers. |
| `GetBlockchainByID` | 16 | Lookup by ID with BlockchainFamily preload. |
| `UpsertDepositByUniqueTxHash` | 91 | **Complex upsert.** Transaction-wrapped (98-line inner closure). Checks for existing deposit by unique_tx_hash. If exists, updates; if not, creates. Handles linking to payment requests and creating initial accounting entries. |
| `MarkStaleConfirmingDeposits` | 23 | Identifies and marks deposits that have been in "confirming" status beyond a threshold as stale. |

**Helper Functions (package-level):**

| Function | Lines | Description |
|----------|-------|-------------|
| `createLiabilityDebit` | 16 | Creates a Liability debit entry for a given account and amount. |
| `createAssetDebitForWithdraw` | 41 | Creates Asset debit entries for withdrawal operations. Handles fee calculations. |
| `createExpenseCreditForWithdraw` | 29 | Creates Expense credit entries for withdrawal blockchain fees. |
| `createRevenueCredit` | 16 | Creates Revenue credit entries for platform fee income. |
| `createDeliveryLogsForPaymentRequest` | 13 | Creates webhook delivery log entries for a payment request. |

**This is the most complex repository** with the deepest integration across models. It serves as the central coordination point for deposit/withdrawal lifecycle management and accounting.

---

### 3.4 Configuration & System Domain

#### ConfigurationRepositoryImpl

**File:** `configuration_repo_impl.go`
**Models:** `Configuration`
**Total Lines:** ~141

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new configuration key-value pair. |
| `BulkCreateConfigurations` | 17 | Transaction-wrapped batch insert of configurations. |
| `FindByKey` | 10 | Retrieves configuration by key string. |
| `Update` | 21 | Updates configuration value. Handles encrypted configurations. |
| `Upsert` | 32 | Creates or updates a configuration entry. Uses `FirstOrCreate` followed by `Updates`. |
| `GetConfigurationsWithPrefix` | 14 | Retrieves all configurations where key starts with a given prefix. Uses `LIKE` query. |
| `DeleteConfigurationsWithPrefix` | 10 | Soft-deletes all configurations matching a key prefix. |

**Query Patterns:**
- `LIKE` prefix matching for configuration groups
- Upsert pattern with `FirstOrCreate`

---

#### ContractAddressRepositoryImpl

**File:** `contract_address_repo_impl.go`
**Models:** `ContractAddress`, `BlockchainContract`
**Total Lines:** ~63

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new contract address record. |
| `Update` | 9 | Updates a contract address record. |
| `GetByBlockchainCodeAndContractType` | 22 | Finds contract addresses by blockchain code and contract type. Preloads `BlockchainContract`. Supports both active and all-status queries. |
| `GetAllByContractType` | 15 | Lists all contract addresses for a given contract type. |
| `GetById` | 7 | Simple ID lookup. |

---

#### GenericDataStoreRepositoryImpl

**File:** `generic_data_store_repo_impl.go`
**Models:** `GenericDataStore`
**Total Lines:** ~53

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new generic data store entry. |
| `GetByBlockchainCurrencyCodeAndMultipleData` | 23 | Complex query filtering by type, status, and multiple data values using IN clause. Also filters by `valid_till` timestamp. |
| `Update` | 9 | Updates a generic data store entry. |
| `BulkCreate` | 12 | Batch insert of data store entries. |

**Purpose:** The GenericDataStore is used as a flexible key-value store with TTL (valid_till) for temporary blockchain data such as nonces, pending transaction hashes, and intermediate processing states.

---

#### AnalyticsRepositoryImpl

**File:** `analytics_repo_impl.go`
**Models:** `AnalyticsGroup`, `AnalyticsUserGroup`, `AnalyticsCustomFilter`, `AnalyticsFilter`, `AnalyticsGraph`
**Total Lines:** ~652

| Method | Lines | Description |
|--------|-------|-------------|
| `GetAnalyticsGroupsByCategory` | 15 | Lists analytics groups filtered by category. Preloads `AnalyticsFilters` and `AnalyticsGraphs`. |
| `GetAnalyticsUserGroups` | 32 | Gets user-specific analytics groups with custom filter overrides. Contains inner function call. |
| `GetAnalyticsCustomFilter` | 10 | Retrieves custom filter values for a specific user group and filter combination. |
| `CreateOrUpdateGroupFilter` | 33 | Transaction-wrapped upsert of analytics custom filter values. |
| `FetchData` | 428 | **The longest single method across all repositories.** Executes dynamic analytics queries based on graph templates. Performs SQL template substitution, applies filter mappings, executes raw SQL, and formats results for chart rendering. Contains 234-line defer wrapper for error recovery. |

**Helper Functions:**

| Function | Lines | Description |
|----------|-------|-------------|
| `getAnalyticsUserGroups` | 87 | Standalone helper for building the user groups query with custom filter merging. |
| `trimAndRemoveFunc` | 8 | String processing helper for query template cleanup. |

**Query Patterns:**
- Raw SQL execution with template substitution (`FetchData`)
- Dynamic query building from stored templates
- Complex preloading chains for nested analytics structures

---

#### RPCNodeRepositoryImpl

**File:** `rpc_node_repo_impl.go`
**Models:** `RPCNode`, `Blockchain`
**Total Lines:** ~65

| Method | Lines | Description |
|--------|-------|-------------|
| `GetRPCNodes` | 20 | Lists RPC nodes with optional filters (blockchain_id, is_active, node_type). |
| `GetRPCNodeByID` | 14 | Retrieves RPC node by ID with Blockchain preload. |
| `GetRPCNodeByCompositeKey` | 14 | Finds RPC node by URL + blockchain_id + node_type composite key. |
| `CreateRPCNode` | 11 | Inserts a new RPC node configuration. |
| `UpdateRPCNode` | 7 | Updates an RPC node configuration. |

---

#### EntrypointSCAddressRepositoryImpl

**File:** `entrypoint_sc_addresses_repo_impl.go`
**Models:** `EntrypointSCAddress`
**Total Lines:** ~6

| Method | Lines | Description |
|--------|-------|-------------|
| `GetEntrypointSCAddressByAddress` | 6 | Finds a smart contract entrypoint address by its address string. |

**Note:** This is the smallest repository -- a single lookup method for the ERC-4337 account abstraction entrypoint addresses.

---

### 3.5 Currency Domain

#### CurrencyRepositoryImpl

**File:** `currency_repo_impl.go`
**Models:** `Currency`
**Total Lines:** ~66

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new currency record. |
| `GetByCode` | 15 | Finds currency by code. Returns preloaded data. |
| `GetAllCurrencies` | 9 | Lists all currencies. |
| `CreateOrUpdateCurrency` | 26 | Upsert pattern for currencies. Uses `FirstOrCreate` + `Updates`. |
| `GetCurrencyByCode` | 9 | Simple code-based lookup. |
| `GetCurrencyByAddress` | 5 | Finds currency by contract address (for tokens). |

---

### 3.6 Deposit Domain

#### DepositAddressesRepositoryImpl

**File:** `deposit_addresses_repo_impl.go`
**Models:** `DepositAddress`, `Wallet`, `WalletXpub`, `BlockchainFamily`, `Member`
**Total Lines:** ~138

| Method | Lines | Description |
|--------|-------|-------------|
| `FindAddressByAddressPreloadAddressXPUBandBlockchainFamily` | 15 | Finds deposit address with full association loading. Preloads Wallet, Wallet.WalletXpubs, BlockchainFamily. |
| `IdentifyAndReturnDepositAddressesFromList` | 18 | Given a list of addresses, returns which ones are registered deposit addresses. Uses `address_lower IN (?)` for case-insensitive matching. |
| `GetDepositAddressByMemberId` | 9 | Lists deposit addresses for a member. |
| `GetDepositAddressWithAddressXPUB` | 9 | Retrieves deposit address with XPUB wallet preload. |
| `FirstOrAssignDepositAddress` | 59 | **Address assignment logic.** Finds or creates a deposit address for a member and blockchain family. If no existing address is found, assigns one from the address pool (status = "opened") and marks it as assigned. Contains transaction closure. |

**Query Patterns:**
- Case-insensitive address matching via lowercase stored column
- Address pool assignment with status transitions
- Transaction for atomic first-or-assign operation

---

#### DepositRepositoryImpl

**File:** `deposit_repo_impl.go`
**Models:** `Deposit`, `PaymentRequest`, `Account`, `Blockchain`, `Asset`, `Liability`, `Expense`, `ExternalPlatform`, `Webhook`, `WebhookDeliveryLog`
**Total Lines:** ~408

| Method | Lines | Description |
|--------|-------|-------------|
| `GetDepositsGreaterThanID` | 13 | Retrieves deposits with ID greater than a given value. Used for incremental processing. |
| `GetAllPendingDeposits` | 11 | Lists all deposits in "pending" status for a given blockchain. |
| `ProcessPendingDeposit` | 156 | **Central deposit processing method.** Transaction-wrapped (149-line inner closure). State machine: transitions deposit from "pending" to "processing". Creates Account balance entry if not exists. Creates Liability (credit) and Asset (debit) accounting entries. Links to payment requests via `handlePaymentRequest`. Updates webhook delivery logs. |
| `handlePaymentRequest` | 109 | **Internal method.** Handles the business logic of matching deposits to payment requests. Checks for open payment requests, calculates filled amounts, determines if payment is complete or partial, updates statuses accordingly. Creates webhook delivery logs for merchant notification. |
| `GetBlockchainByID` | 10 | Blockchain lookup by ID. |

**Helper Functions (package-level):**

| Function | Lines | Description |
|----------|-------|-------------|
| `createLiability` | 32 | Creates Liability credit entry for deposit receipt. |
| `createAsset` | 17 | Creates Asset debit entry for deposit receipt. |
| `createExpense` | 25 | Creates Expense entries for deposit fees. |

**State Machine (Deposit Status):**
```
received -> confirming -> confirmed -> pending -> processing -> completed
                                                              -> failed
```

**Transaction Boundaries:**
- `ProcessPendingDeposit` wraps the entire deposit processing pipeline in a single transaction
- Atomic creation of accounting entries + status updates + payment request linking

---

#### MissedDepositRepositoryImpl

**File:** `missed_deposit_repo_impl.go`
**Models:** `MissedDeposit`
**Total Lines:** ~56

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateMissedDeposit` | 9 | Records a missed deposit for later investigation. |
| `GetMissedDeposits` | 14 | Lists missed deposits with optional status and blockchain code filters. |
| `UpdateMissedDeposits` | 33 | Transaction-wrapped batch update of missed deposit statuses. Contains inner closure (19 lines). |

---

### 3.7 External Platform Domain

#### ExternalPlatformRepositoryImpl

**File:** `external_platform_repo_impl.go`
**Models:** `ExternalPlatform`, `Member`, `MemberExternalPlatformRole`, `Role`, `Permission`, `Webhook`, `APIKey`, `BlockchainCurrency`, `Blockchain`
**Total Lines:** ~370

| Method | Lines | Description |
|--------|-------|-------------|
| `GrantPermissions` | 29 | Grants role-based permissions to a member for a specific external platform. Creates `MemberExternalPlatformRole` entries. |
| `RevokePermissions` | 28 | Revokes role-based permissions from a member for a platform. |
| `Create` | 9 | Inserts a new external platform. |
| `GetAll` | 35 | Lists all external platforms accessible to a member. Filters based on member's platform role assignments. Preloads associations. |
| `GetAllWithDetails` | 6 | Lists platforms with full detail preloads. |
| `GetById` | 29 | Retrieves platform by ID with full association preloading (APIKeys, ExternalPlatformBlockchainCurrencies, DefaultPaymentBlockchain, etc.). Includes permission-based access check. |
| `GetByIds` | 33 | Batch platform lookup by IDs with permission filtering. |
| `SaveProject` | 29 | Updates an external platform record. Handles updating associated records. |
| `GetByIdWithoutPermissionCheck` | 13 | Platform lookup bypassing permission checks (for internal use). |
| `GetWebhooksInternalOnly` | 9 | Lists webhooks for a platform without permission checks. |
| `GetWebhooks` | 42 | Lists webhooks with permission-based filtering. Complex query with member role joins. |
| `CreateWebhook` | 15 | Creates a webhook for a platform. |
| `UpdateWebhook` | 15 | Updates a webhook configuration. |
| `GetWebhookCount` | 6 | Counts webhooks for a platform. |
| `DeleteWebhook` | 4 | Soft-deletes a webhook. |
| `GetWebhookID` | 43 | Retrieves a webhook by ID with permission-based access checks. Complex join query with member roles. |

**Query Patterns:**
- Permission-based filtering through JOINs on member_external_platform_roles
- Heavy association preloading for API responses
- Role-based access control at the query level

---

#### ExternalPlatformBlockchainCurrencyRepositoryImpl

**File:** `external_platform_blockchain_currency_repo_impl.go`
**Models:** `ExternalPlatformBlockchainCurrency`, `BlockchainCurrency`
**Total Lines:** ~58

| Method | Lines | Description |
|--------|-------|-------------|
| `ReplaceSupportedNetworkAndCurrencies` | 35 | Replaces all supported blockchain currencies for a platform. Deletes existing mappings, then bulk-creates new ones. Uses `Unscoped()` to hard-delete. |
| `GetByExternalPlatformID` | 7 | Lists all blockchain currencies supported by a platform. |

**Query Patterns:**
- `Unscoped().Delete()` for hard deletion (not soft delete) of junction table records
- Replacement pattern: delete all + create all in one operation

---

#### ExternalPlatformWalletBlockchainFamilyRepositoryImpl

**File:** `external_platform_wallet_blockchain_family_repo_impl.go`
**Models:** `ExternalPlatformWalletBlockchainFamily`, `Wallet`, `BlockchainFamily`, `ExternalPlatform`
**Total Lines:** ~129

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateProjectWalletMappings` | 17 | Creates associations between platforms and wallets for specific blockchain families. |
| `CreateOrReplaceProjectWalletMappings` | 27 | Replaces existing wallet-family mappings for a platform. Hard-deletes old mappings, creates new ones. |
| `GetByExternalPlatformID` | 26 | Lists wallet-family mappings for a platform. Preloads Wallet, BlockchainFamily. |
| `GetExternalPlatformWalletBlockchainFamily` | 19 | Detailed lookup with additional Wallet associations (WalletXpubs, WalletScws). |
| `GetExternalPlatformWallets` | 16 | Lists wallets associated with a platform. |

---

### 3.8 Identity & Auth Domain

#### AuthRepositoryImpl

**File:** `auth_repo_impl.go`
**Models:** `Member`, `APIKey`, `Role`, `ExternalPlatform`
**Total Lines:** ~92

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateMember` | 10 | Creates a new member record. |
| `CreateAPIKey` | 11 | Creates a new API key associated with a member and platform. |
| `UpdateUser` | 9 | Updates member fields. |
| `UpdateOrCreateMember` | 10 | Upsert for member records using email as the unique key. |
| `GetMemberByEmail` | 10 | Finds member by email. Preloads Roles and Roles.Permissions. |
| `GetMemberById` | 10 | Finds member by ID with role preloads. |
| `GetAPIKeys` | 9 | Lists API keys for a member. |
| `GetMemberByApiKey` | 12 | Finds member associated with an API key. Joins through api_keys table. Preloads Role. |

---

#### AuthRefreshTokenRepositoryImpl

**File:** `auth_refresh_token_repo_impl.go`
**Models:** `AuthRefreshToken`, `Member`
**Total Lines:** ~45

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 4 | Inserts a new refresh token. |
| `GetAuthTokenByToken` | 10 | Finds refresh token by token string. Preloads Member. Filters for non-revoked tokens. |
| `UpdateAuthTokenLastUsed` | 6 | Updates the last_used_at timestamp. |
| `RevokeAuthToken` | 6 | Sets the revoked_at timestamp on a token. |
| `RevokeAllAuthTokensForMember` | 6 | Revokes all refresh tokens for a member. Bulk update. |
| `DeleteExpiredAuthTokens` | 13 | Hard-deletes tokens that are expired or revoked. Uses `Unscoped()` with time-based conditions. |

---

#### APIKeyRepositoryImpl

**File:** `api_key_repo_impl.go`
**Models:** `APIKey`, `Member`, `ExternalPlatform`, `Role`
**Total Lines:** ~109

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 6 | Creates a new API key. Supports `WithPreload` functional options. |
| `Update` | 6 | Updates an API key. Supports `WithPreload` functional options. |
| `Delete` | 4 | Soft-deletes an API key. |
| `GetByAPIKey` | 14 | Finds by key string. Preloads Member, ExternalPlatform, Role. |
| `GetByID` | 10 | Finds by ID with preloads. |
| `GetByExternalPlatformId` | 22 | Lists API keys for a platform with member and role preloads. |
| `GetExternalPlatformApiKeys` | 17 | Lists API keys with additional filtering and preloading. |
| `DeactivateByMemberID` | 9 | Deactivates all API keys for a member. Bulk status update. |

**Unique Pattern:** Uses functional options (`WithPreload`) defined in `option.go` for configurable preloading behavior in Create/Update operations.

---

#### MemberRepositoryImpl

**File:** `member_repo_impl.go`
**Models:** `Member`, `Role`, `Permission`, `MemberExternalPlatformRole`, `MemberRole`, `APIKey`, `ExternalPlatform`
**Total Lines:** ~542

| Method | Lines | Description |
|--------|-------|-------------|
| `GetMemberCount` | 10 | Counts members, optionally filtered by type. |
| `GetAllInternalMembers` | 172 | **Complex paginated query.** Retrieves internal members with role information, platform associations, and activity data. Builds dynamic query with multiple optional filters (email, name, role, status). Supports sorting by various fields. |
| `GetFirst20MembersWithNoRolesOrUserRole` | 20 | Finds members without assigned roles or with only "user" role. Used for onboarding/setup. |
| `CreateRootMember` | 32 | Creates the root/admin member. Assigns admin role. Complex initialization logic. |
| `CreateInternalMember` | 56 | Transaction-wrapped member creation. Creates member, assigns role, creates platform association. 54-line inner closure. |
| `UpdateMember` | 10 | Updates member fields. |
| `DeleteMember` | 67 | **Complex deletion.** Validates member can be deleted (not root, not last admin). Deactivates API keys, removes platform roles, soft-deletes member. Transaction-wrapped. |
| `UpdateOrCreateMember` | 10 | Upsert using email as unique key. |
| `GetInternalMemberByEmail` | 20 | Finds internal member by email with role and platform preloads. |
| `GetMemberById` | 10 | Simple ID lookup. |
| `GetMember` | 4 | Generic member lookup by conditions. |
| `GetInternalMemberById` | 10 | Finds internal member by ID with role preloads. |
| `GetRootMember` | 13 | Finds the root/admin member. |
| `GetInternalMemberByIds` | 15 | Batch ID lookup for internal members. |
| `GetInternalMembersByEmails` | 15 | Batch email lookup for internal members. |
| `Activate` | 41 | Transaction-wrapped member activation. Updates status, sets password hash, may assign initial role. |

**Query Patterns:**
- Dynamic WHERE clause construction with optional filters
- Complex JOINs for role-based queries
- Subqueries for "members with no roles" detection
- Transaction-wrapped multi-model operations

---

#### MemberRoleRepositoryImpl

**File:** `member_role_repo_impl.go`
**Models:** `MemberRole`, `Role`, `Member`
**Total Lines:** ~107

| Method | Lines | Description |
|--------|-------|-------------|
| `AssignAdminRoleToMember` | 56 | Assigns admin role to a member. Validates role exists, checks for existing assignment, creates association. |
| `GetRoleNameByMemberID` | 13 | Retrieves the role name assigned to a member. Uses JOIN query. |
| `RemoveAdminRoleFromMember` | 38 | Removes admin role from a member. Validates at least one admin remains. Hard-deletes the role association. |

---

#### MemberExternalPlatformRoleRepositoryImpl

**File:** `member_external_platform_role_repo_impl.go`
**Models:** `MemberExternalPlatformRole`, `Member`, `ExternalPlatform`, `Role`
**Total Lines:** ~485

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Creates a platform role assignment. |
| `GetByMemberId` | 17 | Lists platform roles for a member with preloads. |
| `CreateOrUpdateMemberWithUserRole` | 119 | **Complex transaction.** Creates a new member (or finds existing) and assigns a user role for a platform. 113-line inner closure. Handles duplicate detection, role assignment, and platform linking. |
| `CreateOrUpdateMemberWithoutUserRole` | 119 | Similar to above but without automatic user role assignment. Used for API key-based member creation. 114-line inner closure. |
| `AssignPlatformRolesToMember` | 52 | Assigns multiple platform roles to a member. Validates roles exist, checks for duplicates. |
| `UpdatePlatformRolesForMember` | 61 | Replaces all platform roles for a member. Deletes existing, creates new assignments. |
| `GetByMemberIdAndExternalPlatformID` | 10 | Finds platform role by member and platform IDs. |
| `DeleteAllPlatformRolesForMember` | 9 | Removes all platform role assignments for a member. |
| `DeleteIndividualPlatformRole` | 11 | Removes a specific platform role assignment. |
| `GetMemberByCustomerIDAndExternalPlatformID` | 20 | Finds member by customer_id within a specific platform context. |
| `GetMembersByExternalPlatformIDs` | 25 | Lists members across multiple platforms. Paginated with role preloading. |

**Transaction Boundaries:**
- `CreateOrUpdateMemberWithUserRole` -- 119-line transaction
- `CreateOrUpdateMemberWithoutUserRole` -- 119-line transaction

---

#### OTPRepositoryImpl

**File:** `otp_repo_impl.go`
**Models:** `OTP`
**Total Lines:** ~109

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateOTP` | 13 | Creates a new OTP record with code, purpose, entity_id, and expiry. |
| `UpdateOTP` | 5 | Updates an OTP record. |
| `GetValidOTP` | 17 | Finds a valid (non-expired, non-verified) OTP for an entity and purpose. |
| `GetLastOtp` | 15 | Retrieves the most recently created OTP for an entity and purpose. |
| `MarkVerified` | 21 | Marks an OTP as verified. Updates verified flag, verified_at timestamp. |
| `InvalidateNonVerifiedOTPs` | 7 | Bulk invalidation of all non-verified OTPs for an entity. |
| `IncrementOTPAttempts` | 21 | Increments the attempt counter for an OTP. Used for rate limiting/lockout. |
| `IsVerified` | 9 | Checks if a verified OTP exists for a given entity and purpose. |

---

#### PermissionRepositoryImpl

**File:** `permission_repo_impl.go`
**Models:** `Permission`
**Total Lines:** ~36

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Creates a new permission record. |
| `GetByNames` | 27 | Retrieves permissions by a list of names. Uses IN clause. |

---

#### RoleRepositoryImpl

**File:** `role_repo_impl.go`
**Models:** `Role`, `Permission`, `RolePermission`
**Total Lines:** ~100

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 10 | Creates a new role record. |
| `UpdateRolePermissions` | 30 | Replaces all permissions for a role. Clears existing associations, assigns new ones via GORM Association mode. |
| `GetAllInternalMemberRoles` | 14 | Lists roles available for internal members. Preloads Permissions. |
| `GetByName` | 20 | Finds role by name. Preloads Permissions. |
| `GetByNames` | 18 | Batch role lookup by names. |
| `GetByIds` | 10 | Batch role lookup by IDs. |
| `GetRoleNameByID` | 7 | Simple ID to name lookup. |

---

#### WebsocketTokenRepositoryImpl

**File:** `websocket_token_repo_impl.go`
**Models:** `WebSocketToken`, `Member`
**Total Lines:** ~24

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateToken` | 10 | Creates a new websocket authentication token. |
| `GetTokenByToken` | 14 | Validates a websocket token. Checks expiry. Preloads Member. |

---

### 3.9 Payment Domain

#### PaymentRepositoryImpl

**File:** `payment_repo_impl.go`
**Models:** `PaymentRequest`, `Member`, `Currency`, `Blockchain`, `Deposit`, `ExternalPlatform`
**Total Lines:** ~343

| Method | Lines | Description |
|--------|-------|-------------|
| `GetPaymentRequestById` | 19 | Retrieves payment request by ID. Preloads Currency, Blockchain, Deposit, Member, ExternalPlatform. |
| `CancelPreviousOpenPaymentRequestsByMemberId` | 28 | Cancels all open payment requests for a member on a specific platform. Bulk status update. |
| `GetClosedPaymentsByMemberId` | 16 | Lists completed payment requests for a member. |
| `CreatePaymentRequest` | 23 | Creates a new payment request and returns it with full preloads. |
| `Summary` | 32 | Aggregates payment summary statistics. Uses `GROUP BY status` with SUM of amounts. Returns counts and totals per status. |
| `Search` | 142 | **Complex search method.** Builds dynamic query with 15+ optional filters (status, date range, currency, blockchain, amount range, invoice_id, reference_id, member). Supports pagination and sorting. Performs COUNT query for total, then fetches paginated results with full preloads. |
| `GetPaymentRequestByReferenceId` | 30 | Finds payment request by external reference ID and platform ID. |
| `GetPaymentRequest` | 9 | Generic payment request lookup by conditions. |
| `GetTotalRevenueFromReferredUsers` | 17 | Calculates total revenue generated from referred users. Joins payment_requests with members, sums amount_in_usd where member was referred by a specific user. |

**Query Patterns:**
- Dynamic filter building with 15+ optional WHERE conditions
- SUM/COUNT aggregations for summaries
- Cross-model JOINs for referral revenue calculation
- Two-phase query: count first, then paginated data fetch

---

#### PaymentChannelRepositoryImpl

**File:** `payment_channel_repo_impl.go`
**Models:** `PaymentChannel`, `DisabledPaymentChannelProject`
**Total Lines:** ~152

| Method | Lines | Description |
|--------|-------|-------------|
| `GetPaymentChannels` | 17 | Lists payment channels with optional filters applied via `ApplyPaymentChannelFilters`. |
| `GetPaymentChannelByID` | 15 | Retrieves payment channel by ID. Preloads `DisabledPaymentChannelProjects`. |
| `GetPaymentChannelAPIKey` | 15 | Retrieves the API key for a payment channel. |
| `UpdatePaymentChannel` | 41 | Transaction-wrapped update. 36-line inner closure. Handles updating channel fields and associated configuration/metadata JSON. |
| `applyPaymentChannelUpdates` | 31 | Private helper for applying partial updates to payment channel fields. |

**Standalone Filter Function:**

`ApplyPaymentChannelFilters` (54 lines, in `payment_channel_repo.go`) -- Applies filtering by channel_type, status, name, is_default, and disabled state per project.

---

#### PaymentChannelProjectRepositoryImpl

**File:** `payment_channel_project_repo_impl.go`
**Models:** `DisabledPaymentChannelProject`, `PaymentChannel`, `ExternalPlatform`, `Member`
**Total Lines:** ~275

| Method | Lines | Description |
|--------|-------|-------------|
| `lockRowWithTimeout` | 49 | **Pessimistic locking helper.** Acquires a row-level lock with a configurable timeout. Uses `SET LOCAL lock_timeout`. Falls back to retry on lock failure. |
| `isLockError` | 13 | Checks if an error is a PostgreSQL lock timeout error (error code 55P03). |
| `lockPaymentChannelSafely` | 37 | Safely acquires lock on a payment channel row using `lockRowWithTimeout`. Uses `FOR UPDATE` clause. |
| `lockDisabledMappingSafely` | 41 | Safely acquires lock on a disabled payment channel project mapping. |
| `UpdatePaymentChannelProjects` | 95 | **Complex transaction with locking.** 90-line inner closure. Enables or disables payment channels for a project. Acquires pessimistic locks on both the payment channel and the disabled mapping rows before making changes. Handles both enable (delete mapping) and disable (create mapping) cases. |
| `GetDisabledPaymentChannelProjectsByProjectID` | 16 | Lists disabled payment channels for a project. Preloads PaymentChannel and DisabledByMember. |

**Locking Pattern Detail:**
```
1. Begin transaction
2. SET LOCAL lock_timeout = '5s'
3. SELECT ... FOR UPDATE (lock payment channel row)
4. SELECT ... FOR UPDATE (lock disabled mapping row, if exists)
5. Perform enable/disable operation
6. Commit transaction
```

---

#### PaymentsAppRepositoryImpl

**File:** `payments_app_repo_impl.go`
**Models:** `PaymentsApp`, `ExternalPlatform`
**Total Lines:** ~75

| Method | Lines | Description |
|--------|-------|-------------|
| `GetPaymentsApps` | 21 | Lists payment apps with optional project_id filter. Preloads Project. |
| `CreatePaymentsApp` | 22 | Creates a new payments app configuration. Validates project exists. |
| `UpdatePaymentsApp` | 22 | Updates payments app settings (sponsorship percentage, cut-off). |
| `GetPaymentsAppByID` | 12 | Retrieves payments app by ID with Project preload. |

---

#### OnramperPaymentsRepositoryImpl

**File:** `onramper_payments_repo_impl.go`
**Models:** `PaymentRequest`, `Member`, `Deposit`
**Total Lines:** ~82

| Method | Lines | Description |
|--------|-------|-------------|
| `GetOnramperPayments` | 34 | Retrieves payment requests related to onramper (card-to-crypto) payments. Complex query with date range and status filters. |
| `GetOnramperPaymentsMetrics` | 48 | Aggregates onramper payment metrics. Uses SUM, COUNT, and GROUP BY for daily/weekly/monthly breakdown. Returns revenue and volume metrics. |

---

### 3.10 Sweep Domain

#### SweepRepositoryImpl

**File:** `sweep_repo_impl.go`
**Models:** `Sweep`, `Currency`, `Blockchain`
**Total Lines:** ~151

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 9 | Inserts a new sweep record. |
| `Update` | 9 | Updates a sweep record. |
| `BulkCreate` | 27 | Transaction-wrapped batch creation of sweep records. 27-line inner closure handles batch insert with `CreateInBatches`. |
| `GetByToAddressAndStatus` | 11 | Finds sweeps by destination address and status. |
| `GetLastSweepByBlockchainCodeAndCurrency` | 11 | Gets the most recent sweep for a blockchain-currency pair. Orders by `created_at DESC`. |
| `GetAllSweeps` | 22 | Paginated sweep listing with filters (status, blockchain_code, currency_code). |
| `GetConfirmingSweeps` | 14 | Lists sweeps in "confirming" status for a specific blockchain. |
| `GetSweepByID` | 10 | Retrieves sweep by ID with Currency and Blockchain preloads. |
| `UpdateSweepStatus` | 11 | Updates sweep status field. |
| `UpdateSweepAfterConfirmation` | 10 | Updates sweep fields after blockchain confirmation (block_number, status, timestamp). |

---

#### SweepTransactionRepositoryImpl

**File:** `sweep_transaction_repo_impl.go`
**Models:** `SweepTransaction`, `AccountAddress`, `UTXO`, `AddressPool`, `Wallet`, `WalletXpub`, `Currency`, `Blockchain`, `BlockchainCurrency`
**Total Lines:** ~720

| Method | Lines | Description |
|--------|-------|-------------|
| `GetAll` | 11 | Lists all sweep transactions. |
| `GetSweepTransactions` | 108 | **Complex paginated query.** Retrieves sweep transactions with extensive preloading (AccountAddresses, UTXOs, Wallet, Currency, Blockchain). Supports filtering by status, blockchain_code, currency_code. Includes computed fields (number_of_utxos, total_amount). |
| `GetAllWithUTXOInfo` | 40 | Retrieves sweep transactions with aggregated UTXO information. Uses subquery for UTXO count and total amount. |
| `UpdateSweepTransaction` | 54 | Transaction-wrapped update. 49-line inner closure. Updates sweep transaction fields and associated account address/UTXO statuses. |
| `AddAddressToTransaction` | 5 | Public entry point delegating to `addAddressToTransaction`. |
| `addAddressToTransaction` | 170 | **Highly complex method.** Transaction-wrapped (87-line inner closure). Adds an EVM address to a sweep transaction: validates address eligibility, checks blockchain currency configuration, verifies balance meets minimum thresholds, locks the address by updating its sweeping_transaction_id, and calculates totals. Enforces batch size limits from blockchain_currency settings. |
| `calculateTotalBalanceAddresses` | 19 | Calculates total balance across addresses in a sweep transaction. |
| `AddUTXOToTransaction` | 5 | Public entry point delegating to `addUTXOToTransaction`. |
| `addUTXOToTransaction` | 198 | **Most complex method in this repo.** Adds a UTXO to a Bitcoin sweep transaction: validates UTXO eligibility, checks it belongs to the correct wallet, verifies it is unspent, links it to the sweep transaction, and updates running totals. Extensive validation and error handling. |
| `GetSweepTransactionIDsByWalletID` | 24 | Finds sweep transaction IDs associated with a wallet. |
| `GetWalletDetails` | 20 | Retrieves wallet details for sweep transaction display. |
| `GetSweepTransactionInfoWithUTXOAndTotal` | 30 | Gets sweep transaction with aggregated UTXO count and total amount. |

**Query Patterns:**
- Subqueries for aggregated UTXO counts and totals
- Pessimistic-style locking via sweeping_transaction_id assignment
- Batch size enforcement from blockchain_currency configuration
- Complex multi-table validation queries

---

#### SweepUTXORepositoryImpl

**File:** `sweep_utxo_repo_impl.go`
**Models:** `UTXO`, `SweepTransaction`, `AccountAddress`, `AddressPool`, `Sweep`, `Withdraw`, `Asset`, `Liability`, `Revenue`, `Expense`, `Account`
**Total Lines:** ~457

| Method | Lines | Description |
|--------|-------|-------------|
| `UpdateUTXOsStatusToSpentAndSweepTxId` | 99 | Transaction-wrapped (87-line inner closure). Marks UTXOs as spent and links them to a sweep transaction. Updates associated account addresses and address pool entries. |
| `ProcessAccountingForBTCSweep` | 206 | **Very complex accounting method.** Processes the full accounting lifecycle for a Bitcoin sweep operation. Within a single transaction: updates sweep transaction status, updates associated UTXOs, creates Liability debit entries (reducing obligations), creates Asset credit entries (funds moved), creates Revenue entries for platform fees, creates Expense entries for mining fees, updates account address balances. Handles the Payminto fee calculation and cold wallet destination. |

**Helper Functions:**

| Function | Lines | Description |
|----------|-------|-------------|
| `updateSweepTransactionsAndRespectiveUTXOsToCompleted` | 69 | Marks sweep transactions and their UTXOs as completed. Batch update with status transitions. |

---

#### UTXORepositoryImpl

**File:** `utxo_repo_impl.go`
**Models:** `UTXO`, `Deposit`, `AddressPool`, `SweepTransaction`
**Total Lines:** ~129

| Method | Lines | Description |
|--------|-------|-------------|
| `GetUnspentUTXOs` | 28 | Retrieves unspent UTXOs for a given blockchain and currency. Preloads Deposit and AddressPool. Supports optional wallet_id filtering. |
| `UpdateUTXOsStatusToSpent` | 24 | Marks UTXOs as spent. Batch status update. |
| `GetAllPendingUTXOsForSweepTransaction` | 29 | Retrieves UTXOs linked to a pending sweep transaction. Preloads AddressPool and Deposit associations. |
| `IdentifyOurUTXOsFromList` | 48 | Given a list of transaction outputs, identifies which ones belong to our address pool. Matches by address and transaction hash. Complex query with OR conditions across multiple UTXO identifiers. |

---

### 3.11 Wallet Domain

#### WalletRepositoryImpl

**File:** `wallet_repo_impl.go`
**Models:** `Wallet`, `WalletXpub`, `WalletSCW`, `WalletFunction`, `SecretsVault`, `BlockchainFamily`, `Blockchain`, `Currency`, `Member`, `ExternalPlatformWalletBlockchainFamily`
**Total Lines:** ~745

| Method | Lines | Description |
|--------|-------|-------------|
| `GetWalletByID` | 24 | Retrieves wallet by ID. Preloads WalletFunctions, WalletXpubs, WalletScws, BlockchainFamily, Blockchain, Currency. |
| `GetWallets` | 19 | Lists wallets using `ApplyWalletQuery` dynamic filter builder. |
| `GetWalletSCWs` | 19 | Lists wallet SCW entries using `ApplyWalletScwQuery`. |
| `EnrichWalletDetails` | 92 | **Complex enrichment method.** Post-processes wallet data by adding computed fields: balance aggregation from WalletXpubs and WalletScws, explorer URL construction, fund sweeper address href, and deletion safety check (CanDelete). Iterates all wallet associations. |
| `CreateWallet` | 8 | Inserts a new wallet record. |
| `CreateWalletSCW` | 9 | Inserts a new wallet SCW record. |
| `CreateWalletWithXpubs` | 31 | Transaction-wrapped. Creates wallet and its associated XPUB records atomically. |
| `CreateWalletWithSCWs` | 45 | Transaction-wrapped (47-line inner closure). Creates wallet with SCW associations. Encrypts private key before storage. |
| `Delete` | 79 | **Complex deletion.** Validates wallet can be deleted (not in use by any project). Checks ExternalPlatformWalletBlockchainFamily associations. Cascades deletion to WalletFunctions, WalletXpubs, WalletScws. Transaction-wrapped. |
| `Update` | 9 | Updates wallet fields. |
| `UpdateWalletXPUB` | 9 | Updates wallet XPUB fields. |
| `UpdateSecretsVaultActivities` | 33 | Updates wallet function (activity) associations. Replaces existing associations. |
| `GetAllActiveWithVaultActivities` | 24 | Lists active wallets with their vault/function activities preloaded. |
| `GetByNames` | 22 | Finds wallets by name list. Supports type filtering. |
| `GetWalletByIds` | 15 | Batch ID lookup with preloads. |
| `SetDefaultWallet` | 78 | Transaction-wrapped. Sets a wallet as the default for its blockchain family. Unsets previous default. Validates wallet exists and is eligible. |

**Standalone Filter Functions:**

| Function | Lines | Description |
|----------|-------|-------------|
| `ApplyWalletQuery` | 95 | Dynamic query builder with 10+ optional filters (name, family, type, status, blockchain_code, etc.). |
| `ApplyWalletScwQuery` | 33 | Dynamic query builder for wallet SCW records. |

**Encryption in Wallet Repo:**

| Function | Lines | Description |
|----------|-------|-------------|
| `decryptRSAEncryptedWithNonce` | 22 | Decrypts RSA-encrypted data combined with a nonce. Used for wallet private key retrieval. |

---

#### WalletFunctionRepositoryImpl

**File:** `wallet_function_repo_impl.go`
**Models:** `WalletFunction`, `Wallet`
**Total Lines:** ~89

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 8 | Creates a new wallet function record. |
| `GetByActivityType` | 20 | Finds wallet functions by activity type. Preloads Wallet. |
| `GetByActivityTypes` | 20 | Batch lookup by multiple activity types. |
| `GetAllVaultActivities` | 19 | Lists all vault activities. Preloads Wallet. |
| `CreateWalletToFunctionsAssociation` | 28 | Creates association between wallet and functions. Handles replacement of existing associations. |

---

### 3.12 Webhook Domain

#### WebhookRepositoryImpl

**File:** `webhook_repo_impl.go`
**Models:** `Webhook`, `ExternalPlatform`, `PaymentRequest`
**Total Lines:** ~72

| Method | Lines | Description |
|--------|-------|-------------|
| `UpdatePayoutWebhook` | 45 | Transaction-wrapped (34-line inner closure). Updates webhook delivery status for payout-related events. Finds relevant webhooks, updates delivery logs, and manages retry scheduling. |

---

#### WebhookDeliveryLogRepositoryImpl

**File:** `webhook_delivery_log_repo_impl.go`
**Models:** `WebhookDeliveryLog`, `PaymentRequest`, `Webhook`
**Total Lines:** ~142

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateDeliveryLogsInTx` | 30 | Creates webhook delivery log entries within an existing transaction context. Used by other repositories when they need to trigger webhook notifications as part of a larger transaction. |
| `GetWebhookDeliveries` | 33 | Paginated query for webhook delivery logs. Supports filtering by status, webhook_id, payment_request_id. Preloads PaymentRequest and Webhook. |
| `UpdateDeliveryAttempt` | 36 | Transaction-wrapped. Updates delivery attempt results: response code, error message, attempt count, next retry time. Implements exponential backoff scheduling for retries. |
| `BulkUpdateStatusInTx` | 14 | Batch status update for delivery logs within an existing transaction. |

---

### 3.13 Withdrawal Domain

#### WithdrawalRepoImpl

**File:** `withdrawal_repo_impl.go`
**Models:** `Withdrawal`, `Member`, `Account`, `Currency`, `Blockchain`, `ExternalPlatform`
**Total Lines:** ~517

| Method | Lines | Description |
|--------|-------|-------------|
| `CreatePayoutRequest` | 31 | Creates a payout (withdrawal) request. Validates account balance, locks funds. |
| `CreatePayoutRequestForMerchant` | 23 | Creates a merchant-initiated payout request. |
| `UpdateWithdrawalStatus` | 193 | **State machine method.** Not a single method but contains multiple status transition handlers. Validates transition legality, updates status. |
| `LockAndUpdateWithdrawalForMerchant` | 37 | Transaction-wrapped with pessimistic locking. Acquires row lock on the withdrawal, validates current status allows transition, then updates. |
| `LockAndUpdateWithdrawal` | 45 | Transaction-wrapped with pessimistic locking (75-line inner closure). Similar to merchant version but for admin-initiated updates. Uses `FOR UPDATE` row locking. |
| `GetPendingWithdrawals` | 24 | Lists withdrawals in "pending" status. Preloads Member, Currency, Blockchain. |
| `GetWithdrawals` | 24 | Paginated withdrawal listing with status filtering. |
| `GetWithdrawalsFromMerchant` | 22 | Lists withdrawals created by merchant API keys. Filters by external_platform_id. |
| `MarkWithdrawalFailed` | 54 | Transaction-wrapped (54-line inner closure). Marks a withdrawal as failed. Unlocks previously locked funds by restoring account balance. |
| `UpdateWithdrawalRetryCount` | 63 | Transaction-wrapped (64-line inner closure). Increments retry count and updates next retry time. Implements maximum retry logic. |
| `Create` | 5 | Simple withdrawal record creation. |

**Helper Functions (package-level):**

| Function | Lines | Description |
|----------|-------|-------------|
| `unlockFundsForRejectedCancelledOrFailedWithdrawal` | 34 | Restores locked funds to the account balance when a withdrawal is cancelled, rejected, or fails. |
| `validateRequiredFields` | 36 | Validates that required fields are set before status transitions. Returns descriptive error messages. |

**State Machine (Withdrawal Status):**
```
pending -> approved -> processing -> sent -> completed
       \-> rejected                      \-> failed
       \-> cancelled
```

**Map Initializers:**
- `map.init0` (16 lines) -- Initializes valid status transitions map
- `map.init1` (5 lines) -- Initializes required fields per status map

---

#### WithdrawalProcessingRepoImpl

**File:** `withdrawal_processing_repo_impl.go`
**Models:** `Withdrawal`, `Account`, `Asset`, `Liability`, `Revenue`, `Expense`, `Webhook`, `WebhookDeliveryLog`
**Total Lines:** ~606

| Method | Lines | Description |
|--------|-------|-------------|
| `CreatePayoutRequest` | 59 | Transaction-wrapped (59-line inner closure). Creates a payout request with full accounting: validates account balance, locks funds by decrementing available balance and incrementing locked amount, creates the withdrawal record. |
| `ProcessPayoutAccounting` | 69 | Transaction-wrapped (71-line inner closure). Processes accounting when a payout is approved: creates Liability debit (obligation to pay), Asset credit (funds earmarked for withdrawal). |
| `ProcessPayoutAccountingForMerchant` | 40 | Transaction-wrapped (42-line inner closure). Merchant-specific payout accounting. Similar to ProcessPayoutAccounting but with merchant fee handling. |
| `ProcessAccountingForWithdrawalSent` | 222 | **Very complex method (second longest across all repos).** Transaction-wrapped (223-line inner closure). Processes complete accounting when a withdrawal transaction is confirmed on-chain. Creates Asset debit (funds left platform), Liability credit (obligation fulfilled), Revenue entries for withdrawal fees, Expense entries for blockchain gas costs. Updates withdrawal status to "completed". Triggers webhook delivery for merchant notification. |
| `ProcessAccountingForWithdrawalSentOfMerchantPayout` | 173 | Transaction-wrapped (158-line inner closure). Merchant-specific variant of above. Handles additional merchant fee splitting and platform fee calculation. Updates webhook delivery logs for merchant callbacks. |

**Accounting Flow:**
```
1. CreatePayoutRequest: Lock funds (Account.Locked += amount, Account.Balance -= amount)
2. ProcessPayoutAccounting: Liability debit + Asset credit
3. ProcessAccountingForWithdrawalSent: Asset debit + Liability credit + Revenue credit + Expense debit
```

---

### 3.14 Miscellaneous Domain

#### RecipientRepositoryImpl

**File:** `recipient_repo_impl.go`
**Models:** `Recipient`, `Member`, `Blockchain`
**Total Lines:** ~80

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 7 | Creates a new recipient (withdrawal address book entry). |
| `GetRecipients` | 11 | Lists recipients for a member. Preloads Blockchain. |
| `Update` | 7 | Updates recipient details. |
| `Activate` | 27 | Transaction-wrapped (27-line inner closure). Activates a recipient after verification. Updates status. |
| `DeleteRecipient` | 25 | Transaction-wrapped (13-line inner closure). Validates recipient can be deleted, then soft-deletes. |

---

#### RewardAccountingRepositoryImpl

**File:** `reward_accounting_repo_impl.go`
**Models:** `ProcessedReward`, `AccountReward`, `Account`, `Asset`, `Liability`, `Revenue`
**Total Lines:** ~258

| Method | Lines | Description |
|--------|-------|-------------|
| `GetLastProcessedRewardID` | 8 | Returns the highest processed reward ID. Used for incremental processing. |
| `GetFailedProcessedRewards` | 16 | Lists rewards that failed processing. |
| `ProcessReward` | 134 | **Complex accounting transaction.** 136-line inner closure. Processes a referral reward: validates reward is not already processed, finds or creates an AccountReward entry, creates Asset debit and Liability credit entries for the reward amount, updates AccountReward balance, creates ProcessedReward record. Handles errors by calling `markRewardFailed`. |
| `markRewardFailed` | 36 | Records a failed reward processing attempt with failure reason. |
| `createAccountingEntries` | 31 | Helper to create paired Asset and Liability entries for rewards. |

---

#### InternalBlockchainTransactionRepositoryImpl

**File:** `internal_blockchain_transaction_repo_impl.go`
**Models:** `InternalBlockchainTransaction`, `Member`, `Currency`, `Blockchain`, `Deposit`
**Total Lines:** ~38

| Method | Lines | Description |
|--------|-------|-------------|
| `Create` | 10 | Records an internal blockchain transaction (e.g., gas transfers, fee payments). |
| `GetByToAddressTransactionTypeAndStatus` | 9 | Finds internal transactions by destination address, type, and status. |
| `GetByTxHash` | 9 | Finds internal transaction by transaction hash. |
| `GetByUniqueTxHash` | 9 | Finds internal transaction by unique transaction hash. |
| `Update` | 7 | Updates an internal transaction record. |

---

## 4. Accounting Integration Pattern

### Double-Entry Ledger System

The repository layer implements a strict double-entry accounting system across four account types:

| Account Type | Model | Description |
|-------------|-------|-------------|
| **Asset** | `models.Asset` | Platform-controlled funds (debit = increase, credit = decrease) |
| **Liability** | `models.Liability` | Obligations to users (debit = decrease, credit = increase) |
| **Revenue** | `models.Revenue` | Platform income from fees (debit = decrease, credit = increase) |
| **Expense** | `models.Expense` | Platform costs like gas fees (debit = increase, credit = decrease) |

### Accounting Entry Structure

Each accounting entry contains:
- `MemberID` -- The member this entry relates to (nullable for platform-level entries)
- `AccountID` -- The account this entry belongs to
- `CurrencyID` / `CurrencyCode` -- The currency of the entry
- `Debit` / `Credit` -- Only one is non-zero per entry (decimal(38,18))
- `ReferenceType` / `ReferenceID` -- Polymorphic reference to the source event (deposit, withdrawal, sweep, etc.)
- `Code` -- Accounting code for categorization

### Repositories with Accounting Logic

The following repositories create accounting entries as part of their operations:

| Repository | Accounting Methods | Events |
|-----------|-------------------|--------|
| `AccountRepositoryImpl` | ProcessBlockchainInternalTransferEthereumGasFees, ProcessERC20Sweep, ProcessDuplicateDepositsAsExpense | Gas fees, sweep operations, duplicate handling |
| `AddressDeploymentRepositoryImpl` | ProcessAddressDeploymentAccounting | Contract deployment costs |
| `BlockchainRepositoryImpl` | CreateDepositReceived, ProcessPaymentRequestAccounting, ProcessWithdrawsAccounting, WithdrawResponse | Deposit receipt, payment completion, withdrawal confirmation |
| `DepositRepositoryImpl` | ProcessPendingDeposit | Deposit processing |
| `RewardAccountingRepositoryImpl` | ProcessReward | Referral rewards |
| `SweepUTXORepositoryImpl` | ProcessAccountingForBTCSweep | Bitcoin sweep operations |
| `WithdrawalProcessingRepoImpl` | CreatePayoutRequest, ProcessPayoutAccounting, ProcessAccountingForWithdrawalSent, ProcessAccountingForWithdrawalSentOfMerchantPayout | Payout lifecycle |

### Accounting Invariant

For every financial event, the total debits must equal total credits:

```
Deposit received:
  Asset DEBIT  = amount      (platform now holds funds)
  Liability CREDIT = amount  (platform owes user)

Withdrawal sent:
  Liability DEBIT = amount   (obligation fulfilled)
  Asset CREDIT = amount      (funds left platform)
  Revenue CREDIT = fee       (platform earned fee)
  Expense DEBIT = gas_cost   (platform paid gas)

Sweep completed:
  Liability DEBIT = amount   (user obligation transferred)
  Asset CREDIT = amount      (funds moved to cold wallet)
  Revenue CREDIT = fee       (platform sweep fee)
```

---

## 5. Encryption Patterns

### Private Key Encryption (AddressPool)

Located in `address_pool_repo_impl.go`, the encryption system protects private keys at rest:

**Encryption Flow (`encryptPrivateKey`, 27 lines):**
1. Retrieve RSA-encrypted password from configuration
2. Decrypt password using RSA private key
3. Combine decrypted password with a per-address nonce
4. Derive AES-256 key via SHA-256 hash
5. Encrypt private key using AES-GCM (authenticated encryption)
6. Store encrypted private key in database

**Decryption Flow (`decryptPrivateKey`, 27 lines):**
1. Retrieve RSA-encrypted password from configuration
2. Decrypt password using RSA private key
3. Combine decrypted password with the stored nonce
4. Derive AES-256 key via SHA-256 hash
5. Decrypt private key using AES-GCM

**Key Derivation (`getKeyFromRSAEncryptedPasswordWithNonce`, 25 lines):**
- Combines RSA-decrypted password with nonce via concatenation
- SHA-256 hash produces the AES-256 key (32 bytes)
- Contains two inner closures for cleanup/error handling

### Wallet Private Key Encryption

Located in `wallet_repo_impl.go`:

**`decryptRSAEncryptedWithNonce` (22 lines):**
- Similar pattern to AddressPool encryption
- Used when creating wallets with SCWs (Smart Contract Wallets)
- Encrypted private key stored in `Wallet.SecretDataByte`

### Security Properties

- **Key hierarchy:** RSA master key -> AES per-address key
- **Nonce-based uniqueness:** Each address has a unique nonce, preventing key reuse even for the same master password
- **Authenticated encryption:** AES-GCM provides both confidentiality and integrity
- **At-rest protection:** Private keys are never stored in plaintext in the database

---

## 6. Locking and Concurrency

### Pessimistic Locking (PaymentChannelProject)

The `PaymentChannelProjectRepositoryImpl` implements the most sophisticated locking strategy:

**`lockRowWithTimeout` (49 lines):**
```sql
SET LOCAL lock_timeout = '5000ms'
SELECT * FROM payment_channels WHERE id = ? FOR UPDATE
```

- Sets PostgreSQL session-local lock timeout
- Acquires exclusive row lock via `FOR UPDATE`
- Returns wrapped error on timeout with descriptive message

**`isLockError` (13 lines):**
- Checks for PostgreSQL error code `55P03` (lock_not_available)
- Used to differentiate lock failures from other database errors

**`lockPaymentChannelSafely` (37 lines):**
- Wraps `lockRowWithTimeout` with payment-channel-specific logic
- Acquires lock, returns the locked record for update

**`lockDisabledMappingSafely` (41 lines):**
- Similar pattern for disabled payment channel project mappings
- Handles case where mapping does not exist (no lock needed for creates)

### Withdrawal Locking

The `WithdrawalRepoImpl` uses pessimistic locking for withdrawal status updates:

**`LockAndUpdateWithdrawal` (45 lines):**
```go
db.Transaction(func(tx *gorm.DB) error {
    // SELECT ... FOR UPDATE locks the withdrawal row
    tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&withdrawal, id)
    // Validate status transition
    // Update status
})
```

**`LockAndUpdateWithdrawalForMerchant` (37 lines):**
- Same pattern with additional merchant-specific validation

### Implicit Locking via Transaction Isolation

Many repositories rely on PostgreSQL's default transaction isolation (Read Committed) combined with GORM's `Transaction()` method:

- `DepositRepositoryImpl.ProcessPendingDeposit` -- atomic deposit processing
- `BlockchainRepositoryImpl.CreateDepositReceived` -- atomic deposit creation
- `WithdrawalProcessingRepoImpl.ProcessAccountingForWithdrawalSent` -- atomic accounting
- `SweepTransactionRepositoryImpl.addAddressToTransaction` -- atomic address assignment

### Sweep Transaction Soft Locking

The sweep system uses a **soft locking** pattern where `sweeping_transaction_id` on `AccountAddress` and `UTXO` models acts as a claim marker:

```
1. Find eligible address (sweeping_transaction_id IS NULL)
2. Set sweeping_transaction_id = current_sweep_tx_id (claims the address)
3. Process sweep
4. Clear or finalize sweeping_transaction_id
```

This prevents two concurrent sweep processes from claiming the same address.

---

## 7. Complex Query Patterns

### Longest Methods by Line Count

| Rank | Method | Repository | Lines | Category |
|------|--------|-----------|-------|----------|
| 1 | `FetchData` | AnalyticsRepositoryImpl | 428 | Raw SQL template execution |
| 2 | `ProcessAccountingForWithdrawalSent` | WithdrawalProcessingRepoImpl | 222 | Accounting transaction |
| 3 | `ProcessAccountingForBTCSweep` | SweepUTXORepositoryImpl | 206 | Accounting transaction |
| 4 | `ProcessERC20Sweep` | AccountRepositoryImpl | 203 | Accounting transaction |
| 5 | `addUTXOToTransaction` | SweepTransactionRepositoryImpl | 198 | UTXO sweep logic |
| 6 | `UpdateWithdrawalStatus` | WithdrawalRepoImpl | 193 | State machine |
| 7 | `ProcessAccountingForWithdrawalSentOfMerchantPayout` | WithdrawalProcessingRepoImpl | 173 | Accounting transaction |
| 8 | `GetAllInternalMembers` | MemberRepositoryImpl | 172 | Complex paginated query |
| 9 | `addAddressToTransaction` | SweepTransactionRepositoryImpl | 170 | Sweep address assignment |
| 10 | `GetEligibleAddressesToSweep` | AddressRepositoryImpl | 158 | Eligibility calculation |

### FetchData (AnalyticsRepositoryImpl, 428 lines)

The longest method in the entire codebase. It:
1. Receives graph configuration with a SQL template and filter mappings
2. Performs string substitution to inject filter values into the template
3. Executes the raw SQL query against PostgreSQL
4. Handles multiple result formats (time series, aggregations, distributions)
5. Maps results to chart-compatible data structures
6. Contains a 234-line defer wrapper for comprehensive error recovery

### ProcessAccountingForWithdrawalSent (WithdrawalProcessingRepoImpl, 222 lines)

Within a single transaction:
1. Loads the withdrawal with all associations
2. Validates the withdrawal is in "processing" status
3. Updates withdrawal status to "sent" with blockchain details (tx_hash, block_number)
4. Creates Asset debit entry (funds left the platform wallet)
5. Creates Liability credit entry (obligation to the user is fulfilled)
6. Calculates and creates Revenue credit entry for withdrawal fee
7. Calculates and creates Expense debit entry for blockchain gas cost
8. Updates the user's Account (unlocks funds, reduces locked amount)
9. Creates webhook delivery logs for merchant notification
10. Returns enriched withdrawal data for API response

### GetEligibleAddressesToSweep (AddressRepositoryImpl, 158 lines)

Builds a multi-table query to identify addresses ready for sweep:
1. JOINs: account_addresses, address_pools, blockchain_currencies, blockchains
2. Filters: minimum collection amount, minimum balance for sweep
3. Time check: address not locked by another sweep within the configured duration
4. Batch size: limits results to the configured sweep batch size
5. Subquery: excludes addresses already assigned to pending sweep transactions
6. Aggregation: groups by address with balance summation

### addUTXOToTransaction (SweepTransactionRepositoryImpl, 198 lines)

Handles Bitcoin UTXO selection for sweep transactions:
1. Validates the UTXO exists and is unspent
2. Verifies the UTXO belongs to an address in our address pool
3. Checks the UTXO amount meets minimum thresholds
4. Verifies the UTXO's deposit is in a confirmed status
5. Links the UTXO to the sweep transaction
6. Updates the UTXO's sweeping_transaction_id (soft lock)
7. Recalculates the sweep transaction's total amount and UTXO count
8. Enforces batch size limits from blockchain_currency configuration

---

## Summary Statistics

| Metric | Value |
|--------|-------|
| Total repositories | 50 |
| Total repository functions | 425+ |
| Total estimated lines of code | ~12,000 |
| Repositories with transaction logic | 25 |
| Repositories with accounting entries | 7 |
| Repositories with encryption | 2 (AddressPool, Wallet) |
| Repositories with pessimistic locking | 2 (PaymentChannelProject, Withdrawal) |
| Largest repository by methods | BlockchainRepositoryImpl (30+ methods) |
| Largest repository by lines | BlockchainRepositoryImpl (~1,155 lines) |
| Smallest repository | EntrypointSCAddressRepositoryImpl (1 method, 6 lines) |
| Longest single method | AnalyticsRepositoryImpl.FetchData (428 lines) |
| Most complex domain | Sweep (3 repositories, ~1,300 lines combined) |
