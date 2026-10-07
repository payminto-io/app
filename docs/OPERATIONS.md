# Payminto — Operations and Development Deployment Guide

Operational reference for the current development deployment. This repository
does **not** yet ship a production-ready Compose topology. Production deployment
remains blocked on versioned migrations, PostgreSQL TLS, backup/restore proof,
and the release gates in
`docs/superpowers/specs/2026-05-25-payminto-production-readiness-design.md`.

## Configuration

All backend config is via environment variables (see `payminto/.env.example` for
the complete annotated list). Constants that are *not* environment-driven live
in one place: `backend/internal/constants/` (intervals, cache TTLs, external
endpoints, SSE/metrics parameters). The frontend mirrors this in
`frontend/lib/constants.ts`.

Required in every environment:

- `POSTGRES_*`, `REDIS_URL`.
- `BLOCKCHAIN_NETWORK_TYPE` — `testnet` | `mainnet` (enforced against the DB).

Required in staging/production:

- `JWT_SECRET` — at least 32 non-placeholder characters from a secrets manager.
- `POSTGRES_PASSWORD` — at least 16 non-placeholder characters.
- `POSTGRES_SCHEMA_MODE=validate`.
- A database role for the server that is not the migration role (see Ledger roles below).
  The server refuses to boot in staging and production when its role can rewrite the ledger.
- `POSTGRES_SSL_MODE=verify-full` for a remote database, with its CA available
  to the runtime. The only insecure exception is an explicitly approved
  loopback database with `POSTGRES_ALLOW_INSECURE_LOCAL=true`.

Optional / feature flags:

- `CUSTODY_ENABLED` — enables vault unlock, wallet derivation, and sweep
  workers. When enabled, `VAULT_PASSPHRASE` must contain at least 24
  non-placeholder characters and come from a secrets manager outside local
  development.
- `AES_KEY` — retained for legacy configuration compatibility but not consumed
  by the current SecretsVault-backed custody implementation.
- `CORS_ALLOWED_ORIGINS` — comma-separated. Empty = any origin without
  credentials (safe default). Set explicit origins for cookie-based flows.
- `METRICS_ENABLED` (default `true`) — exposes Prometheus `/metrics`.
- `SENTRY_DSN` — enables error reporting; empty = structured logging only.
- `POSTGRES_LEDGER_APP_ROLE` — role that `cmd/migrate` narrows to `SELECT, INSERT`
  on the ledger tables after applying migrations (also `--ledger-app-role`).
- `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` — enables real email; otherwise emails
  are logged (no-op transport).

## Database migrations

The development stacks explicitly select `POSTGRES_SCHEMA_MODE=auto-migrate`.
The server permits that mode only under `SERVER=DEVELOPMENT` or `SERVER=TEST`.
Staging and production always validate and never mutate schema at startup.

The current `cmd/migrate` path also relies on GORM AutoMigrate and is therefore
a development bootstrap tool, not an approved production migration pipeline.
Do not point either AutoMigrate path at a production or legacy PayRam database.
A future release must provide checksummed versioned migrations, legacy-schema
transformation, backup-before-change, rollback constraints, and PostgreSQL
integration evidence before a production topology is published.

### Ledger roles

The double-entry ledger (`ledger_accounts`, `ledger_journals`, `ledger_lines`) is append-only
by trigger. A role that owns those tables can disable the triggers, so two roles are required:

- A privileged migration role runs `cmd/migrate`. Migration `2026100701` creates a `NOLOGIN`
  role `ledger_owner`, grants it to the migrator (`WITH SET TRUE, INHERIT TRUE`) and moves the
  ledger tables and trigger functions to it, keeping `SELECT, INSERT` for the migrator. This works
  for a superuser and for a `CREATEROLE` role that owns the database (a managed-Postgres admin user),
  since the new owner must be granted `CREATE` on the schema by its owner.
  Without either, the migration logs a NOTICE, leaves ownership with the migrator, and a DBA runs:
  `CREATE ROLE ledger_owner NOLOGIN; GRANT ledger_owner TO <migrator> WITH SET TRUE, INHERIT TRUE;
  GRANT USAGE, CREATE ON SCHEMA public TO ledger_owner;` then `ALTER TABLE ledger_accounts, ledger_journals, ledger_lines OWNER TO ledger_owner` (one
  statement per table) and `ALTER FUNCTION ledger_* OWNER TO ledger_owner` for the five trigger
  functions, then `GRANT SELECT, INSERT` on the tables and `USAGE, SELECT` on their sequences to the
  migrator. A role that is a member of `ledger_owner` must never be the server's role.
