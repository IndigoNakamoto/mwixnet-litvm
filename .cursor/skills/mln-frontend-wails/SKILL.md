---
name: mln-frontend-wails
description: >-
  Wails v2 + Vite + React desktop UI for MLN taker wallet: frontend structure, build/embed flow, component
  patterns, accessibility, and trust/status UI aligned with wallet wireframes. Use when editing
  mln-cli/desktop/frontend/ or desktop shell assets—defer product copy and taker-first principles to mln-web3-product-design.
---

# MLN frontend (Wails + React)

## When to use

- Editing **`mln-cli/desktop/frontend/`** (Vite, React, CSS).
- Adjusting **Wails**-related **desktop** entry or embed paths documented in **`mln-cli/desktop/README.md`**.
- Improving **trust indicators**, progress states, truncation of hex, or **cold-start** UX per wireframes.

## Workflow

1. Follow **`.cursor/skills/mln-web3-product-design/SKILL.md`** for **taker-first** hierarchy, **Advanced/Developer** surfacing of protocol jargon, and canonical **UX docs** (`research/WALLET_TAKER_FLOW_V1.md`, `research/WALLET_MAKER_FLOW_V1.md`, `research/USER_STORIES_MLN.md`).
2. Match **existing** frontend stack: **React 18**, **Vite 5** (see **`mln-cli/desktop/frontend/package.json`**).
3. Use **`make build-mln-wallet-frontend`** from repo root when validating production bundle behavior (see **`Makefile`**).
4. Keep **layer boundaries** in UI copy: do not imply LitVM **authorizes** every hop unless **`PRODUCT_SPEC.md`** says so (**`AGENTS.md`**).
5. For **Go bindings** or Wails **backend** changes, coordinate with **`.cursor/skills/mln-go-engineer/SKILL.md`**.

## Canonical paths

- **`mln-cli/desktop/README.md`** — build tag **`wails`**, `make build-mln-wallet`.
- **`research/WALLET_TAKER_FLOW_V1.md`**, **`research/WALLET_MAKER_FLOW_V1.md`** — wireframe-level flows.
- **`mlnd/MAKER_DASHBOARD_SETUP.md`** — operator dashboard context when UI overlaps maker surfaces.

## Do not

- Stuff **UI mockup detail** into **`PRODUCT_SPEC.md`** unless the user explicitly expands scope (repo norm; see **`AGENTS.md`**).
- Add **telemetry or third-party analytics** without an explicit product/security decision.
- Introduce **heavy UI frameworks** that fight the current minimal React+Vite setup unless agreed.

## Coordination

- **Product / UX truth:** **`.cursor/skills/mln-web3-product-design/SKILL.md`** + **`.cursor/rules/mln-product-design.mdc`** when editing globbed UX docs.
- **Go / Wails bridge:** **`.cursor/skills/mln-go-engineer/SKILL.md`**.
- **Roadmap / README / phase claims:** **`.cursor/skills/doc-sync/SKILL.md`**.
