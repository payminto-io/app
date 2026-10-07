// Package bitcoin implements the ChainAdapter for Bitcoin. It uses a minimal
// JSON-RPC client (net/http) against Bitcoin Core for block queries, UTXO
// scanning, and broadcast. The network (mainnet / testnet3 / signet / regtest)
// is determined by chaincfg params passed at construction.
package bitcoin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// Adapter is the Bitcoin ChainAdapter implementation.
type Adapter struct {
	pool       *blockchain.RPCPool
	network    *chaincfg.Params
	httpClient *http.Client
}

// NewAdapter creates a Bitcoin adapter for the given network. If pool is nil
// the adapter is in test-only mode — Name/Code/IsMainnet still work.
func NewAdapter(pool *blockchain.RPCPool, network *chaincfg.Params) *Adapter {
	if network == nil {
		network = &chaincfg.MainNetParams
	}
	return &Adapter{
		pool:       pool,
		network:    network,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements ChainAdapter and returns the human-readable chain name.
func (a *Adapter) Name() string { return "Bitcoin" }

// Code implements ChainAdapter and returns the short chain code used as map keys.
func (a *Adapter) Code() string { return "BTC" }

// IsMainnet returns true when the adapter is configured with
// chaincfg.MainNetParams. Testnet3, signet, and regtest all return false.
func (a *Adapter) IsMainnet() bool {
	if a.network == nil {
		return false
	}
	return a.network.Name == chaincfg.MainNetParams.Name
}

// Network returns the configured chaincfg params.
func (a *Adapter) Network() *chaincfg.Params { return a.network }

// GenerateAddress is not implemented here — use crypto.DeriveBtcAddress from
// the HD wallet.
func (a *Adapter) GenerateAddress(index uint32) (string, error) {
	return "", errors.New("use crypto.DeriveBtcAddress for BTC")
}

// GetBalance queries the on-chain balance for an address via
// scantxoutset. token must be empty (BTC has no ERC-20 analog; TRC-20 etc.
// are other chains).
//
// C2: total_amount is decoded as json.Number to avoid float64 precision loss.
func (a *Adapter) GetBalance(ctx context.Context, address, token string) (*big.Int, error) {
	if token != "" && token != "BTC" {
		return nil, fmt.Errorf("bitcoin has no %s token", token)
	}
	// C2: use json.Number for the amount field to avoid float64 precision loss.
	var result struct {
		Success      bool        `json:"success"`
		TotalAmount  json.Number `json:"total_amount"`
		TxOutSetInfo struct {
			TotalAmount json.Number `json:"total_amount"`
		} `json:"txoutsetinfo"`
	}
	if err := a.call(ctx, "scantxoutset", []any{"start", []string{"addr(" + address + ")"}}, &result); err != nil {
		return nil, err
	}
	// scantxoutset returns total_amount in BTC; convert to satoshi via decimal.
	totalStr := string(result.TotalAmount)
	if totalStr == "" || totalStr == "0" {
		totalStr = string(result.TxOutSetInfo.TotalAmount)
	}
	if totalStr == "" {
		return big.NewInt(0), nil
	}
	totalBTC, err := decimal.NewFromString(totalStr)
	if err != nil {
		return big.NewInt(0), nil
	}
	// 1 BTC = 1e8 satoshi
	satoshi := totalBTC.Mul(decimal.New(1, 8)).BigInt()
	return satoshi, nil
}

// GetConfirmations returns how many confirmations a transaction has.
func (a *Adapter) GetConfirmations(ctx context.Context, txHash string) (uint64, error) {
	var result struct {
		Confirmations uint64 `json:"confirmations"`
	}
	if err := a.call(ctx, "getrawtransaction", []any{txHash, true}, &result); err != nil {
		return 0, err
	}
	return result.Confirmations, nil
}

// BroadcastTransaction submits a raw signed tx (hex-encoded) and returns its
// txid.
func (a *Adapter) BroadcastTransaction(ctx context.Context, signedTx []byte) (string, error) {
	hexTx := hex.EncodeToString(signedTx)
	var txid string
	if err := a.call(ctx, "sendrawtransaction", []any{hexTx}, &txid); err != nil {
		return "", err
	}
	return txid, nil
}

// EstimateGas returns the current sat/byte fee rate times an assumed 250-byte
// transaction. Callers who need a precise estimate should construct the tx
// first, then call again with the actual size.
func (a *Adapter) EstimateGas(ctx context.Context, params blockchain.TxParams) (*big.Int, error) {
	if a.pool == nil {
		return big.NewInt(10000), nil // 10k sats default
	}
	var result struct {
		FeeRate float64 `json:"feerate"`
	}
	if err := a.call(ctx, "estimatesmartfee", []any{6}, &result); err != nil {
		return big.NewInt(10000), nil
	}
	// feerate is BTC/kB; convert to sat/byte
	satPerByte := result.FeeRate * 1e8 / 1000
	if satPerByte < 1 {
		satPerByte = 1
	}
	totalSats := int64(satPerByte * 250)
	return big.NewInt(totalSats), nil
}

// MonitorBlocks polls new blocks via getblockcount + getblock. See monitor.go.
func (a *Adapter) MonitorBlocks(ctx context.Context, fromBlock uint64) (<-chan blockchain.Transaction, error) {
	return a.startMonitor(ctx, fromBlock)
}

// LatestBlock returns the current chain tip block height via getblockcount.
func (a *Adapter) LatestBlock(ctx context.Context) (uint64, error) {
	var height uint64
	if err := a.call(ctx, "getblockcount", []any{}, &height); err != nil {
		return 0, fmt.Errorf("getblockcount: %w", err)
	}
	return height, nil
}

// btcVout is the JSON shape of a Bitcoin transaction output.
// C2: Value uses json.Number to avoid float64 precision loss.
type btcVout struct {
	Value        json.Number `json:"value"`
	ScriptPubKey struct {
		Address   string   `json:"address"`
		Addresses []string `json:"addresses"`
	} `json:"scriptPubKey"`
}

// ParseBlock fetches a single block by number and extracts deposit candidates
// against the watched address set. Walks vouts looking for matching
// scriptPubKey.address entries.
//
// C2: vout.Value is decoded as json.Number and converted via decimal.NewFromString
// to avoid float64 precision loss on large amounts.
//
// The watched map is keyed by address strings as-is (BTC addresses are
// case-sensitive; bitcoind returns lowercase bech32 and mixed-case legacy).
// Callers should not normalise BTC addresses.
//
// Returns an empty slice when nothing matches — never nil.
func (a *Adapter) ParseBlock(ctx context.Context, blockNumber uint64, watched map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
	var hash string
	if err := a.call(ctx, "getblockhash", []any{blockNumber}, &hash); err != nil {
		return nil, fmt.Errorf("getblockhash(%d): %w", blockNumber, err)
	}

	var block struct {
		Hash string `json:"hash"`
		Tx   []struct {
			TxID string    `json:"txid"`
			Vout []btcVout `json:"vout"`
		} `json:"tx"`
	}
	if err := a.call(ctx, "getblock", []any{hash, 2}, &block); err != nil {
		return nil, fmt.Errorf("getblock(%s): %w", hash, err)
	}

	var txns []blockchain.Transaction
	for _, tx := range block.Tx {
		for _, vout := range tx.Vout {
			addr := vout.ScriptPubKey.Address
			if addr == "" && len(vout.ScriptPubKey.Addresses) > 0 {
				addr = vout.ScriptPubKey.Addresses[0]
			}
			if addr == "" {
				continue
			}
			if _, ok := watched[addr]; !ok {
				continue
			}
			// C2: parse via decimal to preserve precision.
			btc, err := decimal.NewFromString(vout.Value.String())
			if err != nil {
				continue
			}
			txns = append(txns, blockchain.Transaction{
				TxHash:      tx.TxID,
				ToAddress:   addr,
				Amount:      btc,
				Token:       "",
				BlockNumber: blockNumber,
				BlockHash:   hash,
			})
		}
	}

	if txns == nil {
		txns = []blockchain.Transaction{}
	}
	return txns, nil
}

// ----- Minimal JSON-RPC client -----

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
	ID     int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message) }

