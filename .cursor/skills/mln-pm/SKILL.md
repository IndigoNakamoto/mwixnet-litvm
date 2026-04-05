---
name: mln-pm
description: >-
  MLN program hygiene: derive priorities, blockers, and "on track" status from README, PRODUCT_SPEC section 9,
  PHASE_* playbooks, AGENTS.md, CHANGELOG, and recent git—without inventing scope or economics. Use when the user asks:
  priorities, what to do next, milestone status, roadmap sync, release readiness, production blockers, pick up after a break,
  stay on track, or program management for this repo.
---

# MLN program management (lightweight)

## Purpose

Give **actionable** status and ordering: what matters now, what is blocked, what is explicitly **not** the priority—grounded in **canonical docs and git**, not a parallel roadmap.

## Sources of truth (read in this order)

1. **`README.md`** — phase checkboxes, production disclaimer, "Next steps."
2. **`PRODUCT_SPEC.md`** — section 9 (P0–P3 table + **implementation status** paragraph); open questions in section 10.
3. **`AGENTS.md`** — current phase, canonical table, layer boundaries.
4. **Root `PHASE_*.md`** — depth for the phases the user cares about.
5. **`CHANGELOG.md`** — unreleased / recent shipped items.
6. **`git log --oneline -20`** (or a range the user names) — what actually landed recently.
7. **`research/THREAT_MODEL_MLN.md`** — **gates** and residual risks (e.g. unverified `defenseData`, audit status, ops surfaces).

For **documentation parity and link/CI wording fixes**, follow **`.cursor/skills/doc-sync/SKILL.md`** instead of duplicating that workflow here.

For **UX, trust UI, wallet flows**, follow **`.cursor/skills/mln-web3-product-design/SKILL.md`**.

For **cross-tool pairing** (Cursor + external models) and how rules/skills relate to **`AGENTS.md`**, see **`docs/AGENT_HANDOFF.md`**.

## Priority tiers (labels for the answer—not formal process)

Use these **only as inference from the docs**, and say so explicitly (e.g. "Inferred from README + spec").

| Tier | Typical content (this repo) |
| ---- | --------------------------- |
| **P0** | Blocks credible product or testnet claims: LitVM **public** testnet broadcast pending RPC; **security audit** absent; threat-model **Critical** rows (e.g. on-chain `defenseData` verification TBD); end-to-end MWEB validation path still open per spec section 9. |
| **P1** | README **unchecked** major phases (e.g. Phase 3 integration); fork hardening vs upstream; closing spec open questions the team has committed to. |
| **P2** | Phase 2 Nostr polish, operator UX, docs, CI hygiene—important but not the same as P0 gates. |

## Procedure

1. **Snapshot** — From README + `PRODUCT_SPEC` section 9 + AGENTS "Current phase," list: shipped vs pending vs external-blocked (e.g. LitVM RPC).
2. **Parity glance** — If README, section 9 blurb, and AGENTS disagree on "what is shipped," call it out and recommend a **doc-sync** pass (do not silently pick one narrative unless editing docs is in scope).
3. **Recent work** — `git log --oneline -20`: bucket into contracts, `mln-cli`, `mlnd`, `mln-sidecar`, deploy, docs.
4. **Threat / release gates** — Pull 1–3 **non-negotiable** items from `THREAT_MODEL_MLN.md` (audit, `defenseData`, sidecar bind, keys) that affect "production readiness."
5. **Output shape** (keep short):
   - **Now** — one primary focus.
   - **Next** — up to two follow-ons.
   - **Blocked** — dependencies outside the repo or waiting on third parties.
   - **Explicitly not now** — one de-prioritized item to protect focus.
6. **Optional: release / tag checklist** (when the user asks for release readiness)—verify mentally or list: `CHANGELOG.md` updated, README phase claims still true, contracts/tests green (`make contracts-test` if relevant), threat-model doc history if attack surface changed (see doc-sync optional follow-up).

## Do not

- Invent APIs, economics, LitVM behavior, or dates not in `PRODUCT_SPEC.md`, contracts, or `research/` docs.
- Treat Nostr or README copy as **stake authority**—LitVM registry remains canonical for stake (`AGENTS.md` layer boundaries).
- Rewrite `PRODUCT_SPEC.md` economics or appendices under the guise of "PM"; spec edits stay within team norms (section 9 implementation status only unless the user explicitly expands scope—same as doc-sync).
- If **contract economics in git** appear to **contradict** `PRODUCT_SPEC.md`, **do not** "resolve" by editing the spec—**flag in bold** in chat for a human architect (same as doc-sync).

## Optional follow-up

- If the user wants recurring discipline, suggest a **dated** one-line note in `CHANGELOG.md` or a short milestone note in the relevant `PHASE_*.md` when a gate is cleared—only when they ask for repo edits.
