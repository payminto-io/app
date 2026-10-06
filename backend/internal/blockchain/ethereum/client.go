// Package ethereum implements the ChainAdapter for Ethereum and EVM-compatible
// chains (Base, Polygon, Optimism, Arbitrum, etc.). It uses go-ethereum's
// ethclient wrapped over a Payminto RPCPool for multi-node failover.
package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// Adapter is the Ethereum ChainAdapter implementation. One instance per chain
// (ETH mainnet, Sepolia, Base, Polygon, etc.). chainID is used by IsMainnet()
// and transaction signing.
type Adapter struct {
	code    string
	name    string
	chainID int64 // stored for IsMainnet check and tx signing
	pool    *blockchain.RPCPool

	mu       sync.RWMutex
	client   *ethclient.Client // current ethclient for the active RPC node
	activeID uint              // id of the currently-connected rpc_node row
}

// NewAdapter constructs an Ethereum adapter. The pool argument may be nil for
// tests that only exercise IsMainnet/Name/Code without making RPC calls.
func NewAdapter(code, name string, chainID int64, pool *blockchain.RPCPool) *Adapter {
	return &Adapter{
		code:    code,
		name:    name,
		chainID: chainID,
		pool:    pool,
	}
}

// Name implements ChainAdapter and returns the human-readable chain name.
func (a *Adapter) Name() string { return a.name }

// Code implements ChainAdapter and returns the short chain code used as map keys.
func (a *Adapter) Code() string { return a.code }

// IsMainnet returns true for Ethereum mainnet (chain_id 1). Base and Polygon
// have their own mainnet IDs (8453 and 137) — callers check Code() for those.
// Sepolia (11155111), Base Sepolia (84532), Polygon Amoy (80002), and other
// testnets return false.
func (a *Adapter) IsMainnet() bool {
	// Known mainnet chain IDs for EVM chains we care about
	switch a.chainID {
	case 1, 8453, 137, 10, 42161: // ETH, Base, Polygon, Optimism, Arbitrum
		return true
	}
	return false
}

// ChainID returns the numeric chain ID for transaction signing.
func (a *Adapter) ChainID() int64 { return a.chainID }

// connect establishes an ethclient.Client against a pool-selected node.
// Marks the node failed if dialing errors out.
func (a *Adapter) connect(ctx context.Context) (*ethclient.Client, uint, error) {
	if a.pool == nil {
		return nil, 0, errors.New("rpc pool not configured")
	}
	node, err := a.pool.Pick()
	if err != nil {
		return nil, 0, fmt.Errorf("pick rpc node: %w", err)
	}
	client, err := ethclient.DialContext(ctx, node.URL)
	if err != nil {
		a.pool.MarkFailure(node.ID, err)
		return nil, node.ID, fmt.Errorf("dial %s: %w", node.URL, err)
	}
	return client, node.ID, nil
}

// GenerateAddress is not implemented here — callers should use
// crypto.DeriveEthAddress from the HD wallet instead.
func (a *Adapter) GenerateAddress(index uint32) (string, error) {
	return "", errors.New("use crypto.DeriveEthAddress for EVM chains")
}

// GetBalance queries the on-chain balance for an address. If token is empty
// or "ETH" (native), returns the wei balance. Otherwise token is the
// contract address (0x...) of an ERC-20 and the balance is in token units.
func (a *Adapter) GetBalance(ctx context.Context, address, token string) (*big.Int, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var bal *big.Int
	addr := common.HexToAddress(address)

	if token == "" || token == "ETH" {
		bal, err = client.BalanceAt(ctx, addr, nil)
	} else {
		bal, err = a.erc20BalanceOf(ctx, client, token, address)
	}
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return nil, err
	}
	a.pool.MarkSuccess(nodeID)
	return bal, nil
}

// GetConfirmations returns how many blocks have been mined since the
// transaction's block. Returns 0 if the tx is pending.
func (a *Adapter) GetConfirmations(ctx context.Context, txHash string) (uint64, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return 0, err
	}
	if receipt == nil || receipt.BlockNumber == nil {
		return 0, nil
	}
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return 0, err
	}
	a.pool.MarkSuccess(nodeID)
	latest := header.Number.Uint64()
	txBlock := receipt.BlockNumber.Uint64()
	if latest < txBlock {
		return 0, nil
	}
	return latest - txBlock + 1, nil
}

// BroadcastTransaction submits a signed raw transaction and returns its hash.
func (a *Adapter) BroadcastTransaction(ctx context.Context, signedTx []byte) (string, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(signedTx); err != nil {
		return "", fmt.Errorf("unmarshal tx: %w", err)
	}

	if err := client.SendTransaction(ctx, tx); err != nil {
		a.pool.MarkFailure(nodeID, err)
		return "", fmt.Errorf("send transaction: %w", err)
	}
	a.pool.MarkSuccess(nodeID)
	return tx.Hash().Hex(), nil
}

