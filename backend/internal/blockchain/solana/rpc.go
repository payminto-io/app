package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
)

// Commitment levels; "confirmed" is seen, "finalized" is credited (ticket 09 detection decision).
const (
	CommitmentProcessed = "processed"
	CommitmentConfirmed = "confirmed"
	CommitmentFinalized = "finalized"
)

// Caller issues one JSON-RPC call. PoolCaller is the production implementation; tests use fixtures.
type Caller interface {
	Call(ctx context.Context, method string, params []any, out any) error
}

// RPCError is a JSON-RPC error object returned by the node.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return fmt.Sprintf("solana rpc %d: %s", e.Code, e.Message) }

// PoolCaller routes JSON-RPC calls through the blockchain.RPCPool, marking node health per call.
type PoolCaller struct {
	pool       *blockchain.RPCPool
	httpClient *http.Client
}

// NewPoolCaller builds a Caller over pool.
func NewPoolCaller(pool *blockchain.RPCPool) *PoolCaller {
	return &PoolCaller{pool: pool, httpClient: &http.Client{Timeout: 30 * time.Second}}
}

// StaticCaller targets one URL; used by integration tests and local validators.
type StaticCaller struct {
	URL        string
	HTTPClient *http.Client
}

func (s *StaticCaller) Call(ctx context.Context, method string, params []any, out any) error {
	c := s.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 30 * time.Second}
	}
	return doCall(ctx, c, s.URL, "", method, params, out)
}

