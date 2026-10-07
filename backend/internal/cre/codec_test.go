package cre

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestReportRoundTrip(t *testing.T) {
	gw := GatewayID("https://pay.example.test")
	at := time.Unix(1_790_000_000, 0).UTC()
	cases := []Report{
		{Kind: KindSolvency, GatewayID: gw, ObservedAt: at, Items: []SolvencyItem{
			{CheckpointHash: SubjectKey("cp-1"), Asset: LabelKey("USDC.SOLANA"), Liabilities: big.NewInt(1_000_000), Reserves: big.NewInt(1_500_000), Decimals: 6},
			{CheckpointHash: SubjectKey("cp-1"), Asset: LabelKey("SOL"), Liabilities: big.NewInt(5), Reserves: big.NewInt(0), Decimals: 9},
		}},
		{Kind: KindDepositFinality, GatewayID: gw, ObservedAt: at, Items: []DepositItem{
			{DepositID: SubjectKey("dep-1"), ChainID: LabelKey("solana"), TxRef: SubjectKey("sig"), Token: LabelKey("USDC"), Amount: big.NewInt(42), Destination: SubjectKey("addr"), SlotOrBlock: 99, Verdict: VerdictConfirmed},
		}},
		{Kind: KindConversionReference, GatewayID: gw, ObservedAt: at, Items: []ConversionItem{
			{ConversionID: SubjectKey("cv-1"), Pair: LabelKey("USDC/USD"), ReferenceRate: big.NewInt(99_990_000), ReferenceDecimals: 8, DeviationBps: big.NewInt(-3), Feed: common.HexToAddress("0x1"), RoundID: big.NewInt(7)},
		}},
	}
	for _, in := range cases {
		payload, err := EncodeReport(in)
		if err != nil {
			t.Fatalf("%s: encode: %v", in.Kind, err)
		}
		out, err := DecodeReport(payload)
		if err != nil {
			t.Fatalf("%s: decode: %v", in.Kind, err)
		}
		if out.Kind != in.Kind || out.GatewayID != gw || !out.ObservedAt.Equal(at) {
			t.Fatalf("%s: header = %+v", in.Kind, out)
		}
		switch items := out.Items.(type) {
		case []SolvencyItem:
			want := in.Items.([]SolvencyItem)
			if len(items) != len(want) || items[0].Liabilities.Cmp(want[0].Liabilities) != 0 || items[1].Decimals != 9 || LabelFromKey(items[0].Asset) != "USDC.SOLANA" {
				t.Fatalf("solvency items = %+v", items)
			}
		case []DepositItem:
			want := in.Items.([]DepositItem)
			if len(items) != 1 || items[0] != *(&DepositItem{want[0].DepositID, want[0].ChainID, want[0].TxRef, want[0].Token, items[0].Amount, want[0].Destination, 99, VerdictConfirmed}) || items[0].Amount.Int64() != 42 {
				t.Fatalf("deposit items = %+v", items)
			}
		case []ConversionItem:
			if len(items) != 1 || items[0].DeviationBps.Int64() != -3 || items[0].RoundID.Int64() != 7 || items[0].Feed != common.HexToAddress("0x1") {
				t.Fatalf("conversion items = %+v", items)
			}
		default:
			t.Fatalf("%s: items %T", in.Kind, out.Items)
		}
	}
}

func TestDecodeReportRejectsGarbage(t *testing.T) {
	for _, bad := range [][]byte{nil, make([]byte, 10), append(make([]byte, 31), 9), make([]byte, 128)} {
		if _, err := DecodeReport(bad); err == nil {
			t.Errorf("%x decoded", bad)
		}
	}
	if _, err := EncodeReport(Report{Kind: KindSolvency, Items: []DepositItem{}}); err == nil {
		t.Error("mismatched items encoded")
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	m := Metadata{WorkflowID: SubjectKey("wf"), Owner: [20]byte{1, 2, 3}, ReportID: [2]byte{0, 7}}
	copy(m.WorkflowName[:], "solvency")
	got, err := DecodeMetadata(m.Encode())
	if err != nil || got != m {
		t.Fatalf("got %+v err %v", got, err)
	}
	if _, err := DecodeMetadata(make([]byte, 63)); err == nil {
		t.Fatal("short metadata accepted")
	}
}

func TestLabelKey(t *testing.T) {
	if LabelFromKey(LabelKey("USDC")) != "USDC" || LabelFromKey([32]byte{}) != "" {
		t.Fatal("short label round trip")
	}
	long := "a-label-that-is-longer-than-thirty-two-bytes"
	if LabelFromKey(LabelKey(long))[:2] != "0x" {
		t.Fatal("long label should hash")
	}
}
