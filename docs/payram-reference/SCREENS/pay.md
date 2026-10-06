# Payram

- **Route:** `/pay`
- **Slug:** `pay`
- **Group:** public
- **Auth required:** **no** (public)
- **Screenshot:** [`captures/pay/screenshot.png`](../captures/pay/screenshot.png)
- **Raw DOM:** [`captures/pay/dom.html`](../captures/pay/dom.html)
- **Network log:** [`captures/pay/network.json`](../captures/pay/network.json)
- **Interactions captured:** 0 ([browse](../captures/pay/interactions/))

## Snapshot

> Payram Error loading payment session data.

## Network calls observed on first paint

_(none — static page)_

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
