// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {EvidenceLib} from "../src/EvidenceLib.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {GrievanceCourt} from "../src/GrievanceCourt.sol";

contract EvidenceHashTest is Test {
    /// @dev Appendix 13.5 field order: abi.encodePacked(epochId, accuser, accusedMaker, hopIndex, peeledCommitment, forwardCiphertextHash)
    function test_evidenceHash_matches_manual_encodePacked() public pure {
        uint256 epochId = 42;
        address accuser = address(uint160(0xBEEF));
        address accusedMaker = address(uint160(0xCAFE));
        uint8 hopIndex = 2;
        bytes32 peeled = bytes32(uint256(0x1111));
        bytes32 forwardCt = bytes32(uint256(0x2222));

        bytes32 got =
            EvidenceLib.evidenceHash(epochId, accuser, accusedMaker, hopIndex, peeled, forwardCt);
        bytes32 want = keccak256(
            abi.encodePacked(epochId, accuser, accusedMaker, hopIndex, peeled, forwardCt)
        );
        assertEq(got, want);
    }

    function test_grievanceId_matches_manual_encodePacked() public pure {
        address accuser = address(uint160(0xA));
        address accused = address(uint160(0xB));
        uint256 epochId = 99;
        bytes32 ev = keccak256("evidence payload");

        bytes32 got = EvidenceLib.grievanceId(accuser, accused, epochId, ev);
        bytes32 want = keccak256(abi.encodePacked(accuser, accused, epochId, ev));
        assertEq(got, want);
    }

    function test_grievanceId_matches_openGrievance_storage_key() public {
        MwixnetRegistry registry = new MwixnetRegistry(1 ether);
        GrievanceCourt court = new GrievanceCourt(registry, 1 days, 0.01 ether);
        registry.setGrievanceCourt(address(court));

        address accuser = address(0xACC);
        address accused = address(0xBCC);
        uint256 epochId = 7;
        bytes32 ev = EvidenceLib.evidenceHash(
            epochId, accuser, accused, 1, bytes32(uint256(0xdead)), bytes32(uint256(0xbeef))
        );

        vm.deal(accuser, 1 ether);
        vm.prank(accuser);
        court.openGrievance{value: 0.05 ether}(accused, epochId, ev);

        bytes32 expectedGid = EvidenceLib.grievanceId(accuser, accused, epochId, ev);
        (,,,,,, GrievanceCourt.GrievancePhase ph,) = court.grievances(expectedGid);
        assertEq(uint256(ph), uint256(GrievanceCourt.GrievancePhase.Open));
    }
}
