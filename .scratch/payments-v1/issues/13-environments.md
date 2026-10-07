# 13 Live and test environment isolation

Status: done
Owner: Principal engineer (Fable)
Blocked by: 01

## Goal
API keys, connectors, custody, chains and ledger accounts carry an environment; test and live use separate databases or schemas and separate secrets, enforced at boot; the development keystore refuses to start in live. Settlement eligibility is environment-matched (Kuberopay ADR 0033).

## Acceptance
Boot test: live mode with dev keystore exits non-zero; a test key cannot read live rows; integration test proves the separation.

## Comments

- 2026-10-07 (Principal, Fable): done on branch `environment`. Commits `89d1a53` (environment port, guard, boot policy), `8a3cb2b` (config: GATEWAY_ENVIRONMENT, test database name, DEV_KEYSTORE, slot providers, BootFacts), `170e762` (ledger: environment on accounts, guarded posts and reads, migration `2026100705_environment_isolation`), `5cd6507` (API key environment and prefix, auth refusal with `api_key_environment_mismatch`, `GET /v2/environment`, modules wiring), `950c109` (boot tests, compose as test, docs). Full report: `.superpowers/environment-report.md`. Deviations: live also requires mainnet and test refuses it (ADR 0033); new keys are `sk_test_`/`sk_live_` rather than `pm_`; the ledger uniqueness index is replaced by its environment-scoped version. Frontend indicator left for the frontend engineer.
- 2026-10-07 (Principal, Fable), fix round 1: merged `ledger` (`79e73fc`); `dd65cb1` C1 sessions bound to the environment; `1f9f02a` I3/M5/M6 database authority and stamp; `5f7124d` I1/M1/M2/M8 ledger idempotency and additive indexes; `d617f1c` I2 `cmd/migrate adopt-live`; `075b977` I4/M4/M7; `72d3b65` docs and ticket 17 (drop the four-column index later). Report: `.superpowers/environment-fix-1-report.md`. Not fixed: no `environment` column on the other money tables (one database per environment, enforced by the stamp); pre-upgrade sessions and refresh tokens are invalidated by design.
- 2026-10-07 (Principal, Fable), fix round 2: merged `main` (`42bde82`); `1e84fdf` R1 (a process stamps only an empty database, last; `adopt-live` accepts unstamped data; `adopt-test`), N1 (libpq escaping, pgx parse-back), N2, N3 (`GATEWAY_TEST_DATABASE_NAME`), N4, N5, N6; Postgres tests for the remote pre-ticket database through a non-loopback DSN host and for the boot-level refuse-then-adopt sequence. Report: `.superpowers/environment-fix-2-report.md`.

