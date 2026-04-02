// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

/// @title EvidenceLib
/// @notice Pure helpers for hashes defined in PRODUCT_SPEC.md appendix 13. Off-chain clients should match these encodings.
library EvidenceLib {
    /// @notice Appendix 13.5 — packed preimage then keccak256 (what accusers compute off-chain before `openGrievance`).
    function evidenceHash(
        uint256 epochId,
        address accuser,
        address accusedMaker,
        uint8 hopIndex,
        bytes32 peeledCommitment,
        bytes32 forwardCiphertextHash
    ) internal pure returns (bytes32) {
        return keccak256(
            abi.encodePacked(
                epochId, accuser, accusedMaker, hopIndex, peeledCommitment, forwardCiphertextHash
            )
        );
    }

    /// @notice Grievance storage key — must match `GrievanceCourt.openGrievance` (accuser is `msg.sender` on-chain).
    function grievanceId(address accuser, address accused, uint256 epochId, bytes32 evidenceHash_)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(accuser, accused, epochId, evidenceHash_));
    }
}
