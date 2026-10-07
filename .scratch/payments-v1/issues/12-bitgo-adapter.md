# 12 BitGo custody provider with EVM and Solana, plus Solana in custody

Status: ready-for-agent
Owner: Blockchain engineer with Solana engineer
Blocked by: 08, 09

## Goal
Owner direction 2026-10-07: custody must offer BitGo as a custodian that generates wallets and addresses, manages them, and executes payouts and settlement, and custody must support Solana. Today the custody module (ticket 08) only covers EVM through the direct provider and has no Solana path.

1. **Solana in custody.** The direct provider gains Solana: deposit addresses from the Solana adapter (ticket 09), payouts of USDC and USDT as SPL transfers with the fee payer, the same claim, fence, send-row-before-broadcast and reconciler rules as EVM, finality as finalized commitment.
2. **BitGo provider** (`custody/bitgo`), ported from Kuberopay's `wallet-providers/bitgo.provider.ts` and `processors/bitgo-custody.adapter.ts` (owner decision in CLAUDE.md): wallet and address generation per merchant and chain (EVM chains and Solana, test coins `hteth`/`tsol`, live `eth`/`sol`, token ids for USDC/USDT), transfer proposals mapped to BitGo transfers with approvals, status from BitGo with webhook verification, policy decisions enforced in core before any BitGo call, BitGo access token and wallet passphrases only from configuration or the secrets vault.
3. **Dynamic selection per merchant.** Owner direction: custody must not be fixed at boot. Each merchant chooses its custodian at runtime in dashboard settings (`direct` self-custody with locally generated wallets, or `bitgo`; Fireblocks later through the same interface), stored as a versioned custody account per merchant and environment. The deployment config only lists which providers are available (`CUSTODY_PROVIDERS_AVAILABLE`); live refuses mock. Switching custodian applies to new addresses and payouts; existing balances stay attributed to the custodian that holds them, with a documented migration step for sweeping them across. The dashboard shows the active custodian, its health and the wallets it manages.
4. **Conformance.** The custody conformance suite runs against mock, direct (EVM and Solana) and bitgo (against a local fake BitGo API built from the SDK's request and response shapes; no real BitGo calls in tests).

## Acceptance
Every provider passes the suite; a payout on Solana through direct and through bitgo each produce exactly one custody outcome with a tx hash; deposit addresses on Solana and Base from bitgo; no credential ever logged; switching provider by configuration is covered by a test.