// call picks a node from the pool, POSTs a JSON-RPC request, and unmarshals
// the result into `out`.
func (a *Adapter) call(ctx context.Context, method string, params []any, out any) error {
	if a.pool == nil {
		return errors.New("rpc pool not configured")
	}
	node, err := a.pool.Pick()
	if err != nil {
		return err
	}

	body, _ := json.Marshal(rpcRequest{JSONRPC: "1.0", ID: 1, Method: method, Params: params})
	req, err := http.NewRequestWithContext(ctx, "POST", node.URL, bytes.NewReader(body))
	if err != nil {
		a.pool.MarkFailure(node.ID, err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if node.AuthHeader != nil && *node.AuthHeader != "" {
		req.Header.Set("Authorization", *node.AuthHeader)
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

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		a.pool.MarkFailure(node.ID, err)
		return err
	}
	if rpcResp.Error != nil {
		return rpcResp.Error
	}

	a.pool.MarkSuccess(node.ID)
	if out != nil && len(rpcResp.Result) > 0 {
		// If out is a *string, strip surrounding quotes
		if sp, ok := out.(*string); ok {
			*sp = strings.Trim(string(rpcResp.Result), `"`)
			return nil
		}
		return json.Unmarshal(rpcResp.Result, out)
	}
	return nil
}
