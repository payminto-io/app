# Payminto — Architectural Audit & Documentation Index

> **Version:** 1.0 | **Date:** April 7, 2026  
> **Source:** Reverse-engineered from PayRam Go binary via Ghidra + PostgreSQL dump from Docker container  
> **Status:** Pre-implementation — all documentation complete, ready for scaffolding

---

## 1. Executive Summary

The PayRam codebase has been fully reverse-engineered, analyzed, and documented across **17,079 lines** of implementation-ready documentation. This audit covers every layer of the system — from API endpoints through service logic, repository patterns, background workers, blockchain integration, and data models.

### What Was Extracted (via Ghidra)

| Artifact | Count | Source |
|----------|-------|--------|
| Go model structs | 77 | Binary reverse-engineering |
| Functions (all packages) | 3,450+ | Binary reverse-engineering |
| Packages/modules | 47 | Binary reverse-engineering |
| PostgreSQL tables | 74 | Docker container SQL dump |
| Foreign key relationships | 125+ | SQL dump analysis |
| Database indexes | 86+ | SQL dump analysis |
| API endpoints | 200+ | Handler function analysis |
| Background workers | 13 | Processor function analysis |
| Service implementations | 45+ | Service layer analysis |
| Repository implementations | 50 | Repository layer analysis |

### Documentation Corpus

| Document | Lines | Focus |
|----------|-------|-------|
| **Existing (pre-audit)** | | |
| MASTER_PLAN.md | 647 | 12-phase implementation roadmap |
| PRODUCT_SPEC.md | 930 | Product features, flows, personas |
| DATABASE_SCHEMA.md | 1,581 | 74 tables, all columns, FKs, indexes |
| ARCHITECTURE.md | 811 | System design, security, deployment |
| ARCHITECTURE_DIAGRAMS.md | 539 | Mermaid diagrams for all flows |
| CARD_PAYMENT_FLOW.md | 740 | Deep card-to-crypto + regulatory analysis |
| SOURCE_REFERENCE.md | 70 | Index to raw reference files |
| COMPETITIVE_INTEL.md | 55 | Competitive landscape |
| **New (this audit)** | | |
| API_SPECIFICATION.md | 2,293 | 200+ endpoints, middleware, webhooks, errors |
| SERVICE_LAYER.md | 1,559 | 45+ services, all methods, dependencies |
| REPOSITORY_LAYER.md | 1,599 | 50 repositories, query patterns, accounting |
| WORKER_SPECIFICATION.md | 1,512 | 13 workers, block monitoring, scheduling |
| BLOCKCHAIN_INTEGRATION.md | 2,080 | Chain adapters, HD wallets, SmartSweep, MCP |
| MODEL_REFERENCE.md | 2,663 | 77 models, all fields, state machines |
| **TOTAL** | **17,079** | |

### Raw Reference Data

| File | Size | Content |
|------|------|---------|
| PAYRAM_GO_MODELS.txt | 31 KB | 77 struct definitions with all fields |
| PAYRAM_FUNCTIONS.txt | 239 KB | 3,450+ functions with signatures and line numbers |
| PAYRAM_SOURCE_PROJECTION.txt | 215 KB | Complete file/function/package organization |
| payram_schema_raw.sql | 243 KB | Importable PostgreSQL dump (74 tables) |

---

## 2. Coverage Matrix

### By System Layer

