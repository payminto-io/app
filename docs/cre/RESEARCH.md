# Chainlink CRE for the gateway: research and market fit

Status: ticket 19 deliverable, written 2026-10-07.
Scope: what CRE is today, who pays for it in payments and stablecoin flows, what competes with it per use case, and which gateway use cases survive contact with that evidence.
The module spec that follows from this is `docs/cre/SPEC.md`.

The short version.
CRE is real, live on 24 EVM mainnets, write-only on Solana, deploy access is still gated, and there is no published price.
Its buyers in payments are banks, market infrastructures and stablecoin issuers buying orchestration and reserve attestation; no merchant gateway is a documented buyer.
Two gateway use cases map to things people already pay Chainlink for and are feasible for a self-hoster: a proof-of-solvency attestation (ledger liabilities against custody balances, anchored on-chain) and an independent deposit-finality attestation that a settlement policy can require above a threshold.
One survives in reduced form: stamping a consensus reference rate on executed conversions as audit evidence, not a pre-trade quote.
Three are cut: CCIP settlement orchestration (CCTP does the Solana job cheaper and the bridge slot already owns it), x402 verification (facilitators verify in 75 ms, CRE's HTTP trigger is rate-limited to one call per 30 s) and ACE compliance (private beta, EVM-only, BUSL, and compliance belongs to the deployer's hooks).
Everything is optional, off by default, and the gateway stays complete without a Chainlink account.

## 1. What CRE is as of October 2026

### Product shape

