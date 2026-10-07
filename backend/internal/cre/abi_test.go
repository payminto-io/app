package cre

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// The embedded ABI must equal the compiled contract's (forge build output); skipped when contracts are not built.
func TestEmbeddedABIMatchesTheCompiledContract(t *testing.T) {
	raw, err := os.ReadFile("../../../contracts/out/GatewayAttestations.sol/GatewayAttestations.json")
	if err != nil {
		t.Skip("contracts not built: run `forge build` in contracts/ to enable this check")
	}
	var artifact struct {
		ABI []any `json:"abi"`
	}
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatal(err)
	}
	var embedded []any
	if err := json.Unmarshal([]byte(gatewayAttestationsABI), &embedded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(artifact.ABI, embedded) {
		t.Fatal("backend/internal/cre/abi/GatewayAttestations.json differs from contracts/out; regenerate with forge inspect")
	}
	if _, err := ContractABI(); err != nil {
		t.Fatal(err)
	}
}
