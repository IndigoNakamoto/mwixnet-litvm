---
name: mln-qa
description: >-
  Quality assurance for MLN: Go tests (mln-cli, mlnd, mln-sidecar), Foundry tests and fuzz/invariants,
  Nostr fixture validation, Makefile integration targets, and GitHub Actions workflows. Use when adding
  tests, debugging CI, or validating compose/E2E paths. Desktop browser automation (e.g. Playwright) is
  not in-tree today—treat as future optional work unless explicitly requested.
---

# MLN quality assurance

## When to use

- Adding or fixing **`*_test.go`** under **`mln-cli/`**, **`mlnd/`**, **`mln-sidecar/`**, or **`research/coinswapd/`** (fork tests).
- Adding or fixing **Solidity** tests under **`contracts/test/`** or scripts under **`contracts/script/`** that affect deploy/verify flows.
- Updating **`.github/workflows/`** or validating **Docker Compose**-based local stacks in **`deploy/`**.
- Investigating **regressions** reported from **`make`** targets or CI jobs.

## Workflow

1. **Go:** From the relevant module directory, run **`go test ./...`**; use **`CGO_ENABLED=1`** when SQLite or Wails-related packages require it (see **`Makefile`** and **`mlnd/README.md`**).
2. **Contracts:** Use **`make contracts-test`** or **`make contracts-test-match MATCH=...`** (Docker Foundry) per **`Makefile`** and **`contracts/README.md`**.
3. **Integration / stack:** Use documented targets such as **`make test-operator-smoke`**, **`make test-full-stack`**, **`make testnet-smoke`** (env-dependent) per root **`Makefile`** and phase playbooks (**`PHASE_12_E2E_CRUCIBLE.md`**, **`PHASE_7_END_TO_END.md`**, **`PHASE_16_PUBLIC_TESTNET.md`**).
4. **Nostr wire fixtures:** Align with **`research/NOSTR_MLN.md`** and CI (**`.github/workflows/nostr-fixtures.yml`**, **`nostr/`**).
5. **Before merge:** Run the **narrowest** tests that cover your change, then broaden if you touched shared libraries or contracts.
6. **Coverage:** Aim for **strong tests on new or risky paths** (correctness > a numeric percentage); do not block merges on arbitrary coverage thresholds unless the team has agreed them.

## Canonical paths

- **`.github/workflows/contracts.yml`**, **`mlnd.yml`**, **`test-full-stack.yml`**, **`nostr-fixtures.yml`**, **`mlnd-release.yml`**, **`coinswapd-research.yml`** — CI entrypoints.
- **`Makefile`** — contract and operator smoke commands.
- **`PHASE_12_E2E_CRUCIBLE.md`**, **`deploy/docker-compose.e2e.yml`**, **`scripts/e2e-bootstrap.sh`** — local E2E crucible.

## Do not

- Claim **Playwright** or **desktop UI automation** exists in-repo today; add such tooling only with an explicit task and CI ownership.
- Disable **Slither**, **fuzz**, or **invariant** tests to “go green” without documenting residual risk (**`research/THREAT_MODEL_MLN.md`**).
- Commit **secrets** or **production relay keys** into tests or fixtures.

## Coordination

- **Contract semantics:** **`.cursor/skills/mln-contracts/SKILL.md`**.
- **Go feature work:** **`.cursor/skills/mln-go-engineer/SKILL.md`**.
- **Deploy/compose/release workflow edits:** **`.cursor/skills/mln-deployment/SKILL.md`**.
