# Payminto demo for Crypto World's Fair

Show payment acceptance on Solana and Tempo, then the payment state and ledger evidence that justify the result.
The product URLs are together in [website.md](website.md); the full project context is in [docs/WORLDS_FAIR.md](docs/WORLDS_FAIR.md).
Use the [end-to-end use-case guide](docs/USE_CASES.md) for actual API operations, token/recipient checks, fee boundaries, and reconciliation.

## Recording status

Two local product recordings have been prepared: `solana-localnet-flow.mp4` and `tempo-moderato-flow.mp4`, with step screenshots and a recording evidence manifest.
The first captures a local-validator test-mint invoice and sweep; the second captures a Moderato payment through a local gateway.
The real Tempo transfers were submitted through an external script while checkout waited for backend confirmation.
Wallet-button signing has separate controlled-wallet regression evidence.
The recordings are local files, and current publicly hosted submission demo and founder-pitch URLs have not been supplied in these docs.
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

## Suggested three-minute walkthrough

| Time | Show | Evidence boundary |
| --- | --- | --- |
| 0:00-0:25 | The merchant problem and Payminto's self-hosted payment links, checkout, and ledger. | Card/UPI connectors, custody, and routing are unfinished. |
| 0:25-0:55 | Create a single invoice in the merchant dashboard and open `/pay/<issued-reference>`. | This is the v1 invoice path; it does not demonstrate the unfinished reusable-link hosted flow or switch fee posting. |
| 0:55-1:30 | Solana: the issued token account, exact mint/network, paid invoice, deposit journal, and later hot-wallet sweep. | The prepared recording uses a custom local-validator test mint named USDC, not Circle USDC, devnet, or mainnet. |
| 1:30-2:15 | Tempo: issued pathUSD contract, shared merchant address, UUID memo, wallet/transfer path, and backend-confirmed state. | The later 12.5 pathUSD run is Moderato testnet with a local gateway; the public Payminto Tempo release is unverified. |
| 2:15-2:40 | The independently checked Moderato transfer and matching local payment/journal evidence; optionally distinguish the original 1.5-token proof. | Show the exact receipt/token/recipient/memo; a preview or submitted wallet request is not payment confirmation. |
| 2:40-3:00 | Prior work, AI use, current progress, and next steps. | Use the disclosure below and avoid unverified traction claims. |

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
For reusable links, the management/render/pay APIs are implemented, but the generated `/l/<short_code>` route and Tempo link-payment reference are unfinished in the integrated source.
Demonstrate those as [documented API capabilities and limits](docs/USE_CASES.md#7-merchant-manages-reusable-payment-links), not as the completed single-invoice recording path.

## Disclosures to keep in the recording

Payminto predates the event; the existing private core was imported at `fa8517f` on October 6 UTC.
Earlier Bitcoin/EVM/Tron clients, MCP server, and widget are prior work.
The event-period improvements and October 13 IST Tempo/QA work are described in [the project disclosure](docs/WORLDS_FAIR.md#prior-work-and-ai-disclosure).
Claude Code and Codex assisted development, review, testing, and documentation.
The current documentation makes no new customer, revenue, fundraising, or mainnet-launch claims.
The founder reports sharing the project with hackers in Hong Kong and Singapore for developer feedback; confirm exact events and affiliations before naming them in a pitch.
