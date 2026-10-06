package blockchain

import (
	"context"
	"math/big"
)

// EVMSigner is an optional capability implemented by EVM chain adapters
// (Ethereum, Base, Polygon, …). Services obtain it by type-asserting a
// ChainAdapter from the AdapterRegistry:
//
//	if s, ok := adapter.(blockchain.EVMSigner); ok { ... }
//
// It builds, signs, and broadcasts transfers using a caller-supplied private
// key (resolved from the secrets vault), returning the broadcast tx hash and
// the gas fee paid. Keeping signing behind this interface lets the withdrawal
// and sweep services move funds without depending on the concrete ethereum
// package.
type EVMSigner interface {
	// SendNative signs and broadcasts a native-coin value transfer.
	SendNative(ctx context.Context, privKey []byte, to string, amountWei *big.Int) (txHash string, gasFee *big.Int, err error)
	// SendERC20 signs and broadcasts an ERC-20 transfer(to, amount) call.
	SendERC20(ctx context.Context, privKey []byte, tokenContract, to string, amount *big.Int) (txHash string, gasFee *big.Int, err error)
	// NativeBalance returns the address's native-coin balance in wei.
	NativeBalance(ctx context.Context, address string) (*big.Int, error)
	// SuggestGasPrice returns the current suggested gas price in wei.
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
}