func (p *PoolCaller) Call(ctx context.Context, method string, params []any, out any) error {
	if p.pool == nil {
		return errors.New("solana: rpc pool not configured")
	}
	node, err := p.pool.Pick()
	if err != nil {
		return err
	}
	auth := ""
	if node.AuthHeader != nil {
		auth = *node.AuthHeader
	}
	err = doCall(ctx, p.httpClient, node.URL, auth, method, params, out)
	var rpcErr *RPCError
	switch {
	case err == nil, errors.As(err, &rpcErr):
		// A JSON-RPC error is the node answering; it is not a node failure.
		p.pool.MarkSuccess(node.ID)
	default:
		p.pool.MarkFailure(node.ID, err)
	}
	return err
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params,omitempty"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

func doCall(ctx context.Context, client *http.Client, url, auth, method string, params []any, out any) error {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(url, "/"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("solana rpc http %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	var r rpcResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("solana rpc: decode %s: %w", method, err)
	}
	if r.Error != nil {
		return r.Error
	}
	if out == nil {
		return nil
	}
	if len(r.Result) == 0 || string(r.Result) == "null" {
		return nil
	}
	return json.Unmarshal(r.Result, out)
}

// Client exposes the typed RPC surface the adapter, watcher and sweeper use.
type Client struct{ caller Caller }

// NewClient wraps any Caller.
func NewClient(c Caller) *Client { return &Client{caller: c} }

// Caller returns the underlying transport.
func (c *Client) Caller() Caller { return c.caller }

type contextValue[T any] struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value T `json:"value"`
}

// GetSlot returns the current slot at the commitment.
func (c *Client) GetSlot(ctx context.Context, commitment string) (uint64, error) {
	var slot uint64
	err := c.caller.Call(ctx, "getSlot", []any{map[string]any{"commitment": commitment}}, &slot)
	return slot, err
}

// GetBlockHeight returns the block height at the commitment; blockhash validity is in block heights.
func (c *Client) GetBlockHeight(ctx context.Context, commitment string) (uint64, error) {
	var h uint64
	err := c.caller.Call(ctx, "getBlockHeight", []any{map[string]any{"commitment": commitment}}, &h)
	return h, err
}

// Blockhash is a recent blockhash and the last block height it is valid for.
type Blockhash struct {
	Blockhash            string `json:"blockhash"`
	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
}

// GetLatestBlockhash fetches a blockhash at the commitment.
func (c *Client) GetLatestBlockhash(ctx context.Context, commitment string) (Blockhash, error) {
	var out contextValue[Blockhash]
	err := c.caller.Call(ctx, "getLatestBlockhash", []any{map[string]any{"commitment": commitment}}, &out)
	return out.Value, err
}

// SignatureInfo is one entry of getSignaturesForAddress, newest first.
type SignatureInfo struct {
	Signature          string          `json:"signature"`
	Slot               uint64          `json:"slot"`
	Err                json.RawMessage `json:"err"`
	BlockTime          *int64          `json:"blockTime"`
	ConfirmationStatus string          `json:"confirmationStatus"`
}

// Failed reports whether the transaction errored on chain.
func (s SignatureInfo) Failed() bool { return len(s.Err) > 0 && string(s.Err) != "null" }

// GetSignaturesForAddress lists signatures touching address, newest first, stopping at until (exclusive).
func (c *Client) GetSignaturesForAddress(ctx context.Context, address string, until string, limit int, commitment string) ([]SignatureInfo, error) {
	return c.GetSignaturesForAddressPage(ctx, address, until, "", limit, commitment)
}

// GetSignaturesForAddressPage is GetSignaturesForAddress starting below `before` for paging.
func (c *Client) GetSignaturesForAddressPage(ctx context.Context, address, until, before string, limit int, commitment string) ([]SignatureInfo, error) {
	opts := map[string]any{"commitment": commitment, "limit": limit}
	if until != "" {
		opts["until"] = until
	}
	if before != "" {
		opts["before"] = before
	}
	var out []SignatureInfo
	err := c.caller.Call(ctx, "getSignaturesForAddress", []any{address, opts}, &out)
	return out, err
}

// SignatureStatus is one entry of getSignatureStatuses; nil means the node does not know the signature.
type SignatureStatus struct {
	Slot               uint64          `json:"slot"`
	Confirmations      *uint64         `json:"confirmations"`
	Err                json.RawMessage `json:"err"`
	ConfirmationStatus string          `json:"confirmationStatus"`
}

// Failed reports an on-chain error.
func (s SignatureStatus) Failed() bool { return len(s.Err) > 0 && string(s.Err) != "null" }

// GetSignatureStatuses looks signatures up, searching the ledger history too.
func (c *Client) GetSignatureStatuses(ctx context.Context, signatures []string) ([]*SignatureStatus, error) {
	var out contextValue[[]*SignatureStatus]
	err := c.caller.Call(ctx, "getSignatureStatuses", []any{signatures, map[string]any{"searchTransactionHistory": true}}, &out)
	return out.Value, err
}

// GetTransaction fetches a transaction with jsonParsed encoding; nil when the node has no record of it.
func (c *Client) GetTransaction(ctx context.Context, signature, commitment string) (*ParsedTransaction, error) {
	var out *ParsedTransaction
	err := c.caller.Call(ctx, "getTransaction", []any{signature, map[string]any{
		"encoding":                       "jsonParsed",
		"commitment":                     commitment,
		"maxSupportedTransactionVersion": 0,
	}}, &out)
	return out, err
}

// ParsedBlock is getBlock with jsonParsed transactions; used by ParseBlock.
type ParsedBlock struct {
	Blockhash    string              `json:"blockhash"`
	ParentSlot   uint64              `json:"parentSlot"`
	BlockHeight  *uint64             `json:"blockHeight"`
	Transactions []ParsedTransaction `json:"transactions"`
}

// GetBlock fetches a slot's block; nil when the slot was skipped.
func (c *Client) GetBlock(ctx context.Context, slot uint64, commitment string) (*ParsedBlock, error) {
	var out *ParsedBlock
	err := c.caller.Call(ctx, "getBlock", []any{slot, map[string]any{
		"encoding":                       "jsonParsed",
		"commitment":                     commitment,
		"transactionDetails":             "full",
		"rewards":                        false,
		"maxSupportedTransactionVersion": 0,
	}}, &out)
	return out, err
}

// SendTransaction submits a signed transaction; preflight runs at the confirmed commitment.
func (c *Client) SendTransaction(ctx context.Context, tx Transaction, skipPreflight bool, maxRetries uint) (string, error) {
	var sig string
	err := c.caller.Call(ctx, "sendTransaction", []any{tx.Base64(), map[string]any{
		"encoding":            "base64",
		"skipPreflight":       skipPreflight,
		"preflightCommitment": CommitmentConfirmed,
		"maxRetries":          maxRetries,
	}}, &sig)
	return sig, err
}

// GetBalance returns lamports.
func (c *Client) GetBalance(ctx context.Context, address, commitment string) (uint64, error) {
	var out contextValue[uint64]
	err := c.caller.Call(ctx, "getBalance", []any{address, map[string]any{"commitment": commitment}}, &out)
	return out.Value, err
}

// TokenAmount is the RPC's token quantity shape.
type TokenAmount struct {
	Amount         string `json:"amount"`
	Decimals       uint8  `json:"decimals"`
	UIAmountString string `json:"uiAmountString"`
}

// GetTokenAccountBalance returns a token account's balance; ErrAccountNotFound when it does not exist.
func (c *Client) GetTokenAccountBalance(ctx context.Context, tokenAccount, commitment string) (TokenAmount, error) {
	var out contextValue[TokenAmount]
	err := c.caller.Call(ctx, "getTokenAccountBalance", []any{tokenAccount, map[string]any{"commitment": commitment}}, &out)
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) && strings.Contains(rpcErr.Message, "could not find account") {
		return TokenAmount{}, ErrAccountNotFound
	}
	return out.Value, err
}

