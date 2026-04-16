# Live 3-Maker / 1-Taker MWEB CoinSwap — Tiered Review and Operator Setup

## Goal

Give operators a concrete, tiered path from the local stub lab to a full funded mainnet MWEB CoinSwap across three independent hosts, and ship the ergonomic glue (Makefile targets, doctor command, per-box setup) so a new contributor can execute each tier without rereading every phase playbook.

## In scope

- Tier 1 single-host stub lab (Docker Compose + `mw-rpc-stub`).
- Tier 2 Tor multi-host + LitVM 4441 (no funds): curated relay, three onion makers, `mln-cli doctor` as the discovery gate.
- Tier 3 funded mainnet MWEB run: real `coinswapd-research` on each maker, funded taker, `make phase3-funded-preflight` as the pre-submit gate, live `swap_forward` / `swap_backward` with `pendingOnions == 0` (no dev-clear).
- Operator box setup for Apple Silicon Macs (one relay/maker host plus two additional maker boxes).
- Persisting the plan so handoffs across machines survive.

## Out of scope

- LitVM contract changes — Phase 15 is shipped; chain 4441 addresses are stable.
- New Nostr kinds or v2 sealed reachability ads (future work per `.cursor/plans/2026-04-15-nip42-auth-relay-hardening.md` "Out of scope").
- Upstreaming `mweb_*` to `ltcmweb/coinswapd` — tracked fork `research/coinswapd/` is canonical.
- Rewriting the 2026-04-15 `LIVE_COINSWAP_ATTEMPT` file; future attempts append a new dated record.

## Primary files and canonical docs

| Area | Files |
|------|-------|
| Roadmap + phase table | [`README.md`](../../README.md), [`AGENTS.md`](../../AGENTS.md) |
| Tier 1 stub lab | [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../../PHASE_3_MWEB_HANDOFF_SLICE.md), [`scripts/e2e-mweb-handoff-stub.sh`](../../scripts/e2e-mweb-handoff-stub.sh), [`scripts/e2e-bootstrap.sh`](../../scripts/e2e-bootstrap.sh), [`deploy/docker-compose.e2e.yml`](../../deploy/docker-compose.e2e.yml) |
| Tier 2 discovery + Tor | [`research/PHASE_3_TIER2_RELAY.md`](../../research/PHASE_3_TIER2_RELAY.md), [`research/PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md`](../../research/PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md), [`research/PHASE_3_TOR_OPERATOR_LAB.md`](../../research/PHASE_3_TOR_OPERATOR_LAB.md), [`scripts/phase3-tier2-setup.sh`](../../scripts/phase3-tier2-setup.sh) |
| Tier 3 funded | [`research/PHASE_3_OPERATOR_CHECKLIST.md`](../../research/PHASE_3_OPERATOR_CHECKLIST.md) §D, [`scripts/phase3-funded-env-check.sh`](../../scripts/phase3-funded-env-check.sh) |
| Operator box setup | [`deploy/MAKER_BOX_SETUP.md`](../../deploy/MAKER_BOX_SETUP.md), [`deploy/tier2.maker-box.env.example`](../../deploy/tier2.maker-box.env.example), [`scripts/maker-box-up.sh`](../../scripts/maker-box-up.sh) |
| Wallet demo | [`mln-cli/desktop/app.go`](../../mln-cli/desktop/app.go) (`RunLocalLab`), [`mln-cli/desktop/frontend/src/App.jsx`](../../mln-cli/desktop/frontend/src/App.jsx) "Run local lab" button |
| Preflight / diagnostics | [`mln-cli/cmd/mln-cli/doctor.go`](../../mln-cli/cmd/mln-cli/doctor.go) (`mln-cli doctor`), [`scripts/e2e-status.sh`](../../scripts/e2e-status.sh) |
| Live attempt log | [`LIVE_COINSWAP_ATTEMPT_2026-04-15.md`](../../LIVE_COINSWAP_ATTEMPT_2026-04-15.md) |

## Current status (captured 2026-04-16)

