// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";

contract MwixnetRegistryTest is Test {
    MwixnetRegistry internal registry;
    uint256 internal constant MIN = 1 ether;

    function setUp() public {
        registry = new MwixnetRegistry(MIN);
    }

    function test_deposit_register_withdraw() public {
        vm.deal(address(this), 10 ether);
        registry.deposit{value: 5 ether}();
        assertEq(registry.stake(address(this)), 5 ether);

        registry.registerMaker(keccak256("nostr"));
        assertEq(registry.makerNostrKeyHash(address(this)), keccak256("nostr"));

        uint256 before = address(this).balance;
        registry.withdraw(2 ether);
        assertEq(address(this).balance, before + 2 ether);
        assertEq(registry.stake(address(this)), 3 ether);
    }

    function test_cannot_register_below_min() public {
        vm.deal(address(this), 1 ether);
        registry.deposit{value: 0.5 ether}();
        vm.expectRevert(MwixnetRegistry.BelowMinStake.selector);
        registry.registerMaker(bytes32(0));
    }

    receive() external payable {}
}
