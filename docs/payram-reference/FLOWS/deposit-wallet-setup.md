# Deposit wallet setup

PayRam separates **wallets** (private-key holders) from **deposit addresses**
(per-customer or per-payment receive addresses). The deposit-wallet flow is
how a merchant tells PayRam "use _this_ hot wallet on _this_ chain to issue
deposit addresses for incoming payments".

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/manageWallet` (overview) | [manageWallet.md](../SCREENS/manageWallet.md) |
| 2 | `/manageWallet/wallets` (wallet list) | [manageWallet-wallets.md](../SCREENS/manageWallet-wallets.md) |
| 3 | `/manageWallet/wallets/add` | [manageWallet-wallets-add.md](../SCREENS/manageWallet-wallets-add.md) |
| 4 | `/manageWallet/wallets/cold` | [manageWallet-wallets-cold.md](../SCREENS/manageWallet-wallets-cold.md) |
| 5 | `/manageWallet/wallets/details/{id}` | [manageWallet-wallets-details-id.md](../SCREENS/manageWallet-wallets-details-id.md) |
| 6 | `/manageWallet/deposit-wallet` | [manageWallet-deposit-wallet.md](../SCREENS/manageWallet-deposit-wallet.md) |
| 7 | `/manageWallet/hot-wallet` and `/manageWallet/hot-wallet/{id}` | [manageWallet-hot-wallet.md](../SCREENS/manageWallet-hot-wallet.md) |

## Sequence

1. From `/manageWallet`, the merchant clicks **"Add Wallet"** → routes to
   `/manageWallet/wallets/add`.
2. The add-wallet page is a **3-step wizard**:
   - **Step 1 — Wallet type:** radio cards for `Hot wallet (PayRam-managed)`,
     `Cold wallet (offline)`, `Smart-contract wallet (deploy now)`. Each
     option shows a one-line trade-off summary.
   - **Step 2 — Network selector:** the same chain grid documented in
     [`COMPONENT_LIBRARY.md`](../COMPONENT_LIBRARY.md) §5 (BTC, ETH, BASE,
     POLYGON, TRX). Choosing a chain reveals chain-specific fields.
   - **Step 3 — Secret material:** for hot wallets, generate-or-import a
     mnemonic; for SCW, connect a browser wallet via RainbowKit and choose
     the deployer EOA. Mnemonics surface a **reveal-once** screen with a
     copy-to-clipboard button and a "I've saved it" confirm checkbox.
3. Submit → `POST /api/v1/wallets` (mutating). The new wallet appears in
   `/manageWallet/wallets`. We observed the GET counterparts during the
   crawl: `GET /api/v1/wallets`, `GET /api/v1/wallets/{id}`.
4. The merchant then opens **`/manageWallet/deposit-wallet`** and selects the
   newly-created wallet from a per-network sub-tab strip (BTC / ETH / BASE /
   POLYGON / TRX). They configure:
   - **Sweep frequency** — every N confirmations, every M minutes, or manual
   - **Min sweep amount** — below this threshold deposits accumulate
   - **Destination cold wallet** — picked from `/manageWallet/wallets/cold`
   - **Status indicator** — green dot if the chain listener is running,
     red dot if the listener crashed (we saw the listeners running in the
     container `ps auxf` output, one per chain)
5. Save → `PUT /api/v1/external-platform/all/deposit-wallet/{family}` (mutating).
6. PayRam then issues per-payment addresses by calling the SmartSweep
   factory contract or by deriving HD addresses depending on wallet type.

## Companion: Sweep contract

For ETH/BASE/POLYGON/TRX, the merchant must **deploy a sweep contract** before
hot-wallet sweeps will work. That's a separate flow at
`/manageWallet/sweepContract` → `/manageWallet/sweepContract/deploy`. The
deploy page is a wizard that:

1. Loads the factory ABI via
   `GET /api/v1/blockchain-contract/blockchain/{chain}/contract/factory_contract`
2. Connects the merchant's browser wallet via RainbowKit
3. Calls the factory's `createSweepApproval()` and waits for the receipt
4. POSTs the deployed address to the backend (mutating)

We observed the read side:
- `GET /api/v1/blockchain-contract/contract/factory_contract/address-contract/sweep_approval`
- `GET /api/v1/contract-address/blockchain/ETH/contract/sweep_approval` → 404
  in our snapshot (no contract deployed yet on this chain).

## Companion: Gas-fee wallet

ETH/BASE/POLYGON sweeps consume gas. The merchant funds a **gas-fee wallet**
at `/manageWallet/gasFeeWallet` (and `/add`, `/edit`). It's a separate hot
wallet whose only job is to pay for sweep transactions and approval txs. A
**low-balance warning** banner appears when the gas-fee wallet drops below
the configured threshold.

## Gaps vs. Payminto clone

- Payminto's `payminto/frontend/src/app/(dashboard)/manageWallet/deposit-wallet/page.tsx`
  is currently a single page with no wizard, no per-network sub-tabs, and
  no sweep frequency selector.
- `wallets/add` exists but is single-step.
- `wallets/cold` and `wallets/details/[id]` are **missing entirely**.
- The entire `gasFeeWallet` and `sweepContract` subtrees are missing.
- The chain status indicators are missing (we'd need to expose chain-listener
  health from the backend).
