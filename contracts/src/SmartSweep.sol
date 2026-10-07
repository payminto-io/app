// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";

contract SmartSweep is Pausable {
    using SafeERC20 for IERC20;

    address public immutable coldWallet;
    address public immutable sweeper;
    address public owner;

    event SweptETH(uint256 amount);
    event SweptToken(address indexed token, uint256 amount);

    modifier onlySweeper() {
        require(msg.sender == sweeper, "only sweeper");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "only owner");
        _;
    }

    constructor(address _coldWallet, address _sweeper) {
        require(_coldWallet != address(0), "zero cold wallet");
        require(_sweeper != address(0), "zero sweeper");
        coldWallet = _coldWallet;
        sweeper = _sweeper;
        owner = msg.sender;
    }

    receive() external payable {}

    function sweepETH() external onlySweeper whenNotPaused {
        uint256 balance = address(this).balance;
        require(balance > 0, "no ETH balance");
        (bool success,) = coldWallet.call{value: balance}("");
        require(success, "ETH transfer failed");
        emit SweptETH(balance);
    }

    function sweepToken(address token, uint256 amount) external onlySweeper whenNotPaused {
        require(amount > 0, "zero amount");
        IERC20(token).safeTransfer(coldWallet, amount);
        emit SweptToken(token, amount);
    }

    function sweepAllTokens(address token) external onlySweeper whenNotPaused {
        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance > 0, "no token balance");
        IERC20(token).safeTransfer(coldWallet, balance);
        emit SweptToken(token, balance);
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }
}