// EstimateGas returns gasPrice * 21000 for a simple transfer. For contract
// calls, callers should pass gasLimit in params.Data (not yet threaded).
func (a *Adapter) EstimateGas(ctx context.Context, params blockchain.TxParams) (*big.Int, error) {
	if a.pool == nil {
		// Fallback for tests: classic 21000 * 20 gwei
		return big.NewInt(21000 * 20_000_000_000), nil
	}
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return nil, err
	}
	a.pool.MarkSuccess(nodeID)

	gasLimit := big.NewInt(21000)
	return new(big.Int).Mul(gasPrice, gasLimit), nil
}

// MonitorBlocks returns a stream of Transactions detected on this chain. See
// monitor.go for the implementation.
func (a *Adapter) MonitorBlocks(ctx context.Context, fromBlock uint64) (<-chan blockchain.Transaction, error) {
	return a.startMonitor(ctx, fromBlock)
}

// LatestBlock returns the current chain tip block number.
func (a *Adapter) LatestBlock(ctx context.Context) (uint64, error) {
	header, err := a.latestHeader(ctx)
	if err != nil {
		return 0, err
	}
	return header.Number.Uint64(), nil
}

// ParseBlock fetches a single block by number and extracts deposit candidates
// against the watched address set. Checks both native ETH transfers (tx.To in
// watched set) and ERC-20 Transfer events (log topic[2] resolves to a watched
// address).
//
// Amounts are scaled to human-readable units:
//   - Native ETH: divide by 10^18 (wei → ETH).
//   - ERC-20: divide by 10^info.Decimals using the WatchedAddressInfo metadata.
//
// EVM addresses are lowercased for lookup to handle EIP-55 checksum mismatches
// (C6). filterLogs errors are propagated so the caller can record the block as
// failed and retry (I8).
//
// Returns an empty slice when nothing matches — never nil.
func (a *Adapter) ParseBlock(ctx context.Context, blockNumber uint64, watched map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
	block, err := a.blockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("blockByNumber(%d): %w", blockNumber, err)
	}

	var txns []blockchain.Transaction
	blockHash := block.Hash().Hex()

	signer := types.LatestSignerForChainID(big.NewInt(a.chainID))

	for _, tx := range block.Transactions() {
		if tx.To() == nil {
			continue
		}
		// C6: lowercase the to-address for EVM chains before lookup.
		toHex := strings.ToLower(common.Address(*tx.To()).Hex())

		fromAddr := ""
		if sender, err := signer.Sender(tx); err == nil {
			fromAddr = strings.ToLower(sender.Hex())
		}

		// Native ETH transfer: divide wei by 10^18.
		if _, ok := watched[toHex]; ok && tx.Value() != nil && tx.Value().Sign() > 0 {
			wei := decimal.NewFromBigInt(tx.Value(), 0)
			ethAmount := wei.Div(decimal.New(1, 18))
			txns = append(txns, blockchain.Transaction{
				TxHash:      tx.Hash().Hex(),
				FromAddress: fromAddr,
				ToAddress:   toHex,
				Amount:      ethAmount,
				Token:       "",
				BlockNumber: blockNumber,
				BlockHash:   blockHash,
			})
		}
	}

	// Scan ERC-20 Transfer logs via filterLogs for the block range.
	// I8: propagate filterLogs errors so the block is retried.
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(blockNumber),
		ToBlock:   new(big.Int).SetUint64(blockNumber),
		Topics:    [][]common.Hash{{transferEventTopic}},
	}
	logs, err := a.filterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("filterLogs(block %d): %w", blockNumber, err)
	}

	for _, l := range logs {
		from, to, rawAmount, err := DecodeTransferLog(l)
		if err != nil {
			continue
		}
		// C6: lowercase decoded ERC-20 recipient address.
		toLower := strings.ToLower(to)
		info, ok := watched[toLower]
		if !ok {
			continue
		}
		tokenAddr := strings.ToLower(l.Address.Hex())

		// C3: scale raw base-unit amount by token decimals.
		// info.Decimals comes from the watched map (populated from blockchain_currencies.wallet_precision).
		decimals := info.Decimals
		if decimals == 0 {
			decimals = 18 // safe default; should be set by caller
		}
		scaled := decimal.NewFromBigInt(rawAmount, 0).Div(decimal.New(1, int32(decimals)))

		txns = append(txns, blockchain.Transaction{
			TxHash:      l.TxHash.Hex(),
			FromAddress: strings.ToLower(from),
			ToAddress:   toLower,
			Amount:      scaled,
			Token:       tokenAddr,
			BlockNumber: blockNumber,
			BlockHash:   blockHash,
		})
	}

	if txns == nil {
		txns = []blockchain.Transaction{}
	}
	return txns, nil
}

// latestHeader is a helper used by the monitor.
func (a *Adapter) latestHeader(ctx context.Context) (*types.Header, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return nil, err
	}
	a.pool.MarkSuccess(nodeID)
	return header, nil
}

// blockByNumber is a helper used by the monitor.
func (a *Adapter) blockByNumber(ctx context.Context, n uint64) (*types.Block, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(n))
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return nil, err
	}
	a.pool.MarkSuccess(nodeID)
	return block, nil
}

// filterLogs is used by ERC-20 Transfer event scanning.
func (a *Adapter) filterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return nil, err
	}
	a.pool.MarkSuccess(nodeID)
	return logs, nil
}
