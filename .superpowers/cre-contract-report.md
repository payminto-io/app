# Ticket 22: GatewayAttestations consumer contract

Branch `cre-contract`, worktree `gateway-wt-cre-contract`, 2026-10-07.
Status: done. Nothing was deployed; no key, RPC or funds were used.

## Commits

| Hash | Content |
| --- | --- |
| `8097042` | forge-std v1.17.0 and openzeppelin-contracts v5.6.1 pinned as shallow submodules (`.gitmodules`, `contracts/foundry.lock`); `forge fmt` on the two legacy contracts |
| `9104f90` | `contracts/src/cre/GatewayAttestations.sol`, `contracts/src/cre/IReceiver.sol`, `contracts/test/cre/*` (unit, fuzz, invariant, gas), `contracts/snapshots/GatewayAttestations.json` |
| `53357af` | `contracts/script/DeployGatewayAttestations.s.sol`; ABI at `backend/internal/cre/abi/GatewayAttestations.json`, `frontend/lib/cre/abi.ts`, `cre/contracts/evm/src/GatewayAttestations.abi` |
| `f9d6e27` | `docs/cre/SPEC.md` sections 3, 5, 6 and `contracts/CLAUDE.md` aligned with what shipped |

The ticket file is set to `Status: done` in the commit after these.

## What the contract does

`GatewayAttestations is IReceiver, IERC165, Ownable2Step`. No `receive`, no `fallback`, no token calls, no proxy, no selfdestruct.

Guards on `onReport(bytes metadata, bytes report)`, in order, every one a named custom error and a full revert:

1. `msg.sender == forwarder` (`UnauthorizedForwarder`).
2. `metadata.length == 64`, decoded as `workflowId(32) | workflowName(10) | workflowOwner(20) | reportId(2)` (`InvalidMetadataLength`).
3. `report.length >= 192`; word 0 `== 1` (`UnsupportedReportVersion`); word 1 fits `uint8` and is 1..3 (`MalformedReport` / `UnknownKind`); word 3 fits `uint64`; word 4 (array offset) `== 160`; word 5 (count) `> 0` (`EmptyReport`); `report.length == 192 + count * itemWords * 32` (`MalformedReport`). Only then `abi.decode`.
4. `workflows[kind] == (workflowId, workflowOwner, workflowName)` (`WorkflowNotBound`, `UnexpectedWorkflow`). A deposit workflow's identity cannot deliver a solvency-kind report.
5. `observedAt <= block.timestamp + 5 minutes` (`ObservedAtInFuture`); `observedAt > lastObservedAt[gatewayId][kind]` (`StaleReport`). This is the replay guard: a replayed report has the same `observedAt` and is rejected; so is anything older than the latest accepted.
6. Solvency only: `observedAt > latestSolvency[gatewayId][asset].observedAt` per item (`StaleSolvency`), which also rejects a duplicate asset within one batch. Deposit only: `verdict` in 1..3 (`InvalidVerdict`).

Storage: `latestSolvency[gatewayId][asset]` (`checkpointHash, liabilities, reserves, observedAt, decimals`, four slots) and `lastObservedAt[gatewayId][kind]`. Deposit and conversion records are events only.

Admin, owner only: `setForwarder`, `setWorkflow(kind, id, owner, nameHash)`, `unbindWorkflow(kind)`, plus OpenZeppelin two-step `transferOwnership` / `acceptOwnership`. No thresholds, no pause, no upgrade.

## Report encoding

One batch per report: `abi.encode(uint8 version, uint8 kind, bytes32 gatewayId, uint64 observedAt, Item[] items)`, `version = 1`, items a static-tuple array (canonical offset 160).

| kind | Item tuple |
| --- | --- |
| 1 solvency | `(bytes32 checkpointHash, bytes32 asset, uint256 liabilities, uint256 reserves, uint8 decimals)` |
| 2 deposit finality | `(bytes32 depositId, bytes32 chainId, bytes32 txRef, bytes32 token, uint256 amount, bytes32 destination, uint64 slotOrBlock, uint8 verdict)` with verdict 1 confirmed, 2 not found, 3 mismatch |
| 3 conversion reference | `(bytes32 conversionId, bytes32 pair, int256 referenceRate, uint8 referenceDecimals, int256 deviationBps, address feed, uint80 roundId)` |

Events the verifier and `/verify/[id]` read:

