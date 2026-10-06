# Payram | Sweep USDT

- **Route:** `/sweepIn/usdt`
- **Slug:** `sweepIn-usdt`
- **Group:** sweep
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/sweepIn-usdt/screenshot.png`](../captures/sweepIn-usdt/screenshot.png)
- **Raw DOM:** [`captures/sweepIn-usdt/dom.html`](../captures/sweepIn-usdt/dom.html)
- **Network log:** [`captures/sweepIn-usdt/network.json`](../captures/sweepIn-usdt/network.json)
- **Interactions captured:** 1 ([browse](../captures/sweepIn-usdt/interactions/))

## Snapshot

> Payram | Sweep USDT Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile

## Network calls observed on first paint

- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/blockchains`

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

- Part of the **Funds Consolidation (sweepIn)** flow. See [`FLOWS/sweep-cycle.md`](../FLOWS/sweep-cycle.md).
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
