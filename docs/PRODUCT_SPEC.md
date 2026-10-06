# PayRam Clone - Complete Product Documentation

**Version:** 1.0
**Date:** April 2, 2026
**Purpose:** Comprehensive product spec for building a PayRam clone

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Product Overview](#2-product-overview)
3. [User Personas & Target Market](#3-user-personas--target-market)
4. [Landing Page Structure](#4-landing-page-structure)
5. [Demo App - Full Page Inventory](#5-demo-app---full-page-inventory)
6. [Core Features Specification](#6-core-features-specification)
7. [Payment Flows](#7-payment-flows)
8. [API Specification](#8-api-specification)
9. [Blockchain Integration](#9-blockchain-integration)
10. [Security Architecture](#10-security-architecture)
11. [AI Agent / MCP Integration](#11-ai-agent--mcp-integration)
12. [Mobile App Features](#12-mobile-app-features)
13. [Data Models](#13-data-models)
14. [Technology Stack Recommendation](#14-technology-stack-recommendation)

---

## 1. Executive Summary

PayRam is a **self-hosted, non-custodial cryptocurrency payment gateway** that enables businesses and AI agents to accept crypto and card-to-crypto payments on infrastructure they own. Key differentiators:

- **0% processing fees** (only gas + hosting costs)
- **Self-hosted** on merchant's own VPS (Ubuntu 22.04+)
- **Non-custodial** - merchant retains full fund control
- **No KYC/KYB** required - permissionless onboarding
- **AI-native** - MCP protocol for autonomous agent payments
- **Multi-chain** - BTC, ETH, Base, Polygon, Tron, TON + 20 tokens
- **Card-to-crypto onramp** - 175+ payment methods, 190+ countries
- **SmartSweep** - Smart contract-enforced cold storage transfers

**Founder:** Siddharth Menon (WazirX co-founder, 15M+ users)
**Traction:** $100M+ settled, 850K+ txns, 100+ merchants
**HQ:** New York, USA | Team: 1-10

---

## 2. Product Overview

### What It Does
PayRam is deployed on a merchant's own VPS via a single bash command. Once running, it:
1. Generates unique deposit addresses per transaction
2. Monitors blockchain(s) for incoming payments
3. Confirms payments and sends webhook notifications
4. Uses SmartSweep smart contracts to move funds to cold storage
5. Provides a merchant dashboard for analytics and management
6. Exposes REST API + SDK + MCP for integration

### Product Suite
| Component | Description |
|-----------|-------------|
| **Payment Gateway** | Core self-hosted payment processor |
| **Merchant Dashboard** | Web UI for managing payments, wallets, analytics |
| **Card-to-Crypto Onramp** | Fiat card payments → crypto settlement |
| **SmartSweep** | Smart contract auto-sweep to cold storage |
| **MCP Server** | AI agent payment integration protocol |
| **Mobile App** | iOS/Android business management app |
| **Widget** | Embeddable "Add Credit" button |
| **REST API** | Programmatic payment integration |
| **Node.js SDK** | Developer SDK |
| **Testnet Faucet** | ETH Sepolia faucet for testing |

---

## 3. User Personas & Target Market

### Primary Personas

**1. iGaming/Casino Operator**
- Needs: Instant USDT/USDC payouts, no chargebacks, censorship resistance
- Pain: Traditional processors freeze accounts, high fees, 3-day settlements
- Value: Self-hosted = no account freezes, 0% fees, instant settlement

**2. E-Commerce Merchant**
- Needs: Global payment acceptance, low fees, card + crypto
- Pain: Stripe/PayPal fees (2.9%+), chargebacks, geographic restrictions
- Value: 0% crypto fees, card-to-crypto onramp, 190+ countries

**3. AI Agent Developer**
- Needs: Autonomous payment processing for AI agents
- Pain: Traditional APIs require human approval, KYB, accounts
- Value: MCP native, no KYB, agents can deploy independently

**4. High-Risk Business (Adult, Cannabis)**
- Needs: Payment processing that won't de-platform them
- Pain: Every traditional processor rejects or freezes
- Value: Self-hosted, censorship-resistant, no third-party control

### Target Industries
| Industry | Key Value |
|----------|-----------|
| Casino / iGaming | No KYC, instant settlement, no freezes |
| Adult Entertainment | Censorship-resistant |
| E-Commerce | Global reach, low fees |
| Marketplaces | Multi-vendor support |
| Charities / NGOs | Transparent blockchain records |
| AI Agent Commerce | MCP-native autonomous payments |
| Fintech / PSPs | White-label capability |

---

## 4. Landing Page Structure

### Page Architecture (Single-Page Application)
PayRam uses a single-page marketing site with separate pages only for `/blog`, `/demo`, and external `docs.payram.com`.

### Section-by-Section Breakdown

#### 4.1 Navigation Bar
- Logo: PayRam
- Menu: AI Agents | Supported Coins (dropdown) | Documentation | Blog | Contact Us
- CTAs: "Try Demo" (primary), "Explore Docs" (secondary)

#### 4.2 Hero Section
- **Headline:** "The Self-Hosted Payment Gateway *for Humans & AI Agents.*"
- **Subheadline:** "Accept card and crypto payments on infrastructure you own. No signup, no approvals, no account freezes -- ever."
- **Stats Bar:** $100M+ settled | 100+ merchants | No signup needed
- **CTAs:** "Explore Docs" | "Full Demo Page"
- **Visual:** Dark theme with animated payment flow illustration

#### 4.3 Card-to-Crypto Section
- Badge: "Live"
- Headline: "Magic Conversion"
- Copy: "Customers pay by card and you get in crypto"
- Stats: 40+ currencies, 100+ countries, 300+ payment methods

#### 4.4 Self-Hosted Section
- Copy: "PayRam is a self-hosted cryptocurrency payment processor that is secure, private, & resistant to censorship."

#### 4.5 Quick Setup Section
- Headline: "Deploy in 10 minutes. No signup. No waiting."
- Code block: `/bin/bash -c "$(curl -fsSL https://payram.com/setup_payram.sh)"`
- Requirements: Ubuntu 22.04+, 2 CPU, 8GB RAM

#### 4.6 Payment Orchestration Section
- Visual showing payment flow from checkout to settlement

#### 4.7 AI Agents Section
- Comparison table: "Old Way" vs "PayRam Way"
- Two integration paths: Any AI Agent (MCP) | OpenClaw/NemoClaw
- Tags: No Signup, No KYB, Card-to-Crypto, Self-Custody, 5-Min Deploy, Multi-Chain

#### 4.8 Press Coverage Section
- AP News, Cointelegraph, TechBullion, Analytics Insight, TheGamblest, AccessNewsWire

#### 4.9 Core Features Grid (6 Cards)
1. Control & Custody
2. Payment Options
3. Authentication & Verification
4. Fund Management
5. Compliance Assurance
6. Automation

#### 4.10 Mobile App Section
- "Power in your pocket"
- Wallet Management | Instant Approvals | Secure Transfers
- App Store + Play Store links

#### 4.11 Dashboard Showcase (4 Panels)
1. Own Your Payments
2. Unified Wallet Control
3. Automate Settlements
4. Seamless API Integration

#### 4.12 Supported Cryptocurrencies
- BTC, ETH, USDT, USDC, TRX (Soon), SOL (Soon)

#### 4.13 Testimonial
- "PayRam cut our payment settlement time from 3 days to under 10 minutes..."
- Marcus T., CTO at a Global E-Commerce Platform

#### 4.14 FAQ Section (5 Questions)
- Getting started, transaction limits, server requirements, supported crypto, demo availability

#### 4.15 Blog Preview (3 Recent Posts)

#### 4.16 Footer
- Industries: Casino, iGaming, E-Commerce, Marketplace, Charity
- Supported Coins: BTC, ETH, TRX, SOL, USDT, USDC
- Resources: Documentation, Live Demo, AI Agent Gateway, OpenClaw, Releases, Media Kit, Faucet
- Contact: Support, Sales, Careers
- Legal: Privacy Policy, Terms & Conditions, Cookie Policy
- Newsletter signup
- Social: LinkedIn, Twitter/X, Email
- Security Badge: "Audited by QuillAudits"

### Design System
- **Theme:** Dark background with light text
- **Accent Colors:** Green (#01e46f), pink highlights
- **Typography:** Modern sans-serif
- **Style:** Tech-forward, crypto-native aesthetic
- **Responsive:** Full mobile support

---

## 5. Demo App - Full Page Inventory

### Demo Entry Flow (payram.com/demo)
The demo page is an **interactive live demo**, not a booking form:
1. Select preset amount ($10, $50, $100, $200)
2. Send test funds (simulated payment)
3. View payment confirmation + webhook

### Demo Dashboard (demo.payram.com)
Entry: Email-based login at demo.payram.com/login

#### Sidebar Navigation (Expected)
Based on product features and API:
- **Dashboard** - Overview, stats, recent activity
- **Payments** - Payment list, create payment, payment details
- **Transactions** - Transaction history, filters, export
- **Invoices** - Create/manage invoices
- **Wallets** - Multi-chain wallet management, balances
- **Sweep** - SmartSweep status, configuration
- **Recipients/Contacts** - Payee management
- **Reports/Analytics** - Charts, P&L, volume metrics
- **API Keys** - Developer credentials
- **Webhooks** - Webhook configuration, logs
- **Settings** - Profile, team, security, notifications
- **Integrations** - Widget config, SDK docs

#### Dashboard Page
- **Metrics Cards:** Total settled, Today's volume, Active wallets, Pending sweeps
- **Charts:** Payment volume over time (line/bar chart)
- **Recent Payments Table:**
  - Columns: Reference ID, Amount, Currency, Blockchain, Status, Timestamp
  - Status values: Created → Confirming → Confirmed
- **Quick Actions:** Create Payment, View Reports

#### Payments Page
- **Payment List Table:**
  - Reference ID
  - Amount (USD equivalent)
  - Currency (USDC, USDT, BTC, ETH)
  - Blockchain (Ethereum, Tron, Bitcoin, Base)
  - Customer Email
  - Status (Created/Confirming/Confirmed)
  - Timestamp
- **Filters:** Date range, currency, blockchain, status
- **Actions:** Create new payment, export CSV
- **Payment Detail View:**
  - Full payment info
  - Deposit address (QR code)
  - Blockchain transaction hash (linked to explorer)
  - Confirmations count
  - Webhook delivery status

#### Wallets Page
- **Wallet List:** Per-blockchain wallets with balances
- **Address management:** View deposit/hot/cold addresses
- **Balance breakdown:** By token and chain
- **Sweep configuration:** Auto-sweep thresholds

#### Sweep Management
- **Sweep History:** Table of sweep transactions
- **Status:** Pending → Processing → Completed
- **Configuration:** Cold wallet addresses, thresholds, gas settings

#### Reports/Analytics
- **Volume charts:** Daily/weekly/monthly
- **Revenue breakdown:** By currency, blockchain
- **P&L tracking:** Revenue vs gas costs
- **Export:** CSV, PDF

#### Settings
- **Profile:** Business name, email, timezone
- **Team:** User management, roles (Admin, Viewer)
- **API Keys:** Generate/revoke API keys
- **Webhooks:** Configure endpoints, view delivery logs
- **Security:** 2FA, password change
- **Notifications:** Email preferences

---

## 6. Core Features Specification

### 6.1 Payment Processing

#### Create Payment
- Input: amount (USD), currency, customer_email, customer_id, invoice_id, expiration
- Output: payment object with unique deposit address, reference_id
- Flow: Generates unique blockchain address → monitors for payment → confirms → webhook

#### Payment Status Flow
```
Created → Confirming → Confirmed
```

#### Supported Payment Methods
- Direct crypto deposit (wallet-to-address)
- Card-to-crypto onramp (credit/debit, Apple Pay, Google Pay, bank transfer)

### 6.2 SmartSweep
- Smart contracts with **hardcoded cold wallet destinations**
- Periodic automatic transfer from hot wallet to cold storage
- Private keys never stored on live server
- AES-256 encrypted hot wallet for gas fees only
- Master wallet keys permanently offline
- Configurable sweep thresholds

### 6.3 Card-to-Crypto Onramp
- 175+ local/regional payment methods
- 190+ countries
- 40+ fiat currencies
- Customers pay by card → merchant receives stablecoins (USDC on Base)
- Third-party onramp integration

### 6.4 Widget Embed
```html
<script src="https://payram.com/widget/payram-add-credit-v1.js"
  data-payram-url="your-url"
  data-api-key="your-key"
  data-amounts="10,50,100,200"
  data-theme="dark"
  data-customer-email="user@example.com">
</script>
```

### 6.5 Webhook System
- Event: `payment.confirmed`
- Payload: reference_id, amount, currency, blockchain, txid, confirmations, customer data, timestamp
- Retry: 30m, 1h, 2h, 4h, 8h, 24h, 48h
- Idempotency protection
- Delivery log in dashboard

---

## 7. Payment Flows

### Flow 1: Direct Crypto Payment
```
Customer → Selects amount → Gets unique deposit address (QR + text)
         → Sends crypto from their wallet
         → Blockchain confirms transaction
         → PayRam detects payment → Updates status to "Confirming"
         → Required confirmations met → Status = "Confirmed"
         → Webhook fires to merchant endpoint
         → SmartSweep moves funds to cold storage
```

### Flow 2: Card-to-Crypto (Onramp)
```
Customer → Selects amount → Redirected to onramp partner
         → Pays with card/Apple Pay/bank transfer
         → Onramp converts fiat → stablecoin
         → Stablecoin sent to merchant's deposit address
         → PayRam detects and confirms
         → Webhook fires
         → SmartSweep to cold storage
```

### Flow 3: Widget Integration
```
Merchant → Embeds PayRam widget script on their site
Customer → Clicks "Add Credit" button
         → Widget overlay shows amount options
         → Customer selects amount and pays (crypto or card)
         → Payment flow same as above
         → Widget shows confirmation
```

### Flow 4: API Integration
```
Merchant Backend → POST /api/v1/payment (amount, currency, customer info)
                → Receives payment object with deposit address
                → Shows address/QR to customer
Customer → Sends payment
                → GET /api/v1/payment/:reference_id (poll or webhook)
                → Status = confirmed
                → Merchant fulfills order
```

### Flow 5: AI Agent Payment
```
AI Agent → Connects to MCP server (localhost:3333/mcp)
        → Auto-discovers tools via MCP handshake
        → Calls create-payee → Creates payment recipient
        → Calls send-payment → Executes transaction
        → Calls lookup-payment → Verifies confirmation
        → Autonomous payment completed
```

---

## 8. API Specification

### Base URL
`https://{merchant-server}/api/v1`

### Authentication
- API Key based (header: `X-API-Key` or `Authorization: Bearer`)

### Endpoints

#### POST /api/v1/payment
Create a new payment
```json
Request:
{
  "customerEmail": "user@example.com",
  "customerID": "cust_123",
  "amountInUSD": 100.00
}

Response:
{
  "host": "https://yourdomain.com:8443",
  "reference_id": "c80f5363-0397-4761-aa1a-3155c3a21470",
  "url": "https://yourdomain.com/payments?reference_id=c80f5363...&host=https://yourdomain.com:8443"
}
```

#### GET /api/v1/payment/reference/:reference_id
Check payment status
```json
Response:
{
  "invoiceID": "inv_456",
  "customerID": "cust_123",
  "amountInUSD": 100.00,
  "paymentState": "FILLED",
  "merchantName": "Your Store",
  "referenceID": "c80f5363-0397-4761-aa1a-3155c3a21470",
  "createdAt": "2026-04-02T12:00:00Z"
}
```

**Payment States:** OPEN → CANCELLED | FILLED | PARTIALLY_FILLED | OVER_FILLED

#### GET /api/v1/ticker
Fetch current token prices and supported currencies

#### POST /api/v1/blockchain-currency/reference/:reference_id
Get available blockchain currencies for a payment

#### POST /api/v1/deposit-address/reference/:reference_id
Assign deposit address for a payment (blockchain_code: BTC/ETH/TRX/BASE/POLYGON)

#### POST /api/v1/withdrawal/merchant
Create a payout
```json
Request:
{
  "email": "recipient@example.com",
  "blockChainCode": "ETH",
  "currencyCode": "USDT",
  "amount": "125.50",
  "toAddress": "0x...",
  "customerID": 12345
}
```
Limits: Auto-approve up to $500, Hourly cap $5,000, Daily cap $10,000

#### GET /api/v1/withdrawal/:id/merchant
Get payout status (pending-otp → pending-approval → pending → initiated → sent → processed)

#### GET /api/v1/withdrawal/merchant
List all payouts (supports limit, offset, order, sortBy params)

#### POST /api/v1/referral/referee
Register referral

#### POST /api/v1/referral/event-log
Log referral event

### Webhook Events
- `payment.created` - Payment initiated
- `payment.confirming` - Transaction detected, awaiting confirmations
- `payment.confirmed` - Payment fully confirmed
- `payment.expired` - Payment expired without receipt
- `sweep.completed` - Funds swept to cold storage

### MCP Endpoints
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/mcp` | POST | StreamableHTTP MCP interface |
| `/mcp/sse` | GET | Server-Sent Events streaming |
| `/healthz` | GET | Health check |

### MCP Tools
| Tool | Purpose |
|------|---------|
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

---

## 9. Blockchain Integration

### Supported Networks

| Network | Status | Use Case |
|---------|--------|----------|
| Bitcoin | Live | BTC payments |
| Ethereum | Live | ETH, USDT (ERC-20), USDC (ERC-20) |
| Base | Live | USDC (preferred for low fees) |
| Polygon | Live | Low-cost stablecoin payments |
| Tron | Live | USDT (TRC-20) - high volume |
| TON | Coming | Telegram-integrated payments |
| Solana | Coming | High-speed payments |
| BNB Chain | Planned | BSC ecosystem |
| Ripple | Planned | Cross-border payments |
| Monero | Planned | Privacy payments |

### Token Support

| Token | Networks | Type |
|-------|----------|------|
| USDT | Ethereum (ERC-20), Tron (TRC-20) | Stablecoin |
| USDC | Ethereum (ERC-20), Base | Stablecoin |
| BTC | Bitcoin | Native |
| ETH | Ethereum | Native |
| TRX | Tron | Native |
| SOL | Solana (coming) | Native |
| 20+ others | Various | Mixed |

### Address Generation Strategy
- Unique deposit address per transaction (HD wallet derivation)
- Deterministic address generation from master seed
- No reuse of addresses across transactions

### Block Confirmation Requirements (Recommended)
| Network | Confirmations |
|---------|---------------|
| Bitcoin | 3 |
| Ethereum | 12 |
| Base | 12 |
| Tron | 19 |
| Polygon | 128 |

---

## 10. Security Architecture

### Zero Key Exposure Model
```
┌─────────────────────────────────────┐
│           PayRam Server             │
│  ┌─────────────────────────────┐    │
│  │  Encrypted Hot Wallet       │    │
│  │  (AES-256, gas fees only)   │    │
│  └─────────────────────────────┘    │
│                                     │
│  ❌ NO deposit wallet keys          │
│  ❌ NO master wallet keys           │
│  ❌ NO cold storage keys            │
└─────────────────────────────────────┘
            │
            │ SmartSweep (Smart Contract)
            │ Hardcoded cold wallet destination
            ▼
┌─────────────────────────────────────┐
│        Cold Storage Wallet          │
│  (Master keys PERMANENTLY offline)  │
└─────────────────────────────────────┘
```

### Security Layers
1. **Server-Level:** Encrypted hot wallet (gas only), no deposit keys on server
2. **Smart Contract-Level:** Hardcoded destinations, immutable sweep logic
3. **Key Management:** Master keys permanently offline, HD derivation for deposits
4. **Network-Level:** SSL auto-provisioning, firewall rules
5. **Application-Level:** API key authentication, webhook signatures
6. **Audit:** Smart contracts audited by QuillAudits

### Breach Scenario
Even with root-level server access, attacker CANNOT:
- Access deposit wallet funds
- Redirect sweeps to different addresses (hardcoded in smart contract)
- Access cold storage keys

Attacker CAN only:
- Access encrypted hot wallet (contains minimal gas funds)
- View transaction history

---

## 11. AI Agent / MCP Integration

### Architecture
```
AI Agent (Claude, GPT, etc.)
    │
    │ MCP Protocol (StreamableHTTP + SSE)
    ▼
PayRam MCP Server (localhost:3333/mcp)
    │
    │ Internal API
    ▼
PayRam Gateway (payment processing)
    │
    │ On-chain transactions
    ▼
Blockchain Networks (ETH, BTC, Base, etc.)
```

### MCP Configuration
```json
{
  "mcpServers": {
    "payram": {
      "url": "http://localhost:3333/mcp"
    }
  }
}
```
- No API keys needed for MCP
- Agent auto-discovers tools via MCP handshake
- Works with any MCP-aware client (Claude, Copilot, etc.)

### Protocols
- **MCP (Model Context Protocol):** Standard AI agent integration
- **x402:** HTTP 402-based machine-to-machine payment protocol
- **ERC-8004:** Trustless agent identity, reputation, and validation standard

---

## 12. Mobile App Features

### PayRam Business App (iOS + Android)

#### Wallet Management
- Add/monitor wallets across chains
- Real-time balance updates
- Multi-chain view

#### Transaction Approvals
- Approve transactions individually or in bulk
- Push notification for pending approvals
- Enhanced security (biometric)

#### Secure Transfers
- Transfer funds between wallets
- Initiate sweeps
- Customizable gas fees

---

## 13. Data Models

### Payment
```
Payment {
  id: UUID
  reference_id: String (unique)
  amount: Decimal
  currency: Enum(USDC, USDT, BTC, ETH, TRX, SOL)
  blockchain: Enum(ethereum, bitcoin, base, polygon, tron, ton, solana)
  status: Enum(created, confirming, confirmed, expired)
  deposit_address: String
  txid: String (nullable)
  confirmations: Integer
  customer_email: String (nullable)
  customer_id: String (nullable)
  invoice_id: String (nullable)
  merchant_id: UUID
  expires_at: Timestamp
  created_at: Timestamp
  confirmed_at: Timestamp (nullable)
}
```

### Wallet
```
Wallet {
  id: UUID
  blockchain: Enum
  address: String
  type: Enum(deposit, hot, cold)
  balance: Decimal
  token: String
  merchant_id: UUID
  created_at: Timestamp
}
```

### Sweep
```
Sweep {
  id: UUID
  from_wallet_id: UUID
  to_address: String (cold storage)
  amount: Decimal
  token: String
  blockchain: Enum
  txid: String
  status: Enum(pending, processing, completed, failed)
  gas_used: Decimal
  created_at: Timestamp
  completed_at: Timestamp (nullable)
}
```

### Webhook
```
Webhook {
  id: UUID
  merchant_id: UUID
  url: String
  events: String[] (payment.confirmed, etc.)
  secret: String
  active: Boolean
  created_at: Timestamp
}
```

### WebhookDelivery
```
WebhookDelivery {
  id: UUID
  webhook_id: UUID
  payment_id: UUID
  event: String
  payload: JSON
  status: Enum(pending, delivered, failed)
  response_code: Integer (nullable)
  attempts: Integer
  next_retry_at: Timestamp (nullable)
  delivered_at: Timestamp (nullable)
}
```

### Merchant
```
Merchant {
  id: UUID
  name: String
  email: String
  api_key: String (hashed)
  webhook_secret: String
  cold_wallet_addresses: JSON
  created_at: Timestamp
}
```

### APIKey
```
APIKey {
  id: UUID
  merchant_id: UUID
  key_hash: String
  name: String
  permissions: String[]
  active: Boolean
  created_at: Timestamp
  last_used_at: Timestamp (nullable)
}
```

---

## 14. Technology Stack Recommendation

Based on PayRam's observed stack and best practices for building a clone:

### Backend (Core Gateway)
| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Language** | Go | PayRam uses Go. Fast, concurrent, great for blockchain RPC |
| **API Framework** | Gin or Echo | Lightweight, fast HTTP frameworks for Go |
| **Database** | PostgreSQL | Relational, ACID-compliant, JSON support |
| **Cache** | Redis | Rate limiting, session store, job queues |
| **Message Queue** | Redis Streams or NATS | Webhook delivery, sweep scheduling |
| **Blockchain RPC** | go-ethereum, btcd | Direct node communication |

### Frontend (Dashboard)
| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Framework** | Next.js (TypeScript) | PayRam uses TS/Next.js. SSR + SPA |
| **UI Library** | Tailwind CSS + shadcn/ui | Modern, dark-theme friendly |
| **Charts** | Recharts or Chart.js | Dashboard analytics |
| **State Management** | React Query (TanStack) | Server state sync |
| **Auth** | NextAuth.js | Email + 2FA |

### Blockchain Layer
| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Smart Contracts** | Solidity (EVM chains) | SmartSweep contracts |
| **HD Wallet** | BIP-32/39/44 | Deterministic address generation |
| **Key Management** | Vault or KMS | Secure key storage |
| **Node Access** | Alchemy/Infura + own nodes | Redundant RPC access |

### Infrastructure / DevOps
| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Container** | Docker + Docker Compose | Single-command deployment |
| **OS** | Ubuntu 22.04+ | PayRam requirement |
| **SSL** | Let's Encrypt (Certbot) | Auto-provisioning |
| **Monitoring** | Prometheus + Grafana | System + blockchain monitoring |
| **Logging** | Loki or ELK | Centralized logs |
| **Backup** | Automated DB + wallet backups | Disaster recovery |

### MCP / AI Layer
| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **MCP Server** | TypeScript (Node.js) | PayRam's approach. MCP SDK support |
| **Protocol** | StreamableHTTP + SSE | Standard MCP transport |
| **Tools** | 10 MCP tools | Payment operations |

---

---

## 15. Pricing Model (Actual PayRam)

| Component | Cost |
|-----------|------|
| **Core Processing Fees** | 0% |
| **Fund Orchestration / Sweep Fee** | 1-5% on settlement |
| **Card-to-Crypto Onramp** | Third-party provider rates |
| **VPS Hosting** | ~$20/month (merchant-managed) |
| **Blockchain Gas** | Standard network fees |
| **Setup** | Free |
| **Subscriptions** | None |

---

## 16. Version History (PayRam Actual)

| Version | Date | Key Changes |
|---------|------|-------------|
| Core v1.9.3 | Mar 2026 | BTC error fixes, deposit verification |
| Core v1.9.0 | Mar 2026 | RPC connection pool, ARM64 Docker |
| Core v1.8.0 | Feb 2026 | Deposit confirming state, OnramperPayments |
| Core v1.7.7 | Jan 2026 | Polygon integration |
| Core v1.7.4 | Dec 2025 | JWT auth, Swagger docs |
| Core v1.7.0 | Nov 2025 | Smart contract ETH deposits, Payout APIs |
| Core v1.5 | Aug 2025 | SmartSweep release |
| Frontend v0.4.2 | Mar 2026 | System Updater page |
| Frontend v0.3.0 | Feb 2026 | OnRamp Payments screen |
| MCP v1.2.0 | Mar 2026 | Auto-discovery, auth, analytics |

---

## 17. User Roles & Permissions

| Role | Capabilities |
|------|-------------|
| **Owner** | Full access, cannot be removed |
| **Admin** | Manage all projects/payments, invite/remove users |
| **Project Lead** | Create/update projects, analytics |
| **Project Manager** | View-only, export reports |
| **Project Ops** | View payment/customer data |
| **Platform Referral Admin** | Full referral program access |

---

## 18. SDK Reference (TypeScript)

```typescript
import { Payram } from 'payram';

const payram = new Payram({
  apiKey: process.env.PAYRAM_API_KEY!,
  baseUrl: process.env.PAYRAM_BASE_URL!,
  config: {
    timeoutMs: 10_000,
    maxRetries: 2,
    retryPolicy: 'safe', // 'none' | 'safe' | 'aggressive'
  },
});

// Create payment
const checkout = await payram.payments.initiatePayment({
  customerEmail: 'customer@example.com',
  customerId: 'cust_123',
  amountInUSD: 49.99,
});

// Check status
const payment = await payram.payments.getPaymentRequest(checkout.reference_id);

// Create payout
await payram.payouts.createPayout({
  email: 'merchant@example.com',
  blockchainCode: 'ETH',
  currencyCode: 'USDT',
  amount: '125.50',
  toAddress: '0x...',
  customerID: 414817384,
});

// Webhook handlers (Express, Fastify, Next.js supported)
app.post('/webhook', payram.webhooks.expressWebhook(async (payload) => {
  console.log(payload.event, payload.reference_id);
}));
```

---

## 19. Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AES_KEY` | 32-byte hex encryption key | `openssl rand -hex 32` |
| `SSL_CERT_PATH` | Certificate directory | `/etc/letsencrypt/live/domain.com` |
| `BLOCKCHAIN_NETWORK_TYPE` | testnet or mainnet | `mainnet` |
| `SERVER` | DEVELOPMENT or PRODUCTION | `PRODUCTION` |
| `POSTGRES_HOST` | DB hostname | `localhost` |
| `POSTGRES_PORT` | DB port | `5432` |
| `POSTGRES_DATABASE` | DB name | `payram` |
| `POSTGRES_USERNAME` | DB user | `payram` |
| `POSTGRES_PASSWORD` | DB password | (user-defined) |

---

## 20. Multi-Brand / Multi-Project Support

PayRam supports multiple brands/stores per account:
- Per-project: name, website URL, redirect URLs, logo, colors, social links
- Per-project: API keys, webhook endpoints, payment options
- Consolidated payouts and shared wallet balances
- Role-based access restrictions per project
