# Payminto Backend — CLAUDE.md

## What This Is

The Payminto backend is a Go REST API that serves as the core engine for the entire payment gateway platform. It handles merchant authentication, payment lifecycle management, blockchain address generation and monitoring, wallet operations, webhook delivery, and sweep-to-cold-storage orchestration. It runs on port 8080 and is the single source of truth for all business logic. Frontend dashboard, MCP server, and widget all communicate with this service over HTTP. The module path is `github.com/payminto/payminto/backend`.

## Tech Stack

| Component | Version / Package |
|-----------|------------------|
| Language | Go 1.25.4 |
| HTTP Framework | `github.com/gin-gonic/gin` v1.12.0 |
| ORM | `gorm.io/gorm` v1.31.1 + `gorm.io/driver/postgres` v1.6.0 |
| Database | PostgreSQL 16 |
| Cache / Rate-limiting | `github.com/redis/go-redis/v9` v9.18.0 |
| Decimal arithmetic | `github.com/shopspring/decimal` v1.4.0 |
| JWT | `github.com/golang-jwt/jwt/v5` v5.3.1 |
| UUIDs | `github.com/google/uuid` v1.6.0 |
| Password hashing | `golang.org/x/crypto/bcrypt` |
| CORS | `github.com/gin-contrib/cors` v1.7.7 |
| JSON | `github.com/goccy/go-json` (fast, via Gin) |
| HD wallet | `github.com/tyler-smith/go-bip39` + `btcsuite/btcd/btcutil/hdkeychain` + `chaincfg` |
| Ethereum client | `github.com/ethereum/go-ethereum` (ethclient, core/types, accounts/abi, crypto) |
| Bitcoin | `btcsuite/btcd/btcutil/base58` + minimal JSON-RPC client (net/http) |
| Tron | Minimal TronGrid REST client (net/http) |
| SecretsVault KDF | `golang.org/x/crypto/scrypt` (N=2^15, r=8, p=1) |
| Migrations | `github.com/golang-migrate/migrate/v4` + `/database/postgres` + `/source/file` (currently wraps AutoMigrate for schema; raw SQL for seeds) |
| Integration test DB | `github.com/testcontainers/testcontainers-go` + `gorm.io/driver/sqlite` for lightweight unit tests |

## Build state (as of Phase C complete — 2026-04-07)

**Branch**: `feat/payminto-build` on `github.com/sagarjethi/payminto`

**Completed since the original 12-phase build**:
- **Phase A** (9 commits): 16 repository interfaces + impls, `ServiceRegistry` DI two-pass init (`internal/service/registry.go`), existing services refactored to inject repos, `Configuration` model + `EnforceModeMatch` boot check, golangci-lint config, testcontainer db helper
- **Phase B** (7 commits): BIP-32/39/44 HD wallet (`internal/crypto/hdwallet.go`, BIP-39 vector verified), `SecretsVaultService` with scrypt + AES-256 + activity log, `RPCNode` model/repo + `RPCPool` failover (`internal/blockchain/rpc_pool.go`), real Ethereum adapter (ethclient + ERC-20 decoding), real Bitcoin adapter (JSON-RPC + UTXO monitor), real Tron adapter (TronGrid REST + TRC-20 balanceOf), `IsMainnet()` on `ChainAdapter` interface, thread-safe `AdapterRegistry`
- **Phase C** (8 commits): 41 new models added across 6 domain groups, **73 models total** registered in AutoMigrate, `cmd/migrate` gained `--network=testnet|mainnet` flag, 10 seed SQL files split under `migrations/seeds/testnet/` and `migrations/seeds/mainnet/` (blockchains, blockchain_currencies, rpc_nodes, configurations, rbac)

**What's pending (Phases D–L)**:
- **D** (next): WalletService + AddressPoolService + AddressService + DepositAddressService + DepositService + address pool warmer worker + payment-flow integration + Sepolia E2E smoke
- **E**: BlockchainProcessor + 5 per-chain block processors + the 407-line `handleExtractedDeposit`
- **F**: SweepService + `AccountProcessorJob` (9 goroutines) + double-entry ledger primitives + SCW deposit broadcaster
- **G**: Auth refresh tokens + OTP + EventEmitter→Consumer email pipeline + WithdrawalService + WithdrawalProcessingService
- **H**: RBAC + ActivityLog + Member/Role/Permission services + Configuration runtime + GenericDataStore + Recipient + System worker control endpoints
- **I**: Webhooks CRUD + Analytics (430-line FetchData) + Ticker + Referral + AccountReward + RewardAccounting
- **J**: ExternalPlatform full + PaymentChannel + Onramper + MissedDeposit + Contract management + BlockchainManagement admin
- **K**: PublicAPI + WebSocket + Nonce + zerolog + Prometheus + Sentry + OpenAPI + graceful shutdown
- **L**: Hardening (vault rotation, blacklist, fuzz, security review, E2E testnet)

