# Payminto

Self-hosted, non-custodial cryptocurrency payment gateway

## Quick Start

Install each Node workspace once, then run these commands in separate terminals:

```sh
make backend   # API: http://localhost:8090
make frontend  # dashboard: http://localhost:3003
make checkout  # hosted checkout: http://localhost:3002
make landing   # marketing site: http://localhost:3001
```

The backend target loads `backend/.env`; Go does not load that file by itself.
After the services are ready, verify health, CORS, admin/merchant sign-in, and
the core authenticated APIs with:

```sh
make smoke-local
```

The smoke command reads the disposable local accounts from
`.dev-credentials.local.json` without printing passwords or tokens.

## Architecture

Payminto is structured as a monorepo with the following subsystems:

- **Go Backend** (`backend/`) — REST API built with Gin, GORM, PostgreSQL 14, and Redis. Handles payment processing, blockchain monitoring, and sweep logic.
- **Next.js Frontend** (`frontend/`) — Merchant dashboard built with Next.js (App Router), TypeScript, Tailwind CSS, and shadcn/ui.
- **Hosted Checkout** (`checkout/`) — Isolated public payment UI on port 3002. It proxies only the backend's narrowed public checkout API and never receives dashboard credentials.
- **Solidity Contracts** (`contracts/`) — SmartSweep contracts for on-chain auto-sweep to cold storage, developed with Foundry.
- **MCP Server** (`mcp-server/`) — TypeScript/Node.js server exposing payment gateway capabilities to AI agents via the Model Context Protocol.

## License

MIT
