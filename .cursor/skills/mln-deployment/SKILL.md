---
name: mln-deployment
description: >-
  Deployment and packaging for MLN: Docker Compose stacks under deploy/, root docker-compose, GitHub Actions
  release workflows, binary build Makefile targets, and environment templates. Use when changing operator
  bring-up, CI release jobs, or env examples—keep secrets out of git and align phase playbooks with README.
---

# MLN deployment / release engineering

## When to use

- Editing **`deploy/**`** Compose files (E2E, testnet, sidecar RPC overrides).
- Changing **`.github/workflows/`** especially **`mlnd-release.yml`** and stack tests that drive **`docker compose`**.
- Updating **environment templates**: **`.env.compose.example`**, **`contracts/.env.example`**, **`deploy/.env.testnet.example`**, **`mlnd/.env.example`**.
- Adjusting root **`docker-compose.yml`** or documenting **`make docker-build`**, **`make build`**, **`make build-mln-cli`**, etc.

## Workflow

1. Read the relevant **phase playbook** (**`PHASE_9_ENABLEMENT.md`**, **`PHASE_12_E2E_CRUCIBLE.md`**, **`PHASE_16_PUBLIC_TESTNET.md`**) so compose env vars and services match documented operator paths.
2. **Never commit** real **`PRIVATE_KEY`**, relay passwords, or Tor credentials—only **`.example`** files and docs.
3. After changing **release** or **image** behavior, cross-check **`mlnd/Dockerfile`**, **`Makefile`** **`docker-build`**, and **`README.md`** release section.
4. When **RPC or chain defaults** change, verify **`mln-cli/internal/config/settings.go`** and **`PHASE_16_PUBLIC_TESTNET.md`** stay consistent (see **`AGENTS.md`** canonical table).
5. For **CHANGELOG** and shipped-version claims, coordinate with **`.cursor/skills/doc-sync/SKILL.md`** and **`CHANGELOG.md`**.

## Canonical paths

- **`deploy/docker-compose.e2e.yml`**, **`deploy/docker-compose.testnet.yml`**, **`deploy/docker-compose.e2e.sidecar-rpc.yml`**
- **`docker-compose.yml`**, **`.env.compose.example`**
- **`.github/workflows/mlnd-release.yml`**, **`test-full-stack.yml`**
- **`mlnd/README.md`**, **`PHASE_8_TESTNET_RELEASE.md`**

## Do not

- Turn **operator docs** into speculative **production SLAs** or monitoring promises without product sign-off.
- Add **closed-source** or **non-reproducible** build steps without team agreement.
- Weaken **CI** gates (e.g. drop contract tests) in workflows without explicit rationale in **`research/THREAT_MODEL_MLN.md`** or maintainer consensus.

## Coordination

- **QA / test commands for workflows:** **`.cursor/skills/mln-qa/SKILL.md`**.
- **LitVM deploy scripts inside `contracts/`:** **`.cursor/skills/mln-contracts/SKILL.md`**.
- **README / phase index / RPC claims:** **`.cursor/skills/doc-sync/SKILL.md`**.
