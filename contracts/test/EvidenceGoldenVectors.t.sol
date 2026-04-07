// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {EvidenceLib} from "../src/EvidenceLib.sol";

/// @dev Locks golden outputs in `research/EVIDENCE_GENERATOR.md` to the compiler / EvidenceLib.
contract EvidenceGoldenVectorsTest is Test {
    bytes32 internal constant EXPECTED_EVIDENCE_HASH =
        bytes32(hex"2d4d7ae96f39e2d5037f21782bc831874261ffe22743f74bbf865a39ec4df112");
    bytes32 internal constant EXPECTED_GRIEVANCE_ID =
        bytes32(hex"5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3");

    function test_golden_vectors_match_EVIDENCE_GENERATOR_md() public pure {
        uint256 epochId = 42;
        address accuser = address(uint160(0xBEEF));
        address accusedMaker = address(uint160(0xCAFE));
        uint8 hopIndex = 2;
        bytes32 peeled = bytes32(uint256(0x1111));
        bytes32 forwardCt = bytes32(uint256(0x2222));

        bytes memory preimage =
            abi.encodePacked(epochId, accuser, accusedMaker, hopIndex, peeled, forwardCt);
        assertEq(preimage.length, 137);

        bytes32 ev =
            EvidenceLib.evidenceHash(epochId, accuser, accusedMaker, hopIndex, peeled, forwardCt);
        assertEq(ev, EXPECTED_EVIDENCE_HASH);

        bytes32 gid = EvidenceLib.grievanceId(accuser, accusedMaker, epochId, ev);
        assertEq(gid, EXPECTED_GRIEVANCE_ID);
    }
}
