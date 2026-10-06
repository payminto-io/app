# Payram

- **Route:** `/success`
- **Slug:** `success`
- **Group:** public
- **Auth required:** **no** (public)
- **Screenshot:** [`captures/success/screenshot.png`](../captures/success/screenshot.png)
- **Raw DOM:** [`captures/success/dom.html`](../captures/success/dom.html)
- **Network log:** [`captures/success/network.json`](../captures/success/network.json)
- **Interactions captured:** 0 ([browse](../captures/success/interactions/))

## Snapshot

> Payram Your payment is successful! Own your payments

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
