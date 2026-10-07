# Ticket 09 - Solana adapter with USDC and USDT side by side: report

Branch `solana`, worktree `gateway-wt-solana`, based on main with the double-entry ledger.
Commits (oldest first): `71b2fec`, `b711c0e`, `c99a501`, `da716f9`, `2aac9bd`, `24fcfe5`, plus the docs commit that adds this file and closes the ticket.

## What shipped

New package `backend/internal/blockchain/solana/` (README inside):

- `pubkey.go` - public keys, program ids, `FindProgramAddress`, `AssociatedTokenAddress`, `TokenProgramFor`.
- `tx.go`, `tx_decode.go`, `instructions.go` - legacy message compile/sign/serialize/decode; system, SPL token, ATA, compute budget instructions.
- `rpc.go`, `parsed.go` - JSON-RPC client over `blockchain.RPCPool` (`PoolCaller`) or a single URL (`StaticCaller`); typed `jsonParsed` transaction.
- `parse.go` - `ExtractDeposits` (credits of the exact mint into the ATA, anomalies otherwise), `RentMovements`.
- `adapter.go` - `blockchain.ChainAdapter` (`Code() == "SOLANA"`).
- `sweep.go` - batched sponsored sweep builder, `SendAndConfirm` with blockhash-expiry rebuild.
- `keypair.go`, `scripted_caller.go`, `testdata/tx/*.json` - fee payer key loader, fixture transport, recorded-shape fixtures.

Service, worker and schema:

- `internal/crypto/solana.go` - SLIP-0010 ed25519, `DeriveSolanaAddress(seed, n)` at `m/44'/501'/n'/0'`; verified against `solana-keygen` and the SLIP-0010 spec vectors.
- `internal/models/solana_deposit_account.go`, `internal/repository/solana_deposit_account_repo_impl.go`, migration `2026100702_solana_deposit_accounts.up.sql` (registered in `migrations.go`, `database.go`, `startup.go` manifest; migrations integration test now expects three).
- `internal/service/solana_deposit_address.go` - `DepositAddressService` maps the claimed pool owner to its ATA for SOLANA tokens and records both; `SolanaOwnerAddress` for the API.
- `internal/service/solana_deposit_service.go` - watcher (`PollOnce`, `ConfirmOnce`).
- `internal/service/solana_sweep_service.go` - sweeper (`SweepConfirmed`, `TrackConfirmations`).
- `internal/service/solana_wiring.go` - adapter registration by family, watcher and sweeper construction, devnet USDT mint fill, `familyOfChain`.
- `internal/service/ledger_service.go` - `RecordDepositIn` (journal keyed by deposit, so partial payments do not collide on the payment-keyed key).
- `internal/service/{wallet_service,address_pool_service,key_resolver,hot_wallet_source}.go`, `api/handler/wallet_handler.go` - the `SOLANA` / `sol` / `SOL_Family` cases.
- `internal/api/handler/public_api_handler.go` - `depositOwnerAddress` on the public payment, `ownerAddress` on assign responses.
- `internal/worker/solana_deposit_watcher.go`, `solana_sweep_worker.go`; `cmd/server/main.go` registers the watcher instead of a block processor for `SOLANA` and the sweep worker when custody is on and a hot wallet and fee payer are configured.
- `internal/config/config.go` - `SolanaConfig` (`SOLANA_*`), listed in the package README.
- `migrations/seeds/{mainnet,testnet}/900{1,2,3}_*.sql` - family `sol` (`SOL_Family`), chain `SOLANA` (`min_confirmations` 32), currencies `SOL`, `USDC`, `USDT`, `blockchain_currencies` rows with mints, RPC nodes.

