# Payram | Funds Consolidation

- **Route:** `/sweepIn`
- **Slug:** `sweepIn`
- **Group:** sweep
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/sweepIn/screenshot.png`](../captures/sweepIn/screenshot.png)
- **Raw DOM:** [`captures/sweepIn/dom.html`](../captures/sweepIn/dom.html)
- **Network log:** [`captures/sweepIn/network.json`](../captures/sweepIn/network.json)
- **Interactions captured:** 4 ([browse](../captures/sweepIn/interactions/))

## Snapshot

> Payram | Funds Consolidation Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Funds Consolidation Fu

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `GET /api/v1/blockchains`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/blockchain/ETH`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/configuration/default`
- `GET /api/v1/addresses/balance`
- `GET /api/v1/sweeps`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Funds"
- "Transaction History"

## Layout & components

The page extends the dashboard shell defined in [`COMPONENT_LIBRARY.md`](../COMPONENT_LIBRARY.md):
- **Top bar** — Testnet badge, project switcher, profile menu
- **Left sidebar** — collapsible nav with the groups Dashboard / Payments /
  Onramp / Growth / Funds Consolidation / Wallet management / Withdraw /
  Developers / Settings / Profile
- **Main pane** — page-specific content (see screenshot)

## Notes

- Part of the **Funds Consolidation (sweepIn)** flow. See [`FLOWS/sweep-cycle.md`](../FLOWS/sweep-cycle.md).
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