**Full plan file**: `/Users/sagarjethi/.claude/plans/immutable-drifting-leaf.md`

**Resume command** (if starting a new Claude Code session in this directory):
```bash
git pull
cd payminto/backend
go build ./... && go test ./... -count=1
# expect all 14 test packages green, then continue with Phase D per the plan file
```

## Directory Layout

```
backend/
├── cmd/
│   ├── server/          # main.go — application entrypoint, wires everything
│   └── migrate/         # main.go — runs GORM AutoMigrate standalone
├── internal/
│   ├── api/             # HTTP layer
│   │   ├── router.go    # Gin engine setup, route registration, middleware
│   │   ├── handler/     # One file per domain (payment_handler.go, auth_handler.go, …)
│   │   ├── middleware/  # JWT auth, API key auth, rate limiting, request logging
│   │   └── dto/         # Request/response structs (separate from models)
│   ├── service/         # Business logic (no HTTP, no DB — pure domain logic)
│   │   ├── auth_service.go
│   │   ├── payment_service.go
│   │   ├── onramp_service.go
│   │   ├── webhook_service.go
│   │   ├── deposit_processor.go   # worker: polls for new deposits
│   │   ├── payment_expiry.go      # worker: expires timed-out invoices
│   │   ├── webhook_processor.go   # worker: delivers webhooks with retry
│   │   └── manager.go             # worker orchestrator — starts/stops all workers
│   ├── repository/      # Database queries via GORM (one file per model group)
│   ├── models/          # GORM model structs (28 models)
│   │   ├── base.go              # PaymintoModel (ID, CreatedAt, UpdatedAt, DeletedAt)
│   │   ├── member.go            # User/merchant accounts
│   │   ├── payment.go           # Payment/invoice lifecycle
│   │   ├── wallet.go            # Hot wallet addresses per chain
│   │   ├── blockchain.go        # Chain config, block height tracking
│   │   ├── sweep.go             # Sweep batches and transactions
│   │   ├── webhook.go           # Webhook endpoints and delivery log
│   │   ├── api_key.go           # API key records (hashed)
│   │   ├── account.go           # Double-entry ledger entries
│   │   ├── currency.go          # Supported token config
│   │   ├── role.go              # RBAC roles
│   │   └── external_platform.go # Onramp/fiat integration records
│   ├── blockchain/      # Chain adapter interface + implementations
│   │   ├── adapter.go           # Adapter interface definition
│   │   ├── ethereum/            # ETH + ERC-20 (Base, Polygon share this)
│   │   ├── bitcoin/             # BTC p2pkh/p2wpkh address gen + monitoring
│   │   └── tron/                # TRX + TRC-20 (USDT)
│   ├── crypto/          # AES-256-GCM encryption for hot wallet private keys
│   ├── config/          # Config struct, loads from env vars
│   └── database/        # database.go — Connect() and AutoMigrate()
├── migrations/          # Versioned SQL files (supplementary to AutoMigrate)
├── .air.toml            # Live-reload config for `air`
├── Dockerfile
└── go.mod / go.sum
```

## Common Commands

```bash
# Development (live reload via air, from payminto/ monorepo root)
docker compose -f docker-compose.dev.yml up    # start postgres + redis only
cd backend && air -c .air.toml                 # hot-reload Go server

# Or run directly (requires DB + Redis already running)
go run cmd/server/main.go

# Run migrations standalone
go run cmd/migrate/main.go

# Build production binary
go build -o bin/server ./cmd/server/

# Run all tests
go test ./...

# Run tests for a specific package with verbose output
go test -v ./internal/service/...

# Run tests with race detector
go test -race ./...

# Format code
gofmt -w .
# or
goimports -w .

# Lint
golangci-lint run
```

## Environment Variables

The server reads configuration entirely from environment variables (no config files). See `payminto/.env.example` for the full list. Key vars:

```
DATABASE_URL=postgres://payminto:password@localhost:5432/payminto?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=<32-byte secret>
AES_KEY=<32-byte hex — encrypts hot wallet keys>
PORT=8080
COLD_WALLET_ETH=0x...          # SmartSweep destination
COLD_WALLET_BTC=bc1q...
COLD_WALLET_TRX=T...
```

