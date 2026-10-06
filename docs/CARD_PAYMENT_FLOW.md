# PayRam Card-to-Crypto Payment Flow — Deep Analysis

**Sources:** PayRam docs (docs.payram.com), frontend code reverse-engineering, API analysis
**Date:** April 6, 2026

---

## Why This Flow Exists

PayRam is **non-custodial** — merchants never hand over funds to PayRam. The Cards flow must solve a problem: **how does a customer paying with a credit card end up sending crypto to a merchant's self-hosted deposit address?**

The answer: a **customer wallet** acts as the bridge.

```
Fiat (Card)  →  Onramp Provider  →  Customer's PayRam Wallet (on Base)  →  Merchant Deposit Address
```

This is NOT an assumption. PayRam docs state:
> "Crypto deposits directly into customer's self-custody wallet. 
>  Customer completes transaction with merchant using deposited funds."

---

## The Two Payment Channels

PayRam has **3 distinct channel types** in the frontend code:

| Channel | channelType | What It Does |
|---------|-------------|-------------|
| **Crypto** | `blockchain` | Customer sends crypto directly from their own wallet (MetaMask, etc.) |
| **Cards** | `app` | Customer creates a PayRam Wallet → funds via card → wallet sends to merchant |
| **TransFi** | `onramp` | Direct iframe embed of TransFi — fiat onramp sends crypto directly to merchant address |

### Key Difference: "Cards" (app) vs "TransFi" (onramp)

```
"Cards" (channelType: app):
  Card → Onramp → Customer PayRam Wallet → Merchant Deposit Address
  (Two-step: fund wallet, then pay)
  Uses: wallet.payram.com + x.payram.com

"TransFi" (channelType: onramp):
  Card → TransFi → Merchant Deposit Address DIRECTLY
  (One-step: TransFi sends crypto straight to merchant)
  Uses: iframe to buy.transfi.com
```

---

## Complete "Cards" Flow (channelType: app)

### Step 1: Payment Page Loads

Customer opens payment link:
```
http://merchant.com/payments?reference_id=xxx&host=http://merchant.com:8080
```

Frontend calls these APIs:
```
GET /api/v1/payment/reference/{ref}              → Payment details (amount, status)
GET /api/v1/payment-channels/reference/{ref}     → Available channels [crypto, app, onramp]
GET /api/v1/blockchain-currency/reference/{ref}  → Available currencies per chain
GET /api/v1/external-platform/reference/{ref}    → Merchant info (name, logo)
GET /api/v1/ticker                               → Price tickers
```

### Step 2: Cards Enable Check

Frontend code (verified from source):
```javascript
// Cards is ONLY enabled if USDC on BASE chain exists
const cardsEnabled = useMemo(() => {
    const usdc = currencies.find(c => c.code === "USDC");
    return !!usdc && !!usdc.blockchains?.some(b => b.code === "BASE");
}, [currencies]);

// disabled = channelType === "app" && !cardsEnabled
```

**Why BASE only?** PayRam docs confirm: *"Card payments and other fiat payment options are currently supported on the Base blockchain only."* Base has low gas fees, making the two-hop flow (onramp → wallet → merchant) economically viable.

### Step 3: Customer Selects "Cards"

Frontend automatically:
1. Sets token to **USDC**
2. Sets blockchain to **BASE**
3. Calls `POST /api/v1/deposit-address/{ref}` with `{blockchain_code: "BASE"}` to assign a merchant deposit address

### Step 4: "Continue on Web" / QR Code

**Desktop — "Continue on Web" button:**
Opens a popup (360×750) to `wallet.payram.com` with these URL params:

```
https://wallet.payram.com/?
  recipientAddress=0x...     ← Merchant's deposit address on Base
  amount=1000000             ← USDC amount in smallest unit (6 decimals)
  chainID=8453               ← Base Mainnet (or 84532 for testnet)
  tokenAddress=0x833589...   ← USDC contract on Base
  fiatAmount=100             ← USD amount for display
  referenceID=abc-123        ← Payment reference for matching
  host=http://merchant:8080  ← Merchant's API URL for callbacks
```

