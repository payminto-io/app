# blockchain

Owns the abstract blockchain layer: the `ChainAdapter` interface that every chain implementation must satisfy, the normalised `Transaction` and `TxParams` value types emitted by monitors, the thread-safe `AdapterRegistry` that maps chain code → adapter at runtime, and the `RPCPool` that load-balances RPC calls across healthy nodes with failover and cooldown. Chain-specific implementations live in subpackages (`bitcoin`, `ethereum`, `tron`). Services and workers depend only on this package's interfaces, never on concrete chain code — this is the seam that keeps the rest of the backend chain-agnostic. Depends on `internal/models` and `internal/repository` for RPC node metadata.

## Files

- `adapter.go` — `ChainAdapter` interface, `Transaction`, `TxParams`, `WatchedAddressInfo`.
- `registry.go` — `AdapterRegistry`, the thread-safe code→adapter map.
- `rpc_pool.go` — `RPCPool` multi-node round-robin with health tracking.
- `*_test.go` — interface contract and pool failover tests.

## See also

- `internal/blockchain/bitcoin`, `.../ethereum`, `.../tron` — concrete adapters
- `internal/worker` — block processors that consume adapters
- `internal/service` — sweep/withdrawal services that pick adapters by code
