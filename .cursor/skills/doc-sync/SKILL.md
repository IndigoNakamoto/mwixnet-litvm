---
name: mln-doc-sync
description: >-
  Aligns top-level MLN documentation with git history and the repo tree: README index vs root PHASE_*.md,
  PRODUCT_SPEC section 9 implementation status, AGENTS/THREAT_MODEL consistency, PoC vs production copy,
  and audits for broken relative links plus stale CI or RPC claims. Use when the user asks for a documentation
  synchronization pass, doc review against shipped code, README Documentation table parity, or markdown link audit.
---

# MLN documentation synchronization

## Scope (typical)

- **README.md** — roadmap, Production disclaimer, Documentation table, Makefile mentions.
- **PRODUCT_SPEC.md** — section 9 **implementation status** paragraph only (keep P0–P3 product table authoritative; do not rewrite economics or appendices unless the user explicitly expands scope).
- **AGENTS.md** — “Current phase” and canonical table rows if capabilities or paths changed.
- **research/THREAT_MODEL_MLN.md** — snapshot/changelog row when CI, contracts, or major stack behavior changes.
- **Root `PHASE_*.md`** — cross-links and claims vs `.github/workflows/**`, `Makefile`, and real paths.

## Procedure

1. **Baseline** — `git log --oneline -30` (or since the last `docs:` / doc-sync commit). Bucket commits: contracts/CI, `mln-sidecar`, `mln-cli`, `mlnd`, workflows, deploy scripts.
2. **Parity matrix** — Same “shipped” facts in README roadmap, PRODUCT_SPEC section 9 blurb, and AGENTS “Current phase” where they overlap.
3. **README Documentation table** — **Every** root `PHASE_*.md` file gets a row (numeric order 5–15). There is **no** `PHASE_11_*.md` or `PHASE_13_*.md`: use `mln-cli/desktop/README.md` for Phase 11 and `mln-sidecar/README.md` for Phase 13.
4. **Production vs PoC** — Roadmap area should keep a **single-sentence** disclaimer: in-tree Phases 1–15 PoC can be feature-complete while **not production** until **LitVM testnet broadcast**, **security audit**, and production **`coinswapd` / MWEB integration** (README may say “C++ `coinswapd`” per stakeholder wording; the **published ltcmweb reference is Go** — align with `PRODUCT_SPEC.md` and `research/COINSWAPD_TEARDOWN.md` unless product copy explicitly overrides).
5. **Link and reference audit** — For relative markdown targets in touched files, confirm paths exist. Grep for misleading anchors (e.g. link text `contracts/.env` pointing at `.env.example`). Confirm **Slither** language matches `.github/workflows/contracts.yml` (job present, `fail-on`, `filter-paths`). RPC names **`mweb_submitRoute`** / **`mweb_getBalance`** should stay consistent with `mln-sidecar/README.md` and `research/COINSWAPD_TEARDOWN.md`.

## Do not

- Invent APIs, economics, or LitVM behavior not in `PRODUCT_SPEC.md` or contracts.
- Expand `PRODUCT_SPEC.md` with speculative implementation (repo norm: mark draft/TBD).
- Edit vendored markdown under `contracts/lib/`.

## Optional follow-up

- If new capabilities land, add or adjust a dated line at the bottom of `research/THREAT_MODEL_MLN.md` when the threat/CI picture changes.
