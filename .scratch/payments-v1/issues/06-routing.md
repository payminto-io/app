# 06 Routing engine

Status: ready-for-agent
Owner: Principal engineer (Fable)
Blocked by: 05

## Goal
`backend/internal/routing/`: given an intent (amount, currency, method, country, settlement preference) and the merchant's enabled connectors, choose the attempt's connector. Algorithms, from Hyperswitch: rule-based (ordered conditions), volume split (weighted), success-rate based (rolling window per connector and method). Our additions: rail cost from fee rules (ticket 02), expected finality time, and the merchant's settlement preference (prefer the rail that settles in the chosen asset without conversion). Retry to the next candidate on connector failure.

## Acceptance
Deterministic tests per algorithm; success-rate window tested with injected history; a USDC-settling merchant routes a stablecoin-capable intent to the chain connector; decisions recorded on the attempt with reason.