| Layer | Documented? | Document | Key Content |
|-------|-------------|----------|-------------|
| **API Endpoints** | Full | API_SPECIFICATION.md | 200+ endpoints, 27 domains, 43 handler groups |
| **Middleware** | Full | API_SPECIFICATION.md | 8 middleware types with chain details |
| **Service Layer** | Full | SERVICE_LAYER.md | 45+ services, all method signatures |
| **Repository Layer** | Full | REPOSITORY_LAYER.md | 50 repos, 425+ functions, ~12K LOC |
| **Data Models** | Full | MODEL_REFERENCE.md | 77 structs, all fields, relationships |
| **Database Schema** | Full | DATABASE_SCHEMA.md | 74 tables, 125+ FKs, 86+ indexes |
| **Background Workers** | Full | WORKER_SPECIFICATION.md | 13 workers, scheduling, error handling |
| **Blockchain Clients** | Full | BLOCKCHAIN_INTEGRATION.md | 5 chains, 40+ client methods |
| **HD Wallet/Keys** | Full | BLOCKCHAIN_INTEGRATION.md | BIP-39/32/44, zero-exposure model |
| **SmartSweep Contracts** | Full | BLOCKCHAIN_INTEGRATION.md | Solidity interface, SCW deployment |
| **Block Monitoring** | Full | BLOCKCHAIN_INTEGRATION.md + WORKER_SPECIFICATION.md | Per-chain processors |
| **RPC Management** | Full | BLOCKCHAIN_INTEGRATION.md | Multi-provider, failover, health checks |
| **Payment Flows** | Full | CARD_PAYMENT_FLOW.md + BLOCKCHAIN_INTEGRATION.md | Crypto + Card + Widget |
| **Webhook System** | Full | API_SPECIFICATION.md + WORKER_SPECIFICATION.md | Events, HMAC, retry |
| **Referral System** | Full | SERVICE_LAYER.md + API_SPECIFICATION.md | Campaigns, rewards, analytics |
| **Analytics** | Full | SERVICE_LAYER.md + API_SPECIFICATION.md | Custom filters, graphs |
| **Double-Entry Accounting** | Full | REPOSITORY_LAYER.md + MODEL_REFERENCE.md | Asset/Liability/Revenue/Expense |
| **Auth & RBAC** | Full | API_SPECIFICATION.md + SERVICE_LAYER.md | JWT, API keys, roles, OTP |
| **MCP/AI Agent** | Full | BLOCKCHAIN_INTEGRATION.md | 10 tools, protocols |
| **Card-to-Crypto Onramp** | Full | CARD_PAYMENT_FLOW.md | TransFi + Cards channels |
| **Configuration** | Partial | API_SPECIFICATION.md | Env vars listed, defaults TBD |
| **Email/Notifications** | Partial | WORKER_SPECIFICATION.md | Email processor, 5 event types |
| **Frontend** | Partial | ARCHITECTURE.md | Routes, components listed |
| **Smart Contracts (Solidity)** | Partial | BLOCKCHAIN_INTEGRATION.md | Interface documented, ABI TBD |
| **Mobile App** | Minimal | PRODUCT_SPEC.md | 3 features listed |
| **SDK/Client Libraries** | Minimal | PRODUCT_SPEC.md | Mentioned, not specified |

### By Domain

| Domain | Tables | Models | Services | Repos | Endpoints | Workers |
|--------|--------|--------|----------|-------|-----------|---------|
| Auth & Members | 8 | 11 | 7 | 8 | 30+ | 0 |
| Blockchain | 8 | 10 | 4 | 8 | 22+ | 5 |
| Wallets & Keys | 7 | 9 | 6 | 6 | 15+ | 1 |
| Payments | 5 | 6 | 3 | 4 | 25+ | 1 |
| Deposits | 3 | 5 | 2 | 3 | 8+ | 1 |
| Sweeps & UTXOs | 4 | 4 | 4 | 3 | 7+ | 2 |
| Withdrawals | 3 | 3 | 2 | 2 | 13+ | 0 |
| Accounting | 5 | 6 | 2 | 3 | 0 | 1 |
| Webhooks | 2 | 2 | 2 | 2 | 4+ | 1 |
| External Platforms | 4 | 4 | 3 | 3 | 18+ | 0 |
| Referrals | 10 | 8 | 3 | 1 | 29+ | 0 |
| Analytics | 7 | 6 | 2 | 1 | 3+ | 0 |
| System & Config | 8 | 6 | 5 | 5 | 20+ | 1 |
| **TOTAL** | **74** | **77** | **45+** | **50** | **200+** | **13** |

---

## 3. Architecture Summary

### System Topology

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENTS                               │
│  Merchant Dashboard │ Payment Widget │ AI Agents │ Mobile    │
└─────────┬───────────┴───────┬────────┴─────┬─────┴──────────┘
          │                   │              │
          ▼                   ▼              ▼
┌─────────────────────────────────────────────────────────────┐
│                    NGINX REVERSE PROXY                        │
│              TLS 1.3 │ Rate Limiting │ CORS                  │
└─────────┬───────────────────┬──────────────┬────────────────┘
          │                   │              │
          ▼                   ▼              ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Go API      │  │  Next.js     │  │  MCP Server  │
│  (Gin)       │  │  Dashboard   │  │  (Node.js)   │
│  Port 8080   │  │  Port 3000   │  │  Port 3333   │
└──────┬───────┘  └──────────────┘  └──────┬───────┘
       │                                    │
       ├────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    SERVICE LAYER (45+ services)               │
