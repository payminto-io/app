# 11 Settlement policy, destinations, approvals, runs

Status: ready-for-agent
Owner: Blockchain engineer (Fable)
Blocked by: 08, 10

## Goal
`backend/internal/settlement/`: per-merchant default and per-link override (fiat bank, USDC on chain, hold); verified destinations with cooling period (Kuberopay ADR 0023, 0024, 0027, 0033, 0034, 0056 as design); settlement runs by threshold, schedule or immediate; approval policy M-of-N with caps and kill switch; proposal to custody provider only after policy passes; finality confirmation and reconciliation against the ledger; fiat payout behind a `PayoutAdapter` with a mock.

## Acceptance
Policy rejects before provider is called (mock asserts not called); run reconciles to ledger; freeze on merchant restriction; hash-chained audit entries for approvals.
