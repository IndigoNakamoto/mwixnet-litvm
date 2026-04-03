// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {GrievanceCourt} from "../src/GrievanceCourt.sol";

/// @notice Deploy registry, judicial court, then wire `grievanceCourt` on the registry (one-time).
/// @dev Env: PRIVATE_KEY (vm.envUint). RPC is not read in Solidity — broadcast with:
///      forge script script/Deploy.s.sol:Deploy --rpc-url "$RPC_URL" --broadcast [--verify --etherscan-api-key "$ETHERSCAN_API_KEY"]
contract Deploy is Script {
    function run() external {
        uint256 minStake = vm.envOr("MIN_STAKE", uint256(0.1 ether));
        uint256 cooldownPeriod = vm.envOr("COOLDOWN_PERIOD", uint256(48 hours));
        uint256 challengeWindow = vm.envOr("CHALLENGE_WINDOW", uint256(24 hours));
        uint256 grievanceBondMin = vm.envOr("GRIEVANCE_BOND_MIN", uint256(0.01 ether));
        uint256 slashBps = vm.envOr("SLASH_BPS", uint256(10_000));
        uint256 bountyBps = vm.envOr("BOUNTY_BPS", uint256(1000));
        uint256 burnBps = vm.envOr("BURN_BPS", uint256(9000));
        uint256 slashingWindow = vm.envOr("SLASHING_WINDOW", uint256(7 days));

        uint256 pk = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(pk);

        MwixnetRegistry registry = new MwixnetRegistry(minStake, cooldownPeriod);
        GrievanceCourt court = new GrievanceCourt(
            registry, challengeWindow, grievanceBondMin, slashBps, bountyBps, burnBps, slashingWindow
        );
        registry.setGrievanceCourt(address(court));

        vm.stopBroadcast();

        console.log("MwixnetRegistry:", address(registry));
        console.log("GrievanceCourt:", address(court));
    }
}
