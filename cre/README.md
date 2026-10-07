# cre: Chainlink CRE workflows for the attestation module

The workflows that give the gateway an operator-independent signature on numbers it already holds.
Design: `docs/cre/SPEC.md` (section 5 is the wire format and replay rules). Turning the module on: `docs/cre/OPERATIONS.md`.
Consumer contract: `contracts/src/cre/GatewayAttestations.sol`. Gateway module: `backend/internal/cre/`.

The workflow code talks to the gateway only over its public API and imports nothing from `backend/`.
No file in this directory holds a secret value: `secrets.yaml` maps names to environment variable names, `.env` is git-ignored.

```
cre/
  project.yaml                 targets local-simulation, staging (Base Sepolia), production (Base); RPC URLs from env
  secrets.yaml                 secret names only
  .env.example                 what cre workflow simulate reads; copy to .env
  package.json                 @chainlink/cre-sdk 1.23.0, viem, zod; bun test; typecheck
  workflows/solvency/
    main.ts                    the workflow (ticket 23)
    main.test.ts               unit tests (bun test)
    workflow.yaml              one block per target; the workflow-name is what the consumer binds
    config.local-simulation.json   mode local-simulation: returns a labelled result, never writes
    config.staging.json        mode staging: Base Sepolia (placeholders until you deploy)
    config.production.json     mode production: Base (placeholders until you deploy)
    config.anvil.json          mode staging pointed at the local Anvil stack (scripts/cre-local-stack.sh)
    fixtures/solvency-report.json  bytes printed by contracts/test/cre/SolvencyFixture.t.sol
  contracts/evm/src/GatewayAttestations.abi   for cre generate-bindings (Go workflows); TypeScript uses viem
```

## What `solvency` does

One run, cron (`schedule`, six fields, hourly by default) or the HTTP trigger for a forced run:

1. `GET {gateway.apiBaseUrl}/api/v1/cre/liabilities` with `Authorization: Bearer <CRE_READ_TOKEN_SOLVENCY>`; identical aggregation on the whole body; every `liabilities_minor` must be a uint256 integer string.
2. Reserves, from the workflow config only (never from the gateway):
   - `reserves.evm[]`: ERC-20 `decimals()` and `balanceOf(custodyAddress)` at `LAST_FINALIZED_BLOCK_NUMBER`; the on-chain decimals must equal the configured ones.
   - `reserves.solana`: for each `accounts[]` entry, `getAccountInfo(tokenAccount, jsonParsed, finalized)` against every `rpcUrls[]` entry (two or more) inside node mode; all RPCs must agree on `(mint, owner, amount, slot bucket)` with `slot bucket = floor(slot / slotWindow) * slotWindow`, the account must hold the configured mint for the configured owner, then the DON agrees on the whole list.
   - `reserves.custodian`: one Confidential HTTP `GET` per asset with `Authorization: Bearer {{.CUSTODIAN_API_KEY}}` resolved from the Vault DON (owner `secretOwner`); the endpoint answers `{ "balances": [ { "asset": "USDC", "amount": "<major-unit decimal>", "decimals": 6 } ] }` and the amount is scaled with no rounding.
3. Items: every snapshot asset needs at least one reserve source with the same decimals; the sources are summed. No source for an owed asset, a decimals mismatch, a non-integer amount, a uint256 overflow, a disagreeing RPC, a wrong mint or owner: the run fails and nothing is written.
4. `abi.encode(uint8 1, uint8 1, bytes32 gatewayId, uint64 observedAt, (bytes32 checkpointHash, bytes32 asset, uint256 liabilities, uint256 reserves, uint8 decimals)[])`, `gatewayId = keccak256(gateway.publicBaseUrl)`, `observedAt = runtime.now()` in seconds, `asset` right-padded ASCII as the gateway's `LabelKey`. `main.test.ts` checks the bytes against `ReportEncoder.sol`.
5. `local-simulation` returns `{ mode: "local-simulation", written: false, report, reportHash, items... }` here and stops: no report is signed, no EVM client is constructed. `staging` and `production` sign with `runtime.report`, `writeReport` to `attestation.consumerAddress` with `gasLimit` 2,500,000 (SPEC section 5), fail on anything but `TX_STATUS_SUCCESS` with a transaction hash, then `POST /api/v1/cre/reports` so the gateway polls its own RPC now rather than at the next interval (best effort; the gateway never trusts that body).

Fail mode on the gateway side stays open (SPEC 5.1): a run that fails records nothing and the badge goes stale after two intervals.

## Checks

```bash
cd cre
bun install
bun run typecheck
bun test            # 25 tests: decimal scaling, bigint bounds, mismatch handling, encoder vs ReportEncoder.sol, runs with SDK mocks
```

## Without Chainlink access

Everything below runs with `cre workflow simulate` (CLI v1.37.0, no login) and a local Anvil. Nothing logs in, deploys, or broadcasts to a public network.

### 1. Local simulation: the real workflow against the real gateway, no write

Prerequisites: the gateway running with `CRE_ENABLED=true` and a `CRE_READ_TOKEN_SOLVENCY`, a ledger with at least one member liability, and an Anvil holding a reserve token (step 2 deploys one).

