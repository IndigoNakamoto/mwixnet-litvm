// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Deploy} from "../script/Deploy.s.sol";

/// @notice Ensures Deploy.s.sol refuses unsafe cooldown vs challenge pairings.
contract DeployGuardrailTest is Test {
    function test_deploy_reverts_when_cooldown_not_above_challenge_plus_slack() public {
        // Default anvil account 0 — required because Deploy reads PRIVATE_KEY.
        vm.setEnv("PRIVATE_KEY", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80");
        vm.setEnv("COOLDOWN_PERIOD", "3600"); // 1 hour
        vm.setEnv("CHALLENGE_WINDOW", "86400"); // 24 hours
        vm.setEnv("DEPLOY_MIN_COOLDOWN_CHALLENGE_SLACK", "0");

        Deploy script = new Deploy();
        vm.expectRevert();
        script.run();
    }

    function test_deploy_reverts_when_slack_makes_requirement_stricter() public {
        vm.setEnv("PRIVATE_KEY", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80");
        vm.setEnv("COOLDOWN_PERIOD", "100000"); // ~27.7 h
        vm.setEnv("CHALLENGE_WINDOW", "86400"); // 24 h — strictly greater
        vm.setEnv("DEPLOY_MIN_COOLDOWN_CHALLENGE_SLACK", "20000"); // 86400+20000 > 100000

        Deploy script = new Deploy();
        vm.expectRevert();
        script.run();
    }
}
