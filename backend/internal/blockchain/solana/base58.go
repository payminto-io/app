package solana

import (
	"encoding/base64"

	"github.com/btcsuite/btcd/btcutil/base58"
)

func encodeBase58(b []byte) string { return base58.Encode(b) }

func decodeBase58(s string) []byte { return base58.Decode(s) }

func encodeBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
