# Payram | Integrations

- **Route:** `/settings/integrations`
- **Slug:** `settings-integrations`
- **Group:** settings
- **Auth required:** **yes** (JWT in `localStorage.payram_access_token`)
- **Screenshot:** [`captures/settings-integrations/screenshot.png`](../captures/settings-integrations/screenshot.png)
- **Raw DOM:** [`captures/settings-integrations/dom.html`](../captures/settings-integrations/dom.html)
- **Network log:** [`captures/settings-integrations/network.json`](../captures/settings-integrations/network.json)
- **Interactions captured:** 4 ([browse](../captures/settings-integrations/interactions/))

## Snapshot

> Payram | Integrations Testnet Mode All Projects Dashboard Payments Onramp Growth ASSETS Funds Consolidation Wallet management Withdraw GENERAL Developers Settings Profile Settings / Integrations Integ

## Network calls observed on first paint

- `POST /api/v1/websocket-token/create`
- `GET /api/v1/external-platform/details`
- `GET /api/v1/blockchains`
- `GET /api/v1/external-platform/{id}`
- `GET /api/v1/config/smtp/`
- `GET /api/v1/configuration/key/wallet_connect_id`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Node Details"
- "Email Server"
- "Wallet Connect ID"

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
