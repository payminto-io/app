# Payram | Onramp Payments

- **Route:** `/onramp-payments`
- **Slug:** `onramp-payments`
- **Group:** onramp
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/onramp-payments/screenshot.png`](../captures/onramp-payments/screenshot.png)
- **Raw DOM:** [`captures/onramp-payments/dom.html`](../captures/onramp-payments/dom.html)
- **Network log:** [`captures/onramp-payments/network.json`](../captures/onramp-payments/network.json)
- **Interactions captured:** 4 ([browse](../captures/onramp-payments/interactions/))

## Snapshot

> Payram | Onramp Payments Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Onramp payments See on-ram

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `GET /api/v1/onramper-payments`
- `GET /api/v1/blockchains`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/payment-channels`
- `GET /api/v1/wallets`
- `GET /api/v1/onramper-payments/metrics`
- `GET /api/v1/payments-app`
- `GET /api/v1/payment-channels/project/{id}`
- `GET /api/v1/blockchain-currency`
- `GET /api/v1/project/{id}/blockchain-currency`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Read more"

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
