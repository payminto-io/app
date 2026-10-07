# Hosted checkout redesign report

Branch: `checkout-ui`, worktree `gateway-wt-checkout-ui`. Date: 2026-10-07.
Scope: `checkout/` only. `frontend/app/(public)/pay` shares no components with it and was not touched.
Brand files (`frontend/components/logo.tsx`, `public/brand/`, `app/icon.svg`) were not touched; the footer references `/brand/mark.svg` from the brand branch as a background image so a missing file renders nothing.

## Checks

- `npx tsc --noEmit`: pass
- `npm run lint`: pass (no output)
- `npm test`: 1 file, 12 tests, pass
- Audit snippet (JUNIOR_BRIEF.md) at 390 with a coarse pointer on choose, awaiting, paid, underpaid, card (light and dark): `overrun: false`, no clipped boxes, no control under 44px, no text under 11px. The only hits are the `sr-only` radio inputs, a documented false positive.

## What shipped

### Design

- Ink on canvas, one accent (Tide), amber only for the test band. No gradient, blur, eyebrow, pill, emoji or illustration remains; the old aurora panel, violet buttons and ShieldCheck brand mark are gone.
- 1440: a 1040px frame, summary column on the left (merchant mark, name, total at `display`, line items with one bold "Total due" line), the flow in a bordered card on the right. The summary is not a filled pane.
- 390: single column, the summary collapses to a header row (mark, name, total, chevron) that discloses the lines; the pay button is sticky above the safe area on the choose and card steps; 16px gutters; 48px controls.
- The rail (Received, Final, Settled) is on every chain state; the fill is the one orchestrated motion (400ms). Confirming uses the 3-dot pulse, the only loop. Both become static under `prefers-reduced-motion`.
- Amounts: tabular figures, code after the number at 0.75em floored at 11px, never `$`. Identifiers, addresses and the timer are mono.
- Dark follows `prefers-color-scheme`; `data-theme` on `<html>` overrides it (the preview uses this).

### Files

- `checkout/app/globals.css`: tokens as `@theme` entries, resets in `@layer base` (unlayered resets were overriding Tailwind utilities), `.btn`, `.copyfield`, `.rail-seg`, `.pulse`, `.tap`, skeleton.
- `checkout/lib/model.ts`: `CheckoutPayment` view model, `fromApi` (maps only what the API sends), `viewState`, `remainingDue`, `overage`, `railStep`, countdown helpers.
- `checkout/lib/money.ts`: section 8 formatting plus exact decimal subtract and compare.
- `checkout/lib/checkout.ts`: `getJSON` with `ApiError` (404 becomes the not-found screen), ticker estimate and BIP-21 helpers kept.
- `checkout/components/ui.tsx`: `Amount`, `CopyField`, `Countdown`, `Rail`, `Pulse`, `ExplorerLink`, `Row`, `PoweredBy`.
- `checkout/components/summary.tsx`, `methods.tsx`, `card-slot.tsx`, `chain-pay.tsx`, `status.tsx`, `screens.tsx`, `checkout-view.tsx` (pure render by state), `checkout.tsx` (live container: fetch, 20s poll, SSE, address assignment).
- `checkout/app/preview/`: development-only route, `notFound()` in production. `?state=` picks a state, `?theme=light|dark` forces a scheme. A "Sample data" chip and the amber test band mark every frame.
- `checkout/lib/fixtures.ts`: the sample payments; invented values, never imported by a live route.
- `docs/design/references.md`: new "Checkout" section with the Mobbin screens opened for this pass.
- `docs/design/screens/checkout/`: 36 captures, listed below.

### States (preview `?state=`)

| State | What it renders | Numbers shown come from |
| --- | --- | --- |
| `choose` | Method radios (Card only when the API offers it; USDC with network chips), pay button names the asset and network | `amountInUSD`, `blockchain-currencies` |
| `single-amount` | Same with no line items, a description line instead | same |
| `card` | Labelled hosted-fields mount `#card-hosted-fields` (`data-hosted-fields-mount`), pay button disabled until mounted. No input touches a PAN. | `amountInUSD` |
| `awaiting` | "Send USDC on Solana", exact amount with copy, address with copy, QR on white, network sentence, waiting row with the quote countdown, rail at 0 | `depositAddress`, `blockchainCode`, `currencyCode`, `amountInUSD` (the backend compares the raw asset amount against it), `expiresAt` |
| `confirming` | Pulse, "Seen on Solana. Confirmations n of N", rail at 1, amount, explorer link | not in API (see below) |
| `paid` | Check, "128.00 USDC to Northwind Supply is final", rail at 2 with the Final timestamp, receipt rows, success message or redirect (1.5s, with a Continue link) | `state`; the rest not in API |
| `underpaid` | Amber, "Received X of Y. Send the remaining Z to the same address", rail at 1, the same address and QR with the remaining amount, countdown | `state`; X and Z not in API, so live renders the sentence without numbers |
| `overpaid` | Note tone, "Received X against Y. The payment is complete; the merchant holds the extra Z", rail at 2, receipt | `state`; numbers not in API |
| `expired` | Mute, "This USDC quote ran out before a transfer arrived. Do not send to the old address." Retry button only when `retryAllowed` | `state` or `expiresAt` in the past |
| `cancelled` | "The merchant cancelled this payment. Nothing is due." | `state` |
| `failed` | Bad, reason sentence, rail with the first segment painted bad, retry when allowed | not in API (no FAILED state yet) |
| `loading` | Skeleton in the frame, `role=status` | - |
| `notfound` | "No payment here" | HTTP 404 from the proxy |
| `error` | Real message plus Try again | any other error |
| `empty` | `/` with no reference | - |
| `paid-message`, `paid-plain` | Paid with a merchant success message; paid with no line items | fixture |

