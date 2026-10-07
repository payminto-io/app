package solana

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
)

// AccountMeta is one account an instruction touches.
type AccountMeta struct {
	Pubkey   PublicKey
	Signer   bool
	Writable bool
}

// Instruction is a program call with its accounts and data.
type Instruction struct {
	ProgramID PublicKey
	Accounts  []AccountMeta
	Data      []byte
}

// Message is a compiled legacy (non-versioned) transaction message.
type Message struct {
	NumRequiredSignatures       uint8
	NumReadonlySignedAccounts   uint8
	NumReadonlyUnsignedAccounts uint8
	AccountKeys                 []PublicKey
	RecentBlockhash             PublicKey
	Instructions                []CompiledInstruction
}

// CompiledInstruction references accounts by index into Message.AccountKeys.
type CompiledInstruction struct {
	ProgramIDIndex uint8
	AccountIndexes []uint8
	Data           []byte
}

// Transaction is a message plus one signature per required signer, in account order.
type Transaction struct {
	Signatures [][]byte
	Message    Message
}

// Signer is anything that can sign a message for a public key; ed25519 private keys satisfy it.
type Signer interface {
	PublicKey() PublicKey
	Sign(message []byte) []byte
}

// Ed25519Signer wraps a 64-byte ed25519 private key.
type Ed25519Signer struct{ Key ed25519.PrivateKey }

// NewEd25519Signer validates the key length.
func NewEd25519Signer(priv []byte) (Ed25519Signer, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return Ed25519Signer{}, fmt.Errorf("solana: private key must be %d bytes, got %d", ed25519.PrivateKeySize, len(priv))
	}
	return Ed25519Signer{Key: ed25519.PrivateKey(append([]byte(nil), priv...))}, nil
}

func (s Ed25519Signer) PublicKey() PublicKey {
	pk, _ := PublicKeyFromBytes(s.Key.Public().(ed25519.PublicKey))
	return pk
}

func (s Ed25519Signer) Sign(message []byte) []byte { return ed25519.Sign(s.Key, message) }

// Zero wipes the key material.
func (s Ed25519Signer) Zero() {
	for i := range s.Key {
		s.Key[i] = 0
	}
}

type accountUse struct {
	key      PublicKey
	signer   bool
	writable bool
	order    int
}

// CompileMessage orders accounts per the runtime (fee payer, writable signers, readonly signers,
// writable non-signers, readonly non-signers) and compiles instructions against that table.
func CompileMessage(feePayer PublicKey, recentBlockhash PublicKey, instructions []Instruction) (Message, error) {
	if len(instructions) == 0 {
		return Message{}, errors.New("solana: message has no instructions")
	}
	uses := map[PublicKey]*accountUse{}
	order := 0
	note := func(pk PublicKey, signer, writable bool) {
		u, ok := uses[pk]
		if !ok {
			u = &accountUse{key: pk, order: order}
			order++
			uses[pk] = u
		}
		u.signer = u.signer || signer
		u.writable = u.writable || writable
	}
	note(feePayer, true, true)
	for _, ix := range instructions {
		for _, a := range ix.Accounts {
			note(a.Pubkey, a.Signer, a.Writable)
		}
		note(ix.ProgramID, false, false)
	}
	list := make([]*accountUse, 0, len(uses))
	for _, u := range uses {
		list = append(list, u)
	}
	rank := func(u *accountUse) int {
		switch {
		case u.key == feePayer:
			return 0
		case u.signer && u.writable:
			return 1
		case u.signer:
			return 2
		case u.writable:
			return 3
		default:
			return 4
		}
	}
	sort.Slice(list, func(i, j int) bool {
		ri, rj := rank(list[i]), rank(list[j])
		if ri != rj {
			return ri < rj
		}
		return list[i].order < list[j].order
	})
	msg := Message{RecentBlockhash: recentBlockhash}
	index := map[PublicKey]uint8{}
	for i, u := range list {
		if i > 255 {
			return Message{}, errors.New("solana: more than 256 accounts")
		}
		msg.AccountKeys = append(msg.AccountKeys, u.key)
		index[u.key] = uint8(i)
		if u.signer {
			msg.NumRequiredSignatures++
			if !u.writable {
				msg.NumReadonlySignedAccounts++
			}
		} else if !u.writable {
			msg.NumReadonlyUnsignedAccounts++
		}
	}
	for _, ix := range instructions {
		ci := CompiledInstruction{ProgramIDIndex: index[ix.ProgramID], Data: ix.Data}
		for _, a := range ix.Accounts {
			ci.AccountIndexes = append(ci.AccountIndexes, index[a.Pubkey])
		}
		msg.Instructions = append(msg.Instructions, ci)
	}
	return msg, nil
}

