# Changelog

All notable changes to this project are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **LitVM testnet (chain 4441):** Deployed `MwixnetRegistry` + `GrievanceCourt`; public RPC/WS and addresses in [`README.md`](README.md); committed [`deploy/litvm-addresses.generated.env`](deploy/litvm-addresses.generated.env) for `mlnd` / taker env merge — 2026-04-15.
- **QA:** Ran permanent regression anchors (`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`, `python3 nostr/validate_fixtures.py`, `python3 nostr/check_wire_helpers.py`) — 2026-04-03.
- **Real funded MWEB handoff:** `E2E_MWEB_FUNDED=1` script path, bootstrap `MLND_SWAP_X25519_PUB_HEX` for `swapX25519PubHex` in routes, and `coinswapd-research -mweb-dev-clear-pending-after-batch` (DEV ONLY) for operator `pendingOnions=0` smoke — [`PHASE_3_MWEB_HANDOFF_SLICE.md`](PHASE_3_MWEB_HANDOFF_SLICE.md), [`research/COINSWAPD_MLN_FORK_SPEC.md`](research/COINSWAPD_MLN_FORK_SPEC.md) §2.7a.
- **QA:** Re-ran `python3 nostr/validate_fixtures.py` and `python3 nostr/check_wire_helpers.py` (Nostr wire hygiene; mln-pm quick win).
- **Celebrate:** Full in-repo **MWEB completed swap round-trip** (route submit → **`mweb_runBatch`** → **`pendingOnions=0`**) verified with **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** — same operator knobs apply to **`coinswapd-research`** per Phase 3 runbook.
- **MWEB completed-swap operator path:** `research/coinswapd` **`mweb_getRouteStatus`** / **`mweb_runBatch`**, DB cleanup after **`finalize`**, **`mln-sidecar`** **`GET /v1/route/status`** + **`POST /v1/route/batch`**, **`mln-cli forger`** **`-trigger-batch` / `-wait-batch`**, **`mw-rpc-stub`** virtual pending queue; see [`PHASE_3_MWEB_HANDOFF_SLICE.md`](PHASE_3_MWEB_HANDOFF_SLICE.md) and [`research/COINSWAPD_MLN_FORK_SPEC.md`](research/COINSWAPD_MLN_FORK_SPEC.md).
- **QA:** Ran `python3 nostr/validate_fixtures.py` and `python3 nostr/check_wire_helpers.py` (Nostr wire hygiene before release-candidate workflow).
- **Phase 3 integration slice:** `make build-research-coinswapd`, optional `MWEB_RPC_BACKEND=coinswapd` in `scripts/e2e-mweb-handoff-stub.sh`, Tor-shaped hop URL normalization (`mln-sidecar`, `mln-cli` forger) and pathfind requires non-empty maker Tor — see `PHASE_3_MWEB_HANDOFF_SLICE.md`.
- **`coinswapd-research` backend smoke:** operator-verified `MWEB_RPC_BACKEND=coinswapd` handoff (`mweb_getBalance` OK via sidecar; stub `POST /v1/swap` → expected 502); documented in `PHASE_3_MWEB_HANDOFF_SLICE.md`.
- **`research/coinswapd`:** `-mln-local-taker` skips `getNodes` / mesh pubkey match for MLN RPC + E2E; `scripts/e2e-mweb-handoff-stub.sh` passes it in `MWEB_RPC_BACKEND=coinswapd` mode.
- **Pre-positioned LitVM testnet broadcast path (Phase 16):** `make broadcast-litvm`, `make record-litvm-deploy`, [`scripts/broadcast-litvm-testnet.sh`](scripts/broadcast-litvm-testnet.sh), [`scripts/record-litvm-deploy.py`](scripts/record-litvm-deploy.py); runbook section 0 in [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md).
- **Phase 2 Nostr wire v1 closed loop:** [`PHASE_2_NOSTR.md`](PHASE_2_NOSTR.md), minimal maker-ad fixture, [`nostr/check_wire_helpers.py`](nostr/check_wire_helpers.py) + CI; [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md) marked normative v1.
- **Phase 3a E2E MWEB handoff GREEN:** `E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh` → `Phase 3a stub handoff checks passed.` ([`PHASE_3_MWEB_HANDOFF_SLICE.md`](PHASE_3_MWEB_HANDOFF_SLICE.md)).
- Added full Cursor AI agent team (6 new skills + 6 new rules + architecture diagrams skill + AGENTS.md update).
- **`mln-cli maker onboard`**: Bundle-signing for LitVM `deposit` + `registerMaker`; dry-run by default; `-execute`, `-force-reregister`; `nostrKeyHash` from hex, **`npub1…`**, or `nsec`. See [PHASE_10_TAKER_CLI.md](PHASE_10_TAKER_CLI.md) Phase 10.4.
- **`mlnd` loopback Maker dashboard** (when `MLND_DASHBOARD_ADDR` is set): read-only UI at **`http://127.0.0.1:9842/`** (or your chosen host:port), JSON status, SSE + `opslog` for operator narrative (LitVM / Nostr / MWEB). *Landed on `main` before Phase B CLI; listed here for the next release tag.*
- **Maker operator blueprint**: [research/WALLET_MAKER_FLOW_V1.md](research/WALLET_MAKER_FLOW_V1.md) — *Implementation gap: today vs. target* (A→B→C roadmap).

### Changed

- **Docs:** [`PHASE_3_MWEB_HANDOFF_SLICE.md`](PHASE_3_MWEB_HANDOFF_SLICE.md) + [`README.md`](README.md) permanent regression-anchor note for PRs touching sidecar/forger/coinswapd; [`contracts/.env.example`](contracts/.env.example) / [`deploy/.env.testnet.example`](deploy/.env.testnet.example) cross-refs to [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md) section 0; [`AGENTS.md`](AGENTS.md) Phase 3a table row.
- **`scripts/e2e-mweb-handoff-stub.sh`:** **`E2E_MWEB_FULL=1`** path runs **`mln-cli forger`** with **`-trigger-batch -wait-batch`** (sidecar **`/v1/route/*`** + stub **`mweb_runBatch`**).
- **Docs:** Release-candidate Phase 3a regression checklist in [`PHASE_3_MWEB_HANDOFF_SLICE.md`](PHASE_3_MWEB_HANDOFF_SLICE.md) and [`README.md`](README.md); [`contracts/.env.example`](contracts/.env.example) / [`deploy/.env.testnet.example`](deploy/.env.testnet.example) aligned with `make broadcast-litvm` / `make record-litvm-deploy` and [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md) section 0.
- **`mlnd`**: Reject placeholder / zero registry–court–operator addresses; relays without `MLND_NOSTR_NSEC` → read-only (no publish). Optional **`MLND_FEE_MIN_SAT` / `MLND_FEE_MAX_SAT`**, **`MLND_SWAP_X25519_PUB_HEX`** in maker ads.
- **Docs**: `PHASE_10_TAKER_CLI.md` (Phase 10.4), `WALLET_MAKER_FLOW_V1.md` (Phase B shipped), `mlnd/README.md` (pointer to onboard CLI).

### Fixed

- **`.gitignore`**: `/mlnd.db` at repo root.
