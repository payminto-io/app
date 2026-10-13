# Payminto

An open-source, self-hostable payment gateway for merchants, marketplaces, and fintech teams that want to operate their own stablecoin payment infrastructure.
Payment links, hosted checkout, a merchant dashboard, and a Go backend reconcile deposits and fees through one double-entry ledger.

| Product links | URL |
| --- | --- |
| Website | [payminto.io](https://payminto.io) |
| Merchant dashboard | [app.payminto.io](https://app.payminto.io) |
| Hosted checkout | [checkout.payminto.io](https://checkout.payminto.io) |
| Source | [github.com/payminto-io/app](https://github.com/payminto-io/app) |

These are existing public deployments.
The integrated Tempo update has been verified locally; its rollout to these public services is unverified.
See [all links and legacy media](website.md).

## Crypto World's Fair

The current showcase is for [Colosseum's Crypto World's Fair](https://colosseum.com/worldsfair), with Solana and Tempo as the relevant ecosystem tracks.
Read the [project details and disclosures](docs/WORLDS_FAIR.md) and [demo walkthrough](DEMO.md).

| Track | Implementation and evidence |
| --- | --- |
| [Solana](tracks/solana/README.md) | USDC and USDT deposit detection, ledger posting, and crash-safe sweeps; the existing public test deployment advertises Solana USDC. |
| [Tempo](tracks/tempo/README.md) | Memo-tagged TIP-20 payments on Moderato testnet, wallet checkout, and PostgreSQL ledger regression tests; an original owner-run 1.5 pathUSD payment was credited through a local backend. |

## What is implemented

- An append-only double-entry ledger with separate assets, auditable deposits, and versioned fee snapshots.
- A payment switch with intents, attempts, idempotency, connector evidence, and reconciliation of unknown outcomes.
- Payment links with server-side checkout previews and a hosted checkout with explicit partial, overpaid, confirming, and paid states.
- Solana USDC/USDT watchers and sweep recovery across crashes, delayed RPC answers, and competing workers.
- Tempo invoice memos, receipt and token verification, deduplication, retryable finalization, and exact token amounts in checkout.
- Test/live separation at boot and environment-bound API credentials.

Tempo supports Moderato testnet only and refuses live money configuration.
Card and UPI connectors, custody adapters, cross-rail routing, conversion, and a complete one-command deployment remain unfinished or unmerged.
The inherited MCP server exposes the backend API to AI agents.
Configured CI gates cover backend, dashboard, checkout, contracts, and Tempo PostgreSQL regressions; local verification is separate from a hosted CI run.

## Prior work and AI use

Payminto predates this event.
Commit [fa8517f](https://github.com/payminto-io/app/commit/fa8517fbba5c3b375513b3b773a0a518c84a5d98) imported existing private gateway code and local changes on October 6, 2026 UTC.
Earlier Bitcoin, EVM, and Tron clients, the MCP server, and widget are prior work.
Post-import work includes the ledger, fees, switch, payment links, Solana work, UI, docs, and the October 13 IST Tempo feature and QA.
Claude Code and Codex contributed to development, review, testing, and documentation.
The [detailed disclosure](docs/WORLDS_FAIR.md#prior-work-and-ai-disclosure) separates these contributions.

## Run locally

Follow [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, isolated database setup, and package installation.
From the repository root, run each service in a separate terminal:

```sh
make backend
NEXT_PUBLIC_API_URL=http://localhost:8090/api/v1 make frontend
make checkout
make landing
```

The source API uses port 8090, dashboard 3003, checkout 3002, and landing site 3001.
The backend target loads `backend/.env`; checkout uses Webpack for development and points at that API.
The optional Compose development stack uses API port 8080 instead.

Run checks for the components you change, after inspecting database test harnesses:

```sh
(cd backend && go test ./... && go vet ./...)
npm test --prefix checkout
npm run lint --prefix checkout
npm run build --prefix checkout
make test-integration
```

Integration tests need disposable PostgreSQL targets and a healthy Docker provider by default.
Never use a shared application database for tests or seeds.
See [CONTRIBUTING.md](CONTRIBUTING.md) for dashboard, contract, and other package checks, and for authenticated local smoke checks.

## Repository map

| Path | Purpose |
| --- | --- |
| `backend/` | Go API, ledger, fee rules, payment switch, chain watchers, and workers. |
| `frontend/`, `checkout/`, `landing/` | Merchant dashboard, hosted checkout, and product website. |
| `mcp-server/`, `widget/` | Inherited agent API surface and embeddable widget. |
| `contracts/`, `cre/` | Retained EVM contracts and optional attestation implementation. |
| `docs/WORLDS_FAIR.md`, `tracks/` | Current project details, disclosures, and Solana/Tempo evidence. |
| `.scratch/payments-v1/` | Development specification and historical execution tickets. |

MIT, see [LICENSE](LICENSE), with separately licensed dependencies and notices preserved.
