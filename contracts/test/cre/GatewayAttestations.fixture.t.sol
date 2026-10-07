// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test, Vm} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";
import {ReportEncoder} from "./ReportEncoder.sol";
import {KeystoneForwarder} from "./vendor/keystone/KeystoneForwarder.sol";

/// Writes test/cre/fixtures/forwarder_logs.json: the logs and forwarder calldata of two real deliveries
/// (the second supersedes one asset and repeats another, so SolvencyIgnored appears). The Go reader test
/// in backend/internal/cre/chainlink consumes it, so the reader is exercised against the real contract.
contract GatewayAttestationsFixtureTest is Test {
    KeystoneForwarder internal fwd;
    GatewayAttestations internal c;
    address internal owner = makeAddr("owner");
    address internal workflowOwner = 0x3333333333333333333333333333333333333333;
    uint32 internal constant DON_ID = 7;
    uint32 internal constant CONFIG_VERSION = 1;
    uint8 internal constant F = 1;
    uint256[4] internal oracleKeys = [uint256(0xA11CE), uint256(0xB0B), uint256(0xCA7), uint256(0xD06)];
    bytes32 internal constant GATEWAY = keccak256("https://pay.example.test");
    bytes32 internal constant WF_SOLVENCY = keccak256("wf-solvency");
    uint64 internal constant T0 = 1_800_000_000;

    function setUp() public {
        vm.warp(T0 + 10);
        fwd = new KeystoneForwarder();
        fwd.addForwarder(address(fwd));
        address[] memory signers = new address[](4);
        for (uint256 i = 0; i < 4; ++i) {
            signers[i] = vm.addr(oracleKeys[i]);
        }
        fwd.setConfig(DON_ID, CONFIG_VERSION, F, signers);
        c = new GatewayAttestations(address(fwd), owner);
        bytes10 name = WorkflowName.keystone("solvency");
        vm.prank(owner);
        c.setWorkflow(1, WF_SOLVENCY, workflowOwner, name);
    }

    function rawReport(bytes32 execId, bytes memory report) internal view returns (bytes memory) {
        return abi.encodePacked(
            uint8(1), execId, uint32(block.timestamp), DON_ID, CONFIG_VERSION, WF_SOLVENCY,
            WorkflowName.keystone("solvency"), workflowOwner, bytes2(0x0001), report
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

    function items(uint64 observed, bytes32 a, bytes32 b) internal pure returns (GatewayAttestations.SolvencyItem[] memory out) {
        out = new GatewayAttestations.SolvencyItem[](2);
        out[0] = GatewayAttestations.SolvencyItem({checkpointHash: keccak256("cp-fixture"), asset: a, liabilities: 1_000_000, reserves: 1_500_000, decimals: 6});
        out[1] = GatewayAttestations.SolvencyItem({checkpointHash: keccak256("cp-fixture"), asset: b, liabilities: 7, reserves: 9, decimals: 9});
        observed;
    }

    function logEntry(string memory label, uint256 i, Vm.Log memory log, uint256 blockNumber)
        internal
        returns (string memory)
    {
        string memory key = string.concat(label, "-log-", vm.toString(i));
        vm.serializeAddress(key, "address", log.emitter);
        vm.serializeBytes32(key, "tx_hash", keccak256(bytes(label)));
        vm.serializeUint(key, "block_number", blockNumber);
        vm.serializeUint(key, "index", i);
        vm.serializeBytes32(key, "topics", log.topics);
        return vm.serializeBytes(key, "data", log.data);
    }

    function logsJson(string memory label, Vm.Log[] memory logs, uint256 blockNumber) internal returns (string memory out) {
        out = "[";
        for (uint256 i = 0; i < logs.length; ++i) {
            out = string.concat(out, i == 0 ? "" : ",", logEntry(label, i, logs[i], blockNumber));
        }
        out = string.concat(out, "]");
    }

    function send(bytes memory raw) internal returns (bytes memory calldataBytes, Vm.Log[] memory logs) {
        bytes memory ctx = new bytes(96);
        bytes[] memory sigs = sign(raw, ctx);
        calldataBytes = abi.encodeCall(fwd.report, (address(c), raw, ctx, sigs));
        vm.recordLogs();
        fwd.report(address(c), raw, ctx, sigs);
        logs = vm.getRecordedLogs();
    }

    function deliver(string memory label, bytes32 execId, bytes memory report, uint256 blockNumber)
        internal
        returns (string memory)
    {
        (bytes memory calldataBytes, Vm.Log[] memory logs) = send(rawReport(execId, report));
        vm.serializeString(label, "logs", logsJson(label, logs, blockNumber));
        vm.serializeBytes(label, "calldata", calldataBytes);
        vm.serializeBytes32(label, "report_hash", keccak256(report));
        return vm.serializeBytes32(label, "tx_hash", keccak256(bytes(label)));
    }

    function test_writeForwarderFixture() public {
        bytes32 usdc = bytes32("USDC.SOLANA");
        bytes32 sol = bytes32("SOL");
        bytes memory first = ReportEncoder.solvency(GATEWAY, T0, items(T0, usdc, sol));
        string memory d1 = deliver("delivery-1", keccak256("execution-1"), first, 1001);
        // Older observedAt for the same assets: both items are SolvencyIgnored, the report is still accepted.
        bytes memory second = ReportEncoder.solvency(GATEWAY, T0 - 1, items(T0 - 1, usdc, sol));
        string memory d2 = deliver("delivery-2", keccak256("execution-2"), second, 1002);
        string memory root = "fixture";
        vm.serializeAddress(root, "consumer", address(c));
        vm.serializeAddress(root, "forwarder", address(fwd));
        vm.serializeAddress(root, "workflow_owner", workflowOwner);
        vm.serializeBytes32(root, "workflow_id", WF_SOLVENCY);
        vm.serializeBytes32(root, "gateway_id", GATEWAY);
        vm.serializeString(root, "delivery_1", d1);
        string memory json = vm.serializeString(root, "delivery_2", d2);
        vm.writeJson(json, "./test/cre/fixtures/forwarder_logs.json");
    }
}