│  Auth │ Payment │ Wallet │ Blockchain │ Sweep │ Withdrawal   │
│  Referral │ Analytics │ Webhook │ Platform │ Config │ OTP    │
└─────────┬───────────────────┬──────────────┬────────────────┘
          │                   │              │
          ▼                   ▼              ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  PostgreSQL  │  │    Redis     │  │  Encrypted   │
│  14 (74 tbl) │  │   (Cache)    │  │  Key Store   │
└──────────────┘  └──────────────┘  └──────────────┘
                                           │
       ┌───────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│              BACKGROUND WORKERS (13 goroutines)              │
│  Account Processor │ BTC/ETH/BASE/TRX/Polygon Processors   │
│  Deposit Processor │ Webhook Processor │ Email Processor     │
│  ERC20 Sweep Approval │ SCW Broadcaster │ Seeder │ Dedup    │
└─────────┬───────────────────┬──────────────┬────────────────┘
          │                   │              │
          ▼                   ▼              ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Bitcoin     │  │  EVM Chains  │  │    Tron      │
│  (UTXO)     │  │ ETH/Base/Pol │  │  (TRC-20)    │
└──────────────┘  └──────────────┘  └──────────────┘
          │                   │              │
          └─────────┬─────────┘──────────────┘
                    ▼
          ┌──────────────────┐
          │  SmartSweep      │
          │  Contracts       │
          │  → Cold Wallet   │
          └──────────────────┘