- The server connects as a separate application role with ordinary rights on every other table
  and only `SELECT, INSERT` plus sequence `USAGE` on the ledger. `cmd/migrate --ledger-app-role
  <role>` (or `POSTGRES_LEDGER_APP_ROLE`) applies exactly that grant set.

At boot in staging and production the server checks that its role is not a superuser, does not
own the ledger tables or their trigger functions, holds no `UPDATE`, `DELETE`, `TRUNCATE` or
`TRIGGER` on the tables, can `SELECT` and `INSERT`, and cannot `CREATE` in the database or in any
schema; any other state refuses to start. The trigger functions pin `search_path` and qualify
every reference, so `SET search_path` and temporary objects cannot shadow them.

Backups: a journal is sealed by its posting transaction id and start time. `pg_upgrade` and
physical (base) backups preserve both; a logical `pg_dump`/`pg_restore` into a fresh cluster
keeps the stamps but the new cluster reuses low transaction ids, so restore the ledger only
through `pg_upgrade` or a physical backup. Future migrations that alter a ledger table or
function must `SET ROLE ledger_owner` first. Every environment also refuses to boot
when the ledger tables, triggers or functions are missing, and validate mode additionally requires
migration `2026100701` recorded as applied.

## Observability

- **Metrics:** Prometheus exposition at `GET /metrics` (namespace `payminto_`).
  Includes HTTP request counts/latency (labeled by *route template*, not raw
  path, to bound cardinality), payments created/confirmed, webhook delivery
  outcomes, and live SSE connection count. Point a Prometheus scrape at
  `:<API_PORT>/metrics`.
- **Logs:** structured JSON via `log/slog`, request-ID enriched.
- **Errors:** `SENTRY_DSN` forwards captured errors; otherwise logged.
- **Health:** `/healthz` (DB), `/livez` (always 200), `/readyz` (DB).

## Real-time checkout (SSE)

The hosted checkout subscribes to `GET /api/v1/public/events/:reference_id`
(Server-Sent Events). The backend publishes a `payment` event whenever a
payment's state changes (e.g. deposit confirmed → `FILLED`). The broker is an
in-process `MemoryBroker` (`backend/internal/realtime`). For multi-instance
deployments, replace it with a Redis-backed `Broker` implementation — the
interface is the only integration point; no handler changes are needed.

## Local run

```bash
# Infrastructure only, with backend hot reload run on the host
cd payminto && docker compose -f docker-compose.dev.yml up -d postgres redis

# Backend
cd backend && go run ./cmd/server      # serves :API_PORT (default 8080)

# Frontend (points at NEXT_PUBLIC_API_URL)
cd ../frontend && npm run dev           # serves :3000

# Or build the complete development stack (backend/frontend/MCP included)
cd .. && docker compose up --build
```

Both Compose files are local/development-only. They intentionally use testnet,
plaintext PostgreSQL inside the private Compose network, development secrets,
and AutoMigrate. `docker-compose.yml` is not a production deployment despite
building the complete application stack. Its optional `local-tls` profile is a
developer convenience and does not make the topology production-ready.

## Production topology status

No production Compose file is provided in this phase. Operators must not turn
the development Compose file into a production deployment merely by changing
`SERVER` or exposing its ports. A production topology will require, at minimum:

- [ ] Checksummed versioned migrations and a tested legacy cutover path.
- [ ] PostgreSQL `verify-full`, least-privilege roles, encrypted backups, PITR,
      and a measured restore drill.
- [ ] Strong `JWT_SECRET` and, when custody is enabled, stable
      `VAULT_PASSPHRASE` from a secrets manager.
- [ ] `BLOCKCHAIN_NETWORK_TYPE=mainnet` with a separately migrated and
      reconciled mainnet database.
- [ ] At least 2 healthy RPC nodes per chain (`rpc_nodes.status='healthy'`).
- [ ] `CORS_ALLOWED_ORIGINS` set to the real dashboard origin.
- [ ] `SMTP_*` configured; `SENTRY_DSN` set; Prometheus scraping `/metrics`.
- [ ] Cold-wallet destinations configured and verified.
- [ ] On-chain signing/broadcast (M1) completed and exercised end-to-end.
