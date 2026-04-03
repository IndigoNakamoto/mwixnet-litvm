// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

/// @notice View surface used by `MwixnetRegistry` to gate `requestWithdrawal` / `withdrawStake`.
interface IGrievanceCourtExit {
    function openGrievanceCountAgainst(address accused) external view returns (uint256);

    /// @notice Makers cannot start or complete registry exit until `block.timestamp` exceeds this value (per-case lock extended on each grievance resolution).
    function withdrawalLockUntil(address maker) external view returns (uint256);
}
