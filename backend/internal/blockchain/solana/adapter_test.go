package solana

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
)

func TestAdapter_GetConfirmations(t *testing.T) {
	c := NewScriptedCaller()
	a := NewAdapterWithCaller(c, ClusterDevnet)
	ctx := context.Background()

	c.Result("getSignatureStatuses", ContextValue(10, []any{nil}))
	if n, err := a.GetConfirmations(ctx, "sig"); err != nil || n != 0 {
		t.Fatalf("unknown signature: %d %v", n, err)
	}
	c.Result("getSignatureStatuses", ContextValue(10, []any{map[string]any{"slot": 5, "confirmations": 7, "err": nil, "confirmationStatus": "confirmed"}}))
	if n, _ := a.GetConfirmations(ctx, "sig"); n != 7 {
		t.Fatalf("confirmed: %d, want 7", n)
	}
	c.Result("getSignatureStatuses", ContextValue(10, []any{map[string]any{"slot": 5, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"}}))
	if n, _ := a.GetConfirmations(ctx, "sig"); n != FinalizedConfirmations {
		t.Fatalf("finalized: %d, want %d", n, FinalizedConfirmations)
	}
	c.Result("getSignatureStatuses", ContextValue(10, []any{map[string]any{"slot": 5, "confirmations": nil, "err": map[string]any{"InstructionError": []any{0, "Custom"}}, "confirmationStatus": "finalized"}}))
	if _, err := a.GetConfirmations(ctx, "sig"); err == nil {
		t.Fatal("a failed transaction must error")
	}
	if a.Code() != "SOLANA" || a.IsMainnet() {
		t.Fatal("devnet adapter identity wrong")
	}
	if !NewAdapterWithCaller(c, ClusterMainnet).IsMainnet() {
		t.Fatal("mainnet-beta is mainnet")
	}
}

func TestAdapter_ParseBlockFindsCreditsToWatchedATA(t *testing.T) {
	raw, err := LoadFixtureJSON(filepath.Join("testdata", "tx", "usdc_transfer_checked.json"))
	if err != nil {
		t.Fatal(err)
	}
	c := NewScriptedCaller().On("getBlock", func([]any) (any, error) {
		return map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "parentSlot": 250000122, "transactions": []any{raw}}, nil
	})
	a := NewAdapterWithCaller(c, ClusterDevnet)
	watched := map[string]blockchain.WatchedAddressInfo{
		fxUSDCATA.String(): {BlockchainCurrencyID: 7, Decimals: 6, TokenAddress: fxUSDC.String()},
		fxUSDTATA.String(): {BlockchainCurrencyID: 8, Decimals: 6, TokenAddress: fxUSDT.String()},
	}
	txs, err := a.ParseBlock(context.Background(), 250000123, watched)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || txs[0].ToAddress != fxUSDCATA.String() || txs[0].Amount.String() != "25" || txs[0].BlockNumber != 250000123 {
		t.Fatalf("txs = %+v", txs)
	}
	c.Result("getBlock", nil)
	if txs, err := a.ParseBlock(context.Background(), 1, watched); err != nil || len(txs) != 0 {
		t.Fatalf("skipped slot: %v %v", txs, err)
	}
}

func TestAdapter_GetBalanceTokenUsesATA(t *testing.T) {
	c := NewScriptedCaller()
	c.On("getAccountInfo", func(p []any) (any, error) {
		return ContextValue(1, map[string]any{"lamports": 1, "owner": TokenProgram.String(), "data": map[string]any{"parsed": map[string]any{"type": "mint", "info": map[string]any{"decimals": 6}}}}), nil
	})
	c.On("getTokenAccountBalance", func(p []any) (any, error) {
		if FirstParamString(p) != fxUSDCATA.String() {
			return nil, errors.New("wrong token account queried")
		}
		return ContextValue(1, map[string]any{"amount": "123456", "decimals": 6, "uiAmountString": "0.123456"}), nil
	})
	a := NewAdapterWithCaller(c, ClusterDevnet)
	bal, err := a.GetBalance(context.Background(), fxOwner.String(), fxUSDC.String())
	if err != nil || bal.Int64() != 123456 {
		t.Fatalf("balance = %v err %v", bal, err)
	}
	c.On("getTokenAccountBalance", func([]any) (any, error) {
		return nil, &RPCError{Code: -32602, Message: "Invalid param: could not find account"}
	})
	if bal, err := a.GetBalance(context.Background(), fxOwner.String(), fxUSDC.String()); err != nil || bal.Sign() != 0 {
		t.Fatalf("missing ATA should read as zero: %v %v", bal, err)
	}
}

func TestSendAndConfirm_RetriesWithFreshBlockhashAfterExpiry(t *testing.T) {
	fee := newTestSigner(t)
	c := NewScriptedCaller()
	blockhashes := []string{"GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "7pWqF1vXjQ2nD4sT8kL6mB3cR5yH9wE2aG7uN1xP4zV8"}
	c.On("getLatestBlockhash", func([]any) (any, error) {
		i := c.Count("getLatestBlockhash") - 1
		return ContextValue(1, map[string]any{"blockhash": blockhashes[i], "lastValidBlockHeight": 100 + uint64(i)*100}), nil
	})
	var sent []string
	c.On("sendTransaction", func(p []any) (any, error) {
		sent = append(sent, FirstParamString(p))
		return "sig" + string(rune('A'+len(sent))), nil
	})
	c.On("getSignatureStatuses", func([]any) (any, error) {
		if len(sent) < 2 {
			return ContextValue(1, []any{nil}), nil
		}
		return ContextValue(1, []any{map[string]any{"slot": 42, "confirmations": 1, "err": nil, "confirmationStatus": "confirmed"}}), nil
	})
	c.On("getBlockHeight", func([]any) (any, error) { return 150, nil }) // past the first lastValidBlockHeight

	var used []string
	build := func(bh string) (Message, error) {
		used = append(used, bh)
		return CompileMessage(fee.PublicKey(), MustPublicKey(bh), []Instruction{ComputeBudgetSetUnitLimit(1)})
	}
	res, err := SendAndConfirm(context.Background(), NewClient(c), build, []Signer{fee}, SendOptions{Poll: time.Millisecond, Wait: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || len(used) != 2 || used[0] == used[1] || res.Slot != 42 || !res.Landed {
		t.Fatalf("result = %+v used = %v", res, used)
	}
}

func TestSendAndConfirm_ReportsOnChainFailure(t *testing.T) {
	fee := newTestSigner(t)
	c := NewScriptedCaller()
	c.Result("getLatestBlockhash", ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 100}))
	c.Result("sendTransaction", "sigX")
	c.Result("getSignatureStatuses", ContextValue(1, []any{map[string]any{"slot": 1, "err": map[string]any{"InstructionError": []any{1, map[string]any{"Custom": 1}}}, "confirmationStatus": "confirmed"}}))
	build := func(bh string) (Message, error) {
		return CompileMessage(fee.PublicKey(), MustPublicKey(bh), []Instruction{ComputeBudgetSetUnitLimit(1)})
	}
	if _, err := SendAndConfirm(context.Background(), NewClient(c), build, []Signer{fee}, SendOptions{Poll: time.Millisecond}); err == nil {
		t.Fatal("expected failure")
	}
}
