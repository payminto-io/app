# 09 Solana adapter with USDC and USDT side by side

Status: ready-for-agent
Owner: Solana engineer (Fable)
Blocked by: 08

## Goal
`backend/internal/blockchain/solana/` implementing Payminto's chain adapter: per-payment deposit addresses (associated token accounts) for both USDC and USDT as peer SPL assets, selected per payment, each with its own mint per environment, slot watcher with `confirmed` and `finalized` commitment, under and over payment handling, fee sponsorship for sweeps, durable nonces for retries, RPC pool. USDC and USDT mints per environment (devnet and mainnet), configured as data in the seeds, not hard-coded; Token-2022 mints rejected unless explicitly allowed. Both assets appear side by side in checkout network/asset choice and in the ledger as `USDC.SOLANA` and `USDT.SOLANA`.

## Acceptance
Devnet integration test with a funded keypair from env (skipped if absent); unit tests with recorded RPC responses; a USDC deposit and a USDT deposit each become a succeeded attempt and a ledger journal in their own asset; sending USDT to a USDC payment address is detected and recorded as a wrong-asset anomaly, never credited as USDC; reorg at `confirmed` reverses correctly before `finalized`.
