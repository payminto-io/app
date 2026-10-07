// Vendored unchanged except import paths from smartcontractkit/chainlink-evm@b723176adfe8f2e9eff47730e21a1ac8f64b46d7
// (contracts/cre/src/v1 and contracts/src/v0.8/shared). Test-only: pins the real KeystoneForwarder delivery path.
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ConfirmedOwnerWithProposal} from "./ConfirmedOwnerWithProposal.sol";

/// @title The ConfirmedOwner contract
/// @notice A contract with helpers for basic contract ownership.
contract ConfirmedOwner is ConfirmedOwnerWithProposal {
  constructor(
    address newOwner
  ) ConfirmedOwnerWithProposal(newOwner, address(0)) {}
}