**Mobile — QR Code / Deep Link:**
```
https://x.payram.com/deeplink/start?ref=web&ethereum_uri={encoded_payment_data}

ethereum_uri contains:
  blockchainFamily, recipientAddress, amount, chainId,
  tokenAddress, referenceId, host, walletPrecision
```

### Step 5: Inside PayRam Wallet (wallet.payram.com)

This is PayRam's hosted web wallet service. The flow:

```
5a. CREATE WALLET (first-time customers)
    └── Customer enters email address
    └── OTP sent to email → verified
    └── Self-custodial wallet generated on Base chain
    └── Customer owns the private key (non-custodial)
    └── Takes "a few seconds" per docs

5b. KYC VERIFICATION (one-time)
    └── Basic KYC required on first purchase
    └── Handled by the onramp provider
    └── Subsequent purchases skip this

5c. ADD FUNDS (card onramp)
    └── Onramp widget loads inside the wallet
    └── Customer enters card details (Visa/MC/Amex)
    └── Also supports: Apple Pay, Google Pay, bank transfers (ACH, SEPA), RevolutPay
    └── 175+ payment methods, 190+ countries
    └── Onramp provider converts fiat → USDC
    └── USDC deposited into customer's PayRam Wallet on Base

5d. SEND PAYMENT
    └── Wallet already knows: recipientAddress, amount, tokenAddress (from URL params)
    └── Customer confirms sending USDC to merchant deposit address
    └── On-chain transaction: Customer Wallet → Merchant Deposit Address (Base, USDC)
    └── If merchant sponsors gas: gas fee is covered by merchant (optional)
```

### Step 6: Payment Detection (Back on Merchant's VPS)

```
6a. Base ERC20 Listener (running in Docker container)
    └── Monitors Base chain for incoming USDC transfers
    └── Detects deposit to merchant's assigned address

6b. Deposit Processing
    └── Creates record in `deposits` table
    └── Links to `payment_requests` via deposit address match
    └── Updates payment status: open → confirming → confirmed

6c. Webhook Notification
    └── Fires webhook to merchant's backend
    └── Payload includes: reference_id, amount, tx_hash, status
    └── Retry logic: up to N attempts with exponential backoff

6d. Frontend Polling
    └── Payment page polls every 5 seconds: GET /api/v1/payment/reference/{ref}
    └── Shows: "Checking for new payments" → "Transaction spotted!" → "Payment Successful"
```

---

## Complete "TransFi" Flow (channelType: onramp) — SIMPLER

This is a different, more direct flow:

### How It Works
```
1. Customer selects "TransFi Payments" on payment page
2. Frontend gets merchant's deposit address (same assignDepositAddress API)
3. TransFi iframe loads DIRECTLY in the payment page:

   <iframe src="https://buy.transfi.com/?view=buy
     &apiKey=l8ooOivL2Hx5qZbu          ← TransFi API key (from payment_channels config)
     &redirectUrl={current_page_url}     ← Where to redirect after payment
     &walletAddress=0x...                ← MERCHANT's deposit address directly
     &cryptoTicker=USDC                  ← Token to buy
     &cryptoNetwork=BASE                 ← Network
     &partnerContext={referenceId:xxx}   ← For matching payment
   " />

4. Customer pays with card INSIDE the iframe
5. TransFi converts fiat → USDC
6. TransFi sends USDC DIRECTLY to merchant's deposit address
7. No intermediate wallet needed
8. Base listener detects deposit → payment confirmed
```

### Why This Is Simpler
- **No wallet creation needed** — TransFi sends directly to merchant
- **One step** — customer just pays in the iframe
- **TransFi handles everything**: KYC, card processing, crypto purchase, delivery

