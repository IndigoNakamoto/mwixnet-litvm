// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Script, console2} from "forge-std/Script.sol";
import {EvidenceLib} from "../src/EvidenceLib.sol";

/// @dev Run: `forge script script/PrintEvidenceVectors.s.sol --sig 'run()'`
///      Golden outputs for research/EVIDENCE_GENERATOR.md (same inputs as EvidenceHash.t.sol).
contract PrintEvidenceVectors is Script {
    function run() external {
        uint256 epochId = 42;
        address accuser = address(uint160(0xBEEF));
        address accusedMaker = address(uint160(0xCAFE));
        uint8 hopIndex = 2;
        bytes32 peeled = bytes32(uint256(0x1111));
        bytes32 forwardCt = bytes32(uint256(0x2222));

        bytes memory preimage =
            abi.encodePacked(epochId, accuser, accusedMaker, hopIndex, peeled, forwardCt);
        require(preimage.length == 137, "preimage len");

        bytes32 ev =
            EvidenceLib.evidenceHash(epochId, accuser, accusedMaker, hopIndex, peeled, forwardCt);
        bytes32 gid = EvidenceLib.grievanceId(accuser, accusedMaker, epochId, ev);

        console2.log("inputs:");
        console2.log("  epochId", epochId);
        console2.log("  accuser", accuser);
        console2.log("  accusedMaker", accusedMaker);
        console2.logUint(uint256(hopIndex));
        console2.logBytes32(peeled);
        console2.logBytes32(forwardCt);

        console2.log("preimage_hex (137 bytes):");
        console2.logBytes(preimage);

        console2.log("evidenceHash:");
        console2.logBytes32(ev);

        console2.log("grievanceId (for same accuser/accused/epoch/ev):");
        console2.logBytes32(gid);
    }
}
