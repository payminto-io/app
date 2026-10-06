# Payram | Activity Log

- **Route:** `/settings/activityLog`
- **Slug:** `settings-activityLog`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-activityLog/screenshot.png`](../captures/settings-activityLog/screenshot.png)
- **Raw DOM:** [`captures/settings-activityLog/dom.html`](../captures/settings-activityLog/dom.html)
- **Network log:** [`captures/settings-activityLog/network.json`](../captures/settings-activityLog/network.json)
- **Interactions captured:** 8 ([browse](../captures/settings-activityLog/interactions/))

## Snapshot

> Payram | Activity Log Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings / Activity Log Activ

## Network calls observed on first paint

- `GET /api/v1/blockchains`
- `GET /api/v1/external-platform/{id}`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/activity-log/event-categories`
- `GET /api/v1/external-platform/details`
- `POST /api/v1/internalMembers`
- `GET /api/v1/activity-log`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "1"
- "2"
- "12"

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
