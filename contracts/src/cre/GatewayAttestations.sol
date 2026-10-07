// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Ownable2Step} from "@openzeppelin/contracts/access/Ownable2Step.sol";
import {IERC165} from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import {IReceiver} from "./IReceiver.sol";

/// @title GatewayAttestations
/// @notice Chainlink CRE consumer that records the gateway's attestations (docs/cre/SPEC.md section 5).
/// The contract holds no funds, has no upgrade path and no business thresholds. It accepts a report only from
/// the configured Keystone forwarder, only for the workflow bound to the report's kind, only once per distinct
/// report (keccak256 seen-set), and it rejects anything that does not decode to exactly one versioned batch.
/// Solvency keeps the newest snapshot per (gatewayId, asset) and ignores older items with an event; deposit and
/// conversion kinds are events only, accepted in any order. Replay rules: docs/cre/SPEC.md section 5.
contract GatewayAttestations is IReceiver, IERC165, Ownable2Step {
    uint8 public constant REPORT_VERSION = 1;
    uint8 public constant KIND_SOLVENCY = 1;
    uint8 public constant KIND_DEPOSIT_FINALITY = 2;
    uint8 public constant KIND_CONVERSION_REFERENCE = 3;

    uint8 public constant VERDICT_CONFIRMED = 1;
    uint8 public constant VERDICT_NOT_FOUND = 2;
    uint8 public constant VERDICT_MISMATCH = 3;

    /// @dev DON time may lead the block by a little; anything further ahead is a bad clock, not a report.
    uint64 public constant MAX_FUTURE_DRIFT = 5 minutes;
    /// @dev Keystone metadata: workflowId(32) | workflowName(10) | workflowOwner(20) | reportId(2).
    uint256 public constant METADATA_LENGTH = 64;
    /// @dev abi.encode(uint8 version, uint8 kind, bytes32 gatewayId, uint64 observedAt, Item[] items) head.
    uint256 private constant HEADER_WORDS = 6;
    uint256 private constant ITEMS_OFFSET = 5 * 32;
    uint256 private constant SOLVENCY_ITEM_WORDS = 5;
    uint256 private constant DEPOSIT_ITEM_WORDS = 8;
    uint256 private constant CONVERSION_ITEM_WORDS = 7;

    struct Workflow {
        bytes32 id;
        address owner;
        bytes10 name;
    }

    struct SolvencyItem {
        bytes32 checkpointHash;
        bytes32 asset;
        uint256 liabilities;
        uint256 reserves;
        uint8 decimals;
    }

    struct DepositItem {
        bytes32 depositId;
        bytes32 chainId;
        bytes32 txRef;
        bytes32 token;
        uint256 amount;
        bytes32 destination;
        uint64 slotOrBlock;
        uint8 verdict;
    }

    struct ConversionItem {
        bytes32 conversionId;
        bytes32 pair;
        int256 referenceRate;
        uint8 referenceDecimals;
        int256 deviationBps;
        address feed;
        uint80 roundId;
    }

    struct Metadata {
        bytes32 workflowId;
        bytes10 workflowName;
        address workflowOwner;
        bytes2 reportId;
    }

    struct Solvency {
        bytes32 checkpointHash;
        uint256 liabilities;
        uint256 reserves;
        uint64 observedAt;
        uint8 decimals;
    }

    address public forwarder;
    mapping(uint8 kind => Workflow) public workflows;
    mapping(bytes32 reportHash => bool) public reportSeen;
    /// @dev Highest observedAt accepted per (gatewayId, kind); informational for staleness reads, never a gate.
    mapping(bytes32 gatewayId => mapping(uint8 kind => uint64)) public latestObservedAt;
    mapping(bytes32 gatewayId => mapping(bytes32 asset => Solvency)) public latestSolvency;

    event ForwarderSet(address indexed previous, address indexed current);
    event WorkflowBound(
        uint8 indexed kind, bytes32 indexed workflowId, address indexed workflowOwner, bytes10 workflowName
    );
    event WorkflowUnbound(uint8 indexed kind);
    event ReportAccepted(
        bytes32 indexed gatewayId,
        uint8 indexed kind,
        bytes32 indexed workflowId,
        address workflowOwner,
        bytes10 workflowName,
        bytes2 reportId,
        uint64 observedAt,
        uint256 itemCount,
        bytes32 reportHash
    );
    event SolvencyAttested(
        bytes32 indexed gatewayId,
        bytes32 indexed asset,
        bytes32 checkpointHash,
        uint256 liabilities,
        uint256 reserves,
        uint8 decimals,
        uint64 observedAt
    );
    event SolvencyIgnored(bytes32 indexed gatewayId, bytes32 indexed asset, uint64 observedAt, uint64 latestObservedAt);
    event DepositAttested(
        bytes32 indexed gatewayId,
        bytes32 indexed depositId,
        uint8 indexed verdict,
        bytes32 chainId,
        bytes32 txRef,
        bytes32 token,
        uint256 amount,
        bytes32 destination,
        uint64 slotOrBlock,
        uint64 observedAt
    );
    event ConversionReferenceAttested(
        bytes32 indexed gatewayId,
        bytes32 indexed conversionId,
        bytes32 indexed pair,
        int256 referenceRate,
        uint8 referenceDecimals,
        int256 deviationBps,
        address feed,
        uint80 roundId,
        uint64 observedAt
    );

    error ZeroAddress();
    error UnauthorizedForwarder(address caller);
    error InvalidMetadataLength(uint256 length);
    error UnknownKind(uint8 kind);
    error WorkflowNotBound(uint8 kind);
    error UnexpectedWorkflow(uint8 kind, bytes32 workflowId, address workflowOwner, bytes10 workflowName);
    error UnsupportedReportVersion(uint256 version);
    error MalformedReport(uint256 length);
    error EmptyReport();
    error ObservedAtInFuture(uint64 observedAt, uint256 blockTimestamp);
    error DuplicateReport(bytes32 reportHash);
    error ZeroWorkflowName();
    error RenounceDisabled();
    error InvalidVerdict(uint8 verdict);

    modifier onlyForwarder() {
        _checkForwarder();
        _;
    }

    constructor(address forwarder_, address initialOwner) Ownable(initialOwner) {
        if (forwarder_ == address(0)) revert ZeroAddress();
        forwarder = forwarder_;
        emit ForwarderSet(address(0), forwarder_);
    }

    function _checkForwarder() private view {
        if (msg.sender != forwarder) revert UnauthorizedForwarder(msg.sender);
    }

    // ---- admin ---------------------------------------------------------------------------------------------

    function setForwarder(address forwarder_) external onlyOwner {
        if (forwarder_ == address(0)) revert ZeroAddress();
        emit ForwarderSet(forwarder, forwarder_);
        forwarder = forwarder_;
    }

    /// @notice Binds a report kind to exactly one workflow (id, owner, name hash); a new workflow id must be rebound.
    function setWorkflow(uint8 kind, bytes32 workflowId, address workflowOwner, bytes10 workflowName)
        external
        onlyOwner
    {
        if (!_isKnownKind(kind)) revert UnknownKind(kind);
        if (workflowId == bytes32(0)) revert WorkflowNotBound(kind);
        if (workflowOwner == address(0)) revert ZeroAddress();
        if (workflowName == bytes10(0)) revert ZeroWorkflowName();
        workflows[kind] = Workflow({id: workflowId, owner: workflowOwner, name: workflowName});
        emit WorkflowBound(kind, workflowId, workflowOwner, workflowName);
    }

    function unbindWorkflow(uint8 kind) external onlyOwner {
        if (!_isKnownKind(kind)) revert UnknownKind(kind);
        delete workflows[kind];
        emit WorkflowUnbound(kind);
    }

    /// @notice Forwarder rotation needs an owner for the life of the contract, so ownership cannot be renounced.
    function renounceOwnership() public view override onlyOwner {
        revert RenounceDisabled();
    }

    // ---- IReceiver -----------------------------------------------------------------------------------------

    function onReport(bytes calldata metadata, bytes calldata report) external override onlyForwarder {
        Metadata memory m = _decodeMetadata(metadata);
        (uint8 kind, bytes32 gatewayId, uint64 observedAt, uint256 itemCount) = _decodeHeader(report);
        _requireBoundWorkflow(kind, m);
        bytes32 reportHash = keccak256(report);
        _admit(gatewayId, kind, observedAt, reportHash);

        if (kind == KIND_SOLVENCY) {
            _recordSolvency(report, gatewayId, observedAt);
        } else if (kind == KIND_DEPOSIT_FINALITY) {
            _recordDeposits(report, gatewayId, observedAt);
        } else {
            _recordConversions(report, gatewayId, observedAt);
        }

        emit ReportAccepted(
            gatewayId,
            kind,
            m.workflowId,
            m.workflowOwner,
            m.workflowName,
            m.reportId,
            observedAt,
            itemCount,
            reportHash
        );
    }

    function supportsInterface(bytes4 interfaceId) public pure override returns (bool) {
        return interfaceId == type(IReceiver).interfaceId || interfaceId == type(IERC165).interfaceId;
    }

    // ---- views ---------------------------------------------------------------------------------------------

    function getWorkflow(uint8 kind) external view returns (Workflow memory) {
        return workflows[kind];
    }

    function getLatestSolvency(bytes32 gatewayId, bytes32 asset) external view returns (Solvency memory) {
        return latestSolvency[gatewayId][asset];
    }

    // ---- decoding ------------------------------------------------------------------------------------------

    function _decodeMetadata(bytes calldata metadata) private pure returns (Metadata memory m) {
        if (metadata.length != METADATA_LENGTH) revert InvalidMetadataLength(metadata.length);
        m.workflowId = bytes32(metadata[0:32]);
        m.workflowName = bytes10(metadata[32:42]);
        m.workflowOwner = address(bytes20(metadata[42:62]));
        m.reportId = bytes2(metadata[62:64]);
    }

    function _requireBoundWorkflow(uint8 kind, Metadata memory m) private view {
        Workflow storage expected = workflows[kind];
        if (expected.id == bytes32(0)) revert WorkflowNotBound(kind);
        if (expected.id != m.workflowId || expected.owner != m.workflowOwner || expected.name != m.workflowName) {
            revert UnexpectedWorkflow(kind, m.workflowId, m.workflowOwner, m.workflowName);
        }
    }

    /// @dev Replay guard: each distinct report is accepted once; order between genuine reports is not enforced.
    function _admit(bytes32 gatewayId, uint8 kind, uint64 observedAt, bytes32 reportHash) private {
        if (observedAt > block.timestamp + MAX_FUTURE_DRIFT) revert ObservedAtInFuture(observedAt, block.timestamp);
        if (reportSeen[reportHash]) revert DuplicateReport(reportHash);
        reportSeen[reportHash] = true;
        if (observedAt > latestObservedAt[gatewayId][kind]) latestObservedAt[gatewayId][kind] = observedAt;
    }

    /// @dev Checks the version, the kind, the canonical array offset and the exact byte length before any
    /// abi.decode runs, so a truncated, padded or mis-typed report fails with a named error.
    function _decodeHeader(bytes calldata report)
        private
        pure
        returns (uint8 kind, bytes32 gatewayId, uint64 observedAt, uint256 itemCount)
    {
        if (report.length < HEADER_WORDS * 32) revert MalformedReport(report.length);
        uint256 version = uint256(bytes32(report[0:32]));
        if (version != REPORT_VERSION) revert UnsupportedReportVersion(version);
        uint256 kindWord = uint256(bytes32(report[32:64]));
        if (kindWord > type(uint8).max) revert MalformedReport(report.length);
        kind = uint8(kindWord); // forge-lint: disable-line(unsafe-typecast)
        if (!_isKnownKind(kind)) revert UnknownKind(kind);
        gatewayId = bytes32(report[64:96]);
        uint256 observedWord = uint256(bytes32(report[96:128]));
        if (observedWord > type(uint64).max) revert MalformedReport(report.length);
        observedAt = uint64(observedWord); // forge-lint: disable-line(unsafe-typecast)
        if (uint256(bytes32(report[128:160])) != ITEMS_OFFSET) revert MalformedReport(report.length);
        itemCount = uint256(bytes32(report[160:192]));
        if (itemCount == 0) revert EmptyReport();
        if (itemCount > (report.length - HEADER_WORDS * 32) / 32) revert MalformedReport(report.length);
        uint256 expected = HEADER_WORDS * 32 + itemCount * _itemWords(kind) * 32;
        if (report.length != expected) revert MalformedReport(report.length);
    }

    function _recordSolvency(bytes calldata report, bytes32 gatewayId, uint64 observedAt) private {
        (,,,, SolvencyItem[] memory items) = abi.decode(report, (uint8, uint8, bytes32, uint64, SolvencyItem[]));
        for (uint256 i = 0; i < items.length; ++i) {
            SolvencyItem memory item = items[i];
            Solvency storage latest = latestSolvency[gatewayId][item.asset];
            // Newest snapshot wins per asset; an older or equal one (a late delivery, or a duplicate asset in
            // the batch) is recorded as ignored and never overwrites.
            if (observedAt <= latest.observedAt) {
                emit SolvencyIgnored(gatewayId, item.asset, observedAt, latest.observedAt);
                continue;
            }
            latest.checkpointHash = item.checkpointHash;
            latest.liabilities = item.liabilities;
            latest.reserves = item.reserves;
            latest.observedAt = observedAt;
            latest.decimals = item.decimals;
            emit SolvencyAttested(
                gatewayId, item.asset, item.checkpointHash, item.liabilities, item.reserves, item.decimals, observedAt
            );
        }
    }

    function _recordDeposits(bytes calldata report, bytes32 gatewayId, uint64 observedAt) private {
        (,,,, DepositItem[] memory items) = abi.decode(report, (uint8, uint8, bytes32, uint64, DepositItem[]));
        for (uint256 i = 0; i < items.length; ++i) {
            DepositItem memory item = items[i];
            if (item.verdict < VERDICT_CONFIRMED || item.verdict > VERDICT_MISMATCH) {
                revert InvalidVerdict(item.verdict);
            }
            emit DepositAttested(
                gatewayId,
                item.depositId,
                item.verdict,
                item.chainId,
                item.txRef,
                item.token,
                item.amount,
                item.destination,
                item.slotOrBlock,
                observedAt
            );
        }
    }

    function _recordConversions(bytes calldata report, bytes32 gatewayId, uint64 observedAt) private {
        (,,,, ConversionItem[] memory items) = abi.decode(report, (uint8, uint8, bytes32, uint64, ConversionItem[]));
        for (uint256 i = 0; i < items.length; ++i) {
            ConversionItem memory item = items[i];
            emit ConversionReferenceAttested(
                gatewayId,
                item.conversionId,
                item.pair,
                item.referenceRate,
                item.referenceDecimals,
                item.deviationBps,
                item.feed,
                item.roundId,
                observedAt
            );
        }
    }

    function _isKnownKind(uint8 kind) private pure returns (bool) {
        return kind >= KIND_SOLVENCY && kind <= KIND_CONVERSION_REFERENCE;
    }

    function _itemWords(uint8 kind) private pure returns (uint256) {
        if (kind == KIND_SOLVENCY) return SOLVENCY_ITEM_WORDS;
        if (kind == KIND_DEPOSIT_FINALITY) return DEPOSIT_ITEM_WORDS;
        return CONVERSION_ITEM_WORDS;
    }
}
