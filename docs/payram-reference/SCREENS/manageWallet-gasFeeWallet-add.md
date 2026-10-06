# Payram

- **Route:** `/manageWallet/gasFeeWallet/add`
- **Slug:** `manageWallet-gasFeeWallet-add`
- **Group:** wallets
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/manageWallet-gasFeeWallet-add/screenshot.png`](../captures/manageWallet-gasFeeWallet-add/screenshot.png)
- **Raw DOM:** [`captures/manageWallet-gasFeeWallet-add/dom.html`](../captures/manageWallet-gasFeeWallet-add/dom.html)
- **Network log:** [`captures/manageWallet-gasFeeWallet-add/network.json`](../captures/manageWallet-gasFeeWallet-add/network.json)
- **Interactions captured:** 2 ([browse](../captures/manageWallet-gasFeeWallet-add/interactions/))

## Snapshot

> Payram Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Wallet management Add Secret Vault: Add the 

## Network calls observed on first paint

- `GET /api/v1/blockchains`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/secrets-vault-activities`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

- `Wallet Name`
- `Private key`

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
