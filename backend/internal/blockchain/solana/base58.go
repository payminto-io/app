package solana

import "github.com/btcsuite/btcd/btcutil/base58"

func encodeBase58(b []byte) string { return base58.Encode(b) }

func decodeBase58(s string) []byte { return base58.Decode(s) }
