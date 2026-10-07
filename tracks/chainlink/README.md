# Chainlink track: CRE solvency attestation for a self-hosted payment gateway

A Chainlink Runtime Environment (CRE) workflow that proves a payment gateway is solvent: it compares what the gateway's double-entry ledger owes merchants with what custody actually holds on chain, and writes a signed report on chain that anyone can verify.
Chainlink Proof of Reserve attests what an issuer holds; this also attests what the gateway owes.

## The problem

A merchant who lets a payment gateway hold funds has to trust its balance sheet.
Gateways fail exactly here: Triple-A lost $11.8M from its own treasury across seven chains in July 2026 ([TechNode Global](https://technode.global/2026/07/28/singapore-crypto-payments-firm-triple-a-says-own-digital-assets-hit-by-unauthorized-access/)).
Proof of Reserve verifies "what an issuer has, not what it owes" ([Spark research](https://www.spark.money/research/stablecoin-real-time-attestation-chainlink)).
A gateway can be fully reserved on chain and still owe merchants more than it holds; only a check of liabilities against reserves catches that.

## Market and why CRE

| Signal | Figure | Source |
| --- | --- | --- |
| Chainlink Proof of Reserve in production | 40+ feeds, 56 projects, $17B attested | [Spark](https://www.spark.money/research/stablecoin-real-time-attestation-chainlink), [Chainlink](https://blog.chain.link/largest-proof-of-reserve-provider/) |
| CRE in institutional finance | Swift with 17 banks preparing live transactions; DTCC Collateral AppChain in limited production; UBS fund workflows in pilot | [Chainlink](https://chain.link/press-releases/chainlink-is-enabling-financial-institutions-to-connect-to-swifts-blockchain-ledger), [Q2 2026 review](https://chain.link/blog/quarterly-review-q2-2026) |
| CRE availability | Live on 24 EVM mainnets, write-only on Solana (265-byte reports), deploy in Early Access | [CRE docs](https://docs.chain.link/cre/supported-networks-go) |
| Genuine stablecoin payments | ~$390B a year, $226B of it B2B | [a16z](https://a16zcrypto.com/posts/article/state-of-crypto-report-2025/) |

What institutions pay Chainlink for is continuous, machine-readable attestation that a contract or an auditor can act on.
A self-hosted gateway is exactly the party merchants cannot audit, so an independently signed solvency attestation is the trust feature a self-hoster can sell.

We ranked six CRE use cases by demand evidence (full research: `docs/cre/RESEARCH.md`):

1. **Solvency attestation** - built. Real buyers in the adjacent Proof of Reserve market, and it fills the liabilities gap.
2. **Deposit finality attestation** gating large settlements - next. Addresses the single-RPC trust gap behind gateway losses.
3. **Conversion reference rate** on executed trades - audit evidence only.
4. Cross-chain settlement via CCIP - cut; CCTP V2 already moves USDC to Solana.
5. x402 verification - cut; CRE's trigger quota cannot sit inside a 402 round trip.
6. ACE compliance - cut; private beta and EVM only.

## Architecture

```mermaid
flowchart LR
  subgraph Gateway[Self-hosted gateway]
    LED[(Double-entry ledger)] --> CP[Liabilities checkpoint<br/>one consistent snapshot + hash]
    CP --> API[/GET /api/v1/cre/liabilities<br/>per-workflow bearer token/]
    V[CRE verifier<br/>reads its own RPC] --> AT[(cre_attestations)]
    AT --> PUB[/Public verification page/]
  end
  subgraph CRE[Chainlink CRE DON]
    WF[Solvency workflow<br/>cron + HTTP trigger]
  end
  API --> WF
  WF -->|balanceOf at finalized| EVM[EVM reserve addresses]
  WF -->|SPL balances, 2+ RPCs, consensus| SOL[Solana reserve accounts]
  WF -->|signed report| FWD[Keystone forwarder]
  FWD --> GA[GatewayAttestations contract]
  GA -->|SolvencyAttested, ReportAccepted events| V
```

### End to end

```mermaid
sequenceDiagram
  participant WF as CRE workflow
  participant GW as Gateway
  participant CH as Chains
  participant F as Keystone forwarder
  participant C as GatewayAttestations
  WF->>GW: read liabilities checkpoint (per asset totals + checkpoint hash)
  WF->>CH: read reserves at finalized (EVM balanceOf, SPL balances)
  WF->>WF: DON consensus, encode report (version, kind, gatewayId, observedAt, items)
  WF->>F: writeReport
  F->>C: onReport (forwarder only, bound workflow id, owner, name)
  C->>C: replay check by report hash, store newest per asset
  C-->>GW: SolvencyAttested + ReportAccepted events
  GW->>CH: verifier reads the transaction and logs from its own RPC
  GW->>GW: compare with the checkpoint it served, store as attested or mismatch
```

## What we built

| Piece | Path | State |
| --- | --- | --- |
| Consumer contract `GatewayAttestations` | `contracts/src/cre/` | Audited and re-audited; 81 Foundry tests including fuzz, invariants and delivery through Chainlink's real KeystoneForwarder |
| Workflow name binding | `contracts/src/cre/WorkflowName.sol` | Keystone HashTruncateName, pinned to the documented vector |
| Solvency workflow (TypeScript) | `cre/workflows/solvency/` | 25 tests; encoding matches the contract byte for byte; simulated and broadcast to a local chain |
| Gateway CRE module | `backend/internal/cre/` | Providers `none` (default), `mock`, `chainlink`; verifier reads its own RPC; lossless poller; consistent checkpoints |
| Install-time option | `CRE_ENABLED=false` by default; compose profile `cre` | With CRE off, routes and workers are identical (tested) |
| Dashboard settings | `frontend/app/dashboard/settings/attestations` | Shows on or off, provider, health, last attestation |

Contract rules: only the configured forwarder may call `onReport`; each report kind is bound to a workflow id, owner and name; a report is accepted once (seen-set on its hash) in any order; a solvency item is stored only if newer than the latest for its asset, otherwise it is recorded as ignored; no ETH or token custody; two-step ownership; no upgradeability.

## Proven end to end (local chain)

The workflow ran against a local Anvil chain with the Keystone mock forwarder:

- Simulation printed liabilities 1,250 USDC against reserves 1,300 USDC.
- Broadcast: the forwarder delivered the report, the contract emitted `SolvencyAttested` and `ReportAccepted`.
- The gateway verified the transaction from its own RPC and stored the attestation as `attested` with provider `chainlink`, served at `/api/v1/public/attestations/<id>`.

Reproduce: `cre/README.md` ("Without Chainlink access") and `DEMO.md`.

```bash
cd cre
~/.cre/bin/cre workflow simulate workflows/solvency --target local-simulation --non-interactive --trigger-index 0
```

## Trust model

- CRE attests the comparison; the gateway still decides nothing on CRE's word alone. Every report is re-verified from the gateway's own RPC against the on-chain event before it is shown.
- Reserve addresses come from workflow configuration, never from the gateway, so a compromised gateway cannot point the workflow at someone else's wallet.
- A figure that does not match what the gateway served is stored as `mismatch` with an anomaly, never shown as attested.
- Simulated reports are labelled, never shown as production, and refused in live.
- With CRE off, the gateway behaves exactly as without the module.

## Going to production

Needs Chainlink Early Access: `cre login`, Vault DON secrets, `cre workflow deploy`, and binding the real workflow id and owner in `setWorkflow` and the gateway's environment. The order is in `cre/README.md`.

## Next

- Deposit finality attestation gating settlements above a merchant-set threshold (ticket 24), the settlement gate (21b).
- Attestation badges on settlements and a public verification page in the dashboard (ticket 26).
- A Solana receiver program for 265-byte reports (ticket 28).
