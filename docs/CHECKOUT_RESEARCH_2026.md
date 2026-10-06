# Hosted Crypto Checkout Research (2026)

**Research date:** 2026-08-29  
**Scope:** A customer-facing, hosted web checkout for Payminto payment links. This note covers product UX, a separate-app architecture, invoice states, wallet/QR handling, accessibility, and security. It does not cover merchant-dashboard UI or private-key custody.

## Executive decision

Build checkout as a **separately deployed public web app on a dedicated, recognizable origin** such as `pay.payminto.com`, backed by a narrow public Checkout API. The merchant/dashboard application creates a single-use invoice on the server and receives an opaque checkout URL. The payer app reads only a sanitized checkout view and can perform a very small set of state-checked actions (choose an enabled asset/network, request a wallet handoff, and refresh status). It must never receive merchant/admin API keys, signing keys, or dashboard session credentials.

This is the established hosted-checkout pattern:

- Coinbase Business creates a unique checkout URL and ID, says each checkout is single-use, and requires merchants to fulfill from a success webhook or a polled `COMPLETED` state rather than the browser redirect ([Coinbase Checkout APIs](https://docs.cdp.coinbase.com/coinbase-business/checkout-apis/overview)).
- Stripe redirects stablecoin payers to the separate `crypto.stripe.com` surface to choose a wallet, token, and network, then optionally returns them to the merchant ([Stripe stablecoin payments](https://docs.stripe.com/payments/stablecoin-payments)).
- BitPay creates an invoice URL, redirects the payer to a hosted page for wallet and currency selection, and tells merchants to fulfill only after verified server-side completion ([BitPay checkout integration](https://developer.bitpay.com/docs/checkout-integration)).
- OWASP's payment-gateway flow likewise puts order creation on the merchant backend and payment execution on a gateway-hosted page; it treats redirect parameters as untrusted ([OWASP Third-Party Payment Gateway Integration](https://cheatsheetseries.owasp.org/cheatsheets/Third_Party_Payment_Gateway_Integration_Cheat_Sheet.html)).

The separate app can remain in the same monorepo, but it should have an independent build/deployment, origin, CSP, telemetry configuration, and release rollback. This is both a security boundary and a product boundary: the payer should see a fast, calm payment product—not the authenticated merchant dashboard shell.

## What “best-in-class” means here

The best reference products do not make crypto feel like a developer console. They progressively disclose technical detail while keeping the transaction's significant facts unambiguous:

1. Merchant identity, order reference/description, fiat total, and expiry.
2. A combined **asset + network** choice, such as “USDC · Base,” not two ambiguous symbol-only controls.
3. A wallet choice appropriate to the device.
4. A review state showing exact token amount, network, recipient, rate-lock expiry, and estimated network-fee responsibility.
5. A wallet handoff or scannable payment request, with copy fallbacks.
6. Distinct “detected,” “confirming,” and “confirmed” states.
7. A receipt/result with transaction hash, final amount, network, and safe merchant return.

That design supports OWASP's “what you see is what you sign” principle: users must be able to identify and acknowledge significant transaction data, including the destination and amount ([OWASP Transaction Authorization](https://cheatsheetseries.owasp.org/cheatsheets/Transaction_Authorization_Cheat_Sheet.html)).

## Recommended screen and interaction design

### Desktop

Use a restrained two-column composition:

- **Order rail:** verified merchant logo/name, description or order number, fiat total, support link, and “Powered by Payminto.”
- **Payment card:** one focused step at a time—method, wallet/send, confirmation—with a compact progress header.

The surface can be visually rich (deep neutral background, subtle mesh/aurora lighting, crisp glass or elevated card, chain accents) without turning warnings or status into decoration. The transaction details, primary action, and status must dominate.

### Mobile

Use one column and make **Open in wallet** the primary action. QR is primarily a cross-device desktop affordance; on a phone, requiring a user to scan their own screen is a dead end. WalletConnect documents the wallet-choice → deep-link → approval → return flow and recommends mobile linking to reduce user actions ([WalletConnect mobile-linking best practices](https://docs.walletconnect.network/wallet-sdk/best-practices)). Always preserve a manual “Copy amount” and “Copy address” fallback and explain how to return if the wallet cannot deep-link back.

Do not automatically launch a wallet before a user gesture. Let the payer choose a compatible wallet, show the transaction review, then hand off. If WalletConnect is used, register and verify the checkout domain; WalletConnect exposes `VALID`, `INVALID`, `UNKNOWN`, and malicious-domain states specifically to warn against phishing or origin mismatch ([WalletConnect Verify API](https://docs.walletconnect.network/wallet-sdk/web/verify)).

### Asset and network selection

- Present each option as an inseparable identity: token name/symbol, chain name, icon, and a short fee/speed hint based on live backend capability data.
- Filter options by merchant configuration, regional/risk policy, invoice currency, amount bounds, and current availability on the server. The browser must not invent or enable combinations.
- Never identify a token by symbol alone. The backend contract should carry a canonical chain identifier plus native-asset identifier or token contract/mint.
- Make testnet unmistakable across the whole screen and QR, not just a small badge.
- If the selected asset changes, invalidate the previous quote and address/payment request as one atomic server-side operation.

Stripe's hosted crypto flow explicitly lets the customer choose a currency, network, and wallet ([Stripe stablecoin payments](https://docs.stripe.com/payments/stablecoin-payments)); BitPay similarly combines wallet-provider and payment-currency selection in the hosted flow ([BitPay checkout integration](https://developer.bitpay.com/docs/checkout-integration)).

### Amount, address, wallet handoff, and QR

- Show a human token amount as the hero value (for example `25.37 USDC`), with an exact-copy action. Put atomic units and contract identity behind “Payment details,” not in the primary headline.
- Show the fiat total separately from the locked crypto obligation. Include “Rate locked until …” when conversion applies.
- Show the network next to the amount and next to the address; do not rely on a warning at the bottom.
- Show the full recipient in a copyable field and a human-checkable prefix/suffix near the primary action. Provide a transaction explorer link only after validating and constructing it server-side.
- The payment QR should encode a complete, standards-based payment request where the wallet ecosystem supports it—not a bare address. ERC-681 supports recipient, chain ID, native value, or ERC-20 transfer data in a payment URL ([ERC-681](https://eips.ethereum.org/EIPS/eip-681)). Its security section warns that address or amount substitution is profitable and that source integrity and human-verifiable amounts matter.
- Label WalletConnect pairing QR and payment-request QR differently. They are not interchangeable.
- Treat the QR/URI as convenience, not authority. ERC-681 amounts are suggestions a wallet can modify, so the backend must still match the observed chain, token, recipient, and amount to the invoice.
- Prefer a protocol-mediated transaction proposal when supported. BitPay's payment protocol validates that an invoice is still payable and that address, amount, currency, and fee are acceptable before broadcast, specifically to reduce wrong-amount and late-payment errors ([BitPay JSON Payment Protocol](https://developer.bitpay.com/docs/payment-protocol)).

### Confirmation and waiting

Never collapse “transaction broadcast,” “seen in mempool,” and “final enough to fulfill” into one green success state. BitPay explicitly says `paid` only means a transaction was broadcast and is not a payment guarantee; `confirmed`/`complete` represent later confidence/settlement points ([BitPay invoice states](https://developer.bitpay.com/docs/invoice-states)).

Recommended user copy:

- **Waiting for payment:** “Open your wallet or send the exact amount.”
- **Payment detected:** “We found your transaction. Do not send again.”
- **Confirming:** “1 of 3 confirmations. You may close this page; the merchant will be notified.”
- **Confirmed:** “Payment confirmed” plus receipt details and return action.
- **Delayed:** “The network is taking longer than usual” plus transaction hash and a non-destructive refresh/retry-status action.

Use server-sent events or a WebSocket for immediacy and bounded polling with backoff as a fallback. Realtime delivery improves the payer experience; it is not the merchant's fulfillment authority. BitPay recommends instant payment notifications instead of aggressive API polling ([BitPay checkout integration](https://developer.bitpay.com/docs/checkout-integration)).

## Canonical invoice model and user-facing states

Coinbase's current checkout states are `ACTIVE`, `PROCESSING`, `DEACTIVATED`, `EXPIRED`, `COMPLETED`, `FAILED`, `REFUNDED`, and `PARTIALLY_REFUNDED` ([Coinbase Checkout API reference](https://docs.cdp.coinbase.com/api-reference/business-api/rest-api/checkouts/introduction)). BitPay separates base state from exception state, which is useful for under/overpayment ([BitPay invoice states](https://developer.bitpay.com/docs/invoice-states)). Payminto should likewise keep canonical fulfillment state distinct from exception/reconciliation details.

| Canonical state | Payer-facing state | Can fulfill? | Notes |
|---|---|---:|---|
| `ACTIVE` | Choose how to pay / Waiting for payment | No | Single-use invoice accepts a supported selection/payment. |
| `PAYMENT_DETECTED` | Payment detected | No | Transaction observed but not at required confirmations. Disable “send” actions and show “do not send again.” |
| `CONFIRMING` | Confirming `n/N` | No | Preserve tx hash, amount seen, and live confirmation count. |
| `COMPLETED` | Payment confirmed | Yes | Terminal success at the merchant's configured risk threshold. |
| `EXPIRED` | Link expired | No | No accepted payment before the invoice deadline. Offer “request a new link”; do not silently reuse an old quote/address. |
| `DEACTIVATED` / `CANCELLED` | Link no longer active | No | Merchant-initiated terminal state. |
| `FAILED` / `INVALID` | Payment needs attention | No | Explain next action without implying funds vanished; support late confirmation reconciliation. |
| `REFUND_PENDING` | Refund processing | No | Show amount, asset/network, and refund destination/policy. |
| `REFUNDED` / `PARTIALLY_REFUNDED` | Refunded / Partially refunded | No | Receipt state, not a payable state. |

Store exception/reconciliation state alongside the base state:

- **Underpaid:** show amount received, exact remainder, whether the same quote remains valid, and a clear “send remainder” or refund/support path. BitPay does not credit partial invoices and instead refunds them, demonstrating why the policy must be explicit ([BitPay invoice states](https://developer.bitpay.com/docs/invoice-states)).
- **Overpaid:** show total received, excess, and automatic/manual refund policy. BitPay tracks an overpayment exception separately from the paid base state ([BitPay invoice states](https://developer.bitpay.com/docs/invoice-states)).
- **Late payment:** do not discard it under `EXPIRED`. Record `LATE_PAYMENT_DETECTED` for reconciliation and show a support/refund path.
- **Wrong asset/network:** do not credit by symbol or fiat coincidence. Quarantine for manual recovery if technically possible and never promise recovery in the UI.

All transitions should be monotonic and atomically compare the previous revision/state. Reorg handling may move a transaction's confirmation evidence backward, but it must be represented as an explicit risk/reconciliation event rather than an ordinary browser-controlled transition.

## Separate-app technical boundary

```text
Merchant dashboard/app
  -> authenticated merchant API creates invoice from server-trusted order data
  <- opaque checkout URL: https://pay.payminto.com/c/<random-id>

Payer browser on pay.payminto.com
  -> public Checkout API: sanitized invoice view + allowed actions only
  <- status stream/poll response

Chain observers / confirmation workers
  -> canonical payment state + immutable event record
  -> signed, retried, idempotent merchant webhook

Merchant backend
  -> verifies webhook and/or retrieves canonical state
  -> fulfills only COMPLETED
```

Implementation rules:

- Use a high-entropy opaque checkout ID; do not put customer email, amount, merchant secret, or a meaningful database sequence in the URL.
- Checkout creation and all pricing/rounding happen server-side using integer/decimal arithmetic and a persisted quote. Never derive the payable amount from browser floating-point math or assume a stablecoin market price is exactly 1.
- Allocate a unique invoice address or an otherwise unambiguous invoice correlation mechanism. Persist chain ID, asset identity, amount in atomic units, decimals, quote source/time/expiry, and confirmation policy.
- Return URLs are registered/allowlisted at merchant configuration or validated at invoice creation. Do not accept an arbitrary browser-supplied redirect.
- Keep the checkout origin cookie-free if possible. If an anonymous continuity cookie is needed, make it origin-only, short-lived, `Secure`, `HttpOnly`, and explicit `SameSite`; never share the dashboard cookie domain. OWASP recommends narrow cookie scope and warns that permissive parent-domain cookies enable cross-subdomain attacks ([OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)).
- Do not store credentials, API keys, session IDs, JWTs, or refresh tokens in browser storage. OWASP specifically advises against credentials in `localStorage`/`sessionStorage` ([OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)).
- Use `Cache-Control: no-store` for invoice/status responses and `Referrer-Policy: no-referrer` (or a deliberate strict policy) so opaque checkout URLs and transaction details are not leaked to analytics/support destinations.
- Default-deny CORS. The payer app should call only the exact Checkout API origin(s); merchant/admin APIs remain unavailable cross-origin.

## Security controls required before launch

1. **Server-authoritative order:** Recalculate/validate order, discount, fiat amount, asset/network availability, quote, expiry, and redirect destination on the backend. OWASP says cart details and totals must be validated server-side ([OWASP payment gateway integration](https://cheatsheetseries.owasp.org/cheatsheets/Third_Party_Payment_Gateway_Integration_Cheat_Sheet.html)).
2. **No secrets in checkout:** The browser gets only public merchant display data and invoice instructions. API keys stay in a secret manager and are permission/IP restricted; Coinbase also recommends never embedding keys in code, keeping them outside the source tree, and restricting authorized sources/APIs ([Coinbase API security best practices](https://docs.cdp.coinbase.com/get-started/authentication/security-best-practices)).
3. **Verified fulfillment:** Never trust success-page state, query parameters, redirect callbacks, or a payer-submitted tx hash. Match canonical order ID, expected chain/asset/address/amount and required confirmations on the server. OWASP says only server-to-server callbacks should be trusted for verification and fulfillment ([OWASP payment gateway integration](https://cheatsheetseries.owasp.org/cheatsheets/Third_Party_Payment_Gateway_Integration_Cheat_Sheet.html)).
4. **Webhook authenticity and replay defense:** Verify the signature over the raw body, validate timestamp, reject reused event/transaction IDs, process idempotently, and fetch current state before fulfillment when risk warrants it. Coinbase's webhook guide signs timestamp, selected headers, and raw payload, recommends a timestamp window, and exposes explicit success/failure/expiry events ([Coinbase Checkout webhooks](https://docs.cdp.coinbase.com/coinbase-business/checkout-apis/webhooks)). OWASP requires HMAC/authenticity validation and idempotent one-time order processing ([OWASP payment gateway integration](https://cheatsheetseries.owasp.org/cheatsheets/Third_Party_Payment_Gateway_Integration_Cheat_Sheet.html)).
5. **Browser isolation:** Start with a restrictive CSP: deny by default, allow only required script/connect/image/style sources, use nonces/hashes, set `object-src 'none'`, `base-uri 'none'`, and `frame-ancestors 'none'` unless an explicitly supported merchant-embed mode has a strict allowlist. OWASP recommends a strict nonce/hash CSP and documents `frame-ancestors` for clickjacking defense ([OWASP CSP](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html)).
6. **Minimal third-party code:** Exclude advertising pixels, session replay, chat widgets, tag managers, and unnecessary analytics from the payment origin. They expand data leakage and script-supply-chain risk. If telemetry is essential, use first-party, no-PII events and an explicit CSP allowlist.
7. **Abuse and availability:** Rate-limit invoice reads, wallet sessions, address assignment, and status streams by layered signals. Cap payloads, validate every opaque ID, hide tenant/internal IDs, and return non-enumerable error responses.
8. **Observability without leakage:** Correlate invoice revision, webhook event ID, and chain transaction internally. Do not log full checkout URLs, secrets, wallet session URIs, or customer PII. Alert on repeated signature failures, state conflicts, late/under/overpayments, and confirmation regressions.
9. **Test/live isolation:** Use different origins, keys, webhook subscriptions, databases, visual treatment, and chain configuration. Coinbase's sandbox guidance explicitly recommends isolated configuration and separate sandbox webhook subscriptions ([Coinbase Checkout sandbox](https://docs.cdp.coinbase.com/coinbase-business/checkout-apis/sandbox)).

## Accessibility and resilience

Target WCAG 2.2 AA and test with keyboard, screen reader, 200% zoom, reduced motion, high contrast, slow 3G, offline/reconnect, and mobile wallet return.

- Buttons and wallet rows should be at least 44×44 CSS px where practical; WCAG's enhanced target-size guidance uses 44×44, while AA minimum is 24×24 or sufficient spacing ([W3C target size](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced)).
- Keep visible keyboard focus, logical focus order, semantic headings/controls, text labels on icon buttons, and sufficient contrast. Do not convey network, warning, or success by color alone.
- Give every QR an accessible name and always provide its full action as text/buttons. A QR can never be the only way to pay.
- Announce meaningful state changes such as “payment detected” or “2 of 3 confirmations” through a restrained `role="status"`/`aria-live` region; W3C requires visible status updates to be programmatically exposed without moving focus ([W3C status messages](https://www.w3.org/WAI/WCAG21/Understanding/status-messages)). Do not announce a per-second countdown to screen readers.
- Because invoice/rate expiry is financially essential, show the absolute expiry and a visible countdown. At expiry, preserve context and offer a simple new-link/requote path rather than abruptly redirecting. WCAG's timing guidance generally requires limits to be adjustable unless the limit is essential to the activity ([W3C timing adjustable](https://www.w3.org/WAI/WCAG22/Understanding/timing-adjustable.html)).
- Persist the status view across wallet round trips and page reloads. If realtime disconnects, say so and fall back to bounded polling; WalletConnect recommends displaying lost/re-established connection states ([WalletConnect best practices](https://docs.walletconnect.network/wallet-sdk/best-practices)).

## Payminto-specific findings

The current public route in [`frontend/app/(public)/pay/[referenceId]/checkout-view.tsx`](../frontend/app/(public)/pay/%5BreferenceId%5D/checkout-view.tsx) already has useful bones: a two-panel branded layout, combined currency/network rows, a mobile-responsive shape, copy feedback, a status rail, and realtime subscription in [`page.tsx`](../frontend/app/(public)/pay/%5BreferenceId%5D/page.tsx). The backend also advances `FILLED` only from confirmed deposits in the current repository, so the UI's successful terminal treatment of that backend state is defensible.

The highest-priority gaps are:

1. **Do not compute the obligation in the browser.** The current component uses JavaScript `Number`, formats volatile assets to a display precision, and treats several stablecoin symbols as exactly `$1`. Render the persisted server quote/atomic obligation instead.
2. **Replace the raw-address QR.** Encode a chain-aware, asset-aware, amount-bearing payment request or wallet proposal, and keep explicit copy fallbacks.
3. **Add wallet-first mobile UX.** There is no “Open in wallet,” compatible wallet list, WalletConnect origin verification, or reliable wallet-return state.
4. **Expose real detection evidence.** Show amount received, tx hash, confirmations `n/N`, and reconnect/delay state. “Spotted” should be backed by an observed transaction, not inferred from `PARTIALLY_FILLED` alone.
5. **Design exception states.** `OVER_FILLED`, wrong asset/network, late payment, invalid transaction, and refunds need explicit terminal/reconciliation views. The current terminal list does not include `OVER_FILLED`.
6. **Fix expiry behavior.** The local countdown can reach zero while the parent payment UI remains payable until another server update/re-render. Server expiry is authoritative; the UI should disable payment actions and requote/reload status at the boundary.
7. **Replace generic timing claims.** “Payments are almost instant and should reflect in 1–15 mins” is not valid across every supported network and confirmation policy. Return chain-specific confirmation progress/expectation from the server and phrase it as an estimate.
8. **Copy both amount and address.** The current page copies only the address, increasing wrong-amount risk.
9. **Separate the runtime/origin.** The current checkout is a public route inside the dashboard Next.js application. Extract it into its own app/deployment and security headers while reusing design tokens and shared typed contracts.

There is also a second, stricter checkout component under [`frontend/features/payment-open/checkout.tsx`](../frontend/features/payment-open/checkout.tsx) with server-returned atomic units and stronger contract validation. Its safe integer/identity ideas should inform the shared checkout contract, but its developer-facing “atomic-unit obligation” presentation should live under advanced details, not be the payer's primary amount.

## Delivery order

1. Define and test the canonical invoice/exception state machine, integer quote contract, and server-authoritative public checkout view.
2. Extract a dedicated checkout app/origin with no dashboard auth dependency, strict headers, no third-party scripts, and opaque single-use URLs.
3. Build the responsive method → review → wallet/send → confirming → receipt flow, including expiry/under/over/late/error states.
4. Add standards-based payment URIs, desktop QR, mobile wallet deep links, WalletConnect verification, and manual copy fallbacks.
5. Add signed/idempotent webhooks, monotonic state revisions, realtime plus polling fallback, and merchant fulfillment tests.
6. Complete accessibility, slow-network, wallet-return, chain-reorg, webhook-replay, expired-race, and test/live-isolation testing before enabling real-value payments.

## Launch acceptance checklist

- [ ] Checkout is a separate deployable app on its own HTTPS origin.
- [ ] No admin/merchant secret, dashboard token, or private key appears in the JS bundle, browser storage, network response, URL, or logs.
- [ ] The backend owns amount, integer rounding, chain/token identity, expiry, address assignment, and allowed transitions.
- [ ] Every invoice is single-use and every state-changing request is atomic/idempotent.
- [ ] QR/deep link carries the intended chain, asset, recipient, and amount where the wallet standard permits.
- [ ] Desktop QR, mobile wallet opening, return-to-checkout, and manual copy flows work across supported wallets.
- [ ] Detected, confirming, completed, expired, cancelled, underpaid, overpaid, late, invalid, and refund states have tested UI.
- [ ] Browser redirect never fulfills an order; signed server events/current canonical state do.
- [ ] Webhook signature, raw-body verification, timestamp/replay window, duplicate delivery, out-of-order delivery, and retry behavior are tested.
- [ ] CSP, frame policy, CORS, cache/referrer policy, rate limits, and test/live separation are verified in production-like deployment.
- [ ] Keyboard, screen reader, zoom/reflow, contrast, target size, reduced motion, offline/reconnect, and slow-network tests pass.

