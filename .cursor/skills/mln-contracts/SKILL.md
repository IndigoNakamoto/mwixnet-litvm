---
name: mln-contracts
description: >-
  Foundry Solidity for MLN LitVM: registry, grievance court, evidence hashing, deployment scripts, fuzz and
  invariant tests, and CI alignment (Slither). Use when editing contracts/, Foundry config, or on-chain
  test harnesses—economics and dispute design stay canonical per PRODUCT_SPEC.md.
---

# MLN contracts engineer

## When to use

- Changing **`contracts/src/`** (`MwixnetRegistry`, `GrievanceCourt`, `EvidenceLib`, etc.).
- Adding or updating **`contracts/test/`** (unit, fuzz, invariant) or **`contracts/script/`** deploy helpers.
- Adjusting **`foundry.toml`**, **`Makefile`** contract targets, or **`.github/workflows/contracts.yml`** expectations.

## Workflow

1. Treat **`PRODUCT_SPEC.md`** (especially registry, grievance, and appendix 13 preimage material) as **canonical for product economics and dispute design**. If **code and spec diverge**, **do not silently rewrite the spec to match code**—**flag in bold** for a human architect (same norm as **`.cursor/skills/doc-sync/SKILL.md`**).
2. Read **`AGENTS.md`** layer boundaries: LitVM handles **stake, bonds, slashing, grievances**—not happy-path mix verification on-chain.
3. Prefer **OpenZeppelin** patterns already vendored under **`contracts/lib/`**; follow existing **Solidity 0.8.24** and **`foundry.toml`** settings.
4. Extend **tests** alongside logic changes: unit tests, **fuzz**, and **invariant** runs where the contract surface warrants it (see existing **`contracts/test/`**).
5. Before claiming CI parity, align with **`.github/workflows/contracts.yml`** (**`forge build` / `forge test`**, **Slither** on `contracts/**` when applicable).
6. For **appendix 13** encodings, keep **`contracts/src/EvidenceLib.sol`** and **`contracts/test/EvidenceHash.t.sol`** in sync with the spec narrative.

## Canonical paths

- **`contracts/README.md`** — layout, local Anvil, `make contracts-test`.
- **`contracts/foundry.toml`** — solc, optimizer, invariant profile.
- **`PHASE_15_ECONOMIC_HARDENING.md`** — slash economics, locks, reentrancy guard context.
- **`research/LITVM.md`** — testnet and deployment notes.
- **`scripts/deploy-local-anvil.sh`** — local deploy path.

## Do not

- Invent **zkLTC**, bridge, or **LitVM** chain behavior not documented in **`PRODUCT_SPEC.md`** / **`research/LITVM.md`**.
- Weaken **reentrancy** or **access control** without explicit threat-model review (**`research/THREAT_MODEL_MLN.md`**).
- Edit **vendored** markdown under **`contracts/lib/`** for “doc sync”—exclude per doc-sync skill.

## Coordination

- **Documentation parity / README phase tables:** **`.cursor/skills/doc-sync/SKILL.md`**.
- **Cross-layer protocol claims:** **`.cursor/rules/mln-architecture.mdc`**.
- **QA commands and workflows:** **`.cursor/skills/mln-qa/SKILL.md`**.
