---
name: mln-architecture-diagrams
description: Mermaid templates for MLN layer boundaries, MWEB/LitVM/Nostr/Tor flows, and grievance paths. Citations only — no new protocol claims.
---

## When to use
Use this skill whenever a diagram is requested for architecture, onboarding, or PHASE_*.md updates.

## Workflow
1. Always start with the canonical table in AGENTS.md and PRODUCT_SPEC.md section 9.
2. Output only valid Mermaid diagrams (flowchart, sequence, or class).
3. Cite sources inline (e.g. "per AGENTS.md layer boundaries").
4. Reference existing skills: mln-architecture, coinswapd-reference, mln-core.

## Canonical paths
- AGENTS.md (layer boundaries)
- PRODUCT_SPEC.md
- research/THREAT_MODEL_MLN.md
- research/RED_TEAM_MLN.md

## Do not
- Invent new protocol behavior or economics
- Modify any spec or contract files
- Add non-Mermaid output unless asked

## Coordination
Defer any economics or product questions to mln-pm + mln-web3-product-design + doc-sync.
