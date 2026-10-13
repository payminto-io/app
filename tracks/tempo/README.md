# Tempo payment rail

Payminto accepts TIP-20 stablecoin payments on Tempo (Moderato testnet, chain 42431).
A payer sends `transferWithMemo(merchant, amount, memo)` to one shared merchant address.
The memo carries the payment reference, so many open payments can share one address.

## What is implemented (branch `tempo-rail`)

- `backend/internal/blockchain/tempo`: TransferWithMemo log decoding, memo encoding, verification and de-duplication.
- The client reads only `eth_getLogs`, `eth_blockNumber` and receipts.
  It never fetches blocks, because Tempo type 0x76 transactions do not decode in go-ethereum.
- Memo format: UTF-8 text, left-aligned, zero-padded to 32 bytes ("PM-1042" is `0x504d2d31303432` then zeros).
  A UUID payment reference does not fit in 32 bytes, so its memo is the 32 hex digits without dashes.
- `backend/internal/service/tempo_memo_service.go`: the watcher.
  It scans from a block cursor (`blockchains.height` of the TEMPO row) in ranges of at most 1000 blocks.
  Each log is a candidate only: it must be from an accepted token, to the merchant address, with a positive amount and a successful receipt.
  The memo is resolved to a `payment_requests.reference_id`.
  Every decided log is written to `tempo_memo_credits`, unique on (tx hash, log index), so a rescan never credits twice.
  A credited log becomes a `deposits` row that is confirmed through `finalizeDeposit`.
  `finalizeDeposit` is the journal path extracted from the Solana watcher, so both chains post the ledger the same way (`LedgerService.RecordDepositIn`).
  The payment is then finalized with `DepositService.FinalizePayment`.
  An RPC or DB error holds the cursor below the failing block, so the log is retried on the next poll.
- `backend/internal/worker/tempo_memo_watcher.go`: polls every `TEMPO_POLL_INTERVAL_SECONDS` (default 2).
- Wiring: the watcher runs only when `TEMPO_MERCHANT_ADDRESS` is set.
  Boot fails if it is set but the TEMPO chain is not seeded or the RPC reports a different chain id.
- Seeds: TEMPO chain (1 confirmation, deterministic finality), PATHUSD.TEMPO and ALPHAUSD.TEMPO (6 decimals).
- Migration `2026101301_tempo_memo_credits` plus the schema manifest entry.
- Public checkout API:
  - `GET /api/v1/public/blockchain-currencies` lists TEMPO/PATHUSD and TEMPO/ALPHAUSD, but only when the watcher is configured.
  - `POST /api/v1/public/deposit-address/reference/:id` with `{"blockchainCode":"TEMPO","currencyCode":"PATHUSD"}` attaches the Tempo leg and returns the merchant address and `memo`.
  - `GET /api/v1/public/payment/:id` returns `blockchainCode`, `currencyCode`, `depositAddress`, `amountInUSD` and `memo`.
- `backend/cmd/tempoprobe`: prints the memo for a reference (`-memo <ref>`) or lists memo payments to an address (`-to 0x...`).
- `tracks/tempo/pay.sh`: pays a payment with `cast send ... transferWithMemo`.

Configuration:

| Variable | Default | Meaning |
|---|---|---|
| `TEMPO_MERCHANT_ADDRESS` | empty (off) | Receiving address for all Tempo payments |
| `TEMPO_RPC_URL` | `https://rpc.moderato.tempo.xyz` | Tempo JSON-RPC endpoint |
| `TEMPO_MEMBER_ID` | 0 (any) | Only credit payments of this merchant member |
| `TEMPO_START_BLOCK` | 0 (head - 2000) | First block when the cursor is empty |
| `TEMPO_POLL_INTERVAL_SECONDS` | 2 | Poll interval |

## Evidence

Real Moderato payment, end to end through the local backend (disposable database `payminto_tempo_test`):

- Payment `84933e7d-d0fc-46dd-b616-b602be747f9e` for 1.50 USD, Tempo leg attached through the public API.
- Paid with `tracks/tempo/pay.sh`: 1.5 pathUSD `transferWithMemo` to merchant `0x82c70b91731914e7b678a232b366336Af1436455`, memo `84933e7dd0fc46ddb616b602be747f9e`.
- Transaction: https://explore.testnet.tempo.xyz/tx/0x0bd786d78b7205537319369b9dc30a8d09a79854eae0ac809223aece94a4a00d (block 39362768, log index 26, status 1).
- Within seconds the watcher logged `deposit 3 seen` and `deposit 3 confirmed, payment state FILLED`.
- The payment is `FILLED`, deposit 3 is `confirmed` with 1 confirmation, `tempo_memo_credits` says `credited`.
- Ledger journal 3 for deposit 3: `+1.5 crypto_assets / -1.5 merchant_balance`.
- A plain `cast send` worked for sending on Tempo; fees were paid in a USD TIP-20 automatically.

Tests: `go test ./...` and `go vet ./...` in `backend` pass.
The new tests are table-driven and cover memo encoding, log decoding, verification, de-duplication, rescan idempotency, reverted transactions and cursor hold on RPC errors.

The watcher was also run against a busy third-party recipient on Moderato.
It decided 24 real logs (blocks 39353995 to 39354110) as `no_reference`, because their memos are binary and not Payminto references.

## Not done

- The Moderato faucet was down from about 08:55 to 09:35 IST on 2026-10-13 (`request too large`), so retry if funding fails.
- Sending from the backend (refunds, payouts) and sweeping are out of scope.
- No USD conversion: `FinalizeFromConfirmedDeposits` compares the stablecoin amount to `amountInUSD` 1:1.
- The checkout UI lives on a separate branch (`checkout-tempo`).
