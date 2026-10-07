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
- `FEES_SURCHARGE_FORBIDDEN_METHODS`, `FEES_ASSET_PRECISION`, `FEES_OPERATOR_PLATFORM_ID` - fee
  rules (see Fee rules below and `backend/internal/fees/README.md`).
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

### Environments

One process serves one environment, `GATEWAY_ENVIRONMENT=test|live` (default `test`), and each
environment has its own database. The database is the authority: on first boot the server (or
`cmd/migrate up`, or `cmd/devseed`) writes a one-row `gateway_environment` stamp, and every later
boot of any of the three refuses when the stamp, or the name Postgres reports through
`current_database()`, disagrees with the process. Live refuses a database named `*_test` or equal
to `POSTGRES_TEST_DATABASE`; test accepts a non-`*_test` name only on a loopback host and only
while the stamp says test or the database is new.

#### Adopting an existing database as live

A database that served before environments existed is stamped `test` on its first upgraded boot,
and every row it holds is test money, so a live process refuses it and would see no balances.
Adoption relabels it once:

```bash
GATEWAY_ENVIRONMENT=live SERVER=production ... go run ./cmd/migrate adopt-live --confirm-adopt-live=<database name>
```

It runs only from a live-configured process (the full live boot gate applies), only while the
stamp says `test` and was never adopted, and only when the flag names the database Postgres
reports. In one transaction it calls `ledger_adopt_environment('live')`, a `SECURITY DEFINER`
function owned by `ledger_owner` that pauses the append-only triggers for exactly that relabel and
refuses once any live row exists; relabels legacy API keys (no visible prefix) as live, leaving
`sk_test_` keys as test keys; and restamps the database `live` with `adopted_from = test`.
Only the migrator role holds `EXECUTE` on the function. Run it before the first live boot; there
is no reverse.
### Fee rules

Migration `2026100702` runs `CREATE EXTENSION IF NOT EXISTS btree_gist`, which backs the
constraint that no two fee rules for one scope are active at the same instant.
`btree_gist` is a trusted extension (PostgreSQL 13 and later), so the migration role needs
`CREATE` on the database rather than superuser.
If the migration role lacks it, the migration fails and leaves nothing applied; run once as a
role that has it, then rerun `cmd/migrate`:

```sql
CREATE EXTENSION IF NOT EXISTS btree_gist;
```

Versions are closed only through `fee_rules_close_lineage`, a `SECURITY DEFINER` function the migration
gives to `ledger_owner` (and grants `ledger_owner` `SELECT, UPDATE` on `fee_rules`) when the migrator can
`SET ROLE ledger_owner`; the `fee_rules` trigger accepts an `effective_to` change only from that owner.
If the migration logs `fees: fee_rules_close_lineage stays owned by ...`, hand it over by hand once the
ledger roles exist:

```sql
GRANT SELECT, UPDATE ON fee_rules TO ledger_owner;
ALTER FUNCTION fee_rules_close_lineage(uuid, timestamptz) OWNER TO ledger_owner;
```

The application role needs `SELECT, INSERT` on `fee_rules`, `fee_snapshots` and `fee_postings` (no
`UPDATE` on `fee_rules`), `EXECUTE` on `fee_rules_close_lineage`, and `UPDATE` on `payment_requests` for the
legacy `fee_rule_id`/`fee_rule_version` columns.
Fee rule management is limited to the platform in `FEES_OPERATOR_PLATFORM_ID`; in staging and
production the admin routes answer 403 until it is set.

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
