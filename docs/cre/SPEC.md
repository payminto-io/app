# Chainlink CRE module: specification

Status: ticket 19 deliverable, 2026-10-07. Implementation tickets 20 to 28.
Research and the reasoning for which use cases survive: `docs/cre/RESEARCH.md`.
Module contract this follows: `docs/architecture/MODULES.md`.

## 1. What the module is

`backend/internal/cre/` is an optional slot module called `attestation`.
It obtains an operator-independent signature on numbers the gateway already holds, and records those signatures so the dashboard, the API and a public page can show them.
It never computes a balance, never moves money, and never decides a settlement on its own.

Three workflows, in priority order:

| Workflow | What CRE attests | Default effect on money paths |
| --- | --- | --- |
| `solvency` | Ledger liabilities per asset against custody reserves per asset, plus the ledger checkpoint hash, at an interval | None. Informational badge and public page. |
| `deposit-finality` | That a deposit the gateway credited exists on chain at `finalized`, with the amount, mint or token, destination and slot or block | None unless the settlement policy line `require_attestation_above` is set; then settlement of that deposit's funds waits for the attestation. |
| `conversion-reference` | A consensus reference rate at the time of each executed conversion and the deviation of the executed rate in basis points | None. Audit evidence on the trade. |

Providers:

| Provider | Behaviour |
| --- | --- |
| `none` (default) | The module is wired with a no-op service. No tables are read on request paths, no routes change shape, settlement policy treats `require_attestation_above` as invalid configuration and refuses to boot if it is set. The gateway is byte-for-byte what it is without CRE. |
| `mock` | In-process. Produces attestation records with the same schema as `chainlink`, signed with a dev key, with scriptable delays and failures. Used by unit tests, the conformance suite and `docker compose --profile cre`. |
| `chainlink` | Triggers deployed CRE workflows over the authenticated HTTP trigger, reads attestations back from the consumer contract on the configured EVM chain, verifies them, stores them. |

A fourth path, `simulate`, is not a provider: it is the `mock` provider plus the real workflow code run through `cre workflow simulate` against the gateway's local API, so a deployer without Chainlink access can watch the real workflow produce the real record locally. See section 10.

## 2. Install-time choice

Configuration, one section per `MODULES.md` rule 6:

```
CRE_ENABLED=false                       # false: provider forced to none
CRE_PROVIDER=none|mock|chainlink        # default none; mock allowed only when SERVER != LIVE
CRE_CHAIN=ethereum-testnet-sepolia-base-1  # CRE chain selector name of the attestation chain
CRE_CHAIN_RPC_URL=                      # the gateway's own RPC for reading the consumer contract
CRE_CONSUMER_ADDRESS=                   # GatewayAttestations contract
CRE_FORWARDER_ADDRESS=                  # KeystoneForwarder (or mock forwarder in simulate)
CRE_WORKFLOW_OWNER=                     # EVM address that owns the deployed workflows
CRE_GATEWAY_URL=https://01.gateway.zone-a.cre.chain.link
CRE_WORKFLOW_ID_SOLVENCY=
CRE_WORKFLOW_ID_DEPOSIT_FINALITY=
CRE_WORKFLOW_ID_CONVERSION_REFERENCE=
CRE_TRIGGER_SIGNER=keyring://cre-trigger   # reference to an EVM key held by the signer service, never a raw key
CRE_SOLVENCY_INTERVAL=1h
CRE_FINALITY_BATCH_INTERVAL=60s
CRE_PUBLIC_VERIFY_ENABLED=true
```

Rules:

- `CRE_ENABLED=false` means `none` regardless of `CRE_PROVIDER`.
- `chainlink` without every address, workflow id and the signer reference refuses to boot in live and degrades to `mock` in development and test, printing which keys were missing (rule 6).
- `mock` in live refuses to boot.
- Turning the module on never changes a money path. The only money-path effect in the whole module is the settlement policy line `require_attestation_above = <minor units>` on a merchant's settlement policy (ticket 11), which an owner-role user sets explicitly in the dashboard, and which the policy validator rejects unless `CRE_PROVIDER != none`.
- Compose: a `cre` profile adds nothing to the default stack. `docker compose --profile cre up` starts the backend with `CRE_PROVIDER=mock` and a `cre-simulator` service that runs `cre workflow simulate --listen` for the three workflows against the backend's API (section 10).

