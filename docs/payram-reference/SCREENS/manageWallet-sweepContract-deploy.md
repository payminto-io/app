# Payram | Deploy Contract

- **Route:** `/manageWallet/sweepContract/deploy`
- **Slug:** `manageWallet-sweepContract-deploy`
- **Group:** wallets
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/manageWallet-sweepContract-deploy/screenshot.png`](../captures/manageWallet-sweepContract-deploy/screenshot.png)
- **Raw DOM:** [`captures/manageWallet-sweepContract-deploy/dom.html`](../captures/manageWallet-sweepContract-deploy/dom.html)
- **Network log:** [`captures/manageWallet-sweepContract-deploy/network.json`](../captures/manageWallet-sweepContract-deploy/network.json)
- **Interactions captured:** 2 ([browse](../captures/manageWallet-sweepContract-deploy/interactions/))

## Snapshot

> Payram | Deploy Contract Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Tron TRC 20 Sweep Contract

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/contract-address/blockchain/TRX/contract/sweep_approval`
- `GET /api/v1/blockchains`
- `GET /api/v1/configuration/default`
- `GET /api/v1/blockchain-contract/blockchain/TRX/contract/factory_contract`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

- `Fund Collector Wallet`

## Buttons (initial DOM)

- "Select Wallet"

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
