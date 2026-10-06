# Payram | Documentation

- **Route:** `/developers/documentation`
- **Slug:** `developers-documentation`
- **Group:** developers
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/developers-documentation/screenshot.png`](../captures/developers-documentation/screenshot.png)
- **Raw DOM:** [`captures/developers-documentation/dom.html`](../captures/developers-documentation/dom.html)
- **Network log:** [`captures/developers-documentation/network.json`](../captures/developers-documentation/network.json)
- **Interactions captured:** 1 ([browse](../captures/developers-documentation/interactions/))

## Snapshot

> Payram | Documentation Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Developers

## Network calls observed on first paint

- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/external-platform/details`
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

- Part of the **Developers** area. See [`FLOWS/webhook-config.md`](../FLOWS/webhook-config.md).
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
