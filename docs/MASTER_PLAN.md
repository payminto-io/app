# PayRam Clone - Implementation Plan

**Version:** 1.0
**Date:** April 2, 2026
**Estimated Duration:** 16-20 weeks (4 engineers)

---

## Phase Overview

| Phase | Duration | Focus | Priority |
|-------|----------|-------|----------|
| **Phase 0** | Week 1 | Project Setup & Infrastructure | Critical |
| **Phase 1** | Week 2-4 | Core Payment Engine (Go Backend) | Critical |
| **Phase 2** | Week 3-5 | Blockchain Integration | Critical |
| **Phase 3** | Week 4-6 | SmartSweep Contracts | Critical |
| **Phase 4** | Week 5-8 | Merchant Dashboard (Next.js) | High |
| **Phase 5** | Week 7-9 | Webhook System & API Polish | High |
| **Phase 6** | Week 8-10 | Card-to-Crypto Onramp | High |
| **Phase 7** | Week 10-12 | MCP / AI Agent Server | Medium |
| **Phase 8** | Week 11-13 | Landing Page & Demo | Medium |
| **Phase 9** | Week 12-14 | Mobile App | Medium |
| **Phase 10** | Week 14-16 | Security Hardening & Audit | Critical |
| **Phase 11** | Week 16-18 | Documentation & Deployment Script | High |
| **Phase 12** | Week 18-20 | Testing, QA, Launch Prep | Critical |

---

## Phase 0: Project Setup & Infrastructure (Week 1)

### System Architect Perspective

**Goal:** Establish the foundation - repo, CI/CD, Docker, development environment.

#### Tasks

1. **Repository Setup**
   - Initialize monorepo structure
   - Setup Git with branch protection rules
   - Configure .gitignore, .editorconfig, .prettierrc

2. **Directory Structure**
   ```
   payramclone/
   ├── backend/               # Go API gateway
   │   ├── cmd/               # Entry points
   │   ├── internal/          # Private packages
   │   │   ├── api/           # HTTP handlers
   │   │   ├── service/       # Business logic
   │   │   ├── repository/    # Database layer
   │   │   ├── blockchain/    # Chain adapters
   │   │   ├── worker/        # Background jobs
   │   │   └── config/        # Configuration
   │   ├── pkg/               # Public packages
   │   ├── migrations/        # SQL migrations
   │   └── go.mod
   ├── frontend/              # Next.js dashboard
   │   ├── app/               # App router
   │   ├── components/        # React components
   │   ├── lib/               # Utilities
   │   └── package.json
   ├── mcp-server/            # MCP integration
   │   ├── src/
   │   └── package.json
   ├── contracts/             # Solidity smart contracts
   │   ├── src/
   │   ├── test/
   │   └── foundry.toml
   ├── landing/               # Marketing site
   │   └── (Next.js)
   ├── widget/                # Embeddable payment widget
   ├── scripts/               # Deployment scripts
   │   └── setup.sh
   ├── docker/                # Dockerfiles
   ├── docker-compose.yml
   ├── docker-compose.dev.yml
   └── docs/
   ```

3. **Docker Development Environment**
   - docker-compose.dev.yml with hot-reload
   - PostgreSQL 16 container
   - Redis 7 container
   - Go backend with air (live reload)
   - Next.js with turbopack

4. **CI/CD Pipeline**
   - GitHub Actions: lint, test, build
   - Docker image builds
   - Smart contract compilation + testing

### DevOps Tasks
- Set up development VPS (Ubuntu 22.04, 4 CPU, 8GB RAM)
- Configure Nginx reverse proxy template
- Set up Let's Encrypt automation (certbot)
- Configure monitoring stack (Prometheus + Grafana)

---

## Phase 1: Core Payment Engine (Week 2-4)

### Backend Engineer Perspective

**Goal:** Build the Go API server with payment CRUD, authentication, and core business logic.

#### Tasks

1. **Project Bootstrap (Go)**
   - Initialize Go module
   - Set up Gin/Echo HTTP framework
   - Configure structured logging (zerolog)
   - Set up database connection pool (pgx)
   - Set up Redis client

2. **Database Schema (PostgreSQL)**
   - Create migration files (golang-migrate)
   - Tables: merchants, payments, wallets, sweeps, webhooks, webhook_deliveries, api_keys, activity_logs
   - Indexes on: reference_id, deposit_address, status, merchant_id, created_at

3. **Authentication System**
   - API key generation (crypto/rand, SHA-256 hashed storage)
   - API key validation middleware
   - Rate limiting middleware (Redis-based, per-key)
   - CORS middleware

4. **Payment Service**
   - `POST /api/v1/payment` - Create payment
     - Validate input (amount, currency, blockchain)
     - Generate unique reference_id
     - Request deposit address from WalletService
     - Store payment record
     - Return payment object
   - `GET /api/v1/payment/:reference_id` - Get payment status
   - `GET /api/v1/payments` - List payments (with filters, pagination)
   - Payment expiry worker (background goroutine)

