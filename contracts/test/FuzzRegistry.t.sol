// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";

contract FuzzRegistryTest is Test {
    MwixnetRegistry internal registry;
    uint256 internal constant MIN = 1 ether;
    uint256 internal constant COOLDOWN = 48 hours;

    function setUp() public {
        registry = new MwixnetRegistry(MIN, COOLDOWN);
    }

    function testFuzz_deposit_increases_stake(address user, uint96 amount) public {
        vm.assume(user != address(0));
        vm.assume(uint160(user) > 1024);
        vm.assume(user.code.length == 0);
        amount = uint96(bound(uint256(amount), 1, 10_000 ether));
        vm.deal(user, amount);
        vm.prank(user);
        registry.deposit{value: amount}();
        assertEq(registry.stake(user), amount);
    }

    function testFuzz_withdraw_reduces_stake(address user, uint128 depositAmt, uint128 withdrawAmt)
        public
    {
        vm.assume(user != address(0));
        vm.assume(uint160(user) > 1024);
        vm.assume(user.code.length == 0);
        depositAmt = uint128(bound(uint256(depositAmt), MIN, 1000 ether));
        vm.assume(withdrawAmt <= depositAmt);

        vm.deal(user, depositAmt);
        vm.prank(user);
        registry.deposit{value: depositAmt}();

        vm.prank(user);
        registry.withdraw(withdrawAmt);
        assertEq(registry.stake(user), depositAmt - withdrawAmt);
    }

    function testFuzz_freeze_blocks_withdraw(address user, uint128 depositAmt) public {
        vm.assume(user != address(0));
        vm.assume(uint160(user) > 1024);
        vm.assume(user.code.length == 0);
        depositAmt = uint128(bound(uint256(depositAmt), MIN, 500 ether));

        GrievanceMock court = new GrievanceMock();
        registry.setGrievanceCourt(address(court));

        vm.deal(user, depositAmt);
        vm.prank(user);
        registry.deposit{value: depositAmt}();

        vm.prank(address(court));
        registry.freezeStake(user);

        vm.prank(user);
        vm.expectRevert(MwixnetRegistry.StakeFrozen_.selector);
        registry.withdraw(1);
    }
}

/// @dev Minimal contract to satisfy `onlyGrievanceCourt` on registry.
contract GrievanceMock {}
