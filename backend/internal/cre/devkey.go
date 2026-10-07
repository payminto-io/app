package cre

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// DevKey signs mock attestations in-process. It is for the mock provider and tests only; the chainlink
// provider never holds key material (SPEC section 8).
type DevKey struct {
	key *ecdsa.PrivateKey
}

func NewDevKey() (*DevKey, error) {
	k, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("cre: generate dev key: %w", err)
	}
	return &DevKey{key: k}, nil
}

func (d *DevKey) Address() common.Address { return crypto.PubkeyToAddress(d.key.PublicKey) }

// Sign produces the 65-byte signature evidence over SigningHash(metadata, report).
func (d *DevKey) Sign(metadata, report []byte) ([]byte, error) {
	return crypto.Sign(SigningHash(metadata, report), d.key)
}
