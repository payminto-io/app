// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {ReportEncoder} from "./ReportEncoder.sol";

/// Gas for the batch sizes SPEC section 5 names: 20-asset solvency, 12-item deposit, 10-item conversion.
/// Figures are written to snapshots/GatewayAttestations.json by vm.snapshotGasLastCall.
contract GatewayAttestationsGasTest is Test {
    GatewayAttestations internal c;
    address internal forwarder = makeAddr("forwarder");
    address internal owner = makeAddr("owner");
    address internal workflowOwner = makeAddr("workflowOwner");
    bytes32 internal constant GATEWAY = keccak256("https://pay.example.com");
    bytes10 internal NAME;
    uint64 internal constant T0 = 1_800_000_000;

    // Keep every batch under a conservative EVM write budget; the workflow config sets the real gasLimit.
    uint256 internal constant BUDGET = 2_500_000;

    function setUp() public {
        NAME = bytes10(sha256("name"));
        vm.warp(T0);
        c = new GatewayAttestations(forwarder, owner);
        vm.startPrank(owner);
        c.setWorkflow(1, keccak256("s"), workflowOwner, NAME);
        c.setWorkflow(2, keccak256("d"), workflowOwner, NAME);
        c.setWorkflow(3, keccak256("c"), workflowOwner, NAME);
        vm.stopPrank();
    }

    function _meta(bytes32 id) internal view returns (bytes memory) {
        return ReportEncoder.metadata(id, NAME, workflowOwner, 0x0001);
    }

    function test_gas_solvency20_cold() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(20, keccak256("k")));
        vm.prank(forwarder);
        c.onReport(_meta(keccak256("s")), report);
        uint256 used = vm.snapshotGasLastCall("GatewayAttestations", "onReport_solvency_20_cold");
        assertLt(used, BUDGET);
    }

    function test_gas_solvency20_warm() public {
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, ReportEncoder.solvencyBatch(20, keccak256("k")));
        vm.prank(forwarder);
        c.onReport(_meta(keccak256("s")), report);
        report = ReportEncoder.solvency(GATEWAY, T0 + 3600, ReportEncoder.solvencyBatch(20, keccak256("k2")));
        vm.warp(T0 + 3600);
        vm.prank(forwarder);
        c.onReport(_meta(keccak256("s")), report);
        uint256 used = vm.snapshotGasLastCall("GatewayAttestations", "onReport_solvency_20_warm");
        assertLt(used, BUDGET / 2);
    }

    function test_gas_deposit12() public {
        bytes memory report = ReportEncoder.deposits(GATEWAY, T0, ReportEncoder.depositBatch(12));
        vm.prank(forwarder);
        c.onReport(_meta(keccak256("d")), report);
        uint256 used = vm.snapshotGasLastCall("GatewayAttestations", "onReport_deposit_12");
        assertLt(used, BUDGET / 4);
    }

    function test_gas_conversion10() public {
        bytes memory report = ReportEncoder.conversions(GATEWAY, T0, ReportEncoder.conversionBatch(10));
        vm.prank(forwarder);
        c.onReport(_meta(keccak256("c")), report);
        uint256 used = vm.snapshotGasLastCall("GatewayAttestations", "onReport_conversion_10");
        assertLt(used, BUDGET / 4);
    }

    function test_gas_deploy() public {
        new GatewayAttestations(forwarder, owner);
        vm.snapshotGasLastCall("GatewayAttestations", "deploy");
    }
}
