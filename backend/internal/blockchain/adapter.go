package blockchain

import (
	"context"
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

// Transaction represents a normalised on-chain transfer emitted by a
// ChainAdapter's MonitorBlocks channel.
type Transaction struct {
	TxHash      string
	FromAddress string
	ToAddress   string
	Amount      decimal.Decimal
	Token       string
	BlockNumber uint64
	BlockHash   string
}

// TxParams carries the parameters required to estimate or build a transaction.
type TxParams struct {
	From   string
	To     string
	Amount *big.Int
	Token  string
	Data   []byte
}

// WatchedAddressInfo carries the metadata ParseBlock needs to emit
// correctly-scaled amounts for each watched address. Keyed by the canonical
// (chain-normalised) form of the address in the BlockchainProcessor map.
type WatchedAddressInfo struct {
	// BlockchainCurrencyID is the FK into blockchain_currencies.
	BlockchainCurrencyID uint
	// Decimals is the token's decimal precision (e.g. 18 for ETH/USDC on L1,
	// 6 for USDT). Used to scale raw base-unit amounts.
	Decimals uint
	// TokenAddress is the contract address for ERC-20/TRC-20 tokens, or empty
	// for native transfers (ETH, BTC, TRX).
	TokenAddress string
	// MinDepositAmount is the minimum accepted deposit, in human-readable units.
	// Cached here to avoid a per-tx DB lookup in handleExtractedDeposit.
	MinDepositAmount decimal.Decimal
}

// normalizeAddress returns the canonical lookup form for an address on a given
// chain. EVM addresses are lowercased because Ethereum/Base/Polygon addresses
// are case-insensitive (EIP-55 checksum is display-only). BTC and Tron
// addresses are case-sensitive and pass through unchanged.
func NormalizeAddress(chainCode, addr string) string {
	switch chainCode {
	case "ETH", "BASE", "POLYGON", "ARBITRUM", "OPTIMISM":
		return strings.ToLower(addr)
	}
	return addr
}

// ChainAdapter is the abstraction every blockchain adapter must implement.
// It covers address generation, balance queries, confirmation checks, block
// monitoring, transaction broadcast, and gas estimation.
type ChainAdapter interface {
	Name() string
	Code() string
	IsMainnet() bool
	GenerateAddress(index uint32) (string, error)
	GetBalance(ctx context.Context, address string, token string) (*big.Int, error)
	GetConfirmations(ctx context.Context, txHash string) (uint64, error)
	MonitorBlocks(ctx context.Context, fromBlock uint64) (<-chan Transaction, error)
	BroadcastTransaction(ctx context.Context, signedTx []byte) (string, error)
	EstimateGas(ctx context.Context, params TxParams) (*big.Int, error)

	// ParseBlock fetches a single block by number and extracts deposit candidates
	// against the watched address set. The map value carries token metadata so
	// amounts can be scaled to human-readable units before returning.
	// Implementations are chain-specific:
	//   EVM: walks tx.To + ERC-20 Transfer log topics
	//   BTC: scans vouts
	//   Tron: walks TransferContract entries
	// Returns an empty slice when nothing matches — never nil.
	ParseBlock(ctx context.Context, blockNumber uint64, watched map[string]WatchedAddressInfo) ([]Transaction, error)

	// LatestBlock returns the current chain tip block number.
	LatestBlock(ctx context.Context) (uint64, error)
}