### Why PayRam Has BOTH
- **TransFi (onramp)**: Simpler but depends entirely on TransFi. If TransFi is down or unsupported in a region, no fallback.
- **Cards (app)**: More steps but gives the customer a **reusable wallet**. Customer can fund once and pay multiple merchants. Also supports multiple onramp providers through the wallet.

---

## Onramp Provider: TransFi

**Verified from code:**
```javascript
// Default TransFi API key hardcoded in PayRam frontend:
apiKey: configuration.apiKey || "l8ooOivL2Hx5qZbu"

// TransFi buy widget URL:
https://buy.transfi.com/?view=buy
```

**TransFi** (transfi.com) is the confirmed onramp provider. They handle:
- Card processing (Visa, Mastercard, Amex)
- KYC verification
- Fiat → crypto conversion
- Direct crypto delivery to specified wallet address

PayRam docs don't name the provider publicly, calling them *"regulated, third-party fiat-to-crypto providers"* — but the code reveals it's **TransFi**.

---

## Gas Fee Sponsorship

From PayRam docs: *"Merchants can sponsor gas fees for customers, reducing friction and improving conversion."*

Database: `payments_apps` table has:
```sql
sponsorship_percentage  NUMERIC(5,4)   -- % of gas to sponsor (0-100%)
sponsorship_cut_off     NUMERIC(38,18) -- Max sponsorship amount
```

