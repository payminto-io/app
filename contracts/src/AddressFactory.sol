// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./DepositProxy.sol";

contract AddressFactory {
    address public immutable coldWallet;
    address public immutable sweeper;
    mapping(address => bool) public isProxy;

    event ProxyDeployed(address indexed proxy, bytes32 salt);

    constructor(address _coldWallet, address _sweeper) {
        coldWallet = _coldWallet;
        sweeper = _sweeper;
    }

    function deployProxy(bytes32 salt) external returns (address) {
        DepositProxy proxy = new DepositProxy{salt: salt}();
        address addr = address(proxy);
        isProxy[addr] = true;
        emit ProxyDeployed(addr, salt);
        return addr;
    }

    function predictAddress(bytes32 salt) external view returns (address) {
        bytes32 hash = keccak256(
            abi.encodePacked(
                bytes1(0xff),
                address(this),
                salt,
                keccak256(abi.encodePacked(type(DepositProxy).creationCode))
            )
        );
        return address(uint160(uint256(hash)));
    }
}