Dashboard settings: `/dashboard/settings/attestations` shows provider (`Off`, `Mock`, `Chainlink CRE`), the attestation chain and consumer address with explorer links, each workflow's last run time, last status and last attestation id, and the trigger signer address.
When the provider is `none`, the page explains what the module does, who it is for (deployers that hold balances or settle at size), and that a direct-to-wallet merchant has nothing to attest.
It never shows a figure the module did not record.

## 3. Module layout

```
backend/internal/cre/
  port.go            Attester interface, Attestation and Verification value types, errors
  service.go         scheduling, batching, verification, persistence, events; provider independent
  verify.go          report and event verification (section 6)
  none/              no-op provider
  mock/              in-process provider with dev signing key and scripted failures
  chainlink/         HTTP trigger client (JWT alg ETH), consumer-contract event reader
  conformance/       suite every provider must pass
  README.md
backend/internal/modules/cre.go          func WireCRE(deps Deps) (*CREModule, error)
backend/internal/api/routes_cre.go       func RegisterCRERoutes(rg *gin.RouterGroup, m *modules.CREModule)
backend/internal/database/migrations/<date>NN_cre_attestations.up.sql
contracts/src/cre/GatewayAttestations.sol  consumer contract; tests in contracts/test/cre, deploy script in contracts/script
backend/internal/cre/abi/GatewayAttestations.json  exported ABI (forge inspect), consumed by chainlink/ and verify.go
frontend/lib/cre/abi.ts                  the same ABI as a typed const for the public verification page
cre/
  project.yaml                           targets: local-simulation, staging, production
  secrets.yaml                           secret names only
  workflows/
    solvency/            workflow.yaml, main.ts, config.*.json, main.test.ts
    deposit-finality/    workflow.yaml, main.ts, config.*.json, main.test.ts
    conversion-reference/workflow.yaml, main.ts, config.*.json, main.test.ts
  contracts/evm/src/GatewayAttestations.abi   for cre generate-bindings
  README.md
frontend/app/dashboard/settings/attestations/
frontend/app/verify/[attestation_id]/     public verification page
frontend/components/attestation-badge.tsx
```

Workflows are TypeScript: the official skill defaults to it, the SDK is current (1.23.0), `viem` works in QuickJS, and a Go workflow would tempt an agent to import gateway packages into the DON binary, which must never happen.
The workflow code talks to the gateway only over its public API; it imports nothing from `backend/`.

## 4. Port

```go
package cre

type Kind string
const (
    KindSolvency            Kind = "solvency"
    KindDepositFinality     Kind = "deposit_finality"
    KindConversionReference Kind = "conversion_reference"
)

type Status string // pending, attested, failed, stale

type Attestation struct {
    ID            string    // uuid
    Kind          Kind
    SubjectType   string    // "ledger_checkpoint" | "deposit" | "conversion"
    SubjectID     string
    PayloadHash   []byte    // keccak256 of the ABI-encoded report payload
    Payload       []byte    // ABI-encoded payload exactly as written on chain
    Chain         string    // CRE chain selector name
    TxHash        []byte
    BlockNumber   uint64
    WorkflowID    [32]byte
    WorkflowOwner [20]byte
    ReportID      [2]byte
    ObservedAt    time.Time // DON time from the report
    RecordedAt    time.Time
    Status        Status
    Provider      string    // none|mock|chainlink
}

// Attester is the provider port. It never decides anything.
type Attester interface {
    Name() string
    // Trigger asks the provider to run a workflow with a batch input. Returns a provider execution id.
    Trigger(ctx context.Context, kind Kind, input []byte) (string, error)
    // Poll returns attestations the provider has seen written since the cursor.
    Poll(ctx context.Context, kind Kind, cursor Cursor) ([]RawAttestation, Cursor, error)
    Health(ctx context.Context) Health
}
```

`service.go` owns: the liabilities snapshot endpoint input (reads `ledger` through its port only), the pending-deposit batch builder (reads `switch` attempts through its port), the conversion batch builder (reads `conversion` through its port), verification (section 6), persistence, the `cre.attestation.recorded.v1` and `cre.attestation.failed.v1` events, and the staleness sweep.
`settlement` consumes `cre.attestation.recorded.v1` through the event emitter; it never calls into `cre` (rule 9).

