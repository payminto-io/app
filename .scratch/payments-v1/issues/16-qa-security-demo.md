# 16 QA: money-path tests, security review, demo script

Status: ready-for-agent
Owner: QA (Fable for the security review)
Blocked by: 11, 14, 15

## Goal
End-to-end tests of the demo: create link, pay by card, pay by USDC on Solana, both in the ledger, settle under policy, switch custody by config, repeat. Security review of tickets 01, 02, 08, 09, 11, 13 against a threat model written in `docs/THREAT_MODEL.md`. Demo script and README limits section.

## Acceptance
Threat model findings are tickets; no open high finding before any live merchant.
