// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {IGrievanceCourtExit} from "./interfaces/IGrievanceCourtExit.sol";

/// @title MwixnetRegistry
/// @notice Stake and maker registration on LitVM (zkLTC native token). See PRODUCT_SPEC.md sections 5–6.
/// @dev Not audited. Registered makers use a timelocked exit queue (`requestWithdrawal` / `withdrawStake`) per PRODUCT_SPEC.
contract MwixnetRegistry is ReentrancyGuard {
    uint256 public immutable minStake;
    /// @notice Cooldown after `requestWithdrawal` before `withdrawStake`. Must exceed max epoch length + challenge window in production.
    uint256 public immutable cooldownPeriod;
    address public owner;
    address public grievanceCourt;

    mapping(address => uint256) public stake;
    mapping(address => bool) public stakeFrozen;
    mapping(address => bytes32) public makerNostrKeyHash;
    /// @notice Timestamp when `withdrawStake` becomes callable; 0 if not in exit queue.
    mapping(address => uint256) public exitUnlockTime;

    event StakeDeposited(address indexed maker, uint256 amount);
    event MakerRegistered(address indexed maker, bytes32 nostrKeyHash);
    event Withdrawn(address indexed maker, uint256 amount);
    event ExitRequested(address indexed maker, uint256 unlockTime);
    event StakeWithdrawnAfterExit(address indexed maker, uint256 amount);
    event StakeFrozen(address indexed maker);
    event StakeUnfrozen(address indexed maker);
    event GrievanceCourtSet(address indexed court);
    event StakeSlashed(
        address indexed maker,
        address indexed accuser,
        uint256 totalSlashed,
        uint256 bounty,
        uint256 burned
    );
    event MakerDeregisteredLowStake(address indexed maker);

    error NotOwner();
    error NotGrievanceCourt();
    error BelowMinStake();
    error CourtAlreadySet();
    error ZeroAddress();
    error StakeFrozen_();
    error InsufficientStake();
    error RegisteredMakerMustUseExitQueue();
    error NotRegisteredMaker();
    error AlreadyInExitQueue();
    error GrievanceCourtNotSet();
    error OpenGrievanceBlocksExit();
    error NotInExitQueue();
    error CooldownNotComplete();
    error InExitQueue();
    error WithdrawalLocked();
    error InvalidBountyBurnSplit();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyGrievanceCourt() {
        if (msg.sender != grievanceCourt) revert NotGrievanceCourt();
        _;
    }

    constructor(uint256 minStake_, uint256 cooldownPeriod_) {
        minStake = minStake_;
        cooldownPeriod = cooldownPeriod_;
        owner = msg.sender;
    }

    /// @notice One-time wire of the judicial contract that may freeze stake.
    function setGrievanceCourt(address court) external onlyOwner {
        if (grievanceCourt != address(0)) revert CourtAlreadySet();
        if (court == address(0)) revert ZeroAddress();
        grievanceCourt = court;
        emit GrievanceCourtSet(court);
    }

    /// @dev Native zkLTC / gas token deposit for staking.
    function deposit() external payable {
        stake[msg.sender] += msg.value;
        emit StakeDeposited(msg.sender, msg.value);
    }

    /// @notice Register as maker after meeting `minStake`. `nostrKeyHash` binds off-chain Nostr identity (opaque bytes32).
    function registerMaker(bytes32 nostrKeyHash) external {
        if (exitUnlockTime[msg.sender] != 0) revert InExitQueue();
        if (stake[msg.sender] < minStake) revert BelowMinStake();
        if (stakeFrozen[msg.sender]) revert StakeFrozen_();
        makerNostrKeyHash[msg.sender] = nostrKeyHash;
        emit MakerRegistered(msg.sender, nostrKeyHash);
    }

    /// @notice Partial withdraw only for addresses that have **not** registered as a maker (no `makerNostrKeyHash`).
    function withdraw(uint256 amount) external {
        if (makerNostrKeyHash[msg.sender] != bytes32(0)) revert RegisteredMakerMustUseExitQueue();
        if (stakeFrozen[msg.sender]) revert StakeFrozen_();
        if (stake[msg.sender] < amount) revert InsufficientStake();
        stake[msg.sender] -= amount;
        (bool ok,) = payable(msg.sender).call{value: amount}("");
        require(ok, "withdraw transfer");
        emit Withdrawn(msg.sender, amount);
    }

    /// @notice Begin timelocked exit: stop advertising off-chain, then wait `cooldownPeriod` before `withdrawStake`.
    function requestWithdrawal() external nonReentrant {
        if (makerNostrKeyHash[msg.sender] == bytes32(0)) revert NotRegisteredMaker();
        if (exitUnlockTime[msg.sender] != 0) revert AlreadyInExitQueue();
        address court = grievanceCourt;
        if (court == address(0)) revert GrievanceCourtNotSet();
        if (IGrievanceCourtExit(court).openGrievanceCountAgainst(msg.sender) != 0) {
            revert OpenGrievanceBlocksExit();
        }
        if (block.timestamp < IGrievanceCourtExit(court).withdrawalLockUntil(msg.sender)) {
            revert WithdrawalLocked();
        }
        if (stakeFrozen[msg.sender]) revert StakeFrozen_();

        uint256 unlock = block.timestamp + cooldownPeriod;
        exitUnlockTime[msg.sender] = unlock;
        emit ExitRequested(msg.sender, unlock);
    }

    /// @notice After cooldown and with no open grievance freeze, withdraw full stake and clear maker registration.
    function withdrawStake() external nonReentrant {
        uint256 unlock = exitUnlockTime[msg.sender];
        if (unlock == 0) revert NotInExitQueue();
        if (block.timestamp < unlock) revert CooldownNotComplete();
        if (stakeFrozen[msg.sender]) revert StakeFrozen_();

        address court = grievanceCourt;
        if (court != address(0)) {
            if (IGrievanceCourtExit(court).openGrievanceCountAgainst(msg.sender) != 0) {
                revert OpenGrievanceBlocksExit();
            }
            if (block.timestamp < IGrievanceCourtExit(court).withdrawalLockUntil(msg.sender)) {
                revert WithdrawalLocked();
            }
        }

        uint256 amount = stake[msg.sender];
        stake[msg.sender] = 0;
        makerNostrKeyHash[msg.sender] = bytes32(0);
        exitUnlockTime[msg.sender] = 0;

        (bool ok,) = payable(msg.sender).call{value: amount}("");
        require(ok, "withdrawStake transfer");
        emit StakeWithdrawnAfterExit(msg.sender, amount);
    }

    /// @notice Called by `GrievanceCourt` on upheld grievance: slash `slashAmount` from `maker`, pay bounty to `accuser`, burn remainder to `address(0)`.
    /// @dev `bountyBps + burnBps` must equal 10_000. Remaining stake below `minStake` clears `makerNostrKeyHash` and `exitUnlockTime` (routing pool drop).
    function slashStake(
        address maker,
        uint256 slashAmount,
        address accuser,
        uint256 bountyBps,
        uint256 burnBps
    ) external onlyGrievanceCourt nonReentrant {
        if (accuser == address(0)) revert ZeroAddress();
        if (bountyBps + burnBps != 10_000) revert InvalidBountyBurnSplit();

        uint256 st = stake[maker];
        if (slashAmount > st) {
            slashAmount = st;
        }
        if (slashAmount == 0) {
            return;
        }

        uint256 bounty = (slashAmount * bountyBps) / 10_000;
        uint256 burned = slashAmount - bounty;

        stake[maker] = st - slashAmount;

        if (makerNostrKeyHash[maker] != bytes32(0) && stake[maker] < minStake) {
            makerNostrKeyHash[maker] = bytes32(0);
            exitUnlockTime[maker] = 0;
            emit MakerDeregisteredLowStake(maker);
        }

        if (bounty != 0) {
            (bool okB,) = payable(accuser).call{value: bounty}("");
            require(okB, "slash bounty transfer");
        }
        if (burned != 0) {
            (bool okBr,) = payable(address(0)).call{value: burned}("");
            require(okBr, "slash burn transfer");
        }

        emit StakeSlashed(maker, accuser, slashAmount, bounty, burned);
    }

    function freezeStake(address maker) external onlyGrievanceCourt {
        stakeFrozen[maker] = true;
        emit StakeFrozen(maker);
    }

    function unfreezeStake(address maker) external onlyGrievanceCourt {
        stakeFrozen[maker] = false;
        emit StakeUnfrozen(maker);
    }
}
