// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {GatewayAttestations} from "../src/cre/GatewayAttestations.sol";
import {WorkflowName} from "../src/cre/WorkflowName.sol";

/// Deploys GatewayAttestations, binds all three workflows as the broadcasting deployer, then starts the two-step
/// transfer to CRE_CONSUMER_OWNER; that address must call acceptOwnership. Dry run unless --broadcast is given.
/// Every variable is required; the script reverts on a missing or empty one rather than binding a dead value.
///
///   CRE_FORWARDER_ADDRESS   KeystoneForwarder (live) or MockKeystoneForwarder (simulation); must have code
///   CRE_CONSUMER_OWNER      the multisig that will own the contract (never defaulted to the sender)
///   CRE_WORKFLOW_OWNER      EVM address that owns the deployed workflows
///   CRE_WORKFLOW_ID_SOLVENCY | _DEPOSIT_FINALITY | _CONVERSION_REFERENCE     bytes32 workflow ids
///   CRE_WORKFLOW_NAME_SOLVENCY | _DEPOSIT_FINALITY | _CONVERSION_REFERENCE   names as in workflow.yaml; bound as
///                           WorkflowName.keystone(name), the ten-hex-character truncation Keystone writes
///
///   forge script script/DeployGatewayAttestations.s.sol --rpc-url $RPC --sender $DEPLOYER          # dry run
///   forge script script/DeployGatewayAttestations.s.sol --rpc-url $RPC --sender $DEPLOYER --broadcast --verify
contract DeployGatewayAttestations is Script {
    error ForwarderHasNoCode(address forwarder);
    error EmptyEnv(string key);

    function run() external returns (GatewayAttestations deployed) {
        address forwarder = vm.envAddress("CRE_FORWARDER_ADDRESS");
        if (forwarder.code.length == 0) revert ForwarderHasNoCode(forwarder);
        address owner = vm.envAddress("CRE_CONSUMER_OWNER");
        address workflowOwner = vm.envAddress("CRE_WORKFLOW_OWNER");
        (bytes32 idS, bytes10 nameS) = _binding("CRE_WORKFLOW_ID_SOLVENCY", "CRE_WORKFLOW_NAME_SOLVENCY");
        (bytes32 idD, bytes10 nameD) =
            _binding("CRE_WORKFLOW_ID_DEPOSIT_FINALITY", "CRE_WORKFLOW_NAME_DEPOSIT_FINALITY");
        (bytes32 idC, bytes10 nameC) =
            _binding("CRE_WORKFLOW_ID_CONVERSION_REFERENCE", "CRE_WORKFLOW_NAME_CONVERSION_REFERENCE");

        vm.startBroadcast();
        (, address deployer,) = vm.readCallers();
        deployed = new GatewayAttestations(forwarder, deployer);
        deployed.setWorkflow(1, idS, workflowOwner, nameS);
        deployed.setWorkflow(2, idD, workflowOwner, nameD);
        deployed.setWorkflow(3, idC, workflowOwner, nameC);
        deployed.transferOwnership(owner);
        vm.stopBroadcast();

        console.log("GatewayAttestations:", address(deployed));
        console.log("forwarder:", forwarder);
        console.log("deployer (owner until accepted):", deployer);
        console.log("pending owner, must call acceptOwnership():", owner);
        console.log("workflow owner:", workflowOwner);
        _log("solvency", idS, nameS);
        _log("deposit-finality", idD, nameD);
        _log("conversion-reference", idC, nameC);
    }

    function _binding(string memory idKey, string memory nameKey) internal view returns (bytes32 id, bytes10 name) {
        id = vm.envBytes32(idKey);
        if (id == bytes32(0)) revert EmptyEnv(idKey);
        string memory raw = vm.envString(nameKey);
        if (bytes(raw).length == 0) revert EmptyEnv(nameKey);
        name = WorkflowName.keystone(raw);
    }

    function _log(string memory label, bytes32 id, bytes10 name) internal pure {
        console.log(label);
        console.logBytes32(id);
        console.logBytes10(name);
    }
}
