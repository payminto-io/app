# bitcoin

Owns the Bitcoin implementation of the `blockchain.ChainAdapter` interface. Talks to Bitcoin Core over JSON-RPC with a minimal `net/http` client for block queries, UTXO scanning, transaction construction, and broadcast. Network selection (mainnet / testnet3 / signet / regtest) is controlled by a `*chaincfg.Params` passed to `NewAdapter`. Exposes the `Adapter` type and its block-monitor goroutine which polls `getblockcount` and emits `blockchain.Transaction` records for watched addresses. Depends on `internal/blockchain` for shared types/`RPCPool` and `btcsuite/btcd/chaincfg` for network parameters.

## Files

- `client.go` — `Adapter` struct, constructor, and JSON-RPC client methods.
- `monitor.go` — block polling loop that emits UTXO-like transactions.
- `client_test.go` — RPC client and adapter behaviour tests.

## See also

- `internal/blockchain` — `ChainAdapter` interface and `RPCPool`
- `internal/worker/bitcoin_block_processor.go` — worker that drives this adapter
