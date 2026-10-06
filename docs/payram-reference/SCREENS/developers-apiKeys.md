# Payram | API Keys

- **Route:** `/developers/apiKeys`
- **Slug:** `developers-apiKeys`
- **Group:** developers
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/developers-apiKeys/screenshot.png`](../captures/developers-apiKeys/screenshot.png)
- **Raw DOM:** [`captures/developers-apiKeys/dom.html`](../captures/developers-apiKeys/dom.html)
- **Network log:** [`captures/developers-apiKeys/network.json`](../captures/developers-apiKeys/network.json)
- **Interactions captured:** 1 ([browse](../captures/developers-apiKeys/interactions/))

## Snapshot

> Payram | API Keys Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Developers / API Keys Coming Soon

## Network calls observed on first paint

- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/blockchains`
- `POST /api/v1/websocket-token/create`

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
