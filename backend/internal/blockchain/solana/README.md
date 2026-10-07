# solana

Owns the Solana implementation of `blockchain.ChainAdapter` and the chain-specific pieces around it: SPL token deposits for USDC and USDT as peer assets, per-address signature detection, and fee-sponsored batched sweeps. No Solana SDK: the wire formats are small and stable, so this package carries them itself with one extra dependency (`filippo.io/edwards25519`, for the off-curve check that program-derived addresses require).

Design and acceptance: `.scratch/payments-v1/issues/09-solana-usdc.md`. Report: `.superpowers/solana-report.md`.

## Files

- `pubkey.go` - `PublicKey`, program ids, `FindProgramAddress`, `AssociatedTokenAddress`, `TokenProgramFor` (standard `SPL` or `SPL-2022`).
- `tx.go`, `tx_decode.go` - legacy message compilation (runtime account ordering), signing, shortvec, decoder.
- `instructions.go` - system, SPL token (`transferChecked`, `closeAccount`, `mintTo`, `initializeMint2`), ATA create-idempotent, compute budget.
- `rpc.go`, `parsed.go` - JSON-RPC `Client` over `blockchain.RPCPool` (`PoolCaller`) or one URL (`StaticCaller`); typed `jsonParsed` transaction.
- `parse.go` - `ExtractDeposits`: credits of the expected mint into the watched ATA; anomalies for wrong mint, wrong token program, native SOL to the owner, unexplained balance rises; `RentMovements` for sweep accounting.
- `adapter.go` - `ChainAdapter`; `ParseBlock` exists for the contract but the watcher is the detection path.
- `sweep.go` - `SweepInstructions`, `BuildSweepMessage`, `SignForSend` / `SendSigned` (sign, let the caller persist the signature, then send), `IsRejection`, `SendAndConfirm` (tools and tests).
- `keypair.go` - fee payer key in base58, CLI JSON array, or file path.
- `scripted_caller.go` - fixture transport for tests in this and other packages; fixtures under `testdata/tx`.

Module: `internal/modules/solana.go` (`WireSolana`: chain row, RPC pool with per-node rate limit and 429 backoff, live refuses a pool of only public endpoints, fewer than two distinct endpoints, and non-mainnet clusters, fee payer parsed without echo, seeded mints validated against the chain). Service side: `internal/service/solana_deposit_service.go` (watcher), `solana_sweep_service.go` (sweeper), `solana_deposit_address.go` (owner to ATA at assignment, one transaction), `solana_wiring.go` (registry glue). Workers: `internal/worker/solana_deposit_watcher.go`, `solana_sweep_worker.go`. Tables: `solana_deposit_accounts`, `solana_sweep_attempts`, `solana_sweep_deposits`, `solana_sweep_locks` (migration `2026100708`). Sweep state model: `internal/service/SOLANA_SWEEPS.md`.

## Address model

One owner keypair per payment at SLIP-0010 `m/44'/501'/n'/0'` (`crypto.DeriveSolanaAddress`), drawn from the family `sol` address pool like every other chain. The payment's deposit address (`deposit_addresses.address`) is the owner's associated token account for the chosen mint; `solana_deposit_accounts` records owner, ATA, mint, token program and the watcher cursors. Nothing is created on chain at assignment: the payer's wallet creates the ATA when it sends (payer pays that rent), and the sweep closes it afterwards (rent returns to our fee payer).

Checkout instruction: show the owner address with the token named ("send USDC on Solana to <owner>"). Wallets derive the ATA from an owner address and create it; pasting a token account address into a wallet is the known footgun. The public payment and assign-address responses carry both: `depositAddress` (ATA) and `depositOwnerAddress` / `ownerAddress`.

## Detection

`SolanaDepositService.PollOnce` first expires accounts past `watch_until` (payment expiry plus `SOLANA_LATE_WINDOW_DAYS`), then reads every due token account in `getMultipleAccounts` batches of 100. Only an account whose balance moved, or that holds a held or unresolved signature, pays for `getSignaturesForAddress` on the ATA; the owner address is polled on `OwnerCadence` (one minute) because owner traffic is only anomalies. After payment expiry both drop to `LateCadence` (ten minutes). Signatures are read newest first down to the stored cursor, paging with `before`, and fetched with `getTransaction` (jsonParsed, confirmed) from up to two pool nodes.

