# Nostr discovery route build

**Status:** Completed (merged in-tree). In-repo execution record; original Cursor plan id `nostr_discovery_route_build_d502b1ad`.

## Goal

Unify live Nostr + LitVM + pathfind into **`mln-cli route build`** and exclude makers with **`exitUnlockTime != 0`** from **`VerifyMaker`** so takers do not route through makers who have entered the exit queue. Reuse **`scout.Run`**, **`registry.VerifyMaker`**, and **`pathfind`** without rewriting wire or algorithms.

## Why exit-queue in `VerifyMaker`

Cooldown and exit queue exist to mitigate hit-and-run relative to the grievance window. If **`VerifyMaker`** ignores **`exitUnlockTime`**, a maker who has **`requestWithdrawal`** can still appear verified until stake is fully withdrawn, and pathfind may send new swaps through them. Treating non-zero **`exitUnlockTime`** as **not OK** closes that hole.

## Existing building blocks

- [mln-cli/internal/scout/scout.go](../../mln-cli/internal/scout/scout.go) — Nostr 31250 fetch + parse.
- [mln-cli/internal/registry/verify.go](../../mln-cli/internal/registry/verify.go) — LitVM checks (includes **`exitUnlockTime`**).
- [mln-cli/internal/pathfind/pathfind.go](../../mln-cli/internal/pathfind/pathfind.go) — 3-hop selection.
- Normative wire: [research/NOSTR_MLN.md](../../research/NOSTR_MLN.md).

**Route output shape:** same JSON as **`pathfind -json`** ([pathfind.Route](../../mln-cli/internal/pathfind/pathfind.go)) for **`forger -route-json`**.

## What shipped

| Item | Where |
|------|--------|
| **`exitUnlockTime`** guard; reason **`in exit queue`** | [mln-cli/internal/registry/verify.go](../../mln-cli/internal/registry/verify.go), table tests [verify_decision_test.go](../../mln-cli/internal/registry/verify_decision_test.go) |
| **`MLN_NOSTR_RELAY_URL`** if **`MLN_NOSTR_RELAYS`** empty | [mln-cli/internal/config/config.go](../../mln-cli/internal/config/config.go); tests [scout_env_test.go](../../mln-cli/internal/config/scout_env_test.go) (**no `t.Parallel()` with `t.Setenv`** on Go 1.26+) |
| **`mln-cli route build`** (`-out` default **`route.json`**, **`self-included`** parity) | [mln-cli/cmd/mln-cli/main.go](../../mln-cli/cmd/mln-cli/main.go) |
| Docs | [PHASE_10_TAKER_CLI.md](../../PHASE_10_TAKER_CLI.md); Taker CLI row in [AGENTS.md](../../AGENTS.md) (`route build` → `route.json`) |

**Deferred / out of scope:** Scout package file split; new Nostr kinds; pathfind reimplementation.

## Operator notes (bring-up)

- **Binary:** project builds **`bin/mln-cli`** via `make build-mln-cli` (not on `PATH` unless you `export PATH=.../bin:...` or `go install`). Run `./bin/mln-cli route build` from repo root after build.
- **Env + discovery:** [PHASE_10_TAKER_CLI.md](../../PHASE_10_TAKER_CLI.md) section **“Aligning discovery with LitVM (production vs local)”** — match **`MLN_*`** to **`content.litvm`** on kind **31250**; Phase **12** + local relay for Anvil; raw note inspection via event id from **`rejected <id>:`** lines.
- **`pathfind: need at least 3 verified makers with Tor endpoints, got 0`:** expected if no ads pass chain + registry + LitVM + Tor-endpoint filters for your env.
- **Phase 12 env:** after `./scripts/e2e-bootstrap.sh`, map variables from [deploy/e2e.generated.env](../../deploy/e2e.generated.env) into **`MLN_*`** (see that file’s `E2E_*` names); playbook is **[PHASE_12_E2E_CRUCIBLE.md](../../PHASE_12_E2E_CRUCIBLE.md)**. Use the **local** relay URL (`ws://127.0.0.1:7080/`), not only a public relay.
- **E2E troubleshooting:** host **8545** must be free for Compose Anvil. If **`mlnd`** exits with **`no such column: swap_id`** (or similar), wipe maker volumes **`deploy_mln_e2e_maker{1,2,3}`**, bring the stack up, **re-run bootstrap**, then **`--profile makers`**. Doc placeholders like **`/path/to/mwixnet-litvm`** mean **your clone path** or “already `cd`’d into the repo”.

## Verification

- `cd mln-cli && go test ./...`
- **Unit / manual (env only):** `make build-mln-cli`, set **`MLN_*`**, then **`route build`** and **`forger -route-json … -dry-run=true`**
- **Closed-loop local (Phase 12):** [PHASE_12_E2E_CRUCIBLE.md](../../PHASE_12_E2E_CRUCIBLE.md) — Compose + bootstrap + makers → **`scout`** (3 verified) → **`route build`** → **`forger`** **`dry-run`** and optional **`dry-run=false`** against **`mln-sidecar`** mock on **`http://127.0.0.1:8080`**

## Layer boundary

Nostr = discovery; LitVM = stake and exit truth; do not blur MWEB vs LitVM responsibilities.
