// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {GrievanceCourt} from "../src/GrievanceCourt.sol";

contract MwixnetRegistryTest is Test {
    MwixnetRegistry internal registry;
    uint256 internal constant MIN = 1 ether;
    uint256 internal constant COOLDOWN = 48 hours;

    function setUp() public {
        registry = new MwixnetRegistry(MIN, COOLDOWN);
    }

    /// @dev Non-makers can deposit and partial-withdraw freely.
    function test_deposit_withdraw_without_register() public {
        vm.deal(address(this), 10 ether);
        registry.deposit{value: 5 ether}();
        assertEq(registry.stake(address(this)), 5 ether);

        uint256 before = address(this).balance;
        registry.withdraw(2 ether);
        assertEq(address(this).balance, before + 2 ether);
        assertEq(registry.stake(address(this)), 3 ether);
    }

    function test_register_then_exit_withdrawStake() public {
        vm.deal(address(this), 10 ether);
        registry.deposit{value: 5 ether}();
        registry.registerMaker(keccak256("nostr"));
        assertEq(registry.makerNostrKeyHash(address(this)), keccak256("nostr"));

        vm.expectRevert(MwixnetRegistry.RegisteredMakerMustUseExitQueue.selector);
        registry.withdraw(1 ether);

        address arbiter = address(uint160(0xB008));
        GrievanceCourt court =
            new GrievanceCourt(registry, 1 days, 0.01 ether, 10_000, 1000, 9000, 7 days, arbiter);
        registry.setGrievanceCourt(address(court));

        registry.requestWithdrawal();
        uint256 unlock = registry.exitUnlockTime(address(this));
        assertEq(unlock, block.timestamp + COOLDOWN);

        vm.expectRevert(MwixnetRegistry.CooldownNotComplete.selector);
        registry.withdrawStake();

        vm.warp(unlock);
        uint256 balBefore = address(this).balance;
        registry.withdrawStake();
        assertEq(address(this).balance, balBefore + 5 ether);
        assertEq(registry.stake(address(this)), 0);
        assertEq(registry.makerNostrKeyHash(address(this)), bytes32(0));
    }

    function test_cannot_register_below_min() public {
        vm.deal(address(this), 1 ether);
        registry.deposit{value: 0.5 ether}();
        vm.expectRevert(MwixnetRegistry.BelowMinStake.selector);
        registry.registerMaker(bytes32(0));
    }

    receive() external payable {}
}
