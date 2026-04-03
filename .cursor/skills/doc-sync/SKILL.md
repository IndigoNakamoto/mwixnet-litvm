---
name: mln-doc-sync
description: >-
  Aligns top-level MLN documentation with git history and the repo tree: README index vs root PHASE_*.md,
  PRODUCT_SPEC section 9 implementation status, AGENTS/THREAT_MODEL consistency, PoC vs production copy,
  and audits for broken relative links plus stale CI or RPC claims. Use when the user asks: 'sync docs', 'update readme',
  'run documentation pass', 'align docs with git', or 'check for stale links'. Also when they say: documentation
  synchronization pass, doc review against shipped code, README Documentation table parity, or markdown link audit.
---

# MLN documentation synchronization

## Scope (typical)

- **README.md** — roadmap, Production disclaimer, Documentation table, Makefile mentions.
- **PRODUCT_SPEC.md** — section 9 **implementation status** paragraph only (keep P0–P3 product table authoritative; do not rewrite economics or appendices unless the user explicitly expands scope).
- **Spec vs code (economics):** If git history or contracts show **smart-contract economics** that **contradict** `PRODUCT_SPEC.md` (including sections 5–6 or the P0–P3 table), follow the **Do not** rule below: do not edit the spec to match the code; **flag the discrepancy in chat (or PR text) in bold** for a human architect.
- **AGENTS.md** — “Current phase” and canonical table rows if capabilities or paths changed.
- **research/THREAT_MODEL_MLN.md** — snapshot/changelog row when CI, contracts, or major stack behavior changes.
- **research/RED_TEAM_MLN.md** — keep aligned with the threat model when attack surface or ops guidance changes; not a second audit trail.
- **Root `PHASE_*.md`** — cross-links and claims vs `.github/workflows/**`, `Makefile`, and real paths.

## Procedure

1. **Baseline** — `git log --oneline -30` (or since the last `docs:` / doc-sync commit). Bucket commits: contracts/CI, `mln-sidecar`, `mln-cli`, `mlnd`, workflows, deploy scripts.
2. **Parity matrix** — Same “shipped” facts in README roadmap, PRODUCT_SPEC section 9 blurb, and AGENTS “Current phase” where they overlap.
3. **README Documentation table** — **Every** root `PHASE_*.md` file gets a row (numeric order 5–16). There is **no** `PHASE_11_*.md` or `PHASE_13_*.md`: use `mln-cli/desktop/README.md` for Phase 11 and `mln-sidecar/README.md` for Phase 13.
4. **Production vs PoC** — Roadmap area should keep a **single-sentence** disclaimer: in-tree Phases 1–16 PoC can be feature-complete for documented bring-up while **not production** until **LitVM testnet broadcast** on the official chain, **security audit**, and production **`coinswapd` / MWEB integration**. The **published ltcmweb reference is Go**; the **tracked** MLN fork lives at `research/coinswapd/` — align wording with `PRODUCT_SPEC.md` and `research/COINSWAPD_TEARDOWN.md`.
5. **Link and reference audit** — For relative markdown targets in touched files, confirm paths exist. Grep for misleading anchors (e.g. link text `contracts/.env` pointing at `.env.example`). Confirm **Slither** language matches `.github/workflows/contracts.yml` (job present, `fail-on`, `filter-paths`). RPC names **`mweb_submitRoute`** / **`mweb_getBalance`** and optional hop field **`swapX25519PubHex`** should stay consistent with `mln-sidecar/README.md`, `research/COINSWAPD_TEARDOWN.md`, and `research/COINSWAPD_MLN_FORK_SPEC.md`.
6. **Self-Verification** — Before finishing, run `git diff` to review your own changes. Ensure no markdown formatting was broken (e.g., unclosed links or tables) and that you strictly adhered to the **Do not** constraints.

## Do not

- Invent APIs, economics, or LitVM behavior not in `PRODUCT_SPEC.md` or contracts.
- Expand `PRODUCT_SPEC.md` with speculative implementation (repo norm: mark draft/TBD).
- Edit vendored markdown under `contracts/lib/`.
- If git history shows changes to smart contract economics (e.g., slashing percentages) that contradict `PRODUCT_SPEC.md`, **DO NOT** edit the spec. Instead, leave a **bolded** comment in the chat output flagging the discrepancy for the human architect.

## Optional follow-up

- If new capabilities land, add or adjust a dated line at the bottom of `research/THREAT_MODEL_MLN.md` when the threat/CI picture changes.
