// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

/// @notice View surface used by `MwixnetRegistry` to gate `requestWithdrawal` / `withdrawStake`.
interface IGrievanceCourtExit {
    function openGrievanceCountAgainst(address accused) external view returns (uint256);
}
