---
name: commit
description: >-
  Stages intentional changes and creates a git commit with a clear conventional-style message, respecting MLN boundaries and secret hygiene. Use when the user asks to commit, save work to git, or stage and commit; pair with verification guidance from CONTRIBUTING when the diff touches contracts or Go.
---

# Git commit (`/commit`)

## When to run

- The user says **commit**, **`/commit`**, **stage and commit**, **create a git commit**, or equivalent after edits they want recorded.
- After completing work that should land on the branch as one or more logical commits (not only when paired with `/plans`).

## Before committing

1. Run **`git status`** (and **`git diff`** for non-trivial changes) so the commit matches what the user intended.
2. **Never stage or commit secrets:** real `.env`, private keys, relay passwords, credential-bearing RPC URLs, or generated **`deploy/e2e*.env`** / local vault DBs if they contain sensitive material. Prefer **`.example`** templates and docs. When unsure, stop and flag.
3. **Stage narrowly:** use **`git add <paths>`** for the files that belong to this change. Avoid **`git add -A`** unless the user explicitly wants the whole tree and you have confirmed nothing unrelated is included.
4. If the change is **non-trivial** or **cross-layer**, ensure intent is documented per **[`CONTRIBUTING.md`](../../../CONTRIBUTING.md)** (e.g. plan under [`.cursor/plans/`](../../plans/README.md) when appropriate; layer callout for reviewers).

## Commit message

- **Subject line:** imperative mood, **~50–72 characters**, no trailing period. Optional **conventional prefix** when it helps history: `feat:`, `fix:`, `docs:`, `chore:`, `test:`, `contracts:` — match whatever the repo already uses in recent **`git log`**.
- **Body (optional):** Use when the diff is multi-area or needs reviewer context: what changed, why, and any follow-ups. Complete sentences; no vague one-liners that duplicate the subject.
- **Do not** put secrets, tokens, or long stack traces in the message.

## After commit

- Show **`git log -1 --oneline`** (or short hash + subject) so the user sees the result.
- **`git push`** only when the user asks to push; otherwise leave push to them.
- Do **not** **`--amend`** or **force-push** unless the user explicitly requests it.

## MLN verification reminder (when relevant)

Match blast radius to **[`CONTRIBUTING.md`](../../../CONTRIBUTING.md)** before or after commit as appropriate:

- **`contracts/**`** — e.g. **`make contracts-test`**; see [`.cursor/skills/mln-contracts/SKILL.md`](../mln-contracts/SKILL.md) / [`.cursor/skills/mln-qa/SKILL.md`](../mln-qa/SKILL.md).
- **Go** — targeted **`go test`** for touched packages.
- **Docs-only** — link sanity; use [`.cursor/skills/doc-sync/SKILL.md`](../doc-sync/SKILL.md) when roadmap or index tables moved.

Layer boundaries remain per **[`AGENTS.md`](../../../AGENTS.md)**.

## Plans lifecycle

If the session produced new **`.cursor/plans/*.md`** files that document the same work, **commit them in the same commit or the next commit** on the branch, per [`.cursor/plans/README.md`](../../plans/README.md) lifecycle — not left only on disk.

## Do not

- Commit generated artifacts that are **gitignored** on purpose unless the user explicitly wants them tracked (rare).
- Rewrite history on **shared default branch** without explicit instruction.
