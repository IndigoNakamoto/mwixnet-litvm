---
name: mln-web3-product-design
description: >-
  Senior web3 product design for MLN privacy routing—taker vs maker surfaces, trust UX, information architecture.
  Use when the user asks for product/UX review, wallet flows, dashboard copy, onboarding, cognitive load, trust indicators,
  or when editing canonical UX docs (USER_STORIES, wallet wireframes, maker dashboard setup, mlnd README product blurbs).
  Also when reconciling taker-facing UI with PRODUCT_SPEC without blurring layer boundaries.
---

# MLN senior web3 product design

## Canonical files (edit these when changing product/UX truth)

| File | Owns |
|------|------|
| `research/USER_STORIES_MLN.md` | Personas, user stories, coordination narrative, **UX product principles** (taker-first, trust, IA). |
| `research/WALLET_TAKER_FLOW_V1.md` | Taker wallet wireframes and step-by-step UX. |
| `research/WALLET_MAKER_FLOW_V1.md` | Maker wallet / operator-adjacent UX. |
| `mlnd/MAKER_DASHBOARD_SETUP.md` | Maker dashboard setup and operator-facing surface. |
| `mlnd/README.md` | `mlnd` product/operator overview (keep aligned with dashboard + spec roles). |
| `PRODUCT_SPEC.md` | Architecture, economics, threat model, roadmap — **not** a dumping ground for UI mockups. UI belongs in the research/wallet docs unless the user explicitly asks for a spec appendix. |

When a change spans layers (e.g. new epoch UX), update **user stories + relevant wallet doc** first; touch `PRODUCT_SPEC.md` only for agreed product facts, and mark drafts TBD per `AGENTS.md`.

## Persona split (non-negotiable)

- **Default product surface = Taker.** The end user wants Point A → Point B privately with low mental load. Do not expose LitVM defense modes, Nostr kind numbers, or raw protocol jargon on primary screens.
- **Maker / operator** content lives in maker flows, `mlnd` README, and `MAKER_DASHBOARD_SETUP.md` — not mixed into the default taker path.
- **Advanced / developer:** Infrastructure terms, hex, kinds, and RPC names belong behind **Advanced settings** or **Developer mode** (or equivalent), not the happy path.

## Trust, status, and actionability

- **Persistent connection health:** Always-visible indicator that the bridge to the mixing engine (e.g. sidecar / node) is alive (e.g. green state + short endpoint label). Reduces “is anything running?” anxiety.
- **In-flight narrative:** Replace a lone spinner with **staged progress** (e.g. finding path → fee negotiation → building MWEB onion → queued for batch). Copy should be honest and mapped to real phases where possible.
- **Primary hierarchy:** Prominence order: **Amount**, **Destination**, **Total fee** (and privacy/speed preset if applicable). Technical identifiers are secondary.

## Information architecture (the “hex string” problem)

- **Truncate by default** long hex (e.g. `0x8f1f…ab91`). One-click **copy full value** for support/debug.
- Do not let 64-character strings dominate layout; they read as errors to non-developers.

## Empty and cold-start states

- First launch / pre-sync should not be a blank zero state. Use **warm onboarding copy** (e.g. syncing graph, finding routes) so waiting feels intentional, not broken.

## Coordination with other agents

- **Layer boundaries:** MWEB = mix + default routing fees; LitVM = stake/grievances; Nostr = discovery — do not imply LitVM authorizes every hop in the UI unless the spec says so.
- **Doc sync:** If a pass touches roadmap/README phase tables or CI claims, follow `.cursor/skills/doc-sync/SKILL.md`.
- **Spec edits:** Do not rewrite `PRODUCT_SPEC.md` economics or appendices unless the user expands scope; see `AGENTS.md` editing norms.
