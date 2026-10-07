// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {ReportEncoder} from "./ReportEncoder.sol";

/// Prints the bytes the workflow's encoder must reproduce for cre/workflows/solvency/fixtures/solvency-report.json.
/// Regenerate the fixture with: forge test --match-contract SolvencyFixtureTest -vv
contract SolvencyFixtureTest is Test {
    function test_printSolvencyFixture() public pure {
        GatewayAttestations.SolvencyItem[] memory items = new GatewayAttestations.SolvencyItem[](2);
        items[0] = GatewayAttestations.SolvencyItem({
            checkpointHash: keccak256("checkpoint-1"),
            asset: bytes32("USDC"),
            liabilities: 1_250_000_000,
            reserves: 1_300_000_000,
            decimals: 6
        });
        items[1] = GatewayAttestations.SolvencyItem({
            checkpointHash: keccak256("checkpoint-1"),
            asset: bytes32("ETH"),
            liabilities: 2_000_000_000_000_000_000,
            reserves: 2_500_000_000_000_000_000,
            decimals: 18
        });
        bytes memory report = ReportEncoder.solvency(keccak256("https://pay.example.com"), 1_800_000_000, items);
        console.log("gatewayId");
        console.logBytes32(keccak256("https://pay.example.com"));
        console.log("checkpointHash");
        console.logBytes32(keccak256("checkpoint-1"));
        console.log("report");
        console.logBytes(report);
        console.log("reportHash");
        console.logBytes32(keccak256(report));
    }
}
