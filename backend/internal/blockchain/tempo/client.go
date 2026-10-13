package tempo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client is the minimal Tempo RPC surface the watcher needs: head, logs and receipt status.
// It never fetches blocks (type 0x76 transactions do not decode in go-ethereum).
type Client struct {
	eth *ethclient.Client
}

// Dial connects to a Tempo JSON-RPC endpoint.
func Dial(ctx context.Context, url string) (*Client, error) {
	c, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("tempo: dial %s: %w", url, err)
	}
	return &Client{eth: c}, nil
}

// ChainID returns the endpoint's chain id.
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) { return c.eth.ChainID(ctx) }

// Head returns the latest block number (final on Tempo).
func (c *Client) Head(ctx context.Context) (uint64, error) { return c.eth.BlockNumber(ctx) }

// MemoTransfersTo returns TransferWithMemo logs of the given tokens to recipient in [from, to].
func (c *Client) MemoTransfersTo(ctx context.Context, tokens []common.Address, recipient common.Address, from, to uint64) ([]MemoTransfer, error) {
	logs, err := c.eth.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Addresses: tokens,
		Topics:    [][]common.Hash{{TransferWithMemoTopic}, nil, {common.BytesToHash(recipient.Bytes())}},
	})
	if err != nil {
		return nil, fmt.Errorf("tempo: getLogs %d-%d: %w", from, to, err)
	}
	out := make([]MemoTransfer, 0, len(logs))
	for _, l := range logs {
		t, err := DecodeLog(l)
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	return Dedupe(out), nil
}

// ReceiptStatus returns the receipt status of a transaction (1 = success).
func (c *Client) ReceiptStatus(ctx context.Context, hash common.Hash) (uint64, error) {
	var r struct {
		Status string `json:"status"`
	}
	// Raw call: the receipt "type" of a Tempo transaction is not one go-ethereum's typed decoder needs.
	if err := c.eth.Client().CallContext(ctx, &r, "eth_getTransactionReceipt", hash); err != nil {
		return 0, fmt.Errorf("tempo: receipt %s: %w", hash, err)
	}
	if r.Status == "" {
		return 0, fmt.Errorf("tempo: receipt %s not found", hash)
	}
	v, ok := new(big.Int).SetString(trim0x(r.Status), 16)
	if !ok {
		return 0, fmt.Errorf("tempo: receipt %s status %q", hash, r.Status)
	}
	return v.Uint64(), nil
}

// Close releases the connection.
func (c *Client) Close() { c.eth.Close() }

func trim0x(s string) string {
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		return s[2:]
	}
	return s
}