```bash
anvil --chain-id 84532 --block-time 1 --slots-in-an-epoch 1          # chain id of Base Sepolia, which the targets name
./scripts/cre-local-stack.sh                                          # MockKeystoneForwarder, mock USDC, GatewayAttestations
cp cre/.env.example cre/.env                                          # set SOLVENCY_READ_TOKEN to the gateway's CRE_READ_TOKEN_SOLVENCY
cd cre && cre workflow simulate workflows/solvency --target local-simulation --non-interactive --trigger-index 0
```

`config.local-simulation.json` names the token and custody address the stack script deploys on a fresh Anvil (`0xe7f1...0512`, Anvil account 2); if your gateway is not on `:8090`, copy the file to `config.local-simulation.local.json` (git-ignored), change `gateway.*`, and pass `--config ./config.local-simulation.local.json` (the path is relative to the workflow directory).

Result: the labelled JSON above, with the checkpoint hash and the numbers the report would carry.

### 2. Full delivery on a local chain: the gateway records a verified attestation

`--broadcast` is forbidden on the `local-simulation` target (its handler returns before any write). Use the `staging` target with the RPC pointed at Anvil:

```bash
# cre/.env: CRE_STAGING_EVM_RPC_URL=http://127.0.0.1:8545 and CRE_ETH_PRIVATE_KEY=<an Anvil dev key, printed at start>
cd cre && cre workflow simulate workflows/solvency --target staging --config ./config.anvil.json \
  --non-interactive --trigger-index 0 --broadcast
```

The simulator signs the report with a fixed identity, workflow id `0x11..11`, owner `0xaa..aa`, workflow name `HashTruncateName(<workflow-name>)`, DON id 1, and calls `report()` on the chain's published MockKeystoneForwarder address (`0x82300bd7...` for Base Sepolia). `scripts/cre-local-stack.sh` installs `MockKeystoneForwarder` code at that address and binds the consumer to exactly that identity, so `GatewayAttestations.onReport` accepts the report and emits `SolvencyAttested` and `ReportAccepted`.

Run the gateway with the variables the script prints (`CRE_PROVIDER=chainlink`, your own Anvil RPC, the consumer, the forwarder, the simulator's workflow id and owner, `CRE_POLL_INTERVAL=5s`). It reads the logs over its own RPC, rebuilds the report, checks it hashes to the contract's `reportHash`, and stores the row: `status=attested`, `provider=chainlink`, with the transaction hash and block. `GET /api/v1/public/attestations/<id>` then renders it. Anvil's `finalized` tag lags two epochs, which is why the stack uses `--slots-in-an-epoch 1`; without it the row appears about a minute after the write.

What this proves: the workflow's bytes satisfy the audited consumer and the gateway's verifier. What it does not prove: DON consensus or Keystone signature checks, which only a deployed workflow through the real `KeystoneForwarder` exercises (`contracts/test/cre/GatewayAttestations.forwarder.t.sol` covers that delivery path with the vendored forwarder).

Solana and custodian reads are not part of the local run (there is no local validator or enclave); their code paths are unit-tested with the SDK's HTTP mocks and run unchanged in simulation against real RPCs when `reserves.solana` is configured.

## With Chainlink access

Every step here is the deployer's: nothing in this repository logs in, deploys, creates secrets or spends funds.

1. `cre login`, `cre account access` (Early Access), a linked funded wallet for the target.
2. Fill `config.staging.json` or `config.production.json`: `gateway.*` (the public base URL must equal the gateway's `CRE_PUBLIC_BASE_URL`), `reserves.*` (your custody addresses, token addresses, Solana token accounts, at least two Solana RPCs), `attestation.consumerAddress` once deployed, `authorizedKeys` with the gateway's trigger-signer address for the HTTP trigger.
3. `cre secrets create workflows/solvency --target staging` with `SOLVENCY_READ_TOKEN` (the gateway's `CRE_READ_TOKEN_SOLVENCY`) and `CUSTODIAN_API_KEY_VALUE` in the environment.
4. `cre workflow simulate workflows/solvency --target staging --non-interactive --trigger-index 0` against the public RPC: the full pipeline runs and the run fails if the consumer address is still the placeholder, which is the point.
5. `cre workflow deploy workflows/solvency --target staging`, note the workflow id and owner from `cre workflow list --target staging --output json`.
6. Deploy the consumer with `contracts/script/DeployGatewayAttestations.s.sol`: `CRE_FORWARDER_ADDRESS` = the chain's KeystoneForwarder (`docs.chain.link/cre` forwarder directory), `CRE_WORKFLOW_ID_SOLVENCY` and `CRE_WORKFLOW_OWNER` from step 5, `CRE_WORKFLOW_NAME_SOLVENCY=solvency-staging` (the `workflow-name` in `workflow.yaml` for that target). Then `cre workflow update` with the real `attestation.consumerAddress`, rebind with `setWorkflow` if the id changed, and `cre workflow activate`.
7. Gateway: `docs/cre/OPERATIONS.md` step 6 with the same addresses, ids and owner.

Deployment access, the Vault DON upload and activation are the parts of this ticket that only Chainlink can grant; everything up to and including step 4 was run here.
