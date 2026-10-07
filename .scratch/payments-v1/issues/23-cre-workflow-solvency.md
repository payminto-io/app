# 23 CRE workflow: solvency attestation

Status: ready-for-agent
Owner: Chainlink engineer (Fable)
Blocked by: 21, 22

## Goal
`cre/workflows/solvency/` per `docs/cre/SPEC.md` section 5.1, TypeScript, scaffolded with `cre init --deployment-registry private` and the hello-world material removed. Cron trigger plus HTTP trigger for forced runs. Reads `/api/v1/cre/liabilities` (identical aggregation), EVM `balanceOf` at finalized for EVM reserve addresses, SPL balances through HTTP against two or more configured Solana RPCs (identical aggregation on mint, owner, amount, slot bucket), optional Confidential HTTP custodian read with the key name in `cre/secrets.yaml`. Reserve addresses come from workflow config, never from the gateway. ABI-encodes the solvency array, `runtime.report()`, `writeReport()` to `GatewayAttestations`; fails closed on any validation, report or write error; the `local-simulation` target returns a labelled result before any write. Targets: `local-simulation`, `staging` (Base Sepolia), `production` (Base).

## Acceptance
`cre workflow simulate cre/workflows/solvency --target local-simulation --non-interactive --trigger-index 0` runs against the compose stack and prints the labelled result; `main.test.ts` covers decimal scaling, bigint bounds and mismatch handling; against Anvil with the mock forwarder (`--broadcast` on the local target is forbidden, use the staging target with a local RPC) the gateway records a verified attestation. No secret value in the repo.
