# Payram

- **Route:** `/settings/paymentsApp`
- **Slug:** `settings-paymentsApp`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-paymentsApp/screenshot.png`](../captures/settings-paymentsApp/screenshot.png)
- **Raw DOM:** [`captures/settings-paymentsApp/dom.html`](../captures/settings-paymentsApp/dom.html)
- **Network log:** [`captures/settings-paymentsApp/network.json`](../captures/settings-paymentsApp/network.json)
- **Interactions captured:** 3 ([browse](../captures/settings-paymentsApp/interactions/))

## Snapshot

> Payram Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings Payment QR Generator Generate QR co

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/blockchains`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/external-platform/{id}/members`
- `GET /api/v1/configuration/default`

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
