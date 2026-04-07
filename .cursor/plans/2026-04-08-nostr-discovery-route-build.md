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
- **Env:** [PHASE_10_TAKER_CLI.md](../../PHASE_10_TAKER_CLI.md) — need real **`MLN_REGISTRY_ADDR`** (40 hex chars), not placeholders; **`MLN_LITVM_HTTP_URL`**, **`MLN_LITVM_CHAIN_ID`**, and **`MLN_NOSTR_RELAYS`** or **`MLN_NOSTR_RELAY_URL`**.
- **`scout` / `registry address mismatch`:** each ad’s **`content.litvm.registry`** must equal **`MLN_REGISTRY_ADDR`**. Public relay ads target their deployed registry; local Anvil addresses will not match unless you publish matching ads (E2E / local `mlnd`).
- **`pathfind: need at least 3 verified makers with Tor endpoints, got 0`:** expected if no ads pass chain + registry + LitVM + Tor-endpoint filters for your env.

## Verification

- `cd mln-cli && go test ./...`
- `make build-mln-cli && ./bin/mln-cli route build` (with env set) then **`forger -route-json ... -dry-run=true`** as needed

## Layer boundary

Nostr = discovery; LitVM = stake and exit truth; do not blur MWEB vs LitVM responsibilities.
