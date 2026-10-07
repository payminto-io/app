# Ticket 22 fix round 1: GatewayAttestations

Branch `cre-contract`, 2026-10-07, against `.superpowers/cre-contract-review.md`. No deploys, keys, networks or funds.

## Commits

| Hash | Content |
| --- | --- |
| `9b38d35` | `contracts/src/cre/WorkflowName.sol` + `test/cre/WorkflowName.t.sol`: Keystone `HashTruncateName`, pinned to `"my_workflow" -> 0x62373666336165316465` (H-1) |
| `0b43dfa` | Contract: replay per report hash, newest-wins solvency with `SolvencyIgnored`, `renounceOwnership` reverts, zero name rejected, item count bounded (M-1, L-1, L-2, L-4). Script: every `CRE_*` required, forwarder code check, deployer binds then `transferOwnership(CRE_CONSUMER_OWNER)` (L-3). Invariant handler with owner actions and ghost admin state, `afterInvariant` asserts the accept path ran (L-5). Gas rewrite measured after `vm.cool` (L-7). All tests use `WorkflowName`; one delivers the literal docs bytes |
| `52930f1` | `test/cre/vendor/keystone/` vendored from `smartcontractkit/chainlink-evm@b723176adfe8f2e9eff47730e21a1ac8f64b46d7` (`KeystoneForwarder.sol`, `IReceiver`, `IRouter`, `ITypeAndVersion`, `OwnerIsCreator` chain; imports rewritten only, fmt-ignored). `GatewayAttestations.forwarder.t.sol` signs raw reports with f+1 oracle keys and delivers through the real forwarder (L-6). ABI regenerated in all three places |
| (this commit) | SPEC section 5 final replay rule, seconds note, `reportHash` note, gas limits for ticket 23; `contracts/CLAUDE.md`; ticket; this report |

Vendoring, not a submodule: the forwarder tree is nine small files and the full `chainlink-evm` clone is not needed; the pinned commit is in each file header.

## Final replay rule per kind (also SPEC section 5)

- All kinds: `observedAt <= block.timestamp + 5 min`; `keccak256(report)` (receiver slice, `rawReport[109:]`) must be unseen, then marked seen. Second delivery of the same bytes reverts `DuplicateReport` regardless of execution id, report id or metadata. No ordering between distinct reports.
- Solvency (kind 1): per item, store only if `observedAt > latestSolvency[gatewayId][asset].observedAt`; else emit `SolvencyIgnored(gatewayId, asset, observedAt, latestObservedAt)` and skip; the report still succeeds. Duplicate asset in a batch keeps the first.
- Deposit finality (kind 2) and conversion reference (kind 3): events only, any order, once each.
- `latestObservedAt[gatewayId][kind]` = max accepted `observedAt`, informational.

## Checks

- `forge build`: clean (lint notes only).
- `forge test -vvv`: 9 suites, 80 passed, 0 failed. CRE: 47 unit/fuzz, 2 invariants (256 runs x 500 calls, owner actions included, every run accepted at least one valid report), 4 real-forwarder, 3 WorkflowName, 1 deploy script (sequential, env is process-wide), 5 gas.
- `forge fmt --check`: clean (vendor ignored by `foundry.toml`).
- Gas (`contracts/snapshots/GatewayAttestations.json`): deploy 1,688,110; solvency 20 first write 1,908,705; solvency 20 hourly rewrite with cold access 296,805; deposit 12 at 118,972; conversion 10 at 105,431. The seen-set adds one cold zero-to-nonzero slot (~22k) per report.

## Findings, one line each

- H-1 fixed: library + docs fixture + real-forwarder test that fails with the old derivation (`test_realForwarder_rejectsNameBoundWithWrongDerivation`). Script requires every name/id/owner and reverts on empty.
- M-1 fixed as ruled; see rule above.
- L-1 `renounceOwnership` reverts `RenounceDisabled`. L-2 `ZeroWorkflowName`. L-3 explicit owner, forwarder code check, deployer-then-transfer flow (the deployer key is not the multisig, so binding before transfer is the only order that works). L-4 `MalformedReport` before arithmetic, tested with `2^251`. L-5 owner actions and `afterInvariant`. L-6 real forwarder. L-7 `vm.cool` snapshot, keys renamed.
- I-3 addressed: `StaleSolvency` is gone, `SolvencyIgnored` describes what happens; seconds note added to SPEC. I-4 addressed: SPEC states `reportHash` is the receiver slice. I-6 addressed: gas limits for ticket 23 in SPEC. I-1, I-2, I-5: no change needed.

## Open

- Script deploy flow needs the multisig to call `acceptOwnership()`; the runbook step is in the script header and CLAUDE.md.
- Ticket 23 must encode `observedAt` in seconds and use `WorkflowName`-equivalent names (`cre workflow` names from `workflow.yaml`).