// Signers returns the account keys that must sign, in signature order.
func (m Message) Signers() []PublicKey {
	return m.AccountKeys[:m.NumRequiredSignatures]
}

// Serialize encodes the legacy message wire format.
func (m Message) Serialize() []byte {
	out := []byte{m.NumRequiredSignatures, m.NumReadonlySignedAccounts, m.NumReadonlyUnsignedAccounts}
	out = appendCompactU16(out, len(m.AccountKeys))
	for _, k := range m.AccountKeys {
		out = append(out, k[:]...)
	}
	out = append(out, m.RecentBlockhash[:]...)
	out = appendCompactU16(out, len(m.Instructions))
	for _, ix := range m.Instructions {
		out = append(out, ix.ProgramIDIndex)
		out = appendCompactU16(out, len(ix.AccountIndexes))
		out = append(out, ix.AccountIndexes...)
		out = appendCompactU16(out, len(ix.Data))
		out = append(out, ix.Data...)
	}
	return out
}

// Sign produces a transaction with a signature from every required signer; signers may be in any order.
func Sign(msg Message, signers []Signer) (Transaction, error) {
	byKey := map[PublicKey]Signer{}
	for _, s := range signers {
		byKey[s.PublicKey()] = s
	}
	raw := msg.Serialize()
	tx := Transaction{Message: msg}
	for _, pk := range msg.Signers() {
		s, ok := byKey[pk]
		if !ok {
			return Transaction{}, fmt.Errorf("solana: missing signer for %s", pk)
		}
		tx.Signatures = append(tx.Signatures, s.Sign(raw))
	}
	return tx, nil
}

// Serialize encodes signatures then message.
func (t Transaction) Serialize() []byte {
	out := appendCompactU16(nil, len(t.Signatures))
	for _, s := range t.Signatures {
		out = append(out, s...)
	}
	return append(out, t.Message.Serialize()...)
}

// Base64 is the encoding sendTransaction accepts.
func (t Transaction) Base64() string { return base64.StdEncoding.EncodeToString(t.Serialize()) }

// Signature is the first signature, which is the transaction id.
func (t Transaction) Signature() string {
	if len(t.Signatures) == 0 {
		return ""
	}
	return encodeBase58(t.Signatures[0])
}

// Verify checks every signature against its account key.
func (t Transaction) Verify() error {
	raw := t.Message.Serialize()
	signers := t.Message.Signers()
	if len(signers) != len(t.Signatures) {
		return fmt.Errorf("solana: %d signatures for %d signers", len(t.Signatures), len(signers))
	}
	for i, pk := range signers {
		if !ed25519.Verify(ed25519.PublicKey(pk[:]), raw, t.Signatures[i]) {
			return fmt.Errorf("solana: bad signature for %s", pk)
		}
	}
	return nil
}

// appendCompactU16 writes Solana's shortvec length prefix.
func appendCompactU16(out []byte, n int) []byte {
	v := uint16(n)
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v == 0 {
			return append(out, b)
		}
		out = append(out, b|0x80)
	}
}

// decodeCompactU16 reads a shortvec prefix; used by tests and the message decoder.
func decodeCompactU16(b []byte) (int, int, error) {
	var v, shift int
	for i := 0; i < 3 && i < len(b); i++ {
		v |= int(b[i]&0x7f) << shift
		if b[i]&0x80 == 0 {
			return v, i + 1, nil
		}
		shift += 7
	}
	return 0, 0, errors.New("solana: bad compact-u16")
}

// PackedSize is the serialized transaction length; the packet limit is 1232 bytes.
const MaxTransactionSize = 1232
