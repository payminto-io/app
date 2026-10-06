# Payram | Updater

- **Route:** `/settings/updater`
- **Slug:** `settings-updater`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-updater/screenshot.png`](../captures/settings-updater/screenshot.png)
- **Raw DOM:** [`captures/settings-updater/dom.html`](../captures/settings-updater/dom.html)
- **Network log:** [`captures/settings-updater/network.json`](../captures/settings-updater/network.json)
- **Interactions captured:** 3 ([browse](../captures/settings-updater/interactions/))

## Snapshot

> Payram | Updater Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings System Updater Current Ve

## Network calls observed on first paint

- `GET /api/v1/health`
- `GET /api/v1/system/updater/history`
- `GET /api/v1/version`
- `GET /api/v1/system/updater/status`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/system/updater/inspect`
- `GET /api/v1/external-platform/{id}`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/blockchains`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Check for Updates"
- "View Full History"

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
