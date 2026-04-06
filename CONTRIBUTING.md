---
title: Contributing to MLN Stack
description: How to contribute code, docs, or changes while respecting the project's AI agent setup.
---

# Contributing to MLN Stack

## Layer boundaries and canonical docs

- All changes must respect the layer boundaries and canonical files defined in [`AGENTS.md`](AGENTS.md).
- For non-trivial work, state explicitly which layers are in play (**MWEB**, **LitVM**, **Nostr**, **Tor**) so reviewers and agents do not blur responsibilities.

## Plans for non-trivial work

- Use Cursor Plan mode (or an equivalent research-then-spec workflow) before large or cross-cutting edits.
- Save the **human-reviewed** markdown plan under [`.cursor/plans/`](.cursor/plans/) following [`.cursor/plans/README.md`](.cursor/plans/README.md).
- The maintainer or author should review the plan before merge or alongside the PR.

## Verification (risk-aware)

Match verification to blast radius:

- **`contracts/**`** — Run Foundry tests locally (e.g. `make contracts-test` from the repo root) and expect CI parity with [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml). For command details and related checks, see [`.cursor/skills/mln-qa/SKILL.md`](.cursor/skills/mln-qa/SKILL.md) and [`.cursor/skills/mln-contracts/SKILL.md`](.cursor/skills/mln-contracts/SKILL.md).
- **Go (`mln-cli/`, `mlnd/`, `mln-sidecar/`)** — Run targeted `go test` for packages you change; use the mln-qa skill for workflow-level commands when unsure.
- **Docs-only** — Confirm relative links and, when roadmap or Documentation tables change, consider a doc-sync pass per [`.cursor/skills/doc-sync/SKILL.md`](.cursor/skills/doc-sync/SKILL.md).

This repository does **not** treat a broken default branch as acceptable for protocol-facing or contract changes. Keep [`research/THREAT_MODEL_MLN.md`](research/THREAT_MODEL_MLN.md) and CI gates in mind for security-sensitive edits.

## Agent and chat hygiene (optional)

- Start a **new chat** for a new feature when an old thread has accumulated noise or unrelated context.
- When asking an agent to implement something, attach **branch context** or a **diff summary** when helpful.

## Cursor rules, skills, and handoff

- Cursor agents use rules and skills under [`.cursor/`](.cursor/). See [`docs/AGENT_HANDOFF.md`](docs/AGENT_HANDOFF.md) for how rules and skills relate to `AGENTS.md` and how to pair Cursor with external models (Grok, Gemini, Claude, etc.).
