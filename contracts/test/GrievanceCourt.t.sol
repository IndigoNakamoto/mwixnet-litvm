// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {EvidenceLib} from "../src/EvidenceLib.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {GrievanceCourt} from "../src/GrievanceCourt.sol";

contract GrievanceCourtTest is Test {
    MwixnetRegistry internal registry;
    GrievanceCourt internal court;

    address internal accuser = address(0xA11);
    address internal accused = address(0xB22);
    address internal stranger = address(0xC33);

    uint256 internal constant MIN_STAKE = 1 ether;
    uint256 internal constant COOLDOWN = 48 hours;
    uint256 internal constant BOND = 0.1 ether;
    uint256 internal constant WINDOW = 1 days;

    function setUp() public {
        registry = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        court = new GrievanceCourt(registry, WINDOW, BOND);
        registry.setGrievanceCourt(address(court));

        vm.deal(accuser, 50 ether);
        vm.deal(accused, 50 ether);
        vm.deal(stranger, 50 ether);

        vm.prank(accused);
        registry.deposit{value: 5 ether}();
        vm.prank(accused);
        registry.registerMaker(bytes32(uint256(42)));
    }

    function test_open_freezes_resolve_slash_after_deadline() public {
        bytes32 evidenceHash = keccak256("evidence");
        uint256 epochId = 7;

        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epochId, evidenceHash);

        assertTrue(registry.stakeFrozen(accused));
        assertEq(court.openGrievanceCountAgainst(accused), 1);

        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);
        vm.warp(block.timestamp + WINDOW + 1);

        vm.prank(accuser);
        court.resolveGrievance(gid);

        (
            address accA_,
            address accB_,
            uint256 epochId_,
            bytes32 evidenceHash_,
            uint256 openedAt_,
            uint256 deadline_,
            GrievanceCourt.GrievancePhase phase,
            uint256 bondAmt_
        ) = court.grievances(gid);
        assertEq(accA_, accuser);
        assertEq(accB_, accused);
        assertEq(epochId_, epochId);
        assertEq(evidenceHash_, evidenceHash);
        assertEq(uint256(phase), uint256(GrievanceCourt.GrievancePhase.ResolvedSlash));
        assertEq(bondAmt_, BOND);
        assertLe(openedAt_, deadline_);
        assertFalse(registry.stakeFrozen(accused));
        assertEq(court.openGrievanceCountAgainst(accused), 0);
    }

    function test_defend_then_exonerate() public {
        bytes32 evidenceHash = keccak256("evidence2");
        uint256 epochId = 8;

        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epochId, evidenceHash);

        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);

        vm.prank(accused);
        court.defendGrievance(gid, hex"abcd");

        court.resolveGrievance(gid);

        (
            address accA_,
            address accB_,
            uint256 epochId_,
            bytes32 evidenceHash_,
            uint256 openedAt_,
            uint256 deadline_,
            GrievanceCourt.GrievancePhase phase,
            uint256 bondAmt_
        ) = court.grievances(gid);
        assertEq(accA_, accuser);
        assertEq(accB_, accused);
        assertEq(epochId_, 8);
        assertLe(openedAt_, deadline_);
        assertEq(uint256(phase), uint256(GrievanceCourt.GrievancePhase.ResolvedExonerate));
        assertEq(evidenceHash_, keccak256("evidence2"));
        assertEq(bondAmt_, BOND);
        assertFalse(registry.stakeFrozen(accused));
        assertEq(court.openGrievanceCountAgainst(accused), 0);
    }

    function test_resolve_reverts_before_deadline_if_open() public {
        bytes32 evidenceHash = keccak256("evidence3");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 9, evidenceHash);

        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(9), evidenceHash);

        vm.expectRevert(GrievanceCourt.TooEarly.selector);
        court.resolveGrievance(gid);
    }

    function test_requestWithdrawal_reverts_when_open_grievance() public {
        bytes32 evidenceHash = keccak256("evidence4");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 10, evidenceHash);

        vm.prank(accused);
        vm.expectRevert(MwixnetRegistry.OpenGrievanceBlocksExit.selector);
        registry.requestWithdrawal();
    }

    function test_withdrawStake_blocked_until_grievance_resolved() public {
        vm.prank(accused);
        registry.requestWithdrawal();
        vm.warp(block.timestamp + COOLDOWN);

        bytes32 evidenceHash = keccak256("late");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 11, evidenceHash);

        vm.prank(accused);
        vm.expectRevert(MwixnetRegistry.StakeFrozen_.selector);
        registry.withdrawStake();

        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(11), evidenceHash);
        vm.warp(block.timestamp + WINDOW + 1);
        vm.prank(accuser);
        court.resolveGrievance(gid);

        vm.prank(accused);
        registry.withdrawStake();
        assertEq(registry.stake(accused), 0);
    }

    function test_multipleGrievances_conditionalUnfreeze() public {
        uint256 epoch1 = 100;
        uint256 epoch2 = 101;
        bytes32 hash1 = keccak256("evidence1");
        bytes32 hash2 = keccak256("evidence2");

        bytes32 gId1 = EvidenceLib.grievanceId(accuser, accused, epoch1, hash1);
        bytes32 gId2 = EvidenceLib.grievanceId(accuser, accused, epoch2, hash2);

        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epoch1, hash1);
        assertTrue(registry.stakeFrozen(accused));
        assertEq(court.openGrievanceCountAgainst(accused), 1);

        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epoch2, hash2);
        assertEq(court.openGrievanceCountAgainst(accused), 2);

        vm.warp(block.timestamp + WINDOW + 1);

        vm.prank(accuser);
        court.resolveGrievance(gId1);

        assertEq(court.openGrievanceCountAgainst(accused), 1);
        assertTrue(registry.stakeFrozen(accused));

        vm.prank(accuser);
        court.resolveGrievance(gId2);

        assertEq(court.openGrievanceCountAgainst(accused), 0);
        assertFalse(registry.stakeFrozen(accused));
    }

    function test_open_reverts_insufficientBond() public {
        vm.prank(accuser);
        vm.expectRevert(GrievanceCourt.InsufficientBond.selector);
        court.openGrievance{value: BOND - 1}(accused, 200, keccak256("x"));
    }

    function test_defend_reverts_notAccused() public {
        bytes32 h = keccak256("na");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 201, h);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(201), h);

        vm.prank(stranger);
        vm.expectRevert(GrievanceCourt.NotAccused.selector);
        court.defendGrievance(gid, hex"");
    }

    function test_defend_reverts_badPhase_after_defend() public {
        bytes32 h = keccak256("bd");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 202, h);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(202), h);

        vm.prank(accused);
        court.defendGrievance(gid, hex"aa");

        vm.prank(accused);
        vm.expectRevert(GrievanceCourt.BadPhase.selector);
        court.defendGrievance(gid, hex"bb");
    }

    function test_resolve_reverts_badPhase_unknownGrievance() public {
        vm.expectRevert(GrievanceCourt.BadPhase.selector);
        court.resolveGrievance(bytes32(uint256(0xdead)));
    }

    function test_open_reverts_alreadyExists() public {
        bytes32 h = keccak256("dup");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 203, h);

        vm.prank(accuser);
        vm.expectRevert(GrievanceCourt.AlreadyExists.selector);
        court.openGrievance{value: BOND}(accused, 203, h);
    }

    function test_resolve_reverts_badPhase_after_resolved() public {
        bytes32 h = keccak256("rs");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 204, h);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(204), h);

        vm.warp(block.timestamp + WINDOW + 1);
        vm.prank(accuser);
        court.resolveGrievance(gid);

        vm.expectRevert(GrievanceCourt.BadPhase.selector);
        court.resolveGrievance(gid);
    }
}
