# Payram

- **Route:** `/addresses`
- **Slug:** `addresses`
- **Group:** public
- **Auth required:** **no** (public)
- **Screenshot:** [`captures/addresses/screenshot.png`](../captures/addresses/screenshot.png)
- **Raw DOM:** [`captures/addresses/dom.html`](../captures/addresses/dom.html)
- **Network log:** [`captures/addresses/network.json`](../captures/addresses/network.json)
- **Interactions captured:** 1 ([browse](../captures/addresses/interactions/))

## Snapshot

> Payram Enter xPub or Seed Phrase: Select Network: Ethereum Polygon (Matic) Bitcoin Bitcoin Testnet Tron Generate Addresses Get Balances No addresses generated yet.

## Network calls observed on first paint

_(none — static page)_

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Generate Addresses"
- "Get Balances"

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
