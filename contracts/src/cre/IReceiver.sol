// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

// Chainlink CRE receiver interface. Copied per the CRE consumer-contract guide; it is not a package.
interface IReceiver {
    function onReport(bytes calldata metadata, bytes calldata report) external;
}
