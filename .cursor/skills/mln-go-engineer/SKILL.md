---
name: mln-go-engineer
description: >-
  Go engineering for MLN daemons and clients: mln-cli, mlnd, mln-sidecar. Module layout, HTTP/RPC shims,
  go-nostr usage, go-ethereum v1.14 patterns in main modules, Wails bindings, and error handling—without
  blurring MWEB vs LitVM boundaries. Use when editing or extending Go code under mln-cli/, mlnd/, or mln-sidecar/.
---

# MLN Go engineer

## When to use

- Implementing or refactoring **`mlnd`**, **`mln-cli`**, or **`mln-sidecar`**.
- Wiring JSON-RPC (`mweb_*` via sidecar), HTTP handlers, Nostr broadcast/consume, or LitVM client calls.
- Adjusting **Wails** desktop bindings under `mln-cli/desktop/` (Go side) alongside the frontend skill.

## Workflow

1. Read **`AGENTS.md`** for layer boundaries and the canonical doc table.
2. For **`coinswapd`** behavior, RPC names, and onion shapes, follow **`.cursor/skills/coinswapd-reference/SKILL.md`** and **`research/COINSWAPD_TEARDOWN.md`** before assuming Grin/mwixnet mappings.
3. For stack-wide architecture claims, defer to **`.cursor/rules/mln-architecture.mdc`** and **`PRODUCT_SPEC.md`**.
4. Match existing **Go version and module boundaries**: `mln-cli` and `mlnd` use **Go 1.22** and **`go-ethereum` v1.14.x**; **`mln-sidecar`** uses **Go 1.20** and an older **`go-ethereum`**—do not casually unify versions without an explicit upgrade task.
5. Prefer **small, testable** packages; extend existing patterns (`internal/`, `pkg/`) and naming in each module.
6. After behavior changes, run **`go test ./...`** from the affected module (or repo **`Makefile`** targets when integration is involved).

## Canonical paths

- **`mln-cli/go.mod`**, **`mlnd/go.mod`**, **`mln-sidecar/go.mod`** — module entrypoints and dependency pins.
- **`PHASE_10_TAKER_CLI.md`**, **`PHASE_12_E2E_CRUCIBLE.md`**, **`PHASE_3_MWEB_HANDOFF_SLICE.md`** — CLI, E2E, and sidecar RPC handoff context.
- **`mln-sidecar/README.md`** — `/v1/balance`, `/v1/swap`, `-mode=mock` vs `-mode=rpc`.
- **`mln-cli/desktop/README.md`** — Wails build tag and desktop layout.

## Do not

- Invent **LitVM** economics, Nostr wire kinds, or MWEB cryptography not grounded in **`PRODUCT_SPEC.md`**, **`research/NOSTR_MLN.md`**, or the **coinswapd** teardown/fork.
- Move **stake authority** or fee rails to the wrong layer (MWEB vs LitVM vs Nostr)—see **`AGENTS.md`**.
- Add **production secrets** or real keys to the repo; use **`.env.example`** patterns and operator docs.

## Coordination

- **Architecture / protocol:** **`.cursor/rules/mln-architecture.mdc`** + **`AGENTS.md`**.
- **coinswapd / MWEB RPC:** **`.cursor/skills/coinswapd-reference/SKILL.md`**.
- **Desktop UX copy and taker-first UI:** **`.cursor/skills/mln-web3-product-design/SKILL.md`** (React/Vite side) + this skill (Go bindings).
- **Docs and phase claims:** **`.cursor/skills/doc-sync/SKILL.md`** when README or `PHASE_*` overlap.
