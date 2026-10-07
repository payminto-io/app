package blockchain

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
)

type mockAdapter struct{}

func (m *mockAdapter) Name() string                               { return "mock" }
func (m *mockAdapter) Code() string                               { return "MOCK" }
func (m *mockAdapter) IsMainnet() bool                            { return false }
func (m *mockAdapter) GenerateAddress(idx uint32) (string, error) { return "0xmock", nil }
func (m *mockAdapter) GetBalance(ctx context.Context, addr string, token string) (*big.Int, error) {
	return big.NewInt(1000), nil
}
func (m *mockAdapter) GetConfirmations(ctx context.Context, txHash string) (uint64, error) {
	return 12, nil
}
func (m *mockAdapter) MonitorBlocks(ctx context.Context, fromBlock uint64) (<-chan Transaction, error) {
	ch := make(chan Transaction, 1)
	ch <- Transaction{TxHash: "0xabc", Amount: decimal.NewFromInt(1)}
	close(ch)
	return ch, nil
}
func (m *mockAdapter) BroadcastTransaction(ctx context.Context, signedTx []byte) (string, error) {
	return "0xtxhash", nil
}
func (m *mockAdapter) EstimateGas(ctx context.Context, params TxParams) (*big.Int, error) {
	return big.NewInt(21000), nil
}
func (m *mockAdapter) ParseBlock(ctx context.Context, blockNumber uint64, watched map[string]WatchedAddressInfo) ([]Transaction, error) {
	return []Transaction{}, nil
}
func (m *mockAdapter) LatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func TestChainAdapter_Interface(t *testing.T) {
	var adapter ChainAdapter = &mockAdapter{}
	if adapter.Code() != "MOCK" {
		t.Errorf("expected MOCK, got %s", adapter.Code())
	}
	addr, err := adapter.GenerateAddress(0)
	if err != nil {
		t.Fatalf("GenerateAddress error: %v", err)
	}
	if addr != "0xmock" {
		t.Errorf("expected 0xmock, got %s", addr)
	}
}

func TestMockAdapter_Balance(t *testing.T) {
	adapter := &mockAdapter{}
	bal, err := adapter.GetBalance(t.Context(), "0x1", "ETH")
	if err != nil {
		t.Fatal(err)
	}
	if bal.Int64() != 1000 {
		t.Errorf("expected 1000, got %d", bal.Int64())
	}
}

func TestMockAdapter_MonitorBlocks(t *testing.T) {
	adapter := &mockAdapter{}
	ch, err := adapter.MonitorBlocks(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	tx := <-ch
	if tx.TxHash != "0xabc" {
		t.Errorf("expected 0xabc, got %s", tx.TxHash)
	}
}

func TestMockAdapter_Confirmations(t *testing.T) {
	adapter := &mockAdapter{}
	conf, err := adapter.GetConfirmations(t.Context(), "0x123")
	if err != nil {
		t.Fatal(err)
	}
	if conf != 12 {
		t.Errorf("expected 12, got %d", conf)
	}
}

func TestMockAdapter_EstimateGas(t *testing.T) {
	adapter := &mockAdapter{}
	gas, err := adapter.EstimateGas(t.Context(), TxParams{})
	if err != nil {
		t.Fatal(err)
	}
	if gas.Int64() != 21000 {
		t.Errorf("expected 21000, got %d", gas.Int64())
	}
}

func TestNormalizeAddress_EVM(t *testing.T) {
	// EVM chains should lowercase
	evmChains := []string{"ETH", "BASE", "POLYGON", "ARBITRUM", "OPTIMISM"}
	input := "0xAbCdEf1234567890AbCdEf1234567890AbCdEf12"
	want := "0xabcdef1234567890abcdef1234567890abcdef12"
	for _, code := range evmChains {
		got := NormalizeAddress(code, input)
		if got != want {
			t.Errorf("NormalizeAddress(%q, %q) = %q, want %q", code, input, got, want)
		}
	}
}

func TestNormalizeAddress_BTC(t *testing.T) {
	addr := "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh"
	got := NormalizeAddress("BTC", addr)
	if got != addr {
		t.Errorf("NormalizeAddress(BTC) should pass through, got %q", got)
	}
}

func TestNormalizeAddress_TRX(t *testing.T) {
	addr := "TRX7NNmRaKAeGzQ1vZm1R4PjvCxJn4nPfD"
	got := NormalizeAddress("TRX", addr)
	if got != addr {
		t.Errorf("NormalizeAddress(TRX) should pass through, got %q", got)
	}
}
