# 29 NOWNodes as the RPC provider

Status: ready-for-agent
Owner: DevOps with Solana engineer
Blocked by: 09

## Goal
Owner direction 2026-10-07: use NOWNodes for RPC on Solana and the EVM chains. Verified endpoints (api key in the `api-key` header): `sol.nownodes.io` (Solana mainnet), `sol-testnet.nownodes.io` (Solana testnet; NOWNodes has no Solana devnet endpoint), `eth.nownodes.io`, `base.nownodes.io`, `matic.nownodes.io`, `eth-sepolia.nownodes.io`, `base-sepolia.nownodes.io`.

- RPC pool endpoints support per-endpoint headers; NOWNodes endpoints are configured with `NOWNODES_API_KEY` from the environment or the secrets vault, never committed or logged.
- The Solana test environment moves from devnet to testnet where NOWNodes is the provider: USDC and USDT test mints configured for testnet as data, with a clear note that devnet mints do not exist there.
- Live keeps the rule of at least two distinct RPC providers: NOWNodes plus one other.
- A local developer helper reads the key from the workspace secrets file (`../.secrets/providers.yaml`) into the process environment without printing it.

## Acceptance
Boot with NOWNodes endpoints in test; a read-only smoke test (slot, chain id, a known account balance) passes against each endpoint when `NOWNODES_API_KEY` is set and skips otherwise; no key in logs, errors or the repository.
