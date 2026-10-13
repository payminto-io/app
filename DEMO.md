# Payminto demo for Crypto World's Fair

Show payment acceptance on Solana and Tempo, then the payment state and ledger evidence that justify the result.
The product URLs are together in [website.md](website.md); the full project context is in [docs/WORLDS_FAIR.md](docs/WORLDS_FAIR.md).
Use the [end-to-end use-case guide](docs/USE_CASES.md) for actual API operations, token/recipient checks, fee boundaries, and reconciliation.

## Recording status

A current 2-minute-39-second narrated product story has been rendered from `solana-localnet-flow.mp4` and `tempo-moderato-flow.mp4`, with captions, timed transcript, step screenshots, and a recording evidence manifest.
The first captures a local-validator test-mint invoice and sweep; the second captures a Moderato payment through a local gateway.
The real Tempo transfers were submitted through an external script while checkout waited for backend confirmation.
Wallet-button signing has separate controlled-wallet regression evidence.
The current demo is a local file, and publicly hosted submission demo and founder-pitch URLs have not been supplied in these docs.
It shows actual invoice creation, checkout confirmation, merchant records, and the Tempo explorer.
An explanatory journal card uses the recorded accounting evidence and is clearly separate from the application UI; ledger and sweep evidence is also documented separately.
Login entry is excluded, and local-validator/testnet labels remain visible.
The portal requests a demo up to 3 minutes and a pitch up to 2 minutes on YouTube, Loom, or Vimeo, as recorded during submission preparation.
The [legacy media archive](website.md#legacy-media-archive) contains the earlier launch motion graphic and deck, with earlier branding and track positioning.
Those assets should not be submitted as a completed current demo or pitch.

## Start the local showcase

Follow [CONTRIBUTING.md](CONTRIBUTING.md), using your own disposable test database and test money.
Run these targets from the root in separate terminals:

```sh
make backend
NEXT_PUBLIC_API_URL=http://localhost:8090/api/v1 make frontend
make checkout
```

Source ports are API 8090, dashboard 3003, and checkout 3002.
The optional Compose backend uses 8080 and needs matching frontend API URLs.
Use an actual merchant session and a payment reference created by the backend when demonstrating a real payment.

## Current product-story beats

The nine beats lead with the merchant's operational problem and explain the infrastructure behind the recorded payments.
They cover merchant-operated infrastructure, exact issued checkout instructions, Solana finality, per-asset journals and sweep recovery, Tempo invoice memos, verified receipts, replay/retry protection, and common merchant records.
The separate v2 switch's fee snapshots and failed-attempt retries are identified as tested capabilities outside the recorded v1 invoice flow.

## Current 2:39 product-story cut

| Time | Show | Evidence boundary |
| --- | --- | --- |
| 0:00-0:09 | Open on completed merchant records and the reconciliation problem. | Recorded test-environment payments. |
| 0:09-0:24 | Merchant-operated gateway: checkout, verification and accounting. | Open-source infrastructure; no new production rollout claim. |
| 0:24-0:40 | Solana issued amount, wallet owner destination and QR. | The source recording uses a custom local-validator mint labelled USDC, not Circle-issued USDC. |
| 0:40-0:56 | Finalized transfer, Paid checkout and merchant record. | Transfer submitted outside the browser; this is the v1 invoice path. |
| 0:56-1:18 | Deposit journal and Solana sweep/recovery explanation. | Journal card derives from recorded backend evidence and is separate from the filmed UI; Solana invoice update follows the atomic deposit/journal commit. |
| 1:18-1:38 | Tempo shared recipient and invoice memo. | Issued TIP-20 token and Moderato testnet with a local gateway. |
| 1:38-1:56 | Explorer receipt, Paid checkout and merchant record. | Successful 12.5 pathUSD receipt independently verified. |
| 1:56-2:20 | Both merchant records, separate assets and retry protection. | Separate v2 fee snapshots and failed-attempt retries are source/test capabilities outside the filmed invoice flow. |
| 2:20-2:39 | Product/source links, current environments and disclosure. | Includes prior work; public Tempo integration remains unverified. |

The recorded Tempo payment was 1.5 pathUSD for invoice `84933e7d-d0fc-46dd-b616-b602be747f9e`.
[View its Moderato transaction](https://explore.testnet.tempo.xyz/tx/0x0bd786d78b7205537319369b9dc30a8d09a79854eae0ac809223aece94a4a00d) and [the original local evidence](tracks/tempo/README.md#recorded-real-network-evidence).
Subsequent changes were verified with local unit, browser-fixture, and disposable PostgreSQL tests.
A new public-deployment payment on the integrated release has not been established.

The later recorded Tempo invoice is `3da2ecad-f409-4d3b-9356-966030f52073` for 12.5 pathUSD, with [Moderato transaction](https://explore.testnet.tempo.xyz/tx/0x406a5a9f529666fc04b8c609f187d09b3c419d9a3caa04e114100ce701d1cf8f) and a local deposit journal.
The later Solana invoice is `b7eda30a-ac7d-4e41-95aa-5c4f7b01506b` for 25 local test-mint tokens, with recorded transfer/sweep signatures and later verified hot-wallet token balance.
Its historical transaction receipts were unavailable from the local validator at independent review time.
Use the [evidence table and limitations](docs/USE_CASES.md#12-recorded-demonstrations-and-evidence-levels) rather than presenting either recording as a public or mainnet release.

## Fixture click path

These development previews use sample data and do not prove a payment happened:

- [Dashboard](http://localhost:3003/design/preview), [payment-link builder](http://localhost:3003/design/preview/links/new), and [links list](http://localhost:3003/design/preview/links).
- [Solana address state](http://localhost:3002/preview?state=awaiting&theme=light).
- [Tempo address and memo](http://localhost:3002/preview?state=tempo&theme=light) and [partial-payment state](http://localhost:3002/preview?state=tempo-underpaid&theme=dark).
- [Paid state](http://localhost:3002/preview?state=paid&theme=light), explicitly labelled as a fixture.

For real checkout, open `/pay/<reference>` on port 3002 using an issued reference.
Show receipt amounts only when the API or ledger supplies them, and never resend the full invoice to demonstrate a partial-payment remainder.
For reusable links, the management/render/pay APIs and a basic `/l/<short_code>` method-to-invoice handoff now exist.
The hosted page preserves a rendered form's submission identity for repeat submissions; variable amounts, required customer forms, and Tempo `pl_<UUID>` references remain unsupported.
Demonstrate those as [documented capabilities and limits](docs/USE_CASES.md#7-merchant-manages-reusable-payment-links), separate from the recorded single-invoice path.
The [standalone Anchor receipt prototype](solana/README.md) is not connected to the gateway checkout and was not used in either recorded payment.

## Disclosures to keep in the recording

Payminto predates the event; the existing private core was imported at `fa8517f` on October 6 UTC.
Earlier Bitcoin/EVM/Tron clients, MCP server, and widget are prior work.
The event-period improvements and October 13 IST Tempo/QA work are described in [the project disclosure](docs/WORLDS_FAIR.md#prior-work-and-ai-disclosure).
Claude Code and Codex assisted development, review, testing, and documentation.
The current documentation makes no new customer, revenue, fundraising, or mainnet-launch claims.
The founder reports sharing the project with hackers in Hong Kong and Singapore for developer feedback; confirm exact events and affiliations before naming them in a pitch.
