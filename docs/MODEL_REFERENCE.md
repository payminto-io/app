# Payminto Go Model Reference

**Source:** Reverse-engineered from PayRam Docker container (`payramapp/payram:latest`)
**Total Models:** 77 named structs (+ 2 base model types, + join table structs)
**ORM:** GORM v2
**Database:** PostgreSQL 14
**Decimal Library:** `shopspring/decimal` (`decimal.Decimal` maps to `numeric(38,18)`)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Base Model Types](#2-base-model-types)
3. [Models by Domain](#3-models-by-domain)
   - [A. User & Access Management](#a-user--access-management)
   - [B. Blockchain Infrastructure](#b-blockchain-infrastructure)
   - [C. Wallet Management](#c-wallet-management)
   - [D. Address Pool](#d-address-pool)
   - [E. Deposits & Transactions](#e-deposits--transactions)
   - [F. UTXO & Sweeps](#f-utxo--sweeps)
   - [G. Payment Processing](#g-payment-processing)
   - [H. Webhooks](#h-webhooks)
   - [I. Accounting / Double-Entry Ledger](#i-accounting--double-entry-ledger)
   - [J. Referral & Campaigns](#j-referral--campaigns)
   - [K. Analytics](#k-analytics)
   - [L. External Platforms](#l-external-platforms)
   - [M. System & Utilities](#m-system--utilities)
4. [Relationship Map](#4-relationship-map)
5. [Key Design Patterns](#5-key-design-patterns)
6. [State Machine Diagrams](#6-state-machine-diagrams)

---

## 1. Overview

Payminto uses 77 Go model structs organized into 13 domains. All models follow GORM conventions:

- **Primary keys:** `uint` (bigint auto-increment in PostgreSQL, not UUIDs)
- **Timestamps:** `CreatedAt time.Time`, `UpdatedAt time.Time`, `DeletedAt gorm.DeletedAt` on all entities
- **Soft deletes:** Enabled via `gorm.DeletedAt` (adds `WHERE deleted_at IS NULL` to queries)
- **Monetary values:** `decimal.Decimal` from `shopspring/decimal` (maps to `numeric(38,18)` in PostgreSQL)
- **Relationships:** GORM association tags with `foreignKey`, `references`, and preloading
- **Table naming:** GORM default snake_case pluralization (e.g., `Member` -> `members`)
- **Join tables:** Some use composite primary keys instead of base models (e.g., `MemberRole`, `CampaignEvent`)

Most business entities embed `PayramModel` as their base. Campaign/referral entities embed `BaseModel`. A few join-table structs have no embedded base model and use composite primary keys.

---

## 2. Base Model Types

### PayramModel

Used by most business entities. Provides standard GORM fields.

```go
type PayramModel struct {
    ID        uint           `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

| Field     | Go Type        | DB Column    | DB Type       | Notes                          |
|-----------|----------------|--------------|---------------|--------------------------------|
| ID        | uint           | id           | bigint        | Auto-increment primary key     |
| CreatedAt | time.Time      | created_at   | timestamptz   | Set automatically by GORM      |
| UpdatedAt | time.Time      | updated_at   | timestamptz   | Updated automatically by GORM  |
| DeletedAt | gorm.DeletedAt | deleted_at   | timestamptz   | Soft delete; indexed           |

### BaseModel

Used by campaign/referral domain entities. Structurally identical to PayramModel.

```go
type BaseModel struct {
    ID        uint           `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

| Field     | Go Type        | DB Column    | DB Type       | Notes                          |
|-----------|----------------|--------------|---------------|--------------------------------|
| ID        | uint           | id           | bigint        | Auto-increment primary key     |
| CreatedAt | time.Time      | created_at   | timestamptz   | Set automatically by GORM      |
| UpdatedAt | time.Time      | updated_at   | timestamptz   | Updated automatically by GORM  |
| DeletedAt | gorm.DeletedAt | deleted_at   | timestamptz   | Soft delete; indexed           |

---

## 3. Models by Domain

### A. User & Access Management

#### 1. Member

Central user/merchant entity. All major entities reference this table.

- **Base model:** PayramModel
- **Table name:** `members`

| Field                    | Go Type           | DB Column                 | DB Type       | Nullable | Notes                              |
|--------------------------|-------------------|---------------------------|---------------|----------|------------------------------------|
| PayramModel              | PayramModel       | (embedded)                | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name                     | string            | name                      | varchar(100)  | NO       | Display name                       |
| Email                    | *string           | email                     | text          | YES      | Email address                      |
| CustomerID               | string            | customer_id               | text          | YES      | External customer identifier       |
| Level                    | int               | level                     | bigint        | NO       | Member level (default 0)           |
| Group                    | string            | group                     | varchar(20)   | NO       | Member group tier (default 'vip-0') |
| State                    | string            | state                     | varchar(20)   | NO       | active/inactive/banned             |
| Username                 | *string           | username                  | varchar(20)   | YES      | Login username                     |
| Password                 | *string           | password                  | text          | YES      | Hashed password                    |
| ResetPasswordRequired    | bool              | reset_password_required   | boolean       | NO       | Force password reset flag          |
| ResetPasswordToken       | *string           | reset_password_token      | text          | YES      | Password reset token               |
| ResetPasswordExpiry      | *time.Time        | reset_password_expiry     | timestamptz   | YES      | Token expiry                       |
| MemberType               | string            | member_type               | varchar(20)   | NO       | customer/admin/system              |
| ReferredByMemberID       | *uint             | referred_by_member_id     | bigint        | YES      | FK -> members.id (self-referencing) |
| ReferredByMember         | *Member           | —                         | —             | —        | BelongsTo: Member (self-ref)       |
| Roles                    | []Role            | —                         | —             | —        | ManyToMany via member_roles        |
| APIKeys                  | []APIKey          | —                         | —             | —        | HasMany: APIKey.MemberID           |
| MemberExternalPlatformRoles | []MemberExternalPlatformRole | —          | —             | —        | HasMany                            |

**Foreign keys:**
- `referred_by_member_id` -> `members(id)` (self-referencing)

**Relationships:**
- HasMany: APIKey, MemberExternalPlatformRole
- ManyToMany: Role (via member_roles join table)
- BelongsTo: Member (self-referencing via ReferredByMemberID)

---

#### 2. Role

RBAC role definitions.

- **Base model:** PayramModel
- **Table name:** `roles`

| Field         | Go Type                         | DB Column    | DB Type       | Nullable | Notes                              |
|---------------|---------------------------------|--------------|---------------|----------|------------------------------------|
| PayramModel   | PayramModel                     | (embedded)   | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name          | string                          | name         | varchar(200)  | NO       | Role identifier                    |
| DisplayName   | string                          | display_name | varchar(255)  | NO       | Human-readable name                |
| Description   | *string                         | description  | text          | YES      | Role description                   |
| Permissions   | []Permission                    | —            | —             | —        | ManyToMany via role_permissions    |
| Members       | []Member                        | —            | —             | —        | ManyToMany via member_roles        |
| PlatformRoles | []MemberExternalPlatformRole    | —            | —             | —        | HasMany                            |

**Relationships:**
- ManyToMany: Permission (via role_permissions), Member (via member_roles)
- HasMany: MemberExternalPlatformRole

---

#### 3. Permission

Granular permission definitions for RBAC.

- **Base model:** PayramModel
- **Table name:** `permissions`

| Field       | Go Type     | DB Column    | DB Type       | Nullable | Notes                              |
|-------------|-------------|--------------|---------------|----------|------------------------------------|
| PayramModel | PayramModel | (embedded)   | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name        | string      | name         | varchar(200)  | NO       | Permission identifier              |
| Description | string      | description  | text          | YES      | Permission description             |
| DisplayName | string      | display_name | varchar(255)  | NO       | Human-readable name                |

**Relationships:**
- ManyToMany: Role (via role_permissions)

---

#### 4. MemberRole

Join table linking members to roles.

- **Base model:** None (composite primary key)
- **Table name:** `member_roles`

| Field    | Go Type | DB Column  | DB Type | Nullable | Notes                    |
|----------|---------|------------|---------|----------|--------------------------|
| RoleID   | uint    | role_id    | bigint  | NO       | FK -> roles.id           |
| MemberID | uint    | member_id  | bigint  | NO       | FK -> members.id         |
| Role     | Role    | —          | —       | —        | BelongsTo: Role          |
| Member   | Member  | —          | —       | —        | BelongsTo: Member        |

**Primary key:** (member_id, role_id)
**Foreign keys:**
- `member_id` -> `members(id)`
- `role_id` -> `roles(id)`

---

#### 5. RolePermission

Join table linking roles to permissions.

- **Base model:** None (composite primary key)
- **Table name:** `role_permissions`

| Field        | Go Type     | DB Column     | DB Type | Nullable | Notes                    |
|--------------|-------------|---------------|---------|----------|--------------------------|
| RoleID       | uint        | role_id       | bigint  | NO       | FK -> roles.id           |
| PermissionID | uint        | permission_id | bigint  | NO       | FK -> permissions.id     |
| Role         | *Role       | —             | —       | —        | BelongsTo: Role          |
| Permission   | *Permission | —             | —       | —        | BelongsTo: Permission    |

**Primary key:** (role_id, permission_id)
**Foreign keys:**
- `role_id` -> `roles(id)`
- `permission_id` -> `permissions(id)`

---

#### 6. APIKey

API authentication keys scoped to external platforms.

- **Base model:** PayramModel
- **Table name:** `api_keys`

| Field              | Go Type           | DB Column            | DB Type       | Nullable | Notes                              |
|--------------------|-------------------|----------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel       | (embedded)           | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Key                | string            | key                  | varchar(255)  | NO       | API key string                     |
| Status             | string            | status               | varchar(20)   | NO       | active/revoked (default 'active')  |
| MemberID           | *uint             | member_id            | bigint        | YES      | FK -> members.id                   |
| Member             | *Member           | —                    | —             | —        | BelongsTo: Member                  |
| ExternalPlatformID | uint              | external_platform_id | bigint        | NO       | FK -> external_platforms.id        |
| ExternalPlatform   | *ExternalPlatform | —                    | —             | —        | BelongsTo: ExternalPlatform        |
| RoleID             | *uint             | role_id              | bigint        | YES      | FK -> roles.id                     |
| ExpireAt           | *time.Time        | expire_at            | timestamptz   | YES      | Key expiration                     |
| Role               | *Role             | —                    | —             | —        | BelongsTo: Role                    |
| Description        | *string           | description          | text          | YES      | Key description                    |

**Foreign keys:**
- `member_id` -> `members(id)`
- `external_platform_id` -> `external_platforms(id)`
- `role_id` -> `roles(id)`

---

#### 7. MemberExternalPlatformRole

Role assignment per member per project (ternary join table).

- **Base model:** None (composite primary key with timestamps)
- **Table name:** `member_external_platform_roles`

| Field              | Go Type           | DB Column            | DB Type     | Nullable | Notes                              |
|--------------------|-------------------|----------------------|-------------|----------|------------------------------------|
| CreatedAt          | time.Time         | created_at           | timestamptz | YES      | Timestamp                          |
| MemberID           | uint              | member_id            | bigint      | NO       | FK -> members.id                   |
| ExternalPlatformID | uint              | external_platform_id | bigint      | NO       | FK -> external_platforms.id        |
| RoleID             | uint              | role_id              | bigint      | NO       | FK -> roles.id                     |
| Member             | *Member           | —                    | —           | —        | BelongsTo: Member                  |
| ExternalPlatform   | *ExternalPlatform | —                    | —           | —        | BelongsTo: ExternalPlatform        |
| Role               | *Role             | —                    | —           | —        | BelongsTo: Role                    |

**Primary key:** (member_id, external_platform_id)
**Foreign keys:**
- `member_id` -> `members(id)`
- `external_platform_id` -> `external_platforms(id)`
- `role_id` -> `roles(id)`

---

#### 8. AuthRefreshToken

JWT refresh token management with revocation support.

- **Base model:** PayramModel
- **Table name:** `auth_refresh_tokens`

| Field       | Go Type     | DB Column    | DB Type       | Nullable | Notes                              |
|-------------|-------------|--------------|---------------|----------|------------------------------------|
| PayramModel | PayramModel | (embedded)   | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID    | uint        | member_id    | bigint        | NO       | FK -> members.id                   |
| Member      | *Member     | —            | —             | —        | BelongsTo: Member                  |
| Token       | string      | token        | varchar(255)  | NO       | Unique refresh token               |
| ExpiresAt   | time.Time   | expires_at   | timestamptz   | NO       | Token expiration                   |
| LastUsedAt  | *time.Time  | last_used_at | timestamptz   | YES      | Last usage timestamp               |
| RevokedAt   | *time.Time  | revoked_at   | timestamptz   | YES      | Revocation timestamp               |

**Unique:** `token`
**Foreign keys:**
- `member_id` -> `members(id)`

---

#### 9. OTP

One-time passwords for 2FA/email verification.

- **Base model:** PayramModel
- **Table name:** `otps`

| Field         | Go Type     | DB Column       | DB Type       | Nullable | Notes                              |
|---------------|-------------|-----------------|---------------|----------|------------------------------------|
| PayramModel   | PayramModel | (embedded)      | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| EntityID      | uint        | entity_id       | bigint        | NO       | Referenced entity ID               |
| Purpose       | string      | purpose         | varchar(50)   | NO       | OTP purpose (withdrawal, login)    |
| Code          | string      | code            | varchar(20)   | NO       | OTP code value                     |
| ExpiresAt     | time.Time   | expires_at      | timestamptz   | NO       | Expiration time                    |
| Verified      | bool        | verified        | boolean       | YES      | Whether OTP was verified           |
| VerifiedAt    | *time.Time  | verified_at     | timestamptz   | YES      | Verification timestamp             |
| Attempts      | int         | attempts        | bigint        | YES      | Number of attempts (default 0)     |
| LastAttemptAt | *time.Time  | last_attempt_at | timestamptz   | YES      | Last attempt timestamp             |
| EventMetadata | *string     | event_metadata  | json          | YES      | Additional metadata                |

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 10. WebSocketToken

WebSocket authentication tokens.

- **Base model:** PayramModel
- **Table name:** `web_socket_tokens`

| Field       | Go Type     | DB Column  | DB Type       | Nullable | Notes                              |
|-------------|-------------|------------|---------------|----------|------------------------------------|
| PayramModel | PayramModel | (embedded) | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Token       | string      | token      | varchar(255)  | NO       | WebSocket auth token               |
| ServerURL   | string      | server_url | varchar(255)  | NO       | WebSocket server URL               |
| Expiry      | time.Time   | expiry     | timestamptz   | NO       | Token expiration                   |
| ClientID    | string      | client_id  | varchar(255)  | NO       | Client identifier                  |
| MemberID    | uint        | member_id  | bigint        | NO       | FK -> members.id                   |
| Member      | *Member     | —          | —             | —        | BelongsTo: Member                  |

**Foreign keys:**
- `member_id` -> `members(id)`

---

#### 11. MemberExternalPlatform

Links members to projects/external platforms.

- **Base model:** PayramModel
- **Table name:** `member_external_platforms`

| Field              | Go Type          | DB Column            | DB Type | Nullable | Notes                              |
|--------------------|------------------|----------------------|---------|----------|------------------------------------|
| PayramModel        | PayramModel      | (embedded)           | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID           | uint             | member_id            | bigint  | NO       | FK -> members.id                   |
| ExternalPlatformID | uint             | external_platform_id | bigint  | NO       | FK -> external_platforms.id        |
| Member             | Member           | —                    | —       | —        | BelongsTo: Member                  |
| ExternalPlatform   | ExternalPlatform | —                    | —       | —        | BelongsTo: ExternalPlatform        |

**Foreign keys:**
- `member_id` -> `members(id)`
- `external_platform_id` -> `external_platforms(id)`

---

### B. Blockchain Infrastructure

#### 12. Blockchain

Individual blockchain network configuration.

- **Base model:** PayramModel
- **Table name:** `blockchains`

| Field              | Go Type           | DB Column            | DB Type       | Nullable | Notes                              |
|--------------------|-------------------|----------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel       | (embedded)           | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Code               | string            | code                 | varchar(20)   | NO       | Chain code (eth, btc, base, etc.)  |
| Name               | string            | name                 | varchar(150)  | NO       | Display name                       |
| Family             | string            | family               | varchar(30)   | NO       | Family code (evm, btc, tron)       |
| Client             | string            | client               | varchar(200)  | YES      | Client type                        |
| Height             | uint64            | height               | bigint        | YES      | Current synced block height        |
| HeightTimestamp     | *time.Time        | height_timestamp     | timestamptz   | YES      | Timestamp of Height                |
| ExplorerAddress    | *string           | explorer_address     | varchar(200)  | YES      | Block explorer address URL format  |
| ExplorerTransaction | *string          | explorer_transaction | varchar(200)  | YES      | Block explorer TX URL format       |
| MinConfirmations   | uint              | min_confirmations    | bigint        | NO       | Required confirmations (default 1) |
| ChainID            | *uint64           | chain_id             | bigint        | YES      | EVM chain ID                       |
| Status             | string            | status               | varchar(20)   | NO       | active/disabled (default 'active') |
| IsSCW              | bool              | is_scw               | boolean       | NO       | Supports smart contract wallets    |
| BlockchainFamilyID | uint              | blockchain_family_id | bigint        | NO       | FK -> blockchain_families.id       |
| BlockchainFamily   | *BlockchainFamily | —                    | —             | —        | BelongsTo: BlockchainFamily        |
| IsCreate2Supported | *bool             | is_create2_supported | boolean       | YES      | Supports CREATE2 opcode            |
| RPCNodes           | []RPCNode         | —                    | —             | —        | HasMany: RPCNode.BlockchainID      |
| NetworkMode        | string            | network_mode         | varchar       | YES      | testnet/mainnet                    |

**Foreign keys:**
- `blockchain_family_id` -> `blockchain_families(id)`

**Relationships:**
- BelongsTo: BlockchainFamily
- HasMany: RPCNode

---

#### 13. BlockchainFamily

Groups blockchains by derivation type (EVM, BTC, TRX).

- **Base model:** PayramModel
- **Table name:** `blockchain_families`

| Field              | Go Type     | DB Column          | DB Type       | Nullable | Notes                              |
|--------------------|-------------|--------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel | (embedded)         | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Family             | string      | family             | varchar(30)   | NO       | Family identifier (evm, btc, tron) |
| Path               | *string     | path               | varchar(255)  | YES      | HD derivation path                 |
| GasPath            | *string     | gas_path           | varchar(255)  | YES      | Gas wallet derivation path         |
| SupportsHDWallet   | bool        | supports_hd_wallet | boolean       | NO       | HD wallet support flag             |
| SupportsSCWallet   | bool        | supports_sc_wallet | boolean       | NO       | Smart contract wallet support      |

---

#### 14. Currency

Token/coin definitions.

- **Base model:** PayramModel
- **Table name:** `currencies`

| Field             | Go Type          | DB Column          | DB Type        | Nullable | Notes                              |
|-------------------|------------------|--------------------|----------------|----------|------------------------------------|
| PayramModel       | PayramModel      | (embedded)         | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name              | string           | name               | varchar(10)    | NO       | Currency name                      |
| Code              | string           | code               | varchar(10)    | NO       | Currency code (BTC, ETH, USDC)     |
| Description       | *string          | description        | text           | YES      | Currency description               |
| Homepage          | *string          | homepage           | text           | YES      | Project homepage URL               |
| Type              | string           | type               | varchar(30)    | NO       | coin/token (default 'coin')        |
| Visible           | bool             | visible            | boolean        | NO       | Visible in UI (default true)       |
| DepositEnabled    | bool             | deposit_enabled    | boolean        | NO       | Deposits enabled (default true)    |
| WithdrawalEnabled | bool             | withdrawal_enabled | boolean        | NO       | Withdrawals enabled (default true) |
| WalletPrecision   | uint             | wallet_precision   | bigint         | NO       | Display precision (default 6)      |
| BasePrecision     | uint32           | base_precision     | bigint         | NO       | Base precision (default 8)         |
| IconURL           | *string          | icon_url           | text           | YES      | Currency icon URL                  |
| Price             | *decimal.Decimal | price              | numeric(38,18) | YES      | Current USD price                  |

---

#### 15. BlockchainCurrency

Maps currencies to specific blockchains (e.g., USDC on Ethereum, USDC on Base).

- **Base model:** PayramModel
- **Table name:** `blockchain_currencies`

| Field                                  | Go Type          | DB Column                                  | DB Type        | Nullable | Notes                              |
|----------------------------------------|------------------|--------------------------------------------|----------------|----------|------------------------------------|
| PayramModel                            | PayramModel      | (embedded)                                 | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Address                                | string           | address                                    | varchar(150)   | NO       | Token contract address             |
| Standard                               | string           | standard                                   | varchar(30)    | NO       | Token standard (ERC20, TRC20, native) |
| CurrencyCode                           | string           | currency_code                              | varchar(20)    | NO       | Currency code reference            |
| BlockchainCode                         | string           | blockchain_code                            | varchar(20)    | NO       | Blockchain code reference          |
| DepositFee                             | *decimal.Decimal | deposit_fee                                | numeric(38,18) | YES      | Deposit fee amount                 |
| MinDepositAmount                       | *decimal.Decimal | min_deposit_amount                         | numeric(38,18) | YES      | Minimum deposit                    |
| MinCollectionAmount                    | *decimal.Decimal | min_collection_amount                      | numeric(38,18) | YES      | Min amount to trigger collection   |
| WithdrawFee                            | *decimal.Decimal | withdraw_fee                               | numeric(38,18) | YES      | Withdrawal fee amount              |
| MinWithdrawAmount                      | *decimal.Decimal | min_withdraw_amount                        | numeric(38,18) | YES      | Minimum withdrawal                 |
| WithdrawLimit24hr                      | *decimal.Decimal | withdraw_limit24hr                         | numeric(38,18) | YES      | 24-hour withdrawal limit           |
| WithdrawLimit72hr                      | *decimal.Decimal | withdraw_limit72hr                         | numeric(38,18) | YES      | 72-hour withdrawal limit           |
| Visible                                | bool             | visible                                    | boolean        | NO       | Visible in UI (default true)       |
| DepositEnabled                         | bool             | deposit_enabled                            | boolean        | NO       | Deposits enabled (default true)    |
| WithdrawalEnabled                      | bool             | withdrawal_enabled                         | boolean        | NO       | Withdrawals enabled (default false)|
| WalletPrecision                        | uint             | wallet_precision                           | bigint         | NO       | Display precision (default 6)      |
| ABI                                    | *string          | abi                                        | text           | YES      | Token ABI JSON                     |
| ApprovalFeeAmount                      | *decimal.Decimal | approval_fee_amount                        | numeric(38,18) | YES      | ERC20 approval gas fee             |
| MinBalanceForFeesTransfer              | *decimal.Decimal | min_balance_for_fees_transfer              | numeric(38,18) | YES      | Min native balance for gas         |
| MinBalanceForSweep                     | *decimal.Decimal | min_balance_for_sweep                      | numeric(38,18) | YES      | Min balance to trigger sweep       |
| SweepAddressLockedDurationInMinutes    | *int             | sweep_address_locked_duration_in_minutes   | bigint         | YES      | Lock duration during sweep         |
| SweepBatchSize                         | *int             | sweep_batch_size                           | bigint         | YES      | Addresses per sweep batch          |
| SweepApprovalBatchSize                 | *int             | sweep_approval_batch_size                  | bigint         | YES      | Approvals per batch                |
| SweepMaxWaitTimeInMinutes              | *uint            | sweep_max_wait_time_in_minutes             | bigint         | YES      | Max wait before force sweep        |
| MinAmountInSweepTx                     | *decimal.Decimal | min_amount_in_sweep_tx                     | numeric(38,18) | YES      | Min amount per sweep TX            |
| CurrencyID                             | uint             | currency_id                                | bigint         | NO       | FK -> currencies.id                |
| BlockchainID                           | uint             | blockchain_id                              | bigint         | NO       | FK -> blockchains.id               |
| Currency                               | *Currency        | —                                          | —              | —        | BelongsTo: Currency                |
| Blockchain                             | *Blockchain      | —                                          | —              | —        | BelongsTo: Blockchain              |

**Foreign keys:**
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`

---

#### 16. RPCNode

Blockchain RPC node configuration (Alchemy, Infura, custom).

- **Base model:** PayramModel
- **Table name:** `rpc_nodes`

| Field           | Go Type     | DB Column        | DB Type       | Nullable | Notes                              |
|-----------------|-------------|------------------|---------------|----------|------------------------------------|
| PayramModel     | PayramModel | (embedded)       | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| BlockchainID    | uint        | blockchain_id    | bigint        | NO       | FK -> blockchains.id               |
| URL             | string      | url              | varchar(500)  | NO       | RPC endpoint URL                   |
| NodeType        | string      | node_type        | varchar(20)   | NO       | free/paid (default 'free')         |
| IsPreferred     | bool        | is_preferred     | boolean       | NO       | Preferred node flag                |
| IsActive        | bool        | is_active        | boolean       | NO       | Active flag (default true)         |
| MaxDailyCalls   | *int        | max_daily_calls  | bigint        | YES      | Rate limit                         |
| APIKey          | *string     | api_key          | varchar(500)  | YES      | API key for node                   |
| AuthUsername    | *string     | auth_username    | varchar(200)  | YES      | Basic auth username                |
| AuthPassword   | *string     | auth_password    | varchar(200)  | YES      | Basic auth password                |
| ChainIdentifier | string     | chain_identifier | varchar(100)  | NO       | Chain identifier string            |
| CredentialHash  | string     | credential_hash  | varchar(64)   | NO       | Hash for uniqueness                |
| Priority        | int        | priority         | bigint        | YES      | Node priority ranking              |

**Unique:** (url, chain_identifier, credential_hash)
**Foreign keys:**
- `blockchain_id` -> `blockchains(id)`

---

#### 17. BlockchainContract

Smart contract definitions (ABI + bytecode).

- **Base model:** PayramModel
- **Table name:** `blockchain_contracts`

| Field             | Go Type           | DB Column         | DB Type       | Nullable | Notes                              |
|-------------------|-------------------|-------------------|---------------|----------|------------------------------------|
| PayramModel       | PayramModel       | (embedded)        | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| BlockchainCode    | string            | blockchain_code   | varchar(20)   | NO       | Chain code reference               |
| ContractType      | string            | contract_type     | varchar(100)  | NO       | Contract type identifier           |
| ABI               | *string           | abi               | text          | YES      | Contract ABI JSON                  |
| Bytecode          | *string           | bytecode          | text          | YES      | Contract bytecode                  |
| Address           | *string           | address           | varchar(200)  | YES      | Deployed address                   |
| Description       | string            | description       | text          | YES      | Contract description               |
| Status            | string            | status            | varchar(20)   | NO       | active/disabled (default 'active') |
| BlockchainID      | uint              | blockchain_id     | bigint        | NO       | FK -> blockchains.id               |
| CurrencyStandard  | string            | currency_standard | varchar(30)   | NO       | Token standard (ERC20, etc.)       |
| Blockchain        | *Blockchain       | —                 | —             | —        | BelongsTo: Blockchain              |
| ContractAddresses | []ContractAddress | —                 | —             | —        | HasMany: ContractAddress           |

**Foreign keys:**
- `blockchain_id` -> `blockchains(id)`

**Relationships:**
- BelongsTo: Blockchain
- HasMany: ContractAddress

---

#### 18. ContractAddress

Deployed smart contract instances.

- **Base model:** PayramModel
- **Table name:** `contract_addresses`

| Field                  | Go Type            | DB Column               | DB Type       | Nullable | Notes                              |
|------------------------|--------------------|-------------------------|---------------|----------|------------------------------------|
| PayramModel            | PayramModel        | (embedded)              | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| BlockchainCode         | string             | blockchain_code         | varchar(20)   | NO       | Chain code reference               |
| Name                   | *string            | name                    | varchar(100)  | YES      | Contract instance name             |
| Address                | *string            | address                 | varchar(200)  | YES      | Deployed address                   |
| Status                 | string             | status                  | varchar(20)   | NO       | active/disabled (default 'active') |
| WalletAddress          | *string            | wallet_address          | varchar(200)  | YES      | Associated wallet address          |
| TransactionHash        | *string            | transaction_hash        | text          | YES      | Deployment TX hash                 |
| CurrencyStandard       | string             | currency_standard       | varchar(30)   | NO       | Token standard                     |
| BlockchainContractType | string             | blockchain_contract_type| varchar(100)  | NO       | Contract type reference            |
| BlockchainContractID   | uint               | blockchain_contract_id  | bigint        | NO       | FK -> blockchain_contracts.id      |
| BlockchainContract     | *BlockchainContract| —                       | —             | —        | BelongsTo: BlockchainContract      |
| Attributes             | *string            | attributes              | text          | YES      | Additional attributes JSON         |
| ABI                    | *string            | abi                     | text          | YES      | Instance-specific ABI              |

**Foreign keys:**
- `blockchain_contract_id` -> `blockchain_contracts(id)`

---

#### 19. AddressContractSignature

Smart contract signatures for deposit addresses (SmartSweep approvals).

- **Base model:** PayramModel
- **Table name:** `address_contract_signatures`

| Field                  | Go Type            | DB Column               | DB Type       | Nullable | Notes                              |
|------------------------|--------------------|-------------------------|---------------|----------|------------------------------------|
| PayramModel            | PayramModel        | (embedded)              | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Address                | string             | address                 | varchar(200)  | NO       | Deposit address                    |
| Signature              | string             | signature               | text          | NO       | Unique contract signature          |
| AddressPathIndex       | uint               | address_path_index      | bigint        | NO       | HD derivation index                |
| Family                 | string             | family                  | varchar(30)   | NO       | Blockchain family                  |
| ContractAddr           | string             | contract_addr           | varchar(200)  | YES      | Contract address                   |
| BlockchainCode         | string             | blockchain_code         | varchar(20)   | NO       | Chain code reference               |
| CurrencyCode           | string             | currency_code           | varchar(20)   | NO       | Currency code reference            |
| CurrencyID             | uint               | currency_id             | bigint        | NO       | FK -> currencies.id                |
| BlockchainID           | uint               | blockchain_id           | bigint        | NO       | FK -> blockchains.id               |
| Status                 | string             | status                  | varchar(20)   | NO       | pending/completed (default 'pending') |
| BlockchainContractType | string             | blockchain_contract_type| varchar(100)  | NO       | Contract type reference            |
| BlockchainContractID   | uint               | blockchain_contract_id  | bigint        | NO       | FK -> blockchain_contracts.id      |
| ContractAddressID      | uint               | contract_address_id     | bigint        | NO       | FK -> contract_addresses.id        |
| BlockchainContract     | *BlockchainContract| —                       | —             | —        | BelongsTo: BlockchainContract      |
| ContractAddress        | *ContractAddress   | —                       | —             | —        | BelongsTo: ContractAddress         |

**Unique:** `signature`
**Foreign keys:**
- `blockchain_contract_id` -> `blockchain_contracts(id)`
- `contract_address_id` -> `contract_addresses(id)`

---

#### 20. AddressDeployment

Tracks smart contract wallet deployments per blockchain.

- **Base model:** PayramModel
- **Table name:** `address_deployments`

| Field           | Go Type          | DB Column        | DB Type        | Nullable | Notes                              |
|-----------------|------------------|------------------|----------------|----------|------------------------------------|
| PayramModel     | PayramModel      | (embedded)       | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Address         | string           | address          | varchar(200)   | NO       | Deployed address                   |
| BlockchainCode  | string           | blockchain_code  | varchar(20)    | NO       | Chain code reference               |
| TransactionHash | *string          | transaction_hash | varchar(200)   | YES      | Deployment TX hash                 |
| TransactionFee  | *decimal.Decimal | transaction_fee  | varchar(50)    | YES      | Deployment gas fee                 |
| BlockchainID    | uint             | blockchain_id    | bigint         | NO       | FK -> blockchains.id               |
| Status          | string           | status           | varchar(20)    | NO       | pending/broadcasted/active         |
| BroadcastedAt   | *time.Time       | broadcasted_at   | timestamptz    | YES      | Broadcast timestamp                |
| Blockchain      | *Blockchain      | —                | —              | —        | BelongsTo: Blockchain              |

**Unique:** (address, blockchain_id)
**Foreign keys:**
- `blockchain_id` -> `blockchains(id)`

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 21. EntrypointSCAddress

Smart contract entrypoint addresses (factory/router contracts).

- **Base model:** PayramModel
- **Table name:** `entrypoint_sc_addresses`

| Field       | Go Type     | DB Column   | DB Type       | Nullable | Notes                              |
|-------------|-------------|-------------|---------------|----------|------------------------------------|
| PayramModel | PayramModel | (embedded)  | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name        | string      | name        | varchar(100)  | NO       | Entrypoint name                    |
| Description | *string     | description | text          | YES      | Description                        |
| Address     | string      | address     | varchar(200)  | NO       | Contract address (unique)          |
| Status      | string      | status      | varchar(20)   | NO       | active/disabled (default 'active') |

**Unique:** `address`

---

### C. Wallet Management

#### 22. Wallet

HD wallets and smart contract wallets per member.

- **Base model:** PayramModel
- **Table name:** `wallets`

| Field                                       | Go Type                                       | DB Column                    | DB Type       | Nullable | Notes                              |
|---------------------------------------------|-----------------------------------------------|------------------------------|---------------|----------|------------------------------------|
| PayramModel                                 | PayramModel                                   | (embedded)                   | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name                                        | string                                        | name                         | varchar(255)  | NO       | Wallet name                        |
| Family                                      | *string                                       | family                       | varchar(30)   | YES      | Blockchain family                  |
| BlockchainCode                              | *string                                       | blockchain_code              | varchar(20)   | YES      | Chain code reference               |
| CurrencyCode                                | *string                                       | currency_code                | varchar(20)   | YES      | Currency code reference            |
| SecretType                                  | string                                        | secret_type                  | varchar(255)  | NO       | Secret storage type                |
| WalletType                                  | string                                        | wallet_type                  | varchar(255)  | NO       | Wallet type (hd, scw)             |
| WalletSubType                               | string                                        | wallet_sub_type              | varchar(255)  | NO       | Sub-type (eoa, etc.) default 'eoa' |
| PrivateKey                                  | *string                                       | private_key                  | text          | YES      | Encrypted private key              |
| PublicKey                                    | *string                                       | public_key                   | text          | YES      | Public key                         |
| Address                                     | *string                                       | address                      | text          | YES      | Wallet address                     |
| MemberID                                    | uint                                          | member_id                    | bigint        | NO       | FK -> members.id                   |
| Status                                      | string                                        | status                       | varchar(255)  | NO       | active/disabled (default 'active') |
| BlockchainFamilyID                          | *uint                                         | blockchain_family_id         | bigint        | YES      | FK -> blockchain_families.id       |
| BlockchainID                                | *uint                                         | blockchain_id                | bigint        | YES      | FK -> blockchains.id               |
| CurrencyID                                  | *uint                                         | currency_id                  | bigint        | YES      | FK -> currencies.id                |
| Default                                     | bool                                          | default                      | boolean       | NO       | Default wallet flag (default false)|
| BlockchainFamily                            | *BlockchainFamily                             | —                            | —             | —        | BelongsTo: BlockchainFamily        |
| Blockchain                                  | *Blockchain                                   | —                            | —             | —        | BelongsTo: Blockchain              |
| Currency                                    | *Currency                                     | —                            | —             | —        | BelongsTo: Currency                |
| Member                                      | *Member                                       | —                            | —             | —        | BelongsTo: Member                  |
| ExternalPlatformWalletBlockchainFamilies    | []ExternalPlatformWalletBlockchainFamily       | —                            | —             | —        | HasMany                            |
| SecretDataByte                              | []uint8                                       | secret_data_byte             | bytea         | YES      | Raw secret data                    |
| WalletFunctions                             | []WalletFunction                              | —                            | —             | —        | HasMany: WalletFunction            |
| WalletScws                                  | []WalletSCW                                   | —                            | —             | —        | HasMany: WalletSCW                 |
| WalletXpubs                                 | []WalletXpub                                  | —                            | —             | —        | HasMany: WalletXpub                |
| ABI                                         | *string                                       | abi                          | text          | YES      | Contract ABI (for SCW)             |
| Bytecode                                    | *string                                       | bytecode                     | text          | YES      | Contract bytecode (for SCW)        |
| FundSweeperAddress                          | *string                                       | fund_sweeper_address         | text          | YES      | Fund sweeper contract address      |
| Salt                                        | *string                                       | salt                         | text          | YES      | CREATE2 salt                       |
| MasterAddress                               | *string                                       | master_address               | text          | YES      | Master/factory address             |
| PrivateKeyHash                              | *string                                       | private_key_hash             | char(64)      | YES      | Hash of private key                |
| AddressHash                                 | *string                                       | address_hash                 | char(64)      | YES      | Hash of address                    |
| FundSweeperAddressHash                      | *string                                       | fund_sweeper_address_hash    | char(64)      | YES      | Hash of fund sweeper address       |
| Balance                                     | *decimal.Decimal                              | balance                      | numeric(38,18)| YES      | Computed balance (not persisted?)   |
| CanDelete                                   | *bool                                         | can_delete                   | boolean       | YES      | Computed field                     |

**Foreign keys:**
- `member_id` -> `members(id)`
- `blockchain_family_id` -> `blockchain_families(id)`
- `blockchain_id` -> `blockchains(id)`
- `currency_id` -> `currencies(id)`

**Relationships:**
- BelongsTo: Member, BlockchainFamily, Blockchain, Currency
- HasMany: ExternalPlatformWalletBlockchainFamily, WalletFunction, WalletSCW, WalletXpub

---

#### 23. WalletXpub

BIP32 extended public keys for HD wallet derivation.

- **Base model:** PayramModel
- **Table name:** `wallet_xpubs`

| Field                      | Go Type          | DB Column                  | DB Type        | Nullable | Notes                              |
|----------------------------|------------------|----------------------------|----------------|----------|------------------------------------|
| PayramModel                | PayramModel      | (embedded)                 | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Xpub                      | string           | xpub                       | varchar(500)   | NO       | Extended public key                |
| Family                    | string           | family                     | varchar(30)    | YES      | Blockchain family                  |
| Path                      | string           | path                       | text           | YES      | BIP32 derivation path              |
| WalletID                  | uint             | wallet_id                  | bigint         | NO       | FK -> wallets.id                   |
| BlockchainFamilyID        | uint             | blockchain_family_id       | bigint         | NO       | FK -> blockchain_families.id       |
| FundCollectorAddress      | *string          | fund_collector_address     | text           | YES      | Fund collector address             |
| Wallet                    | *Wallet          | —                          | —              | —        | BelongsTo: Wallet                  |
| BlockchainFamily          | *BlockchainFamily| —                          | —              | —        | BelongsTo: BlockchainFamily        |
| XpubHash                  | *string          | xpub_hash                  | char(64)       | YES      | Hash for uniqueness                |
| Balance                   | *decimal.Decimal | balance                    | numeric(38,18) | YES      | Computed balance                   |
| FundCollectorAddressHref  | *string          | fund_collector_address_href| varchar        | YES      | Explorer link (computed)           |

**Unique:** (wallet_id, blockchain_family_id), (xpub_hash)
**Foreign keys:**
- `wallet_id` -> `wallets(id)`
- `blockchain_family_id` -> `blockchain_families(id)`

---

#### 24. WalletSCW

Smart contract wallet (SCW) instances per blockchain.

- **Base model:** PayramModel
- **Table name:** `wallet_scws`

| Field                      | Go Type           | DB Column                    | DB Type        | Nullable | Notes                              |
|----------------------------|-------------------|------------------------------|----------------|----------|------------------------------------|
| PayramModel                | PayramModel       | (embedded)                   | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Family                     | string            | family                       | varchar(30)    | NO       | Blockchain family                  |
| BlockchainCode             | string            | blockchain_code              | varchar(20)    | NO       | Chain code                         |
| FundCollectorAddress       | string            | fund_collector_address       | text           | YES      | Fund collection address            |
| TransactionHash            | string            | transaction_hash             | text           | YES      | Deployment TX hash                 |
| WalletID                   | uint              | wallet_id                    | bigint         | NO       | FK -> wallets.id                   |
| BlockchainFamilyID         | uint              | blockchain_family_id         | bigint         | NO       | FK -> blockchain_families.id       |
| BlockchainID               | uint              | blockchain_id                | bigint         | NO       | FK -> blockchains.id               |
| Wallet                     | *Wallet           | —                            | —              | —        | BelongsTo: Wallet                  |
| BlockchainFamily           | *BlockchainFamily | —                            | —              | —        | BelongsTo: BlockchainFamily        |
| Blockchain                 | *Blockchain       | —                            | —              | —        | BelongsTo: Blockchain              |
| Balance                    | *decimal.Decimal  | balance                      | numeric(38,18) | YES      | Computed balance                   |
| FundSweeperAddress         | *string           | fund_sweeper_address         | text           | YES      | Sweeper address (computed)         |
| FundSweeperAddressHref     | *string           | fund_sweeper_address_href    | varchar        | YES      | Explorer link (computed)           |
| FundCollectorAddressHref   | *string           | fund_collector_address_href  | varchar        | YES      | Explorer link (computed)           |
| MasterAddressHref          | *string           | master_address_href          | varchar        | YES      | Explorer link (computed)           |
| TransactionHashHref        | *string           | transaction_hash_href        | varchar        | YES      | Explorer link (computed)           |

**Unique:** (blockchain_code, wallet_id), (transaction_hash)
**Foreign keys:**
- `wallet_id` -> `wallets(id)`
- `blockchain_family_id` -> `blockchain_families(id)`
- `blockchain_id` -> `blockchains(id)`

---

#### 25. WalletFunction

Capabilities/activities associated with a wallet.

- **Base model:** PayramModel
- **Table name:** `wallet_functions`

| Field         | Go Type     | DB Column       | DB Type       | Nullable | Notes                              |
|---------------|-------------|-----------------|---------------|----------|------------------------------------|
| PayramModel   | PayramModel | (embedded)      | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| ActivityType  | string      | activity_type   | varchar(255)  | NO       | Activity type identifier           |
| DisplayName   | string      | display_name    | varchar(255)  | NO       | Human-readable name                |
| Description   | *string     | description     | text          | YES      | Activity description               |
| SecretVaultID | *uint       | secret_vault_id | bigint        | YES      | FK -> secrets_vaults.id            |
| WalletID      | *uint       | wallet_id       | bigint        | YES      | FK -> wallets.id                   |
| Wallet        | *Wallet     | —               | —             | —        | BelongsTo: Wallet                  |

**Foreign keys:**
- `secret_vault_id` -> `secrets_vaults(id)`
- `wallet_id` -> `wallets(id)`

---

#### 26. SecretsVault

Encrypted key storage (AES-256 encrypted private keys).

- **Base model:** PayramModel
- **Table name:** `secrets_vaults`

| Field            | Go Type          | DB Column         | DB Type       | Nullable | Notes                              |
|------------------|------------------|-------------------|---------------|----------|------------------------------------|
| PayramModel      | PayramModel      | (embedded)        | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name             | string           | name              | varchar(255)  | NO       | Vault name                         |
| BlockchainCode   | *string          | blockchain_code   | varchar(20)   | YES      | Chain code reference               |
| CurrencyCode     | *string          | currency_code     | varchar(20)   | YES      | Currency code reference            |
| SecretType       | string           | secret_type       | varchar(255)  | NO       | Secret type identifier             |
| SecretData       | string           | secret_data       | text          | NO       | Encrypted secret data              |
| PublicKey         | *string          | public_key        | text          | YES      | Associated public key              |
| MemberID         | uint             | member_id         | bigint        | NO       | FK -> members.id                   |
| EncryptionScheme | *string          | encryption_scheme | varchar(255)  | YES      | Encryption scheme identifier       |
| Status           | string           | status            | varchar(255)  | YES      | active/disabled (default 'active') |
| Member           | *Member          | —                 | —             | —        | BelongsTo: Member                  |
| VaultActivities  | []WalletFunction | —                 | —             | —        | HasMany: WalletFunction.SecretVaultID |

**Foreign keys:**
- `member_id` -> `members(id)`

**Relationships:**
- BelongsTo: Member
- HasMany: WalletFunction (via SecretVaultID)

---

#### 27. ExternalPlatformWalletBlockchainFamily

Links projects to specific wallets per blockchain family.

- **Base model:** None (timestamps only, composite key)
- **Table name:** `external_platform_wallet_blockchain_families`

| Field                 | Go Type           | DB Column            | DB Type       | Nullable | Notes                              |
|-----------------------|-------------------|----------------------|---------------|----------|------------------------------------|
| CreatedAt             | time.Time         | created_at           | timestamptz   | YES      | Timestamp                          |
| UpdatedAt             | time.Time         | updated_at           | timestamptz   | YES      | Timestamp                          |
| ExternalPlatformID    | uint              | external_platform_id | bigint        | NO       | FK -> external_platforms.id        |
| WalletName            | string            | wallet_name          | varchar(255)  | YES      | Wallet name reference              |
| Family                | string            | family               | varchar(20)   | YES      | Blockchain family code             |
| WalletID              | uint              | wallet_id            | bigint        | NO       | FK -> wallets.id                   |
| BlockchainFamilyID    | uint              | blockchain_family_id | bigint        | NO       | FK -> blockchain_families.id       |
| ExternalPlatform      | *ExternalPlatform | —                    | —             | —        | BelongsTo: ExternalPlatform        |
| Wallet                | *Wallet           | —                    | —             | —        | BelongsTo: Wallet                  |
| BlockchainFamily      | *BlockchainFamily | —                    | —             | —        | BelongsTo: BlockchainFamily        |
| ChangeOrDeleteRisky   | bool              | change_or_delete_risky| boolean      | YES      | Safety flag (computed)             |

**Unique:** (external_platform_id, blockchain_family_id)
**Foreign keys:**
- `external_platform_id` -> `external_platforms(id)`
- `wallet_id` -> `wallets(id)`
- `blockchain_family_id` -> `blockchain_families(id)`

---

#### 28. DepositAddress

Addresses assigned to specific members for receiving payments.

- **Base model:** PayramModel
- **Table name:** `deposit_addresses`

| Field              | Go Type          | DB Column            | DB Type       | Nullable | Notes                              |
|--------------------|------------------|----------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel      | (embedded)           | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Address            | string           | address              | varchar(200)  | NO       | Deposit address                    |
| AddressLower       | string           | address_lower        | varchar(200)  | NO       | Lowercase address (default '')     |
| Family             | string           | family               | varchar(30)   | NO       | Blockchain family                  |
| Status             | string           | status               | varchar(30)   | NO       | active/disabled (default 'active') |
| MemberID           | uint             | member_id            | bigint        | NO       | FK -> members.id                   |
| WalletID           | *uint            | wallet_id            | bigint        | YES      | FK -> wallets.id                   |
| BlockchainFamilyID | uint             | blockchain_family_id | bigint        | NO       | FK -> blockchain_families.id       |
| Member             | Member           | —                    | —             | —        | BelongsTo: Member                  |
| BlockchainFamily   | BlockchainFamily | —                    | —             | —        | BelongsTo: BlockchainFamily        |
| Wallet             | Wallet           | —                    | —             | —        | BelongsTo: Wallet                  |

**Foreign keys:**
- `member_id` -> `members(id)`
- `wallet_id` -> `wallets(id)`
- `blockchain_family_id` -> `blockchain_families(id)`

---

### D. Address Pool

#### 29. AddressPool

Pre-generated blockchain addresses for fast assignment.

- **Base model:** PayramModel
- **Table name:** `address_pools`

| Field              | Go Type           | DB Column            | DB Type       | Nullable | Notes                              |
|--------------------|-------------------|----------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel       | (embedded)           | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Address            | string            | address              | varchar(200)  | NO       | Unique blockchain address          |
| PrivateKey         | *string           | private_key          | text          | YES      | Encrypted private key              |
| Family             | string            | family               | varchar(30)   | NO       | Blockchain family                  |
| BlockchainFamilyID | uint             | blockchain_family_id | bigint        | NO       | FK -> blockchain_families.id       |
| Path               | string            | path                 | varchar(100)  | YES      | HD derivation path                 |
| PathIndex          | uint              | path_index           | bigint        | NO       | Derivation index (default 0)       |
| Status             | string            | status               | varchar(20)   | NO       | open/assigned/deployed (default 'open') |
| WalletID           | *uint             | wallet_id            | bigint        | YES      | FK -> wallets.id                   |
| Salt               | *string           | salt                 | text          | YES      | CREATE2 salt                       |
| Type               | string            | type                 | varchar(20)   | NO       | eoa/scw (default 'eoa')           |
| SaltCounter        | *uint             | salt_counter         | bigint        | YES      | Salt generation counter            |
| BlockchainFamily   | *BlockchainFamily | —                    | —             | —        | BelongsTo: BlockchainFamily        |
| Wallet             | *Wallet           | —                    | —             | —        | BelongsTo: Wallet                  |
| AccountAddresses   | []AccountAddress  | —                    | —             | —        | HasMany: AccountAddress (via address) |
| UTXOS              | []UTXO            | —                    | —             | —        | HasMany: UTXO (via address)        |

**Unique:** `address`
**Foreign keys:**
- `blockchain_family_id` -> `blockchain_families(id)`
- `wallet_id` -> `wallets(id)`

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 30. AccountAddress

Balance tracking per blockchain deposit address.

- **Base model:** PayramModel
- **Table name:** `account_addresses`

| Field                  | Go Type           | DB Column                | DB Type        | Nullable | Notes                              |
|------------------------|-------------------|--------------------------|----------------|----------|------------------------------------|
| PayramModel            | PayramModel       | (embedded)               | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Address                | string            | address                  | varchar(200)   | NO       | Blockchain address                 |
| Balance                | decimal.Decimal   | balance                  | numeric(38,18) | NO       | Current balance (default 0.0)      |
| Locked                 | decimal.Decimal   | locked                   | numeric(38,18) | NO       | Locked during sweep (default 0.0)  |
| DepositFee             | decimal.Decimal   | deposit_fee              | numeric(38,18) | NO       | Accumulated deposit fees           |
| ApprovalFee            | decimal.Decimal   | approval_fee             | numeric(38,18) | NO       | ERC20 approval fees                |
| MemberID               | uint              | member_id                | bigint         | NO       | FK -> members.id                   |
| Status                 | string            | status                   | varchar(20)    | NO       | active/locked (default 'active')   |
| BlockchainCode         | string            | blockchain_code          | varchar(20)    | NO       | Chain code reference               |
| CurrencyCode           | string            | currency_code            | varchar(20)    | NO       | Currency code reference            |
| CurrencyID             | uint              | currency_id              | bigint         | NO       | FK -> currencies.id                |
| BlockchainID           | uint              | blockchain_id            | bigint         | NO       | FK -> blockchains.id               |
| SweepingTransactionID  | *uint             | sweeping_transaction_id  | bigint         | YES      | FK -> sweep_transactions.id        |
| SweepingTransaction    | *SweepTransaction | —                        | —              | —        | BelongsTo: SweepTransaction        |
| Member                 | *Member           | —                        | —              | —        | BelongsTo: Member                  |
| Currency               | *Currency         | —                        | —              | —        | BelongsTo: Currency                |
| Blockchain             | *Blockchain       | —                        | —              | —        | BelongsTo: Blockchain              |
| AddressPool            | *AddressPool      | —                        | —              | —        | BelongsTo: AddressPool (via address) |

**Foreign keys:**
- `member_id` -> `members(id)`
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`
- `address` -> `address_pools(address)`
- `sweeping_transaction_id` -> `sweep_transactions(id)`

---

### E. Deposits & Transactions

#### 31. Deposit

On-chain incoming transactions detected by block monitors.

- **Base model:** PayramModel
- **Table name:** `deposits`

| Field          | Go Type          | DB Column       | DB Type        | Nullable | Notes                              |
|----------------|------------------|-----------------|----------------|----------|------------------------------------|
| PayramModel    | PayramModel      | (embedded)      | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID       | uint             | member_id       | bigint         | NO       | FK -> members.id                   |
| CurrencyCode   | string           | currency_code   | varchar(20)    | NO       | Currency code reference            |
| BlockchainCode | string           | blockchain_code | varchar(20)    | NO       | Chain code reference               |
| Amount         | decimal.Decimal  | amount          | numeric(38,18) | NO       | Deposit amount                     |
| PriceInUSD     | *decimal.Decimal | price_in_usd    | numeric(38,18) | YES      | Price at time of deposit           |
| AmountInUSD    | *decimal.Decimal | amount_in_usd   | numeric(38,18) | YES      | USD equivalent                     |
| Fee            | *decimal.Decimal | fee             | numeric(38,18) | YES      | Deposit fee                        |
| Address        | *string          | address         | varchar(200)   | YES      | Receiving address                  |
| FromAddresses  | *string          | from_addresses  | varchar(200)   | YES      | Sender addresses                   |
| TxHash         | string           | tx_hash         | varchar(200)   | NO       | Transaction hash                   |
| UniqueTxHash   | string           | unique_tx_hash  | varchar(250)   | NO       | Unique TX identifier               |
| BlockHash      | *string          | block_hash      | varchar(200)   | YES      | Block hash                         |
| Status         | string           | status          | varchar(20)    | NO       | pending/accepted/skipped/rejected  |
| BlockNumber    | uint64           | block_number    | bigint         | NO       | Block number                       |
| Type           | string           | type            | varchar(20)    | NO       | coin/token (default 'coin')        |
| Timestamp      | time.Time        | timestamp       | timestamptz    | YES      | Block timestamp                    |
| CurrencyID     | uint             | currency_id     | bigint         | NO       | FK -> currencies.id                |
| BlockchainID   | uint             | blockchain_id   | bigint         | NO       | FK -> blockchains.id               |
| Member         | Member           | —               | —              | —        | BelongsTo: Member                  |
| Currency       | Currency         | —               | —              | —        | BelongsTo: Currency                |
| Blockchain     | Blockchain       | —               | —              | —        | BelongsTo: Blockchain              |
| Metadata       | *string          | metadata        | jsonb          | YES      | Additional metadata                |
| Origin         | *string          | origin          | varchar(200)   | YES      | Deposit origin                     |

**Foreign keys:**
- `member_id` -> `members(id)`
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 32. Withdrawal

Merchant-initiated outbound crypto transfers (payouts).

- **Base model:** PayramModel
- **Table name:** `withdrawals`

| Field               | Go Type           | DB Column              | DB Type        | Nullable | Notes                              |
|---------------------|-------------------|------------------------|----------------|----------|------------------------------------|
| PayramModel         | PayramModel       | (embedded)             | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| CurrencyCode        | string            | currency_code          | varchar(200)   | NO       | Currency code                      |
| BlockchainCode      | string            | blockchain_code        | varchar(200)   | NO       | Chain code                         |
| Amount              | decimal.Decimal   | amount                 | numeric(38,18) | NO       | Withdrawal amount                  |
| PriceInUSD          | *decimal.Decimal  | price_in_usd           | numeric(38,18) | YES      | Price at time of withdrawal        |
| AmountInUSD         | *decimal.Decimal  | amount_in_usd          | numeric(38,18) | YES      | USD equivalent                     |
| Fee                 | *decimal.Decimal  | fee                    | numeric(38,18) | YES      | Withdrawal fee                     |
| FromAddress         | *string           | from_address           | varchar(200)   | YES      | Source address                     |
| ToAddress           | string            | to_address             | varchar(200)   | NO       | Destination address                |
| RecipientEmail      | *string           | recipient_email        | varchar(255)   | YES      | Recipient email notification       |
| TokenAddress        | *string           | token_address          | varchar(200)   | YES      | ERC20 token address                |
| TxHash              | *string           | tx_hash                | varchar(200)   | YES      | Transaction hash                   |
| UniqueTxHash        | *string           | unique_tx_hash         | varchar(250)   | YES      | Unique TX identifier               |
| TxIndex             | *int              | tx_index               | bigint         | YES      | Transaction index in block         |
| BlockHash           | *string           | block_hash             | varchar(200)   | YES      | Block hash                         |
| Status              | string            | status                 | varchar(200)   | NO       | Withdrawal status (default 'pending') |
| BlockNumber         | *uint64           | block_number           | bigint         | YES      | Block number                       |
| Type                | string            | type                   | varchar(20)    | NO       | Transfer type                      |
| CurrencyType        | string            | currency_type          | varchar(200)   | NO       | coin/token (default 'coin')        |
| Timestamp           | *time.Time        | timestamp              | timestamptz    | YES      | Block timestamp                    |
| Attributes          | *string           | attributes             | text           | YES      | Additional attributes              |
| SignedTx            | *string           | signed_tx              | text           | YES      | Signed transaction data            |
| FailureReason       | *string           | failure_reason         | text           | YES      | Error description on failure       |
| WebhookStatus       | *string           | webhook_status         | text           | YES      | Webhook delivery status            |
| RetryCount          | int               | retry_count            | bigint         | NO       | Retry attempts (default 0)         |
| LastRetryAt         | *time.Time        | last_retry_at          | timestamptz    | YES      | Last retry timestamp               |
| MemberID            | uint              | member_id              | bigint         | NO       | FK -> members.id                   |
| CreatedByMemberID   | *uint             | created_by_member_id   | bigint         | YES      | FK -> members.id                   |
| CreatedByProjectID  | *uint             | created_by_project_id  | bigint         | YES      | FK -> external_platforms.id        |
| RejectedByMemberID  | *uint             | rejected_by_member_id  | bigint         | YES      | FK -> members.id                   |
| ApprovedByMemberID  | *uint             | approved_by_member_id  | bigint         | YES      | FK -> members.id                   |
| ExternalPlatformID  | *uint             | external_platform_id   | bigint         | YES      | FK -> external_platforms.id        |
| CurrencyID          | uint              | currency_id            | bigint         | NO       | FK -> currencies.id                |
| BlockchainID        | uint              | blockchain_id          | bigint         | NO       | FK -> blockchains.id               |
| Member              | *Member           | —                      | —              | —        | BelongsTo: Member (owner)          |
| CreatedByMember     | *Member           | —                      | —              | —        | BelongsTo: Member (creator)        |
| CreatedByProject    | *ExternalPlatform | —                      | —              | —        | BelongsTo: ExternalPlatform        |
| RejectedByMember    | *Member           | —                      | —              | —        | BelongsTo: Member (rejector)       |
| ApprovedByMember    | *Member           | —                      | —              | —        | BelongsTo: Member (approver)       |
| Currency            | *Currency         | —                      | —              | —        | BelongsTo: Currency                |
| Blockchain          | *Blockchain       | —                      | —              | —        | BelongsTo: Blockchain              |
| CreatedBy           | string            | created_by             | varchar(50)    | NO       | Creator type (default 'system')    |
| ExternalPlatform    | *ExternalPlatform | —                      | —              | —        | BelongsTo: ExternalPlatform        |

**Foreign keys:**
- `member_id` -> `members(id)`
- `created_by_member_id` -> `members(id)`
- `created_by_project_id` -> `external_platforms(id)`
- `rejected_by_member_id` -> `members(id)`
- `approved_by_member_id` -> `members(id)`
- `external_platform_id` -> `external_platforms(id)`
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 33. Withdraw

Bitcoin-specific withdrawal transactions (BTC has special UTXO handling).

- **Base model:** PayramModel
- **Table name:** `withdraws`

| Field                 | Go Type          | DB Column               | DB Type        | Nullable | Notes                              |
|-----------------------|------------------|-------------------------|----------------|----------|------------------------------------|
| PayramModel           | PayramModel      | (embedded)              | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| TxHash                | string           | tx_hash                 | varchar(200)   | NO       | Transaction hash                   |
| CurrencyCode          | string           | currency_code           | varchar(20)    | NO       | Currency code                      |
| BlockNumber           | *uint64          | block_number            | bigint         | YES      | Block number                       |
| Status                | *string          | status                  | varchar(20)    | NO       | initiated/completed (default 'initiated') |
| Amount                | decimal.Decimal  | amount                  | numeric(38,18) | NO       | Withdrawal amount                  |
| PriceInUSD            | *decimal.Decimal | price_in_usd            | numeric(38,18) | YES      | Price at time of withdrawal        |
| AmountInUSD           | *decimal.Decimal | amount_in_usd           | numeric(38,18) | YES      | USD equivalent                     |
| AmountInChangeAddress | *decimal.Decimal | amount_in_change_address| numeric(38,18) | YES      | Change returned to sender          |
| Fee                   | *decimal.Decimal | fee                     | numeric(38,18) | YES      | Network fee                        |
| Address               | *string          | address                 | varchar(200)   | YES      | Destination address                |
| AmountToPayram        | *decimal.Decimal | amount_to_payram        | numeric(38,18) | YES      | Platform fee amount                |
| PayramAddress         | *string          | payram_address          | varchar(200)   | YES      | Platform fee address               |
| BlockHash             | *string          | block_hash              | varchar(200)   | YES      | Block hash                         |
| Timestamp             | *time.Time       | timestamp               | timestamptz    | YES      | Block timestamp                    |
| CurrencyID            | uint             | currency_id             | bigint         | NO       | FK -> currencies.id                |
| BlockchainID          | uint             | blockchain_id           | bigint         | NO       | FK -> blockchains.id               |
| Currency              | Currency         | —                       | —              | —        | BelongsTo: Currency                |
| Blockchain            | Blockchain       | —                       | —              | —        | BelongsTo: Blockchain              |

**Foreign keys:**
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`

---

#### 34. InternalBlockchainTransaction

Internal record of all blockchain transactions (gas transfers, sweeps, etc.).

- **Base model:** PayramModel
- **Table name:** `internal_blockchain_transactions`

| Field           | Go Type          | DB Column        | DB Type        | Nullable | Notes                              |
|-----------------|------------------|------------------|----------------|----------|------------------------------------|
| PayramModel     | PayramModel      | (embedded)       | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID        | *uint            | member_id        | bigint         | YES      | FK -> members.id                   |
| BlockchainCode  | string           | blockchain_code  | varchar(20)    | NO       | Chain code                         |
| CurrencyCode    | *string          | currency_code    | varchar(20)    | YES      | Currency code                      |
| Amount          | *decimal.Decimal | amount           | numeric(38,18) | YES      | Transaction amount                 |
| PriceInUSD      | *decimal.Decimal | price_in_usd     | numeric(38,18) | YES      | Price at time of TX                |
| AmountInUSD     | *decimal.Decimal | amount_in_usd    | numeric(38,18) | YES      | USD equivalent                     |
| Fee             | *decimal.Decimal | fee              | numeric(38,18) | YES      | Gas fee                            |
| FromAddress     | *string          | from_address     | varchar(200)   | YES      | Source address                     |
| ToAddress       | *string          | to_address       | varchar(200)   | YES      | Destination address                |
| TxHash          | string           | tx_hash          | varchar(200)   | NO       | Transaction hash                   |
| UniqueTxHash    | string           | unique_tx_hash   | varchar(250)   | NO       | Unique TX identifier               |
| BlockHash       | *string          | block_hash       | varchar(200)   | YES      | Block hash                         |
| Status          | string           | status           | varchar(20)    | NO       | sent/confirmed (default 'sent')    |
| BlockNumber     | *uint64          | block_number     | bigint         | YES      | Block number                       |
| Type            | *string          | type             | varchar(20)    | NO       | coin/token (default 'coin')        |
| TransactionType | *string          | transaction_type | varchar(20)    | NO       | coin/token (default 'coin')        |
| Timestamp       | *time.Time       | timestamp        | timestamptz    | YES      | Block timestamp                    |
| BlockchainID    | uint             | blockchain_id    | bigint         | NO       | FK -> blockchains.id               |
| CurrencyID      | *uint            | currency_id      | bigint         | YES      | FK -> currencies.id                |
| DepositID       | *uint            | deposit_id       | bigint         | YES      | FK -> deposits.id                  |
| Member          | *Member          | —                | —              | —        | BelongsTo: Member                  |
| Currency        | *Currency        | —                | —              | —        | BelongsTo: Currency                |
| Blockchain      | *Blockchain      | —                | —              | —        | BelongsTo: Blockchain              |
| Deposit         | *Deposit         | —                | —              | —        | BelongsTo: Deposit                 |

**Foreign keys:**
- `member_id` -> `members(id)`
- `blockchain_id` -> `blockchains(id)`
- `currency_id` -> `currencies(id)`
- `deposit_id` -> `deposits(id)`

---

#### 35. MissedDeposit

Deposits detected on-chain but not matched to any known address.

- **Base model:** PayramModel
- **Table name:** `missed_deposits`

| Field           | Go Type     | DB Column        | DB Type       | Nullable | Notes                              |
|-----------------|-------------|------------------|---------------|----------|------------------------------------|
| PayramModel     | PayramModel | (embedded)       | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| BlockchainCode  | string      | blockchain_code  | varchar(20)   | NO       | Chain code                         |
| TransactionHash | string      | transaction_hash | varchar(200)  | NO       | TX hash                            |
| Status          | string      | status           | varchar(20)   | NO       | Status                             |
| BlockNumber     | *uint64     | block_number     | bigint        | YES      | Block number                       |

---

### F. UTXO & Sweeps

#### 36. UTXO

Bitcoin UTXO tracking for BTC sweeps and withdrawals.

- **Base model:** PayramModel
- **Table name:** `utxos`

| Field                 | Go Type           | DB Column               | DB Type        | Nullable | Notes                              |
|-----------------------|-------------------|-------------------------|----------------|----------|------------------------------------|
| PayramModel           | PayramModel       | (embedded)              | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| DepositID             | uint              | deposit_id              | bigint         | NO       | FK -> deposits.id                  |
| WithdrawID            | uint              | withdraw_id             | bigint         | YES      | FK -> withdraws.id                 |
| Amount                | decimal.Decimal   | amount                  | numeric(38,18) | NO       | UTXO amount                        |
| Address               | string            | address                 | varchar(200)   | YES      | UTXO address                       |
| TxHash                | string            | tx_hash                 | varchar(200)   | NO       | Transaction hash                   |
| UniqueTxHash          | string            | unique_tx_hash          | varchar(250)   | NO       | Unique TX identifier               |
| Index                 | uint32            | index                   | bigint         | NO       | Output index                       |
| ScriptPubKeyHex       | string            | script_pub_key_hex      | varchar(300)   | NO       | Script public key hex              |
| RawTransactionHex     | string            | raw_transaction_hex     | text           | NO       | Raw TX hex                         |
| Status                | string            | status                  | varchar(20)    | NO       | unspent/spent (default 'unspent')  |
| CurrencyCode          | string            | currency_code           | varchar(20)    | NO       | Currency code                      |
| BlockchainCode        | string            | blockchain_code         | varchar(20)    | NO       | Chain code                         |
| CurrencyID            | uint              | currency_id             | bigint         | NO       | FK -> currencies.id                |
| BlockchainID          | uint              | blockchain_id           | bigint         | NO       | FK -> blockchains.id               |
| Deposit               | *Deposit          | —                       | —              | —        | BelongsTo: Deposit                 |
| SweepingTransactionID | *uint             | sweeping_transaction_id | bigint         | YES      | FK -> sweep_transactions.id        |
| SweepingTransaction   | *SweepTransaction | —                       | —              | —        | BelongsTo: SweepTransaction        |
| SweepID               | *uint             | sweep_id                | bigint         | YES      | FK -> sweeps.id                    |
| Sweep                 | *Sweep            | —                       | —              | —        | BelongsTo: Sweep                   |
| AddressPool           | *AddressPool      | —                       | —              | —        | BelongsTo: AddressPool (via address) |

**Foreign keys:**
- `deposit_id` -> `deposits(id)`
- `address` -> `address_pools(address)`
- `sweeping_transaction_id` -> `sweep_transactions(id)`
- `sweep_id` -> `sweeps(id)`

---

#### 37. Sweep

Individual on-chain sweep transactions (confirmed on blockchain).

- **Base model:** PayramModel
- **Table name:** `sweeps`

| Field           | Go Type          | DB Column        | DB Type        | Nullable | Notes                              |
|-----------------|------------------|------------------|----------------|----------|------------------------------------|
| PayramModel     | PayramModel      | (embedded)       | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| CurrencyCode    | string           | currency_code    | varchar(20)    | NO       | Currency code                      |
| BlockchainCode  | string           | blockchain_code  | varchar(20)    | NO       | Chain code                         |
| Amount          | *decimal.Decimal | amount           | numeric(38,18) | YES      | Sweep amount                       |
| PriceInUSD      | *decimal.Decimal | price_in_usd     | numeric(38,18) | YES      | Price at time of sweep             |
| AmountInUSD     | *decimal.Decimal | amount_in_usd    | numeric(38,18) | YES      | USD equivalent                     |
| Fee             | *decimal.Decimal | fee              | numeric(38,18) | YES      | Network fee                        |
| FromAddress     | *string          | from_address     | varchar(200)   | YES      | Source (deposit) address           |
| ToAddress       | *string          | to_address       | varchar(200)   | YES      | Destination (cold storage)         |
| ContractAddress | *string          | contract_address | varchar(200)   | YES      | SmartSweep contract address        |
| TokenAddress    | *string          | token_address    | varchar(200)   | YES      | ERC20 token address                |
| TxHash          | string           | tx_hash          | varchar(200)   | NO       | Transaction hash                   |
| UniqueTxHash    | string           | unique_tx_hash   | varchar(250)   | NO       | Unique TX identifier               |
| TxIndex         | *int             | tx_index         | bigint         | YES      | Transaction index in block         |
| BlockHash       | *string          | block_hash       | varchar(200)   | YES      | Block hash                         |
| Status          | string           | status           | varchar(200)   | NO       | pending/confirming/completed       |
| BlockNumber     | *uint64          | block_number     | bigint         | NO       | Block number                       |
| Type            | *string          | type             | varchar(20)    | YES      | coin/token (default 'coin')        |
| Timestamp       | *time.Time       | timestamp        | timestamptz    | YES      | Block timestamp                    |
| Attributes      | *string          | attributes       | text           | YES      | Additional attributes              |
| CurrencyID      | uint             | currency_id      | bigint         | NO       | FK -> currencies.id                |
| BlockchainID    | uint             | blockchain_id    | bigint         | NO       | FK -> blockchains.id               |
| Currency        | *Currency        | —                | —              | —        | BelongsTo: Currency                |
| Blockchain      | *Blockchain      | —                | —              | —        | BelongsTo: Blockchain              |

**Foreign keys:**
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 38. SweepTransaction

Batched sweep operations from deposit addresses to cold storage.

- **Base model:** PayramModel
- **Table name:** `sweep_transactions`

| Field                  | Go Type           | DB Column               | DB Type        | Nullable | Notes                              |
|------------------------|-------------------|-------------------------|----------------|----------|------------------------------------|
| PayramModel            | PayramModel       | (embedded)              | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| BlockchainCode         | string            | blockchain_code         | varchar(20)    | NO       | Chain code                         |
| CurrencyCode           | string            | currency_code           | varchar(20)    | NO       | Currency code                      |
| CurrencyID             | uint              | currency_id             | bigint         | NO       | FK -> currencies.id                |
| BlockchainID           | uint              | blockchain_id           | bigint         | NO       | FK -> blockchains.id               |
| Status                 | string            | status                  | varchar(20)    | NO       | pending/completed (default 'pending') |
| TxHash                 | *string           | tx_hash                 | varchar(200)   | YES      | Transaction hash                   |
| SignedPayload          | *string           | signed_payload          | text           | YES      | Signed transaction payload         |
| RawPayload             | *string           | raw_payload             | text           | YES      | Raw transaction payload            |
| PayloadType            | string            | payload_type            | varchar(20)    | NO       | Payload type                       |
| WalletID               | uint              | wallet_id               | bigint         | NO       | FK -> wallets.id                   |
| FeePercentage          | *decimal.Decimal  | fee_percentage          | numeric(10,6)  | YES      | Platform fee percentage            |
| Currency               | *Currency         | —                       | —              | —        | BelongsTo: Currency                |
| Blockchain             | *Blockchain       | —                       | —              | —        | BelongsTo: Blockchain              |
| UTXOs                  | []UTXO            | —                       | —              | —        | HasMany: UTXO.SweepingTransactionID |
| AccountAddresses       | []AccountAddress  | —                       | —              | —        | HasMany: AccountAddress.SweepingTransactionID |
| Wallet                 | *Wallet           | —                       | —              | —        | BelongsTo: Wallet                  |
| PayramFeeAddress       | *string           | payram_fee_address      | varchar        | YES      | Platform fee address (computed)    |
| BlockchainType         | *string           | blockchain_type         | varchar        | YES      | Blockchain type (computed)         |
| FundCollectorAddress   | *string           | fund_collector_address  | varchar        | YES      | Fund collector address (computed)  |
| NumberOfUTXOs          | *int64            | number_of_utxos         | bigint         | YES      | UTXO count (computed)              |
| TotalAmount            | *decimal.Decimal  | total_amount            | numeric(38,18) | YES      | Total sweep amount (computed)      |
| PayramFees             | *decimal.Decimal  | payram_fees             | numeric(38,18) | YES      | Platform fees (computed)           |

**Foreign keys:**
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`
- `wallet_id` -> `wallets(id)`

**Relationships:**
- BelongsTo: Currency, Blockchain, Wallet
- HasMany: UTXO, AccountAddress

---

#### 39. WithdrawDepositsBTC

Bitcoin-specific withdrawal UTXO tracking.

- **Base model:** PayramModel
- **Table name:** `withdraw_deposits_btcs`

| Field        | Go Type         | DB Column        | DB Type        | Nullable | Notes                              |
|--------------|-----------------|------------------|----------------|----------|------------------------------------|
| PayramModel  | PayramModel     | (embedded)       | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| DepositID    | uint            | deposit_id       | bigint         | NO       | FK -> deposits.id                  |
| WithdrawID   | uint            | withdraw_id      | bigint         | YES      | FK -> withdraws.id                 |
| Amount       | decimal.Decimal | amount           | numeric(38,18) | NO       | Amount                             |
| Address      | string          | address          | varchar(200)   | YES      | Address                            |
| TxHash       | string          | tx_hash          | varchar(200)   | NO       | Transaction hash                   |
| UniqueTxHash | string          | unique_tx_hash   | varchar(250)   | NO       | Unique TX identifier               |
| Status       | string          | status           | varchar(20)    | NO       | unspent/spent (default 'unspent')  |
| CurrencyID   | uint            | currency_id      | bigint         | NO       | FK -> currencies.id                |
| BlockchainID | uint            | blockchain_id    | bigint         | NO       | FK -> blockchains.id               |

**Foreign keys:**
- `deposit_id` -> `deposits(id)`
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`

---

### G. Payment Processing

#### 40. PaymentRequest

Core payment table -- each payment link/invoice creates one record.

- **Base model:** PayramModel
- **Table name:** `payment_requests`

| Field                | Go Type           | DB Column              | DB Type        | Nullable | Notes                              |
|----------------------|-------------------|------------------------|----------------|----------|------------------------------------|
| PayramModel          | PayramModel       | (embedded)             | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Amount               | *decimal.Decimal  | amount                 | numeric(38,18) | YES      | Requested amount (in crypto)       |
| AmountInUSD          | decimal.Decimal   | amount_in_usd          | numeric(38,18) | NO       | USD equivalent                     |
| FilledAmount         | *decimal.Decimal  | filled_amount          | numeric(38,18) | YES      | Amount received so far             |
| FilledAmountInUSD    | *decimal.Decimal  | filled_amount_in_usd   | numeric(38,18) | YES      | USD value received                 |
| SponsoredAmount      | decimal.Decimal   | sponsored_amount       | numeric(38,18) | NO       | Sponsored/subsidized amount        |
| SponsoredAmountInUSD | decimal.Decimal   | sponsored_amount_in_usd| numeric(38,18) | NO       | Sponsored USD value                |
| CurrencyCode         | *string           | currency_code          | varchar(20)    | YES      | Currency code                      |
| BlockchainCode       | *string           | blockchain_code        | varchar(20)    | YES      | Chain code                         |
| InvoiceID            | string            | invoice_id             | varchar(100)   | NO       | Invoice identifier                 |
| ReferenceID          | string            | reference_id           | varchar(100)   | NO       | External reference                 |
| Status               | string            | status                 | varchar(20)    | NO       | open/cancelled/filled/etc.         |
| ExpireAt             | *time.Time        | expire_at              | timestamptz    | YES      | Payment expiration                 |
| MemberID             | uint              | member_id              | bigint         | NO       | FK -> members.id                   |
| CurrencyID           | *uint             | currency_id            | bigint         | YES      | FK -> currencies.id                |
| BlockchainID         | *uint             | blockchain_id          | bigint         | YES      | FK -> blockchains.id               |
| DepositID            | *uint             | deposit_id             | bigint         | YES      | FK -> deposits.id                  |
| CreatedBy            | string            | created_by             | varchar(20)    | NO       | Creator type (default 'user')      |
| ExternalPlatformID   | uint              | external_platform_id   | bigint         | NO       | FK -> external_platforms.id        |
| Member               | *Member           | —                      | —              | —        | BelongsTo: Member                  |
| Currency             | *Currency         | —                      | —              | —        | BelongsTo: Currency                |
| Deposit              | *Deposit          | —                      | —              | —        | BelongsTo: Deposit                 |
| Blockchain           | *Blockchain       | —                      | —              | —        | BelongsTo: Blockchain              |
| ExternalPlatform     | *ExternalPlatform | —                      | —              | —        | BelongsTo: ExternalPlatform        |

**Foreign keys:**
- `member_id` -> `members(id)`
- `currency_id` -> `currencies(id)`
- `blockchain_id` -> `blockchains(id)`
- `deposit_id` -> `deposits(id)`
- `external_platform_id` -> `external_platforms(id)`

**State machine:** See [Section 6](#6-state-machine-diagrams)

---

#### 41. PaymentChannel

Fiat onramp payment channel definitions.

- **Base model:** PayramModel
- **Table name:** `payment_channels`

| Field                              | Go Type                          | DB Column                        | DB Type       | Nullable | Notes                              |
|------------------------------------|----------------------------------|----------------------------------|---------------|----------|------------------------------------|
| PayramModel                        | PayramModel                      | (embedded)                       | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name                               | string                           | name                             | varchar(200)  | NO       | Channel name                       |
| ChannelType                        | PaymentChannelType               | channel_type                     | varchar(50)   | NO       | Channel type enum                  |
| Status                             | PaymentChannelStatus             | status                           | varchar(20)   | NO       | active/disabled (default 'active') |
| Configuration                      | *PaymentChannelConfiguration     | configuration                    | json          | YES      | Channel config JSON                |
| Metadata                           | *PaymentChannelMetadata          | metadata                         | json          | YES      | Channel metadata JSON              |
| ApiKey                             | *string                          | api_key                          | varchar       | YES      | Channel API key                    |
| DisplayName                        | string                           | display_name                     | varchar       | NO       | Human-readable name                |
| Description                        | *string                          | description                      | text          | YES      | Channel description                |
| Icon                               | *string                          | icon                             | varchar       | YES      | Icon URL/path                      |
| DisplayOrder                       | int                              | display_order                    | bigint        | NO       | UI display order                   |
| IsDefault                          | bool                             | is_default                       | boolean       | NO       | Default channel flag               |
| DisabledPaymentChannelProjects     | []DisabledPaymentChannelProject  | —                                | —             | —        | HasMany                            |
| Disabled                           | *bool                            | disabled                         | boolean       | YES      | Computed disabled flag             |

**Relationships:**
- HasMany: DisabledPaymentChannelProject

---

#### 42. PaymentChannelConfiguration

Value object for payment channel configuration (embedded JSON).

- **Base model:** None (value object)
- **Table name:** (embedded in payment_channels)

| Field | Go Type         | DB Column     | DB Type | Nullable | Notes             |
|-------|-----------------|---------------|---------|----------|-------------------|
| Value | json.RawMessage | configuration | json    | YES      | Raw JSON config   |

---

#### 43. PaymentChannelMetadata

Value object for payment channel metadata (embedded JSON).

- **Base model:** None (value object)
- **Table name:** (embedded in payment_channels)

| Field | Go Type         | DB Column | DB Type | Nullable | Notes              |
|-------|-----------------|-----------|---------|----------|--------------------|
| Value | json.RawMessage | metadata  | json    | YES      | Raw JSON metadata  |

---

#### 44. DisabledPaymentChannelProject

Per-project disabled payment channels.

- **Base model:** PayramModel
- **Table name:** `disabled_payment_channel_projects`

| Field              | Go Type           | DB Column            | DB Type | Nullable | Notes                              |
|--------------------|-------------------|----------------------|---------|----------|------------------------------------|
| PayramModel        | PayramModel       | (embedded)           | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| ExternalPlatformID | uint              | external_platform_id | bigint  | NO       | FK -> external_platforms.id        |
| PaymentChannelID   | uint              | payment_channel_id   | bigint  | NO       | FK -> payment_channels.id          |
| Reason             | *string           | reason               | text    | YES      | Disable reason                     |
| DisabledByMemberID | *uint             | disabled_by_member_id| bigint  | YES      | FK -> members.id                   |
| ExternalPlatform   | *ExternalPlatform | —                    | —       | —        | BelongsTo: ExternalPlatform        |
| PaymentChannel     | *PaymentChannel   | —                    | —       | —        | BelongsTo: PaymentChannel          |
| DisabledByMember   | *Member           | —                    | —       | —        | BelongsTo: Member                  |

**Unique:** (external_platform_id, payment_channel_id)
**Foreign keys:**
- `external_platform_id` -> `external_platforms(id)`
- `payment_channel_id` -> `payment_channels(id)`
- `disabled_by_member_id` -> `members(id)`

---

#### 45. PaymentsApp

Per-project payment app settings (sponsorship configuration).

- **Base model:** PayramModel
- **Table name:** `payments_apps`

| Field                   | Go Type           | DB Column               | DB Type        | Nullable | Notes                              |
|-------------------------|-------------------|-------------------------|----------------|----------|------------------------------------|
| PayramModel             | PayramModel       | (embedded)              | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| ProjectID               | uint              | project_id              | bigint         | NO       | FK -> external_platforms.id        |
| SponsorshipPercentage   | decimal.Decimal   | sponsorship_percentage  | numeric(5,4)   | NO       | Sponsorship percentage             |
| SponsorshipCutOff       | decimal.Decimal   | sponsorship_cut_off     | numeric(38,18) | NO       | Sponsorship cap amount             |
| Project                 | *ExternalPlatform | —                       | —              | —        | BelongsTo: ExternalPlatform        |

**Unique:** `project_id`
**Foreign keys:**
- `project_id` -> `external_platforms(id)`

---

### H. Webhooks

#### 46. Webhook

Webhook endpoint configuration per project.

- **Base model:** PayramModel
- **Table name:** `webhooks`

| Field              | Go Type           | DB Column            | DB Type       | Nullable | Notes                              |
|--------------------|-------------------|----------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel       | (embedded)           | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| ExternalPlatformID | uint              | external_platform_id | bigint        | NO       | FK -> external_platforms.id        |
| Name               | string            | name                 | varchar(200)  | NO       | Webhook name                       |
| Url                | string            | url                  | varchar(500)  | NO       | Webhook endpoint URL               |
| AccessKey          | string            | access_key           | varchar(255)  | NO       | HMAC signing key                   |
| Status             | string            | status               | varchar(20)   | NO       | active/disabled (default 'active') |
| Tested             | bool              | tested               | boolean       | NO       | Test delivery completed            |
| ExternalPlatform   | *ExternalPlatform | —                    | —             | —        | BelongsTo: ExternalPlatform        |

**Foreign keys:**
- `external_platform_id` -> `external_platforms(id)`

---

#### 47. WebhookDeliveryLog

Webhook delivery attempt tracking with retry logic.

- **Base model:** PayramModel
- **Table name:** `webhook_delivery_logs`

| Field            | Go Type         | DB Column          | DB Type       | Nullable | Notes                              |
|------------------|-----------------|--------------------|---------------|----------|------------------------------------|
| PayramModel      | PayramModel     | (embedded)         | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| PaymentRequestID | uint            | payment_request_id | bigint        | NO       | FK -> payment_requests.id          |
| WebhookID        | uint            | webhook_id         | bigint        | NO       | FK -> webhooks.id                  |
| Status           | string          | status             | varchar(30)   | NO       | pending/delivered/failed           |
| AttemptCount     | int             | attempt_count      | bigint        | NO       | Delivery attempts (default 0)      |
| NextRetryAt      | *time.Time      | next_retry_at      | timestamptz   | YES      | Next retry timestamp               |
| LastAttemptAt    | *time.Time      | last_attempt_at    | timestamptz   | YES      | Last attempt timestamp             |
| LastResponseCode | *int            | last_response_code | bigint        | YES      | HTTP status code                   |
| LastError        | *string         | last_error         | text          | YES      | Error message                      |
| PaymentRequest   | *PaymentRequest | —                  | —             | —        | BelongsTo: PaymentRequest          |
| Webhook          | *Webhook        | —                  | —             | —        | BelongsTo: Webhook                 |

**Foreign keys:**
- `payment_request_id` -> `payment_requests(id)`
- `webhook_id` -> `webhooks(id)`

---

### I. Accounting / Double-Entry Ledger

#### 48. Account

Member balance per currency (main balance ledger).

- **Base model:** PayramModel
- **Table name:** `accounts`

| Field        | Go Type         | DB Column     | DB Type        | Nullable | Notes                              |
|--------------|-----------------|---------------|----------------|----------|------------------------------------|
| PayramModel  | PayramModel     | (embedded)    | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID     | uint            | member_id     | bigint         | YES      | FK -> members.id                   |
| CurrencyID   | uint            | currency_id   | bigint         | YES      | FK -> currencies.id                |
| CurrencyCode | string          | currency_code | varchar(20)    | NO       | Currency code                      |
| Balance      | decimal.Decimal | balance       | numeric(38,18) | NO       | Available balance (default 0.0)    |
| Locked       | decimal.Decimal | locked        | numeric(38,18) | NO       | Locked balance (default 0.0)       |
| Status       | string          | status        | varchar(20)    | NO       | active/frozen (default 'active')   |
| Member       | Member          | —             | —              | —        | BelongsTo: Member                  |
| Currency     | Currency        | —             | —              | —        | BelongsTo: Currency                |

**Foreign keys:**
- `member_id` -> `members(id)`
- `currency_id` -> `currencies(id)`

---

#### 49. AccountReward

Reward balances per member per currency.

- **Base model:** PayramModel
- **Table name:** `account_rewards`

| Field        | Go Type         | DB Column     | DB Type        | Nullable | Notes                              |
|--------------|-----------------|---------------|----------------|----------|------------------------------------|
| PayramModel  | PayramModel     | (embedded)    | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID     | uint            | member_id     | bigint         | NO       | FK -> members.id                   |
| CurrencyID   | uint            | currency_id   | bigint         | NO       | FK -> currencies.id                |
| CurrencyCode | string          | currency_code | varchar(20)    | NO       | Currency code                      |
| Balance      | decimal.Decimal | balance       | numeric(38,18) | NO       | Available balance (default 0.0)    |
| Locked       | decimal.Decimal | locked        | numeric(38,18) | NO       | Locked balance (default 0.0)       |
| Status       | string          | status        | varchar(20)    | NO       | active/frozen (default 'active')   |
| Member       | *Member         | —             | —              | —        | BelongsTo: Member                  |
| Currency     | *Currency       | —             | —              | —        | BelongsTo: Currency                |

**Primary key:** (id, member_id, currency_id)
**Foreign keys:**
- `member_id` -> `members(id)`
- `currency_id` -> `currencies(id)`

---

#### 50. Asset

Asset ledger entries (double-entry debit/credit).

- **Base model:** PayramModel
- **Table name:** `assets`

| Field         | Go Type          | DB Column      | DB Type        | Nullable | Notes                              |
|---------------|------------------|----------------|----------------|----------|------------------------------------|
| PayramModel   | PayramModel      | (embedded)     | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID      | *uint            | member_id      | bigint         | YES      | FK -> members.id                   |
| AccountID     | *uint            | account_id     | bigint         | YES      | FK -> accounts.id                  |
| CurrencyID    | uint             | currency_id    | bigint         | NO       | FK -> currencies.id                |
| Debit         | decimal.Decimal  | debit          | numeric(38,18) | NO       | Debit amount (default 0.0)         |
| Credit        | decimal.Decimal  | credit         | numeric(38,18) | NO       | Credit amount (default 0.0)        |
| CurrencyCode  | string           | currency_code  | varchar(20)    | NO       | Currency code                      |
| ReferenceType | *string          | reference_type | text           | YES      | Source entity type                 |
| ReferenceID   | *int64           | reference_id   | bigint         | YES      | Source entity ID                   |
| Code          | uint             | code           | bigint         | NO       | Accounting code                    |
| Member        | *Member          | —              | —              | —        | BelongsTo: Member                  |
| Account       | *Account         | —              | —              | —        | BelongsTo: Account                 |
| Currency      | Currency         | —              | —              | —        | BelongsTo: Currency                |

**Foreign keys:**
- `member_id` -> `members(id)`
- `account_id` -> `accounts(id)`
- `currency_id` -> `currencies(id)`

---

#### 51. Liability

Liability ledger entries (double-entry debit/credit).

- **Base model:** PayramModel
- **Table name:** `liabilities`

| Field         | Go Type          | DB Column      | DB Type        | Nullable | Notes                              |
|---------------|------------------|----------------|----------------|----------|------------------------------------|
| PayramModel   | PayramModel      | (embedded)     | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID      | *uint            | member_id      | bigint         | YES      | FK -> members.id                   |
| AccountID     | *uint            | account_id     | bigint         | YES      | FK -> accounts.id                  |
| CurrencyID    | uint             | currency_id    | bigint         | NO       | FK -> currencies.id                |
| Debit         | decimal.Decimal  | debit          | numeric(38,18) | NO       | Debit amount (default 0.0)         |
| Credit        | decimal.Decimal  | credit         | numeric(38,18) | NO       | Credit amount (default 0.0)        |
| CurrencyCode  | string           | currency_code  | varchar(20)    | NO       | Currency code                      |
| ReferenceType | *string          | reference_type | text           | YES      | Source entity type                 |
| ReferenceID   | *int64           | reference_id   | bigint         | YES      | Source entity ID                   |
| Code          | uint             | code           | bigint         | NO       | Accounting code                    |
| Member        | *Member          | —              | —              | —        | BelongsTo: Member                  |
| Account       | *Account         | —              | —              | —        | BelongsTo: Account                 |
| Currency      | Currency         | —              | —              | —        | BelongsTo: Currency                |

**Foreign keys:**
- `member_id` -> `members(id)`
- `account_id` -> `accounts(id)`
- `currency_id` -> `currencies(id)`

---

#### 52. Revenue

Revenue ledger entries (double-entry debit/credit).

- **Base model:** PayramModel
- **Table name:** `revenues`

| Field         | Go Type          | DB Column      | DB Type        | Nullable | Notes                              |
|---------------|------------------|----------------|----------------|----------|------------------------------------|
| PayramModel   | PayramModel      | (embedded)     | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID      | *uint            | member_id      | bigint         | YES      | FK -> members.id                   |
| AccountID     | *uint            | account_id     | bigint         | YES      | FK -> accounts.id                  |
| CurrencyID    | uint             | currency_id    | bigint         | NO       | FK -> currencies.id                |
| Debit         | decimal.Decimal  | debit          | numeric(38,18) | NO       | Debit amount (default 0.0)         |
| Credit        | decimal.Decimal  | credit         | numeric(38,18) | NO       | Credit amount (default 0.0)        |
| CurrencyCode  | string           | currency_code  | varchar(20)    | NO       | Currency code                      |
| ReferenceType | *string          | reference_type | text           | YES      | Source entity type                 |
| ReferenceID   | *int64           | reference_id   | bigint         | YES      | Source entity ID                   |
| Code          | uint             | code           | bigint         | NO       | Accounting code                    |
| Member        | *Member          | —              | —              | —        | BelongsTo: Member                  |
| Account       | *Account         | —              | —              | —        | BelongsTo: Account                 |
| Currency      | Currency         | —              | —              | —        | BelongsTo: Currency                |

**Foreign keys:**
- `member_id` -> `members(id)`
- `account_id` -> `accounts(id)`
- `currency_id` -> `currencies(id)`

---

#### 53. Expense

Expense ledger entries (double-entry debit/credit).

- **Base model:** PayramModel
- **Table name:** `expenses`

| Field         | Go Type          | DB Column      | DB Type        | Nullable | Notes                              |
|---------------|------------------|----------------|----------------|----------|------------------------------------|
| PayramModel   | PayramModel      | (embedded)     | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID      | *uint            | member_id      | bigint         | YES      | FK -> members.id                   |
| AccountID     | *uint            | account_id     | bigint         | YES      | FK -> accounts.id                  |
| CurrencyID    | uint             | currency_id    | bigint         | NO       | FK -> currencies.id                |
| Debit         | decimal.Decimal  | debit          | numeric(38,18) | NO       | Debit amount (default 0.0)         |
| Credit        | decimal.Decimal  | credit         | numeric(38,18) | NO       | Credit amount (default 0.0)        |
| CurrencyCode  | string           | currency_code  | varchar(20)    | NO       | Currency code                      |
| ReferenceType | *string          | reference_type | text           | YES      | Source entity type                 |
| ReferenceID   | *int64           | reference_id   | bigint         | YES      | Source entity ID                   |
| Code          | uint             | code           | bigint         | NO       | Accounting code                    |
| Member        | *Member          | —              | —              | —        | BelongsTo: Member                  |
| Account       | *Account         | —              | —              | —        | BelongsTo: Account                 |
| Currency      | Currency         | —              | —              | —        | BelongsTo: Currency                |

**Foreign keys:**
- `member_id` -> `members(id)`
- `account_id` -> `accounts(id)`
- `currency_id` -> `currencies(id)`

---

### J. Referral & Campaigns

#### 54. Campaign

Referral campaign definitions with budgets, caps, and time-limited rewards.

- **Base model:** BaseModel
- **Table name:** `referral_campaigns`

| Field                       | Go Type          | DB Column                     | DB Type        | Nullable | Notes                              |
|-----------------------------|------------------|-------------------------------|----------------|----------|------------------------------------|
| BaseModel                   | BaseModel        | (embedded)                    | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Project                     | string           | project                       | varchar        | NO       | Project identifier                 |
| Name                        | string           | name                          | varchar        | NO       | Campaign name                      |
| RewardType                  | *string          | reward_type                   | varchar        | YES      | percentage/fixed                   |
| RewardValue                 | *decimal.Decimal | reward_value                  | numeric(38,18) | YES      | Reward amount/percentage           |
| CurrencyCode                | string           | currency_code                 | varchar        | NO       | Reward currency                    |
| RewardCap                   | *decimal.Decimal | reward_cap                    | numeric(38,18) | YES      | Maximum reward per occurrence      |
| InviteeRewardType           | *string          | invitee_reward_type           | varchar        | YES      | Invitee reward type                |
| InviteeRewardValue          | *decimal.Decimal | invitee_reward_value          | numeric(38,18) | YES      | Invitee reward amount              |
| InviteeRewardCap            | *decimal.Decimal | invitee_reward_cap            | numeric(38,18) | YES      | Invitee reward cap                 |
| Budget                      | *decimal.Decimal | budget                        | numeric(38,18) | YES      | Campaign budget                    |
| Description                 | *string          | description                   | text           | YES      | Campaign description               |
| StartDate                   | *time.Time       | start_date                    | timestamptz    | YES      | Campaign start                     |
| EndDate                     | *time.Time       | end_date                      | timestamptz    | YES      | Campaign end                       |
| Status                      | string           | status                        | varchar        | NO       | active/paused/ended                |
| IsDefault                   | bool             | is_default                    | boolean        | NO       | Default campaign flag              |
| CampaignTypePerCustomer     | string           | campaign_type_per_customer    | varchar        | NO       | Per-customer type                  |
| MaxOccurrencesPerCustomer   | *int64           | max_occurrences_per_customer  | bigint         | YES      | Max occurrences per customer       |
| ValidityMonthsPerCustomer   | *int             | validity_months_per_customer  | bigint         | YES      | Validity months per customer       |
| RewardCapPerCustomer        | *decimal.Decimal | reward_cap_per_customer       | numeric(38,18) | YES      | Reward cap per customer            |
| ConsiderEventsFrom          | time.Time        | consider_events_from          | timestamptz    | NO       | Events start date                  |
| Events                      | []Event          | —                             | —              | —        | ManyToMany via referral_campaign_events |

**Relationships:**
- ManyToMany: Event (via CampaignEvent join table)

---

#### 55. Event

Trackable referral events.

- **Base model:** BaseModel
- **Table name:** `referral_events`

| Field       | Go Type   | DB Column   | DB Type | Nullable | Notes                              |
|-------------|-----------|-------------|---------|----------|------------------------------------|
| BaseModel   | BaseModel | (embedded)  | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Project     | string    | project     | varchar | NO       | Project identifier                 |
| Key         | string    | key         | varchar | NO       | Event key (unique identifier)      |
| Name        | string    | name        | varchar | NO       | Event display name                 |
| EventType   | string    | event_type  | varchar | NO       | Event type                         |
| Description | *string   | description | text    | YES      | Event description                  |

---

#### 56. CampaignEvent

Join table linking campaigns to events.

- **Base model:** None (composite key)
- **Table name:** `referral_campaign_events`

| Field      | Go Type  | DB Column   | DB Type | Nullable | Notes                              |
|------------|----------|-------------|---------|----------|------------------------------------|
| Project    | string   | project     | varchar | NO       | Project identifier                 |
| CampaignID | uint    | campaign_id | bigint  | NO       | FK -> referral_campaigns.id        |
| EventID    | uint    | event_id    | bigint  | NO       | FK -> referral_events.id           |
| EventKey   | string  | event_key   | varchar | NO       | Event key                          |
| Campaign   | Campaign | —           | —       | —        | BelongsTo: Campaign                |
| Event      | Event    | —           | —       | —        | BelongsTo: Event                   |

**Foreign keys:**
- `campaign_id` -> `referral_campaigns(id)`
- `event_id` -> `referral_events(id)`

---

#### 57. EventLog

Event trigger log entries.

- **Base model:** BaseModel
- **Table name:** `referral_event_logs`

| Field               | Go Type          | DB Column            | DB Type        | Nullable | Notes                              |
|---------------------|------------------|----------------------|----------------|----------|------------------------------------|
| BaseModel           | BaseModel        | (embedded)           | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Project             | string           | project              | varchar        | NO       | Project identifier                 |
| EventKey            | string           | event_key            | varchar        | NO       | Event key                          |
| MemberID            | uint             | member_id            | bigint         | NO       | FK -> members.id                   |
| MemberReferenceID   | string           | member_reference_id  | varchar        | NO       | External member reference          |
| Amount              | *decimal.Decimal | amount               | numeric(38,18) | YES      | Event amount                       |
| TriggeredAt         | time.Time        | triggered_at         | timestamptz    | NO       | Event trigger timestamp            |
| Data                | *string          | data                 | text           | YES      | Event data                         |
| Status              | string           | status               | varchar        | NO       | processed/failed                   |
| FailureReason       | *string          | failure_reason       | text           | YES      | Error description                  |
| Member              | *Member          | —                    | —              | —        | BelongsTo: Member                  |

**Foreign keys:**
- `member_id` -> `members(id)`

---

#### 58. Reward

Reward ledger for referral program.

- **Base model:** BaseModel
- **Table name:** `referral_rewards`

| Field                       | Go Type         | DB Column                      | DB Type        | Nullable | Notes                              |
|-----------------------------|-----------------|--------------------------------|----------------|----------|------------------------------------|
| BaseModel                   | BaseModel       | (embedded)                     | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Project                     | string          | project                        | varchar        | NO       | Project identifier                 |
| CampaignID                  | uint            | campaign_id                    | bigint         | NO       | FK -> referral_campaigns.id        |
| CurrencyCode                | string          | currency_code                  | varchar        | NO       | Reward currency                    |
| RewardedMemberID            | uint            | rewarded_member_id             | bigint         | NO       | FK -> members.id                   |
| RewardedMemberReferenceID   | string          | rewarded_member_reference_id   | varchar        | NO       | External reference for rewarded    |
| RelatedMemberID             | uint            | related_member_id              | bigint         | NO       | FK -> members.id                   |
| RelatedMemberReferenceID    | string          | related_member_reference_id    | varchar        | NO       | External reference for related     |
| MemberType                  | string          | member_type                    | varchar        | NO       | referrer/referee                   |
| Amount                      | decimal.Decimal | amount                         | numeric(38,18) | NO       | Reward amount                      |
| Status                      | string          | status                         | varchar        | NO       | pending/processed/failed           |
| Reason                      | *string         | reason                         | text           | YES      | Reward reason                      |
| RewardedMember              | *Member         | —                              | —              | —        | BelongsTo: Member (rewarded)       |
| RelatedMember               | *Member         | —                              | —              | —        | BelongsTo: Member (related)        |

**Foreign keys:**
- `campaign_id` -> `referral_campaigns(id)`
- `rewarded_member_id` -> `members(id)`
- `related_member_id` -> `members(id)`

---

#### 59. CampaignEventLog

Per-campaign event processing log with reward tracking.

- **Base model:** BaseModel
- **Table name:** `referral_campaign_event_logs`

| Field               | Go Type   | DB Column            | DB Type | Nullable | Notes                              |
|---------------------|-----------|----------------------|---------|----------|------------------------------------|
| BaseModel           | BaseModel | (embedded)           | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Project             | string    | project              | varchar | NO       | Project identifier                 |
| CampaignID          | uint      | campaign_id          | bigint  | NO       | FK -> referral_campaigns.id        |
| EventID             | uint      | event_id             | bigint  | NO       | FK -> referral_events.id           |
| MemberID            | uint      | member_id            | bigint  | NO       | FK -> members.id                   |
| MemberReferenceID   | string    | member_reference_id  | varchar | NO       | External member reference          |
| Status              | string    | status               | varchar | NO       | processed/failed                   |
| EventLogID          | uint      | event_log_id         | bigint  | NO       | FK -> referral_event_logs.id       |
| ReferredRewardID    | *uint     | referred_reward_id   | bigint  | YES      | FK -> referral_rewards.id          |
| RefereeRewardID     | *uint     | referee_reward_id    | bigint  | YES      | FK -> referral_rewards.id          |
| Campaign            | *Campaign | —                    | —       | —        | BelongsTo: Campaign                |
| Event               | *Event    | —                    | —       | —        | BelongsTo: Event                   |
| Member              | *Member   | —                    | —       | —        | BelongsTo: Member                  |
| ReferredReward      | *Reward   | —                    | —       | —        | BelongsTo: Reward                  |
| RefereeReward       | *Reward   | —                    | —       | —        | BelongsTo: Reward                  |

**Foreign keys:**
- `campaign_id` -> `referral_campaigns(id)`
- `event_id` -> `referral_events(id)`
- `member_id` -> `members(id)`
- `event_log_id` -> `referral_event_logs(id)`
- `referred_reward_id` -> `referral_rewards(id)`
- `referee_reward_id` -> `referral_rewards(id)`

---

#### 60. ProcessedReward

Tracks reward distribution processing.

- **Base model:** PayramModel
- **Table name:** `processed_rewards`

| Field         | Go Type     | DB Column      | DB Type | Nullable | Notes                              |
|---------------|-------------|----------------|---------|----------|------------------------------------|
| PayramModel   | PayramModel | (embedded)     | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| RewardID      | uint        | reward_id      | bigint  | NO       | FK -> referral_rewards.id          |
| MemberID      | uint        | member_id      | bigint  | NO       | FK -> members.id                   |
| ReferenceID   | string      | reference_id   | varchar | NO       | External reference                 |
| Status        | string      | status         | varchar | NO       | completed/failed                   |
| FailureReason | *string     | failure_reason | text    | YES      | Error description                  |

**Foreign keys:**
- `reward_id` -> `referral_rewards(id)`
- `member_id` -> `members(id)`

---

#### 61. MemberCampaign

Join table linking members to campaigns.

- **Base model:** None (composite key)
- **Table name:** `referral_member_campaigns`

| Field      | Go Type  | DB Column   | DB Type | Nullable | Notes                              |
|------------|----------|-------------|---------|----------|------------------------------------|
| Project    | string   | project     | varchar | NO       | Project identifier                 |
| MemberID   | uint    | member_id   | bigint  | NO       | FK -> members.id                   |
| CampaignID | uint    | campaign_id | bigint  | NO       | FK -> referral_campaigns.id        |
| Campaign   | Campaign | —           | —       | —        | BelongsTo: Campaign                |
| Member     | Member   | —           | —       | —        | BelongsTo: Member                  |

**Foreign keys:**
- `member_id` -> `members(id)`
- `campaign_id` -> `referral_campaigns(id)`

---

### K. Analytics

#### 62. AnalyticsGroup

Dashboard analytics group definitions.

- **Base model:** PayramModel
- **Table name:** `analytics_groups`

| Field            | Go Type           | DB Column   | DB Type      | Nullable | Notes                              |
|------------------|-------------------|-------------|--------------|----------|------------------------------------|
| PayramModel      | PayramModel       | (embedded)  | —            | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name             | string            | name        | varchar(100) | NO       | Group name                         |
| Description      | string            | description | text         | YES      | Group description                  |
| IsDefault        | bool              | is_default  | boolean      | YES      | Default group flag                 |
| AnalyticsFilters | []AnalyticsFilter | —           | —            | —        | ManyToMany via analytics_group_filters |
| AnalyticsGraphs  | []AnalyticsGraph  | —           | —            | —        | HasMany: AnalyticsGraph.GroupID    |
| Category         | string            | category    | varchar(50)  | YES      | Group category                     |
| Sequence         | uint              | sequence    | bigint       | NO       | Display order (default 0)          |

**Relationships:**
- ManyToMany: AnalyticsFilter (via AnalyticsGroupFilter join table)
- HasMany: AnalyticsGraph

---

#### 63. AnalyticsFilter

Analytics filter definitions.

- **Base model:** PayramModel
- **Table name:** `analytics_filters`

| Field         | Go Type     | DB Column      | DB Type      | Nullable | Notes                              |
|---------------|-------------|----------------|--------------|----------|------------------------------------|
| PayramModel   | PayramModel | (embedded)     | —            | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name          | string      | name           | varchar(100) | NO       | Filter name                        |
| Type          | string      | type           | varchar(50)  | NO       | Filter type                        |
| Value         | string      | value          | text         | NO       | Default value                      |
| Options       | string      | options        | text         | YES      | Available options JSON             |
| ValueMappings | string      | value_mappings | text         | YES      | Value mapping JSON                 |
| IconMappings  | string      | icon_mappings  | text         | YES      | Icon mapping JSON                  |

---

#### 64. AnalyticsGraph

Individual graph/chart definitions within analytics groups.

- **Base model:** PayramModel
- **Table name:** `analytics_graphs`

| Field          | Go Type        | DB Column       | DB Type      | Nullable | Notes                              |
|----------------|----------------|-----------------|--------------|----------|------------------------------------|
| PayramModel    | PayramModel    | (embedded)      | —            | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| GroupID        | uint           | group_id        | bigint       | NO       | FK -> analytics_groups.id          |
| Name           | string         | name            | varchar(100) | NO       | Graph name                         |
| Description    | string         | description     | text         | YES      | Graph description                  |
| GraphType      | string         | graph_type      | varchar(50)  | NO       | Chart type (bar, line, pie, etc.)  |
| QueryTemplate  | string         | query_template  | text         | NO       | SQL query template                 |
| FilterMappings | string         | filter_mappings | text         | YES      | Filter mapping JSON                |
| ColorMappings  | string         | color_mappings  | text         | YES      | Color mapping JSON                 |
| Attributes     | string         | attributes      | text         | YES      | Additional attributes JSON         |
| Active         | bool           | active          | boolean      | YES      | Active flag (default true)         |
| Sequence       | uint           | sequence        | bigint       | NO       | Display order (default 0)          |
| AnalyticsGroup | AnalyticsGroup | —               | —            | —        | BelongsTo: AnalyticsGroup          |

**Foreign keys:**
- `group_id` -> `analytics_groups(id)`

---

#### 65. AnalyticsUserGroup

Maps members to analytics groups with visibility toggle.

- **Base model:** PayramModel
- **Table name:** `analytics_user_groups`

| Field                  | Go Type                 | DB Column  | DB Type | Nullable | Notes                              |
|------------------------|-------------------------|------------|---------|----------|------------------------------------|
| PayramModel            | PayramModel             | (embedded) | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID               | uint                    | member_id  | bigint  | NO       | FK -> members.id                   |
| GroupID                | uint                    | group_id   | bigint  | NO       | FK -> analytics_groups.id          |
| Visible                | bool                    | visible    | boolean | NO       | Visibility toggle                  |
| AnalyticsCustomFilters | []AnalyticsCustomFilter | —          | —       | —        | HasMany                            |
| Member                 | Member                  | —          | —       | —        | BelongsTo: Member                  |
| AnalyticsGroup         | AnalyticsGroup          | —          | —       | —        | BelongsTo: AnalyticsGroup          |

**Foreign keys:**
- `member_id` -> `members(id)`
- `group_id` -> `analytics_groups(id)`

**Relationships:**
- BelongsTo: Member, AnalyticsGroup
- HasMany: AnalyticsCustomFilter

---

#### 66. AnalyticsCustomFilter

Per-user custom filter values.

- **Base model:** PayramModel
- **Table name:** `analytics_custom_filters`

| Field                   | Go Type     | DB Column                | DB Type | Nullable | Notes                              |
|-------------------------|-------------|--------------------------|---------|----------|------------------------------------|
| PayramModel             | PayramModel | (embedded)               | —       | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| ExternalPlatformIDStr   | string      | external_platform_id_str | varchar | NO       | Platform ID as string              |
| UserGroupID             | uint        | user_group_id            | bigint  | NO       | FK -> analytics_user_groups.id     |
| FilterID                | uint        | filter_id                | bigint  | NO       | FK -> analytics_filters.id         |
| Value                   | string      | value                    | text    | NO       | Custom filter value                |

**Foreign keys:**
- `user_group_id` -> `analytics_user_groups(id)`
- `filter_id` -> `analytics_filters(id)`

---

#### 67. AnalyticsGroupFilter

Join table linking analytics groups to filters.

- **Base model:** None (composite primary key)
- **Table name:** `analytics_group_filters`

| Field              | Go Type | DB Column            | DB Type | Nullable | Notes                              |
|--------------------|---------|----------------------|---------|----------|------------------------------------|
| AnalyticsGroupID   | uint    | analytics_group_id   | bigint  | NO       | FK -> analytics_groups.id          |
| AnalyticsFilterID  | uint    | analytics_filter_id  | bigint  | NO       | FK -> analytics_filters.id         |

**Primary key:** (analytics_group_id, analytics_filter_id)

---

### L. External Platforms

#### 68. ExternalPlatform

Multi-tenant project/merchant entities. Central multi-tenancy anchor.

- **Base model:** PayramModel
- **Table name:** `external_platforms`

| Field                                    | Go Type                                    | DB Column                                  | DB Type       | Nullable | Notes                              |
|------------------------------------------|--------------------------------------------|--------------------------------------------|---------------|----------|------------------------------------|
| PayramModel                              | PayramModel                                | (embedded)                                 | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name                                     | string                                     | name                                       | varchar(200)  | NO       | Project name                       |
| LogoPath                                 | string                                     | logo_path                                  | varchar(200)  | YES      | Logo file path                     |
| BrandColor                               | string                                     | brand_color                                | varchar(20)   | YES      | Brand color hex (default '#000000') |
| Website                                  | string                                     | website                                    | varchar(255)  | YES      | Project website URL                |
| SuccessEndpoint                          | string                                     | success_endpoint                           | varchar(500)  | YES      | Payment success redirect URL       |
| FrontendEndpoint                         | *string                                    | frontend_endpoint                          | varchar(500)  | YES      | Frontend base URL                  |
| CancelEndpoint                           | *string                                    | cancel_endpoint                            | varchar(500)  | YES      | Payment cancel redirect URL        |
| APIKeys                                  | []APIKey                                   | —                                          | —             | —        | HasMany: APIKey                    |
| PlatformRoles                            | []MemberExternalPlatformRole               | —                                          | —             | —        | HasMany                            |
| SupportEmail                             | *string                                    | support_email                              | varchar(255)  | YES      | Support email                      |
| LinkedInURL                              | *string                                    | linked_in_url                              | varchar(255)  | YES      | LinkedIn URL                       |
| TwitterURL                               | *string                                    | twitter_url                                | varchar(255)  | YES      | Twitter/X URL                      |
| DiscordURL                               | *string                                    | discord_url                                | varchar(255)  | YES      | Discord URL                        |
| TelegramURL                              | *string                                    | telegram_url                               | varchar(255)  | YES      | Telegram URL                       |
| DefaultPaymentBlockchainID              | *uint                                      | default_payment_blockchain_id              | bigint        | YES      | FK -> blockchains.id               |
| DefaultPaymentBlockchainCurrencyID      | *uint                                      | default_payment_blockchain_currency_id     | bigint        | YES      | FK -> blockchain_currencies.id     |
| EmailSendRequestFrom                     | *string                                    | email_send_request_from                    | varchar(255)  | YES      | Email sender address               |
| EmailSendRequestReplyTo                  | *string                                    | email_send_request_reply_to                | varchar(255)  | YES      | Email reply-to address             |
| DefaultPaymentBlockchainCurrency        | *BlockchainCurrency                        | —                                          | —             | —        | BelongsTo: BlockchainCurrency      |
| DefaultPaymentBlockchain                | *Blockchain                                | —                                          | —             | —        | BelongsTo: Blockchain              |
| ExternalPlatformBlockchainCurrencies    | []ExternalPlatformBlockchainCurrency        | —                                          | —             | —        | HasMany                            |

**Foreign keys:**
- `default_payment_blockchain_id` -> `blockchains(id)`
- `default_payment_blockchain_currency_id` -> `blockchain_currencies(id)`

**Relationships:**
- BelongsTo: Blockchain, BlockchainCurrency
- HasMany: APIKey, MemberExternalPlatformRole, ExternalPlatformBlockchainCurrency

---

#### 69. ExternalPlatformBlockchainCurrency

Per-project blockchain currency enablement (join table).

- **Base model:** None (timestamps, composite key)
- **Table name:** `external_platform_blockchain_currencies`

| Field                | Go Type            | DB Column              | DB Type     | Nullable | Notes                              |
|----------------------|--------------------|------------------------|-------------|----------|------------------------------------|
| CreatedAt            | time.Time          | created_at             | timestamptz | YES      | Timestamp                          |
| UpdatedAt            | time.Time          | updated_at             | timestamptz | YES      | Timestamp                          |
| ExternalPlatformID   | uint               | external_platform_id   | bigint      | NO       | FK -> external_platforms.id        |
| BlockchainCode       | string             | blockchain_code        | varchar(20) | YES      | Chain code                         |
| CurrencyCode         | string             | currency_code          | varchar(20) | YES      | Currency code                      |
| BlockchainFamily     | string             | blockchain_family      | varchar(20) | YES      | Family code                        |
| BlockchainCurrencyID | uint               | blockchain_currency_id | bigint      | NO       | FK -> blockchain_currencies.id     |
| ExternalPlatform     | *ExternalPlatform  | —                      | —           | —        | BelongsTo: ExternalPlatform        |
| BlockchainCurrency   | *BlockchainCurrency| —                      | —           | —        | BelongsTo: BlockchainCurrency      |

**Unique:** (external_platform_id, blockchain_currency_id)
**Foreign keys:**
- `external_platform_id` -> `external_platforms(id)`
- `blockchain_currency_id` -> `blockchain_currencies(id)`

---

### M. System & Utilities

#### 70. Configuration

System-wide key-value configuration store.

- **Base model:** PayramModel
- **Table name:** `configurations`

| Field               | Go Type     | DB Column             | DB Type       | Nullable | Notes                              |
|---------------------|-------------|-----------------------|---------------|----------|------------------------------------|
| PayramModel         | PayramModel | (embedded)            | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| ConfigKey           | string      | config_key            | varchar(200)  | NO       | Configuration key                  |
| ConfigValue         | string      | config_value          | text          | NO       | Configuration value                |
| OverrideConfigValue | string      | override_config_value | text          | YES      | Override value                     |
| Description         | string      | description           | text          | YES      | Description                        |
| Encrypt             | bool        | encrypt               | boolean       | YES      | Encrypted flag (default false)     |

---

#### 71. GenericDataStore

Temporary/misc data storage with TTL.

- **Base model:** PayramModel
- **Table name:** `generic_data_stores`

| Field       | Go Type     | DB Column  | DB Type       | Nullable | Notes                              |
|-------------|-------------|------------|---------------|----------|------------------------------------|
| PayramModel | PayramModel | (embedded) | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Type        | string      | type       | varchar(200)  | NO       | Data type identifier               |
| ValidTill   | time.Time   | valid_till | timestamptz   | YES      | Expiration time (TTL)              |
| Data        | string      | data       | text          | YES      | Data payload                       |
| Status      | string      | status     | varchar(20)   | YES      | active/expired (default 'active')  |

---

#### 72. Recipient

Payout recipient address book per merchant.

- **Base model:** PayramModel
- **Table name:** `recipients`

| Field              | Go Type     | DB Column            | DB Type       | Nullable | Notes                              |
|--------------------|-------------|----------------------|---------------|----------|------------------------------------|
| PayramModel        | PayramModel | (embedded)           | —             | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| Name               | *string     | name                 | varchar(100)  | YES      | Recipient name                     |
| Email              | *string     | email                | varchar(100)  | YES      | Recipient email                    |
| MobileNumber       | *string     | mobile_number        | varchar(100)  | YES      | Phone number                       |
| ResidentialAddress  | *string    | residential_address  | varchar(200)  | YES      | Physical address                   |
| BlockchainCode     | string      | blockchain_code      | varchar(200)  | NO       | Chain code                         |
| Address            | string      | address              | varchar(200)  | YES      | Blockchain address                 |
| MemberID           | uint        | member_id            | bigint        | NO       | FK -> members.id                   |
| Status             | string      | status               | varchar(70)   | NO       | active/disabled (default 'active') |
| OperatedByMemberID | uint        | operated_by_member_id| bigint        | NO       | FK -> members.id                   |
| LastOperation      | string      | last_operation       | varchar(200)  | NO       | Last operation type                |
| BlockchainID       | uint        | blockchain_id        | bigint        | NO       | FK -> blockchains.id               |
| Member             | *Member     | —                    | —             | —        | BelongsTo: Member (owner)          |
| Blockchain         | *Blockchain | —                    | —             | —        | BelongsTo: Blockchain              |
| OperatedByMember   | *Member     | —                    | —             | —        | BelongsTo: Member (operator)       |

**Unique:** (blockchain_code, address, member_id)
**Foreign keys:**
- `member_id` -> `members(id)`
- `operated_by_member_id` -> `members(id)`
- `blockchain_id` -> `blockchains(id)`

---

#### 73. ActivityLog

Full API request audit trail with geolocation.

- **Base model:** BaseModel
- **Table name:** `activity_logs`

| Field         | Go Type        | DB Column      | DB Type        | Nullable | Notes                              |
|---------------|----------------|----------------|----------------|----------|------------------------------------|
| BaseModel     | BaseModel      | (embedded)     | —              | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| MemberID      | *uint          | member_id      | bigint         | YES      | FK -> members.id                   |
| SessionID     | string         | session_id     | varchar(100)   | YES      | Session identifier                 |
| ProjectIDs    | UintSlice      | project_ids    | jsonb          | YES      | Affected project IDs               |
| Method        | string         | method         | varchar(10)    | NO       | HTTP method (GET, POST, etc.)      |
| APIPart       | string         | api_part       | varchar(255)   | NO       | API endpoint path                  |
| APIStatus     | string         | api_status     | varchar(50)    | NO       | success/error                      |
| StatusCode    | *int           | status_code    | bigint         | YES      | HTTP response status code          |
| Description   | *string        | description    | varchar(255)   | YES      | Action description                 |
| IPAddress     | *string        | ip_address     | varchar(50)    | YES      | Client IP address                  |
| UserAgent     | *string        | user_agent     | varchar(500)   | YES      | Browser user agent                 |
| Referer       | *string        | referer        | varchar(500)   | YES      | HTTP referer                       |
| APIAction     | string         | api_action     | varchar(100)   | NO       | Action type identifier             |
| APIErrorMsg   | *string        | api_error_msg  | varchar(1000)  | YES      | Error message                      |
| RequestBody   | *string        | request_body   | text           | YES      | Request body (redacted)            |
| ResponseBody  | *string        | response_body  | text           | YES      | Response body (redacted)           |
| Metadata      | *string        | metadata       | json           | YES      | Additional metadata                |
| Role          | *string        | role           | varchar(100)   | YES      | Acting role                        |
| EventCategory | *string        | event_category | varchar(100)   | YES      | Event category                     |
| EventName     | *string        | event_name     | varchar(100)   | YES      | Event name                         |
| Country       | *string        | country        | varchar(100)   | YES      | GeoIP country                      |
| CountryCode   | *string        | country_code   | varchar(10)    | YES      | ISO country code                   |
| Region        | *string        | region         | varchar(100)   | YES      | GeoIP region                       |
| City          | *string        | city           | varchar(100)   | YES      | GeoIP city                         |
| Timezone      | *string        | timezone       | varchar(100)   | YES      | GeoIP timezone                     |
| Latitude      | *float64       | latitude       | numeric(10,7)  | YES      | GeoIP latitude                     |
| Longitude     | *float64       | longitude      | numeric(10,7)  | YES      | GeoIP longitude                    |

---

#### 74. SeederLog

Tracks which database seeders have been applied.

- **Base model:** None (simple key-value)
- **Table name:** `seeder_logs`

| Field | Go Type | DB Column | DB Type     | Nullable | Notes                              |
|-------|---------|-----------|-------------|----------|------------------------------------|
| Name  | string  | name      | text        | NO       | Seeder name (primary key)          |
| Hash  | string  | hash      | varchar(64) | NO       | Seeder content hash                |

**Primary key:** `name`

---

#### 75. Tag

Polymorphic tagging system for arbitrary entities.

- **Base model:** PayramModel
- **Table name:** `tags`

| Field       | Go Type     | DB Column  | DB Type      | Nullable | Notes                              |
|-------------|-------------|------------|--------------|----------|------------------------------------|
| PayramModel | PayramModel | (embedded) | —            | —        | ID, CreatedAt, UpdatedAt, DeletedAt |
| TableName   | string      | table_name | varchar(100) | NO       | Target table name                  |
| TableID     | uint        | table_id   | bigint       | NO       | Target row ID                      |
| Name        | string      | name       | varchar(100) | NO       | Tag name                           |

---

## 4. Relationship Map

```
                                    +-----------------+
                                    | BlockchainFamily|
                                    |  (id, family)   |
                                    +--------+--------+
                                             |
                          +------------------+------------------+
                          |                  |                  |
                   +------+------+    +------+------+   +------+------+
                   | Blockchain  |    | AddressPool |   | WalletXpub  |
                   | (id, code)  |    | (id, address)|  | (id, xpub)  |
                   +------+------+    +------+------+   +------+------+
                          |                  |                  |
              +-----------+-----------+      |                  |
              |           |           |      |           +------+------+
       +------+------+ +--+---+ +----+----+ |           |   Wallet    |
       |BlockchainCur| |RPCNod| |Blkchain | |           | (id, name)  |
       |(id, address)| |(id)  | |Contract | |           +------+------+
       +------+------+ +------+ +----+----+ |                  |
              |                       |      |           +------+------+
              |                +------+------+           | WalletSCW   |
              |                |ContractAddr |           | WalletFunc  |
              |                +------+------+           | DepositAddr |
              |                       |                  +-------------+
              |                +------+------+
              |                |AddrContract |
              |                |  Signature  |
              |                +-------------+
              |
       +------+------+
       |  Currency    |
       |  (id, code)  |
       +------+------+
              |
    +---------+---------+---------+---------+
    |         |         |         |         |
+---+---+ +--+---+ +---+---+ +--+----+ +--+----+
|Account| |Accoun| |Asset  | |Liabil.| |Revenue|
|(id)   | |Reward| |(id)   | |(id)   | |(id)   |
+---+---+ +------+ +-------+ +-------+ +-------+
    |                                    +-------+
    |                                    |Expense|
    |                                    +-------+
    |
+---+---+
|Member |------+----------+----------+----------+
|(id)   |      |          |          |          |
+---+---+ +----+----+ +---+---+ +---+---+ +----+----+
    |     |  APIKey  | |MbrRole| |MbrExtP| |AuthRefr |
    |     |(id, key) | |(join) | |(id)   | |Token(id)|
    |     +----+----+ +---+---+ +---+---+ +---------+
    |          |           |        |
    |     +----+----+  +--+---+ +--+------+
    |     |  Role   |  |Member| |MbrExtPl |
    |     |(id,name)|  | Role | |  Role   |
    |     +----+----+  +------+ +--+------+
    |          |                    |
    |     +----+------+     +------+------+
    |     |RolePermis.|     |ExternalPlat.|-----+
    |     |  (join)   |     |(id, name)   |     |
    |     +----+------+     +------+------+     |
    |          |                   |             |
    |     +----+------+     +-----+-------+  +--+--------+
    |     |Permission |     |ExtPlatBlkCur|  |  Webhook   |
    |     |(id, name) |     |  (join)     |  | (id, url)  |
    |     +-----------+     +-------------+  +--+---------+
    |                                           |
    |                                     +-----+--------+
    |                                     |WebhookDelLog  |
    |                                     |(id)           |
    |                                     +---------+-----+
    |                                               |
+---+--------+                               +-----+--------+
|  Deposit   |---------+                     |PaymentRequest|
|(id, txhash)|         |                     |(id, invoice) |
+---+--------+    +----+----+                +--------------+
    |             |IntBlkTx |
    |             |(id)     |
    |             +---------+
    |
+---+--------+    +----------+     +--------------+
|   UTXO     |----|  Sweep   |     |SweepTransact.|
|(id, txhash)|    |(id)      |     |(id)          |
+---+--------+    +----------+     +------+-------+
    |                                      |
    +--------------------------------------+
    |
+---+-----------+
|AccountAddress |
|(id, address)  |
+---------------+

+---------------+     +-----------+     +-------------------+
|   Campaign    |-----|  Event    |     | CampaignEventLog  |
|(id, name)     |     |(id, key)  |     | (id)              |
+------+--------+     +-----+-----+     +-------------------+
       |                     |
+------+--------+     +-----+-----+     +-------------------+
| CampaignEvent |     | EventLog  |     |    Reward         |
| (join)        |     | (id)      |     | (id, amount)      |
+---------------+     +-----------+     +--------+----------+
                                                 |
                                        +--------+----------+
                                        | ProcessedReward   |
                                        | (id)              |
                                        +-------------------+

+---------------+     +-----------+     +-------------------+
|Withdrawal     |     | Withdraw  |     |WithdrawDepositsBTC|
|(id)           |     |(id, BTC)  |     |(id)               |
+---------------+     +-----------+     +-------------------+

+---------------+     +-----------+     +-------------------+
|AnalyticsGroup |-----|Anlyt.Graph|     |AnalyticsUserGroup |
|(id)           |     |(id)       |     |(id)               |
+------+--------+     +-----------+     +--------+----------+
       |                                         |
+------+--------+                       +--------+----------+
|AnlytGrpFilter |                       |AnlytCustomFilter  |
|(join)         |                       |(id)               |
+------+--------+                       +-------------------+
       |
+------+--------+
|AnalyticsFilter|
|(id)           |
+---------------+

Standalone:
+---------------+  +-----------+  +----------+  +-------------+
|Configuration  |  |GenericData|  |SeederLog |  |MissedDeposit|
|(id, key)      |  |Store (id) |  |(name)    |  |(id)         |
+---------------+  +-----------+  +----------+  +-------------+
+---------------+  +-----------+  +----------+  +-------------+
|Recipient      |  |ActivityLog|  |Tag       |  |Entrypoint   |
|(id)           |  |(id)       |  |(id)      |  |SCAddress(id)|
+---------------+  +-----------+  +----------+  +-------------+
+---------------+  +-----------+
|OTP            |  |WebSocket  |
|(id)           |  |Token (id) |
+---------------+  +-----------+
+---------------+  +-----------+
|AddressDeploym.|  |PaymentsApp|
|(id)           |  |(id)       |
+---------------+  +-----------+
```

---

## 5. Key Design Patterns

### 5.1 Soft Deletes via DeletedAt

Every entity with PayramModel or BaseModel has a `DeletedAt gorm.DeletedAt` field indexed in the database. GORM automatically adds `WHERE deleted_at IS NULL` to all queries. Hard deletes require `db.Unscoped().Delete()`.

### 5.2 decimal.Decimal for All Monetary Values

All financial amounts use `shopspring/decimal.Decimal` in Go, mapping to `numeric(38,18)` in PostgreSQL. This provides 18 decimal places of precision, sufficient for all cryptocurrency denominations (Wei = 10^-18 ETH). Fields include: `Amount`, `Balance`, `Locked`, `Fee`, `Debit`, `Credit`, `Price`, and related USD conversion fields.

### 5.3 Bigint Auto-Increment IDs (Not UUIDs)

All tables use `uint` (bigint) auto-increment primary keys via GORM's `gorm:"primarykey"` tag. This provides better index performance and simpler joins compared to UUIDs, at the cost of exposing sequence information.

### 5.4 GORM Preloading Strategies

Relationships use GORM association patterns:
- **BelongsTo:** Child has FK column (e.g., `Deposit.MemberID` -> `Member`)
- **HasMany:** Parent referenced by child FK (e.g., `Wallet.WalletFunctions`)
- **ManyToMany:** Join table (e.g., `Member <-> Role` via `member_roles`)
- Preloading uses `db.Preload("Member").Preload("Currency").Find(&deposits)`

### 5.5 Multi-Tenant via ExternalPlatform (Project) Scoping

`ExternalPlatform` is the multi-tenancy anchor. Key entities scoped to a project:
- `APIKey.ExternalPlatformID` -- API keys belong to projects
- `PaymentRequest.ExternalPlatformID` -- payments scoped to projects
- `Withdrawal.ExternalPlatformID` -- withdrawals scoped to projects
- `Webhook.ExternalPlatformID` -- webhooks per project
- `ExternalPlatformBlockchainCurrency` -- per-project currency enablement
- `ExternalPlatformWalletBlockchainFamily` -- per-project wallet assignment
- Members can belong to multiple projects via `MemberExternalPlatform` and `MemberExternalPlatformRole`

### 5.6 Double-Entry Accounting

Four ledger tables (`Asset`, `Liability`, `Revenue`, `Expense`) share identical structure with `Debit`/`Credit` columns. Each entry references:
- `AccountID` -- the member's balance account
- `ReferenceType` / `ReferenceID` -- polymorphic link to source transaction (deposit, withdrawal, sweep)
- `Code` -- accounting code for categorization

Every financial operation creates balanced entries across these tables. The `Account` model holds the net balance and locked amount per member per currency.

### 5.7 State Machines

Several models implement state machines via `Status` string fields. Valid transitions are enforced in the service layer. See Section 6 for complete state diagrams.

---

## 6. State Machine Diagrams

### PaymentRequest States

```
                    +--------+
                    |  OPEN  |
                    +---+----+
                        |
          +-------------+-------------+
          |             |             |
     (cancelled)   (deposit     (expired)
          |        received)        |
          v             |           v
    +-----------+       |     +---------+
    | CANCELLED |       |     | EXPIRED |
    +-----------+       |     +---------+
                        |
              +---------+---------+
              |         |         |
         (exact)   (partial)  (over)
              |         |         |
              v         v         v
         +--------+ +--------+ +--------+
         | FILLED | |PARTIAL | | OVER   |
         |        | |_FILLED | | _FILLED|
         +--------+ +--------+ +--------+
```

**States:** `open`, `cancelled`, `filled`, `partially_filled`, `over_filled`, `expired`

### Deposit States

```
    +---------+
    | pending |
    +----+----+
         |
    (block monitor detects confirmations)
         |
    +----+------+
    | confirming |
    +----+------+
         |
    (min confirmations reached)
         |
    +----+------+
    | confirmed |
    +----+------+
         |
    (balance credited)
         |
    +----+------+
    | completed |
    +-----------+

    Alternative paths:
    pending --> accepted (valid deposit)
    pending --> skipped  (below minimum / duplicate)
    pending --> rejected (invalid / suspicious)
```

**States:** `pending`, `confirming`, `confirmed`, `completed`, `accepted`, `skipped`, `rejected`

### Withdrawal States

```
    +-------------+
    | pending_otp |  (OTP sent to user)
    +------+------+
           |
    (OTP verified)
           |
    +------+----------+
    | pending_approval |  (awaits admin approval)
    +------+----------+
           |
     +-----+-----+
     |           |
  (approved)  (rejected)
     |           |
     v           v
  +-------+  +----------+
  |pending|  | rejected |
  +---+---+  +----------+
      |
  (TX built)
      |
  +---+------+
  | initiated|
  +---+------+
      |
  (TX broadcast)
      |
  +---+---+
  | sent  |
  +---+---+
      |
  +---+-----+-----+
  |               |
  v               v
+----------+  +--------+
| processed|  | failed |
+----------+  +--------+
```

**States:** `pending_otp`, `pending_approval`, `pending`, `initiated`, `sent`, `processed`, `failed`, `rejected`

### Sweep States

```
    +---------+
    | pending |
    +----+----+
         |
    (TX broadcast, waiting for confirmations)
         |
    +----+------+
    | confirming |
    +----+------+
         |
    +----+-----+-----+
    |                 |
    v                 v
+-----------+   +-----------+
| completed |   | not_found |
+-----------+   +-----------+
```

**States:** `pending`, `confirming`, `completed`, `not_found`

### OTP States

```
    +---------+
    | created |
    +----+----+
         |
    (user submits code)
         |
    +----+------+
    | attempted |
    +----+------+
         |
    +----+-----+-----+
    |                 |
(code correct)  (code wrong / time elapsed)
    |                 |
    v                 v
+----------+    +---------+
| verified |    | expired |
+----------+    +---------+
```

**States:** `created`, `attempted`, `verified`, `expired`

### AddressDeployment States

```
    +---------+
    | pending |
    +----+----+
         |
    (TX broadcast)
         |
    +----+--------+
    | broadcasted |
    +----+--------+
         |
    (TX confirmed)
         |
    +----+---+
    | active |
    +--------+
```

**States:** `pending`, `broadcasted`, `active`

### AddressPool States

```
    +------+
    | open |  (pre-generated, unassigned)
    +--+---+
       |
  (assigned to member deposit)
       |
    +--+------+
    | assigned |  (linked to AccountAddress)
    +--+------+
       |
  (SCW contract deployed -- if applicable)
       |
    +--+-------+
    | deployed |  (smart contract wallet active)
    +----------+
```

**States:** `open`, `assigned`, `deployed`

---

*Generated from PayRam Docker container analysis. Reference source: `payminto/docs/data/PAYRAM_GO_MODELS.txt` and `payminto/docs/DATABASE_SCHEMA.md`.*
