# 24 CRE workflow: deposit finality attestation

Status: ready-for-agent
Owner: Chainlink engineer (Fable) with Solana engineer (Fable)
Blocked by: 09, 21, 22

## Goal
`cre/workflows/deposit-finality/` per `docs/cre/SPEC.md` section 5.2. Cron every `CRE_FINALITY_BATCH_INTERVAL` pulling `/api/v1/cre/pending-deposits?limit=12` with the `CRE_READ_TOKEN` from the Vault DON; HTTP trigger registered for forced runs. EVM deposits: receipt and `Transfer` log at finalized. Solana deposits: `getSignatureStatuses` and `getTransaction` (finalized, jsonParsed) against configured RPCs, identical aggregation on signature, slot, mint, amount, destination and error-free status. Emits verdict 1, 2 or 3 per deposit and writes one batch report. Stays under 15 HTTP calls per execution.

## Acceptance
Simulation with `--http-payload` fixtures for a confirmed, a missing and a mismatched deposit yields verdicts 1, 2, 3; gateway-side integration test (ticket 21's) consumes the resulting events; quota check in `main.test.ts` asserts call counts per batch size; wrong-asset deposits (USDT to a USDC address) are verdict 3, never 1.
