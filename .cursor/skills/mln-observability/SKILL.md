---
name: mln-observability
description: >-
  Observability guidance for MLN Go services: structured logging patterns, operator-facing diagnostics, and
  safe metrics hooks—without inventing product analytics pipelines. Use when improving logs, debug endpoints,
  or documenting what operators should monitor post-deploy.
---

# MLN observability and iteration

## When to use

- Improving **logging** or **diagnostic output** in **`mlnd`**, **`mln-cli`**, or **`mln-sidecar`**.
- Adding **operator-facing** status surfaces (e.g. dashboard or health summaries) that must stay **non-custodial** and **metadata-aware**.
- Drafting **runbooks** for “what to watch” after **`docker compose`** or **release binary** deployment.

## Workflow

1. Prefer **structured, actionable** logs (level, component, correlation id where available); follow existing patterns such as **`mlnd/internal/opslog/`** when extending **`mlnd`**.
2. **Do not** log **secrets**, **full onion payloads**, or **raw PII**; align with privacy goals in **`PRODUCT_SPEC.md`** and **`research/THREAT_MODEL_MLN.md`**.
3. Separate **dev-only verbosity** from **default operator** noise; gate heavy debug behind flags or build tags where the codebase already does so.
4. For **“what to build next”** after incidents or operator feedback, feed summaries into **`.cursor/skills/mln-pm/SKILL.md`** (priorities grounded in docs + git—not a parallel roadmap).
5. **Third-party analytics** (e.g. product telemetry) are **out of scope** unless explicitly approved; do not embed vendors by default.

## Canonical paths

- **`mlnd/internal/opslog/`** — existing logging helper patterns in **`mlnd`**.
- **`mlnd/MAKER_DASHBOARD_SETUP.md`** — operator loopback dashboard expectations.
- **`PHASE_9_ENABLEMENT.md`**, **`research/THREAT_MODEL_MLN.md`** — ops surfaces and residual risks.

## Do not

- Implement **centralized user tracking** of takers/makers without a documented privacy stance.
- Replace **LitVM** or **Nostr** truth with **log-derived** stake or route authority.
- Add **high-cardinality** metrics that could **de-anonymize** users or makers.

## Coordination

- **Go implementation details:** **`.cursor/skills/mln-go-engineer/SKILL.md`**.
- **Deployment / compose / releases:** **`.cursor/skills/mln-deployment/SKILL.md`**.
- **PM prioritization from field feedback:** **`.cursor/skills/mln-pm/SKILL.md`**.
