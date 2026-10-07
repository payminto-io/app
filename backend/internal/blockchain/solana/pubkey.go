// Package solana implements blockchain.ChainAdapter for Solana: SPL token deposits (USDC and USDT
// as peer assets), signature-cursor detection, fee-sponsored batched sweeps. No Solana SDK; the
// wire formats here are small and stable. Design: .scratch/payments-v1/issues/09-solana-usdc.md.
package solana

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
	"github.com/btcsuite/btcd/btcutil/base58"
)

// PublicKey is a 32-byte ed25519 public key or program-derived address.
type PublicKey [32]byte

// Well-known program ids.
var (
	SystemProgram          = MustPublicKey("11111111111111111111111111111111")
	TokenProgram           = MustPublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	Token2022Program       = MustPublicKey("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")
	AssociatedTokenProgram = MustPublicKey("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	ComputeBudgetProgram   = MustPublicKey("ComputeBudget111111111111111111111111111111")
	SysvarRent             = MustPublicKey("SysvarRent111111111111111111111111111111111")
)

// ParsePublicKey decodes a base58 public key.
func ParsePublicKey(s string) (PublicKey, error) {
	var pk PublicKey
	raw := base58.Decode(s)
	if len(raw) != 32 {
		return pk, fmt.Errorf("solana: %q is not a 32-byte base58 public key", s)
	}
	copy(pk[:], raw)
	return pk, nil
}

// MustPublicKey is ParsePublicKey for constants.
func MustPublicKey(s string) PublicKey {
	pk, err := ParsePublicKey(s)
	if err != nil {
		panic(err)
	}
	return pk
}

// PublicKeyFromBytes copies a 32-byte slice.
func PublicKeyFromBytes(b []byte) (PublicKey, error) {
	var pk PublicKey
	if len(b) != 32 {
		return pk, fmt.Errorf("solana: public key must be 32 bytes, got %d", len(b))
	}
	copy(pk[:], b)
	return pk, nil
}

func (pk PublicKey) String() string { return base58.Encode(pk[:]) }

// IsZero reports the all-zero key.
func (pk PublicKey) IsZero() bool { return pk == PublicKey{} }

// IsOnCurve reports whether the bytes decode to an edwards25519 point; PDAs must not.
func (pk PublicKey) IsOnCurve() bool {
	_, err := new(edwards25519.Point).SetBytes(pk[:])
	return err == nil
}

// ErrNoProgramAddress is returned when no bump yields an off-curve address (practically unreachable).
var ErrNoProgramAddress = errors.New("solana: unable to find a viable program address bump seed")

// CreateProgramAddress hashes seeds and program id per the runtime and refuses on-curve results.
func CreateProgramAddress(seeds [][]byte, program PublicKey) (PublicKey, error) {
	h := sha256.New()
	for _, s := range seeds {
		if len(s) > 32 {
			return PublicKey{}, fmt.Errorf("solana: seed longer than 32 bytes")
		}
		h.Write(s)
	}
	h.Write(program[:])
	h.Write([]byte("ProgramDerivedAddress"))
	var pk PublicKey
	copy(pk[:], h.Sum(nil))
	if pk.IsOnCurve() {
		return PublicKey{}, errors.New("solana: invalid seeds, address must fall off the curve")
	}
	return pk, nil
}

// FindProgramAddress returns the first off-curve address and its bump, searching from 255 down.
func FindProgramAddress(seeds [][]byte, program PublicKey) (PublicKey, uint8, error) {
	for bump := 255; bump >= 0; bump-- {
		withBump := append(append([][]byte{}, seeds...), []byte{byte(bump)})
		pk, err := CreateProgramAddress(withBump, program)
		if err == nil {
			return pk, uint8(bump), nil
		}
	}
	return PublicKey{}, 0, ErrNoProgramAddress
}

// AssociatedTokenAddress derives the canonical token account for owner and mint under tokenProgram.
func AssociatedTokenAddress(owner, mint, tokenProgram PublicKey) (PublicKey, error) {
	pk, _, err := FindProgramAddress([][]byte{owner[:], tokenProgram[:], mint[:]}, AssociatedTokenProgram)
	return pk, err
}

// TokenProgramFor maps a blockchain_currencies.standard value to the token program that owns the mint.
func TokenProgramFor(standard string) (PublicKey, error) {
	switch standard {
	case "SPL", "spl", "SPL-Token", "spl-token":
		return TokenProgram, nil
	case "SPL-2022", "spl-2022", "Token-2022", "token-2022":
		return Token2022Program, nil
	}
	return PublicKey{}, fmt.Errorf("solana: unknown token standard %q", standard)
}
