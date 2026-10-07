// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

/// @notice Keystone's HashTruncateName: the metadata workflow name is the ASCII bytes of the first ten hex
/// characters of sha256(name) (chainlink-common pkg/workflows/utils.go; ReceiverTemplate.setExpectedWorkflowName).
/// Documented example: "my_workflow" -> 0x62373666336165316465.
library WorkflowName {
    bytes16 private constant HEX = "0123456789abcdef";

    function keystone(string memory name) internal pure returns (bytes10 out) {
        bytes32 digest = sha256(bytes(name));
        bytes memory ascii = new bytes(10);
        for (uint256 i = 0; i < 5; ++i) {
            uint8 b = uint8(digest[i]);
            ascii[2 * i] = HEX[b >> 4];
            ascii[2 * i + 1] = HEX[b & 0x0f];
        }
        out = bytes10(ascii);
    }
}
