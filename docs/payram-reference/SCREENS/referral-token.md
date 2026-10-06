# Payram

- **Route:** `/referral/demo`
- **Slug:** `referral-token`
- **Group:** public
- **Auth required:** **no** (public)
- **Screenshot:** [`captures/referral-token/screenshot.png`](../captures/referral-token/screenshot.png)
- **Raw DOM:** [`captures/referral-token/dom.html`](../captures/referral-token/dom.html)
- **Network log:** [`captures/referral-token/network.json`](../captures/referral-token/network.json)
- **Interactions captured:** 1 ([browse](../captures/referral-token/interactions/))

## Snapshot

> Payram Your session is over Reload the Page

## Network calls observed on first paint

- `GET /api/v1/referral/referrers`

See [`API_CONTRACTS.md`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_

## Buttons (initial DOM)

- "Reload the Page"

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
