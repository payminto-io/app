# PayRam Clone - System Architecture Diagrams

**Version:** 1.0
**Date:** April 2, 2026

---

## 1. High-Level System Architecture

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│                              PAYRAM SYSTEM ARCHITECTURE                               │
│                                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐               │
│  │  Customer    │  │  Merchant   │  │  AI Agent    │  │  Mobile App │               │
│  │  Browser     │  │  Dashboard  │  │  (MCP)       │  │  (iOS/Andr) │               │
│  └──────┬──────┘  └──────┬──────┘  └──────┬───────┘  └──────┬──────┘               │
│         │                │                │                  │                        │
│         ▼                ▼                ▼                  ▼                        │
│  ┌──────────────────────────────────────────────────────────────────┐                │
│  │                     NGINX / REVERSE PROXY                        │                │
│  │              (SSL Termination, Rate Limiting)                    │                │
│  └──────────────────────────┬───────────────────────────────────────┘                │
│                             │                                                        │
│         ┌───────────────────┼───────────────────┐                                   │
│         ▼                   ▼                   ▼                                     │
│  ┌─────────────┐   ┌──────────────┐    ┌──────────────┐                             │
│  │  Next.js    │   │  Go API      │    │  MCP Server  │                             │
│  │  Frontend   │   │  Gateway     │    │  (TypeScript) │                             │
│  │  (SSR/SPA)  │   │  (Core)      │    │  Port 3333   │                             │
│  │  Port 3000  │   │  Port 8080   │    │              │                             │
│  └─────────────┘   └──────┬───────┘    └──────┬───────┘                             │
│                           │                    │                                      │
│                    ┌──────┴────────────────────┘                                     │
│                    ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────┐                 │
│  │                    SERVICE LAYER (Go)                            │                 │
│  │                                                                 │                 │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────────┐  │                 │
│  │  │ Payment   │ │ Wallet    │ │ Webhook   │ │ Sweep         │  │                 │
│  │  │ Service   │ │ Service   │ │ Service   │ │ Service       │  │                 │
│  │  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └───────┬───────┘  │                 │
│  │        │             │             │               │           │                 │
│  │  ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴───────┐  │                 │
│  │  │ Analytics │ │ Auth      │ │ Activity  │ │ Notification │  │                 │
│  │  │ Service   │ │ Service   │ │ Logger    │ │ Service      │  │                 │
│  │  └───────────┘ └───────────┘ └───────────┘ └──────────────┘  │                 │
│  └──────────────────────┬──────────────────────────────────────────┘                 │
│                         │                                                            │
│         ┌───────────────┼───────────────┐                                            │
│         ▼               ▼               ▼                                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐                                    │
│  │ PostgreSQL  │ │   Redis     │ │ Blockchain  │                                    │
│  │ (Primary DB)│ │ (Cache/     │ │ RPC Nodes   │                                    │
│  │             │ │  Queue)     │ │ (Multi-     │                                    │
│  │             │ │             │ │  Chain)     │                                    │
│  └─────────────┘ └─────────────┘ └──────┬──────┘                                    │
│                                         │                                            │
│                          ┌──────────────┼──────────────┐                             │
│                          ▼              ▼              ▼                              │
│                   ┌──────────┐  ┌──────────┐  ┌──────────┐                          │
│                   │ Ethereum │  │ Bitcoin  │  │ Tron     │  ...more chains          │
│                   │ + Base   │  │          │  │          │                           │
│                   │ + Polygon│  │          │  │          │                           │
│                   └──────────┘  └──────────┘  └──────────┘                          │
│                                                                                      │
│  ┌──────────────────────────────────────────────────────────────────┐                │
│  │                    SMART CONTRACTS (On-Chain)                     │                │
│  │                                                                  │                │
│  │  ┌─────────────────────┐    ┌─────────────────────────┐         │                │
│  │  │   SmartSweep        │    │   Address Factory        │         │                │
│  │  │   - Hardcoded cold  │    │   - HD derivation        │         │                │
│  │  │     wallet dest     │    │   - Unique per txn       │         │                │
│  │  │   - Auto-sweep      │    │   - Deterministic        │         │                │
│  │  │   - Immutable logic │    │                          │         │                │
│  │  └─────────────────────┘    └─────────────────────────┘         │                │
│  └──────────────────────────────────────────────────────────────────┘                │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Backend Architecture (Go)

