# Payminto — Build Documentation

**Project:** Payminto (Self-Hosted Crypto Payment Gateway)
**Based on:** PayRam product research & reverse engineering
**Date:** April 7, 2026
**Status:** Ready for Implementation

---

## What is Payminto?

A self-hosted, non-custodial cryptocurrency payment gateway for businesses and AI agents.

- **Self-hosted** — deploy on your own VPS, full control
- **Non-custodial** — merchant retains fund ownership
- **0% processing fees** — only gas + hosting costs
- **Multi-chain** — Bitcoin, Ethereum, Base, Polygon, Tron
- **Card-to-crypto onramp** — 175+ payment methods, 190+ countries
- **AI-native** — MCP (Model Context Protocol) integration
- **SmartSweep** — smart contract auto-sweep to cold storage

---

## Documentation Index

### Planning & Architecture (Original)

| Document | Lines | Description |
|----------|-------|-------------|
| [MASTER_PLAN.md](./MASTER_PLAN.md) | 647 | 12-phase implementation plan with tasks, team, budget |
| [PRODUCT_SPEC.md](./PRODUCT_SPEC.md) | 930 | Full product specification — features, flows, API, data models |
| [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) | 1,581 | PostgreSQL schema (74 tables), entity relationships, design patterns |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 811 | System architecture, tech stack, deployment topology |
| [ARCHITECTURE_DIAGRAMS.md](./ARCHITECTURE_DIAGRAMS.md) | 539 | Mermaid diagrams — ERD, sequence, state machines, deployment |
| [CARD_PAYMENT_FLOW.md](./CARD_PAYMENT_FLOW.md) | 740 | Card-to-crypto onramp flow deep analysis |
| [SOURCE_REFERENCE.md](./SOURCE_REFERENCE.md) | 70 | Go backend source structure — packages, models, functions |
| [COMPETITIVE_INTEL.md](./COMPETITIVE_INTEL.md) | 55 | Competitor analysis, SWOT, market positioning |

### Implementation-Ready Specifications (Architectural Audit)

| Document | Lines | Description |
|----------|-------|-------------|
| [ARCHITECTURAL_AUDIT.md](./ARCHITECTURAL_AUDIT.md) | 350 | **START HERE** — Master index, coverage matrix, gap analysis, readiness assessment |
| [API_SPECIFICATION.md](./API_SPECIFICATION.md) | 2,293 | 200+ endpoints, 27 domains, middleware, webhooks, error handling |
| [SERVICE_LAYER.md](./SERVICE_LAYER.md) | 1,559 | 45+ services, all method signatures, dependencies, business logic |
| [REPOSITORY_LAYER.md](./REPOSITORY_LAYER.md) | 1,599 | 50 repositories, query patterns, accounting, encryption, locking |
| [WORKER_SPECIFICATION.md](./WORKER_SPECIFICATION.md) | 1,512 | 13 background workers, block monitoring, scheduling, error handling |
| [BLOCKCHAIN_INTEGRATION.md](./BLOCKCHAIN_INTEGRATION.md) | 2,080 | Chain adapters, HD wallets, SmartSweep, RPC management, MCP tools |
| [MODEL_REFERENCE.md](./MODEL_REFERENCE.md) | 2,663 | 77 Go structs, all fields, relationships, state machines |

**Total documentation: 17,079 lines across 15 documents**

### Raw Data Files (`data/`)

| File | Description |
|------|-------------|
| `data/PAYRAM_GO_MODELS.txt` | All Go struct definitions from PayRam binary |
| `data/PAYRAM_FUNCTIONS.txt` | Complete function listing by package |
| `data/PAYRAM_SOURCE_PROJECTION.txt` | Source file structure with function signatures |
| `data/payram_schema_raw.sql` | Raw PostgreSQL schema (from initial extraction) |
| `data/payram_schema_only.sql` | Clean schema-only dump (CREATE TABLE, indexes, constraints) |
| `data/payram_full_dump.sql` | Full dump — schema + all data (2.6MB, ready to import) |
| `data/payram_key_tables_data.txt` | Human-readable dump of key reference tables (blockchains, currencies, configs, roles, etc.) |

---

## Tech Stack (Decided)

| Layer | Technology |
|-------|-----------|
| **Backend** | Go (Gin/Echo), PostgreSQL 14, Redis |
| **Frontend** | Next.js (TypeScript), Tailwind CSS, shadcn/ui |
| **MCP Server** | TypeScript (Node.js) |
| **Smart Contracts** | Solidity (Foundry) |
| **Deployment** | Docker Compose, Nginx, Let's Encrypt |
| **Monitoring** | Prometheus, Grafana, Loki |
| **Mobile** | React Native or Flutter |

## Blockchain Support

| Network | Tokens | Priority |
|---------|--------|----------|
| Bitcoin | BTC | Phase 2 |
| Ethereum | ETH, USDT, USDC | Phase 2 |
| Base | ETH, USDC, cbBTC | Phase 2 |
| Polygon | POL, USDC, USDT | Phase 2 |
| Tron | TRX, USDT | Phase 2 |
| Solana | SOL, USDC | Future |
| TON | Toncoin | Future |

## Target Directory Structure (When Built)

```
payminto/
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
├── landing/               # Marketing site (Next.js)
├── widget/                # Embeddable payment widget
├── scripts/               # Deployment scripts (setup.sh)
├── docker/                # Dockerfiles
├── docker-compose.yml
├── docker-compose.dev.yml
└── docs/                  # This documentation
```
