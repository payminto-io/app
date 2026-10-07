# Payments V1 program spec

Status: approved by owner 2026-10-07. Base: Payminto (this repository). References only: Kuberopay (designs), Hyperswitch (switch and routing shape).

## Where this product stands in a crowded market

There are hundreds of payment gateways and dozens of crypto checkouts. The validation report (`../../../reports/Agent native crypto fiat gateway validation.md`) found that every piece we want is sold by someone and nobody ships the whole thing, and that the unclaimed ground is narrow:

1. **No open-source, self-hostable gateway unifies card connectors with stablecoin settlement.** Hyperswitch is card-first with no crypto rail. BTCPay is Bitcoin-first with no cards. PayRam's core is unpublished. Coinbase Commerce's self-custodial product was shut and replaced by a custodial bundle.
2. **Non-custodial gateways cannot settle in fiat** because they never hold the asset. Every one bolts fiat on through a third party or not at all. A custody interface with a non-custodial default and a managed adapter is the shape nobody offers.
3. **Demand is B2B settlement, not consumer pay-with-crypto.** Visa, Mastercard and Stripe settle on Solana; merchants want to be settled in the asset and place they choose, with one ledger.
4. **Routing exists only inside card-only switches.** Hyperswitch routes between card connectors; no product routes across fiat and crypto rails on cost, success rate and settlement preference.
5. **Agent payments are real in count and tiny in dollars.** x402 on Solana is the only one with traffic. We support it as a rail, we do not lead with it.

What we will not win on: more connectors than Hyperswitch, cheaper cards than Stripe, more chains than a crypto-only checkout. What we can win on, and must demonstrate in every presentation:

- **One ledger across rails.** A card payment and a USDC payment land as the same double-entry lines, with fee, conversion and settlement explicit. Nothing else open-source does this.
- **Settlement choice as a merchant setting.** Fiat to bank, USDC on a chosen chain, or hold in asset, per merchant and per link, enforced by policy in core.
- **Custody as a slot.** Direct-to-wallet (no liability), BitGo, Fireblocks, later Squads and Safe, all behind one conformance-tested interface.
- **Routing across rails.** Hyperswitch's routing model (rule based, volume split, success-rate based) extended with rail cost, finality time and the merchant's settlement preference as inputs.
- **Self-hostable with `docker compose up` and no API key.** Mock connector, mock custody, mock chain. Compliance stays with the deployer through named hooks.

Differentiation claims in decks and READMEs must map to one of these five and to a shipped capability. No claim without a demo path.

## V1 deliverable

From one install, a merchant can create a payment link with the full six-step form, be paid by card (one connector) or USDC on Solana, see both in one ledger, be settled in USDC on Solana or fiat through a payout adapter under policy, with custody chosen by configuration. Routing selects the connector per attempt. Full definition: `../../../reports/Payments V1 on Kuberopay.md` (the form and slice; "Kuberopay" in that title refers to the design source) and `../../../reports/Gateway V1 scope.md`.

## Architecture on the Payminto base

Payminto already provides: EVM, Bitcoin and Tron deposit watching, deposit addresses, sweeps, withdrawals, hot and cold wallets, members, platforms, roles, API keys, webhooks, dashboard, admin, hosted pay page, checkout, MCP server. See `docs/ARCHITECTURE.md`.

V1 adds, in Go under `backend/internal/`:

| Module | Package | Reference design |
| --- | --- | --- |
| Ledger | `ledger/` (replace member balance tracking with double-entry) | Kuberopay ledger, ADR 0030, 0053 |
| Fee rules | `fees/` | Kuberopay `admin-fee-rules`, ADR 0053 |
| Payment links | `links/` + model | Kuberopay `payment-link-builder`, the six-step form spec |
| Switch | `switch/` intents, attempts, connector interface, status map | Hyperswitch `payment_intent`/`payment_attempt`, connector status map |
| Routing | `routing/` | Hyperswitch routing (rule, volume split, success based) |
| Card connector | `connectors/card/<provider>` | Hyperswitch connector shape, hosted fields only |
| Solana chain | `blockchain/solana` | Payminto adapter interface |
| Conversion | `conversion/` | convert-at-receipt, executed rate recorded |
| Settlement | `settlement/` policy, destinations, approvals, runs | Kuberopay ADR 0023, 0024, 0027, 0033, 0034, 0056 |
| Custody | `custody/` interface + mock, direct, bitgo | Gateway V1 scope |
| Environments | `environment/` live/test isolation | Kuberopay ADR 0033 |

Frontend in `frontend/` (dashboard) and `checkout/`.

## Rules (from `CLAUDE.md`, repeated because they are checked in review)

Every number is a ledger line or a receipt. No card data on our servers. Live and test isolated at boot. Signer keys never in the API process. Fee rule snapshot on every payment. Additive migrations. Go backend, TypeScript frontends. Tests on every money path; integration tests use the testcontainer harness (`go test -tags=integration ./...`).

