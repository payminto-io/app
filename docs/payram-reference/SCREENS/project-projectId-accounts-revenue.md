# Payram | Revenue

- **Route:** `/project/all/accounts/revenue`
- **Slug:** `project-projectId-accounts-revenue`
- **Group:** accounts
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/project-projectId-accounts-revenue/screenshot.png`](../captures/project-projectId-accounts-revenue/screenshot.png)
- **Raw DOM:** [`captures/project-projectId-accounts-revenue/dom.html`](../captures/project-projectId-accounts-revenue/dom.html)
- **Network log:** [`captures/project-projectId-accounts-revenue/network.json`](../captures/project-projectId-accounts-revenue/network.json)
- **Interactions captured:** 2 ([browse](../captures/project-projectId-accounts-revenue/interactions/))

## Snapshot

> Payram | Revenue Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Accounts / Revenue Coming Soon! We

## Network calls observed on first paint

- `POST /api/v1/websocket-token/create`
- `GET /api/v1/blockchains`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/external-platform/{id}`

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
