package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain"
)

// --- fakes ---

type fakeKeys struct {
	priv   []byte
	family string
	err    error
}

func (f fakeKeys) PrivateKeyForAddress(string) ([]byte, string, error) {
	return f.priv, f.family, f.err
}

// fakeEVMAdapter embeds ChainAdapter (nil) so it satisfies the interface; only
// the EVM signing methods we exercise are implemented.
type fakeEVMAdapter struct {
	blockchain.ChainAdapter
	nativeHash string
	nativeFee  *big.Int
	nativeErr  error
	balance    *big.Int
	gasPrice   *big.Int
	gotPriv    []byte
	gotTo      string
	gotAmount  *big.Int
}

func (f *fakeEVMAdapter) SendNative(_ context.Context, privKey []byte, to string, amount *big.Int) (string, *big.Int, error) {
	f.gotPriv, f.gotTo, f.gotAmount = privKey, to, amount
	return f.nativeHash, f.nativeFee, f.nativeErr
}

func (f *fakeEVMAdapter) SendERC20(_ context.Context, _ []byte, _, _ string, _ *big.Int) (string, *big.Int, error) {
	return "erc20hash", big.NewInt(1), nil
}

func (f *fakeEVMAdapter) NativeBalance(context.Context, string) (*big.Int, error) {
	return f.balance, nil
}

func (f *fakeEVMAdapter) SuggestGasPrice(context.Context) (*big.Int, error) {
	return f.gasPrice, nil
}

// nonEVMAdapter implements ChainAdapter (nil embed) but NOT EVMSigner.
type nonEVMAdapter struct{ blockchain.ChainAdapter }

type fakeAdapters struct {
	adapter blockchain.ChainAdapter
	err     error
}

func (f fakeAdapters) Get(string) (blockchain.ChainAdapter, error) { return f.adapter, f.err }

// --- tests ---

func TestEVMBroadcaster_SendNative_HappyPath(t *testing.T) {
	fa := &fakeEVMAdapter{nativeHash: "0xabc", nativeFee: big.NewInt(420000)}
	b := NewEVMBroadcaster(
		fakeKeys{priv: []byte{1, 2, 3}, family: "ETH_Family"},
		fakeAdapters{adapter: fa},
	)

	hash, fee, err := b.SendNative(context.Background(), "ETH", "0xfrom", "0xto", big.NewInt(1000))
	if err != nil {
		t.Fatalf("SendNative: %v", err)
	}
	if hash != "0xabc" || fee.Cmp(big.NewInt(420000)) != 0 {
		t.Errorf("got hash=%s fee=%s", hash, fee)
	}
	if fa.gotTo != "0xto" || fa.gotAmount.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("adapter received wrong args: to=%s amount=%s", fa.gotTo, fa.gotAmount)
	}
}

func TestEVMBroadcaster_KeyResolutionFails(t *testing.T) {
	b := NewEVMBroadcaster(
		fakeKeys{err: errors.New("vault locked")},
		fakeAdapters{adapter: &fakeEVMAdapter{}},
	)
	if _, _, err := b.SendNative(context.Background(), "ETH", "0xfrom", "0xto", big.NewInt(1)); err == nil {
		t.Fatal("expected error when key resolution fails")
	}
}

func TestEVMBroadcaster_UnknownChain(t *testing.T) {
	b := NewEVMBroadcaster(
		fakeKeys{priv: []byte{1}},
		fakeAdapters{err: errors.New("not found")},
	)
	if _, _, err := b.SendNative(context.Background(), "DOGE", "0xfrom", "0xto", big.NewInt(1)); err == nil {
		t.Fatal("expected error for unknown chain")
	}
}

func TestEVMBroadcaster_SweepNative_ComputesAmountMinusGas(t *testing.T) {
	// balance 1 ETH, gas price 20 gwei → gasCost = 20e9 * 21000 = 4.2e14 wei.
	balance := big.NewInt(1_000_000_000_000_000_000)
	gasPrice := big.NewInt(20_000_000_000)
	fa := &fakeEVMAdapter{nativeHash: "0xsweep", nativeFee: big.NewInt(420000000000000), balance: balance, gasPrice: gasPrice}
	b := NewEVMBroadcaster(fakeKeys{priv: []byte{9}}, fakeAdapters{adapter: fa})

	hash, amount, _, err := b.SweepNative(context.Background(), "ETH", "0xfrom", "0xcold", nil)
	if err != nil {
		t.Fatalf("SweepNative: %v", err)
	}
	wantAmount := new(big.Int).Sub(balance, big.NewInt(420000000000000))
	if hash != "0xsweep" || amount.Cmp(wantAmount) != 0 {
		t.Errorf("hash=%s amount=%s want amount=%s", hash, amount, wantAmount)
	}
	if fa.gotAmount.Cmp(wantAmount) != 0 {
		t.Errorf("broadcast amount = %s, want %s", fa.gotAmount, wantAmount)
	}
}

func TestEVMBroadcaster_SweepNative_SkipsDust(t *testing.T) {
	// balance < gas cost → nothing to sweep, no error, no broadcast.
	fa := &fakeEVMAdapter{balance: big.NewInt(1000), gasPrice: big.NewInt(20_000_000_000)}
	b := NewEVMBroadcaster(fakeKeys{priv: []byte{9}}, fakeAdapters{adapter: fa})

	hash, amount, _, err := b.SweepNative(context.Background(), "ETH", "0xfrom", "0xcold", nil)
	if err != nil {
		t.Fatalf("SweepNative dust: %v", err)
	}
	if hash != "" || amount != nil {
		t.Errorf("expected skip (empty), got hash=%s amount=%v", hash, amount)
	}
	if fa.gotAmount != nil {
		t.Error("dust sweep must not broadcast")
	}
}

func TestEVMBroadcaster_SweepNative_RespectsMinThreshold(t *testing.T) {
	// net amount above gas but below minSweep → skip.
	balance := big.NewInt(500_000_000_000_000) // 0.0005 ETH
	gasPrice := big.NewInt(1_000_000_000)      // 1 gwei → gasCost 21e12
	fa := &fakeEVMAdapter{balance: balance, gasPrice: gasPrice}
	b := NewEVMBroadcaster(fakeKeys{priv: []byte{9}}, fakeAdapters{adapter: fa})

	minSweep := big.NewInt(1_000_000_000_000_000) // 0.001 ETH
	hash, amount, _, err := b.SweepNative(context.Background(), "ETH", "0xfrom", "0xcold", minSweep)
	if err != nil {
		t.Fatalf("SweepNative: %v", err)
	}
	if hash != "" || amount != nil {
		t.Errorf("expected skip below min threshold, got hash=%s amount=%v", hash, amount)
	}
}

func TestEVMBroadcaster_NonEVMChain(t *testing.T) {
	b := NewEVMBroadcaster(
		fakeKeys{priv: []byte{1}},
		fakeAdapters{adapter: nonEVMAdapter{}},
	)
	if _, _, err := b.SendNative(context.Background(), "BTC", "0xfrom", "0xto", big.NewInt(1)); err == nil {
		t.Fatal("expected error for non-EVM adapter")
	}
}