## 5. Workflows

Common to all three.
Language TypeScript, `@chainlink/cre-sdk`.
Every handler is deterministic in DON mode; all external reads happen through the HTTP capability with an explicit aggregation; time is `runtime.now()`; integers are `bigint` and decimal strings.
Each workflow writes to `GatewayAttestations` on one EVM chain (Base mainnet for live, Base Sepolia for test; configurable) through `runtime.report()` and `EVMClient.writeReport()`.
Each report is one batch: `abi.encode(uint8 version, uint8 kind, bytes32 gatewayId, uint64 observedAt, Item[] items)` with `version = 1`, so one contract and one verifier handle all three.
`gatewayId` is `keccak256` of the deployer's configured public base URL, so a public verifier can tell which gateway a record belongs to.
`observedAt` is DON time in seconds; the consumer rejects it more than five minutes ahead of the block and requires it to be strictly greater than the last accepted `observedAt` for the same `(gatewayId, kind)`, which is the replay and ordering guard (ticket 22, `contracts/src/cre/GatewayAttestations.sol`).
The consumer accepts a report only from the configured forwarder and only when the Keystone metadata `(workflowId, workflowName, workflowOwner)` equals the binding the contract owner set for that `kind`; the per-report `ReportAccepted` event carries that metadata, the report id, the item count and `keccak256(report)` for the verifier.

### 5.1 `solvency`

Trigger: cron, `CRE_SOLVENCY_INTERVAL` (default hourly, six-field schedule `0 0 * * * *`).
Inputs:

1. `GET {gateway}/api/v1/cre/liabilities` (unauthenticated read of a public, already-published snapshot; see section 7): `{ checkpoint_id, checkpoint_hash, assets: [{ asset, liabilities_minor, decimals }] }`. Aggregation: identical on the whole body.
2. Custody reserves per asset: for EVM assets, `balanceOf(custodyAddress)` via EVM read at `LAST_FINALIZED_BLOCK_NUMBER`; for Solana SPL assets, `getTokenAccountBalance` through HTTP against the two or more RPC URLs in config, identical aggregation on `(mint, owner, amount, slot_bucket)` where `slot_bucket` is slot rounded down to the configured window; for a custodian with an API (BitGo), Confidential HTTP with the key from the Vault DON, median aggregation on amount. The list of reserve addresses is in the workflow config, not fetched from the gateway, so a compromised gateway cannot point the attestation at someone else's wallet.
3. Nothing else.

Output, `kind = 1`, items of `(bytes32 checkpointHash, bytes32 asset, uint256 liabilities, uint256 reserves, uint8 decimals)`; several assets are packed as one array in one report (well under 50 KB).
Consensus: identical on every field.
Consumer: `GatewayAttestations.onReport` emits `SolvencyAttested(gatewayId, asset, checkpointHash, liabilities, reserves, decimals, observedAt)` per item and stores the latest per `(gatewayId, asset)`; an asset whose stored `observedAt` is not older than the report's is rejected, which also rejects a duplicate asset inside one batch.
Chains: attestation chain only; reserves may live on any chain.
Fail mode: **fail open.** A missing attestation makes the badge `stale` after two intervals; nothing else changes.

### 5.2 `deposit-finality`

Trigger: cron every `CRE_FINALITY_BATCH_INTERVAL` (default 60 s, minimum 30 s) **pulling** a batch, not per-deposit HTTP triggers (the HTTP trigger is limited to one call per 30 s).
The HTTP trigger is also registered so an operator can force a run from the dashboard.
Inputs:

1. `GET {gateway}/api/v1/cre/pending-deposits?limit=12`: deposits the gateway has credited at its own confidence and that a policy wants attested, with `{ deposit_id, chain, tx, log_index_or_signature, token, expected_amount_minor, destination }`. Aggregation: identical.
2. For each EVM deposit: the receipt and the `Transfer` log via EVM read at finalized; for each Solana deposit: `getSignatureStatuses` and `getTransaction` (jsonParsed, `finalized`) against the configured RPC URLs through HTTP, identical aggregation on `(signature, slot, mint, amount, destination, err==null)`.
   The limit of 12 keeps the execution under the 15-HTTP-call quota with headroom for the pending read.

