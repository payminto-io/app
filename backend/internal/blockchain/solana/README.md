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
- `sweep.go` - `SweepInstructions`, `BuildSweepMessage`, `SendAndConfirm` (recent blockhash with rebuild on expiry).
- `keypair.go` - fee payer key in base58, CLI JSON array, or file path.
- `scripted_caller.go` - fixture transport for tests in this and other packages; fixtures under `testdata/tx`.

Service side: `internal/service/solana_deposit_service.go` (watcher), `solana_sweep_service.go` (sweeper), `solana_deposit_address.go` (owner to ATA at assignment), `solana_wiring.go` (registry). Workers: `internal/worker/solana_deposit_watcher.go`, `solana_sweep_worker.go`. Table: `solana_deposit_accounts` (migration `2026100702`).

## Address model

One owner keypair per payment at SLIP-0010 `m/44'/501'/n'/0'` (`crypto.DeriveSolanaAddress`), drawn from the family `sol` address pool like every other chain. The payment's deposit address (`deposit_addresses.address`) is the owner's associated token account for the chosen mint; `solana_deposit_accounts` records owner, ATA, mint, token program and the watcher cursors. Nothing is created on chain at assignment: the payer's wallet creates the ATA when it sends (payer pays that rent), and the sweep closes it afterwards (rent returns to our fee payer).

Checkout instruction: show the owner address with the token named ("send USDC on Solana to <owner>"). Wallets derive the ATA from an owner address and create it; pasting a token account address into a wallet is the known footgun. The public payment and assign-address responses carry both: `depositAddress` (ATA) and `depositOwnerAddress` / `ownerAddress`.

## Detection

`SolanaDepositService.PollOnce` calls `getSignaturesForAddress` for the ATA and for the owner, newest first down to the stored cursor, paging with `before` when a page is full, then `getTransaction` (jsonParsed, commitment confirmed) per signature. Credits of the exact mint into the ATA become a `deposits` row through `DepositService.RecordDeposit` (pending, required confirmations from `blockchains.min_confirmations`). Anything else that reached the payment's accounts becomes a `missed_deposits` row with the reason prefixed by the anomaly kind, never a credit.

`ConfirmOnce` uses `getSignatureStatuses`: `confirmed` is seen (status `confirming`, the cluster's confirmation count, capped below required), `finalized` is credited (status `confirmed`, confirmations = required, ledger journal posted in the same transaction, then `FinalizePayment` decides filled, partial or over). An error, or a signature the cluster no longer knows after the drop grace, marks the deposit `failed`; nothing was credited, so nothing is reversed in the ledger.

Token-2022 mints are refused unless the `blockchain_currencies.standard` is `SPL-2022`; the ATA is derived under that program and transfers from the other program are anomalies.

## Sweeps

`SolanaSweepService.SweepConfirmed` groups confirmed deposits by mint and token account, claims them (`ClaimForSweep`, at most once), reads each ATA's balance, and builds one transaction per batch of up to five accounts: compute budget, idempotent hot-wallet ATA creation, then `transferChecked` and `closeAccount` per account. The fee payer (`SOLANA_FEE_PAYER_KEY`) signs and pays; each owner signs through `KeyResolver` (family `SOL_Family`). Broadcast waits for `confirmed`; `TrackConfirmations` completes the sweep on `finalized`, booking the token move and the SOL fee through `LedgerService.RecordSweepIn` (gas in `SOL.SOLANA`, never in the token) and rent as a separate `solana_rent` adjustment (closed accounts are income, a newly funded hot ATA is an asset). A signature the cluster never saw past the grace period fails the sweep and returns its deposits to `confirmed`.

Durable nonces are not used; a recent blockhash with a rebuild on expiry covers the retry case, and the fee payer never signs two live versions of the same batch because the first must have expired first.

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

RPC endpoints are `rpc_nodes` rows for the `SOLANA` chain (seeded: public mainnet and devnet endpoints; add a paid provider for production). Mints are `blockchain_currencies.address` in `migrations/seeds`.

## Tests

Unit: `go test ./internal/blockchain/solana/ ./internal/service/ -run 'Solana'` (fixtures in `testdata/tx`, SQLite). End to end: `go test -tags=integration ./internal/service/ -run TestSolanaEndToEnd` starts `solana-test-validator`, creates two mints and runs assignment, four payment shapes, finalization, ledger and sweeps; it skips with a message when the binary is missing.
