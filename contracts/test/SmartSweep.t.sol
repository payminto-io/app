// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/SmartSweep.sol";

contract SmartSweepTest is Test {
    SmartSweep public sweep;
    address public coldWallet = address(0xC01D);
    address public sweeper = address(0x5EE0);
    address public attacker = address(0xBAD);

    function setUp() public {
        sweep = new SmartSweep(coldWallet, sweeper);
    }

    function test_ColdWalletIsImmutable() public view {
        assertEq(sweep.coldWallet(), coldWallet);
    }

    function test_SweeperIsSet() public view {
        assertEq(sweep.sweeper(), sweeper);
    }

    function test_SweepETH() public {
        vm.deal(address(sweep), 1 ether);
        vm.prank(sweeper);
        sweep.sweepETH();
        assertEq(address(sweep).balance, 0);
        assertEq(coldWallet.balance, 1 ether);
    }

    function test_SweepETH_OnlySweeper() public {
        vm.deal(address(sweep), 1 ether);
        vm.prank(attacker);
        vm.expectRevert("only sweeper");
        sweep.sweepETH();
    }

    function test_SweepETH_NoBalance() public {
        vm.prank(sweeper);
        vm.expectRevert("no ETH balance");
        sweep.sweepETH();
    }

    function test_Pause() public {
        sweep.pause();
        vm.deal(address(sweep), 1 ether);
        vm.prank(sweeper);
        vm.expectRevert();
        sweep.sweepETH();
    }

    function test_Unpause() public {
        sweep.pause();
        sweep.unpause();
        vm.deal(address(sweep), 1 ether);
        vm.prank(sweeper);
        sweep.sweepETH();
        assertEq(coldWallet.balance, 1 ether);
    }

    function test_OnlyOwnerCanPause() public {
        vm.prank(attacker);
        vm.expectRevert("only owner");
        sweep.pause();
    }

    function test_ReceiveETH() public {
        vm.deal(address(this), 1 ether);
        (bool success,) = address(sweep).call{value: 1 ether}("");
        assertTrue(success);
        assertEq(address(sweep).balance, 1 ether);
    }

    function test_ZeroColdWallet() public {
        vm.expectRevert("zero cold wallet");
        new SmartSweep(address(0), sweeper);
    }

    function test_ZeroSweeper() public {
        vm.expectRevert("zero sweeper");
        new SmartSweep(coldWallet, address(0));
    }

    function testFuzz_SweepETH_AnyAmount(uint96 amount) public {
        vm.assume(amount > 0);
        vm.deal(address(sweep), amount);
        vm.prank(sweeper);
        sweep.sweepETH();
        assertEq(coldWallet.balance, amount);
    }
}
