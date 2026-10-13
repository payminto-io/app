# Tempo at Crypto World's Fair

Payminto accepts memo-tagged TIP-20 stablecoins on Tempo Moderato testnet, chain 42431.
The current showcase pairs Tempo with [Solana](../solana/README.md) for [Crypto World's Fair](https://colosseum.com/worldsfair).
The rail and wallet checkout are locally integrated; the latest public deployment is unverified.
Read the [project details and prior-work disclosure](../../docs/WORLDS_FAIR.md).

## Implemented path

A payer calls `transferWithMemo(merchant, amount, memo)` on the issued token contract.
Many invoices share one merchant address; a bytes32 memo distinguishes the invoice.
Short references use UTF-8 padded with zeros, and UUID references use the 32 hexadecimal digits without dashes before that encoding.

The watcher checks Moderato's chain id on every poll and reads logs, head height, and receipts without decoding full Tempo transaction blocks.
A credit requires a successful receipt, accepted token, positive amount, merchant recipient, and a matching issued payment leg and merchant ownership.
The `(transaction hash, log index)` key deduplicates replays while preserving distinct transfers from the same transaction.
Deposit journals and payment finalization use the shared gateway path; temporary failures remain retryable.
Partial fills and additional transfers retain the actual deposited amounts and update payment state.

Checkout shows the issued token, merchant address, exact amount, raw base units, and invoice memo.
The wallet path supports seeded pathUSD/PATHUSD and AlphaUSD/ALPHAUSD, preserves the issued asset, handles rejected wallet requests, and listens for confirmation updates.
An unknown partial-payment remainder suppresses the send amount rather than requesting the full invoice again.

## Configuration

Tempo is off when `TEMPO_MERCHANT_ADDRESS` is empty.
Enabling it requires `GATEWAY_ENVIRONMENT=test`, `BLOCKCHAIN_NETWORK_TYPE=testnet`, seeded Tempo currencies, and an RPC reporting chain 42431.
Live money configuration is refused.

| Variable | Default | Purpose |
| --- | --- | --- |
| `TEMPO_MERCHANT_ADDRESS` | Empty | Shared receiving address; a non-zero EVM address is required. |
| `TEMPO_RPC_URL` | `https://rpc.moderato.tempo.xyz` | Moderato endpoint. |
| `TEMPO_MEMBER_ID` | `0` | A positive id restricts assignment and credit to that merchant member. |
| `TEMPO_START_BLOCK` | `0` | Start block when no cursor exists; zero uses the configured lookback. |
| `TEMPO_POLL_INTERVAL_SECONDS` | `2` | Watcher interval; must be positive. |

The public currency list advertises Tempo only when its watcher is configured.
The address-assignment and payment APIs return the merchant address, selected asset, and memo for the issued invoice.
Use [CONTRIBUTING.md](../../CONTRIBUTING.md) for isolated local setup.

## Recorded real-network evidence

The original owner-run payment was executed on Moderato and credited through a local backend with disposable database `payminto_tempo_test`:

- Invoice `84933e7d-d0fc-46dd-b616-b602be747f9e`, for 1.50 USD.
- 1.5 pathUSD sent to `0x82c70b91731914e7b678a232b366336Af1436455` with memo `84933e7dd0fc46ddb616b602be747f9e`.
- [Transaction `0x0bd786d78b7205537319369b9dc30a8d09a79854eae0ac809223aece94a4a00d`](https://explore.testnet.tempo.xyz/tx/0x0bd786d78b7205537319369b9dc30a8d09a79854eae0ac809223aece94a4a00d), block 39362768, log index 26, receipt status 1.
- Payment `FILLED`, deposit 3 confirmed with one confirmation, and memo credit `credited`.
- Journal 3 recorded `+1.5 crypto_assets / -1.5 merchant_balance`.

These invoice and journal values are the recorded local result, not a fresh run of the subsequently integrated release.
The explorer transaction supplies real-network transfer evidence; latest public Tempo deployment remains unverified.
The original test helper [pay.sh](pay.sh) remains for historical reproducibility and broadcasts a transaction when executed.
The current checkout is the preferred place to inspect issued asset, amount, memo, and partial-payment state.

## Code and regression coverage

| Path | Purpose |
| --- | --- |
| [Tempo adapter](../../backend/internal/blockchain/tempo/) | Memo/log decoding, receipt and token verification, and RPC client. |
| [Memo service](../../backend/internal/service/tempo_memo_service.go) | Invoice ownership, cursor, deduplication, and deposit handling. |
| [Configuration guard](../../backend/internal/config/tempo.go) | Moderato-only test configuration. |
| [PostgreSQL regressions](../../backend/internal/service/tempo_memo_integration_test.go) | Concurrent replay, rollback/retry, partial fill, and overpayment. |
| [Checkout wallet UI](../../checkout/components/tempo-pay.tsx) | Exact amount and memo display, wallet request, and confirmation status. |
| [Checkout token/memo helpers](../../checkout/lib/tempo.ts) | Issued-token mapping and bytes32/base-unit encoding. |

Local unit, browser-fixture, and disposable PostgreSQL checks passed for the integrated changes.
The [CI workflow](../../.github/workflows/ci.yml) requires `TestTempoPostgres` tests under the race detector and rejects skipped integration tests.
Preview screens and scripted wallet/SSE tests demonstrate UI behavior, not new chain transfers.

## Remaining limits

Tempo refunds, payouts, and sweeps are not implemented.
Stablecoin amounts are compared to the USD invoice 1:1 without a conversion trade.
Deployment of the integrated update and a current product demo/founder pitch still need verification and supplied media.
