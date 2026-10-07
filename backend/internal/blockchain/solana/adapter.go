package solana

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// ChainCode is the blockchains.code and the ledger chain suffix (USDC.SOLANA).
const ChainCode = "SOLANA"

// Clusters.
const (
	ClusterMainnet = "mainnet-beta"
	ClusterDevnet  = "devnet"
	ClusterTestnet = "testnet"
	ClusterLocal   = "localnet"
)

// FinalizedConfirmations is MAX_LOCKOUT_HISTORY: a rooted slot has at least this many confirmations.
const FinalizedConfirmations = 32

// LamportsPerSignature is the cluster base fee; priority fees come on top.
const LamportsPerSignature = 5000

// Adapter implements blockchain.ChainAdapter for Solana.
type Adapter struct {
	client  *Client
	cluster string
}

// NewAdapter builds an adapter over pool for cluster.
func NewAdapter(pool *blockchain.RPCPool, cluster string) *Adapter {
	return NewAdapterWithCaller(NewPoolCaller(pool), cluster)
}

// NewAdapterWithCaller is NewAdapter with any transport (tests, local validator).
func NewAdapterWithCaller(c Caller, cluster string) *Adapter {
	if cluster == "" {
		cluster = ClusterMainnet
	}
	return &Adapter{client: NewClient(c), cluster: cluster}
}

func (a *Adapter) Name() string    { return "Solana" }
func (a *Adapter) Code() string    { return ChainCode }
func (a *Adapter) IsMainnet() bool { return a.cluster == ClusterMainnet }

// Cluster returns the configured cluster label.
func (a *Adapter) Cluster() string { return a.cluster }

// Client exposes the typed RPC client to the watcher and sweeper.
func (a *Adapter) Client() *Client { return a.client }

// GenerateAddress is not implemented; keys come from crypto.DeriveSolanaAddress through the HD wallet.
func (a *Adapter) GenerateAddress(uint32) (string, error) {
	return "", errors.New("solana: use crypto.DeriveSolanaAddress")
}

// GetBalance returns lamports for an empty token, or base units of the owner's ATA for a mint.
func (a *Adapter) GetBalance(ctx context.Context, address, token string) (*big.Int, error) {
	if token == "" || token == "SOL" {
		lamports, err := a.client.GetBalance(ctx, address, CommitmentConfirmed)
		if err != nil {
			return nil, err
		}
		return new(big.Int).SetUint64(lamports), nil
	}
	owner, err := ParsePublicKey(address)
	if err != nil {
		return nil, err
	}
	mint, err := ParsePublicKey(token)
	if err != nil {
		return nil, err
	}
	info, err := a.client.GetMint(ctx, token)
	if err != nil {
		return nil, err
	}
	ata, err := AssociatedTokenAddress(owner, mint, info.TokenProgram)
	if err != nil {
		return nil, err
	}
	bal, err := a.client.GetTokenAccountBalance(ctx, ata.String(), CommitmentConfirmed)
	if errors.Is(err, ErrAccountNotFound) {
		return big.NewInt(0), nil
	}
	if err != nil {
		return nil, err
	}
	v, ok := new(big.Int).SetString(bal.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("solana: bad token amount %q", bal.Amount)
	}
	return v, nil
}

// GetConfirmations reports the cluster's confirmation count; finalized returns FinalizedConfirmations.
func (a *Adapter) GetConfirmations(ctx context.Context, txHash string) (uint64, error) {
	statuses, err := a.client.GetSignatureStatuses(ctx, []string{txHash})
	if err != nil {
		return 0, err
	}
	if len(statuses) == 0 || statuses[0] == nil {
		return 0, nil
	}
	st := statuses[0]
	if st.Failed() {
		return 0, fmt.Errorf("solana: transaction %s failed on chain", txHash)
	}
	if st.ConfirmationStatus == CommitmentFinalized {
		return FinalizedConfirmations, nil
	}
	if st.Confirmations != nil {
		return *st.Confirmations, nil
	}
	return 0, nil
}

// MonitorBlocks is not supported; deposits are found per address by the Solana deposit watcher.
func (a *Adapter) MonitorBlocks(context.Context, uint64) (<-chan blockchain.Transaction, error) {
	return nil, errors.New("solana: block monitoring is replaced by the signature watcher")
}

// BroadcastTransaction sends an already serialized transaction.
func (a *Adapter) BroadcastTransaction(ctx context.Context, signedTx []byte) (string, error) {
	var sig string
	err := a.client.caller.Call(ctx, "sendTransaction", []any{encodeBase64(signedTx), map[string]any{
		"encoding": "base64", "preflightCommitment": CommitmentConfirmed,
	}}, &sig)
	return sig, err
}

// EstimateGas returns the base fee for one signature in lamports; priority fees are configured per sweep.
func (a *Adapter) EstimateGas(context.Context, blockchain.TxParams) (*big.Int, error) {
	return big.NewInt(LamportsPerSignature), nil
}

// LatestBlock is the confirmed slot.
func (a *Adapter) LatestBlock(ctx context.Context) (uint64, error) {
	return a.client.GetSlot(ctx, CommitmentConfirmed)
}

// ParseBlock scans one slot for SPL credits into watched token accounts. The watcher is the primary
// detection path; this exists so the generic block processor contract still holds. A skipped slot is empty.
func (a *Adapter) ParseBlock(ctx context.Context, slot uint64, watched map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
	block, err := a.client.GetBlock(ctx, slot, CommitmentConfirmed)
	if err != nil {
		return nil, fmt.Errorf("getBlock(%d): %w", slot, err)
	}
	out := []blockchain.Transaction{}
	if block == nil || len(watched) == 0 {
		return out, nil
	}
	for i := range block.Transactions {
		tx := &block.Transactions[i]
		tx.Slot = slot
		for addr, info := range watched {
			ata, err := ParsePublicKey(addr)
			if err != nil {
				continue
			}
			mint, err := ParsePublicKey(info.TokenAddress)
			if err != nil {
				continue
			}
			credit, _ := ExtractDeposits(tx, Watch{TokenAccount: ata, Mint: mint, TokenProgram: TokenProgram, Decimals: uint8(info.Decimals)})
			if credit == nil || credit.Amount.IsZero() {
				continue
			}
			out = append(out, blockchain.Transaction{
				TxHash: credit.Signature, FromAddress: credit.From, ToAddress: addr, Amount: credit.Amount,
				Token: info.TokenAddress, BlockNumber: slot, BlockHash: block.Blockhash,
			})
		}
	}
	return out, nil
}

// LamportsToSOL converts base units to a decimal SOL amount.
func LamportsToSOL(lamports uint64) decimal.Decimal {
	return decimal.NewFromBigInt(new(big.Int).SetUint64(lamports), -9)
}

var _ blockchain.ChainAdapter = (*Adapter)(nil)
