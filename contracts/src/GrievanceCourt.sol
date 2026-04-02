// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.24;

import {MwixnetRegistry} from "./MwixnetRegistry.sol";
import {EvidenceLib} from "./EvidenceLib.sol";
import {IGrievanceCourtExit} from "./interfaces/IGrievanceCourtExit.sol";

/// @title GrievanceCourt
/// @notice Judicial layer: bonds, grievance lifecycle, stake freeze signals. Does not verify MWEB or mix execution.
/// @dev `evidenceHash` is computed off-chain per PRODUCT_SPEC.md appendix 13 (packed preimage then keccak256). This contract
///      only stores the bytes32; it does not re-hash preimage fields on-chain.
contract GrievanceCourt is IGrievanceCourtExit {
    enum GrievancePhase {
        None,
        Open,
        Defended,
        ResolvedSlash,
        ResolvedExonerate
    }

    struct Grievance {
        address accuser;
        address accused;
        uint256 epochId;
        bytes32 evidenceHash;
        uint256 openedAt;
        uint256 deadline;
        GrievancePhase phase;
        uint256 bondAmount;
    }

    MwixnetRegistry public immutable registry;
    uint256 public immutable challengeWindow;
    uint256 public immutable grievanceBondMin;

    mapping(bytes32 grievanceId => Grievance) public grievances;

    /// @notice Number of grievances in `Open` or `Defended` phase against this accused maker (resolved cases decrement).
    mapping(address accused => uint256) public openGrievanceCountAgainst;

    event GrievanceOpened(
        bytes32 indexed grievanceId,
        address indexed accuser,
        address indexed accused,
        uint256 epochId,
        bytes32 evidenceHash,
        uint256 deadline
    );
    event Defended(bytes32 indexed grievanceId, address indexed accused);
    event ResolvedSlash(bytes32 indexed grievanceId);
    event ResolvedExonerate(bytes32 indexed grievanceId);

    error InsufficientBond();
    error BadPhase();
    error NotAccused();
    error TooEarly();
    error AlreadyExists();
    error InvariantOpenCount();

    constructor(MwixnetRegistry registry_, uint256 challengeWindow_, uint256 grievanceBondMin_) {
        registry = registry_;
        challengeWindow = challengeWindow_;
        grievanceBondMin = grievanceBondMin_;
    }

    /// @notice Accuser opens a case; `evidenceHash` must match PRODUCT_SPEC.md appendix 13 (off-chain).
    /// @param accused Registry identity (maker address) being blamed.
    function openGrievance(address accused, uint256 epochId, bytes32 evidenceHash)
        external
        payable
    {
        if (msg.value < grievanceBondMin) revert InsufficientBond();
        bytes32 grievanceId = EvidenceLib.grievanceId(msg.sender, accused, epochId, evidenceHash);
        if (grievances[grievanceId].phase != GrievancePhase.None) revert AlreadyExists();

        uint256 openedAt = block.timestamp;
        grievances[grievanceId] = Grievance({
            accuser: msg.sender,
            accused: accused,
            epochId: epochId,
            evidenceHash: evidenceHash,
            openedAt: openedAt,
            deadline: openedAt + challengeWindow,
            phase: GrievancePhase.Open,
            bondAmount: msg.value
        });

        registry.freezeStake(accused);
        openGrievanceCountAgainst[accused]++;
        emit GrievanceOpened(
            grievanceId, msg.sender, accused, epochId, evidenceHash, openedAt + challengeWindow
        );
    }

    /// @notice Accused submits opaque defense calldata (receipts, signatures); verification is off-chain or future module.
    function defendGrievance(bytes32 grievanceId, bytes calldata defenseData) external {
        Grievance storage g = grievances[grievanceId];
        if (g.phase != GrievancePhase.Open) revert BadPhase();
        if (msg.sender != g.accused) revert NotAccused();
        defenseData; // silence unused; real verifier TBD
        g.phase = GrievancePhase.Defended;
        emit Defended(grievanceId, msg.sender);
    }

    /// @notice If challenge window passes with no defense, slash path (state only; token routing TBD).
    /// @dev If defended, resolves to exonerate (state only). Bond transfers are intentionally minimal in v1.
    function resolveGrievance(bytes32 grievanceId) external {
        Grievance storage g = grievances[grievanceId];
        if (g.phase == GrievancePhase.None) revert BadPhase();

        if (g.phase == GrievancePhase.Open) {
            if (block.timestamp < g.deadline) revert TooEarly();
            g.phase = GrievancePhase.ResolvedSlash;
            _decrementOpenAgainst(g.accused);
            registry.unfreezeStake(g.accused);
            emit ResolvedSlash(grievanceId);
            _refundBond(g.accuser, g.bondAmount);
            return;
        }

        if (g.phase == GrievancePhase.Defended) {
            g.phase = GrievancePhase.ResolvedExonerate;
            _decrementOpenAgainst(g.accused);
            registry.unfreezeStake(g.accused);
            emit ResolvedExonerate(grievanceId);
            _refundBond(g.accuser, g.bondAmount);
            return;
        }

        revert BadPhase();
    }

    function _refundBond(address to, uint256 amount) private {
        if (amount == 0) return;
        (bool ok,) = payable(to).call{value: amount}("");
        require(ok, "bond refund");
    }

    function _decrementOpenAgainst(address accused) private {
        uint256 n = openGrievanceCountAgainst[accused];
        if (n == 0) revert InvariantOpenCount();
        unchecked {
            openGrievanceCountAgainst[accused] = n - 1;
        }
    }

    receive() external payable {}
}
