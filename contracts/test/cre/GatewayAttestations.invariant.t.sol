// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {ReportEncoder} from "./ReportEncoder.sol";

/// Handler: every call the fuzzer makes goes through here. The ghost state mirrors what the contract should
/// hold after each call that did not revert, so the invariant can assert that nothing else moved it.
contract AttestationsHandler is Test {
    GatewayAttestations public immutable c;
    address public immutable forwarder;
    address public immutable workflowOwner;

    bytes32 public constant GATEWAY = keccak256("https://pay.example.com");
    bytes32 public constant WF_SOLVENCY = keccak256("wf-solvency");
    bytes32 public constant WF_DEPOSIT = keccak256("wf-deposit");
    bytes32 public constant WF_CONVERSION = keccak256("wf-conversion");
    bytes10 public NAME;

    // ghost: expected contract state
    mapping(uint8 kind => uint64) public ghostLastObservedAt;
    mapping(bytes32 asset => GatewayAttestations.Solvency) public ghostSolvency;
    bytes32[] public ghostAssets;
    mapping(bytes32 => bool) private assetSeen;

    uint256 public validAccepted;
    uint256 public invalidRejected;
    uint256 public invalidAccepted;

    constructor(GatewayAttestations c_, address forwarder_, address workflowOwner_) {
        NAME = bytes10(sha256("name"));
        c = c_;
        forwarder = forwarder_;
        workflowOwner = workflowOwner_;
    }

    function _meta(uint8 kind) internal view returns (bytes memory) {
        bytes32 id = kind == 1 ? WF_SOLVENCY : kind == 2 ? WF_DEPOSIT : WF_CONVERSION;
        return ReportEncoder.metadata(id, NAME, workflowOwner, 0x0001);
    }

    /// A well-formed solvency report from the real forwarder with an observedAt chosen by the fuzzer.
    function validSolvency(uint64 observedAt, uint8 n, bytes32 seed) external {
        n = uint8(bound(n, 1, 6));
        observedAt = uint64(bound(observedAt, 1, block.timestamp + c.MAX_FUTURE_DRIFT()));
        GatewayAttestations.SolvencyItem[] memory items = ReportEncoder.solvencyBatch(n, seed);
        for (uint256 i = 0; i < n; ++i) {
            items[i].asset = keccak256(abi.encodePacked("asset", uint256(uint8(seed[i % 32]) % 8)));
        }
        bytes memory report = ReportEncoder.solvency(GATEWAY, observedAt, items);

        bool expectOk = observedAt > ghostLastObservedAt[1];
        if (expectOk) {
            for (uint256 i = 0; i < n && expectOk; ++i) {
                for (uint256 j = 0; j < i; ++j) {
                    if (items[j].asset == items[i].asset) expectOk = false;
                }
                if (observedAt <= ghostSolvency[items[i].asset].observedAt) expectOk = false;
            }
        }

        vm.prank(forwarder);
        (bool ok,) = address(c).call(abi.encodeCall(c.onReport, (_meta(1), report)));
        assertEq(ok, expectOk, "solvency acceptance must match the stated rules");
        if (!ok) return;

        validAccepted++;
        ghostLastObservedAt[1] = observedAt;
        for (uint256 i = 0; i < n; ++i) {
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

    function validDeposits(uint64 observedAt, uint8 n) external {
        n = uint8(bound(n, 1, 12));
        observedAt = uint64(bound(observedAt, 1, block.timestamp + c.MAX_FUTURE_DRIFT()));
        bytes memory report = ReportEncoder.deposits(GATEWAY, observedAt, ReportEncoder.depositBatch(n));
        bool expectOk = observedAt > ghostLastObservedAt[2];
        vm.prank(forwarder);
        (bool ok,) = address(c).call(abi.encodeCall(c.onReport, (_meta(2), report)));
        assertEq(ok, expectOk, "deposit acceptance must match the stated rules");
        if (!ok) return;
        validAccepted++;
        ghostLastObservedAt[2] = observedAt;
    }

    function validConversions(uint64 observedAt, uint8 n) external {
        n = uint8(bound(n, 1, 10));
        observedAt = uint64(bound(observedAt, 1, block.timestamp + c.MAX_FUTURE_DRIFT()));
        bytes memory report = ReportEncoder.conversions(GATEWAY, observedAt, ReportEncoder.conversionBatch(n));
        bool expectOk = observedAt > ghostLastObservedAt[3];
        vm.prank(forwarder);
        (bool ok,) = address(c).call(abi.encodeCall(c.onReport, (_meta(3), report)));
        assertEq(ok, expectOk, "conversion acceptance must match the stated rules");
        if (!ok) return;
        validAccepted++;
        ghostLastObservedAt[3] = observedAt;
    }

    /// A well-formed report from anyone but the forwarder.
    function strangerDelivers(address caller, uint64 observedAt, uint8 kind) external {
        vm.assume(caller != forwarder);
        kind = uint8(bound(kind, 1, 3));
        bytes memory report = _validReport(kind, observedAt);
        vm.prank(caller);
        (bool ok,) = address(c).call(abi.encodeCall(c.onReport, (_meta(kind), report)));
        _countInvalid(ok);
    }

    /// The forwarder with metadata that names a workflow other than the bound one.
    function forwarderWrongWorkflow(bytes32 id, address owner_, bytes10 name, uint64 observedAt, uint8 kind) external {
        kind = uint8(bound(kind, 1, 3));
        bytes memory good = _meta(kind);
        bytes memory m = ReportEncoder.metadata(id, name, owner_, 0x0001);
        vm.assume(keccak256(m) != keccak256(good));
        vm.prank(forwarder);
        (bool ok,) = address(c).call(abi.encodeCall(c.onReport, (m, _validReport(kind, observedAt))));
        _countInvalid(ok);
    }

    /// The forwarder with arbitrary bytes as the report.
    function forwarderJunkReport(bytes calldata junk, uint8 kind) external {
        kind = uint8(bound(kind, 1, 3));
        vm.prank(forwarder);
        (bool ok,) = address(c).call(abi.encodeCall(c.onReport, (_meta(kind), junk)));
        // Junk that happens to be a valid, fresh report is accepted by definition; account for it honestly.
        if (ok) {
            (, uint8 k,, uint64 t,) = abi.decode(junk, (uint8, uint8, bytes32, uint64, bytes));
            k;
            t;
            revert("fuzzer produced a canonical report; tighten the junk generator");
        }
        invalidRejected++;
    }

    /// Any address attempting admin.
    function strangerAdmin(address caller, address newForwarder, bytes32 id, uint8 kind) external {
        vm.assume(caller != c.owner() && caller != c.pendingOwner());
        vm.startPrank(caller);
        (bool ok1,) = address(c).call(abi.encodeCall(c.setForwarder, (newForwarder)));
        (bool ok2,) = address(c).call(abi.encodeCall(c.setWorkflow, (kind, id, caller, NAME)));
        (bool ok3,) = address(c).call(abi.encodeCall(c.unbindWorkflow, (kind)));
        (bool ok4,) = address(c).call(abi.encodeCall(c.transferOwnership, (caller)));
        (bool ok5,) = address(c).call(abi.encodeCall(c.acceptOwnership, ()));
        vm.stopPrank();
        assertFalse(ok1 || ok2 || ok3 || ok4 || ok5, "non-owner admin call succeeded");
        invalidRejected += 5;
    }

    function warp(uint32 by) external {
        vm.warp(block.timestamp + bound(by, 0, 7 days));
    }

    function ghostAssetCount() external view returns (uint256) {
        return ghostAssets.length;
    }

    function _validReport(uint8 kind, uint64 observedAt) internal view returns (bytes memory) {
        observedAt = uint64(bound(observedAt, 1, block.timestamp + c.MAX_FUTURE_DRIFT()));
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
        h = new AttestationsHandler(c, forwarder, workflowOwner);
        vm.startPrank(owner);
        c.setWorkflow(1, h.WF_SOLVENCY(), workflowOwner, h.NAME());
        c.setWorkflow(2, h.WF_DEPOSIT(), workflowOwner, h.NAME());
        c.setWorkflow(3, h.WF_CONVERSION(), workflowOwner, h.NAME());
        vm.stopPrank();
        targetContract(address(h));
    }

    function invariant_stateOnlyChangesThroughValidReports() public view {
        assertEq(h.invalidAccepted(), 0, "an invalid delivery was accepted");
        for (uint8 kind = 1; kind <= 3; ++kind) {
            assertEq(c.lastObservedAt(h.GATEWAY(), kind), h.ghostLastObservedAt(kind), "lastObservedAt drifted");
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

    function invariant_adminStateIsOwnerOnly() public view {
        assertEq(c.owner(), owner);
        assertEq(c.pendingOwner(), address(0));
        assertEq(c.forwarder(), forwarder);
        assertEq(c.getWorkflow(1).id, h.WF_SOLVENCY());
        assertEq(c.getWorkflow(2).id, h.WF_DEPOSIT());
        assertEq(c.getWorkflow(3).id, h.WF_CONVERSION());
        assertEq(address(c).balance, 0);
    }
}
