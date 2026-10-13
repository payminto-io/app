# Contributing to Payminto

Start with [README.md](README.md), [the module contract](docs/architecture/MODULES.md), and the issue for your change.
The repository is licensed under [MIT](LICENSE).
Preserve third-party notices and separately licensed dependencies and components.
Follow [AGENTS.md](AGENTS.md) when working with an agent.

## Choose and claim work

Open an issue with the problem, expected behavior, reproduction steps, and acceptance criteria before a substantial change.
Check the current task board and [.scratch/payments-v1/](.scratch/payments-v1/) for dependencies and ownership, then claim available work before editing.
Use an isolated branch and worktree, agree on file ownership for parallel work, and let the coordinator integrate dependent branches.
Report security vulnerabilities through [SECURITY.md](SECURITY.md), without publishing credentials or exploitable details in a public issue.

## Local setup

Use Go 1.25.4 or newer, Node.js 20.9 or newer with npm, PostgreSQL 16, and Redis 7.
Docker with Compose is needed for the development stack and the default PostgreSQL integration harness.
Install Foundry for contract work and Bun for `cre/` workflow work.
Each Node directory has its own manifest and lockfile; install only what you need:

```sh
npm ci --prefix frontend
npm ci --prefix checkout
npm ci --prefix landing
npm ci --prefix mcp-server
npm ci --prefix widget
```

For CRE, run `bun install --frozen-lockfile` inside `cre/`.

For source development, provision your own local PostgreSQL database and Redis instance first.
If `backend/.env` does not already exist, copy `.env.example` to it and edit the database connection to that disposable development database.
Use a database name ending in `_test`, `GATEWAY_ENVIRONMENT=test`, and `BLOCKCHAIN_NETWORK_TYPE=testnet`.
The example enables `POSTGRES_SCHEMA_MODE=auto-migrate`; confirm the target before startup and restrict this mode to development or test.
Run each required service in a separate terminal from the repository root:

```sh
make backend
NEXT_PUBLIC_API_URL=http://localhost:8090/api/v1 make frontend
make checkout
make landing
```

These targets serve the API on port 8090, dashboard on 3003, checkout on 3002, and landing site on 3001.
`make backend` loads `backend/.env`; a direct Go command does not load that file automatically.

Alternatively, `make dev` builds the development backend and starts PostgreSQL and Redis through `docker-compose.dev.yml`.
It serves the API on 8080 without source hot reload; check that ports 5432, 6379, and 8080 are free first.
Point separately started frontends at `http://localhost:8080/api/v1`, including `PAYMINTO_API_URL` for checkout.
Preserve development volumes when stopping services and never stop unrelated containers or prune a shared Docker host.

## Validate the affected components

Read test setup and database harness code before running a suite that can access a database.
Start with the changed package or feature, then run the relevant component checks:

| Component | Commands inside that directory |
| --- | --- |
| `backend/` | `go test ./...`, `go vet ./...`, `go build ./cmd/server/`, `golangci-lint run` |
| `frontend/` | `npm test`, `npm run lint`, `npm run build` |
| `checkout/` | `npm test`, `npm run lint`, `npm run build` |
| `landing/` | `npm run lint`, `npm run build` |
| `mcp-server/`, `widget/` | `npm run build` |
| `contracts/` | `forge test`, `forge fmt --check` |
| `cre/` | `bun test`, `bun run typecheck` |

The root `make test` covers backend, dashboard, and contracts only; check other changed packages explicitly.
For database behavior, inspect `backend/internal/database/testdb.go`, then run `make test-integration` from the root with `PAYMINTO_INTEGRATION_DATABASE_URL` and `PAYMINTO_INTEGRATION_ALLOW_SKIP` unset.
The default harness creates, migrates, and removes its own PostgreSQL 16 containers.
Docker failures must fail integration checks; skipped tests are not passing validation.

The optional `PAYMINTO_INTEGRATION_DATABASE_URL` path creates and drops isolated schemas in the supplied database.
Its URL check does not enforce disposable database naming, so never point it at application, shared, or live data.
Never route destructive tests through application connection variables.

For a running local environment, `make smoke-local` checks health, CORS, sign-in, and authenticated APIs.
It requires `jq` and admin/merchant credentials in `.dev-credentials.local.json`, or a file selected by `PAYMINTO_SMOKE_CREDENTIALS`.
Use `PAYMINTO_SMOKE_API_BASE=http://localhost:8080/api/v1` for the Compose backend.

## Protect payment and migration behavior

Keep backend code in Go, frontends in TypeScript, and capabilities behind module ports.
Preserve existing API payloads and make schema changes additive.
The migration runner applies checksummed migrations from `backend/internal/database/migrations/` and refuses `down`; add a new migration instead of rewriting an applied one.
Coordinate migration ordering and a roll-forward plan before merging.
Never drop, truncate, reset, or reseed shared data; inspect seed tools before using your disposable environment.

Keep ledger history append-only and derive amounts and statuses from ledger entries, connector receipts, or chain evidence.
Use decimal monetary arithmetic and preserve idempotency, retry safety, fee snapshots, explicit conversions, and live/test isolation.
Never accept or store raw card data; use tokenized or hosted payment fields.
Do not commit private keys, credentials, local environment files, customer data, or fabricated payment evidence.
Test failure, replay, and unknown-outcome behavior when changing money movement.

## Submit and complete a pull request

Link the issue and explain the resulting behavior and any migration or configuration changes.
Include checks and results, UI screenshots, and user-flow reproduction evidence for bug fixes.
State unrun checks and their blockers instead of marking work complete.
Format Go changes, follow neighboring TypeScript conventions, and put each full sentence on its own line in long Markdown documents.
Do not manually edit generated files or `CHANGELOG.md`.
Resolve review findings, validate integration, and update the task board after merge with the commit, checks, and remaining follow-up work.
