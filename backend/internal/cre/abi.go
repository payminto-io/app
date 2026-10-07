package cre

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// GatewayAttestations.json is the consumer contract's ABI (ticket 22, contracts/src/cre); kept byte-identical
// with the contract branch so both tracks read one file.
//
//go:embed abi/GatewayAttestations.json
var gatewayAttestationsABI string

var (
	contractABI     abi.ABI
	contractABIOnce sync.Once
	contractABIErr  error
)

// ContractABI parses the embedded ABI once.
func ContractABI() (abi.ABI, error) {
	contractABIOnce.Do(func() {
		contractABI, contractABIErr = abi.JSON(strings.NewReader(gatewayAttestationsABI))
	})
	return contractABI, contractABIErr
}
