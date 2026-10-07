# Ticket 23: CRE solvency attestation workflow

Branch `cre-solvency`, worktree `gateway-wt-cre-solvency`, 2026-10-07.
Status: done up to the simulation boundary. Nothing logged in, nothing deployed, no public network touched, no funds spent.

## Commits

| Hash | Content |
| --- | --- |
| `91afafe` | `contracts/src/cre/mocks/MockKeystoneForwarder.sol` (same external surface and raw-report slicing as KeystoneForwarder, no signature checks, mock-only `ReceiverReverted` event), `MockERC20.sol`, `contracts/script/DeployCRELocalStack.s.sol` (refuses a non-Anvil node via `anvil_nodeInfo`), `contracts/test/cre/MockKeystoneForwarder.t.sol` (5 tests), `contracts/test/cre/SolvencyFixture.t.sol` (prints the fixture bytes) |
| `6ed9e15` | `cre/project.yaml`, `cre/secrets.yaml` (names only), `cre/.env.example`, `cre/package.json`, `cre/workflows/solvency/{main.ts, main.test.ts, workflow.yaml, config.*.json, tsconfig.json, fixtures/solvency-report.json}`, `docker-compose.yml` simulator env mapped to the secret names |
| `af21aaf` | `cre/README.md`, `scripts/cre-local-stack.sh` |
| this commit | ticket 23 `Status: done`, this report |

## What the workflow does (`cre/workflows/solvency/main.ts`)

- Triggers: cron (`schedule`, six fields, default hourly) and the HTTP trigger (`authorizedKeys` mapped to `KEY_TYPE_ECDSA_EVM`).
- Liabilities: `GET /api/v1/cre/liabilities` with the `CRE_READ_TOKEN_SOLVENCY` bearer from `runtime.getSecret`, `consensusIdenticalAggregation` on the validated body. Always authenticated, so the module change the coordinator announced (public route no longer publishes checkpoints) needs nothing here.
- EVM reserves: `decimals()` and `balanceOf(custody)` at `LAST_FINALIZED_BLOCK_NUMBER`, one `EVMClient` per chain; the on-chain decimals must equal the config.
- Solana SPL reserves: `getAccountInfo(jsonParsed, finalized)` against every configured RPC (schema requires two or more) inside `runInNodeMode`; per node every RPC must agree on `(mint, owner, amount, slot bucket)` and the account must hold the configured mint for the configured owner; `consensusIdenticalAggregation` on the list across the DON.
- Custodian: optional Confidential HTTP read per asset, key name `CUSTODIAN_API_KEY` from `secrets.yaml`, owner from config, major-unit decimal scaled with no rounding. SDK 1.23.0's `ConfidentialHTTPClient.sendRequest(runtime, request)` is a DON-mode call with no caller-supplied aggregator, so the SPEC's "median on amount" is not expressible in this SDK version; the capability's own consensus applies. Noted in the README.
- Report: `abi.encode(1, 1, keccak256(publicBaseUrl), observedAt seconds, items[])` with `asset` right-padded like the gateway's `LabelKey`. `main.test.ts` compares the bytes and the keccak with `fixtures/solvency-report.json`, which `forge test --match-contract SolvencyFixtureTest -vv` prints from `ReportEncoder.sol`.
- `local-simulation` returns `{ mode, written: false, report, reportHash, items }` before any report is signed or an EVM client is constructed. `staging` and `production`: `runtime.report`, `writeReport` with `gasLimit` 2500000, `TX_STATUS_SUCCESS` plus a transaction hash required, then a best-effort `POST /api/v1/cre/reports { kind, tx_hash, simulated }` so the gateway polls its own RPC immediately.
- Fail closed: an owed asset with no reserve source, a decimals mismatch, a non-integer amount, a uint256 or uint64 overflow, a disagreeing Solana RPC, a wrong mint or owner, a gateway error, an empty snapshot, a reverted or hashless write.
- Config is a closed zod union validated by `Runner.newRunner({ configSchema })`; `local-simulation` forbids `attestation`, the other modes require it. zod's `.url()` cannot run in QuickJS (no `URL` constructor), so URLs are checked with a regex.

## Tests

