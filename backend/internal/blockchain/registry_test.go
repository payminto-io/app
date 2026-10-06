package blockchain

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
)

// testAdapter is a minimal ChainAdapter for registry tests.
type testAdapter struct {
	code    string
	name    string
	mainnet bool
}

func (t *testAdapter) Name() string                            { return t.name }
func (t *testAdapter) Code() string                            { return t.code }
func (t *testAdapter) IsMainnet() bool                         { return t.mainnet }
func (t *testAdapter) GenerateAddress(uint32) (string, error)  { return "", nil }
func (t *testAdapter) GetBalance(context.Context, string, string) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (t *testAdapter) GetConfirmations(context.Context, string) (uint64, error) { return 0, nil }
func (t *testAdapter) MonitorBlocks(context.Context, uint64) (<-chan Transaction, error) {
	ch := make(chan Transaction)
	close(ch)
	return ch, nil
}
func (t *testAdapter) BroadcastTransaction(context.Context, []byte) (string, error) {
	return "", nil
}
func (t *testAdapter) EstimateGas(context.Context, TxParams) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (t *testAdapter) ParseBlock(_ context.Context, _ uint64, _ map[string]WatchedAddressInfo) ([]Transaction, error) {
	return []Transaction{}, nil
}
func (t *testAdapter) LatestBlock(_ context.Context) (uint64, error) {
	return 0, nil
}

// Compile-time check that testAdapter satisfies the interface.
var _ ChainAdapter = (*testAdapter)(nil)
var _ = decimal.Zero

func TestAdapterRegistry_RegisterAndGet(t *testing.T) {
	r := NewAdapterRegistry()

	eth := &testAdapter{code: "ETH", name: "Ethereum", mainnet: true}
	if err := r.Register(eth); err != nil {
		t.Fatal(err)
	}

	got, err := r.Get("ETH")
	if err != nil {
		t.Fatal(err)
	}
	if got.Code() != "ETH" {
		t.Errorf("expected ETH, got %s", got.Code())
	}
}

func TestAdapterRegistry_Register_Duplicate(t *testing.T) {
	r := NewAdapterRegistry()
	_ = r.Register(&testAdapter{code: "ETH", name: "Ethereum"})
	err := r.Register(&testAdapter{code: "ETH", name: "Ethereum2"})
	if err == nil {
		t.Error("expected error on duplicate registration")
	}
}

func TestAdapterRegistry_Get_Missing(t *testing.T) {
	r := NewAdapterRegistry()
	if _, err := r.Get("XYZ"); err == nil {
		t.Error("expected error for unknown chain")
	}
}

func TestAdapterRegistry_Has(t *testing.T) {
	r := NewAdapterRegistry()
	if r.Has("ETH") {
		t.Error("empty registry should not have ETH")
	}
	_ = r.Register(&testAdapter{code: "ETH", name: "Ethereum"})
	if !r.Has("ETH") {
		t.Error("should have ETH after Register")
	}
}

func TestAdapterRegistry_Codes(t *testing.T) {
	r := NewAdapterRegistry()
	_ = r.Register(&testAdapter{code: "ETH", name: "Ethereum"})
	_ = r.Register(&testAdapter{code: "BTC", name: "Bitcoin"})
	_ = r.Register(&testAdapter{code: "TRX", name: "Tron"})
	codes := r.Codes()
	if len(codes) != 3 {
		t.Errorf("expected 3 codes, got %d", len(codes))
	}
}

func TestAdapterRegistry_Len(t *testing.T) {
	r := NewAdapterRegistry()
	if r.Len() != 0 {
		t.Error("empty registry should have len 0")
	}
	_ = r.Register(&testAdapter{code: "ETH", name: "Ethereum"})
	if r.Len() != 1 {
		t.Error("expected len 1 after one register")
	}
}

func TestAdapterRegistry_CountByMode(t *testing.T) {
	r := NewAdapterRegistry()
	_ = r.Register(&testAdapter{code: "ETH", name: "Ethereum", mainnet: true})
	_ = r.Register(&testAdapter{code: "BASE", name: "Base", mainnet: true})
	_ = r.Register(&testAdapter{code: "SEPOLIA", name: "Sepolia", mainnet: false})

	mainnet, testnet := r.CountByMode()
	if mainnet != 2 {
		t.Errorf("expected 2 mainnet, got %d", mainnet)
	}
	if testnet != 1 {
		t.Errorf("expected 1 testnet, got %d", testnet)
	}
}

func TestAdapterRegistry_All(t *testing.T) {
	r := NewAdapterRegistry()
	_ = r.Register(&testAdapter{code: "ETH", name: "Ethereum"})
	_ = r.Register(&testAdapter{code: "BTC", name: "Bitcoin"})

	all := r.All()
	if len(all) != 2 {
		t.Errorf("expected 2 adapters, got %d", len(all))
	}
	if _, ok := all["ETH"]; !ok {
		t.Error("expected ETH in All()")
	}

	// Mutating the returned map should not affect the registry
	delete(all, "ETH")
	if !r.Has("ETH") {
		t.Error("mutating copy should not affect original")
	}
}
