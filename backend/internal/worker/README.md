# worker

Owns every long-running background goroutine in the Payminto backend. Exposes the `Worker` interface (`Name() string`, `Start(ctx) error`), `WorkerStatus`, and the `Manager` that registers workers, starts them concurrently, tracks heartbeats and terminal errors, and shuts them down on context cancellation. Concrete workers include the generic `BlockchainProcessor` (driven by a `blockchain.ChainAdapter`) with per-chain constructors (`NewBitcoinBlockProcessor`, `NewEthBlockProcessor`, `NewBaseBlockProcessor`, `NewPolygonBlockProcessor`, `NewTronBlockProcessor`), plus the deposit processor, webhook processor, email processor, account processor, payment-expiry sweeper, address-pool warmer, and SCW deposit-wallet broadcaster. Depends on `internal/blockchain`, `internal/service`, `internal/repository`, and `internal/models`. `cmd/server` constructs the `Manager`, registers workers, and runs it alongside the HTTP server.

## Files

- `manager.go` — `Worker` interface, `WorkerStatus`, and concurrent `Manager`.
- `base_block_processor.go` — Base (Coinbase L2) constructor wrapper.
- `bitcoin_block_processor.go` — Bitcoin constructor wrapper.
- `eth_block_processor.go` — Ethereum mainnet/Sepolia constructor wrapper.
- `polygon_block_processor.go` — Polygon constructor wrapper.
- `tron_block_processor.go` — Tron constructor wrapper.
- `blockchain_processor.go` — generic `BlockchainProcessor` that reads from a `ChainAdapter` and fans out deposits.
- `deposit_processor.go` — confirms and advances deposits through state.
- `webhook_processor.go` — delivers queued webhooks with retries.
- `email_processor.go` — renders and sends queued emails.
- `account_processor.go` — reconciles ledger accounts.
- `payment_expiry.go` — expires stale payment requests.
- `address_pool_warmer.go` — pre-generates deposit addresses.
- `scw_deposit_wallet_broadcaster.go` — broadcasts pending SCW deposit wallets on-chain.
- `*_test.go` — per-worker unit tests (`manager_test.go`, `blockchain_processor_test.go`, `account_processor_test.go`, `address_pool_warmer_test.go`, `email_processor_test.go`, `scw_broadcaster_test.go`).

## See also

- `internal/blockchain` — chain adapters driving block processors
- `internal/service` — business-logic dependencies injected via `ProcessorDeps`
- `cmd/server` — constructs and runs the manager