```

### Key Design Decisions (Confirmed from Source)

1. **Non-custodial by design** — Master seed never on server, only XPUBs for address generation
2. **Double-entry accounting** — Every financial operation creates Asset/Liability/Revenue/Expense entries
3. **Multi-tenant via ExternalPlatform** — Each merchant project is an ExternalPlatform with isolated wallets, keys, and webhooks
4. **GORM with soft deletes** — All business entities use `deleted_at` for logical deletion
5. **decimal(38,18) for all amounts** — Prevents floating-point precision loss in crypto calculations
6. **Per-chain block processors** — Separate goroutines for BTC, ETH, Base, Polygon, Tron
7. **SmartSweep immutability** — Cold wallet address hardcoded at contract deployment, cannot be changed
8. **RSA + AES-256 key encryption** — Hot wallet keys RSA-encrypted with nonce, master keys AES-encrypted
9. **Dynamic polling intervals** — Block processors adjust polling based on block time history
10. **HMAC-SHA256 webhook signatures** — With timestamp verification and idempotency keys

---

## 4. Gap Analysis

### Fully Documented (Implementation-Ready)

- All API endpoints with handler → service → repository chain
- All 77 data models with complete field definitions
- All 50 repository implementations with query patterns
- All 45+ service interfaces with method signatures
- All 13 background workers with scheduling and error handling
- Blockchain client architecture for 5 chains
- HD wallet key management and derivation paths
- SmartSweep contract interface and deployment flow
- Double-entry accounting system
- Webhook delivery system with retry strategy
- Payment detection and confirmation flow
- UTXO management for Bitcoin
- Card-to-crypto onramp (TransFi + Cards channels)
- MCP/AI agent integration (10 tools)

### Partially Documented (Needs Enrichment During Implementation)

| Gap | What's Missing | Where to Find It | Priority |
|-----|---------------|-------------------|----------|
| **Request/Response Bodies** | Some endpoints have inferred (not confirmed) payloads | Implement and test against actual PayRam instance | Medium |
| **Error Codes** | Standard format documented, specific codes per endpoint TBD | Define during API implementation | Medium |
| **Configuration Defaults** | Env vars listed, default values not all confirmed | Check PayRam Docker container env | Low |
| **Rate Limit Thresholds** | Strategy documented, specific limits TBD | Define during deployment | Low |
| **Smart Contract ABI** | Interface documented, full ABI not extracted | Extract from deployed contracts or rewrite | High |
| **Frontend Components** | Routes listed, component props/state not documented | Implement from screenshots + PRODUCT_SPEC | Medium |
| **Email Templates** | 5 event types documented, template HTML not extracted | Create during implementation | Low |
| **RPC Node URLs** | Provider types listed, actual endpoints TBD per deployment | Configure per merchant deployment | Low |

### Not Documented (Out of Scope for This Audit)

- Mobile app (React Native / Flutter) — only 3 features listed
- SDK client libraries — mentioned but not specified
- Monitoring/alerting rules — mentioned in architecture
- Performance benchmarks — no targets defined
- Disaster recovery procedures — mentioned, not detailed
- Load testing strategy — not documented

---

## 5. Implementation Readiness Assessment

### Ready to Build (Phase 0-2)

| Component | Documentation | Confidence | Notes |
|-----------|--------------|------------|-------|
| Monorepo scaffold | MASTER_PLAN.md Phase 0 | High | Standard Go + Next.js layout |
| Docker dev environment | DATABASE_SCHEMA.md + ARCHITECTURE.md | High | PostgreSQL + Redis + Go + Node |
| Database migrations | DATABASE_SCHEMA.md + payram_schema_raw.sql | Very High | Importable SQL dump available |
| Go models (GORM) | MODEL_REFERENCE.md | Very High | All 77 structs fully defined |
| Repository layer | REPOSITORY_LAYER.md | High | All 50 repos with methods |
| Service layer | SERVICE_LAYER.md | High | All 45+ services with signatures |
| API handlers + routes | API_SPECIFICATION.md | High | 200+ endpoints mapped |
| Middleware chain | API_SPECIFICATION.md | High | 8 middleware types documented |
| Auth system (JWT + API keys) | API_SPECIFICATION.md + SERVICE_LAYER.md | High | Full auth flow documented |
| Background workers | WORKER_SPECIFICATION.md | High | 13 workers fully specified |
| Blockchain adapters | BLOCKCHAIN_INTEGRATION.md | High | 40+ client methods per chain |
| HD wallet management | BLOCKCHAIN_INTEGRATION.md | High | BIP paths, key management |
| Payment flow | BLOCKCHAIN_INTEGRATION.md + CARD_PAYMENT_FLOW.md | Very High | End-to-end flow documented |
| Webhook system | API_SPECIFICATION.md + WORKER_SPECIFICATION.md | High | Events, HMAC, retry strategy |

### Needs Design Decisions (Phase 3-4)

| Component | Gap | Decision Needed |
|-----------|-----|----------------|
| SmartSweep Solidity | Interface known, ABI not extracted | Write contracts from documented interface or extract ABI |
| Next.js Dashboard | Routes known, components not | Design from screenshots (56 available) |
| MCP Server | 10 tools documented | Implement TypeScript server from tool specs |
| Card-to-Crypto | Two channels documented | Choose: TransFi direct (simpler) vs customer wallet (compliant) |

---

## 6. Document Navigation Guide

### "I want to implement..."

| Task | Start With | Then Read |
|------|-----------|-----------|
| Database setup | DATABASE_SCHEMA.md | payram_schema_raw.sql (importable) |
| Go models | MODEL_REFERENCE.md | PAYRAM_GO_MODELS.txt (raw structs) |
| API endpoint | API_SPECIFICATION.md → find endpoint | SERVICE_LAYER.md → find service | REPOSITORY_LAYER.md → find repo |
| Background worker | WORKER_SPECIFICATION.md → find worker | SERVICE_LAYER.md → find dependencies |
| Blockchain integration | BLOCKCHAIN_INTEGRATION.md | PAYRAM_FUNCTIONS.txt (search for chain) |
| Payment flow | CARD_PAYMENT_FLOW.md | BLOCKCHAIN_INTEGRATION.md §8 |
| Sweep system | BLOCKCHAIN_INTEGRATION.md §5 | WORKER_SPECIFICATION.md §I, §J |
| Auth system | API_SPECIFICATION.md §2-3 | SERVICE_LAYER.md → AuthService |
| Referral system | API_SPECIFICATION.md → Referral | SERVICE_LAYER.md → ReferralService |
| Accounting | REPOSITORY_LAYER.md §4 | MODEL_REFERENCE.md §I |
| Frontend dashboard | ARCHITECTURE.md §3 | PRODUCT_SPEC.md §5 |
| Deployment | ARCHITECTURE.md §4 | MASTER_PLAN.md Phase 0 |

### "I want to understand..."

| Question | Document |
|----------|----------|
| What does the system do? | PRODUCT_SPEC.md |
| How is it built? | ARCHITECTURE.md |
| What's the implementation plan? | MASTER_PLAN.md |
| How does payment detection work? | BLOCKCHAIN_INTEGRATION.md §8 |
| How does card-to-crypto work? | CARD_PAYMENT_FLOW.md |
| What tables exist? | DATABASE_SCHEMA.md |
| What Go structs exist? | MODEL_REFERENCE.md |
| What APIs are available? | API_SPECIFICATION.md |
| How do services interact? | SERVICE_LAYER.md §2 (dependency graph) |
| What runs in the background? | WORKER_SPECIFICATION.md |
| How are keys managed? | BLOCKCHAIN_INTEGRATION.md §4 |
| How does sweeping work? | BLOCKCHAIN_INTEGRATION.md §5 |
| Who are the competitors? | COMPETITIVE_INTEL.md |

---

## 7. Reverse Engineering Methodology

### Tools Used

1. **Ghidra** (NSA) — Primary tool for reverse-engineering the Go binary extracted from PayRam Docker container
   - Extracted function signatures, struct definitions, package organization
   - Mapped call graphs between services, repositories, and handlers
   - Identified background worker goroutine entry points

2. **PostgreSQL dump** — `pg_dump` from the running Docker container
   - 74 tables with complete DDL (columns, types, constraints, indexes)
   - Foreign key relationships mapped
   - Importable SQL for local development

3. **Docker container inspection** — Runtime analysis
   - Environment variables extracted
   - Port mappings and service configuration
   - File system structure of deployed binary

### Extraction Pipeline

```
PayRam Docker Image
    │
    ├─► pg_dump ──► payram_schema_raw.sql (243 KB)
    │                   └─► DATABASE_SCHEMA.md (1,581 lines)
    │
    ├─► Ghidra Binary Analysis
    │       │
    │       ├─► PAYRAM_GO_MODELS.txt (31 KB, 77 structs)
    │       │       └─► MODEL_REFERENCE.md (2,663 lines)
    │       │
    │       ├─► PAYRAM_FUNCTIONS.txt (239 KB, 3,450+ functions)
    │       │       ├─► SERVICE_LAYER.md (1,559 lines)
    │       │       ├─► REPOSITORY_LAYER.md (1,599 lines)
    │       │       ├─► WORKER_SPECIFICATION.md (1,512 lines)
    │       │       └─► API_SPECIFICATION.md (2,293 lines)
    │       │
    │       └─► PAYRAM_SOURCE_PROJECTION.txt (215 KB)
    │               └─► BLOCKCHAIN_INTEGRATION.md (2,080 lines)
    │
    ├─► Runtime Testing + UI Analysis
    │       ├─► PRODUCT_SPEC.md (930 lines)
    │       ├─► CARD_PAYMENT_FLOW.md (740 lines)
    │       └─► 56 screenshots (demo app, landing, docs)
    │
    └─► Architecture Analysis
            ├─► ARCHITECTURE.md (811 lines)
            ├─► ARCHITECTURE_DIAGRAMS.md (539 lines)
            └─► MASTER_PLAN.md (647 lines)
