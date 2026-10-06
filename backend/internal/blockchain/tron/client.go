// Package tron implements the ChainAdapter for Tron. It uses TronGrid's
// REST HTTP API over the Payminto RPCPool for multi-node failover.
package tron

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// Known mainnet genesis block hash (used by IsMainnet if passed in).
// In production we derive this from the chain's genesis on first boot
// rather than hardcoding it.
// TronMainnetGenesisHash is the genesis block hash for Tron mainnet. Used as a
// network identifier when fingerprinting live nodes.
const TronMainnetGenesisHash = "00000000000000001ebf88508a03865c71d452e25f4d51194196a1d22b6653dc"

// Adapter is the Tron ChainAdapter implementation.
type Adapter struct {
	pool       *blockchain.RPCPool
	network    string // "mainnet" | "nile" | "shasta" | "regtest"
	httpClient *http.Client
}

// NewAdapter creates a Tron adapter. `network` is one of "mainnet", "nile",
// "shasta", "regtest".
func NewAdapter(pool *blockchain.RPCPool, network string) *Adapter {
	if network == "" {
		network = "mainnet"
	}
	return &Adapter{
		pool:       pool,
		network:    network,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements ChainAdapter and returns the human-readable chain name.
func (a *Adapter) Name() string { return "Tron" }

// Code implements ChainAdapter and returns the short chain code used as map keys.
func (a *Adapter) Code() string { return "TRX" }

// IsMainnet returns true when the adapter's network is "mainnet".
// Testnet variants (nile, shasta, regtest) return false.
func (a *Adapter) IsMainnet() bool {
	return a.network == "mainnet"
}

// Network returns the configured network label.
func (a *Adapter) Network() string { return a.network }

// GenerateAddress is not implemented — use crypto.DeriveTronAddress.
func (a *Adapter) GenerateAddress(index uint32) (string, error) {
	return "", errors.New("use crypto.DeriveTronAddress for Tron")
}

// GetBalance queries an account's TRX or TRC-20 balance. Empty token =>
// native TRX (in sun). Non-empty token is the hex-encoded contract address
// of a TRC-20 token (e.g., USDT).
func (a *Adapter) GetBalance(ctx context.Context, address, token string) (*big.Int, error) {
	if token == "" || token == "TRX" {
		return a.trxBalance(ctx, address)
	}
	return a.trc20Balance(ctx, token, address)
}

func (a *Adapter) trxBalance(ctx context.Context, address string) (*big.Int, error) {
	var result struct {
		Balance int64 `json:"balance"`
	}
	if err := a.post(ctx, "/wallet/getaccount", map[string]any{
		"address": address,
		"visible": true,
	}, &result); err != nil {
		return nil, err
	}
	return new(big.Int).SetInt64(result.Balance), nil
}

// trc20Balance calls balanceOf(address) on a TRC-20 contract via
// triggerConstantContract. The result is a 32-byte big-endian value in the
// `constant_result` hex string.
func (a *Adapter) trc20Balance(ctx context.Context, tokenAddress, holderAddress string) (*big.Int, error) {
	// Parameter format: the address padded to 64 hex chars
	// strip the 0x41 prefix (Tron uses 0x41 as the mainnet address byte)
	holderHex := strings.TrimPrefix(holderAddress, "0x")
	// If it's a base58 T-address, the caller should convert first — for simplicity
	// we accept already-hex addresses here.
	padded := strings.Repeat("0", 64-len(holderHex)) + holderHex

	params := map[string]any{
		"owner_address":     holderAddress,
		"contract_address":  tokenAddress,
		"function_selector": "balanceOf(address)",
		"parameter":         padded,
		"visible":           true,
	}

	var result struct {
		ConstantResult []string `json:"constant_result"`
	}
	if err := a.post(ctx, "/wallet/triggerconstantcontract", params, &result); err != nil {
		return nil, err
	}
	if len(result.ConstantResult) == 0 {
		return big.NewInt(0), nil
	}
	raw, err := hex.DecodeString(result.ConstantResult[0])
	if err != nil {
		return nil, fmt.Errorf("decode constant_result: %w", err)
	}
	return new(big.Int).SetBytes(raw), nil
}

// GetConfirmations returns how many blocks have been confirmed since the tx.
func (a *Adapter) GetConfirmations(ctx context.Context, txHash string) (uint64, error) {
	var txInfo struct {
		BlockNumber int64 `json:"blockNumber"`
	}
	if err := a.post(ctx, "/wallet/gettransactioninfobyid", map[string]any{
		"value": txHash,
	}, &txInfo); err != nil {
		return 0, err
	}
	if txInfo.BlockNumber == 0 {
		return 0, nil
	}

	var latest struct {
		BlockHeader struct {
			RawData struct {
				Number int64 `json:"number"`
			} `json:"raw_data"`
		} `json:"block_header"`
	}
	if err := a.post(ctx, "/wallet/getnowblock", nil, &latest); err != nil {
		return 0, err
	}
	latestNum := latest.BlockHeader.RawData.Number
	if latestNum < txInfo.BlockNumber {
		return 0, nil
	}
	return uint64(latestNum - txInfo.BlockNumber + 1), nil
}

// BroadcastTransaction submits a signed transaction hex and returns its txid.
func (a *Adapter) BroadcastTransaction(ctx context.Context, signedTx []byte) (string, error) {
	hexTx := hex.EncodeToString(signedTx)
	var result struct {
		Result bool   `json:"result"`
		TxID   string `json:"txid"`
	}
	if err := a.post(ctx, "/wallet/broadcasthex", map[string]any{
		"transaction": hexTx,
	}, &result); err != nil {
		return "", err
	}
	if !result.Result {
		return "", errors.New("tron broadcast rejected")
	}
	return result.TxID, nil
}

// EstimateGas is a no-op for Tron — it uses energy + bandwidth, not gas.
// Returns 0 so callers can detect the chain doesn't support gas estimates.
func (a *Adapter) EstimateGas(ctx context.Context, params blockchain.TxParams) (*big.Int, error) {
	return big.NewInt(0), nil
}

// MonitorBlocks polls getnowblock + getblockbynum. See monitor.go.
func (a *Adapter) MonitorBlocks(ctx context.Context, fromBlock uint64) (<-chan blockchain.Transaction, error) {
	return a.startMonitor(ctx, fromBlock)
}

// LatestBlock returns the current Tron chain tip block number via getnowblock.
func (a *Adapter) LatestBlock(ctx context.Context) (uint64, error) {
	var latest struct {
		BlockHeader struct {
			RawData struct {
				Number int64 `json:"number"`
			} `json:"raw_data"`
		} `json:"block_header"`
	}
	if err := a.post(ctx, "/wallet/getnowblock", nil, &latest); err != nil {
		return 0, fmt.Errorf("getnowblock: %w", err)
	}
	return uint64(latest.BlockHeader.RawData.Number), nil
}

// tronContract is the JSON shape of a Tron transaction contract entry.
// C1: amount is decoded as json.RawMessage so we can parse it as json.Number,
// avoiding float64 precision loss for large TRX amounts.
type tronContract struct {
	Parameter struct {
		Value struct {
			ToAddress    string          `json:"to_address"`
			OwnerAddress string          `json:"owner_address"`
			Amount       json.RawMessage `json:"amount"` // C1: raw to preserve precision
		} `json:"value"`
	} `json:"parameter"`
	Type string `json:"type"`
}

// ParseBlock fetches a single Tron block by number and extracts deposit
// candidates against the watched address set. Walks
// transactions[].raw_data.contract[] for TransferContract types where
// to_address matches a watched address.
//
// C1: amount is parsed via json.Number → decimal to avoid float64 precision
// loss. TRX amounts from the API are in sun (1 TRX = 10^6 sun); we divide
// by 10^6 to get human-readable TRX.
//
// TODO(phase-i-trc20): TriggerSmartContract entries (TRC-20 USDT etc.) are
// not yet parsed. They are silently skipped here. Phase I will add TRC-20
// log decoding analogous to the ETH ERC-20 path.
//
// Tron addresses are base58 + case-sensitive; no normalisation is applied.
//
// Returns an empty slice when nothing matches — never nil.
func (a *Adapter) ParseBlock(ctx context.Context, blockNumber uint64, watched map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
	var block struct {
		BlockID      string `json:"blockID"`
		Transactions []struct {
			TxID    string `json:"txID"`
			RawData struct {
				Contract []tronContract `json:"contract"`
			} `json:"raw_data"`
		} `json:"transactions"`
	}
	if err := a.post(ctx, "/wallet/getblockbynum", map[string]any{"num": blockNumber}, &block); err != nil {
		return nil, fmt.Errorf("getblockbynum(%d): %w", blockNumber, err)
	}

	var txns []blockchain.Transaction
	for _, tx := range block.Transactions {
		for _, contract := range tx.RawData.Contract {
			switch contract.Type {
			case "TransferContract":
				to := contract.Parameter.Value.ToAddress
				from := contract.Parameter.Value.OwnerAddress

				if _, ok := watched[to]; !ok {
					continue
				}

				// C1: parse amount as json.Number to avoid float64 precision loss.
				var amountNum json.Number
				if err := json.Unmarshal(contract.Parameter.Value.Amount, &amountNum); err != nil {
					continue
				}
				amountDec, err := decimal.NewFromString(amountNum.String())
				if err != nil {
					continue
				}
				// TRX amounts in sun (10^6 per TRX); divide to get human-readable TRX.
				trxAmount := amountDec.Div(decimal.New(1, 6))

				txns = append(txns, blockchain.Transaction{
					TxHash:      tx.TxID,
					FromAddress: from,
					ToAddress:   to,
					Amount:      trxAmount,
					Token:       "",
					BlockNumber: blockNumber,
					BlockHash:   block.BlockID,
				})

			case "TriggerSmartContract":
				// TODO(phase-i-trc20): decode TRC-20 Transfer events here.
				// The contract address is in contract.Parameter.Value and the
				// transfer log data follows the ABI-encoded (address,uint256) pattern.
				// For now, skip silently to avoid false positives.

			default:
				// All other contract types (freeze, vote, etc.) are not deposits.
			}
		}
	}

	if txns == nil {
		txns = []blockchain.Transaction{}
	}
	return txns, nil
}

// ----- Minimal REST client -----

func (a *Adapter) post(ctx context.Context, path string, payload any, out any) error {
	if a.pool == nil {
		return errors.New("rpc pool not configured")
	}
	node, err := a.pool.Pick()
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(node.URL, "/")+path, body)
	if err != nil {
		a.pool.MarkFailure(node.ID, err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if node.AuthHeader != nil && *node.AuthHeader != "" {
		req.Header.Set("TRON-PRO-API-KEY", *node.AuthHeader)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.pool.MarkFailure(node.ID, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		err := fmt.Errorf("http %d", resp.StatusCode)
		a.pool.MarkFailure(node.ID, err)
		return err
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			a.pool.MarkFailure(node.ID, err)
			return err
		}
	}
	a.pool.MarkSuccess(node.ID)
	return nil
}
