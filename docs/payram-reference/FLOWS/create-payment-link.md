# Create payment link

The merchant-facing flow that converts a fiat amount + a target chain into a
hosted checkout URL the merchant can ship to a customer. This is the **single
most-used write path in PayRam** — it's the one feature every demo pitches.

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/project/all/payments/createPaymentLink` | [project-projectId-payments-createPaymentLink.md](../SCREENS/project-projectId-payments-createPaymentLink.md) |
| 2 | `/project/all/payments/allPayments` (after success) | [project-projectId-payments-allPayments.md](../SCREENS/project-projectId-payments-allPayments.md) |
| 3 | `/pay?paymentReference=…` (the customer-facing checkout) | [pay.md](../SCREENS/pay.md) |
| 4 | `/success` (customer-facing post-pay) | [success.md](../SCREENS/success.md) |

## Sequence

1. Merchant lands on `/project/all/payments/createPaymentLink`. The SPA fires:
   - `GET /api/v1/external-platform/all`
   - `GET /api/v1/blockchains`
   - `GET /api/v1/blockchain-currency`
   - `GET /api/v1/payment-channels` and `/api/v1/payment-channels/project/all`
   - `GET /api/v1/project/all/blockchain-currency` (the per-project enabled set)
   - `GET /api/v1/configuration/key/withdrawal-payout-min-amount` (sanity bound)
2. Form fields (captured via DOM + research doc):
   - **Amount** + **fiat currency** dropdown (USD/EUR/GBP/INR/…)
   - **Customer email** (optional, required if email receipts are on)
   - **Customer ID** (optional free-text identifier)
   - **Description** (optional, shows on the checkout)
   - **Network** — chain selector grid (BTC, ETH, BASE, POLYGON, TRX)
   - **Asset** — filtered by selected network (USDT, USDC, ETH, BTC, …)
   - **Expiry** — minutes selector (default 30, allowed 5–1440)
   - **Success URL** / **Cancel URL** (optional, used by `/pay` to redirect)
3. Submit → `POST /api/v1/external-platform/all/payment` (mutating; not in
   our safe crawl). Returns the new `payment` object including
   `paymentReference` (a UUID-like string), `address`, `expiresAt`, `status: "open"`.
4. The SPA opens a **success modal** (react-responsive-modal) showing:
   - The hosted checkout URL `https://<host>/pay?paymentReference=<ref>`
   - A QR code (qrcode.react) of the same URL
   - Copy / share buttons
   - "Open" button → opens the URL in a new tab
5. Closing the modal navigates to `/project/all/payments/allPayments` with
   the new row at the top, status pill `Pending`.

## Customer side (`/pay`)

The customer-facing checkout is a **public** route (no auth). It reads
`paymentReference` from the query string and calls:

- `GET /api/v1/payment/reference/{ref}` (404 in our crawl because we visited
  it without a real reference — see the `null` URLs in `API_CONTRACTS.md`)
- Polls payment status every ~5s OR opens a websocket via
  `POST /api/v1/websocket-token/create`
- On chain confirmation, redirects to `/success` (if no custom successURL was set)

## Status state machine

| Status | Meaning | Trigger |
|---|---|---|
| `pending` | Address generated, no funds received | Initial |
| `partially_filled` | Some funds received but `< amount` | Deposit listener |
| `filled` | Exact amount or `>= amount` received | Deposit listener |
| `over_filled` | More than amount received | Deposit listener |
| `cancelled` | Merchant cancelled or expiry hit | `expiresAt < now` |

These match the **5 chip filters** at the top of the All Payments table
captured in the DOM.

## Webhook

On every status transition the backend enqueues a webhook to all configured
endpoints — see [`webhook-config.md`](./webhook-config.md).

## Gaps vs. Payminto clone

- Our `payminto/frontend/src/app/(dashboard)/payments/create/page.tsx`
  currently has only the bare-amount/email form. It's missing: the chain
  selector grid, the asset selector, expiry picker, success/cancel URL,
  and the success modal with QR + share.
- The `pay` and `success` public routes do not exist in the clone yet.
- We have no websocket layer; status updates would currently rely on polling.