5. **Wallet Service**
   - Address derivation (BIP-32/39/44 via go-ethereum/hdwallet)
   - Address tracking (store mapping: address → payment_id)
   - Balance querying interface

6. **Configuration System**
   - Environment variables + config file
   - Supported chains/tokens configuration
   - Sweep thresholds
   - Webhook retry schedule

---

## Phase 2: Blockchain Integration (Week 3-5)

### Blockchain Engineer Perspective

**Goal:** Build chain-specific adapters for monitoring payments and managing addresses.

#### Tasks

1. **Chain Adapter Interface**
   ```go
   type ChainAdapter interface {
       GenerateAddress(index uint32) (string, error)
       MonitorBlocks(ctx context.Context, fromBlock uint64, addresses []string) (<-chan Transaction, error)
       GetConfirmations(txHash string) (uint64, error)
       GetBalance(address string, token string) (*big.Int, error)
       EstimateGas(tx TransactionParams) (*big.Int, error)
       BroadcastTransaction(signedTx []byte) (string, error)
   }
   ```

2. **Ethereum Adapter (+ Base + Polygon)**
   - go-ethereum client
   - ERC-20 token transfer monitoring
   - Block subscription (NewHead)
   - Transaction receipt parsing
   - Gas estimation

3. **Bitcoin Adapter**
   - btcd/rpcclient
   - UTXO monitoring
   - Address derivation (BIP-84 for SegWit)
   - Confirmation tracking

4. **Tron Adapter**
   - TronGrid API integration
   - TRC-20 token monitoring (USDT)
   - Energy/bandwidth estimation

5. **Block Monitor Worker**
   - Per-chain goroutine
   - Scan new blocks for matching deposit addresses
   - Update payment status on match
   - Track confirmations until threshold
   - Trigger webhook on confirmation

6. **RPC Node Management**
   - Health checking
   - Automatic failover
   - Request rate limiting
   - Connection pooling

---

## Phase 3: SmartSweep Contracts (Week 4-6)

### Blockchain Engineer Perspective

**Goal:** Build and audit smart contracts for automated fund sweeping to cold storage.

#### Tasks

1. **SmartSweep Contract (Solidity)**
   ```solidity
   // Key design:
   // - Cold wallet address set at deploy (IMMUTABLE)
   // - Only authorized sweeper can trigger
   // - Sweep destination cannot be changed
   // - Supports ETH + ERC-20 tokens
   ```
   - `sweep(address token, uint256 amount)` - Sweep specific amount
   - `sweepAll(address token)` - Sweep full balance
   - `sweepETH()` - Sweep native ETH
   - Access control (only authorized sweeper address)
   - Emergency pause mechanism

2. **Address Factory Contract**
   - Deploy minimal proxy contracts for deposit addresses
   - Gas-efficient CREATE2 deployment
   - Each proxy can receive tokens and be swept

3. **Testing (Foundry)**
   - Unit tests for all sweep scenarios
   - Fuzz testing for edge cases
   - Gas optimization tests
   - Access control tests
   - Reentrancy protection tests

4. **Deployment Scripts**
   - Deploy to testnets (Sepolia, Base Goerli)
   - Deploy to mainnets
   - Verification on Etherscan/Basescan

5. **Backend Integration**
   - Go bindings generation (abigen)
   - Sweep scheduler worker
   - Threshold-based auto-sweep
   - Gas estimation before sweep
   - Sweep transaction monitoring

---

## Phase 4: Merchant Dashboard (Week 5-8)

### Frontend Engineer Perspective

**Goal:** Build the full Next.js merchant dashboard.

#### Tasks

1. **Project Setup**
   - Next.js 14+ with App Router
   - TypeScript configuration
   - Tailwind CSS + shadcn/ui setup
   - Dark theme as default
   - TanStack Query for server state

2. **Authentication**
   - Email-based login
   - Session management
   - Protected route middleware

