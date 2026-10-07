package cre

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Wire format of every report (SPEC section 5, contracts/test/cre/ReportEncoder.sol):
// abi.encode(uint8 version=1, uint8 kind, bytes32 gatewayId, uint64 observedAt, Item[] items).
// The item tuple depends on the kind. Workflows and the consumer contract encode exactly this.

// ReportVersion is the only version the contract and this verifier accept.
const ReportVersion uint8 = 1

// SolvencyItem is one asset of a solvency report.
type SolvencyItem struct {
	CheckpointHash [32]byte
	Asset          [32]byte
	Liabilities    *big.Int
	Reserves       *big.Int
	Decimals       uint8
}

// DepositItem is one deposit of a deposit-finality report.
type DepositItem struct {
	DepositID   [32]byte
	ChainID     [32]byte
	TxRef       [32]byte
	Token       [32]byte
	Amount      *big.Int
	Destination [32]byte
	SlotOrBlock uint64
	Verdict     uint8
}

// ConversionItem is one trade of a conversion-reference report.
type ConversionItem struct {
	ConversionID      [32]byte
	Pair              [32]byte
	ReferenceRate     *big.Int
	ReferenceDecimals uint8
	DeviationBps      *big.Int
	Feed              common.Address
	RoundID           *big.Int
}

// Report is a decoded payload.
type Report struct {
	Kind       Kind
	GatewayID  [32]byte
	ObservedAt time.Time
	// Items is []SolvencyItem, []DepositItem or []ConversionItem.
	Items any
}

var (
	solvencyArgs   = mustArgs(`[{"name":"checkpointHash","type":"bytes32"},{"name":"asset","type":"bytes32"},{"name":"liabilities","type":"uint256"},{"name":"reserves","type":"uint256"},{"name":"decimals","type":"uint8"}]`)
	depositArgs    = mustArgs(`[{"name":"depositId","type":"bytes32"},{"name":"chainId","type":"bytes32"},{"name":"txRef","type":"bytes32"},{"name":"token","type":"bytes32"},{"name":"amount","type":"uint256"},{"name":"destination","type":"bytes32"},{"name":"slotOrBlock","type":"uint64"},{"name":"verdict","type":"uint8"}]`)
	conversionArgs = mustArgs(`[{"name":"conversionId","type":"bytes32"},{"name":"pair","type":"bytes32"},{"name":"referenceRate","type":"int256"},{"name":"referenceDecimals","type":"uint8"},{"name":"deviationBps","type":"int256"},{"name":"feed","type":"address"},{"name":"roundId","type":"uint80"}]`)
)

func mustArgs(components string) abi.Arguments {
	version, err := abi.NewType("uint8", "", nil)
	if err != nil {
		panic(err)
	}
	kind, err := abi.NewType("uint8", "", nil)
	if err != nil {
		panic(err)
	}
	gateway, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		panic(err)
	}
	observed, err := abi.NewType("uint64", "", nil)
	if err != nil {
		panic(err)
	}
	var comps []abi.ArgumentMarshaling
	if err := jsonUnmarshal(components, &comps); err != nil {
		panic(err)
	}
	items, err := abi.NewType("tuple[]", "", comps)
	if err != nil {
		panic(err)
	}
	return abi.Arguments{{Name: "version", Type: version}, {Name: "kind", Type: kind}, {Name: "gatewayId", Type: gateway}, {Name: "observedAt", Type: observed}, {Name: "items", Type: items}}
}

func argsFor(k Kind) (abi.Arguments, error) {
	switch k {
	case KindSolvency:
		return solvencyArgs, nil
	case KindDepositFinality:
		return depositArgs, nil
	case KindConversionReference:
		return conversionArgs, nil
	}
	return nil, fmt.Errorf("%w: unknown kind %q", ErrInvalidReport, k)
}

