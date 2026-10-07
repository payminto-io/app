// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";
import {MockKeystoneForwarder} from "../../src/cre/mocks/MockKeystoneForwarder.sol";
import {ReportEncoder} from "./ReportEncoder.sol";

/// The mock must slice a raw Keystone report exactly as the real forwarder does, so the consumer sees the same
/// metadata and report bytes whether a report arrives through KeystoneForwarder or through the simulation path.
contract MockKeystoneForwarderTest is Test {
    MockKeystoneForwarder internal fwd;
    GatewayAttestations internal c;

    address internal owner = makeAddr("owner");
    address internal workflowOwner = makeAddr("workflowOwner");
    bytes32 internal constant GATEWAY = keccak256("https://pay.example.com");
    bytes32 internal constant WF = keccak256("wf-solvency");
    uint64 internal constant T0 = 1_800_000_000;
    bytes10 internal name;

    function setUp() public {
        vm.warp(T0);
        name = WorkflowName.keystone("solvency-staging");
        fwd = new MockKeystoneForwarder();
        c = new GatewayAttestations(address(fwd), owner);
        vm.prank(owner);
        c.setWorkflow(1, WF, workflowOwner, name);
    }

    function rawReport(bytes32 execId, bytes2 reportId, bytes memory report) internal view returns (bytes memory) {
        return
            abi.encodePacked(
                uint8(1), execId, uint32(T0), uint32(7), uint32(1), WF, name, workflowOwner, reportId, report
            );
    }

    function test_deliversThroughTheSameSlicing() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(2, keccak256("ckpt")));
        bytes32 execId = keccak256("execution-1");
        vm.expectEmit(true, true, true, true, address(fwd));
        emit MockKeystoneForwarder.ReportProcessed(address(c), execId, bytes2(0x0001), true);
        fwd.report(address(c), rawReport(execId, bytes2(0x0001), report), "", new bytes[](0));

        assertTrue(c.reportSeen(keccak256(report)));
        MockKeystoneForwarder.TransmissionInfo memory info = fwd.getTransmissionInfo(address(c), execId, bytes2(0x0001));
        assertEq(uint256(info.state), uint256(MockKeystoneForwarder.TransmissionState.SUCCEEDED));
        assertEq(fwd.getTransmitter(address(c), execId, bytes2(0x0001)), address(this));
    }

    function test_receiverRevertIsRecordedNotPropagated() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(1, keccak256("ckpt")));
        bytes32 execId = keccak256("execution-2");
        bytes memory raw = abi.encodePacked(
            uint8(1),
            execId,
            uint32(T0),
            uint32(7),
            uint32(1),
            keccak256("other-wf"),
            name,
            workflowOwner,
            bytes2(0x0002),
            report
        );
        fwd.report(address(c), raw, "", new bytes[](0));
        assertFalse(c.reportSeen(keccak256(report)));
        MockKeystoneForwarder.TransmissionInfo memory info = fwd.getTransmissionInfo(address(c), execId, bytes2(0x0002));
        assertEq(uint256(info.state), uint256(MockKeystoneForwarder.TransmissionState.FAILED));
    }

    function test_secondDeliveryOfSameTransmissionReverts() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(1, keccak256("ckpt")));
        bytes32 execId = keccak256("execution-3");
        bytes memory raw = rawReport(execId, bytes2(0x0003), report);
        fwd.report(address(c), raw, "", new bytes[](0));
        vm.expectRevert(
            abi.encodeWithSelector(
                MockKeystoneForwarder.AlreadyAttempted.selector,
                fwd.getTransmissionId(address(c), execId, bytes2(0x0003))
            )
        );
        fwd.report(address(c), raw, "", new bytes[](0));
    }

    function test_nonReceiverIsInvalid() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(1, keccak256("ckpt")));
        bytes32 execId = keccak256("execution-4");
        address eoa = makeAddr("eoa");
        fwd.report(eoa, rawReport(execId, bytes2(0x0004), report), "", new bytes[](0));
        MockKeystoneForwarder.TransmissionInfo memory info = fwd.getTransmissionInfo(eoa, execId, bytes2(0x0004));
        assertEq(uint256(info.state), uint256(MockKeystoneForwarder.TransmissionState.INVALID_RECEIVER));
    }

    function test_shortReportReverts() public {
        vm.expectRevert(MockKeystoneForwarder.InvalidReport.selector);
        fwd.report(address(c), new bytes(108), "", new bytes[](0));
    }
}