When enabled:
- Merchant pre-funds gas on Base chain
- Customer's wallet-to-merchant transaction gas is paid by merchant
- Reduces customer friction (they don't need Base ETH for gas)

---

## Architecture Diagram (Corrected)

```
┌────────────────────────────────────────────────────────────────────────┐
│                    MERCHANT'S VPS (Self-Hosted)                        │
│                                                                        │
│  Payment Page ──→ Go API ──→ PostgreSQL                                │
│  (Next.js)        :8080      payment_requests, deposits,               │
│  :80/443                     blockchain_currencies                     │
│       │                                                                │
│       │           Base ERC20 Listener ──→ Detects USDC deposit         │
│       │           (background worker)     → Confirms payment           │
│       │                                   → Fires webhook              │
└───────┼────────────────────────────────────────────────────────────────┘
        │
        │ Customer selects payment method
        │
        ├──── "Crypto" ──────→ Customer sends from MetaMask/any wallet
        │                      directly to merchant deposit address
        │
        │                      ┌──────────────────────────────────┐
        ├──── "Cards" ────────→│ wallet.payram.com (Hosted)       │
        │     (channelType:    │                                  │
        │      app)            │ 1. Create wallet (email + OTP)   │
        │                      │ 2. Load onramp provider          │
        │                      │ 3. Customer pays with card       │
        │                      │ 4. USDC lands in customer wallet │
        │                      │ 5. Wallet sends USDC to merchant │
        │                      │    deposit address on Base       │
        │                      └──────────────────────────────────┘
        │                            │
        │                            ▼
        │                      ┌──────────────────────────────────┐
        │                      │ Onramp Provider (unnamed)        │
        │                      │ Card → USDC on Base              │
        │                      └──────────────────────────────────┘
        │
        │                      ┌──────────────────────────────────┐
        └──── "TransFi" ─────→│ buy.transfi.com (iframe)         │
              (channelType:    │                                  │
               onramp)         │ 1. Customer pays with card       │
                               │ 2. TransFi converts fiat → USDC │
                               │ 3. USDC sent DIRECTLY to        │
                               │    merchant deposit address      │
                               │    (no intermediate wallet)      │
                               └──────────────────────────────────┘
```

---

## Sequence Diagram (Cards / app channel)

```
Customer        Payment Page      wallet.payram.com    Onramp Provider    Base Chain       Merchant API
   │                │                    │                   │                │                 │
   │─ Open link ───▶│                    │                   │                │                 │
   │                │─ Load APIs ───────▶│                   │                │                 │
   │                │◀─ Channels + ──────│                   │                │                 │
   │                │   currencies       │                   │                │                 │
   │                │                    │                   │                │                 │
   │─ Click Cards ─▶│                    │                   │                │                 │
   │                │─ Assign deposit ──▶│                   │                │                 │
   │                │  address (POST)    │                   │                │                 │
   │                │◀─ 0xMerchant... ───│                   │                │                 │
   │                │                    │                   │                │                 │
   │─ Continue ────▶│─ Open popup ──────▶│                   │                │                 │
   │   on Web       │  (wallet.payram    │                   │                │                 │
   │                │   .com/?params)    │                   │                │                 │
   │                │                    │                   │                │                 │
   │                │              ┌─────┴─────┐             │                │                 │
   │                │              │ Enter email│             │                │                 │
   │                │              │ Verify OTP │             │                │                 │
   │                │              │ Wallet     │             │                │                 │
   │                │              │ created    │             │                │                 │
   │                │              └─────┬─────┘             │                │                 │
   │                │                    │                    │                │                 │
   │                │                    │─ Load onramp ────▶│                │                 │
   │                │                    │   widget          │                │                 │
   │                │                    │                   │                │                 │
   │                │              ┌─────┴─────┐             │                │                 │
   │                │              │ Enter card │             │                │                 │
   │                │              │ KYC (1st   │             │                │                 │
   │                │              │ time only) │             │                │                 │
   │                │              └─────┬─────┘             │                │                 │
   │                │                    │                    │                │                 │
   │                │                    │              Fiat charged          │                 │
   │                │                    │              USDC purchased        │                 │
   │                │                    │◀─ USDC to wallet ─┤                │                 │
   │                │                    │   on Base          │                │                 │
   │                │                    │                    │                │                 │
   │                │              ┌─────┴─────┐             │                │                 │
   │                │              │ Confirm    │             │                │                 │
   │                │              │ send USDC  │─────────────┼──── USDC ────▶│                 │
   │                │              │ to merchant│             │    transfer   │                 │
   │                │              └────────────┘             │   on Base     │                 │
   │                │                                         │                │                 │
   │                │                                         │                │─ Deposit ──────▶│
   │                │                                         │                │  detected       │
   │                │                                         │                │                 │─ Confirm
   │                │◀──── Poll: status = confirmed ──────────┼────────────────┼─────────────────│
   │◀─ Payment ────│                                         │                │                 │─ Webhook
   │   Success!     │                                         │                │                 │
```

---

---

## CRITICAL: How Does the Merchant Server Know Payment Happened?

### The Communication Chain (Verified from Code)

There is **NO direct callback** from `wallet.payram.com` to the merchant server. There is **NO postMessage** between the popup and the payment page. There is **NO webhook from the wallet**.

The flow is:

```
1. wallet.payram.com sends USDC on-chain to merchant deposit address
   (it knows the address from the URL param: recipientAddress=0x...)

2. Merchant's Base ERC20 Listener (background worker in Docker) 
   monitors the Base blockchain for incoming transactions

3. When USDC arrives at the merchant's deposit address:
   - Listener creates a record in `deposits` table
   - Matches deposit address to `payment_requests` 
   - Updates payment_requests.status: "OPEN" → "FILLED"

4. Payment page polls every 5 seconds:
   GET /api/v1/payment/reference/{ref}
   → When paymentState changes from "OPEN" to "FILLED", page updates

5. Webhook fires to merchant's backend (if configured)
```

### Polling Code (Verified)

```javascript
// Y = polling function, called every 5 seconds
Y = async () => {
    let e = await A(t, r);  // GET /api/v1/payment/reference/{ref}
    e && R(e);              // Update local state with new payment data
};

// Poll while payment is OPEN or confirming, stop when done
const shouldPoll = paymentState === "OPEN" || isConfirming;
useInterval(Y, shouldPoll ? 5000 : null);

// Payment states:
// OPEN → FILLED / PARTIALLY_FILLED / OVER_FILLED / CANCELLED
```

### WebSocket (Dashboard Only)

PayRam has a WebSocket (`ws://localhost:8080`) but it's used for the **dashboard** (authenticated), not the public payment page. The `web_socket_tokens` table stores tokens for dashboard sessions.

The public payment page uses **pure HTTP polling** — no WebSocket.

### The `host` Parameter — Why It Matters

The wallet URL includes `host=http://merchant:8080`. This tells `wallet.payram.com`:
- Where the merchant's API is located
- So the wallet can **query payment details** (amount, currency, reference)
- And potentially **verify the deposit address** belongs to this payment
- The wallet does NOT use `host` to send a callback — the blockchain transaction IS the notification

### Summary: No Magic, Just Blockchain

```
wallet.payram.com                    Merchant VPS
      │                                   │
      │─── USDC transfer on Base ────────▶│ (on-chain)
      │    (to deposit address from URL)  │
      │                                   │
      │    No HTTP callback               │─ Base listener detects tx
      │    No WebSocket message           │─ Creates deposit record
      │    No postMessage                 │─ Updates payment status
      │                                   │─ Fires merchant webhook
      │                                   │
      │         Payment page polls ──────▶│
      │         every 5 seconds           │─ Returns updated status
      │◀──────── "FILLED" ────────────────│
```

---

---

## TransFi (onramp) vs Cards (app) — Side by Side

### TransFi Flow (channelType: "onramp") — SIMPLER, NO WALLET

```
┌─────────────────────────────────────────────────────────────────────┐
│                      PAYMENT PAGE (merchant VPS)                    │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    TransFi iframe                             │   │
│  │                                                              │   │
│  │  URL: https://buy.transfi.com/?view=buy                      │   │
│  │    &apiKey=l8ooOivL2Hx5qZbu                                  │   │
│  │    &redirectUrl=http://payment-page/current-url               │   │
│  │    &walletAddress=0xMerchantDepositAddress  ← DIRECT!         │   │
│  │    &cryptoTicker=USDC                                         │   │
│  │    &cryptoNetwork=BASE                                        │   │
│  │    &partnerContext={"referenceId":"abc-123"}                   │   │
│  │                                                              │   │
│  │  1. Customer enters card details                              │   │
│  │  2. TransFi handles KYC (first time)                          │   │
│  │  3. TransFi charges card                                      │   │
│  │  4. TransFi buys USDC on Base                                 │   │
│  │  5. TransFi sends USDC → 0xMerchantDeposit (on-chain)        │   │
│  │  6. TransFi redirects iframe → redirectUrl                    │   │
│  │                                                              │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Payment page polls: GET /api/v1/payment/reference/{ref}            │
│  Every 5 seconds until paymentState changes to FILLED               │
│                                                                     │
│  Meanwhile: Base ERC20 Listener detects USDC at deposit address     │
│  → Creates deposit record → Updates payment_request → Webhook       │
└─────────────────────────────────────────────────────────────────────┘
```

**Key points:**
- **No intermediate wallet** — TransFi sends crypto DIRECTLY to merchant deposit address
- **No popup** — iframe embedded in the payment page itself
- **walletAddress** = merchant's deposit address (assigned via `POST /api/v1/deposit-address/{ref}`)
- **partnerContext** = JSON with referenceId for TransFi's records (they can webhook back)
- **redirectUrl** = payment page URL — TransFi redirects the iframe back when done
- **Notification**: purely on-chain detection by blockchain listener (same as regular crypto)

### Cards Flow (channelType: "app") — WITH WALLET + ERC-4337

```
┌─────────────────┐     ┌──────────────────────────────────────────┐
│  PAYMENT PAGE    │     │  wallet.payram.com (POPUP)                │
│  (merchant VPS)  │     │                                          │
│                  │     │  Params received in URL:                  │
│  Click "Cards"   │────▶│  - recipientAddress = 0xMerchantDeposit  │
│                  │     │  - amount = USDC amount                   │
│  Popup opens     │     │  - chainID = 8453 (Base)                  │
│  360×750 window  │     │  - tokenAddress = USDC contract           │
│                  │     │  - fiatAmount = $100                       │
│                  │     │  - referenceID = abc-123                   │
│                  │     │  - host = http://merchant:8080             │
│                  │     │                                          │
│                  │     │  Step 1: Create Smart Account             │
│                  │     │  ├── Email + OTP verification             │
│                  │     │  └── ERC-4337 account via Pimlico         │
│                  │     │      EntryPoint: 0x4337...08              │
│                  │     │                                          │
│                  │     │  Step 2: Fund Wallet                      │
│                  │     │  ├── Onramp provider widget loads         │
│                  │     │  ├── Card payment processed               │
│                  │     │  └── USDC deposited to smart account      │
│                  │     │                                          │
│                  │     │  Step 3: Send Payment                     │
│                  │     │  ├── Wallet knows recipientAddress        │
│                  │     │  ├── UserOp sent through ERC-4337         │
│                  │     │  │   EntryPoint contract                  │
│                  │     │  └── USDC → 0xMerchantDeposit on Base     │
│                  │     │                                          │
│  Polls every 5s  │     │  (No callback to payment page)            │
│  GET /payment/   │     └──────────────────────────────────────────┘
│  reference/{ref} │
│                  │     ┌──────────────────────────────────────────┐
│  Base Listener   │     │  MERCHANT BACKEND (Go API)               │
│  detects USDC    │────▶│                                          │
│                  │     │  1. ERC20 listener sees USDC deposit      │
│  paymentState    │     │  2. Checks: did tx go through ERC-4337   │
│  → FILLED        │     │     EntryPoint? (0x4337...08)             │
│                  │     │  3. YES → "Payments app transaction       │
│  Shows success   │     │     detected (entrypoint: 0x4337...08)"   │
│                  │     │  4. Calls PAYMENTS_APP_SERVER_URL          │
│                  │     │     (x.payram.com) to:                     │
│                  │     │     - FetchTransactionSponsorship          │
│                  │     │       (was gas sponsored?)                 │
│                  │     │     - Get onramp payment metadata          │
│                  │     │  5. Creates deposit + updates payment      │
│                  │     │  6. Fires merchant webhook                 │
│                  │     └──────────────────────────────────────────┘
```

**Key discoveries:**

1. **ERC-4337 Account Abstraction** — The PayRam Wallet uses smart accounts via the Pimlico EntryPoint (`0x4337084d9e255ff0702461cf8895ce9e3b5ff108`). This is how:
   - Customers can create wallets with just email (no seed phrase)
   - Gas sponsorship works (merchant pays gas via paymaster)
   - The self-hosted server can identify "this deposit came from PayRam Wallet" vs "regular wallet"

2. **`host` parameter** — Passed to `wallet.payram.com` so the wallet can:
   - Query merchant's API for payment details
   - Verify the deposit address is valid for this payment
   - NOT used for callbacks — the blockchain transaction IS the notification

3. **`PAYMENTS_APP_SERVER_URL`** — The self-hosted server calls `x.payram.com` AFTER detecting an entrypoint transaction to:
   - `FetchTransactionSponsorship(url, tx_hash)` — check if gas fee was sponsored
   - Get metadata for the OnramperPayments dashboard page

4. **No cross-window communication** — No postMessage, no WebSocket on public page. Pure polling (every 5 seconds) + blockchain listener.

---

### Why Two Separate Flows?

| Aspect | TransFi (onramp) | Cards (app) |
|--------|-------------------|-------------|
| **Steps for customer** | 1 step (pay in iframe) | 3 steps (create wallet, fund, send) |
| **Wallet needed?** | NO — sends directly to merchant | YES — ERC-4337 smart account |
| **Provider** | TransFi only | Any onramp in the wallet |
| **Gas fees** | Paid by TransFi (included) | Can be sponsored by merchant |
| **Reusable?** | No — one-time payment | Yes — wallet persists |
| **KYC** | In TransFi iframe | In wallet onramp |
| **Where crypto goes** | Directly to merchant address | Wallet → merchant address |
| **Server notification** | On-chain detection only | On-chain + entrypoint check + x.payram.com |

---

---

## Why the Wallet Exists — Compliance & Regulatory Analysis

### The Core Compliance Problem

When a customer pays with a **credit card** and a merchant receives **crypto**, three regulated activities happen:

```
1. FIAT COLLECTION    — Charging a credit card (payment processing)
2. FIAT-TO-CRYPTO     — Converting fiat into cryptocurrency (money transmission)
3. CRYPTO DELIVERY    — Sending crypto to a destination address (money transmission)
```

Each of these requires **licensing**. PayRam is self-hosted and non-custodial — they don't want to be classified as a **money transmitter** or **payment processor**. The wallet architecture solves this.

### How PayRam Avoids Regulatory Classification

```
┌─────────────────────────────────────────────────────────────────┐
│                    WHO DOES WHAT                                 │
│                                                                 │
│  PayRam (software provider):                                    │
│  ├── Provides self-hosted open-source software                  │
│  ├── Does NOT hold funds                                        │
│  ├── Does NOT process payments                                  │
│  ├── Does NOT perform KYC                                       │
│  ├── Does NOT convert fiat to crypto                            │
│  └── NOT a money transmitter — just software                    │
│                                                                 │
│  TransFi / Onramp Provider (regulated entity):                  │
│  ├── Licensed for fiat-to-crypto conversion                     │
│  ├── Handles KYC/AML verification                               │
│  ├── Processes card payments                                    │
│  ├── Converts fiat → crypto                                     │
│  └── Delivers crypto to customer's OWN wallet                   │
│       (NOT directly to merchant — this is critical)             │
│                                                                 │
│  Customer (self-custodial):                                     │
│  ├── Owns their wallet (ERC-4337 smart account)                 │
│  ├── Receives crypto into THEIR wallet                          │
│  ├── VOLUNTARILY sends crypto to merchant                       │
│  └── This is a peer-to-peer transfer (not regulated)            │
│                                                                 │
│  Merchant (self-hosted):                                        │
│  ├── Runs PayRam on their own server                            │
│  ├── Receives crypto at their deposit address                   │
│  ├── Implements own KYC/AML as needed per jurisdiction          │
│  └── Full custody of funds                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Why Funds Go Through Customer Wallet (Not Directly to Merchant)

This is the **compliance architecture**:

| Direct to Merchant | Via Customer Wallet |
|---|---|
| TransFi charges card → sends USDC to merchant | TransFi charges card → sends USDC to **customer's wallet** → customer sends to merchant |
| TransFi is sending funds to a **third party** (the merchant) | TransFi is sending funds to the **same person** who paid (the customer) |
| TransFi acts as a **payment processor** for the merchant | TransFi acts as an **onramp** for the customer |
| Requires merchant KYB, partnership agreements | Requires only customer KYC |
| TransFi needs money transmitter license for B2B transfers | TransFi only needs onramp license (customer buys crypto for themselves) |

**The wallet makes it a "customer buys crypto for themselves" transaction, not a "customer pays merchant via TransFi" transaction.**

From PayRam docs:
> "Merchants do not need to complete KYC/KYB to enable this Onramp method."

This is ONLY possible because the funds go through the customer's wallet first. If TransFi sent directly to the merchant, merchant KYB would be required.

### The TransFi (onramp) Exception

The TransFi iframe channel (`channelType: onramp`) DOES send directly to the merchant address:
```
walletAddress = merchant's deposit address
```

This works because:
1. TransFi has its own **partnership agreements** with PayRam
2. TransFi's `partnerContext` param ties the transaction to the PayRam partner account
3. TransFi handles the compliance burden under their own licenses
4. This is likely a **B2B white-label arrangement** where TransFi takes on the regulatory risk

### KYC Requirements Summary

| Who | KYC Required? | When | By Whom |
|-----|--------------|------|---------|
| **Merchant** | NO (from PayRam) | Never | Merchant implements own KYC/AML per jurisdiction |
| **Customer (Cards)** | YES — one-time basic KYC | First purchase via wallet | Onramp provider inside wallet |
| **Customer (TransFi)** | YES — one-time KYC | First purchase via iframe | TransFi directly |
| **Customer (Crypto)** | NO | Never | No KYC for direct crypto payments |

From PayRam docs:
> "PayRam does not impose mandatory KYC requirements by default. Users have the flexibility to implement their own KYC/AML workflows in accordance with their jurisdictional regulations."

### Gas Fee Sponsorship — Compliance Aspect

```
payments_apps table:
  sponsorship_percentage  NUMERIC(5,4)   — % of gas fee merchant covers
  sponsorship_cut_off     NUMERIC(38,18) — Max USD amount merchant will sponsor
```

- Merchant can sponsor gas fees so customers don't need ETH for Base transactions
- Works via ERC-4337 **Paymaster** (Pimlico infrastructure)
- The merchant pre-deposits gas funds, and the Paymaster pays gas on behalf of customers
- This reduces friction but creates a small compliance consideration: merchant is subsidizing customer transactions
- Currently limited to Base chain where gas is ~$0.01, making it economically trivial

### For Our Clone — Compliance Checklist

```
1. ONRAMP PROVIDER SELECTION
   ├── Transak (you have staging keys) — regulated in multiple jurisdictions
   ├── TransFi — PayRam's current provider
   ├── MoonPay — widest coverage but expensive
   └── Ramp — good for European markets
   
2. CUSTOMER WALLET (if building Cards/app channel)
   ├── Must be SELF-CUSTODIAL (customer owns keys)
   ├── Must use customer's OWN wallet as intermediate step
   ├── Onramp sends crypto to customer wallet, NOT merchant
   ├── Customer then sends to merchant (peer-to-peer)
   └── This keeps us out of money transmitter classification
   
3. DIRECT ONRAMP (if building TransFi/onramp channel)
   ├── Requires partnership with onramp provider
   ├── Provider sends directly to merchant address
   ├── Provider handles all compliance
   ├── We pass partnerContext/referenceId for tracking
   └── Simpler but we depend on provider relationship
   
4. MERCHANT RESPONSIBILITY
   ├── Self-hosted = merchant controls compliance
   ├── PayRam/our software = just a tool
   ├── Merchants implement their own KYC/AML per jurisdiction
   └── We provide monitoring tools (activity_logs, analytics)
   
5. KYC/AML
   ├── Customer KYC: handled by onramp provider (Transak/TransFi)
   ├── Merchant KYC: NOT required by us (merchant's choice)
   ├── AML monitoring: on-chain transaction records
   └── Compliance dashboard: analytics for merchant's own compliance
```

---

## For Our Clone: What This Means

### Option 1: TransFi Direct (Simplest — like "onramp" channel)
- Embed TransFi/Transak iframe directly in payment page
- Pass merchant deposit address as `walletAddress`
- Onramp sends USDC directly to merchant — NO intermediate wallet needed
- **Pros**: Simple, one step
- **Cons**: Dependent on single provider

### Option 2: Customer Wallet (like "app" channel)
- Build our own web wallet (or use existing like Privy, Dynamic, Thirdweb)
- Customer creates wallet, funds via Transak, then sends to merchant
- **Pros**: Customer gets reusable wallet, supports multiple onramp providers
- **Cons**: More friction (2 steps), need to build/host wallet service

### Recommended for MVP: Option 1 with Transak
You already have Transak staging keys. We can embed their widget directly:
```
https://global-stg.transak.com/?
  apiKey=61fafac2-90b9-4c14-9556-680d652f64f8
  &walletAddress={merchantDepositAddress}
  &cryptoCurrencyCode=USDC
  &network=base
  &defaultFiatAmount=100
  &fiatCurrency=USD
```
No intermediate wallet, no hosted service needed.
