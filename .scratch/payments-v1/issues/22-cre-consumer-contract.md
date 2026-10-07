# 22 GatewayAttestations consumer contract

Status: ready-for-agent
Owner: Chainlink engineer (Fable)
Blocked by: 19

## Goal
`contracts/src/GatewayAttestations.sol` per `docs/cre/SPEC.md` section 5: an `IReceiver` built on `ReceiverTemplate` (forwarder check, `setExpectedAuthor`, ERC165), decoding the common prefix `(uint8 kind, bytes32 gatewayId, uint64 observedAt)` then the kind-specific array, emitting `SolvencyAttested`, `DepositAttested`, `ConversionReferenceAttested`, storing latest solvency per `(gatewayId, asset)`. Reject unknown kinds, future `observedAt`, and reports older than the stored latest for the same key. Owner is a deployer-controlled address; no owner-controlled business thresholds. Foundry deploy script taking forwarder and expected author; `contracts/evm/src/GatewayAttestations.abi` copied for `cre generate-bindings`.

## Acceptance
Foundry tests: forwarder-only, author-only, decode round trip for each kind, rejection cases, gas within the EVM write quota for a 12-item deposit batch and a 20-asset solvency batch. Deployed on Base Sepolia by the deployer later, never by an agent.
