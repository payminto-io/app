// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {GatewayAttestations} from "../src/cre/GatewayAttestations.sol";

/// Deploys GatewayAttestations and binds the workflows that are configured. Dry run unless --broadcast is given.
///
///   CRE_FORWARDER_ADDRESS   required; KeystoneForwarder (live) or MockKeystoneForwarder (simulation)
///   CRE_CONSUMER_OWNER      optional; defaults to the broadcasting sender (use the deployer multisig)
///   CRE_WORKFLOW_OWNER      required when any workflow id is set
///   CRE_WORKFLOW_ID_SOLVENCY | _DEPOSIT_FINALITY | _CONVERSION_REFERENCE   optional bytes32 workflow ids
///   CRE_WORKFLOW_NAME_SOLVENCY | _DEPOSIT_FINALITY | _CONVERSION_REFERENCE optional names; the bound value is
///                           bytes10(sha256(name)), how Keystone truncates workflow names into the metadata
///
///   forge script script/DeployGatewayAttestations.s.sol --rpc-url $RPC --sender $DEPLOYER          # dry run
///   forge script script/DeployGatewayAttestations.s.sol --rpc-url $RPC --sender $DEPLOYER --broadcast --verify
contract DeployGatewayAttestations is Script {
    function run() external returns (GatewayAttestations deployed) {
        address forwarder = vm.envAddress("CRE_FORWARDER_ADDRESS");
        address owner = vm.envOr("CRE_CONSUMER_OWNER", msg.sender);

        vm.startBroadcast();
        deployed = new GatewayAttestations(forwarder, owner);
        _bind(deployed, 1, "CRE_WORKFLOW_ID_SOLVENCY", "CRE_WORKFLOW_NAME_SOLVENCY");
        _bind(deployed, 2, "CRE_WORKFLOW_ID_DEPOSIT_FINALITY", "CRE_WORKFLOW_NAME_DEPOSIT_FINALITY");
        _bind(deployed, 3, "CRE_WORKFLOW_ID_CONVERSION_REFERENCE", "CRE_WORKFLOW_NAME_CONVERSION_REFERENCE");
        vm.stopBroadcast();

        console.log("GatewayAttestations:", address(deployed));
        console.log("forwarder:", forwarder);
        console.log("owner:", owner);
    }

    function _bind(GatewayAttestations c, uint8 kind, string memory idKey, string memory nameKey) internal {
        bytes32 id = vm.envOr(idKey, bytes32(0));
        if (id == bytes32(0)) return;
        address workflowOwner = vm.envAddress("CRE_WORKFLOW_OWNER");
        bytes10 name = bytes10(sha256(bytes(vm.envOr(nameKey, string("")))));
        c.setWorkflow(kind, id, workflowOwner, name);
        console.log("bound kind", kind);
        console.logBytes32(id);
        console.logBytes10(name);
    }
}
