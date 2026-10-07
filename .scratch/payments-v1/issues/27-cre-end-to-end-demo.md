# 27 CRE end-to-end demo with cre workflow simulate

Status: ready-for-agent
Owner: QA (Opus) with Chainlink engineer (Fable)
Blocked by: 23, 24, 25, 26

## Goal
`scripts/demo-cre.sh` and `make smoke-cre` per `docs/cre/SPEC.md` section 13: compose `--profile cre` with Anvil holding `GatewayAttestations` behind the mock forwarder, seeded merchant on mock custody with hold-in-asset, a mock USDC-on-Solana payment, the three simulations against the local gateway, the `require_attestation_above` gate shown waiting then proceeding, the conversion stamp, and the `CRE_ENABLED=false` restart showing the product unchanged. `cre/README.md` documents the deployer path to tier 4 (Early Access, `cre login`, `cre secrets create`, deploy, activate, accept workflow ids in settings) without any agent-run deploy.

## Acceptance
`make smoke-cre` passes from a fresh clone with Bun and the CRE CLI installed and no Chainlink login; every assertion reads a ledger line, a stored attestation or an on-chain log; the script never runs `cre workflow deploy`, `activate` or `secrets` commands.