3. **Dashboard Page**
   - Metric cards (total settled, today's volume, active wallets, pending sweeps)
   - Volume chart (line chart, 30-day view)
   - Recent payments table (5 most recent)
   - Quick action buttons

4. **Payments Page**
   - Data table with sorting, filtering, pagination
   - Columns: ref_id, amount, currency, chain, status, time
   - Status badges (color-coded)
   - Click-through to detail view
   - Create payment form (modal/drawer)
   - Export to CSV

5. **Payment Detail Page**
   - Full payment info
   - QR code for deposit address
   - Blockchain explorer link
   - Confirmation progress
   - Webhook delivery log
   - Timeline view

6. **Wallets Page**
   - Per-chain wallet cards
   - Balance by token
   - Deposit address list
   - Cold wallet configuration
   - Sweep threshold settings

7. **Sweep Management**
   - Sweep history table
   - Status tracking
   - Manual sweep trigger
   - Gas cost display

8. **Analytics/Reports**
   - Volume over time (configurable range)
   - Revenue by currency
   - Revenue by blockchain
   - P&L summary
   - Export reports

9. **Settings**
   - Profile settings
   - Team management (invite users, roles)
   - API key management (create/revoke)
   - Webhook configuration
   - Notification preferences
   - Security (2FA setup)

10. **Responsive Design**
    - Mobile-optimized sidebar (collapsible)
    - Touch-friendly data tables
    - Mobile metric cards

---

## Phase 5: Webhook System & API Polish (Week 7-9)

### Backend Engineer Perspective

**Goal:** Production-grade webhook delivery system with retry logic.

#### Tasks

1. **Webhook Registration**
   - CRUD endpoints for webhook management
   - Event type filtering (payment.confirmed, sweep.completed, etc.)
   - Secret generation for signature verification

2. **Webhook Delivery Engine**
   - Queue-based delivery (Redis Streams)
   - HMAC-SHA256 signature on payload
   - Retry schedule: 30m, 1h, 2h, 4h, 8h, 24h, 48h
   - Idempotency keys
   - Delivery logging
   - Dead letter queue for persistent failures

3. **API Documentation**
   - OpenAPI 3.0 specification
   - Interactive API docs (Swagger UI)
   - Code examples (Node.js, Python, Go, cURL)
   - Webhook payload documentation

4. **SDK Development**
   - Node.js SDK (TypeScript)
   - Webhook signature verification helpers
   - Payment creation helpers
   - Status polling helpers

---

## Phase 6: Card-to-Crypto Onramp (Week 8-10)

### Backend + Frontend Engineer Perspective

**Goal:** Integrate card payment → crypto settlement via third-party onramp.

#### Tasks

1. **Onramp Provider Integration**
   - Evaluate providers: MoonPay, Transak, Ramp, Wyre
   - API integration with selected provider
   - Widget/iframe embed option
   - Redirect flow option

2. **Backend**
   - Onramp session creation
   - Payment matching (onramp completion → payment confirmation)
   - Status synchronization

3. **Frontend**
   - Card payment option in checkout
   - Onramp widget integration
   - Status tracking UI

---

## Phase 7: MCP / AI Agent Server (Week 10-12)

### Backend Engineer Perspective

**Goal:** Build MCP server for AI agent integration.

#### Tasks

1. **MCP Server (TypeScript/Node.js)**
   - StreamableHTTP transport
   - SSE event streaming
   - Health check endpoint
   - Tool discovery/registration

2. **MCP Tools Implementation**
   - `create-payee` - Create payment recipient
   - `send-payment` - Execute transaction
   - `get-balance` - Query liquidity
   - `generate-invoice` - Create payment request
   - `test-connection` - Verify connectivity
   - `lookup-payment` - Get transaction details
   - `search-payments` - Query history
   - `get-daily-volume` - Aggregate metrics
   - `get-payment-summary` - Overview reporting
   - `get-unswept-balances` - Track pending sweeps

3. **Agent Configuration**
   - Auto-generated mcp.json config
   - No API key requirement
   - MCP handshake auto-discovery

---

## Phase 8: Landing Page & Demo (Week 11-13)

### Frontend Engineer Perspective

**Goal:** Build the marketing site and interactive demo.

#### Tasks

1. **Landing Page (Next.js)**
   - Hero section with stats
   - Card-to-crypto section
   - Setup command section
   - AI agents section (comparison table)
   - Press coverage section
   - Feature grid (6 cards)
   - Mobile app section
   - Dashboard showcase (4 panels)
   - Supported crypto section
   - Testimonial
   - FAQ accordion
   - Blog preview
   - Footer

2. **Interactive Demo**
   - Amount selector ($10/$50/$100/$200)
   - Live payment simulation
   - Dashboard preview
   - Widget embed code display
   - API/SDK code samples
   - Webhook payload example

3. **Blog System**
   - MDX-based blog
   - Category filtering
   - SEO optimization

4. **Widget Development**
   - Standalone JavaScript widget
   - Configurable via data attributes
   - Responsive overlay/iframe

---

## Phase 9: Mobile App (Week 12-14)

### Mobile Engineer Perspective

**Goal:** Build iOS/Android business app.

#### Tasks

1. **React Native or Flutter app**
   - Wallet management across chains
   - Real-time balance updates
   - Transaction approval (individual + bulk)
   - Push notifications for pending payments
   - Biometric authentication
   - Sweep management
   - QR code scanning

---

## Phase 10: Security Hardening (Week 14-16)

### Security Engineer Perspective

**Goal:** Production-grade security across all layers.

#### Tasks

1. **Smart Contract Audit**
   - Internal audit pass
   - External audit (e.g., QuillAudits, Trail of Bits)
   - Fix findings, re-audit

2. **Application Security**
   - Input validation (all endpoints)
   - SQL injection prevention (parameterized queries)
   - XSS prevention (CSP headers)
   - CSRF protection
   - Rate limiting (per-IP + per-API-key)
   - Timing-attack resistant comparisons

3. **Infrastructure Security**
   - UFW firewall rules (only 80, 443)
   - fail2ban configuration
   - SSH hardening (key-only, no root)
   - Docker security (non-root containers, read-only filesystems)
   - Secret management (no plaintext secrets)

4. **Key Management**
   - AES-256 encryption for hot wallet keys
   - Key rotation procedures
   - Backup/recovery procedures
   - Zero key exposure verification

5. **Penetration Testing**
   - API penetration test
   - Smart contract exploit testing
   - Infrastructure scan

---

## Phase 11: Documentation & Deployment Script (Week 16-18)

### DevOps + Technical Writer Perspective

**Goal:** Build the one-command deployment and comprehensive docs.

#### Tasks

1. **Installation Script (setup.sh)**
   ```bash
   #!/bin/bash
   # 1. Check system requirements
   # 2. Install Docker + Docker Compose
   # 3. Interactive: domain, email, cold wallet address
   # 4. Generate configs (nginx, docker-compose, env)
   # 5. Pull images
   # 6. Provision SSL (certbot)
   # 7. Start services
   # 8. Run migrations
   # 9. Generate API key
   # 10. Health check
   # 11. Print access details
   ```

2. **Documentation Portal** (Docusaurus or similar)
   - Getting Started (5-minute quickstart)
   - Deployment Guide
   - Onboarding Guide (connect wallets)
   - API Reference
   - SDK Reference
   - Webhook Reference
   - MCP Integration Guide
   - Security Best Practices
   - FAQ

3. **Update System**
   - In-place update script
   - Database migration runner
   - Rollback support

---

## Phase 12: Testing, QA, Launch Prep (Week 18-20)

### QA + DevOps Perspective

**Goal:** Comprehensive testing and launch readiness.

#### Tasks

1. **Backend Testing**
   - Unit tests (>80% coverage)
   - Integration tests (API, blockchain, webhooks)
   - Load testing (payment processing throughput)

2. **Frontend Testing**
   - Component tests (React Testing Library)
   - E2E tests (Playwright)
   - Visual regression tests

3. **Smart Contract Testing**
   - Foundry test suite
   - Mainnet fork testing
   - Gas optimization verification

4. **End-to-End Testing**
   - Full payment flow on testnets
   - Multi-chain payment testing
   - Sweep testing
   - Webhook delivery testing
   - Widget integration testing
   - MCP agent testing

5. **Launch Checklist**
   - [ ] All tests passing
   - [ ] Security audit complete
   - [ ] Monitoring alerts configured
   - [ ] Backup/recovery tested
   - [ ] Documentation complete
   - [ ] Installation script tested on fresh VPS
   - [ ] Performance benchmarks met
   - [ ] Legal review (terms, privacy policy)

---

## Team Structure (Recommended)

| Role | Count | Focus |
|------|-------|-------|
| **Tech Lead / System Architect** | 1 | Architecture, code review, blockchain design |
| **Backend Engineer (Go)** | 1 | API, services, workers, database |
| **Blockchain Engineer** | 1 | Smart contracts, chain adapters, security |
| **Frontend Engineer** | 1 | Dashboard, landing page, widget |
| **DevOps** | 0.5 | Infrastructure, deployment, monitoring |
| **QA** | 0.5 | Testing, security testing |

---

## Critical Dependencies & Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Smart contract vulnerability | Critical | External audit, formal verification |
| Blockchain RPC downtime | High | Multi-provider failover |
| Key management failure | Critical | Hardware security modules, backup procedures |
| Regulatory changes | Medium | Modular compliance layer |
| Onramp provider issues | Medium | Multi-provider support |
| Performance bottleneck | Medium | Load testing, horizontal scaling |

---

## Budget Estimate

| Category | Monthly Cost | Notes |
|----------|-------------|-------|
| Development VPS | $50 | 4 CPU, 8GB RAM |
| Production VPS | $50-100 | 8 CPU, 8GB RAM, 100GB SSD |
| RPC Node Access | $50-200 | Alchemy/Infura growth plan |
| Domain + SSL | $15 | Annual |
| Smart Contract Audit | $10,000-30,000 | One-time |
| Monitoring (Grafana Cloud) | $0-50 | Free tier available |
| **Total Monthly (post-launch)** | **$165-400** | |
| **Total One-Time** | **$10,000-30,000** | Audit |
