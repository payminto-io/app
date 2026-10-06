# EVM Settlement — Design & Review (Sweep + Withdrawal)

How Payminto moves EVM funds (ETH / Base / Polygon, native + ERC-20) from
per-payment deposit addresses into cold storage (sweep) and out to merchants
(withdrawal). Grounded in production patterns from the references below.

## Research — how mature systems do it

- **Per-payment HD deposit addresses** receive funds; a background **sweeper**
  consolidates confirmed balances into an offline **treasury (cold) wallet**.
  (AWS "Processing digital asset payments" reference architecture.)
- **Native sweep:** send `balance − gasCost` from the deposit address to cold.
  Servers hold no static keys — the key is derived on demand from the vault
  mnemonic and zeroed after signing.
- **ERC-20 sweep is two-phase:** the deposit address holds tokens but no ETH for
  gas, so a **gas station / hot "gas tank"** first sends a small ETH top-up to
  the deposit address, then the ERC-20 `transfer(cold, amount)` is broadcast.
  (Fireblocks "Ethereum Gas Station"; AWS Sweeper.)
- **Nonce:** a stuck tx blocks every later tx from the same address. Always read
  the latest pending nonce from chain before sending. A deposit address sends
  exactly **one** sweep tx, so chain-nonce is sufficient there; the shared **gas
  tank** and **withdrawal hot wallet** send many txs and need a **serialized
  per-address nonce** (single-flight) to avoid drift.
  (Chainstack / QuickNode / thirdweb nonce guides.)
- **Confirmations:** mark a tx settled only after N confirmations, polling the
  receipt; support fee-bump replacement for stuck txs (future).

## Payminto design

### Components (already built — M1 core)
- `ethereum.Adapter` — EVM signer: `SignNativeTransfer`, `SignERC20Transfer`
  (pure, EIP-155), plus `SendNative`/`SendERC20`, `PendingNonce`,
  `SuggestGasPrice`, `NativeBalance`.
- `blockchain.EVMSigner` — capability interface for the above.
- `service.KeyResolver` — deposit/hot address → private key (vault mnemonic +
  `AddressPool.PathIndex`), re-derive-and-verify guard, key zeroed after use.
- `service.EVMBroadcaster` — resolves key + selects adapter + broadcasts.

### Native sweep (implemented this iteration)
`EVMSweepService.SweepConfirmedNative`:
1. List `confirmed` deposits whose currency is **native** on an EVM chain.
2. For each, `EVMBroadcaster.SweepNative(chain, depositAddr, coldWallet)`:
   read on-chain balance + gas price, compute `amount = balance − gasPrice*21000`,
   skip if ≤ dust, else sign + broadcast.
3. Persist a `Sweep` batch + `SweepTransaction` (status `broadcast`, real txHash,
   real gasFee), and mark the deposit `swept` (new status) so it is never swept
   twice. Idempotent: re-runs skip already-`swept` deposits.

### ERC-20 sweep (next increment — designed, not yet wired)
Two-phase via a hot **gas tank** address: top up deposit addr with estimated
gas, wait for that tx, then broadcast `transfer(cold, tokenBalance)`. Reuses
`EVMBroadcaster.SendERC20`. Requires gas-tank address config + balance alerts.

### Withdrawal (IMPLEMENTED for EVM)
Withdrawals (merchant payouts) are funded from the merchant's registered **hot
wallet** (`POST /wallets/hot`; key stored in the vault under `hot_wallet.{id}`,
address in config). `HotWalletSource.Resolve(memberID, chainCode)` returns the
address + decrypted key. `WithdrawalProcessingService.Execute` now, for EVM
chains: resolves the hot-wallet key, `SendNative`/`SendERC20` (native vs ERC-20
by the currency's contract address) to `ToAddress`, and stores the **real**
txHash / from-address / gas fee. On broadcast failure the claim is reverted
(retry next round) — no Withdraw row or ledger entry is written for an un-sent
payout. Non-EVM chains use a recorded-only fallback until wired.

Remaining refinement: confirmation-gating (advance `sent → processed` only after
N confirmations, mirroring `EVMSweepConfirmer`).

### Confirmation tracking (IMPLEMENTED)
`EVMSweepConfirmer` polls `SweepTransaction` rows in `broadcast`/`confirming`,
calls `adapter.GetConfirmations(txHash)` via `AdapterConfirmationChecker`, and
advances `broadcast → confirming → confirmed` at the chain's `MinConfirmations`.
On `confirmed` it calls `SweepService.MarkCompleted` (idempotent) to record the
ledger. Runs inside `EVMSweepWorker` each tick after the sweep pass.

## Developer backlog (remaining work — the "store")

Tracked, prioritized work to finish EVM settlement and beyond:

1. **ERC-20 two-phase sweep** — gas-fund the deposit address from a hot gas tank,
   wait for that tx, then `EVMBroadcaster.SendERC20(cold, tokenBalance)`. Needs a
   `GAS_TANK_ETH` address (an `AddressPool` entry, resolvable by `KeyResolver`)
   and low-balance alerting. Reuses the existing signer + confirmer.
2. **Withdrawal broadcast** — decide the payout **hot-wallet source** (`HOT_WALLET_ETH`,
   an `AddressPool` entry), then replace the `stub_tx_` path in
   `WithdrawalProcessingService.Execute` with `EVMBroadcaster.SendNative/SendERC20`,
   storing the real txHash/from/gasFee and advancing `sent → processed` only after
   confirmations (reuse the confirmer pattern for `Withdraw` rows).
3. **Failed-sweep handling** — mark sweep tx `failed` + deposit back to `confirmed`
   when a broadcast tx is dropped/replaced after N rounds (fee-bump replacement).
4. **Live testnet verification** — fund a Sepolia/Base-Sepolia deposit address, set
   `COLD_WALLET_ETH`, and run the full deposit → confirm → sweep → confirmed cycle.
5. **TRX** — `createtransaction` + sign txID flow; **BTC** — replace the bitcoind
   JSON-RPC adapter with an esplora (Blockstream) REST adapter, then UTXO/PSBT
   signing. (Deferred per product priority.)
6. **TRC-20 deposit parsing** — verify `getblockbynum` address format with Nile
   fixtures before decoding `TriggerSmartContract`.

## Testing layout (convention)

Tests are colocated `_test.go` files in the package they exercise — Go's standard
convention. The `go` toolchain excludes `_test.go` from production builds, so test
code never ships. Cross-package integration helpers (e.g. `database/testdb.go`)
are guarded by `//go:build integration`. A separate `test/` directory is
intentionally avoided: it is non-idiomatic in Go and would force exporting
internal functions purely for testing.

## Safety properties
- Private keys never logged, never persisted in plaintext, zeroed after signing.
- Sweep amount = balance − gas, computed from live chain state (never overdraw).
- Deposit `swept` status makes sweeping idempotent (no double-spend attempts).
- Address-match guard in `KeyResolver` aborts if derivation ≠ stored address.

## References
- AWS — Processing digital asset payments: https://aws.amazon.com/blogs/web3/processing-digital-asset-payments-on-aws/
- Fireblocks — Ethereum Gas Station: https://www.fireblocks.com/blog/goodbye-failed-erc20-transactions-introducing-ethereum-gas-station
- Chainstack — Ethereum nonce management: https://chainstack.com/ethereum-nonce-management/
- QuickNode — managing nonces: https://www.quicknode.com/guides/ethereum-development/transactions/how-to-manage-nonces-with-ethereum-transactions