- **LitVM judicial layer**: Chain 4441 live. Registry `0x01bD…FfE7`, Court `0xc303…2593` ([`deploy/litvm-addresses.generated.env`](../../deploy/litvm-addresses.generated.env)).
- **Phase 3a stub handoff**: COMPLETE. `make e2e-tier1` runs scout → pathfind → forger → `mweb_*` against `mw-rpc-stub`; `pendingOnions` returns to 0.
- **`coinswapd-research` fork**: builds; `mweb_getBalance` smoke OK; `E2E_MWEB_FUNDED=1` works with dev-clear only.
- **Live attempt on record (2026-04-15)**: blocked at discovery (`mln-cli scout` returned `(no verified makers)` on `wss://relay.damus.io` with `chainId mismatch`). `mln-core` "LIVE_COINSWAP_ATTEMPT" gate satisfied.
- **Gaps vs README Phase 3**: (1) curated relay carrying ads pinned to chain 4441; (2) three makers registered on chain 4441 with non-empty Tor endpoints; (3) live multi-hop finalize (`pendingOnions==0` without dev-clear); (4) funded taker with exact-UTXO coin per `pickCoinExactAmount`.

## Tiered progression

### Tier 1 — single-host stub lab

```bash
make build-mln-cli build-mln-sidecar build-mw-rpc-stub
make e2e-tier1        # E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh
make e2e-status       # optional dashboard
```

Proves control plane; no Tor, no money.

### Tier 2 — three hosts on LitVM 4441

- One curated `nostr-rs-relay` reachable by all makers and the taker (see [`research/PHASE_3_TIER2_RELAY.md`](../../research/PHASE_3_TIER2_RELAY.md)).
- Each maker host: `tor` hidden service + `coinswapd-research` + `mlnd`; uses [`deploy/tier2.maker-box.env.example`](../../deploy/tier2.maker-box.env.example) and [`scripts/maker-box-up.sh`](../../scripts/maker-box-up.sh).
- Taker host: `make phase3-operator-preflight` then `./bin/mln-cli doctor` — the gate is `verified=3, with tor=3`.

### Tier 3 — funded mainnet MWEB

- `make phase3-funded-preflight` must pass (calls `mweb_getBalance`, checks `spendableSat >= E2E_MWEB_AMOUNT`, warns about exact-UTXO precondition).
- Run `mln-cli forger -trigger-batch -wait-batch` via `mln-sidecar -mode=rpc` on top of Tier 2.
- README Phase 3 bar: `pendingOnions==0` without any dev-clear flag.
- Record outcome in a new `LIVE_COINSWAP_ATTEMPT_YYYY-MM-DD.md`; do not overwrite 2026-04-15.

## Operator topology (three Apple Silicon Macs)

| Host | Network | Roles |
|------|---------|-------|
| M4 mini | Router A | Nostr relay + Maker 3 |
| M1 mini | Router B | Maker 2 |
| MacBook Pro | Router B | Maker 1 + taker + repo dev box |

Three distinct IPs, three operators on chain 4441, three `.onion` endpoints. Caveat: MBP colocates taker with Maker 1 — fine for bring-up, not a rigorous anonymity test against the first hop.

## Verification

- `cd mln-cli && go test ./...` and `go build -tags=wails ./desktop/` green after implementation.
- `bash -n scripts/e2e-status.sh scripts/phase3-tier2-setup.sh scripts/phase3-funded-env-check.sh scripts/maker-box-up.sh` clean.
- Tier 1: `make e2e-tier1` exits 0; `make e2e-status` shows `verified >= 3`.
- Tier 2: `./bin/mln-cli doctor` shows `verified=3, with tor=3` with relay + chain 4441 env.
- Tier 3: `make phase3-funded-preflight` exits 0 or 2; `forger -wait-batch` observes `pendingOnions==0`.

Further QA expectations per [`.cursor/skills/mln-qa/SKILL.md`](../skills/mln-qa/SKILL.md).

## Layer-boundary check

All tiers respect MLN separation of concerns:

- **MWEB** — privacy engine and per-hop fee path; Tier 3 adds real finalize / broadcast. No EVM fee duplication.
- **LitVM (chain 4441)** — consulted only for stake verification and maker registration. Happy-path mix is never put on-chain.
- **Nostr** — discovery only. The curated relay is not an authoritative stake source; `mln-cli scout` cross-checks each ad against `MwixnetRegistry`.
- **Tor** — transport only. Does not replace MWEB onion crypto inside `coinswapd-research`.
