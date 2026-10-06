package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// erc20TransferABI is the minimal ABI needed to encode an ERC-20 transfer call.
const erc20TransferABI = `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

// defaultERC20GasLimit is a safe upper bound for a standard ERC-20 transfer when
// a live gas estimate is unavailable (typical transfers cost ~45k-65k gas).
const defaultERC20GasLimit uint64 = 100_000

// nativeTransferGasLimit is the fixed cost of a plain value transfer.
const nativeTransferGasLimit uint64 = 21_000

// EncodeERC20Transfer ABI-encodes a transfer(to, amount) call payload.
func EncodeERC20Transfer(to common.Address, amount *big.Int) ([]byte, error) {
	parsed, err := abi.JSON(strings.NewReader(erc20TransferABI))
	if err != nil {
		return nil, fmt.Errorf("parse transfer abi: %w", err)
	}
	data, err := parsed.Pack("transfer", to, amount)
	if err != nil {
		return nil, fmt.Errorf("pack transfer: %w", err)
	}
	return data, nil
}

// signLegacyTx builds and signs an EIP-155 legacy transaction, returning the
// RLP-encoded raw bytes (ready for BroadcastTransaction) and the tx hash.
//
// This function is pure — it performs no network I/O — so it is fully unit
// testable by recovering the sender from the signed transaction.
func (a *Adapter) signLegacyTx(privKey []byte, nonce uint64, to common.Address, value, gasPrice *big.Int, gasLimit uint64, data []byte) (raw []byte, txHash string, err error) {
	key, err := crypto.ToECDSA(privKey)
	if err != nil {
		return nil, "", fmt.Errorf("parse private key: %w", err)
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &to,
		Value:    value,
		Data:     data,
	})
	signer := types.LatestSignerForChainID(big.NewInt(a.chainID))
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		return nil, "", fmt.Errorf("sign tx: %w", err)
	}
	raw, err = signedTx.MarshalBinary()
	if err != nil {
		return nil, "", fmt.Errorf("marshal tx: %w", err)
	}
	return raw, signedTx.Hash().Hex(), nil
}

// SignNativeTransfer signs a native-coin (ETH/BASE/POLYGON) value transfer.
// Pure: no network I/O. Returns raw signed bytes and the tx hash.
func (a *Adapter) SignNativeTransfer(privKey []byte, nonce uint64, to string, amountWei, gasPrice *big.Int, gasLimit uint64) (raw []byte, txHash string, err error) {
	if !common.IsHexAddress(to) {
		return nil, "", fmt.Errorf("invalid destination address %q", to)
	}
	return a.signLegacyTx(privKey, nonce, common.HexToAddress(to), amountWei, gasPrice, gasLimit, nil)
}

// SignERC20Transfer signs an ERC-20 transfer(to, amount) call to tokenContract.
// Pure: no network I/O. The native value is zero; amount is the token amount.
func (a *Adapter) SignERC20Transfer(privKey []byte, nonce uint64, tokenContract, to string, amount, gasPrice *big.Int, gasLimit uint64) (raw []byte, txHash string, err error) {
	if !common.IsHexAddress(tokenContract) {
		return nil, "", fmt.Errorf("invalid token contract %q", tokenContract)
	}
	if !common.IsHexAddress(to) {
		return nil, "", fmt.Errorf("invalid destination address %q", to)
	}
	data, err := EncodeERC20Transfer(common.HexToAddress(to), amount)
	if err != nil {
		return nil, "", err
	}
	return a.signLegacyTx(privKey, nonce, common.HexToAddress(tokenContract), big.NewInt(0), gasPrice, gasLimit, data)
}

// PendingNonce returns the next nonce to use for an address (includes pending
// txs in the mempool).
func (a *Adapter) PendingNonce(ctx context.Context, from string) (uint64, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()
	n, err := client.PendingNonceAt(ctx, common.HexToAddress(from))
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return 0, fmt.Errorf("pending nonce: %w", err)
	}
	a.pool.MarkSuccess(nodeID)
	return n, nil
}

// SuggestGasPrice returns the node's suggested gas price in wei.
func (a *Adapter) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	client, nodeID, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	gp, err := client.SuggestGasPrice(ctx)
	if err != nil {
		a.pool.MarkFailure(nodeID, err)
		return nil, fmt.Errorf("suggest gas price: %w", err)
	}
	a.pool.MarkSuccess(nodeID)
	return gp, nil
}

// SendNative builds, signs, and broadcasts a native value transfer, returning
// the broadcast tx hash and the gas fee paid (gasPrice * gasLimit). Performs
// network I/O for nonce and gas price.
func (a *Adapter) SendNative(ctx context.Context, privKey []byte, to string, amountWei *big.Int) (txHash string, gasFee *big.Int, err error) {
	from, err := addressFromKey(privKey)
	if err != nil {
		return "", nil, err
	}
	nonce, err := a.PendingNonce(ctx, from)
	if err != nil {
		return "", nil, err
	}
	gasPrice, err := a.SuggestGasPrice(ctx)
	if err != nil {
		return "", nil, err
	}
	raw, _, err := a.SignNativeTransfer(privKey, nonce, to, amountWei, gasPrice, nativeTransferGasLimit)
	if err != nil {
		return "", nil, err
	}
	hash, err := a.BroadcastTransaction(ctx, raw)
	if err != nil {
		return "", nil, err
	}
	gasFee = new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(nativeTransferGasLimit))
	return hash, gasFee, nil
}

// SendERC20 builds, signs, and broadcasts an ERC-20 transfer, returning the
// broadcast tx hash and the gas fee paid. Performs network I/O for nonce/gas.
func (a *Adapter) SendERC20(ctx context.Context, privKey []byte, tokenContract, to string, amount *big.Int) (txHash string, gasFee *big.Int, err error) {
	from, err := addressFromKey(privKey)
	if err != nil {
		return "", nil, err
	}
	nonce, err := a.PendingNonce(ctx, from)
	if err != nil {
		return "", nil, err
	}
	gasPrice, err := a.SuggestGasPrice(ctx)
	if err != nil {
		return "", nil, err
	}
	raw, _, err := a.SignERC20Transfer(privKey, nonce, tokenContract, to, amount, gasPrice, defaultERC20GasLimit)
	if err != nil {
		return "", nil, err
	}
	hash, err := a.BroadcastTransaction(ctx, raw)
	if err != nil {
		return "", nil, err
	}
	gasFee = new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(defaultERC20GasLimit))
	return hash, gasFee, nil
}

// NativeBalance returns the native-coin (wei) balance of an address. Thin
// wrapper over GetBalance for the EVMSigner capability interface.
func (a *Adapter) NativeBalance(ctx context.Context, address string) (*big.Int, error) {
	return a.GetBalance(ctx, address, "")
}

// addressFromKey derives the 0x checksummed address for a raw private key.
func addressFromKey(privKey []byte) (string, error) {
	key, err := crypto.ToECDSA(privKey)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	return crypto.PubkeyToAddress(key.PublicKey).Hex(), nil
}
