// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {GrievanceCourt} from "../src/GrievanceCourt.sol";

/// @notice Deploy registry, judicial court, then wire `grievanceCourt` on the registry (one-time).
/// @dev Load env: PRIVATE_KEY, optionally broadcast with --rpc-url $LITVM_RPC_URL
contract Deploy is Script {
    function run() external {
        uint256 minStake = vm.envOr("MIN_STAKE", uint256(0.1 ether));
        uint256 cooldownPeriod = vm.envOr("COOLDOWN_PERIOD", uint256(48 hours));
        uint256 challengeWindow = vm.envOr("CHALLENGE_WINDOW", uint256(24 hours));
        uint256 grievanceBondMin = vm.envOr("GRIEVANCE_BOND_MIN", uint256(0.01 ether));

        uint256 pk = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(pk);

        MwixnetRegistry registry = new MwixnetRegistry(minStake, cooldownPeriod);
        GrievanceCourt court = new GrievanceCourt(registry, challengeWindow, grievanceBondMin);
        registry.setGrievanceCourt(address(court));

        vm.stopBroadcast();

        console.log("MwixnetRegistry:", address(registry));
        console.log("GrievanceCourt:", address(court));
    }
}
