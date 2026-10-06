# tron

Owns the Tron implementation of `blockchain.ChainAdapter`. Talks to TronGrid-compatible nodes over REST/HTTP through the Payminto `RPCPool` for multi-node failover. Supports `mainnet`, `nile`, `shasta`, and `regtest` networks, and exposes the `Adapter` type together with `TronMainnetGenesisHash` for live-node fingerprinting. The monitor polls `getnowblock` / `getblockbynum` on a short interval (Tron blocks are ~3s) and emits normalised `blockchain.Transaction` records for watched addresses, decoding both TRX transfers and TRC-20 contract calls. Depends on `internal/blockchain` for shared types and `RPCPool`.

## Files

- `client.go` — `Adapter` struct, HTTP/JSON client, and network constants.
- `monitor.go` — block polling loop emitting TRX/TRC-20 transfers.
- `client_test.go` — client and adapter behaviour tests.

## See also

- `internal/blockchain` — `ChainAdapter` interface and `RPCPool`
- `internal/worker/tron_block_processor.go` — worker that drives this adapter
