# 09 Solana adapter with USDC and USDT side by side

Status: done
Owner: Solana engineer (Fable)
Blocked by: 08

## Goal
`backend/internal/blockchain/solana/` implementing Payminto's chain adapter: per-payment deposit addresses (associated token accounts) for both USDC and USDT as peer SPL assets, selected per payment, each with its own mint per environment, slot watcher with `confirmed` and `finalized` commitment, under and over payment handling, fee sponsorship for sweeps, durable nonces for retries, RPC pool. USDC and USDT mints per environment (devnet and mainnet), configured as data in the seeds, not hard-coded; Token-2022 mints rejected unless explicitly allowed. Both assets appear side by side in checkout network/asset choice and in the ledger as `USDC.SOLANA` and `USDT.SOLANA`.

## Acceptance
Devnet integration test with a funded keypair from env (skipped if absent); unit tests with recorded RPC responses; a USDC deposit and a USDT deposit each become a succeeded attempt and a ledger journal in their own asset; sending USDT to a USDC payment address is detected and recorded as a wrong-asset anomaly, never credited as USDC; reorg at `confirmed` reverses correctly before `finalized`.

## Comments

Done on branch `solana` (worktree `gateway-wt-solana`). Commits: `71b2fec` (SLIP-0010 derivation), `b711c0e` (keys, PDA/ATA, messages, instructions), `c99a501` (adapter, RPC client, parser, sweep builder, fixtures), `da716f9` (watcher, sweeper, owner/ATA records, seeds, wiring), `2aac9bd` (service tests), `24fcfe5` (validator end-to-end test, rent booking, README), plus the commit adding `.superpowers/solana-report.md` and this note. Full report: `.superpowers/solana-report.md`. Devnet USDT: no official mint exists; the seed row ships disabled and `SOLANA_DEVNET_USDT_MINT` enables it.