A listed signature no node returns holds the cursor below it (`held_signature`, one attempt per tick); after `MaxHeldAttempts` (12) it is recorded as a `solana_unresolved_signature` anomaly, moved to `unresolved_signatures` and retried on its own, so later signatures proceed and nothing is skipped. `last_balance_raw` is persisted only once a signature poll explained the movement (a signature listed, or a held one); an unexplained movement keeps the account polled under `balance_hold_attempts` and ends in a `solana_unexplained_balance` anomaly. Expiry at `watch_until` reads the balance once more (`solana_late_balance` when it exceeds recorded deposits) and never expires an account with a held signature. Unresolved signatures delay expiry for `MaxUnresolvedPolls` (36) polls; past that the account expires with a `solana_unresolved_at_expiry` anomaly and the expired scan keeps retrying them. Expired accounts keep a daily balance scan for `ExpiredScanFor` (90 days) after `watch_until`. A transport error on the signature poll never clears a held signature and explains no balance movement, so the next tick polls again. The credit is the ATA's balance delta for that transaction; a difference from the parsed instructions is a `withheld_amount` or `unparsed_credit` anomaly. Tokens sent to `ATA(ATA, mint)` (a wallet treating the deposit address as an owner) are `stranded_in_pda_ata`. Anomaly rows are capped per destination per day.

`ConfirmOnce` orders by least recently checked and uses `getSignatureStatuses` with `searchTransactionHistory` only for signatures the status cache does not know. `confirmed` is seen (status `confirming`), `finalized` is credited (status `confirmed`, then `FinalizePayment`). The payment journal is posted in the same transaction unless the switch owns the payment (its `invoice_id` is a `switch_payment_attempts` id), in which case the switch's chaindeposit connector posts it on `Sync`: every finalized deposit gets exactly one payment journal, whether the payment came from the switch or the legacy `POST /payment` route. A deposit is failed only with evidence: past the drop grace, the finalized slot beyond the deposit's slot, and the signature absent at finalized on two distinct endpoints (`NodeCaller`); with a one-endpoint pool the deposit stays pending and a `solana_evidence_unavailable` anomaly says why. A failed deposit seen again within `ReviveWindow` (7 days, checked on the late cadence) is revived and credited once. A deposit finalized for a payment already closed is a `late_payment` anomaly.

Token-2022 mints are refused unless the `blockchain_currencies.standard` is `SPL-2022`; the ATA is derived under that program and transfers from the other program are anomalies.

## Sweeps

The state model, its invariants and the recovery of every state after a restart are in `internal/service/SOLANA_SWEEPS.md`; this is the summary.

`SolanaSweepService.SweepConfirmed` groups confirmed deposits by mint and token account, skips accounts with a sweep in flight (`solana_sweep_locks`, unique per token account), claims the deposits (`ClaimForSweep`, at most once) and reads each ATA's finalized balance against the claimed sum (the transfer is the claim; money beyond it stays for the next sweep and the account is then not closed; less than the claim is a drain). It writes the sweep (`processing`), one `sweep_transactions` per account, the `solana_sweep_deposits` links and the locks in one transaction; then signs, persists the attempt (signature, blockhash, `lastValidBlockHeight`) and only then sends. A JSON-RPC rejection fails the attempt; a transport error leaves it tracked, because the node may have forwarded it. A drained account is never marked swept: a signature of a sweep still tracked is left to the tracker, one of a sweep booked failed revives that sweep (`solana_failed_sweep_landed`), a transaction our fee payer signed that no row knows is recorded as the in-flight sweep's attempt, anything else is a `solana_unexplained_drain` anomaly and the account is set aside.