`cd cre && bun test`: 25 pass, 0 fail (wire format and fixture match, `parseMinor` and `scaleDecimal` bounds and over-precision, item building and every mismatch, Solana cross-check, custodian parsing, config rejections, and runs through the SDK's `HttpActionsMock`, `EvmMock`/`addContractMock` and `ConsensusMock`: local-simulation never reaches `writeReport`, staging delivers exactly the encoded bytes in the raw report body, reverted and hashless writes and gateway errors fail closed).
`cd contracts && forge test --match-path 'test/cre/*'`: all suites pass including the new forwarder-mock tests.
`cd cre && bun run typecheck`: clean (the CLI also typechecks the test file before compiling).

## End to end, locally

Stack I started (all mine, left running for the post-merge re-run): Postgres container `cre-solvency-pg` on :5444, Redis container `cre-solvency-redis` on :6390, Anvil on :8555 (`--chain-id 84532 --block-time 1`), gateway `go run cmd/server/main.go` on :8097 from `backend/.env` (git-ignored) with `CRE_PROVIDER=chainlink`, `CRE_CHAIN_RPC_URL=http://127.0.0.1:8555`, consumer `0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9`, forwarder `0x82300bd7c3958625581cc2F77bC6464dcEcDF3e5`, one seeded member liability of 1250 USDC, mock USDC `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512` with 1300 USDC minted to Anvil account 2. Ports 8090 and 8545 belong to other sessions and were not touched.

1. `cre workflow simulate workflows/solvency --target local-simulation --config ./config.local-simulation.local.json --non-interactive --trigger-index 0`: compiled, read the gateway and Anvil, printed `{"mode":"local-simulation","wouldWrite":true,"written":false,...,"liabilities":"1250000000","reserves":"1300000000","decimals":6,...,"reportHash":"0x386d..."}`. No transaction.
2. `cre workflow simulate workflows/solvency --target staging --config ./config.anvil.json --non-interactive --trigger-index 0 --broadcast` against Anvil. First run (consumer still bound to a placeholder id): the simulator called `report(address,bytes,bytes,bytes[])` on `0x82300bd7...` (the published Base Sepolia mock-forwarder address, where the stack script had installed `MockKeystoneForwarder` code) with a 461-byte raw report: metadata version 1, DON id 1, config version 1, workflowId `0x11..11`, workflowName `78f36fde66` = `HashTruncateName("solvency-staging")`, owner `0xaa..aa`, report id `0x0001`, four signatures. The consumer reverted `UnexpectedWorkflow` (selector `0x3ed9025a`), the forwarder recorded `ReportProcessed(false)`, the gateway stored nothing. That is the binding doing its job.
3. Rebound the consumer to the simulator identity (`setWorkflow(1, 0x11..11, 0xaa..aa, 0x37386633366664653636)`), set the gateway's `CRE_WORKFLOW_ID_SOLVENCY` and `CRE_WORKFLOW_OWNER` to the same, restarted it, ran step 2 again: tx `0xcefb582afed8bf82acd836cba856c99081def93f205530b2ef9b06d542a5c4d2`, block 1441, gas 203,789; logs `SolvencyAttested`, `ReportAccepted` from the consumer and `ReportProcessed(true)` from the forwarder. The gateway answered 202 to the notice, and within 45 s (Anvil's `finalized` lag) stored `cre_attestations` row `e78d154f-51e8-40a1-9227-334192e68b18`: `kind=solvency status=attested provider=chainlink subject=<the checkpoint id it served> tx=0xcefb... block=1441 workflow_id=0x11..11 owner=0xaa..aa`. `GET /api/v1/public/attestations/e78d154f-...` renders it with `independently_signed: true`, `liabilities_minor 1250000000`, `reserves_minor 1300000000`, consumer and forwarder addresses.

`scripts/cre-local-stack.sh` reproduces the chain side on a fresh Anvil (verified on a scratch Anvil on :8556, then stopped) and prints the gateway variables; the simulator identity is documented in `cre/README.md`.

## Observations for the module track (not changed here)

- The `simulated` flag in `POST /api/v1/cre/reports` is only honoured by the mock path; with `CRE_PROVIDER=chainlink` the stored row says `simulated=false` even though the forwarder is a mock. SPEC section 10 says simulate-tier records carry `simulated=true`. One option: the chainlink provider could mark rows `simulated` when `CRE_FORWARDER_ADDRESS` is a known mock-forwarder address or when a `CRE_SIMULATED=true` flag is set in development.
- The gateway's `ReportAccepted` rebuild requires the `workflowName` to match; nothing needed for the solvency workflow, but `CRE_WORKFLOW_NAME_SOLVENCY` on the gateway side (announced in the fix round) must be the `workflow-name` of the deployed target (`solvency-staging` / `solvency-production`), not the folder name.
- Anvil's `finalized` tag lags 64 blocks by default; the README uses `--slots-in-an-epoch 1`.

## Needs Chainlink access (stopped at the simulation boundary)

`cre account access` (Early Access), `cre login`, a linked funded wallet, `cre secrets create` to the Vault DON, `cre workflow deploy` / `update` / `activate`, and the real workflow id and owner to bind in `GatewayAttestations.setWorkflow` and the gateway's `CRE_WORKFLOW_ID_SOLVENCY` / `CRE_WORKFLOW_OWNER`. The README lists the exact order.

## Post-merge re-run

When `cre-module` lands: `git merge cre-module`, restart the gateway from `backend/.env` (add `CRE_WORKFLOW_NAME_SOLVENCY=solvency-staging` if the merged config requires it), then `cd cre && bun test` and the two simulate commands above.
