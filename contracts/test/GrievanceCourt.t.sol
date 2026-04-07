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
    /// @dev Interim judge for adjudicateGrievance (Contested phase).
    address internal judge = address(0xD00);

    uint256 internal constant MIN_STAKE = 1 ether;
    uint256 internal constant COOLDOWN = 48 hours;
    uint256 internal constant BOND = 0.1 ether;
    uint256 internal constant WINDOW = 1 days;
    uint256 internal constant SLASH_BPS = 10_000;
    uint256 internal constant BOUNTY_BPS = 1000;
    uint256 internal constant BURN_BPS = 9000;
    uint256 internal constant SLASHING_WINDOW = 7 days;

    function setUp() public {
        registry = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        court = new GrievanceCourt(
            registry, WINDOW, BOND, SLASH_BPS, BOUNTY_BPS, BURN_BPS, SLASHING_WINDOW, judge
        );
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

        uint256 accuserBefore = accuser.balance;
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
        assertEq(registry.stake(accused), 0);
        assertEq(registry.makerNostrKeyHash(accused), bytes32(0));
        uint256 bounty = (5 ether * BOUNTY_BPS) / 10_000;
        assertEq(accuser.balance, accuserBefore + BOND + bounty);
        assertEq(address(0).balance, (5 ether * BURN_BPS) / 10_000);
    }

    function test_defend_then_judge_exonerate() public {
        bytes32 evidenceHash = keccak256("evidence2");
        uint256 epochId = 8;

        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epochId, evidenceHash);

        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);

        vm.prank(accused);
        court.defendGrievance(gid, hex"abcd");

        vm.expectRevert(GrievanceCourt.BadPhase.selector);
        court.resolveGrievance(gid);

        uint256 accusedBefore = accused.balance;
        vm.prank(judge);
        court.adjudicateGrievance(gid, true);

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
        assertEq(accused.balance, accusedBefore + BOND);
    }

    function test_contested_permissionless_resolve_reverts_bond_unmoved() public {
        bytes32 evidenceHash = keccak256("garbage_defense");
        uint256 epochId = 88;
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epochId, evidenceHash);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);

        vm.prank(accused);
        court.defendGrievance(gid, hex"deadbeef");

        vm.warp(block.timestamp + WINDOW + 1);
        vm.expectRevert(GrievanceCourt.BadPhase.selector);
        court.resolveGrievance(gid);

        assertEq(court.openGrievanceCountAgainst(accused), 1);
        assertTrue(registry.stakeFrozen(accused));
        assertEq(address(court).balance, BOND);
    }

    function test_judge_adjudicate_uphold_slash_matches_timeout() public {
        bytes32 evidenceHash = keccak256("evidence_uphold");
        uint256 epochId = 89;
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epochId, evidenceHash);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);

        vm.prank(accused);
        court.defendGrievance(gid, hex"01");

        uint256 accuserBefore = accuser.balance;
        vm.prank(judge);
        court.adjudicateGrievance(gid, false);

        (,,,,,, GrievanceCourt.GrievancePhase ph,) = court.grievances(gid);
        assertEq(uint256(ph), uint256(GrievanceCourt.GrievancePhase.ResolvedSlash));
        assertEq(registry.stake(accused), 0);
        uint256 bounty = (5 ether * BOUNTY_BPS) / 10_000;
        assertEq(accuser.balance, accuserBefore + BOND + bounty);
        assertEq(court.openGrievanceCountAgainst(accused), 0);
        assertFalse(registry.stakeFrozen(accused));
    }

    function test_adjudicate_revert_not_judge() public {
        bytes32 h = keccak256("adj");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 300, h);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(300), h);
        vm.prank(accused);
        court.defendGrievance(gid, hex"");

        vm.prank(stranger);
        vm.expectRevert(GrievanceCourt.NotJudge.selector);
        court.adjudicateGrievance(gid, true);
    }

    function test_adjudicate_revert_wrong_phase_open() public {
        bytes32 h = keccak256("open_phase");
        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, 301, h);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(301), h);

        vm.prank(judge);
        vm.expectRevert(GrievanceCourt.BadPhase.selector);
        court.adjudicateGrievance(gid, false);
    }

    function test_constructor_revert_zero_judge() public {
        MwixnetRegistry r = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        vm.expectRevert(GrievanceCourt.ZeroJudge.selector);
        new GrievanceCourt(
            r, WINDOW, BOND, SLASH_BPS, BOUNTY_BPS, BURN_BPS, SLASHING_WINDOW, address(0)
        );
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

    /// @dev Uses partial `slashBps` so stake remains after resolution; full slash would clear `exitUnlockTime` via auto-deregister.
    function test_withdrawStake_blocked_until_grievance_resolved() public {
        MwixnetRegistry r = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        GrievanceCourt c =
            new GrievanceCourt(r, WINDOW, BOND, 2500, BOUNTY_BPS, BURN_BPS, SLASHING_WINDOW, judge);
        r.setGrievanceCourt(address(c));

        vm.deal(accuser, 50 ether);
        vm.deal(accused, 50 ether);
        vm.prank(accused);
        r.deposit{value: 5 ether}();
        vm.prank(accused);
        r.registerMaker(bytes32(uint256(42)));

        vm.prank(accused);
        r.requestWithdrawal();
        vm.warp(block.timestamp + COOLDOWN);

        bytes32 evidenceHash = keccak256("late");
        vm.prank(accuser);
        c.openGrievance{value: BOND}(accused, 11, evidenceHash);

        vm.prank(accused);
        vm.expectRevert(MwixnetRegistry.StakeFrozen_.selector);
        r.withdrawStake();

        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, uint256(11), evidenceHash);
        vm.warp(block.timestamp + WINDOW + 1);
        vm.prank(accuser);
        c.resolveGrievance(gid);

        vm.warp(block.timestamp + SLASHING_WINDOW + 1);

        uint256 remaining = 5 ether - (5 ether * 2500) / 10_000;
        vm.prank(accused);
        r.withdrawStake();
        assertEq(r.stake(accused), 0);
        assertEq(accused.balance, 50 ether - 5 ether + remaining);
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

    function test_constructor_reverts_invalid_bounty_burn_split() public {
        MwixnetRegistry r = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        vm.expectRevert(GrievanceCourt.InvalidBountyBurnSplit.selector);
        new GrievanceCourt(r, WINDOW, BOND, SLASH_BPS, 1000, 8000, SLASHING_WINDOW, judge);
    }

    function test_constructor_reverts_slash_bps_too_high() public {
        MwixnetRegistry r = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        vm.expectRevert(GrievanceCourt.SlashBpsTooHigh.selector);
        new GrievanceCourt(r, WINDOW, BOND, 10_001, BOUNTY_BPS, BURN_BPS, SLASHING_WINDOW, judge);
    }

    function test_partial_slash_keeps_maker_when_stake_stays_above_min() public {
        MwixnetRegistry r = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        GrievanceCourt c =
            new GrievanceCourt(r, WINDOW, BOND, 2000, 5000, 5000, SLASHING_WINDOW, judge);
        r.setGrievanceCourt(address(c));

        vm.deal(accuser, 50 ether);
        vm.deal(accused, 50 ether);
        vm.prank(accused);
        r.deposit{value: 5 ether}();
        bytes32 nh = bytes32(uint256(99));
        vm.prank(accused);
        r.registerMaker(nh);

        bytes32 ev = keccak256("partial");
        vm.prank(accuser);
        c.openGrievance{value: BOND}(accused, 77, ev);
        vm.warp(block.timestamp + WINDOW + 1);
        vm.prank(accuser);
        c.resolveGrievance(EvidenceLib.grievanceId(accuser, accused, 77, ev));

        uint256 slashed = (5 ether * 2000) / 10_000;
        assertEq(r.stake(accused), 5 ether - slashed);
        assertEq(r.makerNostrKeyHash(accused), nh);
    }

    function test_requestWithdrawal_reverts_during_slashing_window_after_exonerate() public {
        bytes32 evidenceHash = keccak256("lock");
        uint256 epochId = 66;

        vm.prank(accuser);
        court.openGrievance{value: BOND}(accused, epochId, evidenceHash);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);

        vm.prank(accused);
        court.defendGrievance(gid, hex"01");

        vm.prank(judge);
        court.adjudicateGrievance(gid, true);

        vm.prank(accused);
        vm.expectRevert(MwixnetRegistry.WithdrawalLocked.selector);
        registry.requestWithdrawal();

        vm.warp(block.timestamp + SLASHING_WINDOW + 1);
        vm.prank(accused);
        registry.requestWithdrawal();
        assertGt(registry.exitUnlockTime(accused), 0);
    }
}
