// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {DeployGatewayAttestations} from "../../script/DeployGatewayAttestations.s.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";

/// One sequential test: vm.setEnv is process-wide, so parallel test functions would race on the same keys.
contract DeployScriptTest is Test {
    address internal forwarder = makeAddr("forwarder");
    address internal owner = makeAddr("multisig");
    address internal workflowOwner = makeAddr("workflowOwner");

    function _goodEnv() internal {
        vm.setEnv("CRE_FORWARDER_ADDRESS", vm.toString(forwarder));
        vm.setEnv("CRE_CONSUMER_OWNER", vm.toString(owner));
        vm.setEnv("CRE_WORKFLOW_OWNER", vm.toString(workflowOwner));
        vm.setEnv("CRE_WORKFLOW_ID_SOLVENCY", vm.toString(keccak256("s")));
        vm.setEnv("CRE_WORKFLOW_ID_DEPOSIT_FINALITY", vm.toString(keccak256("d")));
        vm.setEnv("CRE_WORKFLOW_ID_CONVERSION_REFERENCE", vm.toString(keccak256("c")));
        vm.setEnv("CRE_WORKFLOW_NAME_SOLVENCY", "solvency");
        vm.setEnv("CRE_WORKFLOW_NAME_DEPOSIT_FINALITY", "deposit-finality");
        vm.setEnv("CRE_WORKFLOW_NAME_CONVERSION_REFERENCE", "conversion-reference");
    }

    function test_deployScript() public {
        vm.etch(forwarder, hex"6001"); // any code: the script only checks that the forwarder is a contract
        _goodEnv();

        // happy path: deployer owns until the multisig accepts; bindings use Keystone names
        GatewayAttestations c = new DeployGatewayAttestations().run();
        assertEq(c.forwarder(), forwarder);
        assertEq(c.pendingOwner(), owner, "multisig is pending, must accept");
        assertTrue(c.owner() != owner && c.owner() != address(0), "deployer holds ownership until accepted");
        GatewayAttestations.Workflow memory w = c.getWorkflow(1);
        assertEq(w.id, keccak256("s"));
        assertEq(w.owner, workflowOwner);
        assertEq(w.name, WorkflowName.keystone("solvency"));
        assertEq(w.name, bytes10("58c66935b7"));
        assertEq(c.getWorkflow(2).name, WorkflowName.keystone("deposit-finality"));
        assertEq(c.getWorkflow(3).name, WorkflowName.keystone("conversion-reference"));
        vm.prank(owner);
        c.acceptOwnership();
        assertEq(c.owner(), owner);

        // empty name
        vm.setEnv("CRE_WORKFLOW_NAME_DEPOSIT_FINALITY", "");
        DeployGatewayAttestations s = new DeployGatewayAttestations();
        vm.expectRevert(
            abi.encodeWithSelector(DeployGatewayAttestations.EmptyEnv.selector, "CRE_WORKFLOW_NAME_DEPOSIT_FINALITY")
        );
        s.run();
        _goodEnv();

        // zero workflow id
        vm.setEnv("CRE_WORKFLOW_ID_CONVERSION_REFERENCE", vm.toString(bytes32(0)));
        vm.expectRevert(
            abi.encodeWithSelector(DeployGatewayAttestations.EmptyEnv.selector, "CRE_WORKFLOW_ID_CONVERSION_REFERENCE")
        );
        s.run();
        _goodEnv();

        // forwarder without code
        address empty = makeAddr("nothing-here");
        vm.setEnv("CRE_FORWARDER_ADDRESS", vm.toString(empty));
        vm.expectRevert(abi.encodeWithSelector(DeployGatewayAttestations.ForwarderHasNoCode.selector, empty));
        s.run();
        _goodEnv();

        // owner missing
        vm.setEnv("CRE_CONSUMER_OWNER", "");
        vm.expectRevert();
        s.run();
        _goodEnv();
    }
}
