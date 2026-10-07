package chainlink

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/observability"
)

// evmClient is the slice of ethclient the reader uses; tests supply a fake.
type evmClient interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BlockNumber(ctx context.Context) (uint64, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error)
}

// EthLogReader is the LogReader over the gateway's own RPC (CRE_CHAIN_RPC_URL).
type EthLogReader struct {
	client evmClient
	// Confirmations is used when the RPC does not serve the finalized tag; zero refuses the fallback.
	Confirmations uint64
	fallbackOnce  sync.Once
}

func DialReader(ctx context.Context, rpcURL string, confirmations uint64) (*EthLogReader, error) {
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("cre: dial attestation chain: %s", cre.SanitizeError(err))
	}
	return &EthLogReader{client: c, Confirmations: confirmations}, nil
}

// NewReaderWith wraps any evmClient (tests).
func NewReaderWith(c evmClient, confirmations uint64) *EthLogReader {
	return &EthLogReader{client: c, Confirmations: confirmations}
}

// ErrNoFinality is returned when the RPC does not serve the finalized tag and no confirmations are configured;
// the poller retries later rather than treating the latest block as final.
var ErrNoFinality = errors.New("cre: RPC does not serve the finalized tag and CRE_VERIFY_CONFIRMATIONS is 0")

func (r *EthLogReader) FinalizedHead(ctx context.Context) (uint64, error) {
	h, err := r.client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if err == nil && h != nil {
		return h.Number.Uint64(), nil
	}
	if r.Confirmations == 0 {
		return 0, ErrNoFinality
	}
	r.fallbackOnce.Do(func() {
		observability.Logger().Warn("cre: RPC does not serve the finalized tag; using latest minus confirmations", "confirmations", r.Confirmations, "err", cre.SanitizeError(err))
	})
	latest, err := r.client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	if latest < r.Confirmations {
		return 0, nil
	}
	return latest - r.Confirmations, nil
}

func (r *EthLogReader) TransactionInput(ctx context.Context, txHash common.Hash) ([]byte, error) {
	tx, _, err := r.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	return tx.Data(), nil
}

func (r *EthLogReader) FilterLogs(ctx context.Context, address common.Address, from, to uint64, topics [][]common.Hash) ([]Log, error) {
	logs, err := r.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from), ToBlock: new(big.Int).SetUint64(to),
		Addresses: []common.Address{address}, Topics: topics,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Log, 0, len(logs))
	for _, l := range logs {
		out = append(out, Log{Address: l.Address, TxHash: l.TxHash, BlockNumber: l.BlockNumber, Index: l.Index, Topics: l.Topics, Data: l.Data, Removed: l.Removed})
	}
	return out, nil
}
