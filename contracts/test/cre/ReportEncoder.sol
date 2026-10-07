// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";

// Test-side mirror of the workflow's report and Keystone metadata encoding.
library ReportEncoder {
    uint8 internal constant VERSION = 1;
    uint8 internal constant KIND_SOLVENCY = 1;
    uint8 internal constant KIND_DEPOSIT_FINALITY = 2;
    uint8 internal constant KIND_CONVERSION_REFERENCE = 3;

    function metadata(bytes32 workflowId, bytes10 workflowName, address workflowOwner, bytes2 reportId)
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodePacked(workflowId, workflowName, workflowOwner, reportId);
    }

    function solvency(bytes32 gatewayId, uint64 observedAt, GatewayAttestations.SolvencyItem[] memory items)
        internal
        pure
        returns (bytes memory)
    {
        return abi.encode(VERSION, KIND_SOLVENCY, gatewayId, observedAt, items);
    }

    function deposits(bytes32 gatewayId, uint64 observedAt, GatewayAttestations.DepositItem[] memory items)
        internal
        pure
        returns (bytes memory)
    {
        return abi.encode(VERSION, KIND_DEPOSIT_FINALITY, gatewayId, observedAt, items);
    }

    function conversions(bytes32 gatewayId, uint64 observedAt, GatewayAttestations.ConversionItem[] memory items)
        internal
        pure
        returns (bytes memory)
    {
        return abi.encode(VERSION, KIND_CONVERSION_REFERENCE, gatewayId, observedAt, items);
    }

    function solvencyBatch(uint256 n, bytes32 checkpointHash)
        internal
        pure
        returns (GatewayAttestations.SolvencyItem[] memory items)
    {
        items = new GatewayAttestations.SolvencyItem[](n);
        for (uint256 i = 0; i < n; ++i) {
            items[i] = GatewayAttestations.SolvencyItem({
                checkpointHash: checkpointHash,
                asset: keccak256(abi.encodePacked("asset", i)),
                liabilities: 1_000_000 * (i + 1),
                reserves: 1_100_000 * (i + 1),
                decimals: 6
            });
        }
    }

    function depositBatch(uint256 n) internal pure returns (GatewayAttestations.DepositItem[] memory items) {
        items = new GatewayAttestations.DepositItem[](n);
        for (uint256 i = 0; i < n; ++i) {
            items[i] = GatewayAttestations.DepositItem({
                depositId: keccak256(abi.encodePacked("deposit", i)),
                chainId: keccak256("solana-mainnet"),
                txRef: keccak256(abi.encodePacked("sig", i)),
                token: keccak256("USDC"),
                amount: 25_000_000 + i,
                destination: keccak256(abi.encodePacked("dest", i)),
                slotOrBlock: uint64(300_000_000 + i),
                verdict: uint8(1 + (i % 3))
            });
        }
    }

    function conversionBatch(uint256 n) internal pure returns (GatewayAttestations.ConversionItem[] memory items) {
        items = new GatewayAttestations.ConversionItem[](n);
        for (uint256 i = 0; i < n; ++i) {
            items[i] = GatewayAttestations.ConversionItem({
                conversionId: keccak256(abi.encodePacked("conversion", i)),
                pair: keccak256("EUR/USD"),
                referenceRate: int256(108_000_000 + i),
                referenceDecimals: 8,
                deviationBps: int256(i) - 5,
                feed: address(uint160(0xFEED0000 + i)),
                roundId: uint80(18446744073709551616 + i)
            });
        }
    }
}
