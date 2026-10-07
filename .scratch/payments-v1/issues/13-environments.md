# 13 Live and test environment isolation

Status: ready-for-agent
Owner: Principal engineer (Fable)
Blocked by: 01

## Goal
API keys, connectors, custody, chains and ledger accounts carry an environment; test and live use separate databases or schemas and separate secrets, enforced at boot; the development keystore refuses to start in live. Settlement eligibility is environment-matched (Kuberopay ADR 0033).

## Acceptance
Boot test: live mode with dev keystore exits non-zero; a test key cannot read live rows; integration test proves the separation.
