// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test, Vm} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";
import {ReportEncoder} from "./ReportEncoder.sol";
import {KeystoneForwarder} from "./vendor/keystone/KeystoneForwarder.sol";
import {IRouter} from "./vendor/keystone/IRouter.sol";

/// End to end through Chainlink's real KeystoneForwarder (vendored, pinned): the forwarder verifies f+1 oracle
/// signatures over the raw report, slices the 64-byte metadata itself and calls onReport. Nothing here uses our
/// own metadata encoder, so this proves the layout and the Keystone name derivation against the delivery contract.
contract GatewayAttestationsForwarderTest is Test {
    KeystoneForwarder internal fwd;
    GatewayAttestations internal c;

    address internal owner = makeAddr("owner");
    address internal workflowOwner = makeAddr("workflowOwner");
    uint32 internal constant DON_ID = 7;
    uint32 internal constant CONFIG_VERSION = 1;
    uint8 internal constant F = 1;
    uint256[4] internal oracleKeys = [uint256(0xA11CE), uint256(0xB0B), uint256(0xCA7), uint256(0xD06)];

    bytes32 internal constant GATEWAY = keccak256("https://pay.example.com");
    bytes32 internal constant WF_SOLVENCY = keccak256("wf-solvency");
    bytes10 internal nameSolvency;
    uint64 internal constant T0 = 1_800_000_000;

    bytes32 internal constant REPORT_ACCEPTED_TOPIC =
        keccak256("ReportAccepted(bytes32,uint8,bytes32,address,bytes10,bytes2,uint64,uint256,bytes32)");

    function setUp() public {
        vm.warp(T0);
        nameSolvency = WorkflowName.keystone("solvency");
        fwd = new KeystoneForwarder();
        fwd.addForwarder(address(fwd)); // report() routes through this.route(), which requires a registered forwarder
        address[] memory signers = new address[](4);
        for (uint256 i = 0; i < 4; ++i) {
            signers[i] = vm.addr(oracleKeys[i]);
        }
        fwd.setConfig(DON_ID, CONFIG_VERSION, F, signers);

        c = new GatewayAttestations(address(fwd), owner);
        vm.prank(owner);
        c.setWorkflow(1, WF_SOLVENCY, workflowOwner, nameSolvency);
    }

    /// Raw report exactly as KeystoneForwarder._getMetadata documents it; the forwarder, not us, slices [45:109].
    function rawReport(bytes32 execId, bytes10 name, bytes2 reportId, bytes memory report)
        internal
        view
        returns (bytes memory)
    {
        return abi.encodePacked(
            uint8(1),
            execId,
            uint32(block.timestamp),
            DON_ID,
            CONFIG_VERSION,
            WF_SOLVENCY,
            name,
            workflowOwner,
            reportId,
            report
        );
    }

    function sign(bytes memory raw, bytes memory reportContext) internal view returns (bytes[] memory sigs) {
        bytes32 completeHash = keccak256(abi.encodePacked(keccak256(raw), reportContext));
        sigs = new bytes[](F + 1);
        for (uint256 i = 0; i < F + 1; ++i) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(oracleKeys[i], completeHash);
            sigs[i] = abi.encodePacked(r, s, bytes1(v - 27));
        }
    }

    function test_realForwarder_deliversAndContractAccepts() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(3, keccak256("ckpt")));
        bytes32 execId = keccak256("execution-1");
        bytes memory raw = rawReport(execId, nameSolvency, 0x0001, report);
        bytes memory ctx = new bytes(96);

        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.ReportAccepted(
            GATEWAY, 1, WF_SOLVENCY, workflowOwner, nameSolvency, 0x0001, T0, 3, keccak256(report)
        );
        fwd.report(address(c), raw, ctx, sign(raw, ctx));

        assertEq(c.latestObservedAt(GATEWAY, 1), T0);
        assertTrue(c.reportSeen(keccak256(report)));
        bytes32 asset0 = keccak256(abi.encodePacked("asset", uint256(0)));
        assertEq(c.getLatestSolvency(GATEWAY, asset0).liabilities, 1_000_000);

        // The forwarder refuses to re-route the same transmission, independently of our seen-set.
        vm.expectRevert(
            abi.encodeWithSelector(IRouter.AlreadyAttempted.selector, fwd.getTransmissionId(address(c), execId, 0x0001))
        );
        fwd.report(address(c), raw, ctx, sign(raw, ctx));

        // A new execution replaying the same report bytes reaches the contract and is rejected there.
        bytes memory raw2 = rawReport(keccak256("execution-2"), nameSolvency, 0x0001, report);
        vm.recordLogs();
        fwd.report(address(c), raw2, ctx, sign(raw2, ctx));
        assertEq(_countAccepted(vm.getRecordedLogs()), 0, "replay through a new execution must not be accepted");
    }

    /// The bug the audit found: binding bytes10(sha256(name)) never matches what Keystone sends.
    function test_realForwarder_rejectsNameBoundWithWrongDerivation() public {
        bytes10 wrong = bytes10(sha256("solvency"));
        assertTrue(wrong != nameSolvency);
        vm.prank(owner);
        c.setWorkflow(1, WF_SOLVENCY, workflowOwner, wrong);

        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        bytes memory raw = rawReport(keccak256("execution-3"), nameSolvency, 0x0001, report);
        bytes memory ctx = new bytes(96);
        vm.recordLogs();
        fwd.report(address(c), raw, ctx, sign(raw, ctx)); // forwarder swallows the receiver revert
        assertEq(_countAccepted(vm.getRecordedLogs()), 0);
        assertEq(c.latestObservedAt(GATEWAY, 1), 0);
        assertFalse(c.reportSeen(keccak256(report)));
    }

    function test_realForwarder_rejectsUnbindedOwnerAndId() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        bytes memory ctx = new bytes(96);
        // wrong workflow id in the raw metadata
        bytes memory raw = abi.encodePacked(
            uint8(1),
            keccak256("execution-4"),
            uint32(block.timestamp),
            DON_ID,
            CONFIG_VERSION,
            keccak256("other-workflow"),
            nameSolvency,
            workflowOwner,
            bytes2(0x0001),
            report
        );
        vm.recordLogs();
        fwd.report(address(c), raw, ctx, sign(raw, ctx));
        assertEq(_countAccepted(vm.getRecordedLogs()), 0);
        assertFalse(c.reportSeen(keccak256(report)));
    }

    function test_realForwarder_rejectsUnconfiguredSigner() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        bytes memory raw = rawReport(keccak256("execution-5"), nameSolvency, 0x0001, report);
        bytes memory ctx = new bytes(96);
        bytes32 completeHash = keccak256(abi.encodePacked(keccak256(raw), ctx));
        bytes[] memory sigs = new bytes[](2);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(oracleKeys[0], completeHash);
        sigs[0] = abi.encodePacked(r, s, bytes1(v - 27));
        (v, r, s) = vm.sign(uint256(0xBAD), completeHash);
        sigs[1] = abi.encodePacked(r, s, bytes1(v - 27));
        vm.expectRevert(abi.encodeWithSelector(KeystoneForwarder.InvalidSigner.selector, vm.addr(uint256(0xBAD))));
        fwd.report(address(c), raw, ctx, sigs);
    }

    function _countAccepted(Vm.Log[] memory logs) internal view returns (uint256 n) {
        for (uint256 i = 0; i < logs.length; ++i) {
            if (logs[i].emitter == address(c) && logs[i].topics[0] == REPORT_ACCEPTED_TOPIC) n++;
        }
    }
}