// EncodeReport produces the on-chain payload for a report.
func EncodeReport(r Report) ([]byte, error) {
	args, err := argsFor(r.Kind)
	if err != nil {
		return nil, err
	}
	var items any
	switch v := r.Items.(type) {
	case []SolvencyItem:
		if r.Kind != KindSolvency {
			return nil, fmt.Errorf("%w: solvency items under kind %s", ErrInvalidReport, r.Kind)
		}
		items = toSolvencyABI(v)
	case []DepositItem:
		if r.Kind != KindDepositFinality {
			return nil, fmt.Errorf("%w: deposit items under kind %s", ErrInvalidReport, r.Kind)
		}
		items = toDepositABI(v)
	case []ConversionItem:
		if r.Kind != KindConversionReference {
			return nil, fmt.Errorf("%w: conversion items under kind %s", ErrInvalidReport, r.Kind)
		}
		items = toConversionABI(v)
	default:
		return nil, fmt.Errorf("%w: items of type %T", ErrInvalidReport, r.Items)
	}
	return args.Pack(ReportVersion, r.Kind.Code(), r.GatewayID, uint64(r.ObservedAt.Unix()), items)
}

// DecodeReport decodes a payload; version and kind are read from the first two words.
func DecodeReport(payload []byte) (Report, error) {
	if len(payload) < 32*5 {
		return Report{}, fmt.Errorf("%w: payload shorter than its header", ErrInvalidReport)
	}
	if v := new(big.Int).SetBytes(payload[:32]); !v.IsUint64() || v.Uint64() != uint64(ReportVersion) {
		return Report{}, fmt.Errorf("%w: report version %s, want %d", ErrInvalidReport, v, ReportVersion)
	}
	code := new(big.Int).SetBytes(payload[32:64])
	if !code.IsUint64() || code.Uint64() > 255 {
		return Report{}, fmt.Errorf("%w: kind word out of range", ErrInvalidReport)
	}
	kind, ok := KindFromCode(uint8(code.Uint64()))
	if !ok {
		return Report{}, fmt.Errorf("%w: unknown kind code %d", ErrInvalidReport, code.Uint64())
	}
	args, _ := argsFor(kind)
	values, err := args.Unpack(payload)
	if err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	r := Report{Kind: kind, GatewayID: values[2].([32]byte), ObservedAt: time.Unix(int64(values[3].(uint64)), 0).UTC()}
	switch kind {
	case KindSolvency:
		r.Items = fromSolvencyABI(values[4])
	case KindDepositFinality:
		r.Items = fromDepositABI(values[4])
	case KindConversionReference:
		r.Items = fromConversionABI(values[4])
	}
	return r, nil
}

// PayloadHash is keccak256 of the encoded payload; it keys replay detection.
func PayloadHash(payload []byte) []byte { return crypto.Keccak256(payload) }

// GatewayID is keccak256 of the deployment's public base URL (SPEC section 5).
func GatewayID(publicBaseURL string) [32]byte {
	var out [32]byte
	copy(out[:], crypto.Keccak256([]byte(publicBaseURL)))
	return out
}

// SubjectKey is the bytes32 form of a subject id: keccak256 of the id string.
func SubjectKey(id string) [32]byte {
	var out [32]byte
	copy(out[:], crypto.Keccak256([]byte(id)))
	return out
}

// LabelKey is the bytes32 of a short label (asset code, pair, chain), right-padded; longer labels are hashed.
func LabelKey(label string) [32]byte {
	var out [32]byte
	if len(label) <= 32 {
		copy(out[:], label)
		return out
	}
	return SubjectKey(label)
}

// LabelFromKey reverses LabelKey for short labels; hashed labels come back as hex.
func LabelFromKey(key [32]byte) string {
	n := 0
	for n < 32 && key[n] != 0 {
		n++
	}
	if n == 0 {
		return ""
	}
	for i := n; i < 32; i++ {
		if key[i] != 0 {
			return "0x" + common.Bytes2Hex(key[:])
		}
	}
	s := string(key[:n])
	for _, c := range s {
		if c < 0x20 || c > 0x7e {
			return "0x" + common.Bytes2Hex(key[:])
		}
	}
	return s
}

