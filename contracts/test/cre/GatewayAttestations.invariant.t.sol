// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";
import {ReportEncoder} from "./ReportEncoder.sol";

/// Handler: every call the fuzzer makes goes through here. Ghost state mirrors what the contract should hold
/// after each call that did not revert, including owner actions, so the invariants compare against a model
/// that moves rather than against the initial state.
contract AttestationsHandler is Test {
    GatewayAttestations public immutable c;
    bytes32 public constant GATEWAY = keccak256("https://pay.example.com");
    bytes10 public NAME;

    // ghost admin state
    address public ghostOwner;
    address public ghostPending;
    address public ghostForwarder;
    mapping(uint8 kind => bytes32) public ghostWorkflowId;
    address public workflowOwner;

    // ghost report state
    mapping(bytes32 reportHash => bool) public ghostSeen;
    mapping(uint8 kind => uint64) public ghostLatestObservedAt;
    mapping(bytes32 asset => GatewayAttestations.Solvency) public ghostSolvency;
    bytes32[] public ghostAssets;
    mapping(bytes32 => bool) private assetSeen;

    uint256 public validAccepted;
    uint256 public invalidRejected;
    uint256 public invalidAccepted;
    uint256 public ownerActions;

    constructor(GatewayAttestations c_, address owner_, address forwarder_, address workflowOwner_) {
        NAME = WorkflowName.keystone("name");
        c = c_;
        ghostOwner = owner_;
        ghostForwarder = forwarder_;
        workflowOwner = workflowOwner_;
        ghostWorkflowId[1] = keccak256("wf-solvency");
        ghostWorkflowId[2] = keccak256("wf-deposit");
        ghostWorkflowId[3] = keccak256("wf-conversion");
    }

    function _meta(uint8 kind) internal view returns (bytes memory) {
        return ReportEncoder.metadata(ghostWorkflowId[kind], NAME, workflowOwner, 0x0001);
    }

    function _observed(uint64 raw, uint8 kind, bool nearLatest) internal view returns (uint64) {
        uint64 top = uint64(block.timestamp + c.MAX_FUTURE_DRIFT());
        if (nearLatest) {
            uint64 base = ghostLatestObservedAt[kind];
            uint64 t = base + uint64(bound(raw, 0, 120));
            return t > top ? top : (t == 0 ? 1 : t);
        }
        return uint64(bound(raw, 1, top));
    }

    // ---- valid deliveries ----------------------------------------------------------------------------------

    function validSolvency(uint64 raw, bool nearLatest, uint8 n, bytes32 seed) external {
        if (ghostWorkflowId[1] == bytes32(0)) return;
        n = uint8(bound(n, 1, 6));
        uint64 observedAt = _observed(raw, 1, nearLatest);
        GatewayAttestations.SolvencyItem[] memory items = ReportEncoder.solvencyBatch(n, seed);
        for (uint256 i = 0; i < n; ++i) {
            items[i].asset = keccak256(abi.encodePacked("asset", uint256(uint8(seed[i % 32]) % 8)));
        }
        bytes memory report = ReportEncoder.solvency(GATEWAY, observedAt, items);
        bytes32 h = keccak256(report);
        bool expectOk = !ghostSeen[h];

        bytes memory payload = abi.encodeCall(c.onReport, (_meta(1), report));
        vm.prank(ghostForwarder);
        (bool ok,) = address(c).call(payload);
        assertEq(ok, expectOk, "solvency acceptance must match the stated rules");
        if (!ok) return;

        validAccepted++;
        ghostSeen[h] = true;
        if (observedAt > ghostLatestObservedAt[1]) ghostLatestObservedAt[1] = observedAt;
        for (uint256 i = 0; i < n; ++i) {
            if (observedAt <= ghostSolvency[items[i].asset].observedAt) continue; // ignored, newest wins
            ghostSolvency[items[i].asset] = GatewayAttestations.Solvency({
                checkpointHash: items[i].checkpointHash,
                liabilities: items[i].liabilities,
                reserves: items[i].reserves,
                observedAt: observedAt,
                decimals: items[i].decimals
            });
            if (!assetSeen[items[i].asset]) {
                assetSeen[items[i].asset] = true;
                ghostAssets.push(items[i].asset);
            }
        }
    }

    function validDeposits(uint64 raw, bool nearLatest, uint8 n) external {
        _validEventOnly(2, raw, nearLatest, n);
    }

    function validConversions(uint64 raw, bool nearLatest, uint8 n) external {
        _validEventOnly(3, raw, nearLatest, n);
    }

    function _validEventOnly(uint8 kind, uint64 raw, bool nearLatest, uint8 n) internal {
        if (ghostWorkflowId[kind] == bytes32(0)) return;
        uint64 observedAt = _observed(raw, kind, nearLatest);
        bytes memory report = kind == 2
            ? ReportEncoder.deposits(GATEWAY, observedAt, ReportEncoder.depositBatch(bound(n, 1, 12)))
            : ReportEncoder.conversions(GATEWAY, observedAt, ReportEncoder.conversionBatch(bound(n, 1, 10)));
        bytes32 h = keccak256(report);
        bool expectOk = !ghostSeen[h];
        bytes memory payload = abi.encodeCall(c.onReport, (_meta(kind), report));
        vm.prank(ghostForwarder);
        (bool ok,) = address(c).call(payload);
        assertEq(ok, expectOk, "event-only acceptance must match the stated rules");
        if (!ok) return;
        validAccepted++;
        ghostSeen[h] = true;
        if (observedAt > ghostLatestObservedAt[kind]) ghostLatestObservedAt[kind] = observedAt;
    }

    /// Replays a report that was already accepted: must always be rejected.
    function replayLast(uint64 raw, uint8 kind) external {
        kind = uint8(bound(kind, 1, 3));
        if (ghostWorkflowId[kind] == bytes32(0)) return;
        bytes memory report = _validReport(kind, raw);
        bytes32 h = keccak256(report);
        bytes memory payload = abi.encodeCall(c.onReport, (_meta(kind), report));
        vm.prank(ghostForwarder);
        (bool ok,) = address(c).call(payload);
        if (ghostSeen[h]) {
            assertFalse(ok, "replay accepted");
            invalidRejected++;
            return;
        }
        assertTrue(ok, "fresh report rejected");
        validAccepted++;
        ghostSeen[h] = true;
        uint64 observedAt = uint64(bound(raw, 1, block.timestamp + c.MAX_FUTURE_DRIFT()));
        if (observedAt > ghostLatestObservedAt[kind]) ghostLatestObservedAt[kind] = observedAt;
        if (kind == 1) _mirrorSolvency(ReportEncoder.solvencyBatch(2, 0), observedAt);
    }

    function _mirrorSolvency(GatewayAttestations.SolvencyItem[] memory items, uint64 observedAt) internal {
        for (uint256 i = 0; i < items.length; ++i) {
            if (observedAt <= ghostSolvency[items[i].asset].observedAt) continue;
            ghostSolvency[items[i].asset] = GatewayAttestations.Solvency({
                checkpointHash: items[i].checkpointHash,
                liabilities: items[i].liabilities,
                reserves: items[i].reserves,
                observedAt: observedAt,
                decimals: items[i].decimals
            });
            if (!assetSeen[items[i].asset]) {
                assetSeen[items[i].asset] = true;
                ghostAssets.push(items[i].asset);
            }
        }
    }

    // ---- invalid deliveries --------------------------------------------------------------------------------

    function strangerDelivers(address caller, uint64 raw, uint8 kind) external {
        vm.assume(caller != ghostForwarder);
        kind = uint8(bound(kind, 1, 3));
        bytes memory payload = abi.encodeCall(c.onReport, (_meta(kind), _validReport(kind, raw)));
        vm.prank(caller);
        (bool ok,) = address(c).call(payload);
        _countInvalid(ok);
    }

    function forwarderWrongWorkflow(bytes32 id, address owner_, bytes10 name, uint64 raw, uint8 kind) external {
        kind = uint8(bound(kind, 1, 3));
        bytes memory m = ReportEncoder.metadata(id, name, owner_, 0x0001);
        vm.assume(keccak256(m) != keccak256(_meta(kind)));
        bytes memory payload = abi.encodeCall(c.onReport, (m, _validReport(kind, raw)));
        vm.prank(ghostForwarder);
        (bool ok,) = address(c).call(payload);
        _countInvalid(ok);
    }

    function forwarderJunkReport(bytes calldata junk, uint8 kind) external {
        kind = uint8(bound(kind, 1, 3));
        bytes memory payload = abi.encodeCall(c.onReport, (_meta(kind), junk));
        vm.prank(ghostForwarder);
        (bool ok,) = address(c).call(payload);
        if (ok) revert("fuzzer produced a canonical report; tighten the junk generator");
        invalidRejected++;
    }

    function strangerAdmin(address caller, address newForwarder, bytes32 id, uint8 kind) external {
        vm.assume(caller != ghostOwner && caller != ghostPending);
        vm.startPrank(caller);
        (bool ok1,) = address(c).call(abi.encodeCall(c.setForwarder, (newForwarder)));
        (bool ok2,) = address(c).call(abi.encodeCall(c.setWorkflow, (kind, id, caller, NAME)));
        (bool ok3,) = address(c).call(abi.encodeCall(c.unbindWorkflow, (kind)));
        (bool ok4,) = address(c).call(abi.encodeCall(c.transferOwnership, (caller)));
        (bool ok5,) = address(c).call(abi.encodeCall(c.acceptOwnership, ()));
        (bool ok6,) = address(c).call(abi.encodeCall(c.renounceOwnership, ()));
        vm.stopPrank();
        assertFalse(ok1 || ok2 || ok3 || ok4 || ok5 || ok6, "non-owner admin call succeeded");
        invalidRejected += 6;
    }

    // ---- owner actions, mirrored in ghost state --------------------------------------------------------------

    function ownerSetsForwarder(address newForwarder) external {
        vm.assume(newForwarder != address(0));
        vm.prank(ghostOwner);
        c.setForwarder(newForwarder);
        ghostForwarder = newForwarder;
        ownerActions++;
    }

    function ownerRebinds(uint8 kind, bytes32 id) external {
        kind = uint8(bound(kind, 1, 3));
        vm.assume(id != bytes32(0));
        vm.prank(ghostOwner);
        c.setWorkflow(kind, id, workflowOwner, NAME);
        ghostWorkflowId[kind] = id;
        ownerActions++;
    }

    function ownerUnbinds(uint8 kind) external {
        kind = uint8(bound(kind, 1, 3));
        vm.prank(ghostOwner);
        c.unbindWorkflow(kind);
        ghostWorkflowId[kind] = bytes32(0);
        ownerActions++;
    }

    function ownerStartsTransfer(address next) external {
        vm.prank(ghostOwner);
        c.transferOwnership(next);
        ghostPending = next;
        ownerActions++;
    }

    function pendingAccepts() external {
        if (ghostPending == address(0)) return;
        vm.prank(ghostPending);
        c.acceptOwnership();
        ghostOwner = ghostPending;
        ghostPending = address(0);
        ownerActions++;
    }

    function ownerRenounceFails() external {
        vm.prank(ghostOwner);
        (bool ok,) = address(c).call(abi.encodeCall(c.renounceOwnership, ()));
        assertFalse(ok, "renounce must revert");
        invalidRejected++;
    }

    function warp(uint32 by) external {
        vm.warp(block.timestamp + bound(by, 0, 7 days));
    }

    function ghostAssetCount() external view returns (uint256) {
        return ghostAssets.length;
    }

    function _validReport(uint8 kind, uint64 raw) internal view returns (bytes memory) {
        uint64 observedAt = uint64(bound(raw, 1, block.timestamp + c.MAX_FUTURE_DRIFT()));
        if (kind == 1) return ReportEncoder.solvency(GATEWAY, observedAt, ReportEncoder.solvencyBatch(2, 0));
        if (kind == 2) return ReportEncoder.deposits(GATEWAY, observedAt, ReportEncoder.depositBatch(2));
        return ReportEncoder.conversions(GATEWAY, observedAt, ReportEncoder.conversionBatch(2));
    }

    function _countInvalid(bool ok) internal {
        if (ok) invalidAccepted++;
        else invalidRejected++;
    }
}

