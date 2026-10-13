// Package tempo reads TIP-20 TransferWithMemo payments on the Tempo chain.
//
// Tempo blocks carry type 0x76 transactions that go-ethereum's block decoder does not know, so this
// package never fetches whole blocks: it reads logs (eth_getLogs) and receipts only, which decode fine.
// Tempo has deterministic finality (about 0.5s), so a log in a block at or below the head is final.
//
// A payment is a TIP-20 transferWithMemo to the merchant's receiving address whose 32-byte memo
// carries the payment reference. A memo match is only a candidate: the credit is verified against the
// token allow-list, the recipient, a positive amount and a successful receipt before it is recorded.
package tempo

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ChainCode is the blockchains.code of the Tempo chain.
const ChainCode = "TEMPO"

// Moderato testnet constants.
const (
	ModeratoChainID  = 42431
	ModeratoRPC      = "https://rpc.moderato.tempo.xyz"
	ModeratoExplorer = "https://explore.testnet.tempo.xyz"
)

// Well-known TIP-20 test stablecoins on Moderato (6 decimals).
var (
	PathUSD  = common.HexToAddress("0x20c0000000000000000000000000000000000000")
	AlphaUSD = common.HexToAddress("0x20c0000000000000000000000000000000000001")
)

// TransferWithMemoTopic is keccak256("TransferWithMemo(address,address,uint256,bytes32)").
// All of from, to and memo are indexed; data holds the amount.
var TransferWithMemoTopic = crypto.Keccak256Hash([]byte("TransferWithMemo(address,address,uint256,bytes32)"))

// MemoTransfer is one decoded TransferWithMemo log.
type MemoTransfer struct {
	Token       common.Address
	From        common.Address
	To          common.Address
	Amount      *big.Int // raw base units
	Memo        [32]byte
	TxHash      common.Hash
	LogIndex    uint
	BlockNumber uint64
	BlockHash   common.Hash
	Removed     bool
}

// Key is the unique identity of a credit: one log in one transaction.
func (m MemoTransfer) Key() string {
	return fmt.Sprintf("%s:%d", strings.ToLower(m.TxHash.Hex()), m.LogIndex)
}

// ErrNotMemoTransfer is returned for a log that is not a TransferWithMemo event.
var ErrNotMemoTransfer = errors.New("tempo: log is not TransferWithMemo")

// DecodeLog decodes a TransferWithMemo log.
func DecodeLog(l types.Log) (MemoTransfer, error) {
	if len(l.Topics) != 4 || l.Topics[0] != TransferWithMemoTopic {
		return MemoTransfer{}, ErrNotMemoTransfer
	}
	if len(l.Data) != 32 {
		return MemoTransfer{}, fmt.Errorf("tempo: TransferWithMemo data is %d bytes, want 32", len(l.Data))
	}
	return MemoTransfer{
		Token:       l.Address,
		From:        common.BytesToAddress(l.Topics[1].Bytes()),
		To:          common.BytesToAddress(l.Topics[2].Bytes()),
		Amount:      new(big.Int).SetBytes(l.Data),
		Memo:        l.Topics[3],
		TxHash:      l.TxHash,
		LogIndex:    l.Index,
		BlockNumber: l.BlockNumber,
		BlockHash:   l.BlockHash,
		Removed:     l.Removed,
	}, nil
}

// MemoForReference returns the memo text a payer sends for a payment reference. Memos are UTF-8,
// left-aligned and zero-padded to 32 bytes. A UUID reference (36 chars) does not fit, so it is sent as
// its 32 hex digits without dashes; any other reference is sent as-is and must fit in 32 bytes.
func MemoForReference(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("tempo: empty payment reference")
	}
	if u, err := uuid.Parse(ref); err == nil && len(ref) == 36 {
		return hex.EncodeToString(u[:]), nil
	}
	if len(ref) > 32 {
		return "", fmt.Errorf("tempo: reference %q is %d bytes, memo holds 32", ref, len(ref))
	}
	return ref, nil
}

