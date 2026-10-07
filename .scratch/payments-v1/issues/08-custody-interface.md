# 08 Custody provider interface, mock and direct-to-wallet

Status: ready-for-agent
Owner: Blockchain engineer (Fable)
Blocked by: 01

## Goal
`backend/internal/custody/`: `Provider` with `Capabilities()`, `DeriveAddress(ctx, chain, asset, merchant)`, `ListTransfers`/`VerifyWebhook`, `ProposeTransfer(ctx, Transfer) (ProposalID, error)`, `Status(ctx, ProposalID)`. One provider per account. Implementations: `mock` (in-memory, scriptable failures) and `direct` (merchant-supplied destination, funds never held; wraps Payminto's existing xpub/SCW address derivation and sweep). Policy checks (caps, allow-list, approvals) live in ticket 11 and run before any provider call.

## Acceptance
Conformance suite skeleton in `custody/conformance/` run against mock and direct: replayed webhook, regressed status, failed broadcast, outage mid-proposal. Signer keys never reachable from the API process (test: the API binary has no signer config).
