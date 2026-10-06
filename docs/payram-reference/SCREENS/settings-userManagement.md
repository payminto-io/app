# Payram | User Management

- **Route:** `/settings/userManagement`
- **Slug:** `settings-userManagement`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-userManagement/screenshot.png`](../captures/settings-userManagement/screenshot.png)
- **Raw DOM:** [`captures/settings-userManagement/dom.html`](../captures/settings-userManagement/dom.html)
- **Network log:** [`captures/settings-userManagement/network.json`](../captures/settings-userManagement/network.json)
- **Interactions captured:** 4 ([browse](../captures/settings-userManagement/interactions/))

## Snapshot

> Payram | User Management Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings / User Management

## Network calls observed on first paint

- `GET /api/v1/blockchains`
- `GET /api/v1/roles`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/external-platform/{id}`
- `POST /api/v1/internalMembers`
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

- General dashboard route.
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