contract GatewayAttestationsInvariantTest is Test {
    GatewayAttestations internal c;
    AttestationsHandler internal h;

    address internal forwarder = makeAddr("forwarder");
    address internal owner = makeAddr("owner");
    address internal workflowOwner = makeAddr("workflowOwner");

    function setUp() public {
        vm.warp(1_800_000_000);
        c = new GatewayAttestations(forwarder, owner);
        h = new AttestationsHandler(c, owner, forwarder, workflowOwner);
        vm.startPrank(owner);
        c.setWorkflow(1, h.ghostWorkflowId(1), workflowOwner, h.NAME());
        c.setWorkflow(2, h.ghostWorkflowId(2), workflowOwner, h.NAME());
        c.setWorkflow(3, h.ghostWorkflowId(3), workflowOwner, h.NAME());
        vm.stopPrank();
        targetContract(address(h));
    }

    function invariant_stateOnlyChangesThroughValidReports() public view {
        assertEq(h.invalidAccepted(), 0, "an invalid delivery was accepted");
        for (uint8 kind = 1; kind <= 3; ++kind) {
            assertEq(c.latestObservedAt(h.GATEWAY(), kind), h.ghostLatestObservedAt(kind), "latestObservedAt drifted");
        }
        uint256 n = h.ghostAssetCount();
        for (uint256 i = 0; i < n; ++i) {
            bytes32 asset = h.ghostAssets(i);
            (bytes32 ckpt, uint256 liabilities, uint256 reserves, uint64 observedAt, uint8 decimals) =
                h.ghostSolvency(asset);
            GatewayAttestations.Solvency memory s = c.getLatestSolvency(h.GATEWAY(), asset);
            assertEq(s.checkpointHash, ckpt);
            assertEq(s.liabilities, liabilities);
            assertEq(s.reserves, reserves);
            assertEq(s.observedAt, observedAt);
            assertEq(s.decimals, decimals);
        }
    }

    function invariant_adminStateFollowsOwnerOnly() public view {
        assertEq(c.owner(), h.ghostOwner());
        assertEq(c.pendingOwner(), h.ghostPending());
        assertEq(c.forwarder(), h.ghostForwarder());
        for (uint8 kind = 1; kind <= 3; ++kind) {
            assertEq(c.getWorkflow(kind).id, h.ghostWorkflowId(kind));
        }
        assertEq(address(c).balance, 0);
    }

    /// Runs after each sequence: the accept path must actually have been exercised.
    function afterInvariant() public view {
        assertGt(h.validAccepted(), 0, "no valid report accepted in this run");
    }
}
