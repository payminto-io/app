# Payram | Edit Role

- **Route:** `/settings/roleManagement/role/1`
- **Slug:** `settings-roleManagement-role-roleId`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-roleManagement-role-roleId/screenshot.png`](../captures/settings-roleManagement-role-roleId/screenshot.png)
- **Raw DOM:** [`captures/settings-roleManagement-role-roleId/dom.html`](../captures/settings-roleManagement-role-roleId/dom.html)
- **Network log:** [`captures/settings-roleManagement-role-roleId/network.json`](../captures/settings-roleManagement-role-roleId/network.json)
- **Interactions captured:** 2 ([browse](../captures/settings-roleManagement-role-roleId/interactions/))

## Snapshot

> Payram | Edit Role Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings / Role Management Admin

## Network calls observed on first paint

- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/blockchains`
- `POST /api/v1/websocket-token/create`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Edit Permissions"

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