```
┌─────────────────────────────────────────────────────────────────┐
│                     GO BACKEND ARCHITECTURE                      │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    HTTP Router (Gin/Echo)                │    │
│  │                                                         │    │
│  │  /api/v1/payment          POST  → PaymentController     │    │
│  │  /api/v1/payment/:ref_id  GET   → PaymentController     │    │
│  │  /api/v1/deposit-address  POST  → AddressController     │    │
│  │  /api/v1/wallets          GET   → WalletController      │    │
│  │  /api/v1/sweeps           GET   → SweepController       │    │
│  │  /api/v1/webhooks         CRUD  → WebhookController     │    │
│  │  /api/v1/analytics        GET   → AnalyticsController   │    │
│  │  /api/v1/auth             POST  → AuthController        │    │
│  └────────────────────────┬────────────────────────────────┘    │
│                           │                                     │
│  ┌────────────────────────▼────────────────────────────────┐    │
│  │                   MIDDLEWARE CHAIN                        │    │
│  │                                                         │    │
│  │  ┌────────┐ ┌──────┐ ┌──────────┐ ┌─────────┐         │    │
│  │  │API Key │→│ CORS │→│Rate Limit│→│ Logging │→ Handler │    │
│  │  │Auth    │ │      │ │ (Redis)  │ │         │          │    │
│  │  └────────┘ └──────┘ └──────────┘ └─────────┘         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    SERVICE LAYER                          │    │
│  │                                                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ PaymentService                                   │    │    │
│  │  │  - CreatePayment(amount, currency, customer)     │    │    │
│  │  │  - GetPayment(referenceId)                       │    │    │
│  │  │  - UpdatePaymentStatus(id, status, txid)         │    │    │
│  │  │  - ListPayments(filters, pagination)             │    │    │
│  │  │  - ExpireStalePayments()                         │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                                                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ WalletService                                    │    │    │
│  │  │  - GenerateDepositAddress(blockchain, payment)   │    │    │
│  │  │  - GetBalance(walletId)                          │    │    │
│  │  │  - ListWallets(merchantId)                       │    │    │
│  │  │  - DeriveAddress(masterSeed, index, chain)       │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                                                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ SweepService                                     │    │    │
│  │  │  - InitiateSweep(walletId, threshold)            │    │    │
│  │  │  - ExecuteSweep(sweepId)                         │    │    │
│  │  │  - GetSweepStatus(sweepId)                       │    │    │
│  │  │  - ScheduleAutoSweep(config)                     │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                                                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ WebhookService                                   │    │    │
│  │  │  - RegisterWebhook(url, events, secret)          │    │    │
│  │  │  - DeliverWebhook(paymentId, event)              │    │    │
│  │  │  - RetryFailedDeliveries()                       │    │    │
│  │  │  - VerifySignature(payload, signature)           │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                                                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ BlockchainService (per chain)                    │    │    │
│  │  │  - MonitorBlocks(fromBlock)                      │    │    │
│  │  │  - CheckTransaction(txHash)                      │    │    │
│  │  │  - GetConfirmations(txHash)                      │    │    │
│  │  │  - EstimateGas(txParams)                         │    │    │
│  │  │  - BroadcastTransaction(signedTx)                │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 BACKGROUND WORKERS                        │    │
│  │                                                         │    │
│  │  ┌────────────────┐  ┌──────────────┐  ┌─────────────┐ │    │
│  │  │ Block Monitor  │  │ Sweep        │  │ Webhook     │ │    │
│  │  │ (per chain)    │  │ Scheduler    │  │ Retry       │ │    │
│  │  │ - Poll blocks  │  │ - Check      │  │ - Retry     │ │    │
│  │  │ - Match addrs  │  │   thresholds │  │   failed    │ │    │
│  │  │ - Update status│  │ - Execute    │  │   deliveries│ │    │
│  │  └────────────────┘  └──────────────┘  └─────────────┘ │    │
│  │                                                         │    │
│  │  ┌────────────────┐  ┌──────────────┐                  │    │
│  │  │ Payment Expiry │  │ Analytics    │                  │    │
│  │  │ - Expire stale │  │ Aggregator   │                  │    │
│  │  │   payments     │  │ - Daily vol  │                  │    │
│  │  │                │  │ - P&L calc   │                  │    │
│  │  └────────────────┘  └──────────────┘                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    DATA LAYER                            │    │
│  │                                                         │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌────────────────┐  │    │
│  │  │ PostgreSQL  │  │ Redis       │  │ Encrypted      │  │    │
│  │  │             │  │             │  │ Key Store      │  │    │
│  │  │ - Payments  │  │ - Sessions  │  │                │  │    │
│  │  │ - Wallets   │  │ - Rate lim  │  │ - Hot wallet   │  │    │
│  │  │ - Sweeps    │  │ - Pub/Sub   │  │   (AES-256)    │  │    │
│  │  │ - Webhooks  │  │ - Job queue │  │ - NEVER deposit│  │    │
│  │  │ - Merchants │  │ - Cache     │  │   keys         │  │    │
│  │  │ - Analytics │  │             │  │                │  │    │
│  │  └─────────────┘  └─────────────┘  └────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Frontend Architecture (Next.js)

```
┌─────────────────────────────────────────────────────────────────┐
│                   FRONTEND ARCHITECTURE (Next.js)                │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    APP ROUTER                            │    │
│  │                                                         │    │
│  │  /                    → Landing Page (Marketing)        │    │
│  │  /demo                → Interactive Demo                │    │
│  │  /blog                → Blog (SSG)                      │    │
│  │  /blog/[slug]         → Blog Post (SSG)                 │    │
│  │  /industry/[slug]     → Industry Pages (SSG)            │    │
│  │                                                         │    │
│  │  /login               → Email Login                     │    │
│  │  /dashboard           → Dashboard Overview              │    │
│  │  /payments            → Payment List                    │    │
│  │  /payments/[id]       → Payment Detail                  │    │
│  │  /payments/create     → Create Payment                  │    │
│  │  /wallets             → Wallet Management               │    │
│  │  /sweeps              → Sweep Management                │    │
│  │  /transactions        → Transaction History             │    │
│  │  /analytics           → Charts & Reports                │    │
│  │  /recipients          → Payee Management                │    │
│  │  /settings            → Settings (Profile/Team/API)     │    │
│  │  /settings/api-keys   → API Key Management              │    │
│  │  /settings/webhooks   → Webhook Configuration           │    │
│  │  /settings/team       → Team Management                 │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   COMPONENT LIBRARY                      │    │
│  │                                                         │    │
│  │  Layout:                                                │    │
│  │  ├── AppShell (sidebar + topbar)                        │    │
│  │  ├── Sidebar (collapsible, icons + labels)              │    │
│  │  ├── TopBar (search, notifications, profile)            │    │
│  │  └── Footer                                             │    │
│  │                                                         │    │
│  │  Data Display:                                          │    │
│  │  ├── DataTable (sortable, filterable, paginated)        │    │
│  │  ├── MetricCard (icon + value + trend)                  │    │
│  │  ├── ChartWidget (line, bar, pie via Recharts)          │    │
│  │  ├── StatusBadge (created/confirming/confirmed/expired) │    │
│  │  └── AddressDisplay (truncated + copy button)           │    │
│  │                                                         │    │
│  │  Forms:                                                 │    │
│  │  ├── PaymentForm (amount, currency, customer)           │    │
│  │  ├── WebhookForm (url, events, secret)                  │    │
│  │  ├── WalletForm (chain, address)                        │    │
│  │  └── SettingsForm (profile, team, notifications)        │    │
│  │                                                         │    │
│  │  Payment:                                               │    │
│  │  ├── PaymentWidget (embeddable checkout)                │    │
│  │  ├── QRCode (deposit address)                           │    │
│  │  ├── AmountSelector (preset amounts)                    │    │
│  │  └── PaymentStatus (real-time update)                   │    │
│  │                                                         │    │
│  │  Blockchain:                                            │    │
│  │  ├── ChainIcon (per-chain logos)                        │    │
│  │  ├── TokenIcon (per-token logos)                        │    │
│  │  ├── TxHashLink (explorer deep link)                    │    │
│  │  └── WalletBalance (multi-token display)                │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   STATE MANAGEMENT                       │    │
│  │                                                         │    │
│  │  TanStack Query (React Query):                          │    │
│  │  ├── usePayments() - payment list with filters          │    │
│  │  ├── usePayment(id) - single payment detail             │    │
│  │  ├── useWallets() - wallet balances                     │    │
│  │  ├── useSweeps() - sweep history                        │    │
│  │  ├── useAnalytics() - dashboard metrics                 │    │
│  │  └── useWebhookLogs() - delivery status                 │    │
│  │                                                         │    │
│  │  WebSocket / SSE:                                       │    │
│  │  ├── Real-time payment status updates                   │    │
│  │  ├── Live transaction feed                              │    │
│  │  └── Notification stream                                │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   DESIGN SYSTEM                          │    │
│  │                                                         │    │
│  │  Theme: Dark mode primary (light mode optional)         │    │
│  │  Colors: Green (#01e46f), Dark BG (#0a0a0a), Pink       │    │
│  │  Typography: Inter / system sans-serif                  │    │
│  │  Components: shadcn/ui + Tailwind CSS                   │    │
│  │  Icons: Lucide React                                    │    │
│  │  Charts: Recharts                                       │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. DevOps / Infrastructure Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  DEVOPS / DEPLOYMENT ARCHITECTURE                 │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              SINGLE VPS DEPLOYMENT (Default)             │    │
│  │                  Ubuntu 22.04+                           │    │
│  │                  4-8 CPU, 8GB RAM, 100GB SSD            │    │
│  │                                                         │    │
│  │  ┌──────────────────────────────────────────────┐       │    │
│  │  │              Docker Compose Stack             │       │    │
│  │  │                                              │       │    │
│  │  │  ┌──────────────┐  ┌──────────────┐         │       │    │
│  │  │  │ nginx:latest │  │ certbot      │         │       │    │
│  │  │  │ (reverse     │  │ (Let's       │         │       │    │
│  │  │  │  proxy + SSL)│  │  Encrypt)    │         │       │    │
│  │  │  └──────┬───────┘  └──────────────┘         │       │    │
│  │  │         │                                    │       │    │
│  │  │  ┌──────▼───────┐  ┌──────────────┐         │       │    │
│  │  │  │ payram-api   │  │ payram-mcp   │         │       │    │
│  │  │  │ (Go binary)  │  │ (Node.js)    │         │       │    │
│  │  │  │ Port 8080    │  │ Port 3333    │         │       │    │
│  │  │  └──────┬───────┘  └──────┬───────┘         │       │    │
│  │  │         │                 │                  │       │    │
│  │  │  ┌──────▼───────┐  ┌─────▼────────┐        │       │    │
│  │  │  │ payram-web   │  │ payram-      │        │       │    │
│  │  │  │ (Next.js)    │  │ workers      │        │       │    │
│  │  │  │ Port 3000    │  │ (Go, bg jobs)│        │       │    │
│  │  │  └──────────────┘  └──────────────┘        │       │    │
│  │  │                                              │       │    │
│  │  │  ┌──────────────┐  ┌──────────────┐         │       │    │
│  │  │  │ postgres:16  │  │ redis:7      │         │       │    │
│  │  │  │ Port 5432    │  │ Port 6379    │         │       │    │
│  │  │  └──────────────┘  └──────────────┘         │       │    │
│  │  │                                              │       │    │
│  │  │  Volumes:                                    │       │    │
│  │  │  - pg_data (database)                        │       │    │
│  │  │  - redis_data (cache)                        │       │    │
│  │  │  - certs (SSL certificates)                  │       │    │
│  │  │  - keys (encrypted wallet keys)              │       │    │
│  │  │  - logs (application logs)                   │       │    │
│  │  └──────────────────────────────────────────────┘       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              ONE-LINE INSTALL SCRIPT                      │    │
│  │                                                         │    │
│  │  /bin/bash -c "$(curl -fsSL https://payram.com/setup)"  │    │
│  │                                                         │    │
│  │  Script does:                                           │    │
│  │  1. Check system requirements (OS, CPU, RAM)            │    │
│  │  2. Install Docker + Docker Compose                     │    │
│  │  3. Pull PayRam images                                  │    │
│  │  4. Generate configuration (domain, email, wallet)      │    │
│  │  5. Provision SSL via Let's Encrypt                     │    │
│  │  6. Start all services                                  │    │
│  │  7. Run health checks                                   │    │
│  │  8. Print dashboard URL + API key                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              MONITORING & OBSERVABILITY                   │    │
│  │                                                         │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │    │
│  │  │ Prometheus   │  │ Grafana      │  │ Loki         │  │    │
│  │  │              │  │              │  │              │  │    │
│  │  │ Metrics:     │  │ Dashboards:  │  │ Logs:        │  │    │
│  │  │ - API latency│  │ - System     │  │ - API logs   │  │    │
│  │  │ - Payment    │  │ - Payments   │  │ - Blockchain │  │    │
│  │  │   volume     │  │ - Blockchain │  │   events     │  │    │
│  │  │ - Block sync │  │ - Sweeps     │  │ - Errors     │  │    │
│  │  │ - Sweep      │  │ - Webhooks   │  │ - Webhook    │  │    │
│  │  │   status     │  │              │  │   delivery   │  │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              BACKUP & DISASTER RECOVERY                   │    │
│  │                                                         │    │
│  │  - Automated PostgreSQL backups (pg_dump, daily)        │    │
│  │  - Redis RDB snapshots                                  │    │
│  │  - Encrypted key backup to separate storage             │    │
│  │  - Volume snapshots (provider-dependent)                │    │
│  │  - Recovery playbook in documentation                   │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Blockchain Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  BLOCKCHAIN INTEGRATION LAYER                    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │            HD WALLET / KEY MANAGEMENT                    │    │
│  │                                                         │    │
│  │  Master Seed (BIP-39)                                   │    │
│  │       │  ⚠️ PERMANENTLY OFFLINE                         │    │
│  │       ▼                                                 │    │
│  │  Master Key (BIP-32)                                    │    │
│  │       │                                                 │    │
│  │       ├── m/44'/60'/0'/0/0  → ETH deposit addr #0      │    │
│  │       ├── m/44'/60'/0'/0/1  → ETH deposit addr #1      │    │
│  │       ├── m/44'/60'/0'/0/N  → ETH deposit addr #N      │    │
│  │       │                                                 │    │
│  │       ├── m/44'/0'/0'/0/0   → BTC deposit addr #0      │    │
│  │       ├── m/44'/0'/0'/0/N   → BTC deposit addr #N      │    │
│  │       │                                                 │    │
│  │       ├── m/44'/195'/0'/0/0 → TRX deposit addr #0      │    │
│  │       └── ...per chain derivation paths                 │    │
│  │                                                         │    │
│  │  ⚡ Server only stores: address index counter           │    │
│  │  ⚡ Server NEVER stores: private keys, seed phrase      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │            BLOCK MONITORING (per chain)                   │    │
│  │                                                         │    │
│  │  ┌───────────────────────────────────────────┐          │    │
│  │  │  Block Monitor Goroutine                   │          │    │
│  │  │                                           │          │    │
│  │  │  loop:                                    │          │    │
│  │  │    1. Fetch latest block number           │          │    │
│  │  │    2. For each new block since last scan: │          │    │
│  │  │       a. Get all transactions in block    │          │    │
│  │  │       b. Filter: to_address IN            │          │    │
│  │  │          (active deposit addresses)       │          │    │
│  │  │       c. For matching txns:               │          │    │
│  │  │          - Verify amount                  │          │    │
│  │  │          - Update payment status          │          │    │
│  │  │          - Track confirmations            │          │    │
│  │  │       d. When confirmations >= threshold: │          │    │
│  │  │          - Mark payment CONFIRMED         │          │    │
│  │  │          - Fire webhook                   │          │    │
│  │  │    3. Update last_scanned_block           │          │    │
│  │  │    4. Sleep (block time interval)         │          │    │
│  │  └───────────────────────────────────────────┘          │    │
│  │                                                         │    │
│  │  Chain-Specific Adapters:                               │    │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐          │    │
│  │  │  EVM   │ │Bitcoin │ │ Tron   │ │Solana  │          │    │
│  │  │Adapter │ │Adapter │ │Adapter │ │Adapter │          │    │
│  │  │        │ │        │ │        │ │(future)│          │    │
│  │  │go-eth  │ │btcd    │ │tron-go │ │solana  │          │    │
│  │  │client  │ │rpccli  │ │client  │ │go-sdk  │          │    │
│  │  └────────┘ └────────┘ └────────┘ └────────┘          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │            SMARTSWEEP (Smart Contracts)                   │    │
│  │                                                         │    │
│  │  ┌──────────────────────────────────────────────┐       │    │
│  │  │  SmartSweep.sol (Solidity - EVM Chains)      │       │    │
│  │  │                                              │       │    │
│  │  │  State:                                      │       │    │
│  │  │    coldWallet: address (IMMUTABLE)            │       │    │
│  │  │    owner: address                            │       │    │
│  │  │                                              │       │    │
│  │  │  Functions:                                  │       │    │
│  │  │    sweep(token, amount):                     │       │    │
│  │  │      - Transfers from deposit → coldWallet   │       │    │
│  │  │      - Only callable by authorized sweeper   │       │    │
│  │  │      - Destination hardcoded, CANNOT change  │       │    │
│  │  │                                              │       │    │
│  │  │    sweepAll(token):                          │       │    │
│  │  │      - Sweeps entire balance to coldWallet   │       │    │
│  │  │                                              │       │    │
│  │  │  Security:                                   │       │    │
│  │  │    - Cold wallet address set at deployment   │       │    │
│  │  │    - Cannot be changed after deployment      │       │    │
│  │  │    - Even owner cannot redirect funds        │       │    │
│  │  │    - Audited by QuillAudits                  │       │    │
│  │  └──────────────────────────────────────────────┘       │    │
│  │                                                         │    │
│  │  Sweep Scheduler (Go Worker):                           │    │
│  │    1. Check deposit wallet balances                     │    │
│  │    2. If balance > threshold:                           │    │
│  │       a. Estimate gas cost                              │    │
│  │       b. Build sweep transaction                        │    │
│  │       c. Sign with hot wallet (gas payer)               │    │
│  │       d. Execute smart contract sweep                   │    │
│  │       e. Log sweep in database                          │    │
│  │    3. Repeat on schedule (configurable interval)        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │            RPC NODE ACCESS STRATEGY                       │    │
│  │                                                         │    │
│  │  Primary: Self-hosted nodes (for privacy/sovereignty)   │    │
│  │  Fallback: Alchemy / Infura / QuickNode (for uptime)   │    │
│  │                                                         │    │
│  │  ┌────────────────────────────────────────────┐         │    │
│  │  │  RPC Load Balancer                          │         │    │
│  │  │                                            │         │    │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐ │         │    │
│  │  │  │ Own Node │  │ Alchemy  │  │ Infura   │ │         │    │
│  │  │  │ (primary)│  │ (backup) │  │ (backup) │ │         │    │
│  │  │  └──────────┘  └──────────┘  └──────────┘ │         │    │
│  │  └────────────────────────────────────────────┘         │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Security Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    SECURITY ARCHITECTURE                         │
│                                                                 │
│  ┌──────────────────────────── NETWORK LAYER ──────────────┐    │
│  │                                                         │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │    │
│  │  │ UFW/iptables│  │ SSL/TLS 1.3  │  │ DDoS          │  │    │
│  │  │             │  │ (Let's       │  │ Protection    │  │    │
│  │  │ Only expose:│  │  Encrypt)    │  │ (Rate limit   │  │    │
│  │  │ 80, 443     │  │              │  │  + fail2ban)  │  │    │
│  │  └─────────────┘  └──────────────┘  └───────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌──────────────────────── APPLICATION LAYER ──────────────┐    │
│  │                                                         │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │    │
│  │  │ API Key     │  │ HMAC Webhook │  │ Input         │  │    │
│  │  │ Auth        │  │ Signatures   │  │ Validation    │  │    │
│  │  │ (SHA-256    │  │              │  │               │  │    │
│  │  │  hashed)    │  │ HMAC-SHA256  │  │ Strict types  │  │    │
│  │  │             │  │ (payload +   │  │ Amount limits │  │    │
│  │  │             │  │  timestamp)  │  │ Address valid │  │    │
│  │  └─────────────┘  └──────────────┘  └───────────────┘  │    │
│  │                                                         │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │    │
│  │  │ Rate        │  │ CORS         │  │ CSP           │  │    │
│  │  │ Limiting    │  │ (strict      │  │ Headers       │  │    │
│  │  │ (Redis)     │  │  origins)    │  │               │  │    │
│  │  └─────────────┘  └──────────────┘  └───────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌──────────────────────── KEY MANAGEMENT ─────────────────┐    │
│  │                                                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ ZERO KEY EXPOSURE MODEL                          │    │    │
│  │  │                                                 │    │    │
│  │  │ ❌ Master seed → NEVER on server (air-gapped)   │    │    │
│  │  │ ❌ Deposit keys → NEVER on server (HD derived)  │    │    │
│  │  │ ❌ Cold wallet key → NEVER on server             │    │    │
│  │  │ ✅ Hot wallet key → AES-256 encrypted on server │    │    │
│  │  │    (minimal funds for gas fees only)             │    │    │
│  │  │                                                 │    │    │
│  │  │ Smart contracts enforce:                         │    │    │
│  │  │ - Funds can ONLY go to hardcoded cold wallet    │    │    │
│  │  │ - Even contract owner can't change destination  │    │    │
│  │  │ - Server compromise = NO fund theft possible    │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌──────────────────────── DATA LAYER ─────────────────────┐    │
│  │                                                         │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │    │
│  │  │ DB          │  │ Encrypted    │  │ Audit         │  │    │
│  │  │ Encryption  │  │ Backups      │  │ Logging       │  │    │
│  │  │ at Rest     │  │              │  │               │  │    │
│  │  │             │  │ AES-256      │  │ All API calls │  │    │
│  │  │ TDE for PG  │  │ encrypted    │  │ All txns      │  │    │
│  │  │ Sensitive   │  │ off-site     │  │ All sweeps    │  │    │
│  │  │ fields      │  │ daily        │  │ All logins    │  │    │
│  │  └─────────────┘  └──────────────┘  └───────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Payment Flow Sequence Diagram

```
Customer          PayRam Widget      API Gateway      BlockchainSvc     Blockchain      Webhook Svc      Merchant
   │                   │                 │                 │                │                │               │
   │  Select amount    │                 │                 │                │                │               │
   │──────────────────>│                 │                 │                │                │               │
   │                   │  POST /payment  │                 │                │                │               │
   │                   │────────────────>│                 │                │                │               │
   │                   │                 │ Generate addr   │                │                │               │
   │                   │                 │────────────────>│                │                │               │
   │                   │                 │  deposit_addr   │                │                │               │
   │                   │                 │<────────────────│                │                │               │
   │                   │  {ref_id, addr} │                 │                │                │               │
   │                   │<────────────────│                 │                │                │               │
   │  Show QR + addr   │                 │                 │                │                │               │
   │<──────────────────│                 │                 │                │                │               │
   │                   │                 │                 │                │                │               │
   │  Send crypto ─────────────────────────────────────────────────────────>│                │               │
   │                   │                 │                 │                │                │               │
   │                   │                 │                 │ Monitor blocks │                │               │
   │                   │                 │                 │<───────────────│                │               │
   │                   │                 │                 │ Tx detected!   │                │               │
   │                   │                 │  status=        │                │                │               │
   │                   │                 │  confirming     │                │                │               │
   │                   │                 │<────────────────│                │                │               │
   │                   │                 │                 │                │                │               │
   │                   │                 │                 │ N confirmations│                │               │
   │                   │                 │                 │<───────────────│                │               │
   │                   │                 │  status=        │                │                │               │
   │                   │                 │  confirmed      │                │                │               │
   │                   │                 │<────────────────│                │                │               │
   │                   │                 │                 │                │                │               │
   │                   │                 │  Fire webhook   │                │                │               │
   │                   │                 │───────────────────────────────────────────────────>│               │
   │                   │                 │                 │                │                │  POST webhook │
   │                   │                 │                 │                │                │──────────────>│
   │                   │                 │                 │                │                │    200 OK     │
   │                   │                 │                 │                │                │<──────────────│
   │  Confirmation     │                 │                 │                │                │               │
   │<──────────────────│                 │                 │                │                │               │
   │                   │                 │                 │                │                │               │
   │                   │                 │           SmartSweep executes                    │               │
   │                   │                 │                 │───────────────>│                │               │
   │                   │                 │                 │  Funds → Cold  │                │               │
   │                   │                 │                 │<───────────────│                │               │
```

---

## 8. MCP / AI Agent Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                MCP / AI AGENT INTEGRATION                        │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐           │
│  │           AI AGENT (Claude, GPT, etc.)            │           │
│  │                                                  │           │
│  │  Agent discovers tools via MCP handshake:        │           │
│  │  → create-payee                                  │           │
│  │  → send-payment                                  │           │
│  │  → get-balance                                   │           │
│  │  → generate-invoice                              │           │
│  │  → lookup-payment                                │           │
│  │  → search-payments                               │           │
│  │  → get-daily-volume                              │           │
│  │  → get-payment-summary                           │           │
│  │  → get-unswept-balances                          │           │
│  │  → test-connection                               │           │
│  └──────────────────┬───────────────────────────────┘           │
│                     │                                            │
│                     │ StreamableHTTP + SSE                       │
│                     │ POST /mcp (commands)                       │
│                     │ GET /mcp/sse (events)                      │
│                     ▼                                            │
│  ┌──────────────────────────────────────────────────┐           │
│  │           MCP SERVER (TypeScript / Node.js)       │           │
│  │           Port 3333                               │           │
│  │                                                  │           │
│  │  ┌────────────────────────────────┐              │           │
│  │  │ MCP Protocol Handler          │              │           │
│  │  │  - Tool discovery              │              │           │
│  │  │  - Request routing             │              │           │
│  │  │  - SSE event streaming         │              │           │
│  │  │  - Error handling              │              │           │
│  │  └─────────────┬──────────────────┘              │           │
│  │                │                                  │           │
│  │  ┌─────────────▼──────────────────┐              │           │
│  │  │ Tool Handlers                   │              │           │
│  │  │  - PaymentTools                 │              │           │
│  │  │  - WalletTools                  │              │           │
│  │  │  - AnalyticsTools               │              │           │
│  │  │  - ConnectionTools              │              │           │
│  │  └─────────────┬──────────────────┘              │           │
│  │                │                                  │           │
│  │  ┌─────────────▼──────────────────┐              │           │
│  │  │ Internal API Client             │              │           │
│  │  │  → Calls Go API Gateway         │              │           │
│  │  │    (localhost:8080)             │              │           │
│  │  └────────────────────────────────┘              │           │
│  └──────────────────────────────────────────────────┘           │
│                                                                 │
│  Protocols Supported:                                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                     │
│  │   MCP    │  │  x402    │  │ ERC-8004 │                     │
│  │ Standard │  │ HTTP 402 │  │ Agent ID │                     │
│  │ Agent    │  │ Pay-per- │  │ Trustless│                     │
│  │ Protocol │  │ call API │  │ Identity │                     │
│  └──────────┘  └──────────┘  └──────────┘                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 9. Database Schema (ERD)

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│    merchants     │     │    api_keys      │     │    webhooks      │
│──────────────────│     │──────────────────│     │──────────────────│
│ id (PK)         │────<│ merchant_id (FK) │     │ id (PK)         │
│ name            │     │ id (PK)          │     │ merchant_id (FK)│──┐
�� email           │     │ key_hash         │     │ url             │  │
│ password_hash   │────<│ name             │     │ events[]        │  │
│ api_key_hash    │     │ permissions[]    │     │ secret          │  │
│ webhook_secret  │     │ active           │     │ active          │  │
│ cold_wallets{}  │     │ last_used_at     │     │ created_at      │  │
│ settings{}      │     │ created_at       │     └──────────────────┘  │
│ created_at      │     └──────────────────┘                          │
└────────┬─────────┘                                                   │
         │                                                             │
         │  ┌──────────────────┐     ┌──────────────────┐             │
         │  │    payments      │     │ webhook_deliveries│             │
         │  │──────────────────│     │──────────────────│             │
         └─<│ merchant_id (FK) │     │ id (PK)         │             │
            │ id (PK)         │────<│ payment_id (FK)  │             │
            │ reference_id    │     │ webhook_id (FK)  │>────────────┘
            │ amount          │     │ event            │
            │ currency        │     │ payload{}        │
            │ blockchain      │     │ status           │
            │ status          │     │ response_code    │
            │ deposit_address │     │ attempts         │
            │ txid            │     │ next_retry_at    │
            │ confirmations   │     │ delivered_at     │
            │ customer_email  │     └──────────────────┘
            │ customer_id     │
            │ invoice_id      │
            │ expires_at      │
            │ confirmed_at    │
            │ created_at      │
            └────────┬─────────┘
                     │
┌──────────────────┐ │  ┌──────────────────┐
│    wallets       │ │  │    sweeps        │
│──────────────────│ │  │──────────────────│
│ id (PK)         │ │  │ id (PK)         │
│ merchant_id (FK)│ │  │ from_wallet (FK) │>──┐
│ blockchain      │<┘  │ to_address      │   │
│ address         │    │ amount          │   │
│ type (deposit/  │<───│ token           │   │
│  hot/cold)      │    │ blockchain      │   │
│ balance         │    │ txid            │   │
│ token           │    │ status          │   │
│ derivation_idx  │    │ gas_used        │   │
│ created_at      │    │ created_at      │   │
└──────────────────┘    │ completed_at    │   │
                       └──────────────────┘   │
                                              │
┌──────────────────┐                          │
│  activity_logs   │                          │
│──────────────────│                          │
│ id (PK)         │                          │
│ merchant_id (FK)│                          │
│ action          │                          │
│ entity_type     │                          │
│ entity_id       │                          │
│ details{}       │                          │
│ ip_address      │                          │
│ created_at      │                          │
└──────────────────┘
```

---

## 10. Deployment Options

```
OPTION A: Single VPS (Default - matches PayRam's approach)
═══════════════════════════════════════════════════════════
  ┌──────────────────────────┐
  │    Single VPS            │
  │    Ubuntu 22.04+         │
  │    4-8 CPU, 8GB RAM      │
  │    100GB SSD             │
  │                          │
  │    Docker Compose:       │
  │    - All services        │
  │    - PostgreSQL           │
  │    - Redis               │
  │    - Nginx + SSL         │
  └──────────────────────────┘
  Cost: ~$20-50/month
  Best for: Small-medium merchants

OPTION B: Multi-Server (High Availability)
═══════════════════════════════════════════
  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │ App      │  │ App      │  │ Load     │
  │ Server 1 │  │ Server 2 │  │ Balancer │
  └────┬─────┘  └────┬─────┘  └──────────┘
       │              │
  ┌────▼──────────────▼─────┐
  │  Managed PostgreSQL     │
  │  (RDS/Cloud SQL)        │
  └─────────────────────────┘
  ┌─────────────────────────┐
  │  Managed Redis          │
  │  (ElastiCache/Memorystore)│
  └─────────────────────────┘
  Cost: ~$200-500/month
  Best for: High-volume merchants

OPTION C: Kubernetes (Enterprise)
═════════════════════════════════
  ┌─────────────────────────────┐
  │  Kubernetes Cluster         │
  │                             │
  │  ┌───────┐ ┌───────┐      │
  │  │API x3 │ │Web x2 │      │
  │  └───────┘ └───────┘      │
  │  ┌───────┐ ┌───────┐      │
  │  │MCP x2 │ │Worker │      │
  │  └───────┘ │  x3   │      │
  │            └───────┘      │
  │  HPA, PDB, NetworkPolicy  │
  └─────────────────────────────┘
  Cost: ~$500-2000/month
  Best for: Enterprise, multi-tenant
```