Dependency added: `filippo.io/edwards25519` v1.2.0 (Filippo Valsorda's library, the same code the Go standard library vendors). It is needed for the "is this point on the curve" test that program-derived addresses require; `crypto/ed25519` does not expose point decoding. `gagliardetto/solana-go` was not taken: it pulls a large tree (binary codecs, zstd, websocket, rpc types) for what here is a few hundred lines of fixed layouts.

## Decisions

- **Addresses.** One owner keypair per payment from the `sol` family address pool (same `address_pools` and `KeyResolver` path as the other chains). `deposit_addresses.address` is the owner's ATA for the mint; `solana_deposit_accounts` holds owner, ATA, mint, token program, decimals and the two signature cursors. Nothing is created on chain at assignment; the payer's wallet creates the ATA (its rent) and the sweep closes it (rent to our fee payer).
- **Checkout instruction.** Show the owner address with the token named. Both `depositAddress` (ATA) and `depositOwnerAddress` are in the public payload. The checkout already has `SOLANA` in `NETWORK_NAMES` and `USDT` in its stable set, so no checkout change was needed; rendering the owner address is ticket 14's.
- **Detection.** `getSignaturesForAddress` per ATA and per owner down to the persisted cursor (paged with `before`), `getTransaction` jsonParsed at `confirmed`, top-level and inner instructions (`transfer`, `transferChecked`, `mintTo`), mint resolved from `transferChecked` or `postTokenBalances`. Credits only for the exact mint into the ATA. Balance delta is cross-checked: a rise larger than the parsed transfers is an `unparsed_credit` anomaly, not a credit.
- **Commitment.** `confirmed` is seen (`confirming`, with the cluster's confirmation count capped at required minus one); `finalized` is credited (`confirmed`, confirmations = required, journal in the same DB transaction, then `FinalizePayment` which yields FILLED, PARTIALLY_FILLED or OVER_FILLED). Required confirmations are seeded as 32 (`MAX_LOCKOUT_HISTORY`), which is what a rooted slot has; the gate is finalization, not the number.
- **Drops.** A signature the cluster no longer knows (status null) after the drop grace (3 minutes) and with no `getTransaction` record is marked `failed`. Nothing was credited before finalization, so nothing is reversed in the ledger. A transaction that errors on chain is also `failed`.
- **Anomalies.** Wrong mint (USDT to a USDC payment's owner, lands in the owner's USDT ATA), wrong token program, native SOL to the owner, unexplained balance rise: `missed_deposits` rows with `reason` prefixed by the kind, `blockchain_currency_id` set to the received mint's row when we have one, deduplicated per (signature, destination, reason). No new table.
- **Token-2022.** Accepted only for rows seeded with `standard = 'SPL-2022'`; the ATA is derived under that program and the other program's transfers are anomalies.
- **Sweeps.** Grouped by mint and token account, claim-first (`ClaimForSweep`), up to five accounts per transaction (the packet limit with closes: ~152 bytes per account on a ~390-byte base). Instructions: compute unit limit, optional unit price, idempotent hot-wallet ATA creation, then `transferChecked` + `closeAccount` per account. Fee payer signs and pays; owners sign via `KeyResolver`. Waits for `confirmed`; `TrackConfirmations` completes on `finalized`.
- **Ledger.** Deposit: `KindPayment`, reference `deposit:<id>`, `crypto_assets +amount / merchant_balance -amount` in `USDC.SOLANA` or `USDT.SOLANA`. Sweep: existing `RecordSweepIn` (token move, gas as `sweep_gas` expense against `crypto_assets` in `SOL.SOLANA`, from `meta.fee`). Rent: `solana_rent:<sweep>` adjustment, closed accounts' lamports as `rent_reclaimed` income, a newly funded hot ATA as `token_account_rent` asset. `SweepTransaction.gas_fee` carries an equal share of the batch fee.
- **Retries.** Recent blockhash; on expiry (`lastValidBlockHeight` passed, signature unknown) the message is rebuilt with a fresh blockhash, three attempts. Durable nonces are not implemented.
- **Devnet USDT.** Devnet has no official USDT. The testnet seed ships the USDT row disabled with an empty mint; `SOLANA_DEVNET_USDT_MINT` fills and enables it at boot (logged either way). Create one with `spl-token create-token --decimals 6` on devnet.

## RPC methods used

`getSlot`, `getBlockHeight`, `getLatestBlockhash`, `getSignaturesForAddress` (`until`, `before`, `limit`), `getTransaction` (jsonParsed, `maxSupportedTransactionVersion` 0), `getSignatureStatuses` (`searchTransactionHistory`), `sendTransaction` (base64, preflight at confirmed), `getBalance`, `getTokenAccountBalance`, `getAccountInfo` (jsonParsed, for mints), `getMinimumBalanceForRentExemption`, `getBlock` (only by `ParseBlock`), `getHealth`, `getGenesisHash`, `requestAirdrop` (tests).

## Tests

Unit (SQLite in memory, `ScriptedCaller` over fixtures in `internal/blockchain/solana/testdata/tx`):

- parser: USDC `transferChecked`, USDT plain `transfer` (mint from balances), wrong mint, payment to owner that creates the ATA, native SOL to owner, failed transaction, inner CPI transfer, unparsed balance rise, Token-2022 program rejected, rent movements.
- adapter: confirmations (unknown, confirmed count, finalized, failed), `ParseBlock`, token balance via ATA; `SendAndConfirm` rebuild after blockhash expiry and on-chain failure.
- sweep builder: instruction list, signer set (fee payer first, owners, never the hot wallet), close to fee payer, five-account packet fits and six does not; message/transaction decode round trip.
- service: detection then confirmed then finalized with the journal in `USDC.SOLANA`, USDT in `USDT.SOLANA`, wrong mint anomaly never credited and not duplicated, owner-address payment, native SOL anomaly, drop before finalization reversed without a journal, partial then over payment with one journal per deposit, deposit address assignment maps owner to ATA, key resolver for `SOL_Family`, sweep batching with decoded broadcast, completion postings (token, gas in SOL, rent), dropped sweep releases deposits, broadcast failure releases claims, empty account marked swept without a broadcast.

End to end (`-tags=integration`, `internal/service/solana_integration_test.go`): starts `solana-test-validator` on free ports (skips with a message if the binary is missing), Postgres testcontainer, creates USDC-like and USDT-like mints, derives owners from the HD path, assigns four payments, pays to the ATA, to the owner, with the wrong mint, and in two parts; detects, finalizes with journals, sweeps both mints in two sponsored transactions, closes the deposit accounts, checks the hot wallet balances and the fee payer's lamports. Passes in about 20 seconds.

## What needs the switch (05) and custody (08) merges

- Attempts: a Solana deposit lands in `deposits` and `payment_requests` exactly as the other chains do, so the `chaindeposit` connector reads it unchanged. "Succeeded attempt" from the ticket's acceptance is the switch's to emit from those rows.
- Deposit journal: the watcher posts `payment` journals keyed `payminto:deposit:<id>` at finalization (`LedgerService.RecordDepositIn`, added here). The EVM/BTC/Tron processors post nothing today. If the switch posts the payment journal (with fee lines) on attempt success, pass `nil` for the ledger when constructing `SolanaDepositService` in `solana_wiring.go` so the deposit is booked once; the watcher's behaviour does not otherwise change.
- Signing: owner keys come from `KeyProvider` (`KeyResolver`), the fee payer from `solana.ParseKeypair(SOLANA_FEE_PAYER_KEY)`. The custody direct provider can wrap both behind its port; `SolanaSweepService` takes them as values, so swapping is a constructor change in `solana_wiring.go`.
- Hot wallet: `SOLANA_HOT_WALLET_ADDRESS` is the sweep destination owner; custody can supply it instead of config.

## Concerns

- Inherited: SQL seeds leave `blockchains.family` empty for ETH/BTC/TRX and use family codes `evm/btc/trx`, while the registry and `KeyResolver` switch on `ETH_Family` etc. I added `familyOfChain` (registry) and accept `sol`/`SOL_Family` in the resolver so Solana works from the seeds; the other chains' mismatch is unchanged and worth a look.
- The watcher polls every watched account each tick (bounded at 200 per poll, least recently polled first). Accounts stop being polled only when their deposit ATA is swept and closed. Expired payments with no deposit keep being polled; a `watch_until` cut-off tied to payment expiry would bound RPC cost on a busy instance.
- `FinalizeFromConfirmedDeposits` compares token amount to `amount_in_usd` directly; fine for stablecoins, not for a future SOL-priced payment.
- A sweep that lands after we fail it (past the drop grace, cluster forgot the signature, then it lands anyway) would leave the deposits re-queued; the next round sees an empty ATA and marks them swept without a sweep row for the earlier transaction. Rare, logged, and the chain is the source of truth, but worth a reconciliation query later.
- `ParseBlock` fetches whole blocks with jsonParsed; it satisfies the adapter contract and the tests but is not a practical mainnet path. `main.go` deliberately registers the watcher instead of the block processor for `SOLANA`.
