package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Minimal ERC-20 ABI for the balanceOf, transfer, and Transfer event methods
// we need. Copy-pasted from the openzeppelin standard interface.
const erc20ABI = `[
	{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},
	{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}
]`

// transferEventTopic is the keccak256 of "Transfer(address,address,uint256)".
// It's the topic0 of every ERC-20 Transfer event.
var transferEventTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// erc20BalanceOf calls balanceOf(address) on an ERC-20 contract.
func (a *Adapter) erc20BalanceOf(ctx context.Context, client *ethclient.Client, tokenAddr, holderAddr string) (*big.Int, error) {
	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}
	data, err := parsed.Pack("balanceOf", common.HexToAddress(holderAddr))
	if err != nil {
		return nil, fmt.Errorf("pack balanceOf: %w", err)
	}
	to := common.HexToAddress(tokenAddr)
	msg := ethereum.CallMsg{To: &to, Data: data}
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("call balanceOf: %w", err)
	}
	var balance *big.Int
	if err := parsed.UnpackIntoInterface(&balance, "balanceOf", result); err != nil {
		return nil, fmt.Errorf("unpack balance: %w", err)
	}
	return balance, nil
}

// DecodeTransferLog extracts (from, to, amount) from a log entry if and only
// if the log is an ERC-20 Transfer event. Returns nil if the log isn't a
// Transfer.
func DecodeTransferLog(log types.Log) (from, to string, amount *big.Int, err error) {
	if len(log.Topics) < 3 {
		return "", "", nil, errors.New("not a Transfer log")
	}
	if log.Topics[0] != transferEventTopic {
		return "", "", nil, errors.New("topic0 is not Transfer event")
	}

	from = common.HexToAddress(log.Topics[1].Hex()).Hex()
	to = common.HexToAddress(log.Topics[2].Hex()).Hex()

	if len(log.Data) < 32 {
		return "", "", nil, errors.New("data too short for uint256 value")
	}
	amount = new(big.Int).SetBytes(log.Data[:32])
	return from, to, amount, nil
}
