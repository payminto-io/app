# Payminto API Specification

**Version:** 1.0
**Date:** April 7, 2026
**Status:** Implementation-ready specification derived from PayRam reverse-engineering
**Source Confidence:** Handler names, middleware chains, and model structures are **confirmed from source**. HTTP methods, URL paths, and request/response bodies are **inferred** from handler names, route setup function line counts, service method signatures, and Go model definitions unless otherwise noted.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Authentication](#2-authentication)
3. [Middleware Stack](#3-middleware-stack)
4. [API Endpoints by Domain](#4-api-endpoints-by-domain)
   - 4.1 [Authentication (AuthHandler)](#41-authentication-authhandler)
   - 4.2 [Members / Users (MemberHandler)](#42-members--users-memberhandler)
   - 4.3 [OTP (OTPHandler)](#43-otp-otphandler)
   - 4.4 [API Keys (APIKeyHandler)](#44-api-keys-apikeyhandler)
   - 4.5 [Roles and Permissions (RoleHandler)](#45-roles-and-permissions-rolehandler)
   - 4.6 [Payments (PaymentHandler)](#46-payments-paymenthandler)
   - 4.7 [Payment Channels (PaymentChannelHandler)](#47-payment-channels-paymentchannelhandler)
   - 4.8 [Payment Channel Projects (PaymentChannelProjectHandler)](#48-payment-channel-projects-paymentchannelprojecthandler)
   - 4.9 [Deposits (DepositHandler)](#49-deposits-deposithandler)
   - 4.10 [Deposit Addresses (DepositAddressHandler)](#410-deposit-addresses-depositaddresshandler)
   - 4.11 [Addresses (AddressHandler)](#411-addresses-addresshandler)
   - 4.12 [Address Pools (AddressPoolHandler)](#412-address-pools-addresspoolhandler)
   - 4.13 [Wallets (WalletHandler)](#413-wallets-wallethandler)
   - 4.14 [Blockchain Management (BlockchainHandler)](#414-blockchain-management-blockchainhandler)
   - 4.15 [Blockchain Currencies (BlockchainCurrencyHandler)](#415-blockchain-currencies-blockchaincurrencyhandler)
   - 4.16 [Blockchain Families (BlockchainFamilyHandler)](#416-blockchain-families-blockchainfamilyhandler)
   - 4.17 [Blockchain Contracts (BlockchainContractHandler)](#417-blockchain-contracts-blockchaincontracthandler)
   - 4.18 [Contract Addresses (ContractAddressHandler)](#418-contract-addresses-contractaddresshandler)
   - 4.19 [Currencies (CurrencyHandler)](#419-currencies-currencyhandler)
   - 4.20 [Sweeps (SweepHandler)](#420-sweeps-sweephandler)
   - 4.21 [Sweep Transactions (SweepTransactionHandler)](#421-sweep-transactions-sweeptransactionhandler)
   - 4.22 [Sweep UTXOs (UtxoHandler)](#422-sweep-utxos-utxohandler)
   - 4.23 [Withdrawals (WithdrawalHandler)](#423-withdrawals-withdrawalhandler)
   - 4.24 [External Platforms (ExternalPlatformHandler)](#424-external-platforms-externalplatformhandler)
   - 4.25 [External Platform Blockchain Currencies (ExternalPlatformBlockchainCurrencyHandler)](#425-external-platform-blockchain-currencies-externalplatformblockchaincurrencyhandler)
   - 4.26 [External Platform Wallet Mappings (ExternalPlatformWalletBlockchainFamilyHandler)](#426-external-platform-wallet-mappings-externalplatformwalletblockchainfamilyhandler)
   - 4.27 [Analytics (AnalyticsHandler)](#427-analytics-analyticshandler)
   - 4.28 [Analytics Referral (AnalyticsReferralHandler)](#428-analytics-referral-analyticsreferralhandler)
   - 4.29 [Activity Logs (ActivityLogHandler)](#429-activity-logs-activityloghandler)
   - 4.30 [Referral System (ReferralHandler)](#430-referral-system-referralhandler)
   - 4.31 [Missed Deposits (MissedDepositHandler)](#431-missed-deposits-misseddeposithandler)
   - 4.32 [Recipients (RecipientHandler)](#432-recipients-recipienthandler)
   - 4.33 [Configuration (ConfigurationHandler)](#433-configuration-configurationhandler)
   - 4.34 [SMTP Configuration (SMTPConfigHandler)](#434-smtp-configuration-smtpconfighandler)
   - 4.35 [Onramper Payments (OnramperPaymentsHandler)](#435-onramper-payments-onramperpaymenentshandler)
   - 4.36 [Payments App (PaymentsAppHandler)](#436-payments-app-paymentsapphandler)
   - 4.37 [System Admin (SystemHandler)](#437-system-admin-systemhandler)
   - 4.38 [WebSocket (WebsocketHandler, WebsocketTokenHandler)](#438-websocket-websockethandler-websockettokenhandler)
   - 4.39 [Nonce (NonceHandler)](#439-nonce-noncehandler)
   - 4.40 [Public API (PublicAPIHandler)](#440-public-api-publicapihandler)
   - 4.41 [Address Contract Signatures (AddressContractSignatureHandler)](#441-address-contract-signatures-addresscontractsignaturehandler)
   - 4.42 [Account Rewards (AccountRewardHandler)](#442-account-rewards-accountrewardhandler)
   - 4.43 [Secrets Vault Activities (SecretsVaultActivityHandler)](#443-secrets-vault-activities-secretsvaultactivityhandler)
   - 4.44 [Payment Links (v2)](#444-payment-links-v2)
5. [Webhook Events](#5-webhook-events)
6. [Error Response Format](#6-error-response-format)
7. [Rate Limiting](#7-rate-limiting)
8. [Pagination](#8-pagination)

---

## 1. Overview

### Base URL

```
https://{merchant-server}:8443/api/v1
```

All API endpoints are prefixed with `/api/v1`. The Go backend (Gin framework) runs on port 8080 internally, exposed via Nginx reverse proxy on port 8443 (HTTPS).

### Versioning

API versioning is path-based. All current endpoints use `/api/v1`. Future breaking changes will introduce `/api/v2`.

### Content Type

All requests and responses use `application/json` unless otherwise noted (e.g., CSV downloads).

### Authentication Methods

| Method | Header | Usage |
|--------|--------|-------|
| **API Key** | `X-API-Key: <key>` | External/merchant API access |
| **JWT Bearer** | `Authorization: Bearer <access_token>` | Dashboard/admin sessions |
| **Internal** | Localhost-only restriction | Worker-to-API internal calls |

### Rate Limiting

Rate limiting is enforced via Redis. Default limits are applied per API key or per authenticated session. See [Section 7](#7-rate-limiting) for details.

### Request ID

Every request is assigned a unique request ID for tracing. It is returned in the response headers as `X-Request-ID`.

---

## 2. Authentication

### API Key Authentication

API keys are SHA-256 hashed before storage. The raw key is returned only once at creation time.

**Flow:**
1. Admin creates an API key via the dashboard or API
2. Raw key is returned in the creation response (store securely)
3. Subsequent API calls include the key in `X-API-Key` header
4. Server hashes the provided key with SHA-256 and looks up the hash in the `api_keys` table
5. Key must be active (`status = active`) and associated with an active member/platform

**API Key Model (confirmed from source):**
```
APIKey {
  id:                  uint (auto-increment)
  member_id:           uint
  external_platform_id: *uint
  api_key:             string (SHA-256 hash stored)
  status:              string ("active" | "inactive")
  role_id:             *uint
  member:              *Member
  external_platform:   *ExternalPlatform
  role:                *Role
  created_at:          timestamp
  updated_at:          timestamp
  deleted_at:          *timestamp (soft delete)
}
```

### JWT Authentication

JWT tokens are used for dashboard session management. The system issues an access token and a refresh token.

**Flow:**
1. User signs in via `POST /api/v1/auth/signin` with email and password
2. Server returns `access_token` (short-lived) and `refresh_token` (long-lived)
3. Access token is included in `Authorization: Bearer <token>` header
4. When access token expires, use `POST /api/v1/auth/refresh` with the refresh token
5. Refresh tokens are stored in the `auth_refresh_tokens` table with expiry and revocation tracking

**JWT Claims (confirmed from source):**
```
Claims {
  member_id:   uint
  email:       string
  roles:       []string
  issuer:      string
  subject:     string
  audience:    []string
  expires_at:  timestamp
  issued_at:   timestamp
  not_before:  timestamp
}
```

**Token Service Methods (confirmed):**
- `GenerateTokenPair` -- generates access + refresh tokens
- `ValidateAccessToken` -- validates and parses access token
- `ValidateRefreshToken` -- validates refresh token
- `GetRefreshExpiration` -- returns refresh token TTL

**AuthRefreshToken Model (confirmed):**
```
AuthRefreshToken {
  id:          uint
  member_id:   uint
  token:       string
  expires_at:  timestamp
  last_used_at: *timestamp
  revoked_at:  *timestamp
}
```

---

## 3. Middleware Stack

The middleware chain is applied in order for each request. Different route groups apply different middleware combinations.

### 3.1 AdminMiddleware

**Source:** `internal/api/middleware/admin_middleware.go` (lines 30-137, confirmed)
**Applied to:** All authenticated dashboard/admin routes

**Responsibilities:**
- Validates JWT access token from `Authorization` header
- Extracts member ID and roles from JWT claims
- Checks member is active and not soft-deleted
- Validates required permissions via `hasRequiredPermissions` (lines 141-176, confirmed)
- Logs unauthorized access attempts via `LogUnauthorizedAccess` (lines 176-206, confirmed)
- Enriches activity log with response data via `enrichActivityWithResponseData` (lines 228-461, confirmed)
- Detects sensitive activity log paths via `isSensitiveActivityLogPath` (lines 206-228, confirmed)
- Sets `member_id`, `roles`, and `permissions` in Gin context for downstream handlers

### 3.2 ExternalPlatformMiddleware

**Source:** `internal/api/middleware/external_platform_middleware.go` (lines 16-40, confirmed)
**Applied to:** Routes that operate within the context of an external platform (project)

**Responsibilities:**
- Extracts external platform context from the request (via path param, query param, or JWT claim)
- Calls `handleAllPlatforms` (lines 44-72) for requests spanning all platforms
- Calls `handleSinglePlatform` (lines 72-101) for platform-specific requests
- Validates the member has access to the specified platform
- Calls `handleUnauthorisedAccess` (lines 101-106) on failure
- Sets `external_platform_id` and `external_platform` in Gin context

### 3.3 PaymentRequestMiddleware

**Source:** `internal/api/middleware/payment_request_middleware.go` (lines 12-32, confirmed)
**Applied to:** Routes that operate on a specific payment request (via reference_id)

**Responsibilities:**
- Extracts `reference_id` from URL path or query parameters
- Looks up the PaymentRequest by reference ID
- Validates the payment request exists and belongs to the authenticated member/platform
- Sets `payment_request` and `payment_request_id` in Gin context

### 3.4 ReferralServiceMiddleware

**Source:** `internal/api/middleware/referral_service_middleware.go` (lines 15-66, confirmed)
**Applied to:** Referral system routes and API key routes related to referrals

**Responsibilities:**
- Initializes the referral service context
- Validates that the referral system is enabled for the current platform
- Loads referral campaign and configuration data
- Sets referral service instance in Gin context

### 3.5 LowercaseMiddleware

**Source:** `internal/api/middleware/lowercase_middleware.go` (lines 15-61, confirmed)
**Applied to:** Auth routes (signup), member routes (create, update)

**Responsibilities:**
- Normalizes email addresses and identifiers to lowercase before processing
- Ensures case-insensitive email matching throughout the system
- Modifies request body in-place before passing to handler

### 3.6 WebsocketMiddleware

**Source:** `internal/api/middleware/websocket_middleware.go` (lines 13-53, confirmed)
**Applied to:** WebSocket connection routes

**Responsibilities:**
- Validates WebSocket upgrade request
- Authenticates WebSocket connection via token (from query param or initial message)
- Validates token has not expired
- Establishes WebSocket connection and registers client with WebSocketManager

### 3.7 AllowOnlyLocalhostMiddleware (InternalMiddleware)

**Source:** `internal/api/middleware/internal_middleware.go` (lines 13-26 and 30-38, confirmed)
**Applied to:** Internal-only routes (WebSocket trigger, system routes)

**Responsibilities:**
- Checks if the request originates from localhost (127.0.0.1, ::1, or localhost)
- Rejects requests from non-local sources with 403 Forbidden
- Uses `isLocalhost` helper function (lines 30-38)

### 3.8 ErrorHandlingMiddleware

**Source:** `internal/api/errors/errors_handler.go` (lines 60-91, confirmed)
**Applied to:** Global middleware on all routes

**Responsibilities:**
- Catches panics and unhandled errors from handlers
- Wraps errors in the standard `PayramError` response format (lines 23-60, confirmed)
- Returns appropriate HTTP status codes
- Logs error details for debugging

---

## 4. API Endpoints by Domain

### Route Setup Pattern

Each domain has a corresponding route setup function in `internal/api/` (e.g., `SetupAuthRoutes`, `SetupPaymentsRoutes`). These functions receive the Gin router group and handler instances, and register routes with their middleware chains.

---

### 4.1 Authentication (AuthHandler)

**Route Setup:** `SetupAuthRoutes` (lines 13-55, 42 lines, confirmed)
**Handler:** `AuthHandler`
**Service:** `AuthServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/auth/signup` | `AuthHandler.Signup` | LowercaseMiddleware | None | Register new account (root member) |
| POST | `/api/v1/auth/signin` | `AuthHandler.Signin` | None | None | Sign in with email/password, returns JWT pair |
| POST | `/api/v1/auth/refresh` | `AuthHandler.RefreshToken` | None | Refresh Token | Exchange refresh token for new token pair |
| POST | `/api/v1/auth/authenticate` | `AuthHandler.Authenticate` | AdminMiddleware | JWT | Validate current token and return member info |
| POST | `/api/v1/auth/logout` | `AuthHandler.Logout` | AdminMiddleware | JWT | Revoke current refresh token |
| POST | `/api/v1/auth/logout-all` | `AuthHandler.LogoutAll` | AdminMiddleware | JWT | Revoke all refresh tokens for member |

**POST /api/v1/auth/signup**

*Request Body (inferred):*
```json
{
  "email": "admin@example.com",
  "password": "SecureP@ss123",
  "name": "Admin User"
}
```

*Response (inferred):*
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "member": {
    "id": 1,
    "email": "admin@example.com",
    "name": "Admin User",
    "status": "active"
  }
}
```

*Notes:* The `Signup` service method is very large (lines 68-667 in `auth_service_impl.go`, confirmed) indicating it handles initial system setup, seed data creation, default wallet generation, and blockchain configuration in addition to member creation.

**POST /api/v1/auth/signin**

*Request Body (confirmed from PRODUCT_SPEC):*
```json
{
  "email": "admin@example.com",
  "password": "SecureP@ss123"
}
```

*Response (inferred):*
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "expires_in": 3600,
  "member": {
    "id": 1,
    "email": "admin@example.com",
    "name": "Admin User",
    "status": "active",
    "roles": ["owner"]
  }
}
```

**POST /api/v1/auth/refresh**

*Request Body (inferred):*
```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

*Response (inferred):*
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "expires_in": 3600
}
```

---

### 4.2 Members / Users (MemberHandler)

**Route Setup:** `SetupMemberRoutes` (lines 13-100, 87 lines, confirmed)
**Handler:** `MemberHandler`
**Middleware:** AdminMiddleware + LowercaseMiddleware (on create/update routes)

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/member` | `MemberHandler.GetMember` | AdminMiddleware | JWT | Get current authenticated member profile |
| POST | `/api/v1/member` | `MemberHandler.CreateMember` | AdminMiddleware, LowercaseMiddleware | JWT | Create a new member (invite) |
| PUT | `/api/v1/member/reset-password` | `MemberHandler.ResetPassword` | AdminMiddleware, LowercaseMiddleware | JWT | Reset another member's password (admin) |
| PUT | `/api/v1/member/activate/:id` | `MemberHandler.Activate` | AdminMiddleware, LowercaseMiddleware | JWT | Activate a member account |
| DELETE | `/api/v1/member/:id` | `MemberHandler.DeleteMember` | AdminMiddleware, LowercaseMiddleware | JWT | Soft-delete a member |
| PUT | `/api/v1/member/assign-platform-roles` | `MemberHandler.AssignMemberPlatformRoles` | AdminMiddleware, LowercaseMiddleware | JWT | Assign platform-level roles to a member |
| PUT | `/api/v1/member/revoke/:id` | `MemberHandler.Revoke` | AdminMiddleware, LowercaseMiddleware | JWT | Revoke a member's access |
| PUT | `/api/v1/member/internal-roles` | `MemberHandler.UpdateInternalMemberRoles` | AdminMiddleware, LowercaseMiddleware | JWT | Update internal (system-level) roles |
| GET | `/api/v1/member/recent` | `MemberHandler.GetLast20Members` | AdminMiddleware | JWT | Get 20 most recently created members |
| GET | `/api/v1/member/internal` | `MemberHandler.GetAllInternalMembers` | AdminMiddleware | JWT | List all internal (non-platform) members |
| PUT | `/api/v1/member/change-password` | `MemberHandler.ChangePassword` | AdminMiddleware | JWT | Change own password |
| GET | `/api/v1/member/root-exists` | `MemberHandler.CheckRootMemberExist` | None | None | Check if root member exists (for initial setup) |
| POST | `/api/v1/member/forgot-password` | `MemberHandler.ForgotPassword` | LowercaseMiddleware | None | Initiate password reset via email |

**Member Model (confirmed):**
```
Member {
  id:                   uint
  email:                string
  password:             string (bcrypt hashed)
  name:                 string
  mobile_number:        *string
  status:               string ("active" | "inactive" | "revoked")
  member_type:          string ("internal" | "platform")
  verification_token:   *string
  is_verified:          bool
  referred_by_member_id: *uint
  referral_code:        *string
  referred_by_member:   *Member
  created_at:           timestamp
  updated_at:           timestamp
  deleted_at:           *timestamp
}
```

---

### 4.3 OTP (OTPHandler)

**Route Setup:** `SetupOTPRoutes` (lines 13-40, 27 lines, confirmed)
**Handler:** `OTPHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/otp/validate` | `OTPHandler.ValidateOTP` | AdminMiddleware | JWT | Validate a one-time password |
| POST | `/api/v1/otp/validate/merchant` | `OTPHandler.ValidateOTPForMerchant` | AdminMiddleware | JWT | Validate OTP for merchant API context |
| POST | `/api/v1/otp/regenerate` | `OTPHandler.RegenerateOTP` | AdminMiddleware | JWT | Generate and send a new OTP |
| POST | `/api/v1/otp/regenerate/merchant` | `OTPHandler.RegenerateOTPForMerchant` | AdminMiddleware | JWT | Regenerate OTP for merchant API context |

*Request Body for ValidateOTP (inferred):*
```json
{
  "otp": "123456",
  "purpose": "withdrawal_approval"
}
```

*Response (inferred):*
```json
{
  "valid": true,
  "message": "OTP validated successfully"
}
```

---

### 4.4 API Keys (APIKeyHandler)

**Route Setup:** `SetupAPIKeyRoutes` (lines 13-41, 28 lines, confirmed)
**Handler:** `APIKeyHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware
**Service:** `APIKeyServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/api-key/platform/:platform_id` | `APIKeyHandler.CreateForExternalPlatform` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Create API key for a platform/project |
| GET | `/api/v1/api-key/platform/:platform_id` | `APIKeyHandler.GetExternalPlatformAPIKeys` | AdminMiddleware, ExternalPlatformMiddleware | JWT | List all API keys for a platform |
| PUT | `/api/v1/api-key/:id` | `APIKeyHandler.UpdateExternalPlatformAPIKey` | AdminMiddleware | JWT | Update an API key |
| PUT | `/api/v1/api-key/:id/activate` | `APIKeyHandler.ActivateExternalPlatformAPIKey` | AdminMiddleware | JWT | Activate an API key |
| PUT | `/api/v1/api-key/:id/deactivate` | `APIKeyHandler.DeactivateExternalPlatformAPIKey` | AdminMiddleware | JWT | Deactivate an API key |
| POST | `/api/v1/api-key/referral/merchant` | `APIKeyHandler.CreateApiKeyForReferralFromMerchant` | AdminMiddleware, ReferralServiceMiddleware | JWT | Create API key for referral system via merchant API |

*Response for CreateForExternalPlatform (inferred):*
```json
{
  "id": 1,
  "api_key": "pm_live_abc123...",
  "status": "active",
  "external_platform_id": 1,
  "role_id": 2,
  "created_at": "2026-04-07T12:00:00Z"
}
```

*Note:* The raw `api_key` value is only returned at creation time. The stored value is a SHA-256 hash.

---

### 4.5 Roles and Permissions (RoleHandler)

**Route Setup:** `SetupRoleRoutes` (lines 13-48, 35 lines, confirmed)
**Handler:** `RoleHandler`
**Middleware:** AdminMiddleware, LowercaseMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/role` | `RoleHandler.Create` | AdminMiddleware, LowercaseMiddleware | JWT | Create a new role |
| GET | `/api/v1/role` | `RoleHandler.GetAll` | AdminMiddleware, LowercaseMiddleware | JWT | List all roles with permissions |
| GET | `/api/v1/role/:id` | `RoleHandler.GetRole` | AdminMiddleware, LowercaseMiddleware | JWT | Get a specific role |
| PUT | `/api/v1/role/:id/permissions` | `RoleHandler.UpdateRolePermissions` | AdminMiddleware | JWT | Update permissions for a role |

**Role Model (confirmed):**
```
Role {
  id:          uint
  name:        string
  description: *string
  is_default:  bool
  permissions: []Permission
}
```

**Permission Model (confirmed):**
```
Permission {
  id:          uint
  name:        string
  description: *string
  category:    string
  action:      string
  resource:    string
}
```

**Predefined Roles (confirmed from PRODUCT_SPEC):**
- Owner -- Full access, cannot be removed
- Admin -- Manage all projects/payments, invite/remove users
- Project Lead -- Create/update projects, analytics
- Project Manager -- View-only, export reports
- Project Ops -- View payment/customer data
- Platform Referral Admin -- Full referral program access

---

### 4.6 Payments (PaymentHandler)

**Route Setup:** `SetupPaymentsRoutes` (lines 13-85, 72 lines, confirmed)
**Handler:** `PaymentHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware, PaymentRequestMiddleware
**Service:** `PaymentRequestServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/payment` | `PaymentHandler.PaymentRequest` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Create payment request (dashboard) |
| POST | `/api/v1/payment/merchant` | `PaymentHandler.PaymentRequestFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware | API Key | Create payment request (merchant API) |
| GET | `/api/v1/payment/reference/:reference_id` | `PaymentHandler.GetPaymentRequest` | AdminMiddleware, PaymentRequestMiddleware | JWT/API Key | Get payment by reference ID |
| GET | `/api/v1/payment/:id` | `PaymentHandler.GetPaymentDetails` | AdminMiddleware | JWT | Get payment details by internal ID |
| GET | `/api/v1/payment/search` | `PaymentHandler.Search` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Search payments with filters |
| GET | `/api/v1/payment/summary` | `PaymentHandler.Summary` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Get payment summary/stats |
| POST | `/api/v1/payment/:reference_id/confirm` | `PaymentHandler.Payment` | AdminMiddleware, PaymentRequestMiddleware | JWT | Process/confirm a payment |
| POST | `/api/v1/payment/:id/webhook/approve` | `PaymentHandler.PaymentRequestApproveSendWebhook` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Approve and send pending webhook |
| POST | `/api/v1/payment/:id/webhook/discard` | `PaymentHandler.PaymentRequestDiscardWebhook` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Discard pending webhook |
| GET | `/api/v1/payment/webhook/pending` | `PaymentHandler.GetPaymentRequestsPendingWebhookApproval` | AdminMiddleware, ExternalPlatformMiddleware | JWT | List payments with pending webhook approval |
| POST | `/api/v1/payment/dummy-webhook` | `PaymentHandler.DummyWebhook` | AdminMiddleware | JWT | Test webhook delivery (development) |

**POST /api/v1/payment (confirmed from PRODUCT_SPEC):**

*Request Body:*
```json
{
  "customerEmail": "user@example.com",
  "customerID": "cust_123",
  "amountInUSD": 100.00,
  "invoiceID": "inv_456"
}
```

*Response (confirmed from PRODUCT_SPEC):*
```json
{
  "host": "https://yourdomain.com:8443",
  "reference_id": "c80f5363-0397-4761-aa1a-3155c3a21470",
  "url": "https://yourdomain.com/payments?reference_id=c80f5363...&host=https://yourdomain.com:8443"
}
```

**GET /api/v1/payment/reference/:reference_id (confirmed from PRODUCT_SPEC):**

*Response:*
```json
{
  "id": 1,
  "invoiceID": "inv_456",
  "customerID": "cust_123",
  "amountInUSD": 100.00,
  "amount": "0.05",
  "filledAmount": "0.05",
  "filledAmountInUSD": 100.00,
  "currencyCode": "ETH",
  "blockchainCode": "ETH",
  "paymentState": "FILLED",
  "merchantName": "Your Store",
  "referenceID": "c80f5363-0397-4761-aa1a-3155c3a21470",
  "expireAt": "2026-04-07T13:00:00Z",
  "createdAt": "2026-04-07T12:00:00Z"
}
```

**Payment States (confirmed):** `OPEN` | `CANCELLED` | `FILLED` | `PARTIALLY_FILLED` | `OVER_FILLED`

**PaymentRequest Model (confirmed):**
```
PaymentRequest {
  id:                     uint
  amount:                 *decimal.Decimal
  amount_in_usd:          decimal.Decimal
  filled_amount:          *decimal.Decimal
  filled_amount_in_usd:   *decimal.Decimal
  sponsored_amount:       decimal.Decimal
  sponsored_amount_in_usd: decimal.Decimal
  currency_code:          *string
  blockchain_code:        *string
  invoice_id:             string
  reference_id:           string (UUID)
  status:                 string
  expire_at:              *timestamp
  member_id:              uint
  currency_id:            *uint
  blockchain_id:          *uint
  deposit_id:             *uint
  created_by:             string
  external_platform_id:   uint
}
```

**GET /api/v1/payment/search**

*Query Parameters (inferred from `SearchParams`):*
```
?query=<search_text>
&status=OPEN|FILLED|CANCELLED
&blockchain_code=ETH|BTC|BASE|POLYGON|TRX
&currency_code=USDT|USDC|ETH|BTC
&from_date=2026-01-01
&to_date=2026-04-07
&sort_by=created_at|amount_in_usd|status
&order=asc|desc
&limit=20
&offset=0
```

*Response (inferred from `NewPaymentSearchResponse`):*
```json
{
  "data": [
    {
      "id": 1,
      "reference_id": "...",
      "amount_in_usd": 100.00,
      "status": "FILLED",
      "currency_code": "USDT",
      "blockchain_code": "ETH",
      "created_at": "2026-04-07T12:00:00Z"
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

---

### 4.7 Payment Channels (PaymentChannelHandler)

**Route Setup:** `SetupPaymentChannelRoutes` (lines 13-73, 60 lines, confirmed)
**Handler:** `PaymentChannelHandler`
**Middleware:** AdminMiddleware, PaymentRequestMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/payment-channel` | `PaymentChannelHandler.GetPaymentChannels` | AdminMiddleware | JWT | List all payment channels |
| GET | `/api/v1/payment-channel/:id` | `PaymentChannelHandler.GetPaymentChannelByID` | AdminMiddleware | JWT | Get payment channel by ID |
| GET | `/api/v1/payment-channel/project/:project_id` | `PaymentChannelHandler.GetPaymentChannelsByProjectID` | AdminMiddleware | JWT | Get channels by project |
| GET | `/api/v1/payment-channel/reference/:reference_id` | `PaymentChannelHandler.GetPaymentChannelsByReferenceID` | AdminMiddleware, PaymentRequestMiddleware | JWT | Get channels for a payment request |
| PUT | `/api/v1/payment-channel/:id/activate` | `PaymentChannelHandler.ActivatePaymentChannel` | AdminMiddleware | JWT | Activate a payment channel |
| PUT | `/api/v1/payment-channel/:id/deactivate` | `PaymentChannelHandler.DeactivatePaymentChannel` | AdminMiddleware | JWT | Deactivate a payment channel |
| PUT | `/api/v1/payment-channel/:id` | `PaymentChannelHandler.UpdatePaymentChannel` | AdminMiddleware | JWT | Update payment channel settings |
| PUT | `/api/v1/payment-channel/:id/api-key` | `PaymentChannelHandler.UpdatePaymentChannelAPIKey` | AdminMiddleware | JWT | Update channel API key |
| GET | `/api/v1/payment-channel/:id/api-key` | `PaymentChannelHandler.GetPaymentChannelAPIKey` | AdminMiddleware | JWT | Get channel API key |
| GET | `/api/v1/payment-channel/reference/:reference_id/api-key` | `PaymentChannelHandler.GetPaymentChannelAPIKeyByReferenceID` | AdminMiddleware, PaymentRequestMiddleware | JWT | Get channel API key by reference |

**PaymentChannel Model (confirmed):**
```
PaymentChannel {
  id:              uint
  name:            string
  channel_type:    PaymentChannelType
  status:          PaymentChannelStatus
  configuration:   *PaymentChannelConfiguration (JSON)
  metadata:        *PaymentChannelMetadata (JSON)
  api_key:         *string
  display_name:    string
  description:     *string
  icon:            *string
  display_order:   int
  is_default:      bool
  disabled:        *bool
}
```

---

### 4.8 Payment Channel Projects (PaymentChannelProjectHandler)

**Route Setup:** `SetupPaymentChannelProjectRoutes` (lines 13-29, 16 lines, confirmed)
**Handler:** `PaymentChannelProjectHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| PUT | `/api/v1/payment-channel-project/:id/activate` | `PaymentChannelProjectHandler.ActivatePaymentChannelProject` | AdminMiddleware | JWT | Enable a payment channel for a project |
| PUT | `/api/v1/payment-channel-project/:id/deactivate` | `PaymentChannelProjectHandler.DeactivatePaymentChannelProject` | AdminMiddleware | JWT | Disable a payment channel for a project |

---

### 4.9 Deposits (DepositHandler)

**Route Setup:** `SetupDepositRoutes` (lines 13-22, 9 lines, confirmed)
**Handler:** `DepositHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/deposit` | `DepositHandler.GetDeposits` | AdminMiddleware | JWT | List deposits with filters |

**Deposit Model (confirmed):**
```
Deposit {
  id:                uint
  tx_hash:           string
  unique_tx_hash:    string
  currency_code:     string
  blockchain_code:   string
  amount:            decimal.Decimal
  price_in_usd:      *decimal.Decimal
  amount_in_usd:     *decimal.Decimal
  fee:               *decimal.Decimal
  from_address:      *string
  to_address:        string
  block_hash:        *string
  status:            string
  block_number:      *uint64
  timestamp:         *time.Time
  member_id:         uint
  blockchain_id:     uint
  currency_id:       uint
}
```

---

### 4.10 Deposit Addresses (DepositAddressHandler)

**Route Setup:** `SetupDepositAddressRoutes` (lines 12-19, 7 lines, confirmed)
**Handler:** `DepositAddressHandler`
**Middleware:** PaymentRequestMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/deposit-address/reference/:reference_id` | `DepositAddressHandler.AssignAddress` | PaymentRequestMiddleware | API Key | Assign a deposit address for a payment |

**POST /api/v1/deposit-address/reference/:reference_id (confirmed from PRODUCT_SPEC):**

*Request Body:*
```json
{
  "blockchain_code": "ETH"
}
```

*Response (inferred):*
```json
{
  "address": "0x1234567890abcdef...",
  "blockchain_code": "ETH",
  "family": "ethereum",
  "reference_id": "c80f5363-0397-4761-aa1a-3155c3a21470"
}
```

**DepositAddress Model (confirmed):**
```
DepositAddress {
  id:                  uint
  address:             string
  address_lower:       string
  family:              string
  status:              string
  member_id:           uint
  wallet_id:           *uint
  blockchain_family_id: uint
}
```

---

### 4.11 Addresses (AddressHandler)

**Route Setup:** `SetupAddressRoutes` (lines 13-51, 38 lines, confirmed)
**Handler:** `AddressHandler`
**Middleware:** AdminMiddleware
**Service:** `AddressServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/address/balances` | `AddressHandler.Balances` | AdminMiddleware | JWT | Get address balances |
| GET | `/api/v1/address/eligible-to-sweep` | `AddressHandler.GetEligibleAddressesToSweep` | AdminMiddleware | JWT | List addresses eligible for sweep |
| GET | `/api/v1/address/pending-approval` | `AddressHandler.PendingForApproval` | AdminMiddleware | JWT | List addresses pending sweep approval |
| PUT | `/api/v1/address/sweep-initiated` | `AddressHandler.SweepInitiatedForAddresses` | AdminMiddleware | JWT | Mark addresses as sweep initiated |
| PUT | `/api/v1/address/sweep-cancelled` | `AddressHandler.SweepCancelledForAddresses` | AdminMiddleware | JWT | Cancel sweep for addresses |

---

### 4.12 Address Pools (AddressPoolHandler)

**Route Setup:** `SetupAddressPoolRoutes` (lines 13-28, 15 lines, confirmed)
**Handler:** `AddressPoolHandler`
**Middleware:** AdminMiddleware
**Service:** `AddressPoolServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/address-pool/private-keys` | `AddressPoolHandler.AddPrivateKeys` | AdminMiddleware | JWT | Add private keys to address pool |
| GET | `/api/v1/address-pool/unassigned` | `AddressPoolHandler.GetAddressesWithUnassignedPrivateKeys` | AdminMiddleware | JWT | List addresses needing private keys |

**AddressPool Model (confirmed):**
```
AddressPool {
  id:                  uint
  address:             string
  address_lower:       string
  blockchain_family:   string
  status:              string
  member_id:           *uint
  wallet_id:           *uint
  blockchain_family_id: uint
  derivation_path:     *string
  private_key_assigned: bool
}
```

---

### 4.13 Wallets (WalletHandler)

**Route Setup:** `SetupWalletRoutes` (lines 13-111, 98 lines, confirmed)
**Handler:** `WalletHandler`
**Middleware:** AdminMiddleware, PaymentRequestMiddleware
**Service:** `WalletServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/wallet/eoa-deposit` | `WalletHandler.CreateBulkEOADepositWallets` | AdminMiddleware | JWT | Create EOA deposit wallets in bulk |
| POST | `/api/v1/wallet/scw-deposit` | `WalletHandler.CreateSCWDepositWallets` | AdminMiddleware | JWT | Create smart contract deposit wallets |
| POST | `/api/v1/wallet/cold` | `WalletHandler.CreateColdWallet` | AdminMiddleware | JWT | Create a cold storage wallet |
| POST | `/api/v1/wallet/hot` | `WalletHandler.CreateHotWallet` | AdminMiddleware | JWT | Create a hot wallet |
| POST | `/api/v1/wallet/function` | `WalletHandler.CreateHotWalletFunctionAssociation` | AdminMiddleware | JWT | Associate a function with hot wallet |
| PUT | `/api/v1/wallet/:id/default` | `WalletHandler.SetDefaultWallet` | AdminMiddleware | JWT | Set wallet as default for blockchain |
| GET | `/api/v1/wallet` | `WalletHandler.GetWallets` | AdminMiddleware | JWT | List all wallets |
| GET | `/api/v1/wallet/reference/:reference_id` | `WalletHandler.GetWalletsWithReferenceID` | AdminMiddleware, PaymentRequestMiddleware | JWT | Get wallets for a payment reference |
| GET | `/api/v1/wallet/:id` | `WalletHandler.GetWallet` | AdminMiddleware | JWT | Get wallet by ID |
| PUT | `/api/v1/wallet/:id` | `WalletHandler.Update` | AdminMiddleware | JWT | Update wallet settings |
| PUT | `/api/v1/wallet/:id/fund-collector` | `WalletHandler.SetFundCollectorAddress` | AdminMiddleware | JWT | Set fund collector smart contract address |
| PUT | `/api/v1/wallet/:id/activate` | `WalletHandler.Activate` | AdminMiddleware | JWT | Activate a wallet |
| PUT | `/api/v1/wallet/:id/deactivate` | `WalletHandler.Deactivate` | AdminMiddleware | JWT | Deactivate a wallet |
| DELETE | `/api/v1/wallet/:id` | `WalletHandler.Delete` | AdminMiddleware | JWT | Delete a wallet |
| POST | `/api/v1/wallet/:id/generate-addresses` | `WalletHandler.GenerateAddresses` | AdminMiddleware | JWT | Generate new deposit addresses for wallet |

**Wallet Model (confirmed):**
```
Wallet {
  id:                     uint
  name:                   string
  wallet_type:            string ("cold" | "hot" | "deposit")
  status:                 string ("active" | "inactive")
  address:                string
  address_lower:          string
  is_default:             bool
  blockchain_code:        string
  blockchain_id:          uint
  member_id:              uint
  xpub:                   *string
  derivation_path:        *string
  fund_collector_address: *string
  fund_collector_address_lower: *string
  is_scw:                 bool
  scw_factory_address:    *string
  min_balance_for_sweep:  *decimal.Decimal
  sweep_enabled:          bool
}
```

---

### 4.14 Blockchain Management (BlockchainHandler)

**Route Setup:** `SetupBlockchainRoutes` (lines 13-62, 49 lines, confirmed)
**Handler:** `BlockchainHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/blockchain` | `BlockchainHandler.Get` | AdminMiddleware | JWT | List all blockchains |
| GET | `/api/v1/blockchain/:code` | `BlockchainHandler.GetByCode` | AdminMiddleware | JWT | Get blockchain by code |
| PUT | `/api/v1/blockchain/:code` | `BlockchainHandler.Update` | AdminMiddleware | JWT | Update blockchain settings |
| POST | `/api/v1/blockchain/:code/rpc-node` | `BlockchainHandler.CreateRPCNode` | AdminMiddleware | JWT | Add an RPC node for a blockchain |
| PUT | `/api/v1/blockchain/rpc-node/:id` | `BlockchainHandler.UpdateRPCNode` | AdminMiddleware | JWT | Update RPC node configuration |
| POST | `/api/v1/blockchain/:code/check-connection` | `BlockchainHandler.CheckServerConnection` | AdminMiddleware | JWT | Test RPC node connectivity |
| GET | `/api/v1/blockchain/:code/unspent` | `BlockchainHandler.UnspentTransactions` | AdminMiddleware | JWT | List unspent transactions (UTXO chains) |
| GET | `/api/v1/blockchain/:code/spent` | `BlockchainHandler.SpentTransactions` | AdminMiddleware | JWT | List spent transactions |
| POST | `/api/v1/blockchain/:code/withdraw-deposits` | `BlockchainHandler.WithdrawDeposits` | AdminMiddleware | JWT | Withdraw deposit balances |
| GET | `/api/v1/blockchain/:code/withdrawn` | `BlockchainHandler.Withdrawn` | AdminMiddleware | JWT | List withdrawn transactions |

**Blockchain Model (confirmed):**
```
Blockchain {
  id:                       uint
  name:                     string
  code:                     string ("ETH" | "BTC" | "BASE" | "POLYGON" | "TRX")
  blockchain_family_id:     uint
  native_currency_id:       uint
  status:                   string
  confirmations_required:   int
  block_time_seconds:       int
  explorer_url:             *string
  explorer_tx_url:          *string
  explorer_address_url:     *string
  chain_id:                 *int64
  is_testnet:               bool
  rpc_nodes:                []RPCNode
}
```

**RPCNode Model (confirmed):**
```
RPCNode {
  id:                uint
  blockchain_id:     uint
  name:              string
  url:               string
  ws_url:            *string
  api_key:           *string
  status:            string
  priority:          int
  credential_hash:   string
}
```

---

### 4.15 Blockchain Currencies (BlockchainCurrencyHandler)

**Route Setup:** `SetupBlockchainCurrencyRoutes` (lines 13-47, 34 lines, confirmed)
**Handler:** `BlockchainCurrencyHandler`
**Middleware:** AdminMiddleware, PaymentRequestMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/blockchain-currency` | `BlockchainCurrencyHandler.GetAllBlockchainCurrencies` | AdminMiddleware | JWT | List all blockchain-currency pairs |
| GET | `/api/v1/blockchain-currency/supported` | `BlockchainCurrencyHandler.GetSupportedBlockchainCurrencies` | AdminMiddleware | JWT | List supported/active pairs |
| GET | `/api/v1/blockchain-currency/:blockchain_code/:currency_code` | `BlockchainCurrencyHandler.GetByBlockchainAndCurrencyCode` | AdminMiddleware | JWT | Get specific pair |
| PUT | `/api/v1/blockchain-currency/:id` | `BlockchainCurrencyHandler.Update` | AdminMiddleware | JWT | Update blockchain-currency config |
| POST | `/api/v1/blockchain-currency/reference/:reference_id` | `BlockchainCurrencyHandler.GetAllPublicDetailsOfBlockchainCurrencies` | PaymentRequestMiddleware | API Key | Get available currencies for a payment |

**BlockchainCurrency Model (confirmed):**
```
BlockchainCurrency {
  id:                    uint
  blockchain_id:         uint
  currency_id:           uint
  blockchain_code:       string
  currency_code:         string
  contract_address:      *string
  decimal_places:        int
  status:                string
  deposit_enabled:       bool
  withdrawal_enabled:    bool
  sweep_enabled:         bool
  min_deposit_amount:    *decimal.Decimal
  min_withdrawal_amount: *decimal.Decimal
  min_balance_for_sweep: *decimal.Decimal
  blockchain:            *Blockchain
}
```

---

### 4.16 Blockchain Families (BlockchainFamilyHandler)

**Route Setup:** `SetupBlockchainFamilyRoutes` (lines 13-34, 21 lines, confirmed)
**Handler:** `BlockchainFamilyHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/blockchain-family` | `BlockchainFamilyHandler.Create` | AdminMiddleware | JWT | Create a new blockchain family |
| GET | `/api/v1/blockchain-family` | `BlockchainFamilyHandler.GetBlockchainFamilies` | AdminMiddleware | JWT | List all blockchain families |
| GET | `/api/v1/blockchain-family/:family` | `BlockchainFamilyHandler.GetByFamily` | AdminMiddleware | JWT | Get by family name |

**BlockchainFamily Model (confirmed):**
```
BlockchainFamily {
  id:                  uint
  name:                string
  code:                string ("ethereum" | "bitcoin" | "tron")
  supports_hd_wallet:  bool
  supports_scw_wallet: bool
}
```

---

### 4.17 Blockchain Contracts (BlockchainContractHandler)

**Route Setup:** `SetupBlockchainContractRoutes` (lines 13-32, 19 lines, confirmed)
**Handler:** `BlockchainContractHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/blockchain-contract` | `BlockchainContractHandler.Create` | AdminMiddleware | JWT | Register a blockchain contract |
| GET | `/api/v1/blockchain-contract/type/:contract_type` | `BlockchainContractHandler.GetByContractType` | AdminMiddleware | JWT | List contracts by type |
| GET | `/api/v1/blockchain-contract/:blockchain_code/:contract_type` | `BlockchainContractHandler.GetByBlockchainCodeAndContractType` | AdminMiddleware | JWT | Get specific contract |
| GET | `/api/v1/blockchain-contract/address/:contract_type` | `BlockchainContractHandler.GetAllContractsByAddressContractType` | AdminMiddleware | JWT | Get contracts by address contract type |

**BlockchainContract Model (confirmed):**
```
BlockchainContract {
  id:                uint
  blockchain_id:     uint
  blockchain_code:   string
  name:              string
  contract_address:  string
  contract_type:     string
  abi:               *string
  status:            string
  blockchain:        *Blockchain
}
```

---

### 4.18 Contract Addresses (ContractAddressHandler)

**Route Setup:** `SetupContractAddressRoutes` (lines 13-28, 15 lines, confirmed)
**Handler:** `ContractAddressHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/contract-address` | `ContractAddressHandler.Create` | AdminMiddleware | JWT | Create a contract address entry |
| GET | `/api/v1/contract-address/:blockchain_code/:contract_type` | `ContractAddressHandler.GetByBlockchainCodeAndContractType` | AdminMiddleware | JWT | Get contract address by blockchain and type |
| PUT | `/api/v1/contract-address/:id` | `ContractAddressHandler.Update` | AdminMiddleware | JWT | Update a contract address |

**ContractAddress Model (confirmed):**
```
ContractAddress {
  id:                uint
  name:              string
  description:       *string
  address:           string
  address_lower:     string
  blockchain_code:   string
  contract_type:     string
  status:            string
  abi:               *string
  blockchain_id:     uint
}
```

---

### 4.19 Currencies (CurrencyHandler)

**Route Setup:** `SetupCurrencyRoutes` (lines 13-41, 28 lines, confirmed)
**Handler:** `CurrencyHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/currency` | `CurrencyHandler.Create` | AdminMiddleware | JWT | Create a new currency |
| GET | `/api/v1/currency` | `CurrencyHandler.GetAllCurrencies` | AdminMiddleware | JWT | List all currencies |
| GET | `/api/v1/currency/:code` | `CurrencyHandler.GetByCode` | AdminMiddleware | JWT | Get currency by code |
| GET | `/api/v1/currency/referral` | `CurrencyHandler.GetCurrenciesForReferral` | AdminMiddleware | JWT | List currencies eligible for referral rewards |

**Currency Model (confirmed):**
```
Currency {
  id:                  uint
  name:                string
  code:                string ("USDT" | "USDC" | "ETH" | "BTC" | "TRX")
  symbol:              string
  currency_type:       string ("native" | "token")
  decimal_places:      int
  price_in_usd:        *decimal.Decimal
  icon:                *string
  withdrawal_enabled:  bool
}
```

---

### 4.20 Sweeps (SweepHandler)

**Route Setup:** `SetupSweepRoutes` (lines 13-27, 14 lines, confirmed)
**Handler:** `SweepHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/sweep` | `SweepHandler.Sweeps` | AdminMiddleware | JWT | List all sweeps |
| GET | `/api/v1/sweep/:id` | `SweepHandler.Swept` | AdminMiddleware | JWT | Get sweep details by ID |

**Sweep Model (confirmed):**
```
Sweep {
  id:                    uint
  blockchain_code:       string
  currency_code:         string
  from_address:          string
  to_address:            string
  amount:                decimal.Decimal
  fee:                   *decimal.Decimal
  tx_hash:               *string
  block_hash:            *string
  block_number:          *uint64
  status:                string ("pending" | "initiated" | "confirming" | "completed" | "failed")
  sweep_type:            string
  error_message:         *string
  timestamp:             *time.Time
  blockchain_id:         uint
  blockchain:            *Blockchain
}
```

---

### 4.21 Sweep Transactions (SweepTransactionHandler)

**Route Setup:** `SetupSweepTransactionRoutes` (lines 13-39, 26 lines, confirmed)
**Handler:** `SweepTransactionHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/sweep-transaction` | `SweepTransactionHandler.GetSweepTransactions` | AdminMiddleware | JWT | List sweep transactions |
| GET | `/api/v1/sweep-transaction/:id` | `SweepTransactionHandler.GetSweepTransaction` | AdminMiddleware | JWT | Get sweep transaction by ID |
| PUT | `/api/v1/sweep-transaction/:id/status` | `SweepTransactionHandler.UpdateSweepTransactionStatus` | AdminMiddleware | JWT | Update sweep transaction status |
| PUT | `/api/v1/sweep-transaction/:id` | `SweepTransactionHandler.UpdateSweepTransaction` | AdminMiddleware | JWT | Update sweep transaction details |

**SweepTransaction Model (confirmed):**
```
SweepTransaction {
  id:                  uint
  blockchain_code:     string
  currency_code:       string
  from_address:        string
  to_address:          string
  amount:              decimal.Decimal
  fee:                 *decimal.Decimal
  tx_hash:             *string
  status:              string
  blockchain_id:       uint
  wallet_id:           *uint
  blockchain:          *Blockchain
  wallet:              *Wallet
}
```

---

### 4.22 Sweep UTXOs (UtxoHandler)

**Route Setup:** `SetupSweepUtxoRoutes` (lines 11-21, 10 lines, confirmed)
**Handler:** `UtxoHandler`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/sweep-utxo/process` | `UtxoHandler.ProcessUTXOForSweepTx` | AdminMiddleware | JWT | Process UTXOs for a sweep transaction (BTC) |

**UTXO Model (confirmed):**
```
UTXO {
  id:                  uint
  tx_hash:             string
  vout:                uint
  amount:              decimal.Decimal
  address:             string
  status:              string
  block_number:        *uint64
  block_hash:          *string
  script_pub_key:      *string
  blockchain_code:     string
  currency_code:       string
  deposit_id:          *uint
  sweep_transaction_id: *uint
  deposit:             *Deposit
  sweep:               *Sweep
}
```

---

### 4.23 Withdrawals (WithdrawalHandler)

**Route Setup:** `SetupWithdrawalRoutes` (lines 13-105, 92 lines, confirmed)
**Handler:** `WithdrawalHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/withdrawal` | `WithdrawalHandler.Create` | AdminMiddleware | JWT | Create a withdrawal (dashboard) |
| POST | `/api/v1/withdrawal/merchant` | `WithdrawalHandler.CreateForMerchant` | AdminMiddleware, ExternalPlatformMiddleware | API Key | Create a withdrawal (merchant API) |
| GET | `/api/v1/withdrawal` | `WithdrawalHandler.Get` | AdminMiddleware | JWT | List withdrawals (dashboard) |
| GET | `/api/v1/withdrawal/merchant` | `WithdrawalHandler.GetFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware | API Key | List withdrawals (merchant API) |
| POST | `/api/v1/withdrawal/merchant/payout` | `WithdrawalHandler.CreateMerchantPayoutFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware | API Key | Create a merchant payout |
| GET | `/api/v1/withdrawal/merchant/payout` | `WithdrawalHandler.GetMerchantPayoutFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware | API Key | List merchant payouts |
| GET | `/api/v1/withdrawal/merchant/payout/:id` | `WithdrawalHandler.GetPayoutFromMerchantByWithdrawalID` | AdminMiddleware, ExternalPlatformMiddleware | API Key | Get specific payout |
| PUT | `/api/v1/withdrawal/:id/approve` | `WithdrawalHandler.ApproveWithdrawal` | AdminMiddleware | JWT | Approve a pending withdrawal |
| PUT | `/api/v1/withdrawal/:id/approve/merchant` | `WithdrawalHandler.ApproveWithdrawalForMerchant` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Approve withdrawal for merchant |
| PUT | `/api/v1/withdrawal/:id/reject` | `WithdrawalHandler.RejectWithdrawal` | AdminMiddleware | JWT | Reject a pending withdrawal |
| PUT | `/api/v1/withdrawal/:id/cancel` | `WithdrawalHandler.Cancel` | AdminMiddleware | JWT | Cancel a withdrawal |
| GET | `/api/v1/withdrawal/min-amount` | `WithdrawalHandler.WithdrawalMinAmount` | AdminMiddleware | JWT | Get minimum withdrawal amounts |

**POST /api/v1/withdrawal/merchant (confirmed from PRODUCT_SPEC):**

*Request Body:*
```json
{
  "email": "recipient@example.com",
  "blockChainCode": "ETH",
  "currencyCode": "USDT",
  "amount": "125.50",
  "toAddress": "0x...",
  "customerID": 12345
}
```

*Withdrawal Limits (confirmed from PRODUCT_SPEC):*
- Auto-approve: up to $500
- Hourly cap: $5,000
- Daily cap: $10,000

**Withdrawal States (confirmed):** `pending-otp` > `pending-approval` > `pending` > `initiated` > `sent` > `processed`

**Withdrawal Model (confirmed):**
```
Withdrawal {
  id:                    uint
  reference_id:          string
  blockchain_code:       string
  currency_code:         string
  amount:                decimal.Decimal
  amount_in_usd:         *decimal.Decimal
  fee:                   *decimal.Decimal
  from_address:          *string
  to_address:            string
  tx_hash:               *string
  status:                string
  type:                  string
  reason:                *string
  customer_id:           *string
  email:                 *string
  otp_validated_at:      *timestamp
  approved_at:           *timestamp
  rejected_at:           *timestamp
  initiated_at:          *timestamp
  processed_at:          *timestamp
  member_id:             uint
  created_by_member_id:  *uint
  approved_by_member_id: *uint
  rejected_by_member_id: *uint
  blockchain_id:         uint
  currency_id:           uint
  external_platform_id:  *uint
}
```

**GET /api/v1/withdrawal/merchant**

*Query Parameters (inferred from `ApplyGetWithdrawalRequestFromMerchant`):*
```
?status=pending|initiated|sent|processed
&limit=20
&offset=0
&order=desc
&sortBy=created_at
```

---

### 4.24 External Platforms (ExternalPlatformHandler)

**Route Setup:** `SetupExternalPlatformRoutes` (lines 13-101, 88 lines, confirmed)
**Handler:** `ExternalPlatformHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware, PaymentRequestMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/platform` | `ExternalPlatformHandler.Create` | AdminMiddleware | JWT | Create a new platform/project |
| GET | `/api/v1/platform` | `ExternalPlatformHandler.GetPlatforms` | AdminMiddleware | JWT | List all platforms |
| GET | `/api/v1/platform/:id` | `ExternalPlatformHandler.Get` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Get platform by ID |
| GET | `/api/v1/platform/reference/:reference_id` | `ExternalPlatformHandler.GetByReferenceID` | AdminMiddleware, PaymentRequestMiddleware | JWT | Get platform by reference ID |
| PUT | `/api/v1/platform/:id` | `ExternalPlatformHandler.Update` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Update platform settings |
| PUT | `/api/v1/platform/:id/permissions/grant` | `ExternalPlatformHandler.GrantPermissions` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Grant permissions to platform |
| PUT | `/api/v1/platform/:id/permissions/revoke` | `ExternalPlatformHandler.RevokePermissions` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Revoke platform permissions |
| GET | `/api/v1/platform/:id/members` | `ExternalPlatformHandler.GetMembers` | AdminMiddleware, ExternalPlatformMiddleware | JWT | List members of a platform |
| POST | `/api/v1/platform/:id/webhook` | `ExternalPlatformHandler.CreateWebhook` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Create a webhook for a platform |
| GET | `/api/v1/platform/:id/webhook` | `ExternalPlatformHandler.GetWebhooks` | AdminMiddleware, ExternalPlatformMiddleware | JWT | List webhooks for a platform |
| PUT | `/api/v1/platform/:id/webhook/:webhook_id` | `ExternalPlatformHandler.UpdateWebhook` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Update a webhook |
| DELETE | `/api/v1/platform/:id/webhook/:webhook_id` | `ExternalPlatformHandler.DeleteWebhook` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Delete a webhook |
| POST | `/api/v1/platform/:id/webhook/:webhook_id/test` | `ExternalPlatformHandler.TestWebhookConnection` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Test webhook connectivity |

**ExternalPlatform Model (confirmed):**
```
ExternalPlatform {
  id:                                     uint
  name:                                   string
  reference_id:                           string
  website_url:                            *string
  redirect_url:                           *string
  status:                                 string
  logo:                                   *string
  description:                            *string
  color:                                  *string
  default_payment_blockchain_currency_id: *uint
  default_payment_blockchain_currency:    *BlockchainCurrency
  default_payment_blockchain:             *Blockchain
  external_platform_blockchain_currencies: []ExternalPlatformBlockchainCurrency
}
```

**Webhook Model (confirmed):**
```
Webhook {
  id:                    uint
  external_platform_id:  uint
  name:                  string
  url:                   string
  access_key:            string
  status:                string ("active" | "inactive")
  tested:                bool
}
```

---

### 4.25 External Platform Blockchain Currencies (ExternalPlatformBlockchainCurrencyHandler)

**Route Setup:** `SetupExternalPlatformBlockchainCurrencyRoutes` (lines 13-31, 18 lines, confirmed)
**Handler:** `ExternalPlatformBlockchainCurrencyHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/platform-blockchain-currency/:platform_id` | `ExternalPlatformBlockchainCurrencyHandler.Get` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Get supported currencies for platform |
| PUT | `/api/v1/platform-blockchain-currency/:platform_id` | `ExternalPlatformBlockchainCurrencyHandler.ReplaceSupportedNetworkAndCurrencies` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Replace supported networks/currencies for platform |

---

### 4.26 External Platform Wallet Mappings (ExternalPlatformWalletBlockchainFamilyHandler)

**Route Setup:** `SetupExternalPlatformWalletBlockchainFamilyRoutes` (lines 13-36, 23 lines, confirmed)
**Handler:** `ExternalPlatformWalletBlockchainFamilyHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware, PaymentRequestMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| PUT | `/api/v1/platform-wallet-mapping/:platform_id` | `ExternalPlatformWalletBlockchainFamilyHandler.CreateOrReplaceProjectWalletMappings` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Set wallet mappings for platform |
| GET | `/api/v1/platform-wallet-mapping/:platform_id` | `ExternalPlatformWalletBlockchainFamilyHandler.ListMappingsLinkedToExternalPlatform` | AdminMiddleware, ExternalPlatformMiddleware | JWT | List wallet mappings for platform |
| GET | `/api/v1/platform-wallet-mapping/reference/:reference_id` | `ExternalPlatformWalletBlockchainFamilyHandler.ListMappingsLinkedToExternalPlatformWithReferenceID` | AdminMiddleware, PaymentRequestMiddleware | JWT | List mappings by payment reference |

**ExternalPlatformWalletBlockchainFamily Model (confirmed):**
```
ExternalPlatformWalletBlockchainFamily {
  id:                    uint
  external_platform_id:  uint
  wallet_id:             uint
  blockchain_family_id:  uint
  blockchain_family:     string
  external_platform:     *ExternalPlatform
  wallet:                *Wallet
}
```

---

### 4.27 Analytics (AnalyticsHandler)

**Route Setup:** `SetupAnalyticsRoutes` (lines 13-30, 17 lines, confirmed)
**Handler:** `AnalyticsHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware
**Service:** `AnalyticsServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/analytics/groups` | `AnalyticsHandler.Groups` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Get analytics groups/categories |
| POST | `/api/v1/analytics/data` | `AnalyticsHandler.Data` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Fetch analytics data with filters |

**Service Methods (confirmed):**
- `GetAnalyticsUserGroups` -- returns user's configured analytics groups
- `FetchData` -- retrieves analytics data based on group/filter/date parameters
- `SocketUpdate` -- pushes real-time analytics updates via WebSocket

**AnalyticsGroup Model (confirmed):**
```
AnalyticsGroup {
  id:            uint
  name:          string
  code:          string
  description:   *string
  category:      string
  display_order: int
  status:        string
  query:         string
  graph_type:    string
}
```

---

### 4.28 Analytics Referral (AnalyticsReferralHandler)

**Route Setup:** `SetupAnalyticsReferralRoutes` (lines 13-23, 10 lines, confirmed)
**Handler:** `AnalyticsReferralHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware
**Service:** `AnalyticsReferralServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/analytics/referral/data` | `AnalyticsReferralHandler.Data` | AdminMiddleware, ExternalPlatformMiddleware | JWT | Fetch referral analytics data |

**Service Methods (confirmed):**
- `GetPromotersList` -- list of referral promoters with stats
- `GetTotalRevenue` -- total referral revenue
- `GetPromotersCount` -- count of active promoters
- `GetRefereeCount` -- count of referred users
- `GetCampaignCount` -- count of campaigns
- `GetRewardValue` -- total reward value
- `GetRewardClaimed` -- claimed rewards
- `GetRewardInfo` -- reward detail info
- `GetRewardStats` -- reward statistics

---

### 4.29 Activity Logs (ActivityLogHandler)

**Route Setup:** `SetupActivityLogRoutes` (lines 13-24, 11 lines, confirmed)
**Handler:** `ActivityLogHandler`
**Middleware:** AdminMiddleware
**Service:** `ActivityLogServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/activity-log` | `ActivityLogHandler.GetActivityLogs` | AdminMiddleware | JWT | List activity logs with filters |
| GET | `/api/v1/activity-log/categories` | `ActivityLogHandler.GetEventCategories` | AdminMiddleware | JWT | Get available event categories |
| GET | `/api/v1/activity-log/download` | `ActivityLogHandler.DownloadActivityLogsCSV` | AdminMiddleware | JWT | Download activity logs as CSV |

**ActivityLog Model (confirmed):**
```
ActivityLog {
  id:                    uint
  member_id:             uint
  event_type:            string
  event_category:        string
  description:           string
  ip_address:            *string
  user_agent:            *string
  request_method:        *string
  request_path:          *string
  request_body:          *string
  response_status:       *int
  response_body:         *string
  external_platform_id:  *uint
  project_ids:           *string
  role:                  *string
  metadata:              *string
}
```

---

### 4.30 Referral System (ReferralHandler)

**Route Setup:** `SetupReferralRoutes` (lines 13-195, 182 lines, confirmed)
**Handler:** `ReferralHandler`
**Middleware:** AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware

This is the largest route group. The referral system supports campaigns, events, referrers, referees, rewards, and payouts. Many endpoints have dual versions: one for dashboard (JWT) and one for merchant API.

#### Events

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/referral/event` | `ReferralHandler.CreateEvent` | AdminMiddleware, ReferralServiceMiddleware | JWT | Create a referral event type |
| PUT | `/api/v1/referral/event/:id` | `ReferralHandler.UpdateEvent` | AdminMiddleware, ReferralServiceMiddleware | JWT | Update a referral event type |
| GET | `/api/v1/referral/event` | `ReferralHandler.GetAllEvents` | AdminMiddleware, ReferralServiceMiddleware | JWT | List all referral events |

#### Campaigns

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/referral/campaign` | `ReferralHandler.CreateCampaign` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Create a referral campaign |
| GET | `/api/v1/referral/campaign` | `ReferralHandler.GetAllCampaigns` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | List all campaigns |
| GET | `/api/v1/referral/campaign/merchant` | `ReferralHandler.GetAllCampaignsFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | List campaigns (merchant API) |
| PUT | `/api/v1/referral/campaign/:id` | `ReferralHandler.UpdateCampaign` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Update a campaign |
| PUT | `/api/v1/referral/campaign/:id/status` | `ReferralHandler.UpdateCampaignStatus` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Activate/deactivate campaign |
| GET | `/api/v1/referral/campaign/:id/rewards/total` | `ReferralHandler.GetTotalRewardsForCampaign` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Get total rewards for a campaign |

#### Referrers

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/referral/referrer` | `ReferralHandler.CreateReferrer` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Create a referrer |
| POST | `/api/v1/referral/referrer/merchant` | `ReferralHandler.CreateOrUpdateReferrerForMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Create/update referrer (merchant API) |
| PUT | `/api/v1/referral/referrer/:id/campaign` | `ReferralHandler.UpdateCampaignForReferrer` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Assign campaign to referrer |
| PUT | `/api/v1/referral/referrer/:id/status` | `ReferralHandler.UpdateStatusForReferrer` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Update referrer status |
| PUT | `/api/v1/referral/referrer/:id/campaign/merchant` | `ReferralHandler.UpdateCampaignForReferrerFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Update referrer campaign (merchant API) |
| GET | `/api/v1/referral/referrer/merchant` | `ReferralHandler.GetReferrerForMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Get referrer details (merchant API) |
| GET | `/api/v1/referral/referrer` | `ReferralHandler.GetReferrer` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | List referrers |

#### Referees

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/referral/referee/merchant` | `ReferralHandler.CreateRefereeFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Register a referee (merchant API) |
| GET | `/api/v1/referral/referee/total/merchant` | `ReferralHandler.GetTotalRefereeFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Get referee count (merchant API) |
| GET | `/api/v1/referral/referee/total` | `ReferralHandler.GetTotalReferee` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Get total referee count |

#### Event Logs

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/referral/event-log/merchant` | `ReferralHandler.CreateEventLogFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Log referral event (merchant API) |

#### Rewards

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/referral/reward/merchant` | `ReferralHandler.GetRewardsFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | List rewards (merchant API) |
| GET | `/api/v1/referral/reward/total/merchant` | `ReferralHandler.GetTotalRewardsFromMerchant` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | API Key | Get reward total (merchant API) |
| GET | `/api/v1/referral/reward` | `ReferralHandler.GetRewards` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | List rewards |
| GET | `/api/v1/referral/reward/total` | `ReferralHandler.GetTotalRewards` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | Get reward total |

#### Withdrawals and Payouts

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/referral/withdrawal` | `ReferralHandler.GetWithdrawals` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | List referral withdrawals |
| GET | `/api/v1/referral/payout` | `ReferralHandler.GetReferralPayouts` | AdminMiddleware, ExternalPlatformMiddleware, ReferralServiceMiddleware | JWT | List referral payouts |

---

### 4.31 Missed Deposits (MissedDepositHandler)

**Route Setup:** `SetupMissedDepositRoutes` (lines 13-46, 33 lines, confirmed)
**Handler:** `MissedDepositHandler`
**Middleware:** AdminMiddleware, PaymentRequestMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/missed-deposit` | `MissedDepositHandler.CreateMissedDeposit` | AdminMiddleware | JWT | Report a missed deposit (dashboard) |
| GET | `/api/v1/missed-deposit` | `MissedDepositHandler.GetMissedDeposit` | AdminMiddleware | JWT | List missed deposits (dashboard) |
| POST | `/api/v1/missed-deposit/merchant` | `MissedDepositHandler.CreateMissedDepositFromMerchant` | AdminMiddleware, PaymentRequestMiddleware | API Key | Report missed deposit (merchant API) |
| GET | `/api/v1/missed-deposit/merchant` | `MissedDepositHandler.GetMissedDepositFromMerchant` | AdminMiddleware, PaymentRequestMiddleware | API Key | List missed deposits (merchant API) |

**MissedDeposit Model (confirmed):**
```
MissedDeposit {
  id:                uint
  blockchain_code:   string
  transaction_hash:  string
  status:            string
  block_number:      *uint64
}
```

*Query Parameters for GET (inferred from `ApplyMissedDepositGetRequest`):*
```
?status=pending|processed|rejected
&from_date=2026-01-01
&to_date=2026-04-07
&sort_by=created_at|blockchain_code|status
&order=asc|desc
&limit=20
&offset=0
```

---

### 4.32 Recipients (RecipientHandler)

**Route Setup:** `SetupRecipientRoutes` (lines 13-42, 29 lines, confirmed)
**Handler:** `RecipientHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/recipient` | `RecipientHandler.Get` | AdminMiddleware | JWT | List recipients |
| POST | `/api/v1/recipient` | `RecipientHandler.Create` | AdminMiddleware | JWT | Create a recipient |
| PUT | `/api/v1/recipient/:id` | `RecipientHandler.Update` | AdminMiddleware | JWT | Update a recipient |
| DELETE | `/api/v1/recipient/:id` | `RecipientHandler.Delete` | AdminMiddleware | JWT | Delete a recipient |

**Recipient Model (confirmed):**
```
Recipient {
  id:                    uint
  name:                  *string
  email:                 *string
  mobile_number:         *string
  residential_address:   *string
  blockchain_code:       string
  address:               string
  member_id:             uint
  status:                string ("active" | "pending_otp")
  operated_by_member_id: uint
  last_operation:        string
  blockchain_id:         uint
}
```

*Query Parameters for GET (inferred from `ApplyGetRecipientRequest`):*
```
?blockchain_code=ETH|BTC|TRX
&status=active|pending_otp
&limit=20
&offset=0
```

---

### 4.33 Configuration (ConfigurationHandler)

**Route Setup:** `SetupConfigurationRoutes` (lines 13-34, 21 lines, confirmed)
**Handler:** `ConfigurationHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/configuration` | `ConfigurationHandler.CreateConfiguration` | AdminMiddleware | JWT | Create a configuration entry |
| GET | `/api/v1/configuration` | `ConfigurationHandler.GetConfiguration` | AdminMiddleware | JWT | Get all configuration entries |
| GET | `/api/v1/configuration/default` | `ConfigurationHandler.GetDefaultConfiguration` | AdminMiddleware | JWT | Get default configuration values |
| PUT | `/api/v1/configuration/:key` | `ConfigurationHandler.UpdateConfiguration` | AdminMiddleware | JWT | Update a configuration entry |
| PUT | `/api/v1/configuration/bulk` | `ConfigurationHandler.BulkUpsert` | AdminMiddleware | JWT | Bulk create/update configurations |

**Configuration Model (confirmed):**
```
Configuration {
  id:                   uint
  config_key:           string
  config_value:         string
  override_config_value: string
  description:          string
  encrypt:              bool
}
```

---

### 4.34 SMTP Configuration (SMTPConfigHandler)

**Route Setup:** `SetupSMTPConfigRoutes` (lines 12-21, 9 lines, confirmed)
**Handler:** `SMTPConfigHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| PUT | `/api/v1/smtp-config` | `SMTPConfigHandler.UpsertSMTPConfig` | AdminMiddleware | JWT | Create or update SMTP settings |
| GET | `/api/v1/smtp-config` | `SMTPConfigHandler.GetSMTPConfig` | AdminMiddleware | JWT | Get current SMTP configuration |
| POST | `/api/v1/smtp-config/test` | `SMTPConfigHandler.SendTestEmail` | AdminMiddleware | JWT | Send a test email |

*Request Body for UpsertSMTPConfig (inferred):*
```json
{
  "host": "smtp.gmail.com",
  "port": 587,
  "username": "noreply@example.com",
  "password": "app-specific-password",
  "from_email": "noreply@example.com",
  "from_name": "Payminto",
  "encryption": "tls"
}
```

---

### 4.35 Onramper Payments (OnramperPaymentsHandler)

**Route Setup:** `SetupOnramperPaymentsRoutes` (lines 13-21, 8 lines, confirmed)
**Handler:** `OnramperPaymentsHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/onramper-payments` | `OnramperPaymentsHandler.GetOnramperPayments` | AdminMiddleware | JWT | List card-to-crypto onramp payments |
| GET | `/api/v1/onramper-payments/metrics` | `OnramperPaymentsHandler.GetOnramperPaymentsMetrics` | AdminMiddleware | JWT | Get onramp payment metrics/stats |

*Query Parameters (inferred from `ApplyOnramperPaymentsFilters`):*
```
?status=pending|completed|failed
&from_date=2026-01-01
&to_date=2026-04-07
&sort_by=created_at|amount|status
&order=asc|desc
&limit=20
&offset=0
```

---

### 4.36 Payments App (PaymentsAppHandler)

**Route Setup:** `SetupPaymentsAppRoutes` (lines 13-40, 27 lines, confirmed)
**Handler:** `PaymentsAppHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/payments-app` | `PaymentsAppHandler.GetPaymentsApps` | AdminMiddleware | JWT | List all payments app configurations |
| GET | `/api/v1/payments-app/project/:project_id` | `PaymentsAppHandler.GetPaymentsAppByProjectID` | AdminMiddleware | JWT | Get payments app by project |
| POST | `/api/v1/payments-app` | `PaymentsAppHandler.CreatePaymentsApp` | AdminMiddleware | JWT | Create a payments app config |
| PUT | `/api/v1/payments-app/:id` | `PaymentsAppHandler.UpdatePaymentsApp` | AdminMiddleware | JWT | Update payments app config |

**PaymentsApp Model (confirmed):**
```
PaymentsApp {
  id:                       uint
  project_id:               uint
  sponsorship_percentage:   decimal.Decimal
  sponsorship_cut_off:      decimal.Decimal
  project:                  *ExternalPlatform
}
```

---

### 4.37 System Admin (SystemHandler)

**Route Setup:** `SetupSystemRoutes` (lines 13-68, 55 lines, confirmed)
**Handler:** `SystemHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/system/info` | `SystemHandler.SystemInfo` | AdminMiddleware | JWT | Get system information |
| POST | `/api/v1/system/validate-qr` | `SystemHandler.ValidateSystemQrCode` | AdminMiddleware | JWT | Validate system QR code for setup |
| GET | `/api/v1/system/workers` | `SystemHandler.GetAllWorkersHandler` | AdminMiddleware | JWT | List all background workers |
| GET | `/api/v1/system/workers/status` | `SystemHandler.GetAllWorkersStatus` | AdminMiddleware | JWT | Get status of all workers |
| POST | `/api/v1/system/worker/:name/restart` | `SystemHandler.RestartWorkerHandler` | AdminMiddleware | JWT | Restart a specific worker |
| POST | `/api/v1/system/worker/:name/start` | `SystemHandler.StartWorkerHandler` | AdminMiddleware | JWT | Start a specific worker |
| POST | `/api/v1/system/worker/:name/stop` | `SystemHandler.StopWorkerHandler` | AdminMiddleware | JWT | Stop a specific worker |
| POST | `/api/v1/system/workers/restart-all` | `SystemHandler.RestartAllWorkersHandler` | AdminMiddleware | JWT | Restart all workers |
| POST | `/api/v1/system/core/restart` | `SystemHandler.RestartCoreHandler` | AdminMiddleware | JWT | Restart the core API service |

**Worker Names (confirmed from processor package):**
- `account_processor` -- processes deposits, sweeps, withdrawals, rewards
- `eth_erc20_processor` -- monitors Ethereum + ERC20 blocks
- `base_erc20_processor` -- monitors Base chain blocks
- `polygon_erc20_processor` -- monitors Polygon blocks
- `bitcoin_processor` -- monitors Bitcoin blocks
- `deposit_processor` -- processes confirmed deposits
- `webhook_processor` -- delivers webhook notifications
- `email_processor` -- sends email notifications
- `erc20_sweep_approval_processor` -- processes ERC20 sweep approvals
- `broadcast_scw_deposit_wallet_processor` -- broadcasts SCW deposit wallet deployments

---

### 4.38 WebSocket (WebsocketHandler, WebsocketTokenHandler)

**Route Setup:** `WebsocketRouter` (lines 11-22, confirmed), `WebsocketTokenRouter` (lines 12-27, confirmed)
**Handler:** `WebsocketHandler`, `WebsocketTokenHandler`
**Middleware:** AdminMiddleware, WebsocketMiddleware, AllowOnlyLocalhostMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/ws/token` | `WebsocketTokenHandler.Create` | AdminMiddleware | JWT | Generate a WebSocket authentication token |
| GET | `/api/v1/ws/token` | `WebsocketTokenHandler.Get` | AdminMiddleware | JWT | Get current WebSocket token |
| GET | `/api/v1/ws/connect` | `WebsocketHandler.Connect` | WebsocketMiddleware | WS Token | Establish WebSocket connection |
| POST | `/api/v1/ws/trigger` | `WebsocketHandler.Trigger` | AllowOnlyLocalhostMiddleware | Internal | Trigger WebSocket event (internal only) |

**WebSocket Protocol:**
1. Client requests a WS token via `POST /api/v1/ws/token` (JWT authenticated)
2. Client connects to `GET /api/v1/ws/connect?token=<ws_token>`
3. WebSocket middleware validates the token
4. Connection established; client receives real-time updates for:
   - Payment status changes
   - New deposits
   - Sweep completions
   - Analytics updates

---

### 4.39 Nonce (NonceHandler)

**Route Setup:** `SetupNonceRoutes` (lines 13-22, 9 lines, confirmed)
**Handler:** `NonceHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/nonce/wallet/:wallet_id` | `NonceHandler.NonceForWallet` | AdminMiddleware | JWT | Get current nonce for a wallet (for transaction signing) |

*Response (inferred):*
```json
{
  "wallet_id": 1,
  "nonce": 42,
  "blockchain_code": "ETH"
}
```

---

### 4.40 Public API (PublicAPIHandler)

**Route Setup:** `SetupPublicAPIRoutes` (lines 11-24, 13 lines, confirmed)
**Handler:** `PublicAPIHandler`, `BlockchainCurrencyHandler`

These routes require NO authentication. They are publicly accessible.

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/ticker` | `PublicAPIHandler.Ticker` | None | None | Get current token prices and supported currencies |
| GET | `/api/v1/blockchain-currency/public` | `BlockchainCurrencyHandler.GetAllPublicDetailsOfBlockchainCurrencies` | None | None | Get public blockchain currency details |

**GET /api/v1/ticker (confirmed from PRODUCT_SPEC):**

*Response (inferred):*
```json
{
  "currencies": [
    {
      "code": "BTC",
      "name": "Bitcoin",
      "price_in_usd": 68500.00,
      "networks": ["BTC"]
    },
    {
      "code": "USDT",
      "name": "Tether",
      "price_in_usd": 1.00,
      "networks": ["ETH", "TRX", "POLYGON"]
    },
    {
      "code": "USDC",
      "name": "USD Coin",
      "price_in_usd": 1.00,
      "networks": ["ETH", "BASE"]
    }
  ],
  "updated_at": "2026-04-07T12:00:00Z"
}
```

---

### 4.41 Address Contract Signatures (AddressContractSignatureHandler)

**Route Setup:** `SetupAddressContractSignatureRoutes` (lines 13-21, 8 lines, confirmed)
**Handler:** `AddressContractSignatureHandler`
**Middleware:** AdminMiddleware
**Service:** `AddressContractSignatureServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| POST | `/api/v1/address-contract-signature` | `AddressContractSignatureHandler.Create` | AdminMiddleware | JWT | Create address contract signature for SCW deployment |

**AddressContractSignature Model (confirmed):**
```
AddressContractSignature {
  id:                    uint
  address:               string
  address_lower:         string
  blockchain_code:       string
  currency_code:         *string
  contract_type:         string
  status:                string
  signature:             string
  tx_hash:               *string
  block_number:          *uint64
  member_id:             *uint
  blockchain_id:         uint
  currency_id:           *uint
}
```

---

### 4.42 Account Rewards (AccountRewardHandler)

**Route Setup:** `SetupAccountRewardRoutes` (lines 13-31, 18 lines, confirmed)
**Handler:** `AccountRewardHandler`
**Middleware:** AdminMiddleware, ReferralServiceMiddleware
**Service:** `AccountRewardServiceImpl`

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/account-reward/member/:member_id` | `AccountRewardHandler.GetByMemberID` | AdminMiddleware, ReferralServiceMiddleware | JWT | Get rewards for a specific member |
| GET | `/api/v1/account-reward/unclaimed` | `AccountRewardHandler.UnclaimedRewardAmount` | AdminMiddleware, ReferralServiceMiddleware | JWT | Get total unclaimed reward amount |

**AccountReward Model (confirmed):**
```
AccountReward {
  id:                uint
  member_id:         uint
  currency_id:       uint
  balance:           decimal.Decimal
  locked:            decimal.Decimal
  currency_code:     string
  member:            *Member
}
```

---

### 4.43 Secrets Vault Activities (SecretsVaultActivityHandler)

**Route Setup:** `SetupWalletFunctionRoutes` (lines 13-22, 9 lines, confirmed)
**Handler:** `SecretsVaultActivityHandler`
**Middleware:** AdminMiddleware

| Method | Path | Handler | Middleware | Auth | Description |
|--------|------|---------|------------|------|-------------|
| GET | `/api/v1/vault-activity` | `SecretsVaultActivityHandler.GetAllVaultActivities` | AdminMiddleware | JWT | List all secrets vault activities |

**SecretsVault Model (confirmed):**
```
SecretsVault {
  id:                uint
  name:              string
  blockchain_code:   *string
  currency_code:     *string
  secret_type:       string
  secret_data:       string (encrypted)
  public_key:        *string
  member_id:         uint
  encryption_scheme: *string
  status:            string
}
```

---

### 4.44 Payment Links (v2)

Payment links live under `/api/v2` (module `backend/internal/links`, routes `backend/internal/api/routes_links.go`).
JSON is snake_case; money is a decimal string (numbers are accepted on input); request bodies are capped at 64 KiB and unknown fields are refused.
Errors are `{"error": message, "code": code, "field"?: string, "errors"?: [{code, field, message}]}`; `errors` lists every failed rule when there is more than one, and `code`/`field` repeat the first.
The full list of validation codes and when they apply is in `backend/internal/links/README.md`, "Validation".

#### Merchant routes (dashboard session or `X-API-Key`)

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/api/v2/links` | 201 link | Creates a draft. Absent fields take defaults. |
| GET | `/api/v2/links?status=&limit=&offset=` | 200 `{links, total}` | Newest first; `limit` 1-100 (default 25). |
| GET | `/api/v2/links/:id` | 200 link | |
| PATCH | `/api/v2/links/:id` | 200 link | Each top-level key sent replaces that field whole; other fields keep their values. Optional `If-Match: <revision>`; a concurrent change is 409 `link_conflict`. A published link may not change `amount_mode`, `amount`, `amount_min`, `amount_max`, `currency`, `methods`, `line_items` or `fee_bearer` (409 `link_published_immutable`) and is re-validated in full; use limits may not drop below payments taken (409 `use_limit_below_uses`). |
| DELETE | `/api/v2/links/:id` | 204 | Drafts only (409 `link_not_deletable`). |
| POST | `/api/v2/links/:id/publish` | 200 link | Draft or paused to active. Runs full validation; mints `short_code` on first publish. |
| POST | `/api/v2/links/:id/pause` | 200 link | Active to paused. |
| POST | `/api/v2/links/:id/archive` | 200 link | Draft, active or paused to archived (terminal). |
| POST | `/api/v2/links/:id/duplicate` | 201 link | New draft with the same form. |
| POST | `/api/v2/links/preview?link_id=` | 200 `{model, dropped_methods}` | Renders the posted form body exactly as `GET /public/links/:short_code` would once published, without storing it. `model` is the public render model (per-line totals, fees on methods, `merchant_name`); `dropped_methods` is `[{method, chain, asset, code, message}]` for methods checkout would leave out (`method_no_connector`, `method_no_fee_rule`, `surcharge_forbidden`, `surcharge_needs_quote`, `fee_exceeds_amount`). With `link_id`, status, uses and short code come from that link. Shape errors are 422 as on save. |
| GET | `/api/v2/links/options` | 200 `{environment, currencies, methods: [{method, chain, asset, currencies}]}` | What the form may offer in the process environment: each method the payment creator takes that has a connector and an active fee rule, and the link currencies it works in. Empty when the creator cannot list its offerings. |

Link body (create and PATCH), all optional on a draft:

```
title, description,
amount_mode: "fixed" | "customer" | "line_items",
amount, amount_min, amount_max, currency,
reference_id, metadata: {string: string}, category,
customer_field_policy: {name|email|phone: {mode: "required" | "optional" | "hidden", prefill?}},
billing_required, shipping_required, multi_use, use_limit,
methods: [{method: "card" | "upi" | "bank" | "crypto", chain?, asset?}],
capture_mode: "automatic" | "manual", three_ds_policy: "inherit" | "force",
chain_tolerance_bps, quote_expiry_seconds, fee_bearer: "merchant" | "customer",
success_mode: "message" | "redirect", success_url, success_message,
receipt_email, receipt_note, webhook_id, failure_retry, failure_message,
settlement_override: {kind: "fiat" | "crypto", destination_id, chain?, asset?} | null,
hold_in_asset, settlement_timing: "cycle" | "immediate",
expires_at, expires_after_payments,
logo_url, accent_color, language,
line_items: [{name, quantity, unit_price, tax_rate}],
questions: [{key, label, type: "text" | "select" | "checkbox", options?, required, per_order}]
```

Link response: the body fields above plus `id`, `status`, `environment` (`live` | `test`), `short_code`, `url` (`<CHECKOUT_BASE_URL>/l/<short_code>`, the QR payload; both null on a draft), `total` (the fixed amount or the server's line-item sum; null for customer-entered), `uses_count`, `revision`, `published_at`, `created_at`, `updated_at`, `merchant_name`, and on single-link responses `fee_preview`: `[{method, chain, asset, connector, rule_id, rule_version, fee_bearer, fee_currency, amount, fee, tax, customer_total, merchant_net, unavailable}]` (null in lists).

#### Public routes (no authentication, rate limited per IP)

`GET /api/v2/public/links/:short_code` (120 per minute) returns the render model: everything checkout needs and no internal id beyond the short code.
404 `link_not_found` for unknown codes and drafts, 410 `link_archived` for archived links; paused, expired and exhausted links return 200 with `available: false`.

```
short_code, url, available, unavailable_reason: null | "paused" | "expired" | "use_limit_reached" | "no_methods_available",
merchant_name, title, description,
amount_mode, amount, amount_min, amount_max, currency,
line_items: [{name, quantity, unit_price, tax_rate, subtotal, tax, total}], subtotal, tax_total,
customer_fields: {name|email|phone: {mode, prefill}},
billing_required, shipping_required,
questions: [{key, label, type, options, required, per_order}],
methods: [{method, chain, asset, fee, tax, customer_total}],
fee_bearer, chain_tolerance_bps, quote_expiry_seconds,
success_mode, success_message, failure_retry, failure_message, receipt_email, expires_at,
branding: {logo_url, accent_color, language}
```

`fee`, `tax` and `customer_total` on a method are set only for a customer-borne fee on a known amount in the link's currency; otherwise null.
A hidden field's `prefill` is always null.

`GET /api/v2/public/links/:short_code/qr.svg` (120 per minute) returns the link URL as an SVG QR code.

Public limits are per client IP, where the IP is the socket address unless the peer is in `TRUSTED_PROXIES`; they are kept in Redis and fall back to an in-process limit when Redis is unavailable.

`POST /api/v2/public/links/:short_code/pay` (20 per minute) requires an `Idempotency-Key` header (1-128 characters, scoped to the link).

```
{method: {method, chain?, asset?}, amount?, customer: {name?, email?, phone?},
 billing_address?: {line1, line2?, city, state?, postal_code, country}, shipping_address?: {...},
 answers?: {question_key: value}}
```

`amount` is accepted only on customer-entered links. Checkbox answers are `"true"` or `"false"`.
201 on a new payment, 200 on a replay of the same key and body:

```
{payment_reference, amount, currency, fee, tax, customer_total, fee_bearer,
 method: {method, chain, asset}, checkout_url, deposit_address, expires_at,
 success_redirect_url, replayed}
```

`success_redirect_url` is the merchant's URL with `reference_id` appended, for redirect links.
Refusals: 400 `idempotency_key_required`; 409 `idempotency_key_reused`, `payment_in_progress` (with `Retry-After`; retry the same key), `link_paused`, `link_use_limit_reached`, `link_environment_mismatch`; 410 `link_archived`, `link_expired`; 429 `open_payments_limit`, `rate_limit_exceeded`; 422 payer-input and pricing codes; 502 `payment_creation_failed` (the processor refused and the use was released, so the same key may retry).

---

## 5. Webhook Events

### Event Types

| Event | Description | Trigger |
|-------|-------------|---------|
| `payment.created` | Payment request created | New payment via API or dashboard |
| `payment.confirming` | Transaction detected on-chain | Block monitor detects matching deposit |
| `payment.confirmed` | Payment fully confirmed | Required confirmations reached |
| `payment.expired` | Payment expired without receipt | Expiry timer exceeded |
| `sweep.completed` | Funds swept to cold storage | SmartSweep contract execution confirmed |

### Webhook Payload Structure

```json
{
  "event": "payment.confirmed",
  "timestamp": "2026-04-07T12:30:00Z",
  "data": {
    "reference_id": "c80f5363-0397-4761-aa1a-3155c3a21470",
    "invoice_id": "inv_456",
    "customer_id": "cust_123",
    "customer_email": "user@example.com",
    "amount": "0.05",
    "amount_in_usd": 100.00,
    "currency_code": "ETH",
    "blockchain_code": "ETH",
    "status": "FILLED",
    "tx_hash": "0xabc123...",
    "confirmations": 12,
    "deposit_address": "0x1234567890abcdef...",
    "created_at": "2026-04-07T12:00:00Z",
    "confirmed_at": "2026-04-07T12:30:00Z"
  }
}
```

### Webhook Signature (HMAC-SHA256)

Each webhook delivery includes an HMAC-SHA256 signature for verification.

**Headers:**
```
X-Webhook-Signature: sha256=<hex_encoded_signature>
X-Webhook-Timestamp: 1712496600
X-Webhook-ID: <delivery_id>
```

**Signature Computation:**
```
signature = HMAC-SHA256(
  key: webhook.access_key,
  message: timestamp + "." + JSON.stringify(payload)
)
```

**Verification (pseudocode):**
```
expected = HMAC_SHA256(access_key, timestamp + "." + body)
valid = constant_time_compare(provided_signature, "sha256=" + hex(expected))
```

### Retry Strategy (confirmed from PRODUCT_SPEC)

Retries are exponentially increasing: 30m, 1h, 2h, 4h, 8h, 24h, 48h.

| Attempt | Delay | Cumulative |
|---------|-------|------------|
| 1 | Immediate | 0 |
| 2 | 30 minutes | 30m |
| 3 | 1 hour | 1h 30m |
| 4 | 2 hours | 3h 30m |
| 5 | 4 hours | 7h 30m |
| 6 | 8 hours | 15h 30m |
| 7 | 24 hours | 39h 30m |
| 8 | 48 hours | 87h 30m |

**WebhookDeliveryLog Model (confirmed):**
```
WebhookDeliveryLog {
  id:                  uint
  payment_request_id:  uint
  webhook_id:          uint
  status:              string ("pending" | "delivered" | "failed")
  attempt_count:       int
  next_retry_at:       *timestamp
  last_attempt_at:     *timestamp
  last_response_code:  *int
  last_error:          *string
}
```

### Idempotency

Each webhook delivery carries a unique `X-Webhook-ID`. Merchants should use this to deduplicate deliveries and ensure idempotent processing.

---

## 6. Error Response Format

### Standard Error Structure

All API errors are returned using the `PayramError` format (confirmed from `errors_handler.go`).

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request body",
    "details": [
      {
        "field": "amountInUSD",
        "message": "must be greater than 0"
      }
    ]
  }
}
```

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Request body validation failed |
| 400 | `BAD_REQUEST` | Malformed or invalid request |
| 401 | `UNAUTHORIZED` | Missing or invalid authentication |
| 401 | `TOKEN_EXPIRED` | JWT access token has expired |
| 401 | `INVALID_API_KEY` | API key is invalid or inactive |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 403 | `PLATFORM_ACCESS_DENIED` | No access to specified platform |
| 404 | `NOT_FOUND` | Resource not found |
| 404 | `PAYMENT_NOT_FOUND` | Payment request not found |
| 404 | `WALLET_NOT_FOUND` | Wallet not found |
| 409 | `CONFLICT` | Resource already exists |
| 409 | `DUPLICATE_PAYMENT` | Duplicate payment reference |
| 422 | `UNPROCESSABLE_ENTITY` | Request understood but cannot be processed |
| 429 | `RATE_LIMIT_EXCEEDED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
| 503 | `SERVICE_UNAVAILABLE` | Service temporarily unavailable |

### Validation Errors

The system uses custom validators (confirmed from `validators.go`, lines 187-256) for:
- `currency_type` -- validates currency type enum
- `blockchain_family` -- validates blockchain family enum
- `currency_code` -- validates currency code
- `password_complexity` -- enforces password requirements
- `identifier_format` -- validates identifier format
- `payment_channel_status` -- validates payment channel status enum
- `payment_channel_type` -- validates payment channel type enum
- `order_direction` -- validates sort order (asc/desc)
- `configuration_settings` -- validates configuration keys
- `secret_vault_type` -- validates vault type enum

---

## 7. Rate Limiting

### Strategy

Rate limiting is implemented using Redis as a token bucket. Limits are applied per:
- API Key (for merchant API requests)
- JWT session (for dashboard requests)
- IP address (for public endpoints)

### Default Limits (inferred)

| Endpoint Category | Rate Limit | Window |
|-------------------|-----------|--------|
| Public API (`/ticker`, `/blockchain-currency/public`) | 60 requests | per minute |
| Payment creation | 30 requests | per minute |
| Payment status lookup | 120 requests | per minute |
| Dashboard API | 120 requests | per minute |
| Webhook test | 5 requests | per minute |
| Auth endpoints | 10 requests | per minute |

### Rate Limit Headers

```
X-RateLimit-Limit: 120
X-RateLimit-Remaining: 115
X-RateLimit-Reset: 1712496660
Retry-After: 30
```

When rate limited, the API returns:
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please retry after 30 seconds."
  }
}
```

---

## 8. Pagination

### Standard Pagination Parameters

All list endpoints support consistent pagination via query parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 20 | Number of items per page (max 100) |
| `offset` | int | 0 | Number of items to skip |
| `sortBy` | string | `created_at` | Field to sort by |
| `order` | string | `desc` | Sort direction (`asc` or `desc`) |

### Pagination Response Format

```json
{
  "data": [...],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

### Implementation Details

Pagination is implemented by `ApplyPaginationConditions` (confirmed from `pagination_conditions.go`, lines 24-75). This function:
1. Parses `limit` and `offset` from query params
2. Clamps `limit` to max 100
3. Applies GORM `.Limit()` and `.Offset()` calls
4. Executes a count query for the `total` field
5. Builds the sort clause from `sortBy` and `order` parameters

### Specialized Search/Filter Parameters

Some endpoints use the `SearchParams` structure (confirmed from `payment_search_request.go`):
- `ParseSearchQuery` (lines 26-57) -- parses free-text search across multiple fields
- `BuildSortClause` (lines 57-88) -- constructs SQL ORDER BY with validation
- `ValidateSearchParams` (lines 88-102) -- validates sort column and order direction
- `FromJSON` (line 102) -- deserializes search params from JSON body

Member listing uses `MemberListParams` (confirmed from `members_list_request.go`):
- `ParseSearchQuery` (lines 20-40) -- parses member search
- `BuildSortClause` (lines 40-58) -- builds sort clause
- `ValidateSearchParams` (lines 58-65) -- validates params

---

## Appendix A: Complete Route Setup Functions

This table lists every route setup function with its confirmed line count, providing a measure of route complexity:

| Route Setup Function | File | Lines | Endpoint Count (est.) |
|----------------------|------|-------|----------------------|
| `SetupAuthRoutes` | auth_routes.go | 42 | 6 |
| `SetupMemberRoutes` | member_routes.go | 87 | 13 |
| `SetupOTPRoutes` | otp_routes.go | 27 | 4 |
| `SetupAPIKeyRoutes` | api_key_routes.go | 28 | 6 |
| `SetupRoleRoutes` | role_routes.go | 35 | 4 |
| `SetupPaymentsRoutes` | payment_routes.go | 72 | 11 |
| `SetupPaymentChannelRoutes` | payment_channel_routes.go | 60 | 10 |
| `SetupPaymentChannelProjectRoutes` | payment_channel_project_routes.go | 16 | 2 |
| `SetupDepositRoutes` | deposit_routes.go | 9 | 1 |
| `SetupDepositAddressRoutes` | deposit_address_routes.go | 7 | 1 |
| `SetupAddressRoutes` | address_routes.go | 38 | 5 |
| `SetupAddressPoolRoutes` | address_pool_routes.go | 15 | 2 |
| `SetupWalletRoutes` | wallet_routes.go | 98 | 15 |
| `SetupBlockchainRoutes` | blockchain_routes.go | 49 | 10 |
| `SetupBlockchainCurrencyRoutes` | blockchain_currency_routes.go | 34 | 5 |
| `SetupBlockchainFamilyRoutes` | blockchain_family_routes.go | 21 | 3 |
| `SetupBlockchainContractRoutes` | blockchain_contract_routes.go | 19 | 4 |
| `SetupContractAddressRoutes` | contract_address_routes.go | 15 | 3 |
| `SetupCurrencyRoutes` | currency_routes.go | 28 | 4 |
| `SetupSweepRoutes` | sweep_routes.go | 14 | 2 |
| `SetupSweepTransactionRoutes` | sweep_transaction_routes.go | 26 | 4 |
| `SetupSweepUtxoRoutes` | sweep_utxo_routes.go | 10 | 1 |
| `SetupWithdrawalRoutes` | withdrawal_routes.go | 92 | 12 |
| `SetupExternalPlatformRoutes` | external_platform_routes.go | 88 | 13 |
| `SetupExternalPlatformBlockchainCurrencyRoutes` | external_platform_blockchain_currency_routes.go | 18 | 2 |
| `SetupExternalPlatformWalletBlockchainFamilyRoutes` | external_platform_wallet_blockchain_family_routes.go | 23 | 3 |
| `SetupAnalyticsRoutes` | analytics_routes.go | 17 | 2 |
| `SetupAnalyticsReferralRoutes` | analytics_referral_routes.go | 10 | 1 |
| `SetupActivityLogRoutes` | activity_log_routes.go | 11 | 3 |
| `SetupReferralRoutes` | referral_routes.go | 182 | 26 |
| `SetupMissedDepositRoutes` | missed_deposit_routes.go | 33 | 4 |
| `SetupRecipientRoutes` | recipient_routes.go | 29 | 4 |
| `SetupConfigurationRoutes` | configuration_routes.go | 21 | 5 |
| `SetupSMTPConfigRoutes` | smtp_config_routes.go | 9 | 3 |
| `SetupOnramperPaymentsRoutes` | onramper_payments_routes.go | 8 | 2 |
| `SetupPaymentsAppRoutes` | payments_app_routes.go | 27 | 4 |
| `SetupSystemRoutes` | system_routes.go | 55 | 9 |
| `WebsocketRouter` | websocket_routes.go | 11 | 2 |
| `WebsocketTokenRouter` | websocket_token_routes.go | 15 | 2 |
| `SetupNonceRoutes` | nonce_routes.go | 9 | 1 |
| `SetupPublicAPIRoutes` | public_api_routes.go | 13 | 2 |
| `SetupAddressContractSignatureRoutes` | address_contract_signature_routes.go | 8 | 1 |
| `SetupAccountRewardRoutes` | account_reward_routes.go | 18 | 2 |
| `SetupWalletFunctionRoutes` | secrets_vault_activity_routes.go | 9 | 1 |
| `SetupSwaggerRoutes` | swagger_routes.go | 23 | (Swagger UI) |

**Total: 44 route groups, ~200+ individual endpoints**

---

## Appendix B: Swagger / OpenAPI

The system includes auto-generated Swagger documentation:

**Route Setup:** `SetupSwaggerRoutes` (lines 11-34, confirmed)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/swagger/*` | Swagger UI and JSON spec |

The Swagger routes include a redirect from `/swagger` to `/swagger/index.html` and serve the auto-generated OpenAPI specification.

---

## Appendix C: MCP Server Endpoints

The MCP (Model Context Protocol) server runs as a separate TypeScript/Node.js service on port 3333.

| Endpoint | Method | Protocol | Description |
|----------|--------|----------|-------------|
| `/mcp` | POST | StreamableHTTP | MCP interface |
| `/mcp/sse` | GET | SSE | Server-Sent Events streaming |
| `/healthz` | GET | HTTP | Health check |

### MCP Tools (confirmed from PRODUCT_SPEC)

| Tool | Description |
|------|-------------|
| `create-payee` | Create payment recipient |
| `send-payment` | Execute transaction |
| `get-balance` | Query account liquidity |
| `generate-invoice` | Create payment request |
| `test-connection` | Verify connectivity |
| `lookup-payment` | Get transaction details |
| `search-payments` | Query payment history |
| `get-daily-volume` | Aggregate daily metrics |
| `get-payment-summary` | Overview reporting |
| `get-unswept-balances` | Track pending sweeps |

### MCP Configuration

```json
{
  "mcpServers": {
    "payminto": {
      "url": "http://localhost:3333/mcp"
    }
  }
}
```

No API keys are needed for MCP. The MCP server connects to the Go API internally.