- `ReportAccepted(bytes32 indexed gatewayId, uint8 indexed kind, bytes32 indexed workflowId, address workflowOwner, bytes10 workflowName, bytes2 reportId, uint64 observedAt, uint256 itemCount, bytes32 reportHash)` once per report, `reportHash = keccak256(report)`.
- `SolvencyAttested(bytes32 indexed gatewayId, bytes32 indexed asset, bytes32 checkpointHash, uint256 liabilities, uint256 reserves, uint8 decimals, uint64 observedAt)` per item.
- `DepositAttested(bytes32 indexed gatewayId, bytes32 indexed depositId, uint8 indexed verdict, bytes32 chainId, bytes32 txRef, bytes32 token, uint256 amount, bytes32 destination, uint64 slotOrBlock, uint64 observedAt)` per item.
- `ConversionReferenceAttested(bytes32 indexed gatewayId, bytes32 indexed conversionId, bytes32 indexed pair, int256 referenceRate, uint8 referenceDecimals, int256 deviationBps, address feed, uint80 roundId, uint64 observedAt)` per item.
- Admin: `ForwarderSet`, `WorkflowBound`, `WorkflowUnbound`, `OwnershipTransferStarted`, `OwnershipTransferred`.

## Checks run

- `forge build`: clean (forge-lint warnings only on the legacy `SmartSweep` modifier and two documented casts).
- `forge test -vvv`: 6 suites, 67 passed, 0 failed. CRE: 42 unit/fuzz (256 runs each), 2 invariants (256 runs x 500 calls, 128,000 calls each), 5 gas.
- `forge fmt --check`: clean project-wide.
- `forge script script/DeployGatewayAttestations.s.sol --sender 0x...bEEF` with env vars: dry run succeeded, bound kind 1, printed addresses. No `--broadcast`, no RPC.

Gas (`contracts/snapshots/GatewayAttestations.json`):

| Case | Gas |
| --- | --- |
| deploy | 1,624,235 |
| `onReport` solvency, 20 assets, first write (all slots cold and zero) | 1,886,423 |
| `onReport` solvency, 20 assets, same-transaction warm rewrite | 100,523 |
| `onReport` deposit finality, 12 items | 96,686 |
| `onReport` conversion reference, 10 items | 83,042 |

The "warm" figure is measured inside one forge test transaction, so the access list is already warm; a real hourly re-write of 20 assets with changed values is closer to 300k to 400k (two changed slots at 5,000 and two unchanged at 2,200 per asset, plus events).

## Concerns

1. **First-write gas for solvency.** 1.9M for 20 new assets is one-off per asset set but must fit the `gasLimit` the solvency workflow passes to `writeReport`. The module branch should set that limit from the batch size, or split a first run into smaller batches. Storing `liabilities`/`reserves` as `uint128` would cut ~440k but would silently cap the spec's `uint256`; I did not do that.
2. **Workflow name hash derivation.** The contract compares `bytes10 workflowName` from the Keystone metadata byte-for-byte. The deploy script derives it as `bytes10(sha256(name))`, which is how Keystone's `HashTruncateName` works as far as the reference material shows, but no document in the skill states it. Before binding on Base Sepolia, read the name bytes from a real `MockKeystoneForwarder` delivery (or the workflow registry) and pass them through `setWorkflow` directly if they differ.
3. **Metadata layout assumption.** `workflowId(32) | workflowName(10) | workflowOwner(20) | reportId(2)` follows the `ReceiverTemplate` assembly (`mload(add(metadata, 74))` for the owner). RESEARCH.md agrees. A simulation run against the mock forwarder is the proof; the invariant suite cannot supply it.
4. **Departures from the ticket text.** Leading `uint8 version` field; per-kind `setWorkflow` instead of `ReceiverTemplate.setExpectedAuthor`; `Ownable2Step`. All written into SPEC in `f9d6e27`. Ticket 23 (workflows) and the Go verifier must encode and decode exactly as SPEC section 5 now states; `contracts/test/cre/ReportEncoder.sol` is the executable reference.
5. **`cre/` path.** The ticket asked for `contracts/evm/src/GatewayAttestations.abi`; SPEC section 3 puts it under `cre/contracts/evm/src/` (the CRE project root that `cre generate-bindings` reads). I used the SPEC path.
6. **Submodules in a worktree.** The submodule metadata lives in the shared `.git/modules`; `git submodule update --init --depth 1` is needed after checkout on any other worktree or clone. The full OpenZeppelin clone kept dropping mid-transfer, so the submodule is shallow and pinned to `v5.6.1`.
7. **Legacy contracts formatted.** `forge fmt` touched `AddressFactory.sol` and `SmartSweep.sol` (whitespace only) so `forge fmt --check` passes for the project.
