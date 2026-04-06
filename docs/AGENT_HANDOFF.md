---
title: MLN Agent Handoff Guide – Cursor + Grok (or any external model)
description: How the .cursor/rules + .cursor/skills system works and how to collaborate between Cursor and external agents (Grok, Gemini, Claude, etc.)
---

# MLN Agent Handoff Guide: Cursor + Grok (or any external model)

Here's a concise review of how things are wired today and how to pair Cursor with another chat.

## What `.cursor` is doing

You have a **three-tier system**:

1. **Rules** (`.cursor/rules/*.mdc`) — short, Cursor-injected instructions. Frontmatter controls scope:
   - **`mln-core.mdc`** is `alwaysApply: true` on `**/*`, so every conversation gets MLN identity, pointers to `AGENTS.md` / `PRODUCT_SPEC.md`, layer boundaries, and a map of which **skills** exist.
   - Other rules are **globs**-scoped (e.g. `mln-go-engineer.mdc` for `mln-cli/**`, `mlnd/**`, `mln-sidecar/**`) so Go work pulls in the right skill pointer without loading everything everywhere.
2. **Skills** (`.cursor/skills/*/SKILL.md`) — longer playbooks: when to use them, workflow steps, canonical paths, "do not" lists. Rules **route** to them; capable agents are supposed to **read** the matching `SKILL.md` when the task fits.
3. **Plans** (`.cursor/plans/*.md`, plus [`.cursor/plans/README.md`](../.cursor/plans/README.md)) — optional, human-reviewed markdown specs for non-trivial work. Checked in when they drive a change so the next session or another tool sees the same intent. Portable to external models as a paste bundle together with excerpts from `AGENTS.md`.

That split keeps always-on context small while still having deep procedures and reviewable intent in-repo.

## How `AGENTS.md` is used

`AGENTS.md` is the **human- and agent-facing index**:

- **"Where truth lives"** table → which file is canonical for Nostr, contracts, phases, threat model, etc.
- **Layer boundaries** → the non-negotiable MWEB / LitVM / Nostr / Tor split.
- **Current phase** → long-form status (what's shipped, what's stub vs production).

Cursor's always-on rule **does not duplicate** the whole table; it **points** to `AGENTS.md` so the agent is expected to open it when routing or when the task touches multiple layers. Skills then add **task-specific** workflows on top.

**Strength:** One source of truth in git; rules stay maintainable.  
**In practice** we mitigate hallucinations by always starting Cursor prompts with the relevant agent name and letting `mln-core.mdc` inject the rest.

## Using Cursor chat together with Grok or Gemini

There is **no built-in** "Cursor agent talks to Grok" bridge. Treat them as **parallel brains** with **shared artifacts**:

| Role               | Cursor                              | Grok / Gemini                              |
| ------------------ | ----------------------------------- | ------------------------------------------ |
| Repo truth         | Reads files, applies rules/skills, edits code | Only what you paste or upload             |
| Best for           | Implementation, refactors, tests, grep-driven debugging | Broad reasoning, alternative designs, long docs review |

**Practical workflows:**

1. **Cursor implements, external reviews** — Finish a change in Cursor; export or copy the diff summary + the relevant section of `PRODUCT_SPEC.md` or `AGENTS.md` into the other chat and ask for protocol/security/product review.
2. **External designs, Cursor executes** — In Grok/Gemini, explore options with explicit constraint: "Must respect MWEB vs LitVM separation per this paste." Paste **Layer boundaries** + one paragraph of task from `AGENTS.md`. Bring the **conclusion** back to Cursor as a short spec bullet list. When the work is non-trivial, save an approved plan under `.cursor/plans/` (see [`CONTRIBUTING.md`](../CONTRIBUTING.md)) and **paste that file path** into the Cursor prompt so implementation follows the same constraints.
3. **Portable "context pack" for any external model** — Minimal paste: link or paste `AGENTS.md` layer-boundary section + the **single** canonical doc row for your topic (from the table) + file paths you care about. We often also paste the latest `mln-pm` status report when the task touches priorities.

**Caveats:** External tools won't honor `alwaysApply` rules; they may blur layers unless you paste boundaries. Don't put secrets or keys into any chat.

**Bottom line:** `.cursor` is your **scoped automation** of "read AGENTS + the right skill (+ optional saved plan)"; `AGENTS.md` is the **repo router and boundary contract**. Grok/Gemini are useful as **second opinions and design space**, with **you** carrying the same boundaries and canonical paths between tools.