## Team roles

| Role | Owns | Model |
| --- | --- | --- |
| Principal engineer | ledger, switch, routing, environment isolation, final review | Fable |
| Blockchain engineer | custody interface and adapters, settlement, conversion | Fable |
| Solana engineer | Solana adapter, USDC SPL deposits, fee sponsorship, Jupiter | Fable |
| Backend engineer | fee rules, payment links, webhooks, API, migrations | Opus |
| Frontend engineer | link form, checkout, settlement and custody settings | Opus |
| UI/UX | form design, checkout states, dashboard reviews | Opus |
| DevOps | compose, CI, mocks, release pipeline, secrets | Opus |
| QA | money-path tests, conformance suite, security review | Fable for security review, Opus otherwise |
| Senior designer | Mobbin reference board, design system, auth, shell, checkout | Fable |
| Junior designer | restyles remaining pages from `docs/design/JUNIOR_BRIEF.md` | Opus |

Design bar: Stripe-level craft with our own identity, never a copy. Every UI ticket reads `docs/design/DESIGN.md` and `docs/design/references.md` (Mobbin references by surface) before starting, and is verified by screenshot at 390px and 1440px in light and dark.

Security review of the money path (tickets 01, 02, 08, 09, 11) is a gate before any live merchant.

## Tickets

Pick the lowest-numbered unblocked ticket with `Status: ready-for-agent`. Set `Status: claimed` before starting, `Status: done` with commit hashes when finished.

| # | Ticket | Owner | Blocked by |
| 01 | Double-entry ledger in Go | Principal | - |
| 02 | Versioned fee rules and fee preview | Backend | 01 |
| 03 | Payment link model and API | Backend | 02 |
| 04 | Payment link form with live preview | Frontend + UI/UX | 03 |
| 05 | Switch core: intents, attempts, connector interface, status map | Principal | 01 |
| 06 | Routing engine | Principal | 05 |
| 07 | Card connector through hosted fields | Backend | 05 |
| 08 | Custody provider interface, mock and direct-to-wallet | Blockchain | 01 |
| 09 | Solana adapter, USDC and USDT side by side | Solana | 08 |
| 10 | Conversion at receipt | Blockchain | 01, 09 |
| 11 | Settlement policy, destinations, approvals, runs | Blockchain | 08, 10 |
| 12 | BitGo custody adapter and conformance suite | Blockchain | 08, 11 |
| 13 | Live and test environment isolation | Principal | 01 |
| 14 | Hosted checkout rendering every link option | Frontend | 04, 07, 09 |
| 15 | Compose with mocks, CI, release pipeline | DevOps | 05, 08 |
| 16 | QA: money-path tests, security review, demo script | QA | 11, 14, 15 |
| 17 | Public checkout projection fields | Backend | 03, 05, 09 |
| 18 | Dashboard per-asset totals and mobile tables | Frontend | 01 |
| 19 | Chainlink CRE research, market fit and module spec | Chainlink | - |

## Distribution: open core, self-hosted by default, white-label in the paid tier

Decided 2026-10-07.

- **Core is open source, Apache 2.0.** Switch, routing, ledger, fee rules, payment links, checkout, dashboard, custody interface with mock, direct-to-wallet and BitGo adapters, Solana and EVM adapters, mock connector, one card connector. Complete and runnable with `docker compose up` and no API key. Self-hosted is the deployment mode of this edition, not a separate product.
- **Enterprise edition is source-available, licensed per deployment.** Lives in `backend/internal/ee/` and `frontend/ee/` behind a licence key (Payminto's existing `ee_event` model is the seed of this boundary). Contains: white-label (brand removal, custom domains, themed checkout, multi-tenant platform mode with sub-merchants), SSO and audit export, additional custody adapters (Fireblocks, MPC vendors, Squads and Safe multisig), PCI vault integration, conversion connectors beyond the first, fiat payout partners, priority support.
- **Hosted tier** runs the enterprise edition for merchants who do not want to self-host.

Why not closed self-hosted only: the research found the unclaimed ground is specifically the open-source, self-hostable fiat-plus-stablecoin gateway. A closed self-hosted product has no distribution edge over the hundred gateways already selling, and open source is how Hyperswitch and BTCPay earned their deployers. Why not everything open: custody adapters and white-label are what deployers pay for, and giving them away leaves no business under the maintainers. Why Apache 2.0 rather than a source-available core: connector and adapter contributions from outsiders are a stated measure of done, and contributors do not arrive for a licence that forbids hosting.

Rule for every ticket: a feature goes in core unless it is in the enterprise list above. Core must never depend on `ee/`; `ee/` plugs in through the same interfaces an outside contributor would use.
