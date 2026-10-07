# 09 Solana adapter and USDC deposits

Status: ready-for-agent
Owner: Solana engineer (Fable)
Blocked by: 08

## Goal
`backend/internal/blockchain/solana/` implementing Payminto's chain adapter: per-payment deposit addresses (associated token accounts for USDC), slot watcher with `confirmed` and `finalized` commitment, under and over payment handling, fee sponsorship for sweeps, durable nonces for retries, RPC pool. USDC mint per environment (devnet and mainnet).

## Acceptance
Devnet integration test with a funded keypair from env (skipped if absent); unit tests with recorded RPC responses; a USDC deposit becomes a succeeded attempt and ledger journal; reorg at `confirmed` reverses correctly before `finalized`.
