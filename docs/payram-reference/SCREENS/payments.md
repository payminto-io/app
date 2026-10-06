# Payram | Payments

- **Route:** `/payments`
- **Slug:** `payments`
- **Group:** public
- **Auth required:** **no** (public)
- **Screenshot:** [`captures/payments/screenshot.png`](../captures/payments/screenshot.png)
- **Raw DOM:** [`captures/payments/dom.html`](../captures/payments/dom.html)
- **Network log:** [`captures/payments/network.json`](../captures/payments/network.json)
- **Interactions captured:** 1 ([browse](../captures/payments/interactions/))

## Snapshot

> Payram | Payments To, Amount in USD $0.00 Send 0.00 PayRam is a self-hosted payments software that is publicly downloadable and independently deployed by merchants or individuals. PayRam makes no repr

## Network calls observed on first paint

- `GET /null/api/v1/blockchain-currency/reference/null`
- `GET /null/api/v1/wallets/reference/null`
- `GET /null/api/v1/wallet/reference/null`
- `GET /null/api/v1/payment-channels/reference/null`
- `GET /null/api/v1/payment/reference/null`
- `GET /null/api/v1/external-platform/reference/null`
- `GET /api/v1/ticker`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

_(none in initial DOM — likely icon buttons without text content; check interactions/ for the screenshots)_

## Layout & components

The page extends the dashboard shell defined in [`COMPONENT_LIBRARY.md`](../COMPONENT_LIBRARY.md):
- **Top bar** — Testnet badge, project switcher, profile menu
- **Left sidebar** — collapsible nav with the groups Dashboard / Payments /
  Onramp / Growth / Funds Consolidation / Wallet management / Withdraw /
  Developers / Settings / Profile
- **Main pane** — page-specific content (see screenshot)

## Notes

- General dashboard route.
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
