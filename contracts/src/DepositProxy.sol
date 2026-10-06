// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract DepositProxy {
    address public immutable factory;

    constructor() {
        factory = msg.sender;
    }

    receive() external payable {}

    function execute(address target, bytes calldata data) external returns (bytes memory) {
        require(msg.sender == factory, "only factory");
        (bool success, bytes memory result) = target.call(data);
        require(success, "execution failed");
        return result;
    }
}
