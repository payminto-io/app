# ethereum

Owns the EVM implementation of `blockchain.ChainAdapter`, shared by Ethereum mainnet/Sepolia, Base, Polygon, and other EVM L1s and L2s. Uses `go-ethereum/ethclient` wrapped over the Payminto `RPCPool` for multi-node failover. Exposes `Adapter` (one instance per chain, parameterised by `code`/`name`/`chainID`/pool), the block monitor goroutine that decodes native transfers and ERC-20 `Transfer` events, and the internal ERC-20 helpers (`erc20BalanceOf`, the minimal ERC-20 ABI, and the Transfer event topic). Depends on `internal/blockchain` for shared types and `go-ethereum` for RPC, ABI encoding, and event decoding.

## Files

- `client.go` — `Adapter` struct, ethclient lifecycle, and RPC-pool rebinding.
- `erc20.go` — minimal ERC-20 ABI, `balanceOf`, and Transfer event decoding.
- `monitor.go` — block poller that emits native + ERC-20 transfers.
- `client_test.go` — adapter and ERC-20 decode tests.

## See also

- `internal/blockchain` — `ChainAdapter` interface and `RPCPool`
- `internal/worker/eth_block_processor.go`, `.../polygon_block_processor.go`, `.../base_block_processor.go` — EVM-family workers