`TrackConfirmations` runs one reconciler pass per in-flight sweep, paging by id. The first finalized attempt is booked exactly once (sweep transactions confirmed with their fee share, `RecordSweepIn` for the token move and SOL gas, `solana_rent` for reclaimed and funded rent, deposits swept, lock released, accounts closed), and the hot ATA's balance delta is reconciled against the booked total (`solana_sweep_mismatch` anomaly on any difference). Attempts are rebuilt only with evidence: every attempt unknown, the finalized block height past the latest attempt's `lastValidBlockHeight`, the signature absent at finalized on two distinct endpoints, and every account still holding its claim. After `MaxAttempts` (3) the same balance guard applies before the sweep fails and releases exactly its own deposits; an account below its claim with nothing finalized waits, and after `DrainWait` (one hour) its history must explain the drain or the sweep fails with an anomaly. A `processing` sweep with no attempt past the blockhash validity window (a crash before signing) is resolved from the chain: every account holding its claim fails it with `solana_sweep_never_sent` and releases its deposits for the next round.

Durable nonces are deferred for V1 (ticket comments): tracking every attempt and rebuilding only on finalized evidence closes the double-version window a nonce account would close, without a nonce account to fund and advance.

## Configuration

| Variable | Meaning |
| --- | --- |
| `SOLANA_CLUSTER` | `mainnet-beta`, `devnet`, `testnet`, `localnet`; defaults from `BLOCKCHAIN_NETWORK_TYPE` |
| `SOLANA_HOT_WALLET_ADDRESS` | owner whose ATAs receive sweeps; empty disables sweeping |
| `SOLANA_FEE_PAYER_KEY` | base58 secret, Solana CLI JSON array, or a path to one; funds fees and hot ATA rent |
| `SOLANA_DEVNET_USDT_MINT` | fills and enables the seeded devnet USDT row (devnet has no official USDT) |
| `SOLANA_PRIORITY_FEE_MICROLAMPORTS` | compute unit price, default 0 |
| `SOLANA_COMPUTE_UNIT_LIMIT` | default 120000 |
| `SOLANA_SWEEP_BATCH_SIZE` | 1 to 5, default 5 (packet limit) |
| `SOLANA_CLOSE_DEPOSIT_ACCOUNTS` | default true |
| `SOLANA_POLL_INTERVAL_SECONDS`, `SOLANA_SWEEP_INTERVAL_SECONDS` | defaults 5 and 30 |
| `SOLANA_LATE_WINDOW_DAYS` | how long after payment expiry an account stays watched, default 7 |
| `SOLANA_RPC_REQUESTS_PER_SECOND` | per-node budget, default 10 (the public endpoint's) |

RPC endpoints are `rpc_nodes` rows for the `SOLANA` chain (seeded: public mainnet and devnet endpoints). Live needs at least one provider endpoint and at least two distinct endpoints, or boot is refused: drop and expiry evidence reads two.

## Polling cost

| Path | Per tick | Bound |
| --- | --- | --- |
| Watching accounts | one `getMultipleAccounts` per 100 due accounts; signature polls only on a moved balance, a held or unresolved signature, or the late cadence | `AccountsPerPoll` (500) |
| Expiring accounts | one balance read per 100 | 200 per tick |
| Expired scan | one balance read per 100, each account once per `ExpiredScan` (daily), plus a retry of its unresolved signatures | 500 per tick; only accounts within `ExpiredScanFor` (90 days) of `watch_until`, so the set does not grow with lifetime accounts |
| Sweep tracking | `getSignatureStatuses` per in-flight sweep; height and two-endpoint fetch only for an attempt past its height | pages of 200 in-flight sweeps |
| Drain explanation | up to `HistoryPages` (5) pages of 100 signatures, then up to 20 `getTransaction` for a fee-payer match | only on a balance below the claim | Mints are `blockchain_currencies.address` in `migrations/seeds`.

## Tests

Unit: `go test ./internal/blockchain/solana/ ./internal/service/ -run 'Solana'` (fixtures in `testdata/tx`, SQLite). End to end: `go test -tags=integration ./internal/service/ -run TestSolanaEndToEnd` starts `solana-test-validator`, creates two mints and runs assignment, four payment shapes, finalization, ledger and sweeps; it skips with a message when the binary is missing. `solana_devnet_integration_test.go` runs the same shapes against devnet with `SOLANA_DEVNET_PAYER_KEY` and `SOLANA_DEVNET_USDT_MINT` (skips when absent).
