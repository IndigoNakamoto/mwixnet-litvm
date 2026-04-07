# Phase 12: E2E Crucible — completed run

## Goal

Prove the full local MLN matrix end-to-end: Anvil (LitVM stand-in), local Nostr relay, mock `mln-sidecar`, three `mlnd` makers, then `mln-cli` discovery → route → forger (with vault receipt) → `grievance file` → on-chain `Open` grievance and `stakeFrozen` on the accused maker. Canonical playbook: [`PHASE_12_E2E_CRUCIBLE.md`](../../PHASE_12_E2E_CRUCIBLE.md).

## In scope / out of scope

**In scope:** `deploy/docker-compose.e2e.yml` stack, `./scripts/e2e-bootstrap.sh`, maker profile, `MLN_*` env mapping from `deploy/e2e.generated.env`, `route build`, `forger` dry-run and mock POST with `-vault`, `grievance file` from vault, read-only verification via `cast` in Docker.

**Out of scope:** Full `./scripts/e2e-mweb-handoff-stub.sh` with RPC overlay (would replace sidecar mode on shared ports); real `coinswapd` multi-hop failure receipts; editing the external Cursor plan file under `~/.cursor/plans/`.

## Primary files and canonical docs

- [`deploy/docker-compose.e2e.yml`](../../deploy/docker-compose.e2e.yml) — `anvil`, `nostr`, `mln-sidecar` (mock), `maker{1,2,3}` profile.
- [`scripts/e2e-bootstrap.sh`](../../scripts/e2e-bootstrap.sh) — deploy + `deploy/e2e.generated.env`, maker env files, wallet settings JSON.
- [`PHASE_12_E2E_CRUCIBLE.md`](../../PHASE_12_E2E_CRUCIBLE.md), [`PHASE_10_TAKER_CLI.md`](../../PHASE_10_TAKER_CLI.md) — taker CLI and env vars.
- [`scripts/e2e-mweb-handoff-stub.sh`](../../scripts/e2e-mweb-handoff-stub.sh) — reference for `E2E_*` → `MLN_*` exports.
- [`mln-cli/cmd/mln-cli/main.go`](../../mln-cli/cmd/mln-cli/main.go) — `forger` / `grievance file` flags.
- [`mln-sidecar/internal/mweb/bridge.go`](../../mln-sidecar/internal/mweb/bridge.go) — `MockBridge` golden receipt when epoch/accuser metadata is present.

## Execution results

1. **Stack reset and infra** — `docker compose -f deploy/docker-compose.e2e.yml down -v` then `up -d` brought up Anvil, nostr-rs-relay (platform warning on Apple Silicon is expected), and mock `mln-sidecar` on host port 8080.

2. **Bootstrap** — `./scripts/e2e-bootstrap.sh` succeeded: `MwixnetRegistry` and `GrievanceCourt` deployed to chain id 31337, three makers deposited and `registerMaker`’d; artifacts written under `deploy/` (gitignored except examples — do not commit generated keys as secrets).

3. **Makers** — First `docker compose ... --profile makers up -d --build` failed with **`network ... not found`** (stale `deploy-maker*-1` containers referencing an old network). **Workaround:** `docker rm -f deploy-maker1-1 deploy-maker2-1 deploy-maker3-1` (names may vary), then `docker compose -f deploy/docker-compose.e2e.yml --profile makers up -d` again. Logs showed **Nostr maker-ad broadcaster** and **`published kind=31250`**.

4. **Taker env** — Sourcing `deploy/e2e.generated.env` is **not** enough: map to `MLN_*` as in the handoff script, e.g. `MLN_REGISTRY_ADDR` from `E2E_MWIXNET_REGISTRY`, `MLN_GRIEVANCE_COURT_ADDR` from `E2E_GRIEVANCE_COURT`, `MLN_LITVM_HTTP_URL` / `MLN_LITVM_CHAIN_ID`, `MLN_NOSTR_RELAYS` from `E2E_NOSTR_RELAY_WS`. Set `MLN_SCOUT_TIMEOUT=45s` if ads are slow.

5. **CLI** — `make build-mln-cli`; `./bin/mln-cli route build -out /tmp/route.json`; `forger -route-json /tmp/route.json -dry-run=true` OK. Real mock swap: `-dry-run=false` **requires** `-dest` and `-amount`; with `MLN_RECEIPT_EPOCH_ID`, `MLN_ACCUSER_ETH_KEY` (well-known **local-only** funded Anvil dev account — see bootstrap docs), and `-vault /tmp/taker-vault.db`, forger printed **`Receipt vault: swap_id=...`** and an `evidenceHash`.

6. **Grievance** — **`grievance file` requires flags before the positional `swap_id`** (Go `flag` stops at first non-flag). Correct shape:  
   `mln-cli grievance file -vault /tmp/taker-vault.db <swap_id>`  
   plus `MLN_LITVM_HTTP_URL`, `MLN_GRIEVANCE_COURT_ADDR`, accuser key. Broadcast succeeded (`txHash` emitted on stdout).

7. **On-chain checks** — Host had no `cast` on PATH; used Foundry image with **`docker run ... --entrypoint cast ... call`** and `host.docker.internal:8545`. `grievances(bytes32)` returned **`phase == 1` (`Open`)**; registry **`stakeFrozen(accused) == true`**.

8. **Optional RPC path** — `make build-mw-rpc-stub` verified the stub binary builds; full `e2e-mweb-handoff-stub.sh` not run in the same session to avoid port conflicts with the mock stack.

## Verification

- Makers: `docker logs deploy-maker1-1` (and 2, 3) — expect `published kind=31250`.
- Taker: `route build` produces 3-hop JSON; `forger -dry-run` lists Tor URLs; `forger -dry-run=false` with sidecar returns success and optional vault lines.
- Judicial: `grievance file` broadcasts; confirm grievance struct and `stakeFrozen` via `cast call` or explorer.
- Deeper automation: [`.cursor/skills/mln-qa/SKILL.md`](../skills/mln-qa/SKILL.md); compose details in [`PHASE_12_E2E_CRUCIBLE.md`](../../PHASE_12_E2E_CRUCIBLE.md).

## Layer-boundary check

**Nostr** used only for discovery (31250 ads); **LitVM** (Anvil) for registry reads, `openGrievance`, and `freezeStake`; **MWEB** path exercised via mock **sidecar** HTTP (not live hop crypto); **Tor** represented as cleartext localhost URLs in the dev matrix only — no blurring of stake truth onto Nostr or full mix verification onto the EVM.

## Follow-ups

- If `network not found` recurs after compose churn, remove stale `deploy-maker*` containers before `--profile makers up`.
- For RPC/stub failure and `mweb_getLastReceipt` behavior, run [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../../PHASE_3_MWEB_HANDOFF_SLICE.md) / `e2e-mweb-handoff-stub.sh` in a dedicated session (may need to stop the mock-only stack first).
