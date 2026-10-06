# Payram | Missed Payments

- **Route:** `/project/all/payments/missed-payments`
- **Slug:** `project-projectId-payments-missed-payments`
- **Group:** payments
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/project-projectId-payments-missed-payments/screenshot.png`](../captures/project-projectId-payments-missed-payments/screenshot.png)
- **Raw DOM:** [`captures/project-projectId-payments-missed-payments/dom.html`](../captures/project-projectId-payments-missed-payments/dom.html)
- **Network log:** [`captures/project-projectId-payments-missed-payments/network.json`](../captures/project-projectId-payments-missed-payments/network.json)
- **Interactions captured:** 5 ([browse](../captures/project-projectId-payments-missed-payments/interactions/))

## Snapshot

> Payram | Missed Payments Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Missed blockchain payments

## Network calls observed on first paint

- `GET /api/v1/external-platform/details`
- `GET /api/v1/external-platform/{id}`
- `POST /api/v1/websocket-token/create`
- `GET /api/v1/missed-deposit`
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

- Part of the **Payments** flow. See [`FLOWS/create-payment-link.md`](../FLOWS/create-payment-link.md).
- This file was generated from the Phase 2 crawl. For complex routes
  (`dashboard`, `payments-allPayments`, `manageWallet/*`, `settings/*`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