// ErrAccountNotFound is returned for accounts the chain has no record of.
var ErrAccountNotFound = errors.New("solana: account not found")

// AccountInfo is getAccountInfo with jsonParsed data.
type AccountInfo struct {
	Lamports uint64          `json:"lamports"`
	Owner    string          `json:"owner"`
	Data     json.RawMessage `json:"data"`
}

// GetAccountInfo returns nil when the account does not exist.
func (c *Client) GetAccountInfo(ctx context.Context, address, commitment string) (*AccountInfo, error) {
	var out contextValue[*AccountInfo]
	err := c.caller.Call(ctx, "getAccountInfo", []any{address, map[string]any{"commitment": commitment, "encoding": "jsonParsed"}}, &out)
	return out.Value, err
}

// MintInfo is the parsed data of a mint account.
type MintInfo struct {
	Decimals     uint8
	TokenProgram PublicKey
}

// GetMint reads a mint's decimals and owning token program; the watcher refuses mints that do not match the seed.
func (c *Client) GetMint(ctx context.Context, mint string) (MintInfo, error) {
	info, err := c.GetAccountInfo(ctx, mint, CommitmentConfirmed)
	if err != nil {
		return MintInfo{}, err
	}
	if info == nil {
		return MintInfo{}, ErrAccountNotFound
	}
	owner, err := ParsePublicKey(info.Owner)
	if err != nil {
		return MintInfo{}, err
	}
	var parsed struct {
		Parsed struct {
			Type string `json:"type"`
			Info struct {
				Decimals uint8 `json:"decimals"`
			} `json:"info"`
		} `json:"parsed"`
	}
	if err := json.Unmarshal(info.Data, &parsed); err != nil || parsed.Parsed.Type != "mint" {
		return MintInfo{}, fmt.Errorf("solana: %s is not a mint account", mint)
	}
	return MintInfo{Decimals: parsed.Parsed.Info.Decimals, TokenProgram: owner}, nil
}

// GetMinimumBalanceForRentExemption returns lamports needed for an account of size bytes.
func (c *Client) GetMinimumBalanceForRentExemption(ctx context.Context, size uint64) (uint64, error) {
	var out uint64
	err := c.caller.Call(ctx, "getMinimumBalanceForRentExemption", []any{size}, &out)
	return out, err
}

// RequestAirdrop is only honoured by devnet and local validators.
func (c *Client) RequestAirdrop(ctx context.Context, address string, lamports uint64) (string, error) {
	var sig string
	err := c.caller.Call(ctx, "requestAirdrop", []any{address, lamports, map[string]any{"commitment": CommitmentConfirmed}}, &sig)
	return sig, err
}

// GetGenesisHash identifies the cluster.
func (c *Client) GetGenesisHash(ctx context.Context) (string, error) {
	var h string
	err := c.caller.Call(ctx, "getGenesisHash", nil, &h)
	return h, err
}

// GetHealth is "ok" when the node is caught up.
func (c *Client) GetHealth(ctx context.Context) (string, error) {
	var h string
	err := c.caller.Call(ctx, "getHealth", nil, &h)
	return h, err
}
