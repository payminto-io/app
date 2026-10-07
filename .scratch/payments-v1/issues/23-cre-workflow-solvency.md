# 23 CRE workflow: solvency attestation

Status: done
Owner: Chainlink engineer (Fable)
Blocked by: 21, 22

## Goal
`cre/workflows/solvency/` per `docs/cre/SPEC.md` section 5.1, TypeScript, scaffolded with `cre init --deployment-registry private` and the hello-world material removed. Cron trigger plus HTTP trigger for forced runs. Reads `/api/v1/cre/liabilities` (identical aggregation), EVM `balanceOf` at finalized for EVM reserve addresses, SPL balances through HTTP against two or more configured Solana RPCs (identical aggregation on mint, owner, amount, slot bucket), optional Confidential HTTP custodian read with the key name in `cre/secrets.yaml`. Reserve addresses come from workflow config, never from the gateway. ABI-encodes the solvency array, `runtime.report()`, `writeReport()` to `GatewayAttestations`; fails closed on any validation, report or write error; the `local-simulation` target returns a labelled result before any write. Targets: `local-simulation`, `staging` (Base Sepolia), `production` (Base).

## Acceptance
`cre workflow simulate cre/workflows/solvency --target local-simulation --non-interactive --trigger-index 0` runs against the compose stack and prints the labelled result; `main.test.ts` covers decimal scaling, bigint bounds and mismatch handling; against Anvil with the mock forwarder (`--broadcast` on the local target is forbidden, use the staging target with a local RPC) the gateway records a verified attestation. No secret value in the repo.

## Done (2026-10-07, branch `cre-solvency`)

Commits: `91afafe` (MockKeystoneForwarder, MockERC20, DeployCRELocalStack, forwarder tests, fixture printer), `6ed9e15` (cre project, `workflows/solvency` main.ts and main.test.ts, configs, fixture, compose secret mapping), `af21aaf` (`cre/README.md`, `scripts/cre-local-stack.sh`).
Report: `.superpowers/cre-solvency-report.md`.
Verified locally: 25 unit tests pass; `cre workflow simulate --target local-simulation` prints the labelled result; `--target staging --config ./config.anvil.json --broadcast` against a local Anvil delivers through the forwarder mock, `GatewayAttestations` emits `SolvencyAttested` and `ReportAccepted`, and the gateway (`CRE_PROVIDER=chainlink` on its own Anvil RPC) stores `status=attested, provider=chainlink`.
Needs Chainlink: Early Access, `cre workflow deploy`, Vault DON secrets, activation.