```

---

## 8. Next Steps

### Immediate (Phase 0 — Scaffolding)

1. **Initialize monorepo** — Go backend, Next.js frontend, MCP server, contracts, Docker
2. **Import database schema** — Run `payram_schema_raw.sql` or generate GORM migrations from MODEL_REFERENCE.md
3. **Generate Go models** — Translate all 77 structs from MODEL_REFERENCE.md to Go code
4. **Set up Docker Compose** — PostgreSQL 14, Redis, Go API, Next.js, Nginx
5. **Configure CI/CD** — Linting, testing, build pipeline

### Phase 1 — Core Backend

6. **Implement repository layer** — 50 repositories from REPOSITORY_LAYER.md
7. **Implement service layer** — 45+ services from SERVICE_LAYER.md
8. **Implement API handlers** — 200+ endpoints from API_SPECIFICATION.md
9. **Implement middleware** — 8 middleware types from API_SPECIFICATION.md
10. **Implement auth system** — JWT + API key + OTP from SERVICE_LAYER.md

### Phase 2 — Blockchain

11. **Implement chain adapters** — ETH, BTC, Tron from BLOCKCHAIN_INTEGRATION.md
12. **Implement block processors** — 5 chain workers from WORKER_SPECIFICATION.md
13. **Implement HD wallet** — Key derivation from BLOCKCHAIN_INTEGRATION.md §4
14. **Implement all workers** — 13 workers from WORKER_SPECIFICATION.md

### Phase 3+ — Frontend, Contracts, MCP

15. **Write SmartSweep contracts** — From BLOCKCHAIN_INTEGRATION.md §5
16. **Build merchant dashboard** — From ARCHITECTURE.md + PRODUCT_SPEC.md + screenshots
17. **Build MCP server** — From BLOCKCHAIN_INTEGRATION.md §12
18. **Build payment widget** — From PRODUCT_SPEC.md

---

*This audit confirms the PayRam codebase has been comprehensively reverse-engineered and documented. All architectural layers, business logic, and data structures are captured at implementation-ready detail. The documentation corpus (17,079 lines + 728 KB raw reference data) provides sufficient foundation to rebuild the system as Payminto.*
