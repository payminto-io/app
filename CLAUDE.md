# gateway — agent instructions

## What this is

This repository is the payments product: an open-source, self-hostable payment gateway where fiat (cards, UPI, bank) and crypto (stablecoins, major assets) are peer rails, settlement is on-chain with Solana first, custody is a pluggable provider, and AI agents are first-class payers. It is built end to end here, with its own git history, starting from Payminto.

Decided on 2026-10-07 after false starts; do not reopen without the owner:

- **Base: Payminto.** This repository was bootstrapped from `~/project/product2026/payramclone/payminto` (Go backend, Next.js dashboard, checkout and landing, Solidity sweep contracts, MCP server). That folder is now frozen reference. Never commit product work there.
- **Kuberopay is a reference for designs, not a source of code.** `~/project/client/techfleek/paywingProject/Kuberopay` is a client's repository. Its ledger model, versioned fee rules, payment-link object and form, settlement policy and onboarding gate are adopted as designs and re-implemented here in Go. Copy a file from it only if the owner says that file is theirs to reuse. Never commit product work there.
- **Hyperswitch is a blueprint, not a dependency.** `~/project/client/techfleek/hyperswitch` is a reference checkout for the switch core's shape (intent and attempt, connector status map, scheduler).

If a task would change one of those three folders, stop: the work belongs here.

## Where things are

- `backend/` Go 1.25, Gin, GORM on Postgres 16, Redis; `cmd/server`, `cmd/migrate`, `cmd/devseed`; `internal/{api,blockchain,database,models,repository,service,worker,paymentlifecycle,...}`.
- `frontend/` merchant dashboard (Next.js, port 3003). `checkout/` hosted checkout (3002). `landing/` marketing site (3001). `widget/` embeddable widget. `mcp-server/` agent surface.
- `contracts/` Foundry: `SmartSweep.sol`, `DepositProxy.sol`, `AddressFactory.sol`.
- `.scratch/payments-v1/` the execution board: `spec.md` (program spec and team roles) and `issues/NN-*.md` tickets with owner roles. Agents pick the lowest unblocked `ready-for-agent` ticket.
- `docs/superpowers/plans/` bite-sized implementation plans; `docs/` Payminto's architecture, security and blockchain documents, still accurate for the inherited code.
- Product definition and research: `../reports/` (one level up).

## Rules that outrank the rest

- Every number shown is a ledger line or a connector or chain receipt. No placeholders, no inferred statuses.
- No card data on these servers; tokenised or hosted fields only.
- Live and test are isolated at boot; the development keystore refuses to run live.
- Switch, treasury and signer keys are separate services, databases and secrets.
- Fee rule snapshot on every payment; conversions are explicit ledger trades.
- Additive schema changes only; existing API payloads keep working.
- One backend language: Go. Frontends are TypeScript.
- Open core: everything is Apache 2.0 unless the spec's enterprise list names it (white-label, SSO, extra custody adapters, PCI vault, extra conversion and payout connectors). Enterprise code lives only in `backend/internal/ee/` and `frontend/ee/`, behind a licence key, and core never imports it.

## Modules

Every capability is a module behind a port, with providers chosen by configuration. Read `docs/architecture/MODULES.md` before writing backend code; it is the contract that lets agents work in parallel.

## Running and checks

```bash
make backend      # API :8090, loads backend/.env
make frontend     # dashboard :3003
make checkout     # checkout :3002
make landing      # landing :3001
cd backend && go test ./...
make test-integration   # Postgres testcontainers; fails loudly without Docker
make smoke-local  # health, CORS, admin and merchant sign-in, core APIs
```

Environment: `GATEWAY_ENVIRONMENT=test|live` (default `test`), one per process. A test process needs a database named `*_test` or on a loopback host; a live process refuses the test database, `DEV_KEYSTORE`/`AES_KEY`, any `*_PROVIDER=mock`, a non-hardened `SERVER`, weak `POSTGRES_SSL_MODE` and a non-mainnet network, all before opening a connection (`internal/environment`, `internal/modules/environment.go`). API keys are `sk_test_`/`sk_live_`; the auth middleware answers 401 `api_key_environment_mismatch` to a key from the other environment and `session_environment_mismatch` to a JWT minted by the other one. The database carries a one-row `gateway_environment` stamp that every boot checks and that a process writes only onto an empty database; a pre-ticket database boots only after `cmd/migrate adopt-live` or `adopt-test` (docs/OPERATIONS.md). A remote test database without a `_test` suffix needs `GATEWAY_TEST_DATABASE_NAME`.

Database: `POSTGRES_SCHEMA_MODE` unset means validate only; `auto-migrate` is allowed in development and test only (`internal/database/startup.go`). Integration tests use a Postgres testcontainer behind the `integration` build tag (`internal/database/testdb.go`). Never point tests at a shared database.
