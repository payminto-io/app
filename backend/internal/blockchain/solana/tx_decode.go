package solana

import (
	"encoding/base64"
	"errors"
	"fmt"
)

// DecodeTransaction parses a serialized legacy transaction; tests and reconciliation use it to
// inspect what was broadcast.
func DecodeTransaction(raw []byte) (Transaction, error) {
	var tx Transaction
	n, used, err := decodeCompactU16(raw)
	if err != nil {
		return tx, err
	}
	raw = raw[used:]
	for i := 0; i < n; i++ {
		if len(raw) < 64 {
			return tx, errors.New("solana: truncated signatures")
		}
		tx.Signatures = append(tx.Signatures, append([]byte(nil), raw[:64]...))
		raw = raw[64:]
	}
	msg, err := DecodeMessage(raw)
	if err != nil {
		return tx, err
	}
	tx.Message = msg
	return tx, nil
}

// DecodeTransactionBase64 is DecodeTransaction for the sendTransaction wire encoding.
func DecodeTransactionBase64(s string) (Transaction, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Transaction{}, err
	}
	return DecodeTransaction(raw)
}

// DecodeMessage parses the legacy message wire format.
func DecodeMessage(raw []byte) (Message, error) {
	var m Message
	if len(raw) < 3 {
		return m, errors.New("solana: truncated message header")
	}
	if raw[0]&0x80 != 0 {
		return m, errors.New("solana: versioned messages are not supported")
	}
	m.NumRequiredSignatures, m.NumReadonlySignedAccounts, m.NumReadonlyUnsignedAccounts = raw[0], raw[1], raw[2]
	raw = raw[3:]
	n, used, err := decodeCompactU16(raw)
	if err != nil {
		return m, err
	}
	raw = raw[used:]
	for i := 0; i < n; i++ {
		pk, err := PublicKeyFromBytes(take(&raw, 32))
		if err != nil {
			return m, fmt.Errorf("solana: account key %d: %w", i, err)
		}
		m.AccountKeys = append(m.AccountKeys, pk)
	}
	bh, err := PublicKeyFromBytes(take(&raw, 32))
	if err != nil {
		return m, fmt.Errorf("solana: blockhash: %w", err)
	}
	m.RecentBlockhash = bh
	n, used, err = decodeCompactU16(raw)
	if err != nil {
		return m, err
	}
	raw = raw[used:]
	for i := 0; i < n; i++ {
		if len(raw) < 1 {
			return m, errors.New("solana: truncated instruction")
		}
		ci := CompiledInstruction{ProgramIDIndex: raw[0]}
		raw = raw[1:]
		cnt, used, err := decodeCompactU16(raw)
		if err != nil {
			return m, err
		}
		raw = raw[used:]
		ci.AccountIndexes = append([]uint8(nil), take(&raw, cnt)...)
		dl, used, err := decodeCompactU16(raw)
		if err != nil {
			return m, err
		}
		raw = raw[used:]
		ci.Data = append([]byte(nil), take(&raw, dl)...)
		if len(ci.AccountIndexes) != cnt || len(ci.Data) != dl {
			return m, errors.New("solana: truncated instruction body")
		}
		m.Instructions = append(m.Instructions, ci)
	}
	return m, nil
}

func take(raw *[]byte, n int) []byte {
	if len(*raw) < n {
		out := *raw
		*raw = nil
		return out
	}
	out := (*raw)[:n]
	*raw = (*raw)[n:]
	return out
}

// Instruction expands a compiled instruction back to program id and account keys.
func (m Message) Instruction(ci CompiledInstruction) (Instruction, error) {
	if int(ci.ProgramIDIndex) >= len(m.AccountKeys) {
		return Instruction{}, errors.New("solana: program index out of range")
	}
	ix := Instruction{ProgramID: m.AccountKeys[ci.ProgramIDIndex], Data: ci.Data}
	for _, idx := range ci.AccountIndexes {
		if int(idx) >= len(m.AccountKeys) {
			return Instruction{}, errors.New("solana: account index out of range")
		}
		ix.Accounts = append(ix.Accounts, AccountMeta{Pubkey: m.AccountKeys[idx], Signer: int(idx) < int(m.NumRequiredSignatures), Writable: m.isWritable(int(idx))})
	}
	return ix, nil
}

func (m Message) isWritable(i int) bool {
	signed := int(m.NumRequiredSignatures)
	if i < signed {
		return i < signed-int(m.NumReadonlySignedAccounts)
	}
	return i < len(m.AccountKeys)-int(m.NumReadonlyUnsignedAccounts)
}
