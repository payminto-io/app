# Payminto Service Layer Documentation

> Complete reference for the service layer architecture reverse-engineered from PayRam's Go backend.
> 982 functions across 45+ service implementations, organized by domain.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Service Dependency Graph](#2-service-dependency-graph)
3. [Service Registry](#3-service-registry)
4. [Authentication & Authorization Services](#4-authentication--authorization-services)
5. [Member & Role Services](#5-member--role-services)
6. [Blockchain Services](#6-blockchain-services)
7. [Wallet Services](#7-wallet-services)
8. [Address Services](#8-address-services)
9. [Deposit Services](#9-deposit-services)
10. [Sweep Services](#10-sweep-services)
11. [Payment Services](#11-payment-services)
12. [Withdrawal Services](#12-withdrawal-services)
13. [External Platform Services](#13-external-platform-services)
14. [Webhook Services](#14-webhook-services)
15. [Referral & Rewards Services](#15-referral--rewards-services)
16. [Analytics Services](#16-analytics-services)
17. [Infrastructure Services](#17-infrastructure-services)
18. [Background Worker Jobs](#18-background-worker-jobs)
19. [Cross-Cutting Patterns](#19-cross-cutting-patterns)
20. [Business Logic Concentrations](#20-business-logic-concentrations)

---

## 1. Overview

### Service Layer Architecture

The Payminto service layer follows a **clean architecture** pattern with interface-based dependency injection:

```
Handler (HTTP) --> Service Interface --> Service Implementation --> Repository Interface --> Repository Implementation --> PostgreSQL
```

**Key design principles:**

- **Interface segregation**: Every service exposes a Go interface. Handlers depend on interfaces, not implementations.
- **Constructor injection via ServiceRegistry**: All services are registered in a central `ServiceRegistry` that wires dependencies at startup.
- **Init pattern**: Each service implements `Init()` and `Name()` methods. `Init()` is called by the registry to inject the `*gorm.DB` handle and resolve cross-service dependencies.
- **Setter injection for circular dependencies**: Services that depend on each other use `Set*Service()` methods to break circular initialization (e.g., `BlockchainServiceImpl.SetPaymentService()`).
- **GORM transactions**: Complex multi-table operations use `db.Transaction(func(tx *gorm.DB) error { ... })` at the repository layer. The service layer orchestrates but delegates transactional boundaries downward.
- **Soft deletes**: All models use GORM's `DeletedAt` field for soft-delete support.

### File Organization

```
internal/service/
  |-- service_registry.go              # Central DI container
  |-- jwt_token_service.go             # JWT token generation/validation
  |-- auth_service_impl.go             # ~680 lines (largest service file)
  |-- blockchain_processor_impl.go     # ~2565 lines (block monitoring)
  |-- blockchain_service_impl.go       # ~1019 lines
  |-- wallet_service_impl.go           # Wallet CRUD + address generation
  |-- payment_service_impl.go          # Payment request lifecycle
  |-- withdrawal_service_impl.go       # Payout creation and policies
  |-- withdrawal_processing_service_impl.go  # Payout execution
  |-- sweep_transaction_service_impl.go      # Sweep orchestration
  |-- external_platform_service_impl.go      # Project/merchant management
  |-- member_service_impl.go           # Member CRUD + password management
  |-- event_emitter_service_impl.go    # Email/notification events
  |-- referral_service_impl.go         # Referral program management
  |-- analytics_referral_service_impl.go # Referral analytics
  |-- contract_address_service_impl.go # Smart contract deployment
  |-- address_service_impl.go          # Balance queries + sweep eligibility
  |-- account_processor_job.go         # Background sweep + accounting
  |-- ... (40+ additional service files)
```

---

## 2. Service Dependency Graph

```
                              ServiceRegistry
                                    |
           +------------------------+-------------------------+
           |                        |                         |
     AuthService            BlockchainService          PaymentService
        |    \                  /     |    \               /     |
        |  TokenService       /      |     \             /      |
        |       |            /       |      \           /       |
  MemberService |    BlockchainCurrency   DepositService    WebhookService
    |     \     |          |         |                        |
    |  MemberRole    WalletService   |                 WebhookDelivery
    |      |          /    |    \    |                   Service
    |  RoleService   /     |     \   |
    |      |        /      |      AddressPoolService
    | PermissionSvc/       |           |
    |            /    WalletFunction   AddressDeployment
    |           /      Service         Service
    |     ExternalPlatformService
    |        /          |        \
    |   EPBlockchain    |    EPWalletBlockchain
    |   CurrencySvc     |    FamilyService
    |                   |
    |            APIKeyService
    |
    +-- MemberExternalPlatformRoleService
    |
    +-- OTPService --> RecipientService (via OTPValidatable interface)
    |              --> WithdrawalProcessingService
    |
    +-- ConfigurationService --> BlockchainService, BlockchainCurrencyService
    |
    +-- EventEmitterService --> ExternalPlatformService, SMTPConfigService
    |
    +-- ConsumerService (email processor, uses event-consumer lib)
    |
    +-- ReferralService --> go-referral library
    |
    +-- AnalyticsReferralService --> ReferralService, PaymentService
    |
    +-- AnalyticsService --> AnalyticsRepository
    |
    +-- AccountRewardService, RewardAccountingService
    |
    +-- SweepService, SweepTransactionService, SweepUTXOService, UTXOService
    |
    +-- WithdrawalService, WithdrawalProcessingService
    |
    +-- GenericDataStoreService, NonceService, SystemService
    |
    +-- ActivityLogService (uses activity-log library)
    |
    +-- PublicAPIService, WebsocketTokenService
    |
    +-- PaymentChannelService, PaymentChannelProjectService
    |
    +-- OnramperPaymentsService, PaymentsAppService
    |
    +-- AddressContractSignatureService, ContractAddressService, EntrypointSCAddressService
    |
    +-- SMTPConfigService
```

---

## 3. Service Registry

### `ServiceRegistry`

**File**: `service_registry.go`

The `ServiceRegistry` is the central dependency injection container. It initializes all services, wires dependencies, and provides typed getters.

**Key Methods:**

| Method | Description |
|--------|-------------|
| `newServiceRegistry()` | Constructs all services, calls `Init()` on each, resolves circular deps via setters |
| `InitialiseBClients()` | Initializes blockchain clients (BTC, ETH, TRX, Base, Polygon) |
| `InitialiseBClient(blockchain)` | Initializes a single blockchain client |
| `InitRedisClient()` | Initializes Redis connection |
| `InitializeSentry()` | Initializes Sentry error tracking |
| `Get*Service()` | 50+ typed getter methods, one per service |

**Service Getter Methods (complete list):**

```go
GetAPIKeyService() APIKeyService
GetAccountAddressService() AccountAddressService
GetAccountRewardService() AccountRewardService
GetAccountService() AccountService
GetActivityLogService() ActivityLogService
GetAddressContractSignatureService() AddressContractSignatureService
GetAddressDeploymentService() AddressDeploymentService
GetAddressPoolService() AddressPoolService
GetAddressService() AddressService
GetAnalyticsReferralService() AnalyticsReferralService
GetAnalyticsService() AnalyticsService
GetAuthRefreshTokenService() AuthRefreshTokenService
GetAuthService() AuthService
GetBlockchainContractService() BlockchainContractService
GetBlockchainCurrencyService() BlockchainCurrencyService
GetBlockchainFamilyService() BlockchainFamilyService
GetBlockchainService() BlockchainService
GetConfigurationService() ConfigurationService
GetConsumerService() ConsumerService
GetContractAddressService() ContractAddressService
GetCurrencyService() CurrencyService
GetDepositAddressesService() DepositAddressesService
GetDepositService() DepositService
GetEntrypointSCAddressService() EntrypointSCAddressService
GetEventEmitterService() EventEmitterService
GetExternalPlatformBlockchainCurrencyService() ExternalPlatformBlockchainCurrencyService
GetExternalPlatformService() ExternalPlatformService
GetExternalPlatformWalletBlockchainFamilyService() ExternalPlatformWalletBlockchainFamilyService
GetGenericDataStoreService() GenericDataStoreService
GetInternalBlockchainTransactionService() InternalBlockchainTransactionService
GetMemberExternalPlatformRoleService() MemberExternalPlatformRoleService
GetMemberRoleService() MemberRoleService
GetMemberService() MemberService
GetMissedDepositService() MissedDepositService
GetNonceService() NonceService
GetOTPService() OTPService
GetOTPValidatable() OTPValidatable
GetOnramperPaymentsService() OnramperPaymentsService
GetPaymentChannelProjectService() PaymentChannelProjectService
GetPaymentChannelService() PaymentChannelService
GetPaymentService() PaymentService
GetPaymentsAppService() PaymentsAppService
GetPermissionService() PermissionService
GetPublicAPIService() PublicAPIService
GetRecipientService() RecipientService
GetReferralService() ReferralService
GetRewardAccountingService() RewardAccountingService
GetRoleService() RoleService
GetSMTPConfigService() SMTPConfigService
GetSecretsVaultService() SecretsVaultService
GetService(name) Service
GetSweepService() SweepService
GetSweepTransactionService() SweepTransactionService
GetSweepUTXOService() SweepUTXOService
GetSystemService() SystemService
GetUTXOService() UTXOService
GetWalletFunctionService() WalletFunctionService
GetWalletService() WalletService
GetWebhookDeliveryService() WebhookDeliveryService
GetWebhookService() WebhookService
GetWebsocketTokenService() WebsocketTokenService
GetWithdrawalProcessingService() WithdrawalProcessingService
GetWithdrawalService() WithdrawalService
```

---

## 4. Authentication & Authorization Services

### 4.1 AuthService

**Implementation**: `AuthServiceImpl`
**File**: `auth_service_impl.go` (~680 lines -- largest service file)
**Dependencies**: `AuthRepository`, `TokenService`, `AuthRefreshTokenService`, `MemberService`, `APIKeyService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Signup(ctx, request) (*models.Member, error)` | ~104 | Creates root member or internal member with password hashing, generates JWT pair, creates refresh token record |
| `Signin(ctx, request) (*TokenPair, error)` | ~203 | Validates email/password, checks member state, generates token pair, creates refresh token |
| `Authenticate(ctx, apiKey) (*models.Member, error)` | ~70 | Authenticates via API key, preloads roles and permissions, returns member with platform context |
| `RefreshAccessToken(ctx, refreshToken) (*TokenPair, error)` | ~146 | Validates refresh token hash, checks expiry, rotates tokens, updates last-used timestamp |
| `RevokeRefreshToken(ctx, token) error` | ~22 | Revokes a single refresh token |
| `RevokeAllRefreshTokens(ctx, memberID) error` | ~4 | Revokes all refresh tokens for a member (logout-all) |
| `ValidateAccessToken(ctx, token) (*Claims, error)` | ~29 | Validates JWT access token, returns claims |
| `GetMemberByIDForAuth(ctx, memberID) (*models.Member, error)` | ~1 | Passthrough to member repository |

**Error handling**: Returns typed `PayramError` with HTTP status codes. Password validation uses bcrypt. Token validation checks expiry and revocation status.

**Transaction boundaries**: Signup wraps member creation + API key creation in a single transaction at the repository level.

### 4.2 AuthRefreshTokenService

**Implementation**: `AuthRefreshTokenServiceImpl`
**File**: `auth_refresh_token_service_impl.go` (~53 lines)
**Dependencies**: `AuthRefreshTokenRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateRefreshToken(ctx, token) error` | ~4 | Persists a new refresh token record |
| `GetAuthTokenByToken(ctx, tokenHash) (*models.AuthRefreshToken, error)` | ~4 | Looks up token by hash |
| `UpdateAuthTokenLastUsed(ctx, tokenID) error` | ~4 | Updates last-used timestamp on token rotation |
| `RevokeAuthToken(ctx, tokenID) error` | ~4 | Sets revoked_at timestamp |
| `RevokeAllAuthTokensForMember(ctx, memberID) error` | ~4 | Bulk revoke for logout-all |
| `DeleteExpiredAuthTokens(ctx) error` | ~1 | Cleanup of expired tokens |

### 4.3 TokenService (JWT)

**Implementation**: `TokenService`
**File**: `jwt_token_service.go` (~148 lines)
**Dependencies**: Configuration (JWT secrets, expiry durations)

| Method | Lines | Description |
|--------|-------|-------------|
| `NewTokenService(accessSecret, refreshSecret, accessExpiry, refreshExpiry)` | ~54 | Constructor with secret validation |
| `GenerateTokenPair(memberID, email, roles) (*TokenPair, error)` | ~19 | Generates access + refresh JWT tokens |
| `ValidateAccessToken(token) (*Claims, error)` | ~4 | Validates access token with access secret |
| `ValidateRefreshToken(token) (*Claims, error)` | ~4 | Validates refresh token with refresh secret |
| `GetRefreshExpiration() time.Duration` | ~1 | Returns configured refresh token TTL |

**Claims structure**: Custom `Claims` struct embedding `jwt.RegisteredClaims` with `MemberID`, `Email`, `Roles` fields.

---

## 5. Member & Role Services

### 5.1 MemberService

**Implementation**: `MemberServiceImpl`
**File**: `member_service_impl.go` (~428 lines)
**Dependencies**: `MemberRepository`, `MemberExternalPlatformRoleService`, `MemberRoleService`, `EventEmitterService`, `OTPService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateRootMember(ctx, request) (*models.Member, error)` | ~11 | Creates the initial admin/root member |
| `CreateInternalMember(ctx, request) (*models.Member, error)` | ~191 | Creates internal team member with password, assigns roles |
| `GetMember(ctx, memberID, ...opts) (*models.Member, error)` | ~203 | Gets member with optional preloads |
| `GetMemberById(ctx, memberID) (*models.Member, error)` | ~4 | Simple member lookup |
| `GetInternalMemberById(ctx, memberID) (*models.Member, error)` | ~4 | Internal member lookup with role preload |
| `GetRootMember(ctx) (*models.Member, error)` | ~4 | Returns the root/admin member |
| `GetInternalMemberByIds(ctx, ids) ([]models.Member, error)` | ~4 | Batch member lookup |
| `GetInternalMembersByEmails(ctx, emails) ([]models.Member, error)` | ~4 | Lookup by email addresses |
| `GetAllInternalMembers(ctx, params) ([]models.Member, int64, error)` | ~4 | Paginated internal member list |
| `GetLast20MembersWithNoRolesOrUserRole(ctx) ([]models.Member, error)` | ~7 | For admin dashboard quick view |
| `GetMemberCount(ctx) (int64, error)` | ~4 | Total member count |
| `GetMemberByResetPasswordToken(ctx, token) (*models.Member, error)` | ~12 | Lookup for password reset flow |
| `UpdateMember(ctx, member) error` | ~4 | General member update |
| `UpdateOrCreateMember(ctx, member) error` | ~4 | Upsert member record |
| `DeleteMember(ctx, memberID) error` | ~4 | Soft-delete member |
| `DeleteMemberByEmail(ctx, email) error` | ~62 | Delete member and cascade cleanup |
| `GetInternalMemberByEmail(ctx, email) (*models.Member, error)` | ~4 | Lookup by email |
| `Activate(ctx, memberID) error` | ~35 | Activates a pending member account |
| `ChangePassword(ctx, memberID, oldPassword, newPassword) error` | ~39 | Validates old password, hashes and stores new |
| `CheckRootMemberExist(ctx) (bool, error)` | ~5 | Checks if root member has been created |
| `ForgotPassword(ctx, email) error` | ~41 | Generates reset token, emits reset email event |
| `ResetPassword(ctx, token, newPassword) error` | ~22 | Validates token, sets new password |

### 5.2 MemberRoleService

**Implementation**: `MemberRoleServiceImpl`
**File**: `member_role_service_impl.go` (~90 lines)
**Dependencies**: `MemberRoleRepository`, `RoleRepository`, `MemberRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `AssignAdminRoleToMember(ctx, memberID) error` | ~23 | Assigns the admin role to a member |
| `GetRoleNameByMemberID(ctx, memberID) (string, error)` | ~4 | Returns primary role name for a member |
| `RemoveAdminRoleFromMember(ctx, memberID) error` | ~26 | Removes admin role, prevents removing from root member |

### 5.3 MemberExternalPlatformRoleService

**Implementation**: `MemberExternalPlatformRoleServiceImpl`
**File**: `member_external_platform_role_service_impl.go` (~347 lines)
**Dependencies**: `MemberExternalPlatformRoleRepository`, `RoleRepository`, `MemberRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, memberExtPlatformRole) error` | ~9 | Creates a platform-specific role assignment |
| `GetByMemberId(ctx, memberID) ([]models.MemberExternalPlatformRole, error)` | ~4 | Gets all platform roles for a member |
| `CreateOrUpdateMember(ctx, request) (*models.Member, error)` | ~11 | Creates or updates member with platform role |
| `AssignPlatformRolesToMember(ctx, memberID, platformID, roleIDs) error` | ~12 | Assigns multiple roles to member for a platform |
| `AssignMemberPlatformRoles(ctx, request) error` | ~24 | Validates and assigns roles with permission checks |
| `Revoke(ctx, memberID, platformID) error` | ~24 | Revokes all platform roles for a member |
| `UpdateMemberPlatformRoles(ctx, request) error` | ~29 | Updates existing role assignments |
| `DeleteAllPlatformRolesForMember(ctx, memberID, platformID) error` | ~18 | Removes all roles for member on a platform |
| `RemovePlatformRolesForMember(ctx, request) error` | ~17 | Removes specific roles |
| `DeleteAllRolesForMember(ctx, memberID) error` | ~11 | Removes all roles across all platforms |
| `GetByMemberIdAndExternalPlatformID(ctx, memberID, platformID) ([]MemberExternalPlatformRole, error)` | ~4 | Gets roles for specific platform |
| `GetMemberByCustomerIDAndExternalPlatformID(ctx, customerID, platformID) (*models.Member, error)` | ~4 | Merchant member lookup |
| `GetMembersByExternalPlatformIDs(ctx, platformIDs) ([]models.Member, error)` | ~2 | Batch platform member lookup |

**Private methods**: `validateRoleAssignments()` (~84 lines), `validateRoleRemovals()` (~51 lines), `checkMemberPermission()` (~28 lines).

### 5.4 RoleService

**Implementation**: `RoleServiceImpl`
**File**: `role_service_impl.go`
**Dependencies**: `RoleRepository`, `MemberService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, role) (*models.Role, error)` | ~10 | Creates a new role |
| `GetAllInternalMemberRoles(ctx, ...opts) ([]models.Role, error)` | ~14 | Lists all roles with optional preloads |
| `GetByName(ctx, name) (*models.Role, error)` | ~4 | Role lookup by name |
| `GetByNames(ctx, names) ([]models.Role, error)` | ~4 | Batch role lookup |
| `GetByIds(ctx, ids) ([]models.Role, error)` | ~4 | Batch role lookup by IDs |
| `GetRoleNameByID(ctx, roleID) (string, error)` | ~4 | Gets role display name |
| `UpdateRolePermissions(ctx, roleID, permissionIDs) error` | ~8 | Updates permissions for a role |

**Setter**: `SetMemberService(svc)` -- breaks circular dependency with MemberService.

### 5.5 PermissionService

**Implementation**: `PermissionServiceImpl`
**File**: `permission_service_impl.go`
**Dependencies**: `PermissionRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, permission) (*models.Permission, error)` | ~4 | Creates a permission record |
| `GetByNames(ctx, names) ([]models.Permission, error)` | ~4 | Batch permission lookup by name |

### 5.6 OTPService

**Implementation**: `OTPServiceImpl`
**File**: `otp_service_impl.go`
**Dependencies**: `OTPRepository`, `EventEmitterService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GenerateOTP(ctx, entityID, purpose) (*models.OTP, error)` | ~30 | Generates 6-digit OTP, stores hashed, emits email event |
| `RegenerateOTP(ctx, entityID, purpose) (*models.OTP, error)` | ~20 | Invalidates existing OTPs, generates new one |
| `Validate(ctx, entityID, purpose, code) error` | ~25 | Validates OTP code, checks attempts and expiry |
| `InvalidateOTP(ctx, entityID, purpose) error` | ~4 | Marks all non-verified OTPs as invalid |
| `IsVerified(ctx, entityID, purpose) (bool, error)` | ~4 | Checks if entity has a verified OTP |
| `RegisterValidator(ctx, purpose, validator OTPValidatable)` | ~4 | Registers a callback for OTP validation events |

**OTPValidatable interface**: Services that need OTP verification (RecipientService, WithdrawalProcessingService) implement:
```go
type OTPValidatable interface {
    ValidateOTPSource(ctx, entityID, purpose) error
    OnOTPValidated(ctx, entityID, purpose, eventMetadata) error
}
```

---

## 6. Blockchain Services

### 6.1 BlockchainService

**Implementation**: `BlockchainServiceImpl`
**File**: `blockchain_service_impl.go` (~1019 lines)
**Dependencies**: `BlockchainRepository`, `RPCNodeRepository`, `PaymentService`, `DepositAddressesService`, `InternalBlockchainTransactionService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Get(ctx, params) ([]models.Blockchain, error)` | ~13 | Lists blockchains with filters |
| `GetByID(ctx, blockchainID) (*models.Blockchain, error)` | ~9 | Blockchain by ID with RPC nodes preloaded |
| `GetByCode(ctx, code) (*models.Blockchain, error)` | ~17 | Blockchain lookup by code (e.g., "eth", "btc") |
| `Update(ctx, blockchain) (*models.Blockchain, error)` | ~60 | Updates blockchain config (confirmations, status, etc.) |
| `CreateDeposit(ctx, deposit) (*models.Deposit, error)` | ~50 | Records a new blockchain deposit |
| `GetDeposits(ctx, params) ([]models.Deposit, error)` | ~5 | Queries deposits with filters |
| `UpdateDepositAfterConfirmation(ctx, deposit) error` | ~5 | Updates deposit status after block confirmations |
| `UpdateDepositStatus(ctx, deposit) error` | ~6 | Updates deposit status directly |
| `UpsertDepositAfterVerification(ctx, deposit) error` | ~6 | Idempotent deposit upsert by unique tx hash |
| `MarkStaleConfirmingDeposits(ctx, blockchain, cutoff) error` | ~8 | Marks old confirming deposits as stale |
| `ProcessAllValidatedWithdrawals(ctx) error` | ~17 | Processes accounting for validated withdrawals |
| `ProcessAllPendingPaymentRequestsAccounting(ctx) error` | ~16 | Runs double-entry accounting for confirmed payments |
| `GetAllUnspentTransactions(ctx, params) ([]models.Deposit, error)` | ~54 | Gets unspent deposits for UTXO-based chains |
| `Withdrawn(ctx, params) ([]models.Withdraw, error)` | ~15 | Gets withdrawal records |
| `GetDefaultFeePercentage(ctx, blockchainCode) (decimal.Decimal, error)` | ~22 | Gets fee percentage from config |
| `GetSpentUTXOs(ctx, params) ([]models.WithdrawDepositsBTC, error)` | ~14 | Gets spent UTXO records |
| `CreateUTXOEntries(ctx, entries) error` | ~56 | Creates UTXO records for BTC deposits |
| `GetWithdrawsByStatus(ctx, status) ([]models.Withdraw, error)` | ~4 | Gets withdrawals by status |
| `UpdateWithdraw(ctx, withdraw) error` | ~6 | Updates withdrawal record |
| `GetPaymentRequestsWithConfirmingDeposits(ctx) ([]models.PaymentRequest, error)` | ~4 | Gets payments with unconfirmed deposits |
| `UpdateBlockHeightForBlockchainId(ctx, blockchainID, height) error` | ~4 | Updates chain height after block processing |
| `CheckServerURLConnection(ctx, url, blockchain) error` | ~13 | Tests RPC node connectivity |
| `CheckAllNodeConnections(ctx, blockchain) ([]NodeStatus, error)` | ~508 | Tests all RPC nodes, validates chain IDs |
| `CreateRPCNode(ctx, node) (*models.RPCNode, error)` | ~63 | Creates RPC node with uniqueness check |
| `UpdateRPCNode(ctx, nodeID, updates) (*models.RPCNode, error)` | ~128 | Updates RPC node config, reloads pool |

**Setter methods**: `SetPaymentService()`, `SetDepositAddressServiceService()`, `SetInternalBlockchainTransactionService()`.

**Private methods**: `ensureUniqueRPCNode()`, `fetchChainIdentifier()`, `reloadRPCPool()`, `validateActiveNodesChainID()`.

### 6.2 BlockchainCurrencyService

**Implementation**: `BlockchainCurrencyServiceImpl`
**File**: `blockchain_currency_service_impl.go` (~365 lines)
**Dependencies**: `BlockchainCurrencyRepository`, `BlockchainRepository`, `CurrencyRepository`, `PaymentService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, request) (*models.BlockchainCurrency, error)` | ~37 | Creates blockchain-currency mapping (e.g., USDT on Ethereum) |
| `Update(ctx, bcID, request) (*models.BlockchainCurrency, error)` | ~81 | Updates currency params (fees, limits, sweep config) |
| `GetByBlockchainAndCurrencyCode(ctx, bcCode, ccCode) (*models.BlockchainCurrency, error)` | ~15 | Lookup by blockchain + currency code pair |
| `GetAllBlockchainCurrencies(ctx) ([]models.BlockchainCurrency, error)` | ~4 | Lists all configured currencies |
| `GetByBlockchainAndCurrencyCodeWithBlockchainAndCurrencyPreload(ctx, bcCode, ccCode) (*models.BlockchainCurrency, error)` | ~15 | Lookup with full preloads |
| `GetBlockchainCurrenciesByCurrencyIdWithBlockchainPreload(ctx, currencyID) ([]models.BlockchainCurrency, error)` | ~4 | Gets all chains for a currency |
| `GetBlockchainCurrenciesByBlockchainIdWithBlockchainAndCurrencyPreload(ctx, bcID) ([]models.BlockchainCurrency, error)` | ~4 | Gets all currencies on a chain |
| `GetByCurrencyAddressAndBlockchainIDWithBlockchainAndCurrencyPreload(ctx, addr, bcID) (*models.BlockchainCurrency, error)` | ~4 | Token contract address lookup |
| `GetAllPublicDetailsBlockchainCurrencies(ctx) ([]BlockchainCurrencyPublic, error)` | ~24 | Public-facing currency list (no internal fields) |
| `GetAllPublicDetailsBlockchainCurrenciesForMemberId(ctx, memberID) ([]BlockchainCurrencyPublic, error)` | ~79 | Currency list filtered by member's platform |
| `GetSupportedBlockchainCurrencies(ctx, platformID) ([]BlockchainCurrencyPublic, error)` | ~45 | Gets currencies enabled for a specific platform |

**Setter**: `SetPaymentService()`.

### 6.3 BlockchainFamilyService

**Implementation**: `BlockchainFamilyServiceImpl`
**File**: `blockchain_family_service_impl.go` (~48 lines)
**Dependencies**: `BlockchainFamilyRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, family) (*models.BlockchainFamily, error)` | ~8 | Creates a blockchain family (ethereum, bitcoin, tron) |
| `GetByFamily(ctx, family) (*models.BlockchainFamily, error)` | ~4 | Lookup by family name |
| `GetAll(ctx) ([]models.BlockchainFamily, error)` | ~4 | Lists all families |

### 6.4 BlockchainContractService

**Implementation**: `BlockchainContractServiceImpl`
**File**: `blockchain_contract_service_impl.go` (~57 lines)
**Dependencies**: `BlockchainContractRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, contract) (*models.BlockchainContract, error)` | ~10 | Creates a blockchain contract record |
| `GetByID(ctx, contractID) (*models.BlockchainContract, error)` | ~4 | Contract lookup |
| `GetByBlockchainCodeAndContractType(ctx, bcCode, contractType) (*models.BlockchainContract, error)` | ~4 | Lookup by chain + type |
| `GetByContractType(ctx, contractType) (*models.BlockchainContract, error)` | ~4 | Lookup by type |
| `GetAllContractsByAddressContractType(ctx, contractType) ([]models.BlockchainContract, error)` | ~1 | Lists contracts by address type |

---

## 7. Wallet Services

### 7.1 WalletService

**Implementation**: `WalletServiceImpl`
**File**: `wallet_service_impl.go`
**Dependencies**: `WalletRepository`, `BlockchainFamilyService`, `BlockchainService`, `AddressPoolService`, `WalletFunctionService`, `ExternalPlatformWalletBlockchainFamilyService`, `ContractAddressService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetWalletByID(ctx, walletID) (*models.Wallet, error)` | ~4 | Wallet lookup with enriched details |
| `GetWallets(ctx, params) ([]models.Wallet, error)` | ~4 | Lists wallets with filters |
| `GetWalletsWithReferenceID(ctx, params) ([]models.Wallet, error)` | ~8 | Lists wallets filtered by reference/platform |
| `GetWalletSCWs(ctx, params) ([]models.WalletSCW, error)` | ~4 | Gets smart contract wallet deployments |
| `CreateHotWallet(ctx, request) (*models.Wallet, error)` | ~50 | Creates hot wallet with encrypted private key |
| `CreateColdWallet(ctx, request) (*models.Wallet, error)` | ~30 | Creates cold/watch-only wallet with xpub |
| `CreateBulkEOADepositWallet(ctx, request) (*models.Wallet, error)` | ~40 | Creates bulk EOA deposit wallet, triggers address generation |
| `CreateDepositWalletSCW(ctx, request) (*models.Wallet, error)` | ~60 | Creates SCW deposit wallet with factory deployment |
| `CreateHotWalletFunctionsAssociation(ctx, request) error` | ~20 | Associates hot wallet with specific functions (sweeping, gas) |
| `Activate(ctx, walletID) error` | ~10 | Sets wallet status to active |
| `Deactivate(ctx, walletID) error` | ~10 | Sets wallet status to inactive |
| `Delete(ctx, walletID) error` | ~20 | Soft-deletes wallet with safety checks |
| `Update(ctx, walletID, updates) (*models.Wallet, error)` | ~15 | General wallet update |
| `SetDefaultWallet(ctx, walletID) error` | ~10 | Sets wallet as default for its family |
| `SetFundCollectorAddress(ctx, walletID, address) error` | ~15 | Sets the fund collector (cold storage) address |
| `GenerateAddresses(ctx, walletID, count) error` | ~30 | Generates deposit addresses for HD wallets |
| `GenerateAddressesIfPoolLow(ctx, walletID) error` | ~25 | Auto-generates addresses when pool is low |
| `GetActiveWalletXPUB(ctx, family) (*models.WalletXpub, error)` | ~10 | Gets the active xpub for a family |
| `UpdateSecretsVaultActivities(ctx, request) error` | ~15 | Updates wallet function associations |

**Private methods**: `createUnassignedProjectsWalletMappings()`, `generateDepositWalletAddressesForETHFamily()`, `generateDepositWalletAddressesForTRXFamily()`.

### 7.2 WalletFunctionService

**Implementation**: `WalletFunctionServiceImpl`
**File**: `wallet_function_service_impl.go`
**Dependencies**: `WalletFunctionRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, walletFunction) (*models.WalletFunction, error)` | ~8 | Creates a wallet function record |
| `GetByActivityType(ctx, activityType) (*models.WalletFunction, error)` | ~4 | Lookup by activity type (e.g., "sweep", "gas_transfer") |
| `GetByActivityTypes(ctx, types) ([]models.WalletFunction, error)` | ~4 | Batch lookup |
| `GetAllVaultActivities(ctx) ([]models.WalletFunction, error)` | ~4 | Lists all wallet functions |
| `CreateWalletToFunctionsAssociation(ctx, walletID, functionIDs) error` | ~8 | Associates wallet with functions |

---

## 8. Address Services

### 8.1 AddressService

**Implementation**: `AddressServiceImpl`
**File**: `address_service_impl.go` (~243 lines)
**Dependencies**: `AddressRepository`, `GenericDataStoreService`, `BlockchainCurrencyService`, `WalletService`, `BlockchainService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Balances(ctx, params) ([]AddressBalance, error)` | ~21 | Gets aggregated address balances by blockchain/currency |
| `PendingForApproval(ctx, params) ([]AddressApproval, error)` | ~4 | Gets addresses pending ERC20 approval for sweep |
| `GetEligibleAddressesToTransferFees(ctx, params) ([]AddressFeeTransfer, error)` | ~4 | Gets addresses that need gas fee top-up |
| `GetEligibleSCWAddressPoolsToBroadcast(ctx, params) ([]AddressPool, error)` | ~4 | Gets SCW addresses ready to deploy |
| `GetEligibleAddressesToSweep(ctx, params) ([]AddressSweep, error)` | ~8 | Gets addresses eligible for sweep (above min amount) |
| `CreateDataStoreForSweepInitiatedAddresses(ctx, addresses) error` | ~77 | Creates generic data store records to track sweep-locked addresses |
| `UpdateDataStoreForSweepCancelledAddresses(ctx, addresses) error` | ~22 | Unlocks addresses when sweep is cancelled |
| `ProcessAddressBalanceAndUpdateStatus(ctx) error` | ~1 | Processes balance updates for all addresses |

**Private method**: `createAddressEntryToGenericDatastore()` (~47 lines) -- builds data store entries with blockchain/currency/address info.

### 8.2 AddressPoolService

**Implementation**: `AddressPoolServiceImpl`
**File**: `address_pool_service_impl.go` (~78 lines)
**Dependencies**: `AddressPoolRepository`, `BlockchainFamilyRepository`, `ConfigurationRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetByBlockchainFamilyAndAddressAndStatus(ctx, family, address, status) (*models.AddressPool, error)` | ~4 | Address pool lookup |
| `GetAddressesWithUnassignedPrivateKeys(ctx, ...opts) ([]models.AddressPool, error)` | ~1 | Gets addresses needing key assignment |
| `AddPrivateKeys(ctx, addresses) error` | ~4 | Adds encrypted private keys to address pool |
| `GetPrivateKeys(ctx, addresses) ([]AddressKey, error)` | ~4 | Decrypts and returns private keys |
| `GetCountOfUnassignedAddresses(ctx, family) (int64, error)` | ~4 | Count of available deposit addresses |
| `GetLastGeneratedAddress(ctx, walletID, family) (*models.AddressPool, error)` | ~4 | Gets the last generated address for path index tracking |
| `BulkSaveAddresses(ctx, addresses) error` | ~4 | Bulk creates address pool entries |
| `GetByAddress(ctx, address) (*models.AddressPool, error)` | ~4 | Address lookup |
| `ProcessDepositWalletDeployment(ctx) error` | ~1 | Processes pending SCW deployments |

### 8.3 AddressDeploymentService

**Implementation**: `AddressDeploymentServiceImpl`
**File**: `address_deployment_service_impl.go` (~231 lines)
**Dependencies**: `AddressDeploymentRepository`, `BlockchainService`

| Method | Lines | Description |
|--------|-------|-------------|
| `BulkCreateAddressDeployment(ctx, deployments) error` | ~36 | Creates deployment records for SCW addresses |
| `BulkMarkAsBroadcasted(ctx, deployments, txHashes) error` | ~38 | Marks deployments as broadcasted with tx hash |
| `BulkMarkAsActive(ctx, deployments) error` | ~41 | Marks deployments as confirmed/active |
| `GetAddressDeployments(ctx, params) ([]models.AddressDeployment, error)` | ~4 | Lists deployments with filters |
| `DeleteAddressDeployment(ctx, deploymentID) error` | ~8 | Removes a deployment record |
| `ProcessAddressDeploymentsAccounting(ctx, deployments) error` | ~67 | Processes accounting for deployment gas costs |

### 8.4 AddressContractSignatureService

**Implementation**: `AddressContractSignatureServiceImpl`
**File**: `address_contract_signature_service_impl.go` (~119 lines)
**Dependencies**: `AddressContractSignatureRepository`, `BlockchainCurrencyService`, `ContractAddressService`, `AddressPoolService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, request) (*models.AddressContractSignature, error)` | ~64 | Creates an ERC20 approval signature for sweep |
| `UpdateStatusByAddressAndContractType(ctx, address, contractType, status) error` | ~4 | Updates approval status |
| `GetByBlockchainCurrencyCodesAddressContractTypeAndAddress(ctx, bcCode, ccCode, type, addr) (*models.AddressContractSignature, error)` | ~4 | Specific signature lookup |
| `GetByBlockchainCodeContractTypeAndStatus(ctx, bcCode, type, status) ([]models.AddressContractSignature, error)` | ~1 | Batch status lookup |

### 8.5 AccountAddressService

**Implementation**: `AccountAddressServiceImpl`
**File**: `account_address_service_impl.go` (~37 lines)
**Dependencies**: `AccountAddressRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetByBlockchainCurrencyCodeAndAddress(ctx, bcCode, ccCode, address) (*models.AccountAddress, error)` | ~4 | Account address lookup |
| `GetAllPendingAccountsForSweepTransaction(ctx) ([]models.AccountAddress, error)` | ~1 | Gets accounts pending sweep |

---

## 9. Deposit Services

### 9.1 DepositService

**Implementation**: `DepositServiceImpl`
**File**: `deposit_service_impl.go` (~123 lines)
**Dependencies**: `DepositRepository`, `BlockchainService`, `PaymentService`, `EventEmitterService`, `WebhookDeliveryService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetDepositsGreaterThanID(ctx, id, blockchain) ([]models.Deposit, error)` | ~5 | Gets deposits after a given ID for incremental processing |
| `ProcessAllPendingDeposits(ctx) error` | ~66 | Processes all pending deposits: runs accounting, emits payment events, triggers webhooks. Spawns goroutines for webhook and event emission |

**Business logic**: `ProcessAllPendingDeposits` is the core deposit processing method. For each pending deposit, it:
1. Runs double-entry accounting (debit asset, credit liability)
2. Links deposit to open payment request
3. Emits payment received event for email notification
4. Triggers webhook delivery to merchant

### 9.2 MissedDepositService

**Implementation**: `MissedDepositServiceImpl`
**File**: `missed_deposit_service_impl.go` (~238 lines)
**Dependencies**: `MissedDepositRepository`, `BlockchainService`, blockchain clients

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateMissedDeposit(ctx, request) (*models.MissedDeposit, error)` | ~114 | Creates a missed deposit record, verifies tx on-chain, processes if valid |
| `GetMissedDeposits(ctx, params) ([]models.MissedDeposit, error)` | ~4 | Lists missed deposits |
| `UpdateMissedDeposit(ctx, deposit) error` | ~25 | Updates missed deposit status |

**Private method**: `getBlockchainTransactionDetails()` (~19 lines) -- fetches transaction details from blockchain client.

---

## 10. Sweep Services

### 10.1 SweepService

**Implementation**: `SweepServiceImpl`
**File**: `sweep_service_impl.go`
**Dependencies**: `SweepRepository`, `BlockchainService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, sweep) (*models.Sweep, error)` | ~9 | Creates a sweep record |
| `CreateEntry(ctx, sweep) error` | ~4 | Creates sweep entry without return |
| `CreateBitcoinFamilySweepEntry(ctx, sweep) error` | ~10 | Creates BTC-specific sweep record |
| `CreateSmartContractSweepEntry(ctx, sweep) error` | ~10 | Creates EVM/TRX sweep record |
| `Update(ctx, sweep) error` | ~9 | Updates sweep record |
| `BulkCreate(ctx, sweeps) error` | ~27 | Batch creates sweep records |
| `GetByToAddressAndStatus(ctx, toAddr, status) ([]models.Sweep, error)` | ~4 | Sweep lookup |
| `GetAllSweeps(ctx, params) ([]models.Sweep, error)` | ~22 | Paginated sweep list |
| `GetConfirmingSweeps(ctx) ([]models.Sweep, error)` | ~14 | Gets sweeps pending confirmation |
| `CompleteSweep(ctx, sweepID) error` | ~10 | Marks sweep as completed |
| `MarkSweepAsNotFound(ctx, sweepID) error` | ~10 | Marks sweep tx as not found on chain |
| `Swept(ctx, params) (int64, error)` | ~8 | Count of completed sweeps |
| `GetLastSweepByBlockchainCodeAndCurrency(ctx, bcCode, ccCode) (*models.Sweep, error)` | ~4 | Gets last sweep for a chain/currency |

### 10.2 SweepTransactionService

**Implementation**: `SweepTransactionServiceImpl`
**File**: `sweep_transaction_service_impl.go`
**Dependencies**: `SweepTransactionRepository`, `BlockchainService`, `WalletService`, `AddressService`, `ConfigurationService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Get(ctx, ...opts) ([]models.SweepTransaction, error)` | ~10 | Gets sweep transactions with flexible WHERE clauses |
| `GetSweepTransactions(ctx, params) ([]SweepTransactionResponse, error)` | ~15 | Paginated sweep transaction list with enriched details |
| `GetSweepTransaction(ctx, txID) (*SweepTransactionDetail, error)` | ~10 | Single sweep transaction detail |
| `GetSweepTransactionDeprecated(ctx, ...opts) (*models.SweepTransaction, error)` | ~10 | Legacy getter |
| `GetConfirmingSweeps(ctx) ([]models.SweepTransaction, error)` | ~4 | Gets sweep txns pending confirmation |
| `AddAddressToTransaction(ctx, txID, address) error` | ~5 | Adds an EVM/TRX address to a sweep batch |
| `AddUTXOToTransaction(ctx, txID, utxo) error` | ~5 | Adds a UTXO to a BTC sweep batch |
| `UpdateSweepTransaction(ctx, txID, updates) error` | ~8 | Updates sweep transaction |
| `UpdateSweepTransactionStatus(ctx, txID, status) error` | ~8 | Status-only update |
| `EthAutoSweepTransaction(ctx, params) error` | ~30 | Creates and initiates an automatic ETH sweep |
| `CompleteSweep(ctx, txID) error` | ~10 | Marks sweep transaction as completed |
| `MarkSweepAsConfirming(ctx, txID, txHash) error` | ~8 | Marks as confirming with tx hash |
| `MarkSweepAsNotFound(ctx, txID) error` | ~8 | Marks sweep tx as not found |

**Private methods**: `initiateSweep()`, `initiateSweepWithAdaptiveBatchSize()`, `attemptSweepWithRetry()`, `createSweepTransactionWithUTXODetails()`, `getFeePercentage()`, `populateSweepTransactionsOtherDetails()`.

### 10.3 SweepUTXOService (UTXOs for sweep)

**Implementation**: `SweepUTXOServiceImpl`
**File**: `sweep_utxo_service_impl.go`
**Dependencies**: `SweepUTXORepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `ProcessUTXOForSweepTransaction(ctx) error` | ~20 | Processes UTXOs ready for sweep batching |
| `UpdateUTXOsStatusToSpentAndSweepTxId(ctx, utxoIDs, sweepTxID) error` | ~4 | Marks UTXOs as spent after sweep |

### 10.4 UTXOService

**Implementation**: `UTXOServiceImpl`
**File**: `utxo_service_impl.go`
**Dependencies**: `UTXORepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetUnspentUTXOs(ctx, params) ([]models.UTXO, error)` | ~4 | Gets unspent UTXOs for a chain/currency |
| `UpdateUTXOsStatusToSpent(ctx, utxoIDs) error` | ~4 | Marks UTXOs as spent |
| `GetAllPendingUTXOsForSweepTransaction(ctx) ([]models.UTXO, error)` | ~4 | Gets UTXOs pending sweep |
| `IdentifyOurUTXOsFromList(ctx, utxos) ([]models.UTXO, error)` | ~4 | Filters UTXOs to identify owned ones |

---

## 11. Payment Services

### 11.1 PaymentService

**Implementation**: `PaymentServiceImpl`
**File**: `payment_service_impl.go`
**Dependencies**: `PaymentRepository`, `BlockchainCurrencyService`, `ExternalPlatformService`, `MemberExternalPlatformRoleService`, `EventEmitterService`, `WebhookDeliveryService`, `WebsocketTokenService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateNewPaymentRequest(ctx, request) (*models.PaymentRequest, error)` | ~40 | Creates payment request, cancels previous open requests, emits new payment event. Spawns goroutine for cancellation events |
| `CreatePaymentRequest(ctx, request) (*models.PaymentRequest, error)` | ~20 | Direct payment request creation |
| `GetPaymentRequestById(ctx, requestID) (*models.PaymentRequest, error)` | ~4 | Payment request lookup |
| `GetPaymentRequestByReferenceId(ctx, referenceID) (*models.PaymentRequest, error)` | ~4 | Lookup by merchant reference ID |
| `GetPaymentRequestDetailsByReferenceId(ctx, referenceID) (*PaymentDetail, error)` | ~10 | Detailed payment info for checkout page |
| `GetPaymentDetailsByReferenceId(ctx, referenceID) (*PaymentDetail, error)` | ~10 | Public payment details |
| `GetClosedPaymentsByMemberId(ctx, memberID) ([]models.PaymentRequest, error)` | ~4 | Gets completed payments for a member |
| `GetPaymentRequestsPendingWebhookApproval(ctx, params) ([]models.PaymentRequest, error)` | ~10 | Payments needing manual webhook approval |
| `GetTotalRevenueFromReferredUsers(ctx, memberIDs) (decimal.Decimal, error)` | ~4 | Revenue from referred users |
| `SearchPaymentRequests(ctx, params) (*SearchResult, error)` | ~10 | Full-text search across payments |
| `Summary(ctx, params) (*PaymentSummary, error)` | ~10 | Payment summary/dashboard metrics |
| `ProcessAllPendingWebhooksToMerchant(ctx) error` | ~20 | Processes pending webhook deliveries for confirmed payments |
| `SendWebhookForMissedDeposit(ctx, paymentRequestID) error` | ~10 | Sends webhook for missed deposit recovery |
| `DiscardWebhookForMissedDeposit(ctx, paymentRequestID) error` | ~10 | Discards missed deposit webhook |
| `SocketUpdate(ctx, paymentRequest) error` | ~10 | Sends real-time update via WebSocket |

**Private methods**: `processConfirmedPaymentsWebhooks()`, `processConfirmingDepositsWebhooks()`.

### 11.2 PaymentChannelService

**Implementation**: `PaymentChannelServiceImpl`
**File**: `payment_channel_service_impl.go`
**Dependencies**: `PaymentChannelRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetPaymentChannels(ctx, params) ([]models.PaymentChannel, error)` | ~4 | Lists payment channels (crypto, card, etc.) |
| `GetPaymentChannelByID(ctx, channelID) (*models.PaymentChannel, error)` | ~4 | Channel lookup |
| `GetPaymentChannelAPIKey(ctx, channelID) (string, error)` | ~4 | Gets API key for a channel |
| `GetPaymentChannelAPIKeyByReferenceID(ctx, refID) (string, error)` | ~4 | API key by reference |
| `GetPaymentChannelsByProjectID(ctx, projectID) ([]models.PaymentChannel, error)` | ~8 | Channels for a project |
| `GetPaymentChannelsByReferenceID(ctx, refID) ([]models.PaymentChannel, error)` | ~8 | Channels by reference |
| `UpdatePaymentChannel(ctx, channelID, updates) (*models.PaymentChannel, error)` | ~10 | Updates channel config |
| `UpdatePaymentChannelAPIKey(ctx, channelID, apiKey) error` | ~4 | Updates channel API key |
| `ActivatePaymentChannel(ctx, channelID) error` | ~4 | Activates a payment channel |
| `DeactivatePaymentChannel(ctx, channelID) error` | ~4 | Deactivates a payment channel |

### 11.3 PaymentChannelProjectService

**Implementation**: `PaymentChannelProjectServiceImpl`
**File**: `payment_channel_project_service_impl.go`
**Dependencies**: `PaymentChannelProjectRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `ActivatePaymentChannelProject(ctx, projectID, channelID) error` | ~10 | Enables a channel for a project |
| `DeactivatePaymentChannelProject(ctx, projectID, channelID, reason) error` | ~10 | Disables a channel for a project |
| `GetDisabledPaymentChannelProjectsByProjectID(ctx, projectID) ([]models.DisabledPaymentChannelProject, error)` | ~4 | Lists disabled channels |
| `UpdatePaymentChannelProjects(ctx, projectID, request) error` | ~15 | Bulk update channel-project mappings |

---

## 12. Withdrawal Services

### 12.1 WithdrawalService

**Implementation**: `WithdrawalServiceImpl`
**File**: `withdrawal_service_impl.go`
**Dependencies**: `WithdrawalRepository`, `BlockchainService`, `BlockchainCurrencyService`, `ConfigurationService`, `MemberService`, `OTPService`, `EventEmitterService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreatePayout(ctx, request) (*models.Withdrawal, error)` | ~40 | Creates a payout/withdrawal request with validation |
| `CreatePayoutForMerchant(ctx, request) (*models.Withdrawal, error)` | ~30 | Creates payout from merchant API |
| `CreateMerchantPayoutFromMerchant(ctx, request) (*models.Withdrawal, error)` | ~30 | Creates merchant-to-merchant payout |
| `ApproveWithdrawal(ctx, withdrawalID, approverID) error` | ~20 | Approves a pending withdrawal |
| `ApproveWithdrawalForMerchant(ctx, withdrawalID, approverID) error` | ~20 | Merchant-side approval |
| `RejectWithdrawal(ctx, withdrawalID, rejectorID, reason) error` | ~15 | Rejects a withdrawal |
| `Cancel(ctx, withdrawalID) error` | ~10 | Cancels a withdrawal |
| `GetWithdrawals(ctx, params) ([]models.Withdrawal, int64, error)` | ~10 | Paginated withdrawal list |
| `GetWithdrawalsFromMerchant(ctx, params) ([]models.Withdrawal, int64, error)` | ~10 | Merchant-filtered withdrawals |
| `GetWithdrawalsEligibleToSent(ctx) ([]models.Withdrawal, error)` | ~10 | Gets approved withdrawals ready for broadcast |
| `WithdrawalInitiated(ctx, withdrawalID, txHash) error` | ~10 | Marks withdrawal as initiated with tx hash |
| `WithdrawalSent(ctx, withdrawalID) error` | ~10 | Marks withdrawal as sent/confirming |
| `MarkWithdrawalFailed(ctx, withdrawalID, reason) error` | ~10 | Marks withdrawal as failed |
| `UpdateWithdrawalRetryCount(ctx, withdrawalID) error` | ~10 | Increments retry counter |
| `IsAboveMinimumAmount(ctx, amount, bcCode, ccCode) (bool, error)` | ~10 | Validates minimum withdrawal amount |
| `GetAmountInUsd(ctx, amount, ccCode) (decimal.Decimal, error)` | ~10 | Converts amount to USD |
| `GetDefaultPayoutPolicy(ctx) (*PayoutPolicy, error)` | ~10 | Gets default payout policy from config |
| `GetMemberPayoutPolicy(ctx, memberID) (*PayoutPolicy, error)` | ~10 | Gets member-specific payout policy |
| `GetMemberTotalPayoutForToday(ctx, memberID) (decimal.Decimal, error)` | ~10 | Gets member's daily payout total |
| `GetMerchantHourlyTotalPayout(ctx, platformID) (decimal.Decimal, error)` | ~10 | Gets merchant hourly payout total |
| `ValidateAmountForPayoutPolicy(ctx, amount, policy) error` | ~10 | Validates amount against policy |
| `ValidatePayoutPolicy(ctx, memberID, amount) error` | ~10 | Full policy validation |

### 12.2 WithdrawalProcessingService

**Implementation**: `WithdrawalProcessingServiceImpl`
**File**: `withdrawal_processing_service_impl.go`
**Dependencies**: `WithdrawalProcessingRepository`, `WithdrawalService`, `BlockchainService`, `WalletService`, blockchain clients

| Method | Lines | Description |
|--------|-------|-------------|
| `ProcessWithdrawals(ctx) error` | ~30 | Processes all approved withdrawals -- signs and broadcasts transactions |
| `VerifyInitiatedWithdrawals(ctx) error` | ~20 | Checks on-chain confirmation of initiated withdrawals |
| `CheckForTransactionConfirmation(ctx, withdrawal) error` | ~15 | Checks single withdrawal tx confirmation |
| `ProcessAccountingForWithdrawalSent(ctx, withdrawal) error` | ~20 | Runs double-entry accounting after withdrawal confirms |
| `MarkWithdrawalFailed(ctx, withdrawalID, reason) error` | ~10 | Marks withdrawal failed with accounting reversal |
| `ValidateOTPSource(ctx, entityID, purpose) error` | ~10 | OTP validation source (implements OTPValidatable) |
| `OnOTPValidated(ctx, entityID, purpose, metadata) error` | ~15 | Called when OTP is validated -- approves withdrawal |

**Private methods**: `withdraw()` (~50 lines, signs and broadcasts), `checkForTransactionConfirmationObj()`, `updateWithdrawalRetryCount()`.

---

## 13. External Platform Services

### 13.1 ExternalPlatformService

**Implementation**: `ExternalPlatformServiceImpl`
**File**: `external_platform_service_impl.go` (~701 lines)
**Dependencies**: `ExternalPlatformRepository`, `MemberExternalPlatformRoleService`, `BlockchainCurrencyService`, `WalletService`, `APIKeyService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, request) (*models.ExternalPlatform, error)` | ~46 | Creates a project/merchant with API key, wallet mappings, currency mappings (transactional) |
| `GetAll(ctx, params) ([]models.ExternalPlatform, error)` | ~21 | Lists all platforms with permission filtering |
| `GetAllWithDetails(ctx) ([]models.ExternalPlatform, error)` | ~29 | Lists with full preloads |
| `GetById(ctx, platformID) (*models.ExternalPlatform, error)` | ~17 | Platform lookup with permission check |
| `GetByIds(ctx, platformIDs) ([]models.ExternalPlatform, error)` | ~22 | Batch platform lookup |
| `GetByIdWithoutPermissionCheck(ctx, platformID) (*models.ExternalPlatform, error)` | ~14 | Internal-use platform lookup |
| `Update(ctx, platformID, request) (*models.ExternalPlatform, error)` | ~94 | Updates platform config, handles logo upload |
| `GrantPermissions(ctx, platformID, memberID) error` | ~19 | Grants platform access to a member |
| `RevokePermissions(ctx, platformID, memberID) error` | ~19 | Revokes platform access |
| `GetMembersByExternalPlatformIDs(ctx, platformIDs) ([]models.Member, error)` | ~16 | Gets members for platforms |
| `GetWebhooksInternalOnly(ctx, platformID) ([]models.Webhook, error)` | ~9 | Internal webhook listing |
| `GetWebhooks(ctx, platformID) ([]models.Webhook, error)` | ~9 | Webhook listing with access check |
| `CreateWebhook(ctx, platformID, webhook) (*models.Webhook, error)` | ~31 | Creates webhook with URL validation and access key generation |
| `UpdateWebhook(ctx, platformID, webhookID, updates) (*models.Webhook, error)` | ~40 | Updates webhook config |
| `DeleteWebhook(ctx, platformID, webhookID) error` | ~8 | Deletes a webhook |
| `GetWebhookByID(ctx, platformID, webhookID) (*models.Webhook, error)` | ~8 | Webhook lookup |
| `TestConnectionWebhook(ctx, platformID, webhookID) error` | ~26 | Sends test webhook to verify connectivity |

**Private methods**: `createAPIKey()` (~13 lines), `createProjectCurrenciesMappings()` (~18 lines), `createDefaultWalletsMappings()` (~56 lines).

### 13.2 ExternalPlatformBlockchainCurrencyService

**Implementation**: `ExternalPlatformBlockchainCurrencyServiceImpl`
**File**: `external_platform_blockchain_currency_service_impl.go` (~106 lines)
**Dependencies**: `ExternalPlatformBlockchainCurrencyRepository`, `BlockchainCurrencyService`, `BlockchainService`

| Method | Lines | Description |
|--------|-------|-------------|
| `ReplaceSupportedNetworkAndCurrencies(ctx, platformID, currencies) error` | ~62 | Replaces all currency mappings for a platform (atomic) |
| `GetByExternalPlatformID(ctx, platformID) ([]models.ExternalPlatformBlockchainCurrency, error)` | ~6 | Lists currencies enabled for platform |

### 13.3 ExternalPlatformWalletBlockchainFamilyService

**Implementation**: `ExternalPlatformWalletBlockchainFamilyServiceImpl`
**File**: `external_platform_wallet_blockchain_family_service_impl.go` (~202 lines)
**Dependencies**: `ExternalPlatformWalletBlockchainFamilyRepository`, `WalletService`, `BlockchainFamilyService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateProjectWalletMappings(ctx, platformID, mappings) error` | ~45 | Creates wallet-to-platform mappings |
| `CreateOrReplaceProjectWalletMappings(ctx, platformID, mappings) error` | ~56 | Atomically replaces all mappings |
| `ListMappings(ctx, platformID) ([]models.ExternalPlatformWalletBlockchainFamily, error)` | ~15 | Lists mappings for a platform |
| `ListMappingsWithReferenceID(ctx, referenceID) ([]models.ExternalPlatformWalletBlockchainFamily, error)` | ~38 | Lists mappings by reference |
| `GetByProjectAndBlockchainFamily(ctx, platformID, family) (*models.ExternalPlatformWalletBlockchainFamily, error)` | ~4 | Specific mapping lookup |
| `GetExternalPlatformWallets(ctx, platformID) ([]models.Wallet, error)` | ~1 | Gets wallets for a platform |

### 13.4 APIKeyService

**Implementation**: `APIKeyServiceImpl`
**File**: `api_key_service_impl.go` (~194 lines)
**Dependencies**: `APIKeyRepository`, `AuthService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, request) (*models.APIKey, error)` | ~42 | Creates API key with random generation and hashing |
| `Update(ctx, apiKeyID, request) (*models.APIKey, error)` | ~88 | Updates API key properties |
| `Activate(ctx, apiKeyID) error` | ~67 | Activates an API key |
| `Deactivate(ctx, apiKeyID) error` | ~49 | Deactivates an API key |
| `DeactivateByMemberID(ctx, memberID) error` | ~15 | Deactivates all keys for a member |
| `GetByAPIKey(ctx, key) (*models.APIKey, error)` | ~4 | API key lookup |
| `GetByID(ctx, apiKeyID) (*models.APIKey, error)` | ~5 | Lookup by ID |
| `GetByExternalPlatformId(ctx, platformID) ([]models.APIKey, error)` | ~4 | Lists keys for a platform |
| `GetExternalPlatformApiKeys(ctx, platformID) ([]models.APIKey, error)` | ~6 | Lists with details |

**Setter**: `SetAuthService()`.

---

## 14. Webhook Services

### 14.1 WebhookService

**Implementation**: `WebhookServiceImpl`
**File**: `webhook_service_impl.go`
**Dependencies**: `WebhookRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `ProcessPayoutWebhook(ctx, withdrawal) error` | ~20 | Sends payout status webhook to merchant |
| `UpdatePayoutWebhookReceived(ctx, withdrawal) error` | ~10 | Updates webhook status to received |
| `UpdatePayoutWebhookFailed(ctx, withdrawal, reason) error` | ~10 | Updates webhook status to failed |

### 14.2 WebhookDeliveryService

**Implementation**: `WebhookDeliveryServiceImpl`
**File**: `webhook_delivery_service_impl.go`
**Dependencies**: `WebhookDeliveryLogRepository`, `ExternalPlatformService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetWebhookDeliveries(ctx, params) ([]models.WebhookDeliveryLog, error)` | ~4 | Lists delivery logs |
| `GetPendingApprovalDeliveries(ctx) ([]models.WebhookDeliveryLog, error)` | ~4 | Gets deliveries pending manual approval |
| `MarkDeliveryAsSent(ctx, deliveryID) error` | ~4 | Marks delivery as sent |
| `UpdateDeliveryAttemptWithRetry(ctx, deliveryID, responseCode, error) error` | ~10 | Updates attempt count, schedules retry |
| `GetRetryIntervals() []time.Duration` | ~4 | Returns configured retry intervals |
| `SendWebhookForMissedDeposit(ctx, paymentRequestID) error` | ~20 | Sends webhook for missed deposit recovery |
| `DiscardWebhookForMissedDeposit(ctx, paymentRequestID) error` | ~15 | Discards pending webhook |

**Private method**: `getRetryIntervals()` -- exponential backoff schedule.

---

## 15. Referral & Rewards Services

### 15.1 ReferralService

**Implementation**: `ReferralServiceImpl`
**File**: `referral_service_impl.go`
**Dependencies**: `go-referral` library (external), `MemberService`, `PaymentService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateCampaign(ctx, request) (*models.Campaign, error)` | ~15 | Creates a referral campaign |
| `UpdateCampaign(ctx, campaignID, updates) error` | ~10 | Updates campaign config |
| `UpdateCampaignStatus(ctx, campaignID, status) error` | ~10 | Activates/deactivates campaign |
| `GetAllCampaigns(ctx, params) ([]models.Campaign, int64, error)` | ~10 | Lists campaigns |
| `GetCampaignsCount(ctx) (int64, error)` | ~4 | Total campaign count |
| `CreateEvent(ctx, request) (*models.Event, error)` | ~10 | Creates a referral event type |
| `UpdateEvent(ctx, eventID, updates) error` | ~10 | Updates event config |
| `GetAllEvents(ctx, params) ([]models.Event, error)` | ~4 | Lists event types |
| `CreateOrUpdateReferrer(ctx, request) (*models.Member, error)` | ~15 | Creates or updates a referrer/promoter |
| `CreateReferee(ctx, request) (*models.Member, error)` | ~10 | Creates a referee (referred user) |
| `GetReferrers(ctx, params) ([]models.Member, int64, error)` | ~10 | Lists referrers |
| `GetReferrersFromMerchant(ctx, params) ([]models.Member, int64, error)` | ~10 | Merchant-filtered referrers |
| `GetReferrersWithStats(ctx, params) ([]ReferrerStats, error)` | ~15 | Referrers with reward statistics |
| `UpdateReferrer(ctx, referrerID, updates) error` | ~10 | Updates referrer |
| `UpdateReferrerStatus(ctx, referrerID, status) error` | ~10 | Activates/deactivates referrer |
| `CreateEventLog(ctx, request) (*models.EventLog, error)` | ~10 | Logs a referral event occurrence |
| `GetEventLogs(ctx, params) ([]models.EventLog, error)` | ~4 | Lists event logs |
| `GetCampaignEventLogs(ctx, params) ([]models.CampaignEventLog, error)` | ~4 | Lists campaign event logs |
| `ProcessPendingEvents(ctx) error` | ~10 | Processes pending event logs, calculates rewards |
| `GetRewards(ctx, params) ([]models.Reward, int64, error)` | ~10 | Lists rewards |
| `GetRewardsValue(ctx, params) (decimal.Decimal, error)` | ~10 | Total reward value |
| `GetTotalRewards(ctx, params) (int64, error)` | ~4 | Total reward count |
| `GetTotalRewardsForCampaign(ctx, campaignID) (int64, error)` | ~4 | Rewards per campaign |
| `GetTotalReferees(ctx, params) (int64, error)` | ~4 | Total referee count |
| `GetRefereeCount(ctx) (int64, error)` | ~4 | New referee count |
| `GetReferrerCount(ctx) (int64, error)` | ~4 | New referrer count |
| `GetRewardStats(ctx) (*RewardStats, error)` | ~10 | Aggregated reward statistics |
| `GetWithdrawals(ctx, params) ([]models.Withdrawal, error)` | ~10 | Referral reward withdrawals |
| `GetReferralPayouts(ctx, params) ([]ReferralPayout, error)` | ~10 | Referral payout history |

### 15.2 AnalyticsReferralService

**Implementation**: `AnalyticsReferralServiceImpl`
**File**: `analytics_referral_service_impl.go` (~541 lines)
**Dependencies**: `ReferralService`, `PaymentService`, `ExternalPlatformService`, `MemberService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetPromotersList(ctx, params) ([]PromoterInfo, error)` | ~97 | Detailed promoter list with stats |
| `GetTotalRevenue(ctx, params) (decimal.Decimal, error)` | ~22 | Total revenue from referred users |
| `GetPromotersCount(ctx, params) (int64, error)` | ~52 | Filtered promoter count |
| `GetRefereeCount(ctx, params) (int64, error)` | ~44 | Filtered referee count |
| `GetCampaignCount(ctx, params) (int64, error)` | ~42 | Filtered campaign count |
| `GetRewardValue(ctx, params) (decimal.Decimal, error)` | ~42 | Total reward value filtered |
| `GetRewardClaimed(ctx, params) (decimal.Decimal, error)` | ~51 | Total claimed rewards |
| `GetRewardInfo(ctx, params) (*RewardInfo, error)` | ~53 | Detailed reward breakdown |
| `GetRewardStats(ctx, params) (*RewardStats, error)` | ~70 | Comprehensive reward statistics |

**Private method**: `filterExternalPlatforms()` (~21 lines) -- filters by platform access permissions.

### 15.3 AccountRewardService

**Implementation**: `AccountRewardServiceImpl`
**File**: `account_reward_service_impl.go` (~65 lines)
**Dependencies**: `AccountRewardRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetByMemberID(ctx, memberID) ([]models.AccountReward, error)` | ~5 | Gets reward accounts for a member |
| `GetByMemberIDAndCurrencyID(ctx, memberID, currencyID) (*models.AccountReward, error)` | ~5 | Gets specific reward account |
| `UnclaimedRewardAmount(ctx, memberID) (decimal.Decimal, error)` | ~19 | Calculates total unclaimed reward amount |
| `GetTotalLockedAndBalance(ctx) ([]RewardSummary, error)` | ~1 | Aggregated locked/balance totals |

### 15.4 RewardAccountingService

**Implementation**: `RewardAccountingServiceImpl`
**File**: `reward_accounting_service_impl.go`
**Dependencies**: `RewardAccountingRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `ProcessRewards(ctx) error` | ~15 | Processes pending rewards into accounting entries (double-entry) |
| `ProcessFailedRewards(ctx) error` | ~10 | Retries failed reward processing |

---

## 16. Analytics Services

### 16.1 AnalyticsService

**Implementation**: `AnalyticsServiceImpl`
**File**: `analytics_service_impl.go` (~112 lines)
**Dependencies**: `AnalyticsRepository`, `WebsocketTokenService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetAnalyticsUserGroups(ctx, memberID) ([]models.AnalyticsUserGroup, error)` | ~4 | Gets analytics groups configured for a user |
| `FetchData(ctx, groupID, filters) (*AnalyticsData, error)` | ~24 | Fetches analytics data using configured query templates |
| `SocketUpdate(ctx, data) error` | ~37 | Pushes analytics update via WebSocket |

### 16.2 ActivityLogService

**Implementation**: `ActivityLogServiceImpl`
**File**: `activity_log_service_impl.go` (~241 lines)
**Dependencies**: `activity-log` library (external), `ExternalPlatformService`, `MemberService`, `ConfigurationService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetActivityLogs(ctx, params) ([]ActivityLog, int64, error)` | ~54 | Paginated activity log query with member/project enrichment |
| `GetActivityLogEventCategories(ctx) ([]string, error)` | ~18 | Lists distinct event categories |
| `TrackServiceOperation(ctx, operation) error` | ~35 | Tracks a service-level operation (non-HTTP) |

**Private provider methods**: `(*activityLogProvider).ResolveAccess()` (~34 lines), `GetInt()`, `GetMembersByIDs()`, `GetProjectsByIDs()`.

---

## 17. Infrastructure Services

### 17.1 ConfigurationService

**Implementation**: `ConfigurationServiceImpl`
**File**: `configuration_service_impl.go` (~164 lines)
**Dependencies**: `ConfigurationRepository`, `BlockchainService`, `BlockchainCurrencyService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateConfiguration(ctx, config) (*models.Configuration, error)` | ~4 | Creates a configuration entry |
| `BulkCreateConfigurations(ctx, configs) error` | ~4 | Batch creates configs |
| `GetConfigurationByKey(ctx, key) (*models.Configuration, error)` | ~4 | Config lookup by key |
| `UpdateConfiguration(ctx, key, value) error` | ~4 | Updates config value |
| `BulkUpsert(ctx, configs) error` | ~24 | Batch upsert configurations |
| `GetConfigurationsWithPrefix(ctx, prefix) ([]models.Configuration, error)` | ~4 | Gets configs by key prefix |
| `GetDefaultConfigs(ctx) ([]ConfigGroup, error)` | ~69 | Returns grouped default configs with current values |
| `DeleteConfigurationsWithPrefix(ctx, prefix) error` | ~5 | Deletes configs by prefix |

**Setters**: `SetBlockchainService()`, `SetBlockchainCurrencyService()`.

### 17.2 CurrencyService

**Implementation**: `CurrencyServiceImpl`
**File**: `currency_service_impl.go` (~56 lines)
**Dependencies**: `CurrencyRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, currency) (*models.Currency, error)` | ~4 | Creates a currency record |
| `GetByCode(ctx, code) (*models.Currency, error)` | ~4 | Currency lookup by code |
| `GetAllCurrencies(ctx) ([]models.Currency, error)` | ~4 | Lists all currencies |
| `GetCurrenciesForReferral(ctx) ([]models.Currency, error)` | ~11 | Gets currencies available for referral rewards |

### 17.3 SystemService

**Implementation**: `SystemServiceImpl`
**File**: `system_service_impl.go`
**Dependencies**: `ConfigurationService`

| Method | Lines | Description |
|--------|-------|-------------|
| `SystemInfo(ctx) (*SystemInfo, error)` | ~20 | Returns system info (version, uptime, config) |
| `GetAllWorkers(ctx) ([]WorkerInfo, error)` | ~10 | Lists all background worker processes |
| `GetAllWorkerStatuses(ctx) ([]WorkerStatus, error)` | ~10 | Gets status of all workers |
| `StartWorker(ctx, workerName) error` | ~10 | Starts a supervisord worker |
| `StopWorker(ctx, workerName) error` | ~10 | Stops a supervisord worker |
| `RestartWorker(ctx, workerName) error` | ~10 | Restarts a single worker |
| `RestartAllWorkers(ctx) error` | ~20 | Restarts all workers |
| `RestartCore(ctx) error` | ~10 | Restarts the core API process |
| `ValidateSystemQrCode(ctx, code) (bool, error)` | ~10 | Validates admin QR code for system access |

### 17.4 NonceService

**Implementation**: `NonceServiceImpl`
**File**: `nonce_service_impl.go` (~53 lines)
**Dependencies**: `WalletService`, blockchain clients

| Method | Lines | Description |
|--------|-------|-------------|
| `GenerateNonceWithKeysForWallet(ctx, walletID) (*NonceResponse, error)` | ~22 | Generates blockchain nonce for a wallet's address, used for transaction signing |

### 17.5 GenericDataStoreService

**Implementation**: `GenericDataStoreServiceImpl`
**File**: `generic_data_store_service_impl.go` (~43 lines)
**Dependencies**: `GenericDataStoreRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, dataStore) (*models.GenericDataStore, error)` | ~4 | Creates a generic data store entry |
| `GetByBlockchainCurrencyCodeAndMultipleData(ctx, type, data) ([]models.GenericDataStore, error)` | ~4 | Queries data store by type and data |
| `Update(ctx, dataStore) error` | ~4 | Updates data store entry |
| `BulkCreate(ctx, entries) error` | ~1 | Batch creates entries |

### 17.6 EventEmitterService

**Implementation**: `EventEmitterServiceImpl`
**File**: `event_emitter_service_impl.go` (~923 lines)
**Dependencies**: `event-emitter` library (external), `ExternalPlatformService`, `SMTPConfigService`, `ConfigurationService`

| Method | Lines | Description |
|--------|-------|-------------|
| `EmitHotWalletBalanceLow(ctx, wallet, balance) error` | ~54 | Emits low balance alert for hot wallet |
| `EmitResetPassword(ctx, member, resetToken) error` | ~34 | Emits password reset email event |
| `EmitPaymentReceivedWebhookSent(ctx, paymentRequest) error` | ~66 | Emits event after webhook delivery |
| `EmitPaymentReceived(ctx, paymentRequest) error` | ~67 | Emits payment received notification |
| `EmitCancelledEventForOldPayments(ctx, paymentRequests) error` | ~59 | Emits cancellation events for expired payments |
| `EmitEventForNewPayment(ctx, paymentRequest) error` | ~78 | Emits event for new payment request |
| `EmitEventForTestEmail(ctx, email) error` | ~29 | Sends test email |
| `EmitEventForOTP(ctx, member, otp, purpose) error` | ~62 | Sends OTP via email |
| `QueryEvents(ctx, params) ([]Event, error)` | ~5 | Queries emitted events |
| `EmitPayoutCreated(ctx, withdrawal) error` | ~105 | Emits payout created notification |
| `EmitPayoutUpdated(ctx, withdrawal) error` | ~103 | Emits payout status update notification |
| `EmitPayoutWebhook(ctx, withdrawal) error` | ~46 | Emits payout webhook to merchant |

**Private methods**: `populateExternalPlatformDetails()` (~29 lines), `populateFromAndReplyToFromSMTPConfig()` (~17 lines).

### 17.7 ConsumerService

**Implementation**: `ConsumerServiceImpl`
**File**: `consumer_service_impl.go` (~137 lines)
**Dependencies**: `event-consumer` library (external), `ConfigurationService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Run(ctx) error` | ~9 | Starts the email consumer service |

**Private methods**: `initializeConsumerServiceIfNeeded()` (~79 lines) -- lazy initialization with SMTP config lookup, `isInitialized()` (~3 lines).

### 17.8 PublicAPIService

**Implementation**: `PublicAPIServiceImpl`
**File**: `public_api_service_impl.go`
**Dependencies**: `CurrencyService`

| Method | Lines | Description |
|--------|-------|-------------|
| `Ticker(ctx) ([]TickerData, error)` | ~10 | Returns price ticker data for all currencies |

### 17.9 WebsocketTokenService

**Implementation**: `WebsocketTokenServiceImpl`
**File**: `websocket_token_service_impl.go`
**Dependencies**: `WebsocketTokenRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, memberID) (*models.WebSocketToken, error)` | ~10 | Creates a websocket auth token |
| `Get(ctx, token) (*models.WebSocketToken, error)` | ~5 | Validates and returns token |

### 17.10 RecipientService

**Implementation**: `RecipientServiceImpl`
**File**: `recipient_service_impl.go`
**Dependencies**: `RecipientRepository`, `BlockchainService`

| Method | Lines | Description |
|--------|-------|-------------|
| `CreateRecipient(ctx, request) (*models.Recipient, error)` | ~15 | Creates a withdrawal recipient with address validation |
| `GetRecipients(ctx, params) ([]models.Recipient, error)` | ~4 | Lists recipients for a member |
| `UpdateRecipient(ctx, recipientID, updates) (*models.Recipient, error)` | ~10 | Updates recipient details |
| `DeleteRecipient(ctx, recipientID) error` | ~10 | Soft-deletes a recipient (requires OTP) |
| `ValidateOTPSource(ctx, entityID, purpose) error` | ~10 | OTP validation (implements OTPValidatable) |
| `OnOTPValidated(ctx, entityID, purpose, metadata) error` | ~15 | Completes action after OTP validation |

### 17.11 OnramperPaymentsService

**Implementation**: `OnramperPaymentsServiceImpl`
**File**: `onramper_payments_service_impl.go`
**Dependencies**: `OnramperPaymentsRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetOnramperPayments(ctx, params) ([]OnramperPayment, error)` | ~10 | Lists card-to-crypto onramp payments |
| `GetOnramperPaymentsMetrics(ctx, params) (*OnramperMetrics, error)` | ~10 | Aggregated onramp metrics |

### 17.12 PaymentsAppService

**Implementation**: `PaymentsAppServiceImpl`
**File**: `payments_app_service_impl.go`
**Dependencies**: `PaymentsAppRepository`, `ExternalPlatformService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetPaymentsApps(ctx) ([]models.PaymentsApp, error)` | ~4 | Lists payment apps |
| `GetPaymentsAppByProjectID(ctx, projectID) (*models.PaymentsApp, error)` | ~4 | Gets payment app for a project |
| `CreatePaymentsApp(ctx, request) (*models.PaymentsApp, error)` | ~15 | Creates a payment app with sponsorship config |
| `UpdatePaymentsApp(ctx, appID, updates) (*models.PaymentsApp, error)` | ~15 | Updates payment app config |

**Private method**: `validateProjectAccess()`.

### 17.13 ContractAddressService

**Implementation**: `ContractAddressServiceImpl`
**File**: `contract_address_service_impl.go` (~598 lines -- complex)
**Dependencies**: `ContractAddressRepository`, `BlockchainContractService`, `BlockchainService`, `WalletService`, blockchain clients

| Method | Lines | Description |
|--------|-------|-------------|
| `Create(ctx, request) (*models.ContractAddress, error)` | ~493 | Deploys smart contract, verifies on-chain, extracts admin/fund-sweeper addresses. Supports ETH and TRX chains |
| `Update(ctx, contractAddressID, updates) (*models.ContractAddress, error)` | ~91 | Updates contract address with on-chain verification |
| `GetByBlockchainCodeAndContractType(ctx, bcCode, contractType) ([]models.ContractAddress, error)` | ~4 | Lists contract addresses |
| `GetAllByContractType(ctx, contractType) ([]models.ContractAddress, error)` | ~4 | Lists by contract type |
| `GetById(ctx, contractAddressID) (*models.ContractAddress, error)` | ~1 | Contract address lookup |

**Private method**: `getRPCURLForBlockchain()` (~12 lines).

### 17.14 EntrypointSCAddressService

**Implementation**: `EntrypointSCAddressServiceImpl`
**File**: `entrypoint_sc_addresses_service_impl.go` (~35 lines)
**Dependencies**: `EntrypointSCAddressRepository`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetEntrypointSCAddressByAddress(ctx, address) (*models.EntrypointSCAddress, error)` | ~2 | Looks up an entrypoint smart contract address |

### 17.15 SMTPConfigService

**Implementation**: `SMTPConfigServiceImpl`
**File**: `smtp_config_service_impl.go`
**Dependencies**: `ConfigurationService`, `EventEmitterService`

| Method | Lines | Description |
|--------|-------|-------------|
| `GetSMTPConfig(ctx) (*SMTPConfig, error)` | ~15 | Reads SMTP config from configuration store |
| `UpsertSMTPConfig(ctx, config) error` | ~20 | Saves SMTP config, spawns goroutine to test connection |
| `SendTestEmail(ctx, email) error` | ~15 | Sends test email via configured SMTP |

---

## 18. Background Worker Jobs

These are long-running goroutines that process tasks in the background:

### 18.1 AccountProcessorJob

**File**: `account_processor_job.go` (~625 lines)
**Dependencies**: All sweep/withdrawal/reward services

The central background job that runs 9 concurrent goroutines:

| Goroutine | Method | Description |
|-----------|--------|-------------|
| 1 | `processETHAutoSweep()` | Auto-sweeps ETH/EVM deposits to cold storage |
| 2 | `processBitcoinSweeps()` | Creates BTC sweep transactions |
| 3 | `processERC20Sweeps()` | Processes ERC20 token sweeps |
| 4 | `createSweepTransactionPayload()` | Signs sweep transaction payloads |
| 5 | `processRewards()` | Processes referral rewards accounting |
| 6 | `processFailedRewards()` | Retries failed reward processing |
| 7 | `processWithdrawals()` | Processes approved withdrawals |
| 8 | `retryStaleBTCSweepTransactions()` | Retries stale BTC sweeps |
| 9 | `processConfirmingBTCSweeps()` | Checks BTC sweep confirmations |

**Additional methods**: `createAllPendingUTXOsAndAccountForSweepTransaction()` (~54 lines), `processStaleInitiatedSweeps()` (~51 lines), `processConfirmingSweepsOnce()` (~50 lines).

### 18.2 BlockchainProcessorImpl (Block Listeners)

**File**: `blockchain_processor_impl.go` (~2565 lines -- largest service file)
**Dependencies**: All blockchain/deposit/sweep/wallet services, blockchain clients

Created via factory: `CreateBlockchainProcessor(blockchain, services) BlockchainProcessor`

| Method | Lines | Description |
|--------|-------|-------------|
| `Start()` | ~45 | Starts the block listener |
| `Stop()` | ~8 | Stops the block listener |
| `startPolling()` | ~47 | Begins polling for new blocks |
| `polling()` | ~135 | Core polling loop with adaptive interval |
| `ProcessBlockRange(startBlock, endBlock)` | ~139 | Processes a range of blocks |
| `ProcessSingleBlock(blockNumber)` | ~49 | Processes a single block |
| `handleExtractedDeposit(deposit)` | ~407 | Main deposit handler -- identifies addresses, creates deposits, links to payments |
| `handleSweepEvent(event)` | ~75 | Processes sweep contract events |
| `handleFactoryEvent(event)` | ~106 | Processes factory contract events (SCW deployment) |
| `processConfirmingDeposits()` | ~32 | Checks confirmation status of pending deposits |
| `processConfirmingSweeps()` | ~161 | Checks confirmation of sweep transactions |
| `verifyAndUpsertDepositsForTxHash(txHash)` | ~183 | Verifies deposits on-chain and upserts |
| `buildDepositDataFromExtracted(deposit)` | ~271 | Transforms raw block data into deposit model |
| `RecoverMissedDeposits()` | ~70 | Re-scans blocks for missed deposits |
| `startWithdrawAuditProcessor()` | ~42 | Starts withdrawal audit goroutine |
| `withdrawAuditProcess()` | ~34 | Audits withdrawal transactions on-chain |
| `auditSingleWithdrawal(withdrawal)` | ~107 | Audits a single withdrawal |

**Factory methods**: `EtherAndERC20ProcessorJob()` (~39 lines), `TronAndTRC20ProcessorJob()` (~38 lines), `BitcoinProcessorJob()` (~11 lines).

### 18.3 DepositProcessorJob

**File**: `deposit_processor_job.go` (~86 lines)

| Method | Lines | Description |
|--------|-------|-------------|
| `StartDepositProcessor()` | ~26 | Polls for pending deposits, runs accounting |
| `ProcessPendingDeposits()` | ~14 | Delegates to DepositService.ProcessAllPendingDeposits |

### 18.4 WebhookProcessorJob

**File**: `webhook_processor_job.go`

| Method | Lines | Description |
|--------|-------|-------------|
| `StartWebhookProcessor()` | ~26 | Polls for pending webhook deliveries, processes them |

### 18.5 EmailProcessorJob

**File**: `email_processor_job.go` (~64 lines)

| Method | Lines | Description |
|--------|-------|-------------|
| `StartEmailProcessor()` | ~26 | Starts the SMTP email consumer service |

### 18.6 SweepApprovalProcessorJob

**File**: `sweep_approval_processor.go`

| Method | Lines | Description |
|--------|-------|-------------|
| `StartSweepApprovalListener()` | ~39 | Processes ERC20 sweep approvals |
| `addressesBalancesAccountingAfterSweep()` | ~20 | Runs accounting after sweep completion |

### 18.7 BroadcastSCWDepositWalletProcessorJob

**File**: `broadcast_scw_deposit_wallets_processor.go` (~417 lines)

| Method | Lines | Description |
|--------|-------|-------------|
| `StartBroadcastSCWDepositListener()` | ~39 | Starts SCW deployment broadcaster |
| `BroadcastSCWDepositWalletProcessorJob()` | ~184 | Main processing loop -- deploys SCW wallets on-chain |
| `ValidatePendingAndBroadcastedAddressDeployments()` | ~66 | Validates deployment transaction status |
| `ProcessAddressDeploymentsAccounting()` | ~12 | Accounting for deployment costs |

### 18.8 AccountingDuplicateDepositsJob

**File**: `accounting_duplicate_deposits_job.go` (~30 lines)

| Method | Lines | Description |
|--------|-------|-------------|
| `StartProcessing()` | ~7 | Processes duplicate deposit accounting corrections |

---

## 19. Cross-Cutting Patterns

### 19.1 Activity Logging

Activity logging is implemented as middleware (Gin middleware) + service-level tracking:

- **HTTP middleware**: `ginmiddleware.Middleware()` captures request/response bodies, IP, user agent, session ID, geo-location
- **Service tracking**: `ActivityLogServiceImpl.TrackServiceOperation()` logs non-HTTP operations
- **Enrichment**: `EnrichActivityLogUpdateRequest()` adds project IDs and response data to activity logs
- **Geo-lookup**: `GeoLookup` service caches IP-to-location lookups
- **Sensitive data**: `RedactJSONKeys()` redacts passwords, tokens, private keys from logs

### 19.2 Webhook Management

Webhooks follow a delivery-log pattern with retry:

1. **Creation**: When a payment is confirmed, `createDeliveryLogsForPaymentRequest()` creates `WebhookDeliveryLog` entries for each webhook URL
2. **Processing**: `WebhookProcessorJob` polls for pending deliveries
3. **Delivery**: HTTP POST to merchant webhook URL with HMAC signature
4. **Retry**: Exponential backoff (configurable intervals), tracked via `attempt_count` and `next_retry_at`
5. **Manual approval**: Payments can be configured to require manual webhook approval via `GetPaymentRequestsPendingWebhookApproval()`
6. **Payout webhooks**: Separate webhook flow for withdrawal status updates via `WebhookService.ProcessPayoutWebhook()`

### 19.3 Multi-Blockchain Support

Chain abstraction follows a strategy pattern:

```
BlockchainClient (interface)
  |-- BTCBlockchainClient  (Bitcoin, UTXO-based)
  |-- ETHBlockchainClient  (Ethereum, Base, Polygon -- EVM)
  |-- TRXBlockchainClient  (Tron)
```

Each client implements ~50 methods (GetBalance, GetBlock, SignTransaction, BroadcastRawTransaction, etc.).

The `blockchainProcessorImpl` uses `BlockchainClient` interface to process blocks uniformly while the service layer handles chain-specific logic:

- **BTC**: UTXO tracking, raw transaction construction, segwit support
- **EVM**: Native + ERC20 transfers, event log parsing, CREATE2 address derivation
- **TRX**: TRC20 transfers, base58 addresses, energy/bandwidth estimation

### 19.4 OTP Verification

The OTP system uses a **validator registration** pattern:

1. `OTPService` maintains a map of `purpose -> OTPValidatable`
2. Services register themselves: `otpService.RegisterValidator("withdrawal_approval", withdrawalProcessingService)`
3. When OTP is validated, `OTPService` calls `validator.OnOTPValidated()`
4. This decouples OTP mechanics from business-specific post-validation logic

Currently registered validators:
- `RecipientService` -- for recipient deletion/modification
- `WithdrawalProcessingService` -- for withdrawal approval

### 19.5 Referral Tracking

Referrals use the external `go-referral` library with these components:

- **Campaigns**: Time-bounded reward programs with budgets
- **Events**: Trigger types (signup, payment, etc.)
- **Members**: Referrers (promoters) and referees
- **Rewards**: Calculated based on event type and campaign rules
- **Processing**: `ProcessPendingEvents()` runs in `AccountProcessorJob`, calculates rewards, creates accounting entries

### 19.6 Double-Entry Accounting

All financial operations use double-entry bookkeeping with four account types:

| Account Type | Model | Usage |
|-------------|-------|-------|
| Asset | `models.Asset` | Platform assets (received deposits) |
| Liability | `models.Liability` | Obligations to merchants |
| Revenue | `models.Revenue` | Platform fees |
| Expense | `models.Expense` | Gas costs, deployment fees |

Key accounting methods in repositories:
- `ProcessPaymentRequestAccounting()` -- deposit confirmed
- `ProcessWithdrawsAccounting()` -- withdrawal completed
- `ProcessERC20Sweep()` -- sweep completed
- `ProcessAccountingForWithdrawalSent()` -- payout sent
- `ProcessDuplicateDepositsAsExpense()` -- duplicate deposit handling
- `ProcessReward()` -- referral reward accounting

---

## 20. Business Logic Concentrations

### Most Complex Methods (by line count and business logic density)

| Method | File | Lines | Description |
|--------|------|-------|-------------|
| `blockchainProcessorImpl.handleExtractedDeposit()` | blockchain_processor_impl.go | 407 | Identifies deposit addresses, checks blacklists, creates deposits, links to payments, handles UTXO vs account model |
| `blockchainProcessorImpl.buildDepositDataFromExtracted()` | blockchain_processor_impl.go | 271 | Complex transformation from raw block data to deposit model with all enrichments |
| `BroadcastSCWDepositWalletProcessorJob.BroadcastSCWDepositWalletProcessorJob()` | broadcast_scw_deposit_wallets_processor.go | 184 | Deploys SCW wallets on-chain -- builds deployment args, signs, broadcasts, tracks |
| `blockchainProcessorImpl.verifyAndUpsertDepositsForTxHash()` | blockchain_processor_impl.go | 183 | Re-verifies deposits on-chain, handles chain reorganizations |
| `blockchainProcessorImpl.processConfirmingSweeps()` | blockchain_processor_impl.go | 161 | Checks sweep confirmation status across multiple chains |
| `blockchainProcessorImpl.ProcessBlockRange()` | blockchain_processor_impl.go | 139 | Processes a range of blocks with native + token + UTXO + contract events |
| `blockchainProcessorImpl.polling()` | blockchain_processor_impl.go | 135 | Main polling loop with adaptive interval based on block production rate |
| `MissedDepositServiceImpl.CreateMissedDeposit()` | missed_deposit_service_impl.go | 114 | Verifies missed deposit on-chain, classifies transaction type, creates records |
| `blockchainProcessorImpl.auditSingleWithdrawal()` | blockchain_processor_impl.go | 107 | Audits withdrawal on-chain -- checks confirmation, amount, addresses |
| `blockchainProcessorImpl.handleFactoryEvent()` | blockchain_processor_impl.go | 106 | Processes factory contract events for SCW deployment tracking |
| `EventEmitterServiceImpl.EmitPayoutCreated()` | event_emitter_service_impl.go | 105 | Constructs and emits payout creation event with full platform details |
| `AnalyticsReferralServiceImpl.GetPromotersList()` | analytics_referral_service_impl.go | 97 | Complex promoter list with multi-source aggregation |
| `ExternalPlatformServiceImpl.Update()` | external_platform_service_impl.go | 94 | Updates platform with logo upload, config changes |
| `ContractAddressServiceImpl.Update()` | contract_address_service_impl.go | 91 | Updates contract address with on-chain verification |
| `APIKeyServiceImpl.Update()` | api_key_service_impl.go | 88 | Updates API key with status transition logic |
| `BlockchainCurrencyServiceImpl.Update()` | blockchain_currency_service_impl.go | 81 | Updates currency config with validation |
| `MemberServiceImpl.GetMember()` | member_service_impl.go | 203 | Complex member retrieval with role/permission resolution |
| `AddressServiceImpl.CreateDataStoreForSweepInitiatedAddresses()` | address_service_impl.go | 77 | Creates lock records for sweep-initiated addresses |

### AuthServiceImpl.Signup (~599 lines combined with Signin)

This is the most complex authentication flow:

1. Validates email format and uniqueness
2. Hashes password with bcrypt
3. Checks if this is the first user (root member)
4. Creates member record in transaction
5. Generates API key for platform access
6. Creates JWT access + refresh token pair
7. Stores refresh token hash
8. Returns member with tokens

### BlockchainProcessorImpl.handleExtractedDeposit (~407 lines)

The core deposit processing logic:

1. Checks if deposit address is in our address pool
2. Checks against blacklisted addresses
3. Verifies the deposit is not a duplicate
4. Determines deposit type (native, ERC20, UTXO)
5. Creates deposit record
6. Links deposit to open payment request
7. Updates account address balance
8. Creates webhook delivery logs
9. Handles confirmation tracking
10. Emits events for notifications

### ContractAddressServiceImpl.Create (~493 lines)

Smart contract deployment and verification:

1. Validates the blockchain contract exists
2. Connects to blockchain RPC node
3. For ETH: calls Ethereum contract, extracts admin/fund-sweeper
4. For TRX: connects via gRPC, calls Tron contract
5. Verifies contract bytecode matches expected
6. Extracts fund collector address
7. Creates contract address record
8. Associates with blockchain contract

---

*This document was generated from reverse-engineered PayRam source analysis. All service names, methods, and line counts are derived from binary analysis of the Go application. When implementing Payminto, use this as the authoritative reference for service layer architecture.*
