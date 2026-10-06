# Payram | Hot Wallet

- **Route:** `/manageWallet/hot-wallet/1`
- **Slug:** `manageWallet-hot-wallet-id`
- **Group:** wallets
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/manageWallet-hot-wallet-id/screenshot.png`](../captures/manageWallet-hot-wallet-id/screenshot.png)
- **Raw DOM:** [`captures/manageWallet-hot-wallet-id/dom.html`](../captures/manageWallet-hot-wallet-id/dom.html)
- **Network log:** [`captures/manageWallet-hot-wallet-id/network.json`](../captures/manageWallet-hot-wallet-id/network.json)
- **Interactions captured:** 2 ([browse](../captures/manageWallet-hot-wallet-id/interactions/))

## Snapshot

> Payram | Hot Wallet Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Wallet management / Hot wallet 

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/blockchains`
- `GET /api/v1/wallets/{id}`
- `GET /api/v1/configuration/key/wallet_connect_id`
- `GET /api/v1/configuration/default`
- `GET /api/v1/blockchain-currency`

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

- Part of the **Wallet management** flow. See [`FLOWS/deposit-wallet-setup.md`](../FLOWS/deposit-wallet-setup.md).
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