### Accessibility

- The state sentence under each heading is the `aria-live="polite"` region; the heading receives focus on every state change.
- Every interactive element: 2px Tide `:focus-visible` ring; `.tap` grows hit areas to 44px on coarse pointers; the chips are 36px tall on fine pointers and 44px on coarse.
- Copy buttons announce "Copied" through a live span; the QR carries a title; the test band carries `role=note`.
- Reduced motion: transitions and the shimmer collapse to 0.01ms; the pulse becomes three static dots.

### Screenshots

`docs/design/screens/checkout/<state>-<width>-<scheme>.png`.
Light at 390 and 1440: choose, card, awaiting, confirming, paid, underpaid, overpaid, expired, cancelled, failed, notfound; plus single-amount-390, loading-1440, error-390.
Dark at 390 and 1440: choose, card, confirming, paid, underpaid; plus awaiting-1440.
Four captures were refused by the session's permission classifier mid-batch (loading-390-light, single-amount-1440-light, error-1440-light, awaiting-390-dark). They were not retried; the states are identical to their sibling captures apart from width or scheme, and `/preview` renders them on demand.
The awaiting and underpaid 390 captures are full-page because the QR pushes the rail below the fold.

## API fields the checkout cannot render yet

Current payload (`GET /api/v1/public/payment/:reference_id`): `referenceID`, `amountInUSD`, `state`, `expiresAt`, `depositAddress`, `blockchainCode`, `currencyCode`, `merchantName`. `GET /public/blockchain-currencies` gives the method list. The SSE stream carries only `state`.

Everything below is typed as optional on `CheckoutPayment`, rendered only when present, and filled only by the preview fixtures:

| Field | Used by | Without it |
| --- | --- | --- |
| `merchant.logoUrl` | header mark | ink monogram from the merchant name |
| `lineItems[]` (`name`, `quantity`, amount + code), `description` | order summary | total only; the mobile header has no chevron |
| `chain.received` (amount + code) | confirming, under paid, over paid, receipt "Paid" row | the sentences drop the numbers ("Part of the amount has arrived. Send the rest to the same address.") |
| `chain.confirmations`, `chain.requiredConfirmations` | "Confirmations n of N" | sentence stops at "Seen on Solana" |
| `chain.txHash`, `chain.explorerUrl` | explorer link, receipt "Transaction" row | row and link omitted |
| `state = CONFIRMING` (a deposit seen but below threshold) | confirming screen | an OPEN payment with a seen-but-unconfirmed deposit still shows "awaiting" |
| `state = FAILED`, `failureReason` | failed screen | never reached today |
| `paidAt` (`confirmedAt` exists on the model but is not in the public projection) | rail timestamp under Final, receipt "Date" | omitted |
| `success.message`, `success.redirectUrl` | paid screen copy and redirect | default sentence, no redirect |
| `retryAllowed` and an endpoint to re-quote an expired payment | "Get a new quote" on expired, "Try another method" on failed | a sentence asking the merchant for a new link |
| `methods.card` (a card connector on the link) | the Card radio and the hosted-fields step | card is never offered; `fromApi` sets it false |
| `settled` (settlement run completed) | third rail segment | Settled stays unfilled |
| `environment` (`test`/`live`) | 3px amber band | no band |
| `card.brand`, `card.last4` | receipt "Card" row | omitted |
| A quoted asset amount for non-stable assets | exact amount block | only USD stablecoins show "Send exactly"; other assets show the ticker estimate path from the old build is removed from the UI and the sentence says the amount is counted on arrival |

Two behaviours to raise with the backend owner:

1. "Change method" after an address is assigned: the public API has no way to release an address, so the live container just refreshes and lands on awaiting again. The button is honest in the preview and harmless live; it needs an endpoint or should be hidden once `depositAddress` is set.
2. Quote expiry is `expiresAt` on the payment, not a per-address quote; the countdown labels it "Expires in" and the expired copy says "quote" because a reader sees it next to the amount. If the backend keeps one expiry per payment, the copy still holds.

## Commits

- `8fc20b9` Redesign the hosted checkout on the shared tokens
- (this commit) Screens, references, report, preview polish
