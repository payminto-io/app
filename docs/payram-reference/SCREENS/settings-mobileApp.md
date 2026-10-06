# Payram | Integrations

- **Route:** `/settings/mobileApp`
- **Slug:** `settings-mobileApp`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-mobileApp/screenshot.png`](../captures/settings-mobileApp/screenshot.png)
- **Raw DOM:** [`captures/settings-mobileApp/dom.html`](../captures/settings-mobileApp/dom.html)
- **Network log:** [`captures/settings-mobileApp/network.json`](../captures/settings-mobileApp/network.json)
- **Interactions captured:** 2 ([browse](../captures/settings-mobileApp/interactions/))

## Snapshot

> Payram | Integrations Payram Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings / Mobile Appl

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `GET /api/v1/blockchains`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/system`

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
