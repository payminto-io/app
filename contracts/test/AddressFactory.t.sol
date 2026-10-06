// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/AddressFactory.sol";
import "../src/DepositProxy.sol";

contract AddressFactoryTest is Test {
    AddressFactory public factory;
    address public coldWallet = address(0xC01D);
    address public sweeper = address(0x5EE0);

    function setUp() public {
        factory = new AddressFactory(coldWallet, sweeper);
    }

    function test_DeployProxy() public {
        bytes32 salt = keccak256(abi.encodePacked("payment_1"));
        address proxy = factory.deployProxy(salt);
        assertTrue(proxy != address(0));
        assertTrue(factory.isProxy(proxy));
    }

    function test_DeployProxy_Deterministic() public {
        bytes32 salt = keccak256(abi.encodePacked("payment_1"));
        address predicted = factory.predictAddress(salt);
        address actual = factory.deployProxy(salt);
        assertEq(predicted, actual);
    }

    function test_ProxyReceivesETH() public {
        bytes32 salt = keccak256(abi.encodePacked("payment_2"));
        address proxy = factory.deployProxy(salt);
        vm.deal(proxy, 1 ether);
        assertEq(proxy.balance, 1 ether);
    }

    function test_ProxyFactoryIsSet() public {
        bytes32 salt = keccak256(abi.encodePacked("payment_3"));
        address proxyAddr = factory.deployProxy(salt);
        DepositProxy proxy = DepositProxy(payable(proxyAddr));
        assertEq(proxy.factory(), address(factory));
    }

    function test_ProxyExecuteOnlyFactory() public {
        bytes32 salt = keccak256(abi.encodePacked("payment_4"));
        address proxyAddr = factory.deployProxy(salt);
        DepositProxy proxy = DepositProxy(payable(proxyAddr));
        vm.expectRevert("only factory");
        proxy.execute(address(0), "");
    }
}
