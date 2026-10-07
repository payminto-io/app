# 15 Compose with mocks, CI, release pipeline

Status: ready-for-agent
Owner: DevOps (Opus)
Blocked by: 05, 08

## Goal
`docker compose up` with no API key: Postgres, Redis, API, worker, dashboard, checkout, mock connector, mock custody, local Solana validator or recorded mock. CI runs unit, integration (testcontainers) and frontend checks; signed images; secrets only through env; smoke script extended for links, card and USDC.

## Acceptance
Fresh clone to running demo in one command; CI green; smoke passes.
