# crypto

Owns Payminto's symmetric encryption and HD-wallet derivation primitives. Exposes `Encrypt`/`Decrypt` and `EncryptBytes`/`DecryptBytes` (AES-256-GCM with prepended nonce, hex-encoded output) used by the secrets vault for at-rest encryption of mnemonics and API keys, and BIP-39/BIP-32 helpers `NewMnemonic`, `SeedFromMnemonic`, plus extended-key derivation and per-chain address generation for Bitcoin and Ethereum. Depends on `btcsuite/btcd` (HD keychain, base58), `go-ethereum/crypto` (secp256k1, Keccak), and `tyler-smith/go-bip39`. Callers should never roll their own crypto — they should call into this package and nowhere else.

## Files

- `encryption.go` — AES-256-GCM encrypt/decrypt for string and byte keys.
- `hdwallet.go` — BIP-39 mnemonic, seed derivation, and HD key utilities.
- `encryption_test.go`, `hdwallet_test.go` — round-trip and known-vector tests.

## See also

- `internal/service/secrets_vault_service.go` — primary consumer
- `internal/models/secrets_vault.go` — encrypted field storage