// The abi package needs anonymous structs whose fields match the tuple components by name.

type solvencyABI struct {
	CheckpointHash [32]byte
	Asset          [32]byte
	Liabilities    *big.Int
	Reserves       *big.Int
	Decimals       uint8
}

type depositABI struct {
	DepositId   [32]byte
	ChainId     [32]byte
	TxRef       [32]byte
	Token       [32]byte
	Amount      *big.Int
	Destination [32]byte
	SlotOrBlock uint64
	Verdict     uint8
}

type conversionABI struct {
	ConversionId      [32]byte
	Pair              [32]byte
	ReferenceRate     *big.Int
	ReferenceDecimals uint8
	DeviationBps      *big.Int
	Feed              common.Address
	RoundId           *big.Int
}

func orZero(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v
}

func toSolvencyABI(in []SolvencyItem) []solvencyABI {
	out := make([]solvencyABI, len(in))
	for i, it := range in {
		out[i] = solvencyABI{it.CheckpointHash, it.Asset, orZero(it.Liabilities), orZero(it.Reserves), it.Decimals}
	}
	return out
}

func toDepositABI(in []DepositItem) []depositABI {
	out := make([]depositABI, len(in))
	for i, it := range in {
		out[i] = depositABI{it.DepositID, it.ChainID, it.TxRef, it.Token, orZero(it.Amount), it.Destination, it.SlotOrBlock, it.Verdict}
	}
	return out
}

func toConversionABI(in []ConversionItem) []conversionABI {
	out := make([]conversionABI, len(in))
	for i, it := range in {
		out[i] = conversionABI{it.ConversionID, it.Pair, orZero(it.ReferenceRate), it.ReferenceDecimals, orZero(it.DeviationBps), it.Feed, orZero(it.RoundID)}
	}
	return out
}

func fromSolvencyABI(v any) []SolvencyItem {
	rows := v.([]struct {
		CheckpointHash [32]byte `json:"checkpointHash"`
		Asset          [32]byte `json:"asset"`
		Liabilities    *big.Int `json:"liabilities"`
		Reserves       *big.Int `json:"reserves"`
		Decimals       uint8    `json:"decimals"`
	})
	out := make([]SolvencyItem, len(rows))
	for i, r := range rows {
		out[i] = SolvencyItem{r.CheckpointHash, r.Asset, r.Liabilities, r.Reserves, r.Decimals}
	}
	return out
}

func fromDepositABI(v any) []DepositItem {
	rows := v.([]struct {
		DepositId   [32]byte `json:"depositId"`
		ChainId     [32]byte `json:"chainId"`
		TxRef       [32]byte `json:"txRef"`
		Token       [32]byte `json:"token"`
		Amount      *big.Int `json:"amount"`
		Destination [32]byte `json:"destination"`
		SlotOrBlock uint64   `json:"slotOrBlock"`
		Verdict     uint8    `json:"verdict"`
	})
	out := make([]DepositItem, len(rows))
	for i, r := range rows {
		out[i] = DepositItem{r.DepositId, r.ChainId, r.TxRef, r.Token, r.Amount, r.Destination, r.SlotOrBlock, r.Verdict}
	}
	return out
}

func fromConversionABI(v any) []ConversionItem {
	rows := v.([]struct {
		ConversionId      [32]byte       `json:"conversionId"`
		Pair              [32]byte       `json:"pair"`
		ReferenceRate     *big.Int       `json:"referenceRate"`
		ReferenceDecimals uint8          `json:"referenceDecimals"`
		DeviationBps      *big.Int       `json:"deviationBps"`
		Feed              common.Address `json:"feed"`
		RoundId           *big.Int       `json:"roundId"`
	})
	out := make([]ConversionItem, len(rows))
	for i, r := range rows {
		out[i] = ConversionItem{r.ConversionId, r.Pair, r.ReferenceRate, r.ReferenceDecimals, r.DeviationBps, r.Feed, r.RoundId}
	}
	return out
}
