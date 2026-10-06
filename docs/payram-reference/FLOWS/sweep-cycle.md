# Sweep cycle (Funds Consolidation)

PayRam calls its periodic batch-consolidation job a **sweep cycle**. Funds
that customers paid into per-payment deposit addresses are batched up and
swept into a single hot wallet (and optionally forwarded onward to a cold
wallet). The dashboard surface for this is the entire `/sweepIn/*` subtree.

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/sweepIn` | [sweepIn.md](../SCREENS/sweepIn.md) |
| 2 | `/sweepIn/btc` | [sweepIn-btc.md](../SCREENS/sweepIn-btc.md) |
| 3 | `/sweepIn/eth` | [sweepIn-eth.md](../SCREENS/sweepIn-eth.md) |
| 4 | `/sweepIn/usdc` | [sweepIn-usdc.md](../SCREENS/sweepIn-usdc.md) |
| 5 | `/sweepIn/usdt` | [sweepIn-usdt.md](../SCREENS/sweepIn-usdt.md) |

## Sequence (per asset)

1. Merchant lands on `/sweepIn`. The SPA fires:
   - `GET /api/v1/sweeps` — list of sweep cycles, currently empty in our
     snapshot (no cycles run yet)
   - `GET /api/v1/wallets`, `GET /api/v1/blockchains`, `GET /api/v1/blockchain-currency`
2. The page renders **per-asset cards** (BTC, ETH, USDC, USDT) showing:
   - Pending balance (sum of un-swept deposits)
   - Address count (number of deposit addresses with balance > 0)
   - **Cycle countdown timer** (next automated cycle ETA, sourced from
     `--pr-pending` ticker)
   - "Sweep Now" button (lime CTA)
3. Clicking a card → routes to `/sweepIn/{asset}` showing the per-address
   list with checkboxes:
   - Address, balance, age of last deposit, included/excluded toggle
   - **Step 1 — Approve:** for ERC-20s, the addresses must individually call
     `approve()` on the token contract granting the SmartSweep contract
     spending rights. This is the first sub-step of the cycle and consumes
     gas from the gas-fee wallet.
   - **Step 2 — Sweep:** the SmartSweep contract pulls from each approved
     address and forwards into the hot wallet, all in a single batched tx.
   - **Step 3 — (optional) Forward to cold:** if a cold-wallet destination is
     configured for this asset, a final transfer happens to the cold address.
4. Submit → `POST /api/v1/sweeps` (mutating). The job is enqueued and the
   page tails progress via the websocket channel.

## Background workers

PayRam ships **dedicated worker processes** for each step (visible in the
container's `ps auxf` output):

- `payram start-sweep-approval-processor` — issues `approve()` calls
- `payram start-deposit-processor` — credits incoming deposits
- `payram start-broadcast-scw-deposit-wallet-processor` — broadcasts the
  smart-contract-wallet (SCW) deposit-address creation txs
- `payram start-{btc,eth,base,polygon,trx}-listener` — chain block listeners
  feeding the deposit processor
- `payram start-account-processor` — credits the merchant's internal ledger
- `payram start-webhook-processor` — fires merchant webhooks on transitions

## Cycle data model

The sweep cycle row (from `GET /api/v1/sweeps` shape — empty in our crawl
but documented in `research/PAYRAM_TECHNICAL_DOCUMENTATION.md`):

```
{
  id, createdAt, status,                          // queued | approving | sweeping | forwarding | done | failed
  blockchainCode, currencyCode,
  approvalTxHashes: [...], sweepTxHash, forwardTxHash,
  inputAddresses: [{ address, amount, txHash }],
  totalAmount, gasFeeUsed,
  destinationWallet: { id, address }
}
```

## Batch sizes

PayRam exposes two env-driven knobs that the SPA reads at boot:

- `NEXT_PUBLIC_SWEEP_BATCH_SIZE` (default 50) — max addresses per sweep tx
- `NEXT_PUBLIC_APPROVAL_BATCH_SIZE` (default 50) — max approvals per cycle

Confirmed via `GET /api/v1/configuration/default` which returns
`sweepBatchSizeETH`, `sweepApprovalBatchSizeETH`, `sweepBatchSizeTRX`,
`sweepApprovalBatchSizeTRX`.

## Gaps vs. Payminto clone

- `/sweepIn` exists in Payminto as an empty-state placeholder.
- The four per-asset pages (`/sweepIn/btc|eth|usdc|usdt`) are **missing**.
- We have no cycle timer, no approve/sweep/forward 3-step UI, and no
  websocket plumbing for live progress.
- The settings → batch-size knobs are missing.
