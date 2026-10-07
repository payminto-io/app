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

