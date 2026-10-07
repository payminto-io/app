# 21b CRE settlement gate

Status: ready-for-agent
Owner: Chainlink engineer (Fable) with Blockchain engineer (Fable)
Blocked by: 11, 21, 24

## Goal
Enforce the deposit-finality attestation inside settlement: when a deployer sets `require_attestation_above` in settlement policy, a settlement run above that amount waits for a verified finality attestation from the CRE module (ticket 21's `SettlementGate` port). Fails open by default, closed only when the policy line is set. See `docs/cre/SPEC.md`.

## Acceptance
Policy off: settlement unchanged. Policy on: run above threshold blocked until attestation verified; attestation mismatch raises an anomaly and blocks; CRE down with policy on blocks and alerts; tests for each.