Output, `kind = 2`, items of `(bytes32 depositId, bytes32 chainId, bytes32 txRef, bytes32 token, uint256 amount, bytes32 destination, uint64 slotOrBlock, uint8 verdict)` where verdict is 1 confirmed, 2 not found, 3 mismatch; any other verdict is rejected.
Consensus: identical.
Consumer: emits `DepositAttested(gatewayId, depositId, verdict, chainId, txRef, token, amount, destination, slotOrBlock, observedAt)` per item; nothing per deposit is stored, the event is the record.
Fail mode: **fail open by default, fail closed by policy.** With no `require_attestation_above` line nothing waits. With the line set, settlement of funds from a deposit above the threshold stays in `awaiting_attestation` until `verdict=1` arrives; `verdict=2` or `3` freezes the deposit and raises `cre.attestation.failed.v1` to the ops queue; a CRE outage longer than `3 x interval` surfaces in the dashboard and an owner-role user can release a specific deposit with a hash-chained audit entry (ticket 11's approval machinery), never by flipping the policy.

### 5.3 `conversion-reference`

Trigger: cron, default every 15 minutes.
Inputs:

1. `GET {gateway}/api/v1/cre/conversions?since=<cursor>&limit=10`: executed trades `{ conversion_id, executed_at, base, quote, executed_rate_decimal, amount_minor }`. Identical aggregation.
2. For each pair: the Chainlink Data Feed proxy on the attestation chain (`latestRoundData`, finalized), or the Data Stream report for FX pairs when configured; staleness checked against the feed heartbeat; `decimals()` read, never assumed.

Output, `kind = 3`, items of `(bytes32 conversionId, bytes32 pair, int256 referenceRate, uint8 referenceDecimals, int256 deviationBps, address feed, uint80 roundId)`.
Consensus: identical (the feed read is deterministic at a finalized block).
Consumer: emits `ConversionReferenceAttested(gatewayId, conversionId, pair, referenceRate, referenceDecimals, deviationBps, feed, roundId, observedAt)` per item.
Fail mode: **fail open.** Trades without a reference show no reference; the ledger trade is unchanged.

## 6. Trust model

What CRE attests: that a set of nodes independently observed the inputs listed above and agreed on them, at DON time, and signed the result; and that the forwarder delivered exactly that result to the consumer contract.
What CRE does not attest: that the gateway's liabilities are honestly computed (the ledger does that and the checkpoint hash lets an auditor replay it), that a reserve address belongs to the deployer (the deployer configures it and the public page shows it), or that a deposit is final beyond the chain's own definition of `finalized`.

What the gateway still decides: credit of a deposit (the chain adapter and the switch), every ledger line, settlement eligibility, caps, approvals, destinations.
An attestation is an input to policy, never a substitute for it.

How the gateway verifies an attestation before using it (`verify.go`):

1. The attestation is read from the consumer contract's events through the gateway's own `CRE_CHAIN_RPC_URL`, never from a callback. A compromised CRE account or a spoofed HTTP response cannot produce a record.
2. The log's emitting address equals `CRE_CONSUMER_ADDRESS`.
3. The contract itself accepted the report only from `CRE_FORWARDER_ADDRESS` and only for the `(workflowId, workflowOwner, workflowName)` bound to that kind by `setWorkflow`; the gateway additionally reads them from the `ReportAccepted` event and checks `workflowId` against the configured id for that kind and `owner` against `CRE_WORKFLOW_OWNER`.
4. `gatewayId` in the payload equals this deployment's `gatewayId`.
5. For `solvency`, `checkpointHash` matches a checkpoint the ledger actually published; for `deposit-finality`, `depositId` is a deposit the gateway asked about and the attested `(token, amount, destination)` equals what the gateway credited, otherwise the record is stored with `status=failed` and the mismatch is raised; for `conversion-reference`, `conversionId` exists.
6. The block holding the event is at least `CRE_VERIFY_CONFIRMATIONS` behind head (default: the chain's finality).

Only after all six does the record become `attested` and the event fire.
The `mock` provider goes through the same verifier with a mock forwarder and a local dev chain or recorded logs, so the verifier is exercised in CI.

Threats considered: compromised gateway host (cannot forge attestations because the workflow config, not the gateway, names the reserve addresses, and the signer set is the DON's); compromised RPC used by the gateway (the deposit workflow uses the DON's RPCs, and the verifier's read of the consumer contract can be pointed at a second RPC); compromised CRE account (can trigger, pause or update workflows; cannot change the consumer contract's forwarder or workflow bindings, which only the contract owner, the deployer's multisig under a two-step transfer, can set; an updated workflow gets a new workflow id, which the contract rejects until the owner rebinds it and the gateway rejects until an owner-role user accepts it in settings with an audit entry); CRE outage (section 5 fail modes per kind).

## 7. Data flow

```
gateway cron ──(nothing: cron workflows pull)──▶ CRE DON
gateway ops ──HTTP trigger, JWT alg=ETH signed by cre-trigger key──▶ CRE gateway ──▶ workflow
workflow ──HTTPS GET, unauthenticated public reads or CRE_READ_TOKEN──▶ gateway /api/v1/cre/*
workflow ──signed report──▶ KeystoneForwarder ──▶ GatewayAttestations.onReport ──▶ event
gateway poller ──eth_getLogs over own RPC──▶ verify.go ──▶ cre_attestations row ──▶ event emitter
dashboard, API, /verify/<id> ──▶ cre_attestations
```

Gateway endpoints read by workflows (`routes_cre.go`), all read-only, all returning only data the gateway already publishes or that carries no customer information:

- `GET /api/v1/cre/liabilities`: the latest ledger checkpoint totals per asset. Public when `CRE_PUBLIC_VERIFY_ENABLED=true` because the public verification page shows the same numbers; otherwise requires `CRE_READ_TOKEN` as a bearer held in the Vault DON and sent via Confidential HTTP.
- `GET /api/v1/cre/pending-deposits`: deposit ids and chain references only; requires `CRE_READ_TOKEN`.
- `GET /api/v1/cre/conversions`: conversion ids, pair, executed rate; requires `CRE_READ_TOKEN`.
- `POST /api/v1/cre/runs/{kind}` (dashboard, owner role): force a run through the HTTP trigger.
- `GET /api/v1/cre/attestations`, `GET /api/v1/cre/attestations/{id}`, `GET /api/v1/cre/status`: dashboard reads.
- `GET /api/v1/public/attestations/{id}`: the public verification read, no auth, when enabled.

Responses are snake_case under the standard envelope.
Nothing in these endpoints writes.
The gateway never accepts a workflow callback that changes state.

## 8. Secrets

- The CRE trigger-signing key is an EVM key with no funds and no custody role, held by the signer service and referenced as `keyring://cre-trigger`; the API process asks the signer to produce the JWT signature and never holds the key (CLAUDE.md: signer keys never in the API process).
- `CRE_READ_TOKEN` is generated by the gateway, stored hashed, and uploaded to the Vault DON by the deployer with `cre secrets create`; it grants read of the three `/api/v1/cre/*` inputs and nothing else.
- Custodian API keys for Confidential HTTP live only in the Vault DON under names listed in `cre/secrets.yaml`; the repository holds names, never values.
- `cre/.env` is git-ignored; simulation reads `CRE_ETH_PRIVATE_KEY` from it or uses the CLI's default simulation key, and no gateway code reads that file.
- The deploy wallet and the workflow owner are the deployer's; the repo never contains them.

## 9. Observability

Metrics (`internal/metrics`): `cre_runs_total{kind,provider,result}`, `cre_attestation_age_seconds{kind}`, `cre_pending_deposits_awaiting_attestation`, `cre_verify_failures_total{reason}`, `cre_trigger_latency_seconds`.
Logs: one structured line per run and per verification with workflow id, execution id, tx hash; never a secret, never a JWT.
Events: `cre.attestation.recorded.v1`, `cre.attestation.failed.v1`, `cre.workflow.stale.v1`.
Dashboard: the settings page shows the last ten runs per kind with CRE execution ids linking to the CRE dashboard, and the attestation chain explorer link for every record.
`cre workflow get` output for each deployed workflow is captured in the deploy runbook, not polled by the gateway.

## 10. A deployer without Chainlink access still gets a complete product

Three tiers, every one honest in the UI:

1. `none`: the product is complete; attestations do not exist; badges do not render; the public verification route returns 404 and the settings page says why.
2. `mock`: attestations exist with `provider=mock`, every badge says `Mock`, the public page says `Mock attestation, not independently signed`. This is the compose default under `--profile cre` and the CI path.
3. `simulate`: `mock` provider plus the real workflows run locally with `cre workflow simulate` (no login needed; verified today with CLI v1.37.0) against the local gateway and a local Anvil chain holding `GatewayAttestations` with the mock forwarder. Records carry `provider=mock` and `simulated=true`. This is the demo path and proves the workflow code is real without an account.
4. `chainlink`: requires Early Access and deployed workflows; records carry `provider=chainlink` and the explorer link.

Nothing in the product depends on tier 4 existing.

## 11. Open-core placement

Core (Apache 2.0): the module, `none` and `mock`, the `chainlink` provider, the three workflows, the consumer contract, the verifier, the settings page, badges and the public verification page.
The spec's enterprise list does not name attestation, and the whole point is that any deployer can run it.
`ee/` gains nothing from this ticket.
A future Solana receiver program (ticket 28) is also core.
If a managed PoR feed or an ACE provider is ever added, that is a separate `ee/` decision.

## 12. Frontend surfaces

- `/dashboard/settings/attestations` (ticket 20 for on/off and status, ticket 26 for runs and records): provider, chain, contract, signer, workflow ids with accept-new-id flow, last runs, force-run buttons for owner role, the policy line `require_attestation_above` is edited on the settlement policy page and linked from here.
- Attestation badge (`attestation-badge.tsx`): on a settlement row and detail (deposit-finality), on a conversion row (reference rate and deviation), and on the custody and treasury overview (solvency ratio with the two numbers it is made of, never a ratio alone). States: `attested` (with chain link), `pending`, `stale`, `failed`, `mock`, `simulated`; absent when the provider is `none`.
- `/verify/[attestation_id]`: public, no auth, server-rendered: kind, gateway id, observed at, chain, tx link, the decoded payload, the consumer contract and forwarder addresses, the workflow id and owner, and a short explanation of what is and is not proven. Reads only the stored row and re-checks the log exists at render time; if the re-check fails it says so instead of rendering a stale record as valid.

Copy lives in `frontend/lib/copy/attestations.ts` per the repository convention.
Design: `docs/design/DESIGN.md` applies; screenshots at 390 and 1440, light and dark.

## 13. Demo path (ticket 27)

1. `docker compose --profile cre up`: Postgres, Redis, backend with `CRE_PROVIDER=mock`, Anvil with `GatewayAttestations` deployed behind the mock forwarder, dashboard, checkout, and the `cre-simulator` service.
2. Seed a merchant on BitGo-mock custody with hold-in-asset, pay a USDC-on-Solana payment through the mock chain, see the ledger lines.
3. `cre workflow simulate cre/workflows/solvency --target local-simulation --non-interactive --trigger-index 0`: the workflow reads `/api/v1/cre/liabilities`, reads the mock reserves, writes to Anvil; the gateway poller verifies and records; the dashboard shows the badge with both numbers; `/verify/<id>` renders.
4. Set `require_attestation_above` on the merchant's settlement policy; pay above the threshold; show settlement in `awaiting_attestation`; run `deposit-finality` simulate with `--http-payload`; show the settlement proceed and the audit trail.
5. Run `conversion-reference`; show the deviation stamp on the trade.
6. Flip `CRE_ENABLED=false`, restart, show the product unchanged and the badges gone.

Each step is scripted in `scripts/demo-cre.sh` and asserted by `make smoke-cre`.

## 14. Non-goals

No pre-trade attested quote shown to a payer.
No CCIP or CCTP orchestration inside CRE (the `bridge` slot owns cross-chain delivery).
No x402 verification through CRE.
No ACE.
No Confidential Workflows (private beta); Confidential HTTP only for custodian keys.
No Solana read or log trigger until Chainlink ships them; ticket 28 covers Solana write for a mirrored attestation record.