The Chainlink Runtime Environment is an orchestration layer: a workflow is a WASM binary (TypeScript via Bun and Javy/QuickJS, or Go) that runs on every node of a Decentralized Oracle Network, with capability results aggregated by BFT consensus (typically 2f+1 of 3f+1) before anything leaves the DON ([consensus concepts](https://docs.chain.link/cre/concepts/consensus-computing), [Chainlink CRE skill, concepts.md](../../../references/chainlink-agent-skills/chainlink-cre-skill/references/concepts.md)).
It was announced 2025-11-04 as live with deployment in Early Access ([CoinDesk](https://www.coindesk.com/web3/2025/11/04/chainlink-introduces-cre-to-fast-track-institutional-tokenization), [PR Newswire](https://www.prnewswire.com/news-releases/chainlink-runtime-environment-goes-live-unlocking-institutional-tokenization-at-scale-302604067.html)).
Chainlink positions it as the successor to Functions and Automation; the official agent skill tells agents to "recommend CRE instead" for either.

### Triggers

Three trigger types exist: cron (six-field schedule, minimum 30 s interval), HTTP (JSON-RPC `workflows.execute` through a Chainlink gateway, request signed by an authorized EVM key) and EVM log (contract events at a chosen confidence level, `finalized` recommended for money) ([triggers reference](https://docs.chain.link/cre/guides/workflow/using-triggers/cron-trigger-ts), [service quotas](https://docs.chain.link/cre/service-quotas)).
Up to ten triggers per workflow.

The HTTP trigger is the one a gateway would call.
The production endpoint is `https://01.gateway.zone-a.cre.chain.link`; the body is `{"jsonrpc":"2.0","method":"workflows.execute","params":{"input":{...},"workflow":{"workflowID":"<64 hex>"}}}`; the request carries a JWT with header `{"alg":"ETH"}` and claims `digest` (sha256 of the sorted body), `iss` (the caller's EVM address), `iat`, `exp` (at most five minutes) and `jti`, signed with the caller's private key under the Ethereum signed-message prefix; the response is `{"status":"ACCEPTED","workflow_execution_id":...}` ([triggering deployed workflows](https://docs.chain.link/cre/guides/workflow/using-triggers/http-trigger/triggering-deployed-workflows)).
The quota is one HTTP trigger request per 30 seconds per workflow, burst one.
That number shapes every design below: a workflow cannot be triggered per payment at any realistic volume; it must be triggered per batch, or pull batches on a cron.

### Capabilities

HTTP (per-node fetch, then median, identical, common-prefix, frequency-list or per-field aggregation; 15 requests per execution, 250 KB response), Confidential HTTP (enclave execution with Vault DON secret injection, in production since 2026-02-25 per the [dev changelog](https://dev.chain.link/changelog/cre-supports-confidential-http-capability-for-production-workflows)), EVM read (finalized by default, 15 calls per execution, 100-block query window), EVM write (signed report delivered through the KeystoneForwarder to a consumer contract's `onReport(bytes metadata, bytes report)`, 50 KB payload, 10 destination chains per workflow) and Solana write ([capabilities](https://docs.chain.link/cre/capabilities), [quotas](https://docs.chain.link/cre/service-quotas)).
Confidential Workflows (whole handler in an AWS Nitro enclave, `us-west-2` only) are private beta for deployment; simulation works ([confidential workflows](https://docs.chain.link/cre/concepts/confidential-workflows)).
Execution timeout is five minutes, memory 100 MB, 50 concurrent executions per workflow, 200 per owner.

### Consensus and on-chain write

A consumer contract trusts only the chain's KeystoneForwarder address, decodes the exact ABI the workflow encoded, and may additionally pin the expected workflow owner through `ReceiverTemplate.setExpectedAuthor` ([building consumer contracts](https://docs.chain.link/cre/guides/workflow/using-evm-client/onchain-write/building-consumer-contracts)).
The 64-byte metadata packs workflow id, a 10-byte workflow name hash, the owner address and a report id.
Simulation writes go through a different `MockKeystoneForwarder` address per chain; the skill's chain-selectors table lists both sets for testnets ([chain-selectors.md](../../../references/chainlink-agent-skills/chainlink-cre-skill/references/chain-selectors.md)).
Reports are ECDSA over keccak256; that is also the signing scheme for Solana.

### Chains, including Solana

Supported networks on 2026-09-16: 24 EVM mainnets (Ethereum, Base, Arbitrum One, OP Mainnet, Polygon, Avalanche, BNB Chain, Linea, Scroll, ZKSync Era, Sonic, Mantle, Gnosis, Celo, Ink, World Chain, XLayer, Hyperliquid, MegaETH, Monad, Plasma, Pharos, Jovay, ADI) and 33 testnets, all with read, write and log trigger ([supported networks](https://docs.chain.link/cre/supported-networks-go)).
Solana Mainnet and Devnet are **write only**: "Read and Log Trigger capabilities are in development" (same page; chain selectors `124615329519749607` and `16423721717087811551`).
Solana write landed in CLI v1.24.0: the workflow Borsh-encodes a payload, signs through `runtime.report()`, and the Keystone Forwarder program CPIs the receiver's `on_report` instruction; bindings come from an Anchor IDL via `cre generate-bindings solana`; the default compute budget is 290,000 CU ([Solana write](https://docs.chain.link/cre/capabilities/solana-write), [release notes](https://docs.chain.link/cre/release-notes)).
The simulator printed the production limit today: `solana_report=265b`, so a Solana attestation payload is a hash plus a few integers, not a document.
Consequence for this Solana-first gateway: CRE can attest *to* Solana but cannot yet observe Solana itself; any Solana fact a workflow needs comes through the HTTP capability against several RPC providers with identical aggregation.

### SDK, CLI, lifecycle

TypeScript SDK `@chainlink/cre-sdk` (installed 1.23.0 today), Go SDK `github.com/smartcontractkit/cre-sdk-go` (v1.21.0 on 2026-09-18), CLI v1.37.0 installed locally ([release notes](https://docs.chain.link/cre/release-notes)).
Lifecycle: `cre init` (built-in templates, `--deployment-registry private` for a local scaffold) → `cre workflow simulate` (local, no DON, no gas unless `--broadcast`) → `cre workflow deploy` (compiles WASM, registers, starts paused) → `activate` / `pause` / `update` / `delete`; secrets go to the Vault DON through `cre secrets create` ([operations.md](../../../references/chainlink-agent-skills/chainlink-cre-skill/references/operations.md)).
Verified today in this worktree: `cre init --non-interactive --deployment-registry private --template hello-world-ts` scaffolds `project.yaml`, `<workflow>/workflow.yaml`, `main.ts`, `config.staging.json`, `config.production.json`, `secrets.yaml`; `bun install` then `cre workflow simulate por-attest --target staging-settings --non-interactive --trigger-index 0` compiled and ran without any login, printing the production limits and the result (`.superpowers/cre-scratch/demo`).
That is the fact that makes a mock-free local demo possible for a deployer with no Chainlink account.

### Access and pricing

Deployment requires Early Access approval, requested with `cre account access` or at `app.chain.link/cre/request-access`; accounts need 2FA; an organization may link at most two keys (quotas page says one per public registry); private registry allows 3 workflows per organization, public registry 3 per linked key ([account](https://docs.chain.link/cre/account), [quotas](https://docs.chain.link/cre/service-quotas)).
The quotas page says limits are "actively tuned ahead of general availability", so CRE is pre-GA as of 2026-10-01 (CLI v1.36.0 release).
**No price is published anywhere.** The docs, release notes and account pages carry no billing section; searches for CRE execution fees return only the legacy Functions, Automation and VRF billing pages.
The validation report's note that Chainlink BUILD is "being wound into case-by-case deals" is consistent with this: expect a negotiated contract, not a rate card.
For a self-hosted product this means CRE can only ever be optional: nobody can put an unpriced, approval-gated dependency on the default path.

## 2. Who uses CRE and Chainlink oracles in payments and stablecoin flows

| Who | What | Status | Source |
| --- | --- | --- | --- |
| Swift, 17 banks (ANZ, BNP Paribas, BNY, Citi, DBS, HSBC, MUFG, Standard Chartered, UBS, Wells Fargo and others) | CRE connects bank systems and key-signing infrastructure to Swift's blockchain ledger for 24/7 tokenized-deposit payments; banks keep their keys | "Preparing to pilot initial live transactions", 2026-09-28 | [Chainlink press release](https://chain.link/press-releases/chainlink-is-enabling-financial-institutions-to-connect-to-swifts-blockchain-ledger), [FintechSpecs](https://fintechspecs.com/blog/chainlink-swift-blockchain-ledger-tokenized-deposits-2026/) |
| UBS, Swift | Tokenized fund subscription and redemption orchestrated by CRE from Swift ISO 20022 messages | Pilot, 2025-10 | [PR Newswire](https://www.prnewswire.com/news-releases/chainlink-advances-tokenized-fund-workflows-with-swift-messaging-in-collaboration-with-ubs-302570072.html) |
| DTCC | Collateral AppChain uses CRE for pricing, valuations, margin, collateral eligibility | Limited production trades from 2026-07, commercial rollout 2026-10 | [Q2 2026 review](https://chain.link/blog/quarterly-review-q2-2026) |
| Kinexys by J.P. Morgan, Ondo | Cross-chain DvP: payment leg on Kinexys Digital Payments, asset leg on Ondo Chain, CRE orchestrating | Testnet transaction, 2025-05-14 | [Chainlink blog](https://chain.link/blog/cre-dvp-kinexys-jp-morgan-ondo-finance) |
| ANZ, e-HKD | PvP settlement of a CBDC leg against ANZ's stablecoin over CCIP | Pilot (Hong Kong e-HKD programme) | search summary, [Chainlink stablecoin issuers](https://chain.link/blog/chainlink-for-stablecoin-issuers) |
| TUSD, KRWQ, Wenia USDW and COPW, Matrixdock STBT, 21BTC, Ethena USDe | Proof of Reserve feeds, several with Secure Mint (mint reverts if supply would exceed attested reserves) | Production; 40+ feeds, 56 projects, $17B attested | [Spark research](https://www.spark.money/research/stablecoin-real-time-attestation-chainlink), [Chainlink PoR](https://blog.chain.link/largest-proof-of-reserve-provider/), [Cryptonomist on KRWQ](https://en.cryptonomist.ch/2026/06/16/krwq-stablecoin-adoption-chainlink-reserve/) |
| Coinbase x402 | "First AI payments partner for CRE": agents pay per call in USDC on Base to trigger CRE workflows | Live integration, developer-facing | [Coinbase launch page](https://www.coinbase.com/developer-platform/discover/launches/chainlink-cre-x402), [CRE template](https://docs.chain.link/cre-templates/x402-price-alerts) |
| CRE Connect, ACE | Tenant-scoped verifiable events and gas-less operations; on-chain compliance policies | Private beta on mainnet and testnet, Q2 2026 | [Q2 2026 review](https://chain.link/blog/quarterly-review-q2-2026) |

What they pay for, read across these rows: orchestration between an institution's existing system and a ledger it does not control, with the institution keeping its keys (Swift, UBS, Kinexys); continuous, machine-readable reserve attestation that a contract can act on (every stablecoin row); and delivery of a signed fact to a chain (DTCC pricing, PoR).
What is announced versus in production: PoR and Data Feeds are production with named customers and dollar figures; DTCC is the only CRE deployment described as production; Swift and Kinexys are pilots; CRE Connect and ACE are private beta; Confidential Workflows are private beta; every CRE payments example on Chainlink's own "5 ways to build with CRE" page (stablecoin issuance, asset servicing, prediction settlement, x402 agents, custom PoR feed) is a demo repository with no named customer ([5 ways to build](https://chain.link/blog/5-ways-to-build-with-cre)).

No payment processor or merchant gateway appears as a CRE or Data Feeds customer in any source found.
BitPay, Coinbase Commerce and the gateways in the 2026 round-ups lock a rate for 10 to 15 minutes from their own exchange partners and sell on confirmation; none cites an oracle ([crypto.news round-up](https://crypto.news/5-leading-crypto-payment-gateways/)).
The hackathon record is unchanged from the validation report: HTTPayer and BLINK (CCIP plus Data Feeds gateways, Chromion 2025) are dark, and the 2026 CRE hackathon's one payments-shaped winner, `x402-chainlink`, is a developer toolkit, not a gateway ([hackathon winners](https://chain.link/hackathon/winners)).

## 3. Competing approaches per use case

| Need | CRE | Alternatives | Where the alternative wins |
| --- | --- | --- | --- |
| Attested reserves and solvency | Workflow reads custody balances (EVM read, HTTP for Solana and custodians), reads the gateway's liabilities, writes a signed attestation on-chain; the PoR by CRE template exists (`bring-your-own-data-ts`) | Chainlink PoR managed feeds ($1k to $50k per month for a sponsored feed per [The Signal](https://thesignal.directory/intelligence/blockchain-oracles-chainlink-pyth-redstone-compared-2026)); CPA attestations from The Network Firm or Hacken (exchanges: Kraken, OKX, Bybit, MEXC); Merkle-tree self-publishing; a plain backend signing its own numbers | A plain backend is free but is the party being audited; auditors are periodic; a managed PoR feed is for issuers, not a self-hosted gateway that has to work for every deployer. CRE's edge is that a deployer can run the same workflow and the consumer contract does not care who the deployer is. |
| Independent deposit finality | EVM log trigger at `finalized`, or HTTP consensus against several Solana RPCs; signed batch report | Gateway's own multi-RPC quorum in Go (two or three providers, agree on signature status and slot); CRE Connect watchers (private beta, EVM); Helius or Triton webhooks | A Go multi-RPC quorum is cheaper and has no 30 s limit; CRE adds an operator-independent signer set, which is what "second source of truth against a compromised RPC or host" actually requires. Use CRE only above a value threshold. |
| Attested FX and crypto rates | Data Feeds read inside a workflow (EUR/USD stream on all networks since 2026-09-16), median across sources, report written with the conversion id | Pyth pull (about $0.01 per update, deepest Solana integration), RedStone ($0.001 to $0.005 per data point), the executed trade itself (Jupiter route) | The executed rate is the truth and is already a ledger line (ticket 10). An attested reference rate is audit evidence, not a quote. Pyth is cheaper if all you want is a number on Solana. |
| Cross-chain settlement into Solana | CRE orchestrating CCIP sends | Circle CCTP V2 (Solana since 2025-10, Fast Transfer 8 to 20 s, 14 chains, burn-and-mint, trust narrowed to Circle); CCIP v1.6 Solana lanes (Arbitrum, Base, BNB, Ethereum, OP, Sonic) but no documented USDC Solana lane | For USDC, CCTP is native, cheaper and already the obvious first `bridge` provider. CRE adds a DON in front of a bridge call the gateway can make itself. |
| Agent and x402 verification | x402 agents trigger CRE workflows (Coinbase integration) | x402 facilitators `/verify` 75 ms and `/settle` 900 ms on Solana ([Solana Compass](https://solanacompass.com/news/coinbase-upgrades-x402-facilitator-on-solana-upto-scheme-live-verify-latency-cut-66)); self-verification from pre/post balances via RPC ([solana-x402](https://github.com/SaylorInnovations/solana-x402)) | The direction is inverted: CRE is a thing agents pay *for*, not a verifier of agent payments. A 30 s trigger quota cannot sit in a 402 round trip. |
| Compliance checks | ACE PolicyEngine, Evaluation API (TRM wallet risk permits) | Deployer's own KYC and fraud providers behind the `kyc` and `fraud` slots; TRM, Chainalysis, Elliptic APIs directly | ACE is private beta, EVM-only, BUSL-licensed, and contracts deployed outside the platform are invisible to it. The spec keeps compliance with the deployer. |

## 4. Candidate gateway use cases, ranked

Ranking criteria, in order: demand evidence (someone pays for this shape today), feasibility for a self-hoster (works with mock and simulate, no approval needed to be complete), cost (CRE quotas and on-chain writes), and differentiation against the program spec's five claims (one ledger, settlement choice, custody slot, cross-rail routing, self-hostable with no key).

### 1. Proof of solvency: ledger liabilities against custody balances, attested on-chain

Demand: **real product with buyers**, in the adjacent market.
Reserve attestation is the one oracle product stablecoin issuers and custodians pay for at scale (40+ feeds, 56 projects, $17B), and the Spark analysis names its structural gap: PoR "verifies what an issuer has, not what it owes".
A gateway that holds balances (BitGo custody, hold-in-asset merchants, the hosted tier, platform mode with sub-merchants) has exactly the data PoR lacks: a double-entry ledger of liabilities per asset.
Attesting both sides, liabilities and reserves, with the ledger checkpoint hash, is a solvency statement that neither a PoR feed nor a CPA letter gives.
Who buys: the deployer that holds customer funds (hosted operator, platform), to show merchants and auditors; a merchant on direct-to-wallet custody has nothing to attest and the feature must say so rather than invent a ratio.
Feasibility: high.
Cron trigger (hourly or daily), HTTP to the gateway's liabilities endpoint (identical aggregation on a hash), EVM reads of custody balances at `finalized`, HTTP to several Solana RPCs for SPL balances (identical aggregation), optionally Confidential HTTP to a custodian API with the API key in the Vault DON, then one EVM write of `(asset, liabilities, reserves, ledgerCheckpointHash, observedAt)` to a consumer contract.
Everything simulates locally; the mock provider produces the same record shape.
Cost: one write per asset per interval on Base or Arbitrum, cents per day; CRE execution cost unknown (unpriced).
Differentiation: directly extends claim 1 (one ledger across rails) and claim 3 (custody as a slot) into something a merchant can verify without trusting the deployer.
Verdict: **build first.**

### 2. Independent deposit finality attestation gating high-value settlement

Demand: **thin as a named ask, strong as a loss pattern.**
Nobody in the evidence asked a gateway for an oracle-confirmed deposit, but the failure the feature guards against is the one that kills payment companies: key and infrastructure compromise (Triple-A, $11.8M across seven chains in 2026-07) and flaky chain connectivity (BLINK's WebSocket instability on Base).
CRE Connect's "verifiable events" product is this exact shape sold to enterprises, which says Chainlink sees demand for operator-independent event confirmation even if the buyers are banks.
Feasibility: medium.
EVM deposits: a log trigger on the gateway's deposit contracts at `finalized` confidence, batching recent events into one signed report.
Solana deposits: no read or log trigger yet, so a cron (every 30 to 60 s) pulls the gateway's pending list over HTTP, each node queries two or more Solana RPC providers (`getSignatureStatuses` at `finalized`, `getTransaction` for the SPL transfer amount and mint), identical aggregation on `(signature, slot, mint, amount, destination)`, then one write of the batch.
The gateway reads the attestation event from the consumer contract through its own RPC and marks the deposit attested.
The 1-per-30 s HTTP trigger quota and the 15-HTTP-calls-per-execution cap bound throughput to roughly 15 to 30 Solana deposits per execution, which is fine for a threshold-gated feature and wrong for every deposit.
Cost: one write per batch; the on-chain write is the only hard cost.
Differentiation: supports claim 2 (settlement choice enforced by policy in core): `require attestation above X` becomes a settlement policy line, and the "every number is a receipt" rule gains a second receipt.
Verdict: **build second**, as a policy-gated option that defaults off and fails open unless the policy says otherwise.

### 3. Attested reference rate stamped on executed conversions

Demand: **thin.**
The validation evidence is unambiguous that survivors convert at receipt and that no shutdown cited FX loss; ticket 10 records the executed rate as the ledger trade.
What remains is an audit question, "was this execution within tolerance of a reference", which a merchant's finance team or an auditor may ask once volumes matter, and which the locked-rate quote in phase F will need as its reference anyway.
Feasibility: high and cheap: a cron workflow pulls the day's conversion ids and executed rates from the gateway, reads the matching Data Feed or Data Stream (EUR/USD exists on every network; USDC/USD and SOL/USD feeds exist), computes the deviation in basis points, and writes a batch report.
Cost: trivial, one write per batch.
Differentiation: weak; Pyth gives a cheaper number, and the honest framing is "evidence", not "a verified quote".
Verdict: **survives in reduced form**, lowest priority; never a pre-trade quote, never a price the checkout shows.

### 4. Cross-chain settlement orchestration into Solana via CCIP or CCTP

Demand: real at Swift, Kinexys and ANZ scale; absent at merchant scale, and the two gateways that built it died.
CCTP V2 reaches Solana natively with 8 to 20 s Fast Transfer and Circle as the only trusted party; CCIP's documented USDC lanes do not include Solana.
A CRE workflow in front of a bridge call adds a DON, a quota and an approval to something the `bridge` slot module (already in `MODULES.md`) does in Go with a CCTP provider first.
Verdict: **cut.** Revisit only when a merchant with cross-chain settlement volume asks for an operator-independent executor, and then as a `bridge` provider, not a CRE workflow.

### 5. Agent payment and x402 verification

Demand: x402 is the only agent rail with traffic and is CRE's "first AI payments partner", but the integration is agents paying to run workflows.
Verification of an x402 payment is a 75 ms facilitator call or a local balance diff; CRE's trigger quota is one call per 30 s.
Verdict: **cut.** The gateway's x402 receiver (spec claim 5) stays a plain Go rail; a CRE workflow could itself be an x402-paid resource later, which is a marketing demo, not a gateway feature.

### 6. Compliance checks via ACE

Demand: ACE has a private-beta Evaluation API for TRM wallet-risk permits and on-chain PolicyEngine contracts under BUSL; both are EVM-only and platform-provisioned.
The program spec deliberately leaves compliance with the deployer through named hooks.
Verdict: **cut** as a CRE workflow.
If a deployer wants ACE, it is a future `kyc` or `fraud` provider under `ee/`, called from Go, with nothing to do with this module.

### 7. Other candidates found and set aside

Ledger checkpoint anchoring on its own (daily hash of the journal on-chain) is folded into use case 1, since the attestation record carries the checkpoint hash.
Confidential custodian reads (API keys in the Vault DON, reads inside an enclave) are an implementation detail of use case 1 for BitGo-style custody, not a use case.
CRE Connect gas-less operations and Smart Accounts are private beta, EVM-only, and would replace the custody slot's signer boundary rather than add to it; out of scope.
Prediction-market style AI settlement, the most common CRE demo, has no gateway analogue.

## 5. Constraints that shape the spec

- HTTP trigger: one request per 30 s per workflow, burst one; design every gateway-to-CRE call as a batch, or let the workflow pull on a cron.
- Cron: 30 s minimum.
- Per execution: 15 HTTP calls, 15 EVM reads, 50 consensus calls, 5 min timeout, 25 KB per observation, 100 KB result.
- EVM write: 50 KB report, 10 destination chains; Solana write: 265 bytes per report (printed by the simulator today), write only, no Solana read or log trigger.
- Deployment and secrets need Early Access; simulation does not need login (verified).
- Private registry: 3 workflows per organization; this caps the module at three deployed workflows per deployer, which is exactly what is specified.
- No published price; the module must be complete and demoable with `mock` and with local simulation, and must never be a dependency of a money path unless a policy line says so.
- Secrets: never in config or repo; Vault DON for deployed workflows, `.env` consumed by the CLI for simulation; the gateway's trigger-signing key is an EVM key with no funds and no custody role.

## 6. Decision

Build the module as specified in `docs/cre/SPEC.md`: optional slot module, providers `none` (default), `mock` and `chainlink`, three workflows in priority order (solvency attestation, deposit finality attestation, conversion reference stamp), one EVM consumer contract, an optional Solana receiver later, a verifier in Go that trusts the chain rather than a callback, dashboard settings and badges, and a public verification page that says "unattested" honestly when CRE is off.
State the demand plainly in every surface: this is for deployers who hold funds or settle at size and want a second, operator-independent signature on their numbers; a direct-to-wallet merchant gains nothing and the UI must not pretend otherwise.

## Sources

- https://docs.chain.link/cre/service-quotas
- https://docs.chain.link/cre/supported-networks-go
- https://docs.chain.link/cre/capabilities
- https://docs.chain.link/cre/capabilities/solana-write
- https://docs.chain.link/cre/release-notes
- https://docs.chain.link/cre/account
- https://docs.chain.link/cre/guides/workflow/using-triggers/http-trigger/triggering-deployed-workflows
- https://docs.chain.link/cre/guides/workflow/using-evm-client/onchain-write/building-consumer-contracts
- https://docs.chain.link/cre/concepts/confidential-workflows
- https://docs.chain.link/cre-templates/bring-your-own-data
- https://dev.chain.link/changelog/cre-supports-confidential-http-capability-for-production-workflows
- https://chain.link/blog/quarterly-review-q2-2026
- https://chain.link/blog/5-ways-to-build-with-cre
- https://chain.link/press-releases/chainlink-is-enabling-financial-institutions-to-connect-to-swifts-blockchain-ledger
- https://chain.link/blog/cre-dvp-kinexys-jp-morgan-ondo-finance
- https://chain.link/hackathon/winners
- https://www.coinbase.com/developer-platform/discover/launches/chainlink-cre-x402
- https://www.spark.money/research/stablecoin-real-time-attestation-chainlink
- https://blog.chain.link/largest-proof-of-reserve-provider/
- https://thesignal.directory/intelligence/blockchain-oracles-chainlink-pyth-redstone-compared-2026
- https://solana.com/docs/tools/x402-facilitator
- https://solanacompass.com/news/coinbase-upgrades-x402-facilitator-on-solana-upto-scheme-live-verify-latency-cut-66
- https://blog.chain.link/ccip-v1-6-is-now-live/
- https://developers.circle.com/cctp
- https://www.prnewswire.com/news-releases/chainlink-runtime-environment-goes-live-unlocking-institutional-tokenization-at-scale-302604067.html
- https://fintechspecs.com/blog/chainlink-swift-blockchain-ledger-tokenized-deposits-2026/
- Local: `references/chainlink-agent-skills/` (CRE, CRE Connect, Data Streams, Data Feeds, CCIP, ACE, Confidential AI Attester skills), `reports/Agent native crypto fiat gateway validation.md`, `reports/Payments full product.md`
