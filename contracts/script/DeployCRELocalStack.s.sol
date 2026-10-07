// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {MockKeystoneForwarder} from "../src/cre/mocks/MockKeystoneForwarder.sol";
import {MockERC20} from "../src/cre/mocks/MockERC20.sol";

/// Local Anvil only: deploys the MockKeystoneForwarder and a reserve token minted to a custody address, so the
/// solvency workflow's EVM reserve read and `cre workflow simulate --broadcast` have something to talk to.
/// Refuses any node that is not Anvil (the `anvil_nodeInfo` RPC exists nowhere else). Runbook: cre/README.md.
///
///   CRE_LOCAL_CUSTODY_ADDRESS   address the reserve token is minted to (the workflow config's custodyAddress)
///   CRE_LOCAL_RESERVE_MINOR     amount to mint in minor units
///   CRE_LOCAL_FORWARDER_AT      optional: also install the forwarder code at this address with anvil_setCode,
///                               for a simulator that insists on a chain's published mock-forwarder address
contract DeployCRELocalStack is Script {
    error NotAnvil();

    function run() external returns (MockKeystoneForwarder forwarder, MockERC20 token) {
        try vm.rpc("anvil_nodeInfo", "[]") returns (bytes memory) {}
        catch {
            revert NotAnvil();
        }
        address custody = vm.envAddress("CRE_LOCAL_CUSTODY_ADDRESS");
        uint256 reserve = vm.envUint("CRE_LOCAL_RESERVE_MINOR");

        vm.startBroadcast();
        forwarder = new MockKeystoneForwarder();
        token = new MockERC20("Mock USD Coin", "USDC", 6);
        token.mint(custody, reserve);
        vm.stopBroadcast();

        address pinned = vm.envOr("CRE_LOCAL_FORWARDER_AT", address(0));
        if (pinned != address(0)) {
            vm.rpc(
                "anvil_setCode",
                string.concat('["', vm.toString(pinned), '","', vm.toString(address(forwarder).code), '"]')
            );
            console.log("forwarder code also installed at:", pinned);
        }
        console.log("MockKeystoneForwarder:", address(forwarder));
        console.log("MockERC20 (USDC, 6):", address(token));
        console.log("custody:", custody);
        console.log("reserve minted (minor units):", reserve);
    }
}
