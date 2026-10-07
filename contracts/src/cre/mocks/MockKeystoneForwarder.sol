// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {ERC165Checker} from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import {IReceiver} from "../IReceiver.sol";

/// @title MockKeystoneForwarder
/// @notice Simulation stand-in for Chainlink's KeystoneForwarder with the same external surface and the same
/// raw-report slicing (metadata = rawReport[45:109], report = rawReport[109:]) but no signature verification.
/// `cre workflow simulate --broadcast` delivers through a contract of this shape; this one lives on a local
/// Anvil only. Deploying it to a public network would let anyone write to a consumer that trusts it.
contract MockKeystoneForwarder {
    enum TransmissionState {
        NOT_ATTEMPTED,
        SUCCEEDED,
        INVALID_RECEIVER,
        FAILED
    }

    struct TransmissionInfo {
        bytes32 transmissionId;
        TransmissionState state;
        address transmitter;
        bool invalidReceiver;
        bool success;
        uint80 gasLimit;
    }

    struct Transmission {
        address transmitter;
        bool invalidReceiver;
        bool success;
        uint80 gasLimit;
    }

    uint256 internal constant METADATA_LENGTH = 109;
    uint256 internal constant FORWARDER_METADATA_LENGTH = 45;

    string public constant typeAndVersion = "MockKeystoneForwarder 1.0.0 (gateway simulation only)";

    mapping(bytes32 transmissionId => Transmission transmission) internal s_transmissions;

    event ReportProcessed(
        address indexed receiver, bytes32 indexed workflowExecutionId, bytes2 indexed reportId, bool result
    );
    /// @dev Mock-only: the real forwarder drops the receiver's revert data; keeping it makes a local failure legible.
    event ReceiverReverted(address indexed receiver, bytes32 indexed workflowExecutionId, bytes reason);

    error InvalidReport();
    error AlreadyAttempted(bytes32 transmissionId);

    /// @notice Same signature as KeystoneForwarder.report; reportContext and signatures are accepted and ignored.
    function report(address receiver, bytes calldata rawReport, bytes calldata, bytes[] calldata) external {
        if (rawReport.length < METADATA_LENGTH) revert InvalidReport();
        bytes32 workflowExecutionId = bytes32(rawReport[1:33]);
        bytes2 reportId = bytes2(rawReport[107:109]);
        bytes32 transmissionId = getTransmissionId(receiver, workflowExecutionId, reportId);

        Transmission storage transmission = s_transmissions[transmissionId];
        if (transmission.success || transmission.invalidReceiver) revert AlreadyAttempted(transmissionId);
        transmission.transmitter = msg.sender;
        transmission.gasLimit = uint80(gasleft());

        if (!ERC165Checker.supportsInterface(receiver, type(IReceiver).interfaceId)) {
            transmission.invalidReceiver = true;
            emit ReportProcessed(receiver, workflowExecutionId, reportId, false);
            return;
        }

        (bool success, bytes memory reason) = receiver.call(
            abi.encodeCall(
                IReceiver.onReport, (rawReport[FORWARDER_METADATA_LENGTH:METADATA_LENGTH], rawReport[METADATA_LENGTH:])
            )
        );
        if (success) {
            transmission.success = true;
        } else {
            emit ReceiverReverted(receiver, workflowExecutionId, reason);
        }
        emit ReportProcessed(receiver, workflowExecutionId, reportId, success);
    }

    function getTransmissionId(address receiver, bytes32 workflowExecutionId, bytes2 reportId)
        public
        pure
        returns (bytes32)
    {
        return keccak256(bytes.concat(bytes20(uint160(receiver)), workflowExecutionId, reportId));
    }

    function getTransmissionInfo(address receiver, bytes32 workflowExecutionId, bytes2 reportId)
        external
        view
        returns (TransmissionInfo memory)
    {
        bytes32 transmissionId = getTransmissionId(receiver, workflowExecutionId, reportId);
        Transmission memory transmission = s_transmissions[transmissionId];
        TransmissionState state;
        if (transmission.transmitter == address(0)) {
            state = TransmissionState.NOT_ATTEMPTED;
        } else if (transmission.invalidReceiver) {
            state = TransmissionState.INVALID_RECEIVER;
        } else {
            state = transmission.success ? TransmissionState.SUCCEEDED : TransmissionState.FAILED;
        }
        return TransmissionInfo({
            transmissionId: transmissionId,
            state: state,
            transmitter: transmission.transmitter,
            invalidReceiver: transmission.invalidReceiver,
            success: transmission.success,
            gasLimit: transmission.gasLimit
        });
    }

    function getTransmitter(address receiver, bytes32 workflowExecutionId, bytes2 reportId)
        external
        view
        returns (address)
    {
        return s_transmissions[getTransmissionId(receiver, workflowExecutionId, reportId)].transmitter;
    }
}
