# Agent execution plans (optional)

This directory holds **human-reviewed** markdown plans for non-trivial work—typically produced in Cursor Plan mode (or equivalent) and saved here so intent survives context resets and handoffs.

## When to add a plan

Recommended when any of the following apply:

- **Multi-file** or cross-package changes (e.g. contracts + Go + deploy).
- **Cross-layer** work touching more than one of MWEB, LitVM, Nostr, or Tor (see [`AGENTS.md`](../../AGENTS.md) layer boundaries).
- **Protocol- or economics-adjacent** edits (defer to [`PRODUCT_SPEC.md`](../../PRODUCT_SPEC.md); do not invent economics in a plan).
- **Multi-session** agent or human work where the next session needs the same constraints.

Small, single-file doc fixes or obvious one-liners usually do not need a plan.

## Naming

Use ASCII filenames such as `YYYY-MM-DD-short-slug.md`, optionally including a branch or PR identifier in the slug. Keep names descriptive.

## Suggested sections (per plan file)

1. **Goal** — What success looks like in one short paragraph.
2. **In scope / out of scope** — Explicit boundaries to prevent scope creep.
3. **Primary files and canonical docs** — Paths to read before editing; cite [`AGENTS.md`](../../AGENTS.md) rows where helpful.
4. **Verification** — Commands or workflows to run after implementation. Prefer pointers to the [`Makefile`](../../Makefile) and [`.cursor/skills/mln-qa/SKILL.md`](../skills/mln-qa/SKILL.md) instead of copying a full matrix that will go stale.
5. **Layer-boundary check** — One explicit sentence: which layers are touched and that responsibilities are not blurred (MWEB vs LitVM vs Nostr vs Tor).

## Safety

Do **not** store secrets in plan files: private keys, live RPC URLs with credentials, or contents of `.env`. Reference **templates** only (e.g. `deploy/.env.testnet.example`, `contracts/.env.example`).

## Lifecycle

- If the plan drove a merged change, **commit the plan with the PR** (or in a preceding commit on the same branch) so history matches intent.
- If a plan is abandoned, **delete it** or add a short superseded note at the top pointing to the replacement approach.

Workflow norms for contributors and agents: [`CONTRIBUTING.md`](../../CONTRIBUTING.md).
