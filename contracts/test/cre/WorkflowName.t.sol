// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";

contract WorkflowNameTest is Test {
    // Fixture from the Chainlink consumer-contract docs, not from our own encoder.
    function test_matchesChainlinkDocsExample() public pure {
        assertEq(WorkflowName.keystone("my_workflow"), bytes10(0x62373666336165316465));
    }

    function test_isAsciiHexOfDigestPrefix() public pure {
        bytes10 n = WorkflowName.keystone("solvency");
        // sha256("solvency") starts 0x58c66935..., so the name is the ASCII of "58c66935b7"
        assertEq(n, bytes10("58c66935b7"));
    }

    function testFuzz_alwaysLowercaseHexAscii(string memory name) public pure {
        bytes10 n = WorkflowName.keystone(name);
        for (uint256 i = 0; i < 10; ++i) {
            uint8 ch = uint8(n[i]);
            assertTrue((ch >= 0x30 && ch <= 0x39) || (ch >= 0x61 && ch <= 0x66), "not lowercase hex ascii");
        }
    }
}
