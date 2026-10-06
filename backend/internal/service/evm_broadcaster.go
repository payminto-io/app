package service

import (
	"context"
	"fmt"
	"math/big"

	"github.com/payminto/payminto/backend/internal/blockchain"
)

// KeyProvider resolves the signing private key for a managed address. KeyResolver
// implements it; the interface keeps EVMBroadcaster testable with a fake.
type KeyProvider interface {
	PrivateKeyForAddress(address string) (privKey []byte, familyCode string, err error)
}

// adapterSource looks up a chain adapter by code. *blockchain.AdapterRegistry
// implements it.
type adapterSource interface {
	Get(code string) (blockchain.ChainAdapter, error)
}

// EVMBroadcaster is the shared primitive for moving EVM funds out of a managed
// address (a deposit/hot address whose key we can derive). It resolves the
// source address's private key, selects the chain's adapter, and broadcasts a
// signed transfer. Used by both the sweep path (→ cold wallet) and the
// withdrawal path (→ merchant address).
//
// It deliberately knows nothing about sweeps or withdrawals — callers own the
// state machine and persistence; this only signs+broadcasts and reports the
// result.
type EVMBroadcaster struct {
	keys     KeyProvider
	adapters adapterSource
}

// NewEVMBroadcaster constructs an EVMBroadcaster.
func NewEVMBroadcaster(keys KeyProvider, adapters adapterSource) *EVMBroadcaster {
	return &EVMBroadcaster{keys: keys, adapters: adapters}
}

// evmSignerFor resolves the chain adapter for chainCode and asserts it supports
// EVM signing.
func (b *EVMBroadcaster) evmSignerFor(chainCode string) (blockchain.EVMSigner, error) {
	adapter, err := b.adapters.Get(chainCode)
	if err != nil {
		return nil, fmt.Errorf("no adapter for chain %q: %w", chainCode, err)
	}
	signer, ok := adapter.(blockchain.EVMSigner)
	if !ok {
		return nil, fmt.Errorf("chain %q adapter does not support EVM signing", chainCode)
	}
	return signer, nil
}

// SendNative resolves the source address key and broadcasts a native-coin
// transfer of amountWei to `to`. Returns the broadcast tx hash and gas fee paid.
func (b *EVMBroadcaster) SendNative(ctx context.Context, chainCode, fromAddress, to string, amountWei *big.Int) (txHash string, gasFee *big.Int, err error) {
	signer, err := b.evmSignerFor(chainCode)
	if err != nil {
		return "", nil, err
	}
	privKey, _, err := b.keys.PrivateKeyForAddress(fromAddress)
	if err != nil {
		return "", nil, fmt.Errorf("resolve key for %s: %w", fromAddress, err)
	}
	defer zeroBytes(privKey)
	return signer.SendNative(ctx, privKey, to, amountWei)
}

// SendERC20 resolves the source address key and broadcasts an ERC-20 transfer of
// `amount` tokens to `to` on the given tokenContract.
func (b *EVMBroadcaster) SendERC20(ctx context.Context, chainCode, fromAddress, tokenContract, to string, amount *big.Int) (txHash string, gasFee *big.Int, err error) {
	signer, err := b.evmSignerFor(chainCode)
	if err != nil {
		return "", nil, err
	}
	privKey, _, err := b.keys.PrivateKeyForAddress(fromAddress)
	if err != nil {
		return "", nil, fmt.Errorf("resolve key for %s: %w", fromAddress, err)
	}
	defer zeroBytes(privKey)
	return signer.SendERC20(ctx, privKey, tokenContract, to, amount)
}

// nativeTransferGasUnits is the fixed gas a plain value transfer consumes.
const nativeTransferGasUnits = 21_000

// maxReasonableGasPriceWei caps the gas price we will act on (1000 gwei). A node
// returning anything above this is treated as an error rather than silently
// under-sweeping or burning funds during a congestion/RPC anomaly.
var maxReasonableGasPriceWei = big.NewInt(1_000_000_000_000) // 1000 gwei

// SweepNative consolidates a deposit address's entire native balance to `to`
// (the cold wallet), leaving exactly the gas cost behind. It reads the live
// balance and gas price, computes amount = balance − gasPrice*21000, and skips
// (returns sweptAmount == nil, no error) when the net is at or below dust.
//
// Returning the computed amount lets the caller record the exact swept value.
func (b *EVMBroadcaster) SweepNative(ctx context.Context, chainCode, fromAddress, to string, minSweepWei *big.Int) (txHash string, sweptAmount, gasFee *big.Int, err error) {
	signer, err := b.evmSignerFor(chainCode)
	if err != nil {
		return "", nil, nil, err
	}
	balance, err := signer.NativeBalance(ctx, fromAddress)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read balance %s: %w", fromAddress, err)
	}
	gasPrice, err := signer.SuggestGasPrice(ctx)
	if err != nil {
		return "", nil, nil, fmt.Errorf("gas price: %w", err)
	}
	if gasPrice.Sign() <= 0 || gasPrice.Cmp(maxReasonableGasPriceWei) > 0 {
		return "", nil, nil, fmt.Errorf("refusing to sweep at unreasonable gas price %s wei", gasPrice)
	}
	gasCost := new(big.Int).Mul(gasPrice, big.NewInt(nativeTransferGasUnits))
	amount := new(big.Int).Sub(balance, gasCost)

	// Skip dust / insufficient balance — not an error, just nothing to sweep.
	if amount.Sign() <= 0 || (minSweepWei != nil && amount.Cmp(minSweepWei) < 0) {
		return "", nil, nil, nil
	}

	privKey, _, err := b.keys.PrivateKeyForAddress(fromAddress)
	if err != nil {
		return "", nil, nil, fmt.Errorf("resolve key for %s: %w", fromAddress, err)
	}
	defer zeroBytes(privKey)

	hash, fee, err := signer.SendNative(ctx, privKey, to, amount)
	if err != nil {
		return "", nil, nil, err
	}
	return hash, amount, fee, nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
