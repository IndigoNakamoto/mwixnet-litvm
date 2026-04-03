---
name: coinswapd-reference
description: Navigate the coinswapd teardown and a local ltcmweb/coinswapd clone (optional path research/coinswapd/, gitignored). Use when tracing RPCs, onion format, MWEB tx assembly, or comparing Grin/mwixnet assumptions to this codebase.
---

# coinswapd reference

## When to use

- Locating **swap RPCs**, forward/backward flows, or **onion** peel/build logic.
- Confirming **JSON field names** for the taker onion or inter-node payloads.
- Tracing **MWEB primitives** (outputs, kernels, range proofs) — follow imports to **`ltcd`**, not `mwebd` inside this binary.

## Workflow

1. Read `research/COINSWAPD_TEARDOWN.md` for the curated map (API, onion shape, crypto hotspots, and the **MLN sidecar** note: `mln-cli` route JSON vs upstream `swap_Swap`).
2. Open the cited paths in your local clone (e.g. `research/coinswapd/main.go`, `onion/onion.go`, `swap.go` — not committed here).
3. For Pedersen, bulletproofs, wire types — continue into **`ltcd`** as documented in the teardown, not guessed from Grin-only docs.

## Out of scope for this skill

- Product requirements and economics — use `PRODUCT_SPEC.md`.
- LitVM contract design — spec §5–6; do not infer Solidity from `coinswapd` alone.
