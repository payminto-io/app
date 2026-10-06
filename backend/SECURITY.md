# Payminto Backend — Security Checklist

Status column legend: ✅ implemented, 🟡 partial, 🔴 missing.

## Credentials & secrets at rest

- ✅ API keys SHA-256 hashed (plaintext returned exactly once)
- ✅ Passwords bcrypt (default cost)
- ✅ JWT refresh tokens hashed at rest, rotation-family reuse detection
- ✅ Private keys encrypted in `secrets_vault` with scrypt-derived AES-256-GCM
- ✅ Per-vault random salt on first boot

## Input validation

- ✅ Gin `ShouldBindJSON` binding tags on every request DTO
- ✅ All SQL via GORM parameter binding (no string concatenation)
- ✅ Decimal amounts use `shopspring/decimal` — never `float64`
- ✅ Destination address blacklist check during withdrawal creation
- 🟡 Per-chain address regex validation (deferred — relies on chain adapters)

## Authorization

- ✅ JWT middleware for dashboard routes, API-key middleware for server-to-server
- ✅ RBAC `RequirePermission` middleware with cached lookups + invalidation
- ✅ Tenant isolation via `externalPlatformID` in every repo WHERE clause
- ✅ Admin endpoints gated by `system.admin`
- ✅ Public widget endpoints return narrowed projections (no tenant fields)

## Cryptographic correctness

- ✅ HMAC-SHA256 webhook signature verification is constant-time (`hmac.Equal`)
- ✅ Fuzz test (`FuzzOnramperSignatureVerify`) proves no false positives
- ✅ JWT signing uses HS256 with ≥32-byte secret (enforced at boot)
- ✅ BIP-32/39/44 HD wallet derivation verified against BIP-39 reference vectors

## Race / concurrency correctness

- ✅ Withdrawal state machine uses atomic conditional UPDATEs (`rowsAffected` guards)
- ✅ Sweep state transitions atomic
- ✅ UTXO `MarkSpent` atomic conditional UPDATE
- ✅ OTP attempt counter atomic increment
- ✅ Nonce `Consume` atomic
- ✅ Full suite passes under `go test -race`

## Rate limiting & abuse

- ✅ Global rate-limit middleware (Redis-backed in production)
- ✅ OTP max 5 attempts per code
- 🟡 Per-route rate limits (deferred — single global limiter today)

## Observability

- ✅ Structured logging via `log/slog` JSON handler
- ✅ Request ID middleware propagates `X-Request-ID` into log context
- ✅ `/livez` (always 200) and `/readyz` (DB probe)
- ✅ Graceful shutdown: SIGINT/SIGTERM → 30s drain → worker stop
- ✅ Activity log middleware redacts sensitive body fields
- 🔴 Prometheus `/metrics` (deferred)
- 🔴 Sentry panic capture (deferred)

## Mainnet pre-flight (run before flipping `BLOCKCHAIN_NETWORK_TYPE=mainnet`)

1. Rotate `AES_KEY` and `JWT_SECRET` to fresh ≥32-byte random values — never reuse dev secrets.
2. Verify `configurations.mode = 'mainnet'` in the target database. The boot gate refuses to start otherwise.
3. Confirm at least 2 healthy mainnet RPC nodes per `blockchains` row.
4. Seed real USDT/USDC contract addresses and mainnet cold-wallet destinations in `blockchain_currencies`.
5. Initialize `SecretsVault` with a production passphrase; back up the vault; test recovery.
6. Populate `address_blacklist` configuration key with the OFAC + Chainalysis-derived block list.
7. Configure per-platform withdrawal limits via `external_platform_blockchain_currencies`.
8. Set `gin.Mode = release`, rate limiter pointed at real Redis.
9. Run an end-to-end $1 USDC payment → sweep → withdrawal cycle on Base mainnet and verify ledger balances to zero.
10. Wire PagerDuty / Slack alerts for worker failures, RPC outages, ledger drift, missed deposits.
