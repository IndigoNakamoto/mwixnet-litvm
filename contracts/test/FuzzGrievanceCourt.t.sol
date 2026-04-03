// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {EvidenceLib} from "../src/EvidenceLib.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {GrievanceCourt} from "../src/GrievanceCourt.sol";

contract FuzzGrievanceCourtTest is Test {
    uint256 internal constant MIN_STAKE = 1 ether;
    uint256 internal constant COOLDOWN = 48 hours;
    uint256 internal constant BOND_MIN = 0.01 ether;
    uint256 internal constant WINDOW = 1 days;
    uint256 internal constant SLASH_BPS = 10_000;
    uint256 internal constant BOUNTY_BPS = 5000;
    uint256 internal constant BURN_BPS = 5000;
    uint256 internal constant SLASHING_WINDOW = 1 days;

    function testFuzz_open_reverts_if_bond_below_min(uint72 bondWei) public {
        bondWei = uint72(bound(uint256(bondWei), 0, BOND_MIN - 1));

        MwixnetRegistry registry = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        GrievanceCourt court = new GrievanceCourt(
            registry, WINDOW, BOND_MIN, SLASH_BPS, BOUNTY_BPS, BURN_BPS, SLASHING_WINDOW
        );
        registry.setGrievanceCourt(address(court));

        address accuser = address(0xA1);
        address accused = address(0xB1);
        vm.deal(accuser, 1 ether);
        vm.deal(accused, 5 ether);
        vm.prank(accused);
        registry.deposit{value: 5 ether}();

        vm.prank(accuser);
        vm.expectRevert(GrievanceCourt.InsufficientBond.selector);
        court.openGrievance{value: bondWei}(accused, 1, bytes32(uint256(1)));
    }

    function testFuzz_double_open_same_id_reverts(
        address accuser,
        address accused,
        uint256 epochId,
        bytes32 evidenceHash
    ) public {
        vm.assume(accuser != address(0) && accused != address(0));
        vm.assume(accuser != accused);

        MwixnetRegistry registry = new MwixnetRegistry(MIN_STAKE, COOLDOWN);
        GrievanceCourt court = new GrievanceCourt(
            registry, WINDOW, BOND_MIN, SLASH_BPS, BOUNTY_BPS, BURN_BPS, SLASHING_WINDOW
        );
        registry.setGrievanceCourt(address(court));

        vm.deal(accuser, 10 ether);
        vm.deal(accused, 10 ether);
        vm.prank(accused);
        registry.deposit{value: 5 ether}();

        vm.prank(accuser);
        court.openGrievance{value: 1 ether}(accused, epochId, evidenceHash);

        vm.prank(accuser);
        vm.expectRevert(GrievanceCourt.AlreadyExists.selector);
        court.openGrievance{value: 1 ether}(accused, epochId, evidenceHash);
    }

    function testFuzz_grievanceId_matches_evidenceLib(
        address accuser,
        address accused,
        uint256 epochId,
        bytes32 evidenceHash
    ) public pure {
        bytes32 a = EvidenceLib.grievanceId(accuser, accused, epochId, evidenceHash);
        bytes32 b = keccak256(abi.encodePacked(accuser, accused, epochId, evidenceHash));
        assertEq(a, b);
    }
}
