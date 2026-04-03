---
phase: 15
slug: economic-hardening
title: "Phase 15: Economic Hardening"
overview: |
  Move LitVM contracts from freeze-only grievance scaffolding to production-oriented economics:
  real stake slashing on upheld grievances with configurable slash fraction (slashBps) and bounty/burn splits;
  exoneration path forfeits the accuser bond to the accused maker (anti-griefing);
  post-resolution slashing window (withdrawalLockUntil) blocking registry exit alongside open-grievance gating;
  auto-deregistration when slash leaves stake below minStake; OpenZeppelin ReentrancyGuard on sensitive registry paths;
  and CI/documentation hooks for invariant testing (Foundry) and Slither static analysis.
todos:
  - id: registry-slash-api
    content: Court-only slashStake + bounty to accuser + burn to address(0); auto-clear makerNostrKeyHash when stake < minStake
  - id: court-slash-resolve
    content: slashBps + bountyBps/burnBps immutables; resolve slash calls registry; exonerate transfers bond to accused
  - id: withdrawal-slashing-window
    content: withdrawalLockUntil + slashingWindow; IGrievanceCourtExit extension; gate requestWithdrawal/withdrawStake
  - id: tests-fuzz
    content: Foundry coverage for splits, locks, partial slash, constructor validation
  - id: invariant-todo
    content: contracts/test/InvariantRegistryStake.t.sol — prove registry balance vs sum(stake) / slash paths
  - id: slither-ci-todo
    content: .github/workflows/contracts.yml — enable slither job after triage
  - id: phase15-doc-readme
    content: This file + README roadmap + AGENTS.md cross-links
dependencies:
  - contracts/src/MwixnetRegistry.sol
  - contracts/src/GrievanceCourt.sol
  - contracts/src/interfaces/IGrievanceCourtExit.sol
  - contracts/script/Deploy.s.sol
  - .github/workflows/contracts.yml
  - README.md
spec_refs:
  - PRODUCT_SPEC.md (sections 5–6, slash distribution, exit queue)
---

# Phase 15: Economic Hardening

This phase replaces the **judicial scaffold** (freeze/unfreeze and nominal “slash” state without moving stake) with **enforced economics** aligned to [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) sections 5–6: slashing, bounty/burn routing, bond handling on exoneration, and stricter exit gating.

## What changed (scaffold → hardened)

| Area | Before | After |
|------|--------|--------|
| Upheld grievance (`ResolvedSlash`) | Phase flip only; stake unchanged | `slashBps` of accused stake removed; **bounty** to accuser, **burn** to `address(0)` |
| Exoneration (`ResolvedExonerate`) | Bond refunded to accuser | **Bond forfeited** to accused maker |
| Maker below `minStake` after slash | N/A | `makerNostrKeyHash` cleared; `exitUnlockTime` cleared (drops from active routing pool) |
| Exit after any resolution | Only `openGrievanceCountAgainst` mattered | Also **`withdrawalLockUntil`** = max prior, `block.timestamp + slashingWindow` |
| Reentrancy | No guard | `MwixnetRegistry` uses OpenZeppelin **`ReentrancyGuard`** on `slashStake`, `requestWithdrawal`, `withdrawStake` |

## Parameters (deploy / env)

| Symbol | Location | Meaning |
|--------|----------|---------|
| `minStake`, `cooldownPeriod` | `MwixnetRegistry` | Minimum maker stake; **exit cooldown** (must exceed max epoch + challenge window per spec) |
| `challengeWindow`, `grievanceBondMin` | `GrievanceCourt` | Defense deadline; minimum accuser bond |
| `slashBps` | `GrievanceCourt` | Fraction of **current** accused stake to slash on upheld grievance (`10_000` = 100%) |
| `bountyBps`, `burnBps` | `GrievanceCourt` | Split of **slashed amount**; must sum to `10_000` |
| `slashingWindow` | `GrievanceCourt` | After **any** resolution affecting an accused, registry `requestWithdrawal` / `withdrawStake` blocked until this interval passes |

[`contracts/script/Deploy.s.sol`](contracts/script/Deploy.s.sol) reads optional env overrides: `SLASH_BPS`, `BOUNTY_BPS`, `BURN_BPS`, `SLASHING_WINDOW` (see [`contracts/.env.example`](contracts/.env.example)).

## On-chain interfaces

- [`IGrievanceCourtExit`](contracts/src/interfaces/IGrievanceCourtExit.sol): `openGrievanceCountAgainst`, **`withdrawalLockUntil(address)`**.
- [`MwixnetRegistry.slashStake`](contracts/src/MwixnetRegistry.sol): **only** `grievanceCourt`; enforces `bountyBps + burnBps == 10_000`.

## Security follow-ups (tracked in repo)

1. **Foundry invariants:** [`contracts/test/InvariantRegistryStake.t.sol`](contracts/test/InvariantRegistryStake.t.sol) documents the target: prove **`address(registry).balance == Σ stake[m]`** (or equivalent global accounting) across `deposit`, `withdraw`, `withdrawStake`, and `slashStake`.
2. **Slither:** [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml) includes a top-of-file **TODO** and a disabled `slither` job until the analyzer is wired and findings are triaged.

## Operator and client notes

- **`mlnd` / Scout:** New immutables on `GrievanceCourt` change the **constructor ABI**; redeploy scripts and env must pass seven constructor arguments. Scout continues to use `eth_call` views (`stake`, `makerNostrKeyHash`, `stakeFrozen`, `minStake`); makers with **`makerNostrKeyHash == 0`** after slash must stop advertising until they re-register with sufficient stake.
- **Defense data:** On-chain verification of `defenseData` remains **out of scope**; economics and exit rules are hardened first (see [`research/THREAT_MODEL_MLN.md`](research/THREAT_MODEL_MLN.md)).

## Build and test

From repo root (or `contracts/` via Docker per CI):

```bash
make contracts-test
# or
docker run --rm --entrypoint forge -v "$(pwd)/contracts:/work" -w /work ghcr.io/foundry-rs/foundry:latest test -vv
```

OpenZeppelin remapping: [`contracts/remappings.txt`](contracts/remappings.txt) (`@openzeppelin/contracts/` → `lib/openzeppelin-contracts/contracts/`).
