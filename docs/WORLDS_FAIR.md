# Payminto at Crypto World's Fair

Project name: **Payminto**.
This is the current project-detail document for [Colosseum's Crypto World's Fair](https://colosseum.com/worldsfair), focused on Solana and Tempo.
The [official event rules](https://colosseum.com/legal/Crypto%20World%27s%20Fair%20Hackathon%20Rules.pdf) identify both ecosystem tracks.
Implementation status below describes the runtime source integrated at `1d63fc8` on October 13, 2026, with later local recording evidence described separately.
A local integration is separate from the public deployment and from a completed portal submission.

## All product URLs

| Destination | URL |
| --- | --- |
| Website | [payminto.io](https://payminto.io) |
| Merchant dashboard | [app.payminto.io](https://app.payminto.io) |
| Hosted checkout | [checkout.payminto.io](https://checkout.payminto.io) |
| Source code | [github.com/payminto-io/app](https://github.com/payminto-io/app) |

[website.md](../website.md) also lists legacy media separately.
The existing public test deployment advertises Solana USDC.
Deployment of the latest integrated Tempo rail and checkout has not been verified publicly.

## Product and audience

Payminto lets merchants operate their own payment gateway rather than reconcile disconnected chain receipts and provider records manually.
The Go API, merchant dashboard, payment links, and hosted checkout share an append-only double-entry ledger.
Versioned fee rules preserve the calculation used on each payment; intents and attempts make provider operations idempotent and unknown outcomes reconcilable.
The intended users are online merchants, marketplaces, and fintech teams accepting stablecoins alongside their existing payment providers.
The backend REST API supports merchant integrations.
Inherited MCP tool definitions require an attached transport before a working AI-client connection can be claimed; the widget also needs integration with the issued-invoice checkout.
The founder reports sharing the project with hackers in Hong Kong and Singapore for developer feedback.
Specific event participation, dates, official acceptance, residence, and regional prize eligibility have not been established by that statement.

The [end-to-end use cases](USE_CASES.md) trace invoice creation, payer selection, chain/token/recipient checks, confirmation, journals, partial/extra payments, and merchant reconciliation.
They distinguish working single-invoice checkout from the reusable-link API and unfinished hosted `/l` integration, and separate v1 invoices from v2 switch fee snapshots and postings.

## Current chain integrations

| Rail | Implemented flow | Evidence and limits |
| --- | --- | --- |
| [Solana](../tracks/solana/README.md) | USDC/USDT addresses, signature detection, finality checks, deposit journals, and recoverable sweeps. | Recorded RPC fixtures, local-validator tests, and a later recorded 25-token local USDC test-mint invoice/sweep; the existing public test deployment advertises USDC. |
| [Tempo](../tracks/tempo/README.md) | Moderato TIP-20 `transferWithMemo`, issued invoice-leg validation, shared merchant address, exact token amounts, wallet checkout, and deduplicated deposit journals. | Original 1.5 pathUSD proof, later 12.5 pathUSD local checkout recording, and local unit/browser/PostgreSQL verification; public release unverified. |
| Ethereum L1 | Inherited EVM adapter plus Ethereum/Sepolia chain, token, and RPC seed configuration. | Prior work; no fresh Ethereum payment proof in the current recordings. |
| Arbitrum | Inherited EVM chain recognition and checkout naming. | No Arbitrum chain/token/RPC seed rows in the audited current seeds, or demonstrated configured payment. |
| Robinhood Chain | No chain-specific implementation/configuration found in the audited source. | Submission selection does not establish an implemented integration. |

Tempo configuration requires test mode and testnet, verifies chain 42431, and supports the seeded PATHUSD and ALPHAUSD assets.
Receipt success and an invoice memo alone are insufficient: token, recipient, merchant ownership, and the issued payment leg must match before credit.
Concurrent replays book once; finalization failures remain retryable; partial and additional transfers update payment state without replacing the deposited amount.
The checkout preserves the issued token and memo, reports wallet rejection, and suppresses a send amount when a partial-payment remainder is unknown.
Tempo refunds, payouts, and sweeps are not implemented, and stablecoin-to-USD comparison currently assumes 1:1 without a conversion trade.

## Recorded real-network payment

The original owner-run Tempo integration recorded this payment through the local backend and disposable `payminto_tempo_test` database:

| Field | Recorded value |
| --- | --- |
| Invoice | `84933e7d-d0fc-46dd-b616-b602be747f9e`, amount 1.50 USD. |
| Asset and amount | 1.5 pathUSD on Moderato testnet. |
| Merchant recipient | `0x82c70b91731914e7b678a232b366336Af1436455`. |
| Invoice memo | `84933e7dd0fc46ddb616b602be747f9e`, encoded as bytes32. |
| Transaction | [Moderato explorer](https://explore.testnet.tempo.xyz/tx/0x0bd786d78b7205537319369b9dc30a8d09a79854eae0ac809223aece94a4a00d). |
| Chain evidence | Block 39362768, log index 26, receipt status 1. |
| Local result | Payment `FILLED`; deposit 3 confirmed; memo credit recorded as `credited`. |
| Local journal | Journal 3: `+1.5 crypto_assets / -1.5 merchant_balance`. |

The explorer transaction is real-network evidence; the invoice and journal values are the recorded local result from that run.
This document does not present them as a fresh payment on the subsequently integrated public deployment.
[Tempo details](../tracks/tempo/README.md) link the relevant code and regression tests.

## Later local product recordings

The recording bundle contains `solana-localnet-flow.mp4` and `tempo-moderato-flow.mp4`, with step screenshots and `local-demo/EVIDENCE.md`.
These recordings show the local gateway's merchant creation, customer checkout, and payment-result screens.
Tempo transfers in these recordings were submitted externally while checkout awaited confirmation; the browser-wallet button was tested separately with controlled test wallets.
The Solana invoice is `b7eda30a-ac7d-4e41-95aa-5c4f7b01506b` for 25 units of a custom six-decimal local-validator test mint.
The recorded transfer and landed sweep have separate journals; later independent inspection found the deposit token account closed and 25 tokens in the hot wallet.
Historical Solana transaction/status queries no longer returned those signatures, so the old receipts could not be independently reconstructed from that validator at review time.
This is local-chain evidence, not Circle USDC, devnet, or mainnet evidence.

The later Tempo invoice is `3da2ecad-f409-4d3b-9356-966030f52073` for 12.5 pathUSD on Moderato.
Its [transaction](https://explore.testnet.tempo.xyz/tx/0x406a5a9f529666fc04b8c609f187d09b3c419d9a3caa04e114100ce701d1cf8f) was independently checked through RPC for receipt success, token amount, recipient, and invoice memo; the local invoice reported `FILLED` and the recording manifest records the corresponding deposit journal.
Full identifiers and evidence limits are in [the recorded-demonstrations table](USE_CASES.md#12-recorded-demonstrations-and-evidence-levels).
These local video assets do not establish a public Tempo deployment or a hosted submission-video URL.

## Verification and unfinished work

Local checks of the integrated release covered backend build, vet, lint, race/unit tests, disposable PostgreSQL Tempo replay/rollback/overpayment tests, and checkout tests, typecheck, lint, and build.
Checkout browser checks used controlled fixtures in mobile/desktop and light/dark layouts, including wallet calls and confirmation events.
Fixture screenshots are demonstrations of rendering and interaction, not independent evidence of transferred money.
The [CI workflow](../.github/workflows/ci.yml) includes a required Tempo PostgreSQL race gate and rejects skipped integration tests.
A hosted CI result requires the integrated commits to be pushed.

Card and UPI connectors, custody adapters, routing across rails, conversion at receipt, and complete deployment automation remain unfinished or unmerged.
The latest public Tempo rollout and new payment evidence on that public release remain outstanding.
Local product recordings exist; public hosting and the final submission demo/pitch media still require supplied URLs.
Existing EVM contracts and optional attestation code are retained in the repository, outside this Solana/Tempo showcase.
No new traction, client names, funding status, or mainnet date is claimed.

## Prior work and AI disclosure

Payminto predates this event.
Commit [fa8517f](https://github.com/payminto-io/app/commit/fa8517fbba5c3b375513b3b773a0a518c84a5d98) imported existing private gateway code and local development on October 6, 2026 UTC.
Earlier Bitcoin, EVM, and Tron clients, the MCP server, and widget are prior work.
Post-import changes include the ledger, fee rules, switch, payment links, Solana work, UI, and documentation.
The Tempo rail, wallet checkout integration, and accompanying safety/regression fixes were developed on October 13 IST before the event's recorded 06:59 UTC close.
Only event-period improvements are presented for judging; the imported baseline is not presented as newly created work.
Captures in `docs/payram-reference/` are research material, not shipped product code.
Unmerged development branches are not presented as deployed features.
Claude Code and Codex contributed to development, review, testing, and documentation.
Human ownership and agent-assisted implementation are disclosed together.

## Current media and access needs

[DEMO.md](../DEMO.md) supplies a recording outline and separates sample previews from real payment evidence.
Submission preparation recorded a demo-video limit of 3 minutes and founder-pitch limit of 2 minutes, hosted on YouTube, Loom, or Vimeo.
A current 2-minute-36-second narrated product demo, captions, transcript, and reviewed six-slide editable Payminto demo-day deck have been prepared locally.
Public video hosting, the required founder pitch, and complete judge access/profile answers remain outstanding.
The earlier deck and launch motion graphic are [legacy assets](../website.md#legacy-media-archive).