// EncodeMemo turns a payment reference into the bytes32 memo: MemoForReference's UTF-8 bytes,
// left-aligned and zero-padded ("PM-1042" -> 0x504d2d31303432 then zeros).
func EncodeMemo(ref string) ([32]byte, error) {
	var out [32]byte
	text, err := MemoForReference(ref)
	if err != nil {
		return out, err
	}
	copy(out[:], text)
	return out, nil
}

// EncodeMemoHex is EncodeMemo as a 0x-prefixed hex string (what a wallet or cast takes).
func EncodeMemoHex(ref string) (string, error) {
	m, err := EncodeMemo(ref)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(m[:]), nil
}

// MemoCandidates returns the payment references a memo may stand for: the memo's UTF-8 text (zero
// padding trimmed) and, when that text is 32 hex digits, the dashed UUID it encodes. Binary memos that
// are not printable UTF-8 yield nothing. The caller looks the candidates up; a miss is not an error.
func MemoCandidates(memo [32]byte) []string {
	text := bytes.TrimRight(memo[:], "\x00")
	if len(text) == 0 || bytes.Contains(text, []byte{0}) || !utf8.Valid(text) || !printable(text) {
		return nil
	}
	out := []string{string(text)}
	if len(text) == 32 {
		if raw, err := hex.DecodeString(string(text)); err == nil {
			if u, err := uuid.FromBytes(raw); err == nil {
				out = append(out, u.String())
			}
		}
	}
	return out
}

func printable(b []byte) bool {
	for _, r := range string(b) {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// Token is one accepted TIP-20 token.
type Token struct {
	Address              common.Address
	Decimals             int32
	BlockchainCurrencyID uint
	Code                 string // e.g. PATHUSD
}

// Rejection says why a candidate credit was not accepted.
type Rejection string

const (
	RejectNone           Rejection = ""
	RejectRemoved        Rejection = "log_removed"
	RejectUnknownToken   Rejection = "unknown_token"
	RejectWrongRecipient Rejection = "wrong_recipient"
	RejectZeroAmount     Rejection = "zero_amount"
	RejectTxFailed       Rejection = "tx_failed"
	RejectNoReference    Rejection = "no_reference"
)

// Verify checks a decoded transfer against the token allow-list, the merchant address and the receipt
// status (1 = success). It returns the matched token and the amount in whole units.
func Verify(t MemoTransfer, tokens map[common.Address]Token, merchant common.Address, receiptStatus uint64) (Token, decimal.Decimal, Rejection) {
	if t.Removed {
		return Token{}, decimal.Zero, RejectRemoved
	}
	tok, ok := tokens[t.Token]
	if !ok {
		return Token{}, decimal.Zero, RejectUnknownToken
	}
	if t.To != merchant {
		return tok, decimal.Zero, RejectWrongRecipient
	}
	if t.Amount == nil || t.Amount.Sign() <= 0 {
		return tok, decimal.Zero, RejectZeroAmount
	}
	if receiptStatus != types.ReceiptStatusSuccessful {
		return tok, decimal.Zero, RejectTxFailed
	}
	if len(MemoCandidates(t.Memo)) == 0 {
		return tok, decimal.Zero, RejectNoReference
	}
	return tok, decimal.NewFromBigInt(t.Amount, -tok.Decimals), RejectNone
}

// Dedupe drops repeated (tx hash, log index) pairs, keeping first occurrence order.
func Dedupe(in []MemoTransfer) []MemoTransfer {
	seen := make(map[string]bool, len(in))
	out := in[:0:0]
	for _, t := range in {
		k := t.Key()
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t)
	}
	return out
}

// ExplorerTxURL links a transaction on the Moderato explorer.
func ExplorerTxURL(hash string) string { return ModeratoExplorer + "/tx/" + hash }
