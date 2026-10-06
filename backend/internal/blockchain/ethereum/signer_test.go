package ethereum

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	hdwallet "github.com/payminto/payminto/backend/internal/crypto"
)

// testKey derives a deterministic ETH private key + address from a fixed seed.
func testKey(t *testing.T) (priv []byte, addr string) {
	t.Helper()
	// 64-byte seed (BIP-39 seed length) — fixed for determinism.
	seed := make([]byte, 64)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	a, p, err := hdwallet.DeriveEthAddress(seed, 0, 0)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	return p, a
}

func TestSignNativeTransfer_RecoversSender(t *testing.T) {
	priv, from := testKey(t)
	a := NewAdapter("ETH", "Ethereum Sepolia", 11155111, nil)

	to := "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
	value := big.NewInt(1_000_000_000_000_000) // 0.001 ETH
	gasPrice := big.NewInt(20_000_000_000)     // 20 gwei

	raw, hash, err := a.SignNativeTransfer(priv, 7, to, value, gasPrice, nativeTransferGasLimit)
	if err != nil {
		t.Fatalf("SignNativeTransfer: %v", err)
	}
	if hash == "" || len(raw) == 0 {
		t.Fatal("empty hash or raw")
	}

	// Decode and verify every field + recover the sender.
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tx.ChainId().Int64() != 11155111 {
		t.Errorf("chainID = %d, want 11155111", tx.ChainId().Int64())
	}
	if tx.Nonce() != 7 {
		t.Errorf("nonce = %d, want 7", tx.Nonce())
	}
	if tx.To() == nil || tx.To().Hex() != common.HexToAddress(to).Hex() {
		t.Errorf("to = %v, want %s", tx.To(), to)
	}
	if tx.Value().Cmp(value) != 0 {
		t.Errorf("value = %s, want %s", tx.Value(), value)
	}
	if tx.Gas() != nativeTransferGasLimit {
		t.Errorf("gas = %d, want %d", tx.Gas(), nativeTransferGasLimit)
	}
	if len(tx.Data()) != 0 {
		t.Errorf("native transfer should have empty data, got %x", tx.Data())
	}

	signer := types.LatestSignerForChainID(big.NewInt(11155111))
	sender, err := types.Sender(signer, tx)
	if err != nil {
		t.Fatalf("recover sender: %v", err)
	}
	if sender.Hex() != common.HexToAddress(from).Hex() {
		t.Errorf("recovered sender = %s, want %s (key/signature mismatch)", sender.Hex(), from)
	}
	if hash != tx.Hash().Hex() {
		t.Errorf("returned hash %s != tx hash %s", hash, tx.Hash().Hex())
	}
}

func TestSignERC20Transfer_EncodesAndRecovers(t *testing.T) {
	priv, from := testKey(t)
	a := NewAdapter("ETH", "Ethereum Sepolia", 11155111, nil)

	token := "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238" // USDC Sepolia (example)
	to := "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
	amount := big.NewInt(5_000_000) // 5 USDC (6 decimals)
	gasPrice := big.NewInt(20_000_000_000)

	raw, _, err := a.SignERC20Transfer(priv, 3, token, to, amount, gasPrice, defaultERC20GasLimit)
	if err != nil {
		t.Fatalf("SignERC20Transfer: %v", err)
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// ERC-20 transfer: To is the token contract, Value is zero, Data is the call.
	if tx.To().Hex() != common.HexToAddress(token).Hex() {
		t.Errorf("to = %s, want token %s", tx.To().Hex(), token)
	}
	if tx.Value().Sign() != 0 {
		t.Errorf("erc20 transfer value should be 0, got %s", tx.Value())
	}
	// transfer selector = first 4 bytes of keccak256("transfer(address,uint256)")
	wantSelector := crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]
	if len(tx.Data()) < 4 || string(tx.Data()[:4]) != string(wantSelector) {
		t.Errorf("data selector mismatch: got %x want %x", tx.Data()[:4], wantSelector)
	}
	// The encoded recipient is the last 20 bytes of the first 32-byte arg word.
	gotTo := common.BytesToAddress(tx.Data()[4+12 : 4+32])
	if gotTo.Hex() != common.HexToAddress(to).Hex() {
		t.Errorf("encoded recipient = %s, want %s", gotTo.Hex(), to)
	}

	signer := types.LatestSignerForChainID(big.NewInt(11155111))
	sender, err := types.Sender(signer, tx)
	if err != nil {
		t.Fatalf("recover sender: %v", err)
	}
	if sender.Hex() != common.HexToAddress(from).Hex() {
		t.Errorf("recovered sender = %s, want %s", sender.Hex(), from)
	}
}

func TestSignNativeTransfer_InvalidAddress(t *testing.T) {
	priv, _ := testKey(t)
	a := NewAdapter("ETH", "Ethereum Sepolia", 11155111, nil)
	if _, _, err := a.SignNativeTransfer(priv, 0, "not-an-address", big.NewInt(1), big.NewInt(1), 21000); err == nil {
		t.Fatal("expected error for invalid destination address")
	}
}

func TestAddressFromKey_MatchesDerivation(t *testing.T) {
	priv, want := testKey(t)
	got, err := addressFromKey(priv)
	if err != nil {
		t.Fatalf("addressFromKey: %v", err)
	}
	if common.HexToAddress(got).Hex() != common.HexToAddress(want).Hex() {
		t.Errorf("addressFromKey = %s, want %s", got, want)
	}
}
