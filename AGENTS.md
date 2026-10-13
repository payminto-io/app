# Agent guidance

## Start here and own your scope

Read applicable parent `AGENTS.md`, root `CLAUDE.md`, the relevant directory guidance, and your task before editing.
Read `docs/architecture/MODULES.md` before backend work and `docs/design/DESIGN.md` plus `docs/design/references.md` before UI work.
Treat current code, manifests, and the root `LICENSE` as authoritative when historical guidance conflicts with them.
Do not repeat the blanket Apache licensing claim in `CLAUDE.md`.
Preserve applicable third-party notices and separately licensed dependencies.

Follow the active task board and confirm ownership before claiming unblocked, unowned work.
When using the legacy Payments V1 board, read `.scratch/payments-v1/spec.md` and tickets in `.scratch/payments-v1/issues/`.
For that board, claim the lowest unblocked `ready-for-agent` ticket with `Status: claimed`.
Use an isolated Git worktree and a branch named for the task.
Keep edits within the assigned module and coordinate shared files, task ownership, and migration numbering before merging.
Mark work done only after integration and relevant checks, recording commit hashes and any remaining limitations on the task.
Do not overwrite another agent's changes or manually edit `CHANGELOG.md` or generated files.
Put each full sentence on its own line and do not use em dashes.

## Repository map

| Directory | Responsibility |
| --- | --- |
| `backend/` | Go 1.25.4 API, services, repositories, workers, Postgres and Redis integration |
| `frontend/` | Next.js merchant dashboard, local Makefile port 3003 |
| `checkout/` | Next.js hosted checkout, port 3002 |
| `landing/` | Next.js marketing site, local Makefile port 3001 |
| `widget/` | TypeScript embeddable widget, bundled with esbuild |
| `mcp-server/` | TypeScript agent surface over the backend API |
| `contracts/` | Foundry Solidity contracts and tests |
| `cre/` | Chainlink CRE TypeScript workflows, Bun tests |
| `tracks/` | Hackathon track descriptions and demo evidence |

Product work belongs in this repository unless the owner explicitly authorizes another destination.
The original Payminto checkout is frozen reference, Hyperswitch is an architectural reference, and Kuberopay is a design reference with only the specifically authorized adapter ports listed in root `CLAUDE.md`.
Do not copy credentials or key files from reference repositories.

## Payment integrity and architecture

- Use Go for backend work and TypeScript for frontends.
- Use decimal monetary arithmetic, never floating-point amounts.
- Derive displayed amounts and statuses from ledger entries, connector receipts, or chain receipts.
- Only the ledger writes balances; money movement posts a journal.
- Snapshot fee rules on each payment and record conversions as explicit ledger trades.
- Keep card data off these servers through tokenized or hosted fields.
- Keep switch, treasury, and signer keys in separate services, databases, and secrets.
- Preserve boot-time test/live isolation, environment stamps, credential matching, and live-mode provider restrictions.
- Never store plaintext private keys or expose secrets in logs or output.
- Keep existing API payloads compatible; new module routes follow the snake_case contract in `MODULES.md`.

Modules depend on sibling ports, never sibling provider implementations or internal types.
Apply policy in core before calling providers and choose providers through configuration and module registries.
Each module owns its wiring file in `backend/internal/modules/`, routes file, tables, and versioned events.
Slot modules need a test mock and conformance tests for their providers.
Core must not import `backend/internal/ee/`.

Migrations are additive new files in `backend/internal/database/migrations/`.
The coordinator assigns the next migration number at merge time so applied history remains a contiguous manifest prefix.
Do not insert migrations before already applied history or rewrite applied migrations.
`POSTGRES_SCHEMA_MODE` defaults to validation; automatic migration is limited to development and test.

## Data and shared-machine safety

Read the setup and cleanup harness before the first run of any suite, seed, or script that touches a database.
Never drop, truncate, reset, bulk-delete, or reseed an existing shared or non-disposable target.
Destructive tooling must use dedicated test configuration or an explicit disposable target, refuse application connection variables, and enforce that guard in code.
Verify backups when created through non-zero size and a restorable listing.

Integration tests use `backend/internal/database/testdb.go` behind the `integration` build tag.
The default harness starts its own Postgres 16 testcontainer and terminates that container at cleanup.
The optional `PAYMINTO_INTEGRATION_DATABASE_URL` path creates and drops isolated schemas; use it only for an explicitly disposable test database after inspecting the harness.
Its URL parsing does not itself prove that the database is disposable.
Never point that override at the application database or a shared target.

Docker also hosts unrelated projects on this machine.
Never prune Docker resources, remove containers, stop containers you did not start, or delete shared volumes.
`make dev-down` uses `down -v`; do not use it against an existing shared stack.
If disk space prevents safe work, report the limitation instead of deleting resources.

## Run scoped checks

Reproduce bugs through the user flow before fixing them, then verify the changed behavior.
Start with checks for the changed packages or files and expand when the change crosses boundaries.
Inspect each test harness first and use isolated data for money-path and integration tests.

| Scope | Available checks |
| --- | --- |
| Backend | From `backend/`: `go test ./internal/<module>/...`, `go build ./...`, and `golangci-lint run` as applicable |
| Postgres integration | `make test-integration`, or a scoped `go test -tags=integration -count=1 ./internal/<module>/...` from `backend/` |
| Dashboard and checkout | From the package directory: `npm test`, `npm run lint`, `npm run build` |
| Landing | From `landing/`: `npm run lint`, `npm run build` |
| Widget and MCP server | From the package directory: `npm run build` |
| Contracts | From `contracts/`: scoped `forge test` |
| CRE workflows | From `cre/`: `npm run typecheck`, `bun test workflows` |
| Running local stack | `make smoke-local`, after reading `scripts/smoke-local.sh` and checking its targets and credential file |

The frontend Vitest setup uses jsdom and Testing Library cleanup.
Integration tests require a healthy Docker provider by default; skipping them is not evidence that they passed.
`make test` covers backend, dashboard, and contracts only, so check other changed packages explicitly.
For Next.js changes, follow nested `AGENTS.md` and read the installed `node_modules/next/dist/docs/` guidance before coding.
Keep the generated Next.js guidance blocks intact.
Verify UI changes with screenshots at 390px and 1440px in light and dark themes.
Report the checks actually run and any blockers without presenting unrun checks as passed.
