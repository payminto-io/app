// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/SmartSweep.sol";

contract GasTest is Test {
    SmartSweep public sweep;

    function setUp() public {
        sweep = new SmartSweep(address(0xC01D), address(this));
    }

    function test_SweepETH_Gas() public {
        vm.deal(address(sweep), 1 ether);
        uint256 gasBefore = gasleft();
        sweep.sweepETH();
        uint256 gasUsed = gasBefore - gasleft();
        assertLt(gasUsed, 50000, "sweepETH should use < 50k gas");
    }
}
