# 12 BitGo custody adapter and conformance suite

Status: ready-for-agent
Owner: Blockchain engineer (Fable)
Blocked by: 08, 11

## Goal
`custody/bitgo` implementing the provider interface (wallet per merchant, address derivation, transfer proposals mapped to BitGo approvals, webhook verification). Complete the conformance suite and run it against mock, direct and bitgo (recorded fixtures). Document the provider switch.

## Acceptance
All three adapters pass the suite; switching provider by config is covered by a test that re-runs the demo flow.
