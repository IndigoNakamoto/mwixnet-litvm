// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

/// @title MwixnetRegistry
/// @notice Stake and maker registration on LitVM (zkLTC native token). See PRODUCT_SPEC.md sections 5–6.
/// @dev Not audited. Withdrawal and freeze semantics are minimal scaffolding for Phase 1.
contract MwixnetRegistry {
    uint256 public immutable minStake;
    address public owner;
    address public grievanceCourt;

    mapping(address => uint256) public stake;
    mapping(address => bool) public stakeFrozen;
    mapping(address => bytes32) public makerNostrKeyHash;

    event StakeDeposited(address indexed maker, uint256 amount);
    event MakerRegistered(address indexed maker, bytes32 nostrKeyHash);
    event Withdrawn(address indexed maker, uint256 amount);
    event StakeFrozen(address indexed maker);
    event StakeUnfrozen(address indexed maker);
    event GrievanceCourtSet(address indexed court);

    error NotOwner();
    error NotGrievanceCourt();
    error BelowMinStake();
    error CourtAlreadySet();
    error ZeroAddress();
    error StakeFrozen_();
    error InsufficientStake();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyGrievanceCourt() {
        if (msg.sender != grievanceCourt) revert NotGrievanceCourt();
        _;
    }

    constructor(uint256 minStake_) {
        minStake = minStake_;
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
        if (stake[msg.sender] < minStake) revert BelowMinStake();
        if (stakeFrozen[msg.sender]) revert StakeFrozen_();
        makerNostrKeyHash[msg.sender] = nostrKeyHash;
        emit MakerRegistered(msg.sender, nostrKeyHash);
    }

    function withdraw(uint256 amount) external {
        if (stakeFrozen[msg.sender]) revert StakeFrozen_();
        if (stake[msg.sender] < amount) revert InsufficientStake();
        stake[msg.sender] -= amount;
        (bool ok,) = payable(msg.sender).call{value: amount}("");
        require(ok, "withdraw transfer");
        emit Withdrawn(msg.sender, amount);
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
