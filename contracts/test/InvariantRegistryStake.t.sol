// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {MwixnetRegistry} from "../src/MwixnetRegistry.sol";
import {IGrievanceCourtExit} from "../src/interfaces/IGrievanceCourtExit.sol";

/// @dev Court stub: benign exit views + forwards judicial calls so `msg.sender` on the registry is this contract.
contract RegistryCourtMock is IGrievanceCourtExit {
    MwixnetRegistry public immutable registry;

    constructor(MwixnetRegistry registry_) {
        registry = registry_;
    }

    function openGrievanceCountAgainst(address) external pure returns (uint256) {
        return 0;
    }

    function withdrawalLockUntil(address) external pure returns (uint256) {
        return 0;
    }

    function courtSlashStake(
        address maker,
        uint256 slashAmount,
        address accuser,
        uint256 bountyBps,
        uint256 burnBps
    ) external {
        registry.slashStake(maker, slashAmount, accuser, bountyBps, burnBps);
    }

    function courtFreeze(address maker) external {
        registry.freezeStake(maker);
    }

    function courtUnfreeze(address maker) external {
        registry.unfreezeStake(maker);
    }
}

/// @dev Stateful fuzz target: only these code paths mutate registry stake / balance (no stray ETH).
contract RegistryHandler is Test {
    MwixnetRegistry public immutable registry;
    RegistryCourtMock public immutable court;
    uint256 public immutable minStake;
    uint256 public immutable cooldown;

    address[16] internal actorPool;

    constructor(
        MwixnetRegistry registry_,
        RegistryCourtMock court_,
        uint256 minStake_,
        uint256 cooldown_
    ) {
        registry = registry_;
        court = court_;
        minStake = minStake_;
        cooldown = cooldown_;
        for (uint256 i = 0; i < 16; i++) {
            actorPool[i] = address(uint160(10_000 + i));
        }
    }

    function actorAt(uint256 index) public view returns (address) {
        return actorPool[index];
    }

    function _actor(uint256 seed) internal view returns (address) {
        return actorPool[bound(seed, 0, 15)];
    }

    /// @notice Sum of `stake` over the fixed actor pool (all handler deposits use these addresses).
    function sumTrackedStake() public view returns (uint256 s) {
        for (uint256 i = 0; i < 16; i++) {
            s += registry.stake(actorPool[i]);
        }
    }

    function deposit(uint256 seed, uint256 amt) external {
        address u = _actor(seed);
        amt = bound(amt, 1, 5000 ether);
        vm.deal(u, amt);
        vm.prank(u);
        registry.deposit{value: amt}();
    }

    function withdraw(uint256 seed, uint256 amt) external {
        address u = _actor(seed);
        if (registry.makerNostrKeyHash(u) != bytes32(0)) return;
        if (registry.stakeFrozen(u)) return;
        uint256 st = registry.stake(u);
        if (st == 0) return;
        amt = bound(amt, 1, st);
        vm.prank(u);
        try registry.withdraw(amt) {} catch (bytes memory) {}
    }

    function registerMaker(uint256 seed, bytes32 nh) external {
        address u = _actor(seed);
        if (registry.stake(u) < minStake) return;
        if (registry.stakeFrozen(u)) return;
        if (registry.exitUnlockTime(u) != 0) return;
        if (nh == bytes32(0)) nh = bytes32(uint256(1));
        vm.prank(u);
        try registry.registerMaker(nh) {} catch (bytes memory) {}
    }

    function requestWithdrawal(uint256 seed) external {
        address u = _actor(seed);
        vm.prank(u);
        try registry.requestWithdrawal() {} catch (bytes memory) {}
    }

    function withdrawStake(uint256 seed) external {
        address u = _actor(seed);
        uint256 unlock = registry.exitUnlockTime(u);
        if (unlock == 0) return;
        if (block.timestamp <= unlock) {
            vm.warp(unlock + 1);
        }
        vm.prank(u);
        try registry.withdrawStake() {} catch (bytes memory) {}
    }

    function slash(uint256 makerSeed, uint256 accuserSeed, uint256 slashAmt, uint256 bountyBps)
        external
    {
        address maker = _actor(makerSeed);
        address accuser = _actor(accuserSeed);
        if (accuser == address(0)) return;
        bountyBps = bound(bountyBps, 0, 10_000);
        uint256 burnBps = 10_000 - bountyBps;
        uint256 st = registry.stake(maker);
        if (st == 0) return;
        slashAmt = bound(slashAmt, 1, st);
        vm.deal(accuser, 0);
        court.courtSlashStake(maker, slashAmt, accuser, bountyBps, burnBps);
    }

    function freeze(uint256 seed) external {
        court.courtFreeze(_actor(seed));
    }

    function unfreeze(uint256 seed) external {
        court.courtUnfreeze(_actor(seed));
    }
}

contract InvariantRegistryStakeTest is Test {
    uint256 internal constant MIN = 1 ether;
    uint256 internal constant COOLDOWN = 48 hours;

    MwixnetRegistry internal registry;
    RegistryCourtMock internal court;
    RegistryHandler internal handler;

    function setUp() public {
        registry = new MwixnetRegistry(MIN, COOLDOWN);
        court = new RegistryCourtMock(registry);
        registry.setGrievanceCourt(address(court));
        handler = new RegistryHandler(registry, court, MIN, COOLDOWN);
        targetContract(address(handler));
        excludeContract(address(registry));
        excludeContract(address(court));
    }

    function invariant_registryBalanceEqualsSumTrackedStake() public view {
        assertEq(address(registry).balance, handler.sumTrackedStake());
    }

    function invariant_registeredMakersMeetMinStake() public view {
        for (uint256 i = 0; i < 16; i++) {
            address u = handler.actorAt(i);
            if (registry.makerNostrKeyHash(u) != bytes32(0)) {
                assertGe(registry.stake(u), MIN);
            }
        }
    }

    function test_stateless_accounting_depositSlash_balanceEqualsSum() public {
        address maker = address(uint160(20_001));
        address accuser = address(uint160(20_002));
        vm.deal(maker, 100 ether);
        vm.deal(accuser, 0);

        MwixnetRegistry r = new MwixnetRegistry(MIN, COOLDOWN);
        RegistryCourtMock c = new RegistryCourtMock(r);
        r.setGrievanceCourt(address(c));

        vm.prank(maker);
        r.deposit{value: 10 ether}();

        c.courtSlashStake(maker, 4 ether, accuser, 2500, 7500);

        assertEq(r.stake(maker), 6 ether);
        assertEq(address(r).balance, 6 ether);
        assertEq(r.stake(accuser), 0);
    }

    function test_stateless_accounting_exit_clearsRegistryBalance() public {
        address maker = address(uint160(20_003));
        vm.deal(maker, 100 ether);

        MwixnetRegistry r = new MwixnetRegistry(MIN, COOLDOWN);
        RegistryCourtMock c = new RegistryCourtMock(r);
        r.setGrievanceCourt(address(c));

        vm.startPrank(maker);
        r.deposit{value: MIN}();
        r.registerMaker(bytes32(uint256(0xabc)));
        r.requestWithdrawal();
        vm.stopPrank();

        uint256 unlock = r.exitUnlockTime(maker);
        vm.warp(unlock + 1);

        vm.prank(maker);
        r.withdrawStake();

        assertEq(r.stake(maker), 0);
        assertEq(r.makerNostrKeyHash(maker), bytes32(0));
        assertEq(address(r).balance, 0);
    }
}
