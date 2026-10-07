// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IERC165} from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import {GatewayAttestations} from "../../src/cre/GatewayAttestations.sol";
import {IReceiver} from "../../src/cre/IReceiver.sol";
import {ReportEncoder} from "./ReportEncoder.sol";
import {WorkflowName} from "../../src/cre/WorkflowName.sol";

contract GatewayAttestationsTest is Test {
    GatewayAttestations internal c;

    address internal forwarder = makeAddr("forwarder");
    address internal owner = makeAddr("owner");
    address internal workflowOwner = makeAddr("workflowOwner");
    address internal stranger = makeAddr("stranger");

    bytes32 internal constant GATEWAY = keccak256("https://pay.example.com");
    bytes32 internal constant WF_SOLVENCY = keccak256("wf-solvency");
    bytes32 internal constant WF_DEPOSIT = keccak256("wf-deposit");
    bytes32 internal constant WF_CONVERSION = keccak256("wf-conversion");
    // Keystone's truncated name hash (WorkflowName), computed once in setUp so no sha256 precompile call sits
    // between a cheatcode and the call it targets.
    bytes10 internal NAME_SOLVENCY;
    bytes10 internal NAME_DEPOSIT;
    bytes10 internal NAME_CONVERSION;
    bytes2 internal constant REPORT_ID = 0x0001;

    uint64 internal constant T0 = 1_800_000_000;

    function setUp() public {
        NAME_SOLVENCY = WorkflowName.keystone("solvency");
        NAME_DEPOSIT = WorkflowName.keystone("deposit-finality");
        NAME_CONVERSION = WorkflowName.keystone("conversion-reference");
        vm.warp(T0);
        c = new GatewayAttestations(forwarder, owner);
        vm.startPrank(owner);
        c.setWorkflow(c.KIND_SOLVENCY(), WF_SOLVENCY, workflowOwner, NAME_SOLVENCY);
        c.setWorkflow(c.KIND_DEPOSIT_FINALITY(), WF_DEPOSIT, workflowOwner, NAME_DEPOSIT);
        c.setWorkflow(c.KIND_CONVERSION_REFERENCE(), WF_CONVERSION, workflowOwner, NAME_CONVERSION);
        vm.stopPrank();
    }

    // ---- helpers -------------------------------------------------------------------------------------------

    function meta(bytes32 workflowId, bytes10 name) internal view returns (bytes memory) {
        return ReportEncoder.metadata(workflowId, name, workflowOwner, REPORT_ID);
    }

    function deliver(bytes memory metadata, bytes memory report) internal {
        vm.prank(forwarder);
        c.onReport(metadata, report);
    }

    function solvencyReport(uint64 observedAt, uint256 n) internal pure returns (bytes memory) {
        return ReportEncoder.solvency(GATEWAY, observedAt, ReportEncoder.solvencyBatch(n, keccak256("ckpt")));
    }

    // ---- construction and admin ----------------------------------------------------------------------------

    function test_constructor_setsForwarderAndOwner() public view {
        assertEq(c.forwarder(), forwarder);
        assertEq(c.owner(), owner);
        assertEq(c.pendingOwner(), address(0));
    }

    function test_constructor_rejectsZeroForwarder() public {
        vm.expectRevert(GatewayAttestations.ZeroAddress.selector);
        new GatewayAttestations(address(0), owner);
    }

    function test_supportsInterface() public view {
        assertTrue(c.supportsInterface(type(IReceiver).interfaceId));
        assertTrue(c.supportsInterface(type(IERC165).interfaceId));
        assertFalse(c.supportsInterface(0xffffffff));
    }

    function test_setForwarder_onlyOwner() public {
        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, stranger));
        c.setForwarder(stranger);

        address next = makeAddr("forwarder2");
        vm.prank(owner);
        vm.expectEmit(true, true, false, true);
        emit GatewayAttestations.ForwarderSet(forwarder, next);
        c.setForwarder(next);
        assertEq(c.forwarder(), next);

        vm.prank(owner);
        vm.expectRevert(GatewayAttestations.ZeroAddress.selector);
        c.setForwarder(address(0));
    }

    function test_setWorkflow_guards() public {
        vm.startPrank(owner);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnknownKind.selector, 0));
        c.setWorkflow(0, WF_SOLVENCY, workflowOwner, NAME_SOLVENCY);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnknownKind.selector, 4));
        c.setWorkflow(4, WF_SOLVENCY, workflowOwner, NAME_SOLVENCY);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.WorkflowNotBound.selector, 1));
        c.setWorkflow(1, bytes32(0), workflowOwner, NAME_SOLVENCY);
        vm.expectRevert(GatewayAttestations.ZeroAddress.selector);
        c.setWorkflow(1, WF_SOLVENCY, address(0), NAME_SOLVENCY);
        vm.expectRevert(GatewayAttestations.ZeroWorkflowName.selector);
        c.setWorkflow(1, WF_SOLVENCY, workflowOwner, bytes10(0));
        vm.stopPrank();

        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, stranger));
        c.setWorkflow(1, WF_SOLVENCY, workflowOwner, NAME_SOLVENCY);
    }

    function test_unbindWorkflow() public {
        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, stranger));
        c.unbindWorkflow(1);

        vm.prank(owner);
        vm.expectEmit(true, false, false, true);
        emit GatewayAttestations.WorkflowUnbound(1);
        c.unbindWorkflow(1);
        assertEq(c.getWorkflow(1).id, bytes32(0));

        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.WorkflowNotBound.selector, 1));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), solvencyReport(T0, 1));
    }

    function test_ownershipTransfer_isTwoStep() public {
        address next = makeAddr("nextOwner");

        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, stranger));
        c.transferOwnership(next);

        vm.prank(owner);
        c.transferOwnership(next);
        assertEq(c.owner(), owner, "owner unchanged until accepted");
        assertEq(c.pendingOwner(), next);

        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, stranger));
        c.acceptOwnership();

        vm.prank(next);
        c.acceptOwnership();
        assertEq(c.owner(), next);
        assertEq(c.pendingOwner(), address(0));

        vm.prank(owner);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, owner));
        c.setForwarder(stranger);
    }

    function test_renounceOwnership_reverts() public {
        vm.prank(owner);
        vm.expectRevert(GatewayAttestations.RenounceDisabled.selector);
        c.renounceOwnership();
        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, stranger));
        c.renounceOwnership();
        assertEq(c.owner(), owner);
    }

    function test_docsNameBytes_acceptedThroughOnReport() public {
        // The literal bytes from the Chainlink docs example ("my_workflow"), bound and delivered as-is.
        bytes10 docsName = bytes10(0x62373666336165316465);
        assertEq(WorkflowName.keystone("my_workflow"), docsName);
        vm.prank(owner);
        c.setWorkflow(1, WF_SOLVENCY, workflowOwner, docsName);
        deliver(ReportEncoder.metadata(WF_SOLVENCY, docsName, workflowOwner, REPORT_ID), solvencyReport(T0, 1));
        assertEq(c.latestObservedAt(GATEWAY, 1), T0);
    }

    // ---- forwarder and workflow binding guards -----------------------------------------------------------

    function test_onReport_rejectsNonForwarder() public {
        bytes memory report = solvencyReport(T0, 1);
        vm.prank(stranger);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnauthorizedForwarder.selector, stranger));
        c.onReport(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        vm.prank(owner);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnauthorizedForwarder.selector, owner));
        c.onReport(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsWrongWorkflowId() public {
        vm.expectRevert(
            abi.encodeWithSelector(
                GatewayAttestations.UnexpectedWorkflow.selector, 1, WF_DEPOSIT, workflowOwner, NAME_SOLVENCY
            )
        );
        deliver(meta(WF_DEPOSIT, NAME_SOLVENCY), solvencyReport(T0, 1));
    }

    function test_onReport_rejectsWrongWorkflowOwner() public {
        bytes memory m = ReportEncoder.metadata(WF_SOLVENCY, NAME_SOLVENCY, stranger, REPORT_ID);
        vm.expectRevert(
            abi.encodeWithSelector(
                GatewayAttestations.UnexpectedWorkflow.selector, 1, WF_SOLVENCY, stranger, NAME_SOLVENCY
            )
        );
        deliver(m, solvencyReport(T0, 1));
    }

    function test_onReport_rejectsWrongWorkflowName() public {
        vm.expectRevert(
            abi.encodeWithSelector(
                GatewayAttestations.UnexpectedWorkflow.selector, 1, WF_SOLVENCY, workflowOwner, NAME_DEPOSIT
            )
        );
        deliver(meta(WF_SOLVENCY, NAME_DEPOSIT), solvencyReport(T0, 1));
    }

    function test_onReport_rejectsKindWorkflowCrossing() public {
        // A valid deposit workflow identity cannot deliver a solvency-kind report.
        vm.expectRevert(
            abi.encodeWithSelector(
                GatewayAttestations.UnexpectedWorkflow.selector, 1, WF_DEPOSIT, workflowOwner, NAME_DEPOSIT
            )
        );
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), solvencyReport(T0, 1));
    }

    function test_onReport_rejectsMetadataLength() public {
        bytes memory report = solvencyReport(T0, 1);
        bytes memory short = abi.encodePacked(WF_SOLVENCY, NAME_SOLVENCY, workflowOwner);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.InvalidMetadataLength.selector, 62));
        deliver(short, report);

        bytes memory long = abi.encodePacked(meta(WF_SOLVENCY, NAME_SOLVENCY), uint8(0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.InvalidMetadataLength.selector, 65));
        deliver(long, report);

        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.InvalidMetadataLength.selector, 0));
        deliver("", report);
    }

    // ---- report shape guards ---------------------------------------------------------------------------

    function test_onReport_rejectsWrongVersion() public {
        bytes memory report =
            abi.encode(uint8(2), c.KIND_SOLVENCY(), GATEWAY, T0, ReportEncoder.solvencyBatch(1, keccak256("ckpt")));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnsupportedReportVersion.selector, 2));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        report = abi.encode(uint8(0), c.KIND_SOLVENCY(), GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnsupportedReportVersion.selector, 0));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsUnknownKind() public {
        bytes memory report = abi.encode(uint8(1), uint8(0), GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnknownKind.selector, 0));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        report = abi.encode(uint8(1), uint8(4), GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnknownKind.selector, 4));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        // a kind word that does not fit a uint8 is malformed, never truncated to a valid kind
        report = abi.encode(uint8(1), uint256(257), GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, report.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsEmptyBatch() public {
        bytes memory report = solvencyReport(T0, 0);
        vm.expectRevert(GatewayAttestations.EmptyReport.selector);
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsTruncatedAndPaddedReports() public {
        bytes memory good = solvencyReport(T0, 2);

        bytes memory truncated = new bytes(good.length - 32);
        for (uint256 i = 0; i < truncated.length; ++i) {
            truncated[i] = good[i];
        }
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, truncated.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), truncated);

        bytes memory padded = abi.encodePacked(good, uint256(0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, padded.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), padded);

        bytes memory tiny = new bytes(191);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, 191));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), tiny);

        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, 0));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), "");
    }

    function test_onReport_rejectsItemCountMismatch() public {
        // Deposit-sized body under a solvency header: the byte length no longer matches the declared count.
        bytes memory report = abi.encode(uint8(1), uint8(1), GATEWAY, T0, ReportEncoder.depositBatch(2));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, report.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsNonCanonicalOffset() public {
        bytes memory good = solvencyReport(T0, 1);
        // Move the items pointer one word later and add a word so the length still matches a 1-item batch.
        bytes memory report = abi.encodePacked(
            bytes32(uint256(1)),
            bytes32(uint256(1)),
            GATEWAY,
            bytes32(uint256(T0)),
            bytes32(uint256(192)),
            bytes32(uint256(0)),
            bytes32(uint256(1)),
            bytes32(uint256(0)),
            bytes32(uint256(0)),
            bytes32(uint256(0)),
            bytes32(uint256(0))
        );
        assertEq(report.length, good.length, "test setup: same length as a canonical 1-item report");
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, report.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsObservedAtOverflow() public {
        bytes memory report =
            abi.encode(uint8(1), uint8(1), GATEWAY, uint256(type(uint64).max) + 1, ReportEncoder.solvencyBatch(1, 0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, report.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsAbsurdItemCountWithNamedError() public {
        bytes memory report = abi.encodePacked(
            bytes32(uint256(1)),
            bytes32(uint256(1)),
            GATEWAY,
            bytes32(uint256(T0)),
            bytes32(uint256(160)),
            bytes32(uint256(1) << 251),
            new bytes(5 * 32)
        );
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, report.length));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsOutOfRangeFieldInsideItem() public {
        // decimals word > uint8: abi.decode's own validation rejects it.
        bytes memory report = abi.encodePacked(
            bytes32(uint256(1)),
            bytes32(uint256(1)),
            GATEWAY,
            bytes32(uint256(T0)),
            bytes32(uint256(160)),
            bytes32(uint256(1)),
            keccak256("ckpt"),
            keccak256("asset"),
            bytes32(uint256(1)),
            bytes32(uint256(1)),
            bytes32(uint256(256))
        );
        vm.expectRevert();
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function test_onReport_rejectsInvalidVerdict() public {
        GatewayAttestations.DepositItem[] memory items = ReportEncoder.depositBatch(1);
        items[0].verdict = 0;
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.InvalidVerdict.selector, 0));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), ReportEncoder.deposits(GATEWAY, T0, items));

        items[0].verdict = 4;
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.InvalidVerdict.selector, 4));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), ReportEncoder.deposits(GATEWAY, T0, items));
    }

    // ---- time and replay -----------------------------------------------------------------------------

    function test_onReport_rejectsFutureObservedAt() public {
        uint64 tooFar = T0 + c.MAX_FUTURE_DRIFT() + 1;
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.ObservedAtInFuture.selector, tooFar, T0));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), solvencyReport(tooFar, 1));

        // exactly at the drift bound is accepted
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), solvencyReport(T0 + c.MAX_FUTURE_DRIFT(), 1));
    }

    function test_onReport_rejectsReplay_everyKind() public {
        bytes memory report = solvencyReport(T0, 2);
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.DuplicateReport.selector, keccak256(report)));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        report = ReportEncoder.deposits(GATEWAY, T0, ReportEncoder.depositBatch(2));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), report);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.DuplicateReport.selector, keccak256(report)));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), report);

        report = ReportEncoder.conversions(GATEWAY, T0, ReportEncoder.conversionBatch(2));
        deliver(meta(WF_CONVERSION, NAME_CONVERSION), report);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.DuplicateReport.selector, keccak256(report)));
        deliver(meta(WF_CONVERSION, NAME_CONVERSION), report);
        assertTrue(c.reportSeen(keccak256(report)));
    }

    function test_onReport_replayRejectedEvenWithDifferentMetadata() public {
        bytes memory report = ReportEncoder.deposits(GATEWAY, T0, ReportEncoder.depositBatch(1));
        deliver(ReportEncoder.metadata(WF_DEPOSIT, NAME_DEPOSIT, workflowOwner, 0x0001), report);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.DuplicateReport.selector, keccak256(report)));
        deliver(ReportEncoder.metadata(WF_DEPOSIT, NAME_DEPOSIT, workflowOwner, 0x0002), report);
    }

    function test_eventOnlyKinds_acceptOutOfOrderGenuineReports() public {
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), ReportEncoder.deposits(GATEWAY, T0, ReportEncoder.depositBatch(2)));
        GatewayAttestations.DepositItem[] memory late = ReportEncoder.depositBatch(1);
        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.DepositAttested(
            GATEWAY,
            late[0].depositId,
            late[0].verdict,
            late[0].chainId,
            late[0].txRef,
            late[0].token,
            late[0].amount,
            late[0].destination,
            late[0].slotOrBlock,
            T0 - 60
        );
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), ReportEncoder.deposits(GATEWAY, T0 - 60, late));
        assertEq(c.latestObservedAt(GATEWAY, 2), T0, "latest is the max, not the last delivered");

        deliver(
            meta(WF_CONVERSION, NAME_CONVERSION),
            ReportEncoder.conversions(GATEWAY, T0, ReportEncoder.conversionBatch(1))
        );
        deliver(
            meta(WF_CONVERSION, NAME_CONVERSION),
            ReportEncoder.conversions(GATEWAY, T0 - 900, ReportEncoder.conversionBatch(3))
        );
        assertEq(c.latestObservedAt(GATEWAY, 3), T0);
    }

    function test_solvency_olderSnapshotIsIgnoredPerAsset() public {
        GatewayAttestations.SolvencyItem[] memory items = ReportEncoder.solvencyBatch(2, keccak256("new"));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), ReportEncoder.solvency(GATEWAY, T0, items));

        // A late report: asset 0 older (ignored), asset 2 unseen (stored), asset 1 equal time (ignored).
        GatewayAttestations.SolvencyItem[] memory late = ReportEncoder.solvencyBatch(3, keccak256("old"));
        late[0].liabilities = 1;
        late[1].liabilities = 2;
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0 - 3600, late);
        // report-level: asset 1 is only at T0, so with T0 - 3600 both known assets are ignored
        vm.expectEmit(true, true, false, true, address(c));
        emit GatewayAttestations.SolvencyIgnored(GATEWAY, late[0].asset, T0 - 3600, T0);
        vm.expectEmit(true, true, false, true, address(c));
        emit GatewayAttestations.SolvencyIgnored(GATEWAY, late[1].asset, T0 - 3600, T0);
        vm.expectEmit(true, true, false, true, address(c));
        emit GatewayAttestations.SolvencyAttested(
            GATEWAY, late[2].asset, late[2].checkpointHash, late[2].liabilities, late[2].reserves, 6, T0 - 3600
        );
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        assertEq(c.getLatestSolvency(GATEWAY, items[0].asset).liabilities, items[0].liabilities, "not overwritten");
        assertEq(c.getLatestSolvency(GATEWAY, items[0].asset).observedAt, T0);
        assertEq(c.getLatestSolvency(GATEWAY, items[1].asset).liabilities, items[1].liabilities, "not overwritten");
        assertEq(c.getLatestSolvency(GATEWAY, late[2].asset).observedAt, T0 - 3600, "new asset stored");
        assertEq(c.latestObservedAt(GATEWAY, 1), T0);
    }

    function test_latestObservedAt_isPerGatewayAndKind() public {
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), solvencyReport(T0, 1));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), ReportEncoder.deposits(GATEWAY, T0 - 1, ReportEncoder.depositBatch(1)));
        bytes32 other = keccak256("https://other.example.com");
        deliver(
            meta(WF_SOLVENCY, NAME_SOLVENCY),
            ReportEncoder.solvency(other, T0 - 2, ReportEncoder.solvencyBatch(1, keccak256("ckpt")))
        );
        assertEq(c.latestObservedAt(GATEWAY, 1), T0);
        assertEq(c.latestObservedAt(GATEWAY, 2), T0 - 1);
        assertEq(c.latestObservedAt(other, 1), T0 - 2);
        assertEq(c.latestObservedAt(other, 2), 0);
    }

    function test_solvency_duplicateAssetWithinBatchKeepsFirst() public {
        GatewayAttestations.SolvencyItem[] memory items = ReportEncoder.solvencyBatch(2, keccak256("ckpt"));
        items[1].asset = items[0].asset;
        items[1].liabilities = 999;
        vm.expectEmit(true, true, false, true, address(c));
        emit GatewayAttestations.SolvencyAttested(
            GATEWAY, items[0].asset, items[0].checkpointHash, items[0].liabilities, items[0].reserves, 6, T0
        );
        vm.expectEmit(true, true, false, true, address(c));
        emit GatewayAttestations.SolvencyIgnored(GATEWAY, items[0].asset, T0, T0);
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), ReportEncoder.solvency(GATEWAY, T0, items));
        assertEq(c.getLatestSolvency(GATEWAY, items[0].asset).liabilities, items[0].liabilities);
    }

    // ---- happy paths and round trips -----------------------------------------------------------------

    function test_solvency_roundTrip() public {
        GatewayAttestations.SolvencyItem[] memory items = ReportEncoder.solvencyBatch(3, keccak256("ckpt"));
        bytes memory report = ReportEncoder.solvency(GATEWAY, T0, items);

        for (uint256 i = 0; i < items.length; ++i) {
            vm.expectEmit(true, true, false, true, address(c));
            emit GatewayAttestations.SolvencyAttested(
                GATEWAY, items[i].asset, items[i].checkpointHash, items[i].liabilities, items[i].reserves, 6, T0
            );
        }
        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.ReportAccepted(
            GATEWAY, 1, WF_SOLVENCY, workflowOwner, NAME_SOLVENCY, REPORT_ID, T0, 3, keccak256(report)
        );
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);

        for (uint256 i = 0; i < items.length; ++i) {
            GatewayAttestations.Solvency memory s = c.getLatestSolvency(GATEWAY, items[i].asset);
            assertEq(s.checkpointHash, items[i].checkpointHash);
            assertEq(s.liabilities, items[i].liabilities);
            assertEq(s.reserves, items[i].reserves);
            assertEq(s.decimals, items[i].decimals);
            assertEq(s.observedAt, T0);
        }
        assertEq(c.latestObservedAt(GATEWAY, 1), T0);
    }

    function test_solvency_laterReportReplacesLatest() public {
        GatewayAttestations.SolvencyItem[] memory items = ReportEncoder.solvencyBatch(1, keccak256("a"));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), ReportEncoder.solvency(GATEWAY, T0, items));
        items[0].liabilities = 5;
        items[0].reserves = 7;
        items[0].checkpointHash = keccak256("b");
        vm.warp(T0 + 3600);
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), ReportEncoder.solvency(GATEWAY, T0 + 3600, items));
        GatewayAttestations.Solvency memory s = c.getLatestSolvency(GATEWAY, items[0].asset);
        assertEq(s.liabilities, 5);
        assertEq(s.reserves, 7);
        assertEq(s.checkpointHash, keccak256("b"));
        assertEq(s.observedAt, T0 + 3600);
    }

    function test_deposit_roundTrip() public {
        GatewayAttestations.DepositItem[] memory items = ReportEncoder.depositBatch(3);
        bytes memory report = ReportEncoder.deposits(GATEWAY, T0, items);
        for (uint256 i = 0; i < items.length; ++i) {
            vm.expectEmit(true, true, true, true, address(c));
            emit GatewayAttestations.DepositAttested(
                GATEWAY,
                items[i].depositId,
                items[i].verdict,
                items[i].chainId,
                items[i].txRef,
                items[i].token,
                items[i].amount,
                items[i].destination,
                items[i].slotOrBlock,
                T0
            );
        }
        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.ReportAccepted(
            GATEWAY, 2, WF_DEPOSIT, workflowOwner, NAME_DEPOSIT, REPORT_ID, T0, 3, keccak256(report)
        );
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), report);
        assertEq(c.latestObservedAt(GATEWAY, 2), T0);
    }

    function test_conversion_roundTrip() public {
        GatewayAttestations.ConversionItem[] memory items = ReportEncoder.conversionBatch(2);
        bytes memory report = ReportEncoder.conversions(GATEWAY, T0, items);
        for (uint256 i = 0; i < items.length; ++i) {
            vm.expectEmit(true, true, true, true, address(c));
            emit GatewayAttestations.ConversionReferenceAttested(
                GATEWAY,
                items[i].conversionId,
                items[i].pair,
                items[i].referenceRate,
                items[i].referenceDecimals,
                items[i].deviationBps,
                items[i].feed,
                items[i].roundId,
                T0
            );
        }
        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.ReportAccepted(
            GATEWAY, 3, WF_CONVERSION, workflowOwner, NAME_CONVERSION, REPORT_ID, T0, 2, keccak256(report)
        );
        deliver(meta(WF_CONVERSION, NAME_CONVERSION), report);
        assertEq(c.latestObservedAt(GATEWAY, 3), T0);
    }

    function test_rejectedReport_leavesNoState() public {
        bytes memory report = solvencyReport(T0, 2);
        vm.prank(stranger);
        vm.expectRevert();
        c.onReport(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
        assertEq(c.latestObservedAt(GATEWAY, 1), 0);
        assertFalse(c.reportSeen(keccak256(report)));
        assertEq(c.getLatestSolvency(GATEWAY, keccak256(abi.encodePacked("asset", uint256(0)))).observedAt, 0);
    }

    function test_contract_holdsNoValue() public {
        (bool ok,) = address(c).call{value: 1 ether}("");
        assertFalse(ok, "no receive or fallback");
        assertEq(address(c).balance, 0);
    }

    // ---- fuzz --------------------------------------------------------------------------------------------

    function testFuzz_solvency_decode(
        bytes32 gatewayId,
        uint64 observedAt,
        bytes32 checkpointHash,
        bytes32 asset,
        uint256 liabilities,
        uint256 reserves,
        uint8 decimals
    ) public {
        observedAt = uint64(bound(observedAt, 1, T0 + c.MAX_FUTURE_DRIFT()));
        GatewayAttestations.SolvencyItem[] memory items = new GatewayAttestations.SolvencyItem[](1);
        items[0] = GatewayAttestations.SolvencyItem(checkpointHash, asset, liabilities, reserves, decimals);
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), ReportEncoder.solvency(gatewayId, observedAt, items));
        GatewayAttestations.Solvency memory s = c.getLatestSolvency(gatewayId, asset);
        assertEq(s.checkpointHash, checkpointHash);
        assertEq(s.liabilities, liabilities);
        assertEq(s.reserves, reserves);
        assertEq(s.decimals, decimals);
        assertEq(s.observedAt, observedAt);
        assertEq(c.latestObservedAt(gatewayId, 1), observedAt);
    }

    function testFuzz_deposit_decode(GatewayAttestations.DepositItem memory item, uint64 observedAt) public {
        observedAt = uint64(bound(observedAt, 1, T0 + c.MAX_FUTURE_DRIFT()));
        item.verdict = uint8(bound(item.verdict, 1, 3));
        GatewayAttestations.DepositItem[] memory items = new GatewayAttestations.DepositItem[](1);
        items[0] = item;
        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.DepositAttested(
            GATEWAY,
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
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), ReportEncoder.deposits(GATEWAY, observedAt, items));
    }

    function testFuzz_conversion_decode(GatewayAttestations.ConversionItem memory item, uint64 observedAt) public {
        observedAt = uint64(bound(observedAt, 1, T0 + c.MAX_FUTURE_DRIFT()));
        GatewayAttestations.ConversionItem[] memory items = new GatewayAttestations.ConversionItem[](1);
        items[0] = item;
        vm.expectEmit(true, true, true, true, address(c));
        emit GatewayAttestations.ConversionReferenceAttested(
            GATEWAY,
            item.conversionId,
            item.pair,
            item.referenceRate,
            item.referenceDecimals,
            item.deviationBps,
            item.feed,
            item.roundId,
            observedAt
        );
        deliver(meta(WF_CONVERSION, NAME_CONVERSION), ReportEncoder.conversions(GATEWAY, observedAt, items));
    }

    function testFuzz_batchLength_mustMatchCount(uint8 n, uint8 extraWords) public {
        n = uint8(bound(n, 1, 24));
        extraWords = uint8(bound(extraWords, 1, 8));
        bytes memory good = ReportEncoder.deposits(GATEWAY, T0, ReportEncoder.depositBatch(n));
        bytes memory padded = abi.encodePacked(good, new bytes(32 * uint256(extraWords)));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, padded.length));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), padded);

        bytes memory truncated = new bytes(good.length - 32 * uint256(bound(extraWords, 1, 8)));
        for (uint256 i = 0; i < truncated.length; ++i) {
            truncated[i] = good[i];
        }
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.MalformedReport.selector, truncated.length));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), truncated);
    }

    function testFuzz_randomBytes_neverAccepted(bytes calldata junk) public {
        vm.assume(junk.length < 192 || junk.length % 32 != 0);
        vm.expectRevert();
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), junk);
        assertEq(c.latestObservedAt(GATEWAY, 1), 0);
    }

    function testFuzz_wrongVersion_neverAccepted(uint8 version) public {
        vm.assume(version != c.REPORT_VERSION());
        bytes memory report = abi.encode(version, uint8(1), GATEWAY, T0, ReportEncoder.solvencyBatch(1, 0));
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnsupportedReportVersion.selector, version));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), report);
    }

    function testFuzz_wrongForwarder_neverAccepted(address caller) public {
        vm.assume(caller != forwarder);
        vm.prank(caller);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.UnauthorizedForwarder.selector, caller));
        c.onReport(meta(WF_SOLVENCY, NAME_SOLVENCY), solvencyReport(T0, 1));
    }

    function testFuzz_solvency_newestSnapshotWins(uint64 a, uint64 b) public {
        a = uint64(bound(a, 1, T0));
        b = uint64(bound(b, 1, T0));
        bytes memory first = ReportEncoder.solvency(GATEWAY, a, ReportEncoder.solvencyBatch(1, keccak256("a")));
        bytes memory second = ReportEncoder.solvency(GATEWAY, b, ReportEncoder.solvencyBatch(1, keccak256("b")));
        bytes32 asset = keccak256(abi.encodePacked("asset", uint256(0)));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), first);
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), second);
        GatewayAttestations.Solvency memory s = c.getLatestSolvency(GATEWAY, asset);
        assertEq(s.observedAt, a > b ? a : b);
        assertEq(s.checkpointHash, b > a ? keccak256("b") : keccak256("a"));
        assertEq(c.latestObservedAt(GATEWAY, 1), a > b ? a : b);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.DuplicateReport.selector, keccak256(first)));
        deliver(meta(WF_SOLVENCY, NAME_SOLVENCY), first);
    }

    function testFuzz_eventOnly_anyOrderOnceEach(uint64 a, uint64 b) public {
        a = uint64(bound(a, 1, T0));
        b = uint64(bound(b, 1, T0));
        vm.assume(a != b);
        bytes memory first = ReportEncoder.deposits(GATEWAY, a, ReportEncoder.depositBatch(1));
        bytes memory second = ReportEncoder.deposits(GATEWAY, b, ReportEncoder.depositBatch(1));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), first);
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), second);
        assertEq(c.latestObservedAt(GATEWAY, 2), a > b ? a : b);
        vm.expectRevert(abi.encodeWithSelector(GatewayAttestations.DuplicateReport.selector, keccak256(second)));
        deliver(meta(WF_DEPOSIT, NAME_DEPOSIT), second);
    }
}
