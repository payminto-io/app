package tempo

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
)

func TestTransferWithMemoTopic(t *testing.T) {
	// Verified against live Moderato TransferWithMemo logs.
	if got := TransferWithMemoTopic.Hex(); got != "0x57bc7354aa85aed339e000bccffabbc529466af35f0772c8f8ee1145927de7f0" {
		t.Fatalf("topic = %s", got)
	}
}

func TestMemoRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		ref     string
		wantHex string
		wantErr bool
	}{
		{name: "uuid as 32 hex chars", ref: "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b", wantHex: "0x3666316332613362346435653466363038613762396330643165326633613462"},
		{name: "checkout contract PM-1042", ref: "PM-1042", wantHex: "0x504d2d3130343200000000000000000000000000000000000000000000000000"},
		{name: "short ascii", ref: "INV-12345", wantHex: "0x494e562d31323334350000000000000000000000000000000000000000000000"},
		{name: "exactly 32 bytes", ref: "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"},
		{name: "too long", ref: "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456", wantErr: true},
		{name: "empty", ref: "  ", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := EncodeMemoHex(tc.ref)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if tc.wantHex != "" && h != tc.wantHex {
				t.Fatalf("hex = %s, want %s", h, tc.wantHex)
			}
			m, _ := EncodeMemo(tc.ref)
			found := false
			for _, c := range MemoCandidates(m) {
				if c == tc.ref {
					found = true
				}
			}
			if !found {
				t.Fatalf("candidates %v do not contain %q", MemoCandidates(m), tc.ref)
			}
		})
	}
}

func TestMemoCandidatesRejectsNoise(t *testing.T) {
	if got := MemoCandidates([32]byte{}); len(got) != 0 {
		t.Fatalf("zero memo candidates = %v", got)
	}
	// A real third-party Moderato memo: binary, non-zero tail - not one of ours.
	m := common.HexToHash("0xef1ed71201e6dc21f2ed5ac85546af00000000000000000000cdb4311cd93abc")
	if got := MemoCandidates(m); len(got) != 0 {
		t.Fatalf("binary memo candidates = %v", got)
	}
}

func memoLog(token, from, to common.Address, amount int64, memo [32]byte, tx string, idx uint) types.Log {
	return types.Log{
		Address: token,
		Topics:  []common.Hash{TransferWithMemoTopic, common.BytesToHash(from.Bytes()), common.BytesToHash(to.Bytes()), memo},
		Data:    common.LeftPadBytes(big.NewInt(amount).Bytes(), 32),
		TxHash:  common.HexToHash(tx), Index: idx, BlockNumber: 100,
	}
}

func TestDecodeLog(t *testing.T) {
	merchant := common.HexToAddress("0x82c70b91731914e7b678a232b366336Af1436455")
	payer := common.HexToAddress("0x3AB22D906d5Ee0e17C44b60F45DCb56aC6371993")
	memo, _ := EncodeMemo("INV-1")
	got, err := DecodeLog(memoLog(AlphaUSD, payer, merchant, 1_500_000, memo, "0x01", 7))
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != AlphaUSD || got.From != payer || got.To != merchant || got.Amount.Int64() != 1_500_000 || got.Memo != memo || got.LogIndex != 7 {
		t.Fatalf("decoded %+v", got)
	}
	bad := memoLog(AlphaUSD, payer, merchant, 1, memo, "0x01", 0)
	bad.Topics = bad.Topics[:3] // plain Transfer shape
	if _, err := DecodeLog(bad); err != ErrNotMemoTransfer {
		t.Fatalf("err = %v", err)
	}
	short := memoLog(AlphaUSD, payer, merchant, 1, memo, "0x01", 0)
	short.Data = short.Data[:31]
	if _, err := DecodeLog(short); err == nil {
		t.Fatal("short data decoded")
	}
}

func TestVerify(t *testing.T) {
	merchant := common.HexToAddress("0x82c70b91731914e7b678a232b366336Af1436455")
	other := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	tokens := map[common.Address]Token{AlphaUSD: {Address: AlphaUSD, Decimals: 6, BlockchainCurrencyID: 9, Code: "ALPHAUSD"}}
	memo, _ := EncodeMemo("INV-1")
	ok := MemoTransfer{Token: AlphaUSD, To: merchant, Amount: big.NewInt(2_500_000), Memo: memo}
	cases := []struct {
		name   string
		mut    func(*MemoTransfer)
		status uint64
		want   Rejection
		amount string
	}{
		{name: "valid", status: 1, want: RejectNone, amount: "2.5"},
		{name: "reverted tx", status: 0, want: RejectTxFailed},
		{name: "unknown token", mut: func(m *MemoTransfer) { m.Token = PathUSD }, status: 1, want: RejectUnknownToken},
		{name: "wrong recipient", mut: func(m *MemoTransfer) { m.To = other }, status: 1, want: RejectWrongRecipient},
		{name: "zero amount", mut: func(m *MemoTransfer) { m.Amount = big.NewInt(0) }, status: 1, want: RejectZeroAmount},
		{name: "removed log", mut: func(m *MemoTransfer) { m.Removed = true }, status: 1, want: RejectRemoved},
		{name: "empty memo", mut: func(m *MemoTransfer) { m.Memo = [32]byte{} }, status: 1, want: RejectNoReference},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := ok
			if tc.mut != nil {
				tc.mut(&m)
			}
			tok, amt, rej := Verify(m, tokens, merchant, tc.status)
			if rej != tc.want {
				t.Fatalf("rejection = %q, want %q", rej, tc.want)
			}
			if tc.want == RejectNone && (!amt.Equal(decimal.RequireFromString(tc.amount)) || tok.BlockchainCurrencyID != 9) {
				t.Fatalf("amount %s token %+v", amt, tok)
			}
		})
	}
}

func TestDedupe(t *testing.T) {
	a := MemoTransfer{TxHash: common.HexToHash("0xaa"), LogIndex: 1}
	b := MemoTransfer{TxHash: common.HexToHash("0xaa"), LogIndex: 2}
	c := MemoTransfer{TxHash: common.HexToHash("0xbb"), LogIndex: 1}
	got := Dedupe([]MemoTransfer{a, b, a, c, b})
	if len(got) != 3 || got[0].Key() != a.Key() || got[1].Key() != b.Key() || got[2].Key() != c.Key() {
		t.Fatalf("dedupe = %v", got)
	}
}
