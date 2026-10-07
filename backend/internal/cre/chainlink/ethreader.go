package chainlink

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// EthLogReader is the LogReader over the gateway's own RPC (CRE_CHAIN_RPC_URL).
type EthLogReader struct {
	client *ethclient.Client
	// Confirmations is used when the RPC does not serve the finalized tag.
	Confirmations uint64
}

func DialReader(ctx context.Context, rpcURL string, confirmations uint64) (*EthLogReader, error) {
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("cre: dial attestation chain: %w", err)
	}
	return &EthLogReader{client: c, Confirmations: confirmations}, nil
}

func (r *EthLogReader) FinalizedHead(ctx context.Context) (uint64, error) {
	if h, err := r.client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64())); err == nil && h != nil {
		return h.Number.Uint64(), nil
	}
	latest, err := r.client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	if latest < r.Confirmations {
		return 0, nil
	}
	return latest - r.Confirmations, nil
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
		if l.Removed {
			continue
		}
		out = append(out, Log{Address: l.Address, TxHash: l.TxHash, BlockNumber: l.BlockNumber, Index: l.Index, Topics: l.Topics, Data: l.Data})
	}
	return out, nil
}