## Code Conventions

### Model layer

Every model embeds `PaymintoModel` from `internal/models/base.go`:

```go
type PaymintoModel struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"createdAt"`
    UpdatedAt time.Time      `json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
```

Each model must implement `TableName() string` to avoid GORM's pluralisation guessing:

```go
func (Payment) TableName() string { return "payments" }
```

Use `*uint` for nullable foreign keys. Use `decimal.Decimal` (shopspring) for all monetary amounts — never `float64`. JSON tags are camelCase. Example:

```go
type Payment struct {
    PaymintoModel
    MerchantID  uint            `gorm:"not null;index" json:"merchantId"`
    Amount      decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
    Currency    string          `gorm:"size:10;not null" json:"currency"`
    WalletID    *uint           `gorm:"index" json:"walletId,omitempty"`
}
```

### Service layer

Services receive repository interfaces (not concrete types) for testability. They never import `net/http` — HTTP concerns live in handlers. Return `(result, error)` — never panic in service code.

### Handler layer

Handlers live in `internal/api/handler/`. They bind the request DTO, call a service method, and render JSON. Use Gin's `c.ShouldBindJSON()` for request binding with validation tags. Always return structured error responses:

```go
c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
```

### Authentication

Two auth mechanisms, both implemented as Gin middleware:
- **JWT**: `Authorization: Bearer <token>` — issued at login, 24h expiry by default
- **API key**: `X-API-Key: <raw key>` — stored SHA-256 hashed in DB, compared on each request

### Encryption

Hot wallet private keys are encrypted with AES-256-GCM before storage. The `internal/crypto/` package wraps encrypt/decrypt. The `AES_KEY` env var is the 32-byte key. Never store raw private keys in the database.

## Integration Points

| System | Direction | Protocol |
|--------|-----------|----------|
| Frontend dashboard | inbound | HTTP REST `GET /api/v1/...` |
| MCP server | inbound | HTTP REST (same API) |
| Widget | inbound | HTTP REST (creates payment) |
| PostgreSQL | outbound | GORM over TCP |
| Redis | outbound | go-redis over TCP |
| Blockchain nodes (ETH/BTC/TRX) | outbound | RPC (JSON-RPC / REST) |
| SmartSweep contract | outbound | EVM call via ethclient |
| Webhook endpoints (merchant URLs) | outbound | HTTP POST |

### REST API base: `http://localhost:8080/api/v1`

Key route groups (see `internal/api/router.go` for full list):
- `POST /auth/login`, `POST /auth/register`
- `GET /payments`, `POST /payments`, `GET /payments/:id`
- `GET /wallets`, `POST /wallets`
- `GET /sweeps`, `POST /sweeps/trigger`
- `GET /webhooks`, `POST /webhooks`, `DELETE /webhooks/:id`
- `GET /analytics/summary`
- `GET /settings/api-keys`, `POST /settings/api-keys`
- `GET /healthz` (unauthenticated health check)

## Key Files to Read First

1. `internal/config/config.go` — Config struct and env-var loading logic
2. `internal/database/database.go` — `Connect()` (DSN → *gorm.DB) and `AutoMigrate()` (all 28 models)
3. `internal/api/router.go` — all route registrations and middleware ordering
4. `internal/models/base.go` — `PaymintoModel` base + shared model patterns
5. `internal/service/payment_service.go` — core payment creation and state machine
6. `internal/service/auth_service.go` — login, register, JWT issuance, API key creation
7. `internal/blockchain/adapter.go` — `BlockchainAdapter` interface (GenerateAddress, GetBalance, MonitorDeposits)
8. `cmd/server/main.go` — startup wiring (config → db → services → router → workers)

## Background Workers

The worker manager (`internal/service/manager.go`) starts these goroutines at boot:

| Worker | File | Purpose |
|--------|------|---------|
| Deposit Processor | `deposit_processor.go` | Polls blockchain for incoming payments, updates payment status |
| Payment Expiry | `payment_expiry.go` | Marks invoices expired after TTL |
| Webhook Processor | `webhook_processor.go` | Delivers pending webhooks with exponential backoff retry |

Workers use a `context.Context` for graceful shutdown. The main process calls `manager.Stop()` on SIGTERM/SIGINT.

## Testing

Tests are colocated with source (`*_test.go`). The test base (`internal/api/base_test.go`, `internal/models/base_test.go`) sets up an in-memory test database using SQLite or a real Postgres test DB depending on env. Currently 67 tests across 13 packages.

```bash
# Run all with coverage
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```
