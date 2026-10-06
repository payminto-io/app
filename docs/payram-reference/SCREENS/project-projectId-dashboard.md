# Payram | Dashboard

- **Route:** `/project/all/dashboard`
- **Slug:** `project-projectId-dashboard`
- **Group:** dashboard
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/project-projectId-dashboard/screenshot.png`](../captures/project-projectId-dashboard/screenshot.png)
- **Raw DOM:** [`captures/project-projectId-dashboard/dom.html`](../captures/project-projectId-dashboard/dom.html)
- **Network log:** [`captures/project-projectId-dashboard/network.json`](../captures/project-projectId-dashboard/network.json)
- **Interactions captured:** 7 ([browse](../captures/project-projectId-dashboard/interactions/))

## Snapshot

> Payram | Dashboard Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Dashboard Last 30 Days All Time 

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `GET /api/v1/blockchains`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/external-platform/{id}/analytics/groups`
- `POST /api/v1/external-platform/{id}/analytics/groups/{id}/graph/{id}/data`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Network"
- "Currency"
- "Network"
- "Currency"

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
