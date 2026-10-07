# 22 GatewayAttestations consumer contract

Status: done
Owner: Chainlink engineer (Fable)
Blocked by: 19

## Goal
`contracts/src/GatewayAttestations.sol` per `docs/cre/SPEC.md` section 5: an `IReceiver` built on `ReceiverTemplate` (forwarder check, `setExpectedAuthor`, ERC165), decoding the common prefix `(uint8 kind, bytes32 gatewayId, uint64 observedAt)` then the kind-specific array, emitting `SolvencyAttested`, `DepositAttested`, `ConversionReferenceAttested`, storing latest solvency per `(gatewayId, asset)`. Reject unknown kinds, future `observedAt`, and reports older than the stored latest for the same key. Owner is a deployer-controlled address; no owner-controlled business thresholds. Foundry deploy script taking forwarder and expected author; `contracts/evm/src/GatewayAttestations.abi` copied for `cre generate-bindings`.

## Acceptance
Foundry tests: forwarder-only, author-only, decode round trip for each kind, rejection cases, gas within the EVM write quota for a 12-item deposit batch and a 20-asset solvency batch. Deployed on Base Sepolia by the deployer later, never by an agent.

## Done (2026-10-07)

Commits on `cre-contract`:

- `8097042` pin forge-std v1.17.0 and openzeppelin-contracts v5.6.1 as submodules (nothing in `contracts/` built before), format legacy contracts
- `9104f90` `contracts/src/cre/GatewayAttestations.sol`, `IReceiver.sol`, unit + fuzz + invariant + gas tests
- `53357af` `contracts/script/DeployGatewayAttestations.s.sol`, ABI exported to `backend/internal/cre/abi/GatewayAttestations.json`, `frontend/lib/cre/abi.ts`, `cre/contracts/evm/src/GatewayAttestations.abi`
- `f9d6e27` SPEC section 5 and 6 and `contracts/CLAUDE.md` aligned with the shipped encoding

Departures from the ticket text, all recorded in SPEC:

- The report carries a leading `uint8 version` (= 1) before `kind`, so the prefix is `(version, kind, gatewayId, observedAt)` and the items follow as one array.
- `ReceiverTemplate` was not copied: it binds one expected author for the whole contract, and the requirement is one `(workflowId, workflowOwner, workflowName)` binding per kind. `setWorkflow(kind, id, owner, name)` replaces `setExpectedAuthor`. Forwarder check, ERC165 and `onReport` follow the template.
- Ownership is `Ownable2Step`, not `Ownable`.

Checks: `forge build` clean, `forge test -vvv` 67 passed (49 CRE), `forge fmt --check` clean. Gas (snapshots/GatewayAttestations.json): solvency 20 assets first write 1,886,423; deposit 12 items 96,686; conversion 10 items 83,042; deploy 1,624,235.
Report: `.superpowers/cre-contract-report.md`.
