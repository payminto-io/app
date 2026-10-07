# 28 CRE Solana receiver program for mirrored attestations

Status: ready-for-agent
Owner: Solana engineer (Fable) with Chainlink engineer (Fable)
Blocked by: 09, 23

## Goal
An Anchor program `gateway_attestations` with an `on_report` instruction invoked by the Keystone Forwarder program, accepting a Borsh payload within the 265-byte Solana report limit: `(kind u8, gateway_id [u8;32], observed_at u64, checkpoint_or_subject [u8;32], liabilities u64, reserves u64, asset [u8;8])`, storing the latest per `(gateway_id, asset)` PDA and emitting an event. Extend the `solvency` workflow with a second write target (Solana devnet, then mainnet) through `cre generate-bindings solana`, so a Solana-native merchant can verify the solvency record on the chain it settles on. Gateway verifier gains a Solana event reader mirroring the EVM path.

## Acceptance
Anchor tests for forwarder-only CPI and payload bounds; `cre workflow simulate` of the solvency workflow with the Solana target dry-runs the write; the public verification page shows both chain records for one attestation.
