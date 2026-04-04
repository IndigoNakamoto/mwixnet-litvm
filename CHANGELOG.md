# Changelog

All notable changes to this project are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **Pre-positioned LitVM testnet broadcast path (Phase 16):** `make broadcast-litvm`, `make record-litvm-deploy`, [`scripts/broadcast-litvm-testnet.sh`](scripts/broadcast-litvm-testnet.sh), [`scripts/record-litvm-deploy.py`](scripts/record-litvm-deploy.py); runbook section 0 in [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md).
- **Phase 2 Nostr wire v1 closed loop:** [`PHASE_2_NOSTR.md`](PHASE_2_NOSTR.md), minimal maker-ad fixture, [`nostr/check_wire_helpers.py`](nostr/check_wire_helpers.py) + CI; [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md) marked normative v1.
- **Phase 3a E2E MWEB handoff GREEN:** `E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh` → `Phase 3a stub handoff checks passed.` ([`PHASE_3_MWEB_HANDOFF_SLICE.md`](PHASE_3_MWEB_HANDOFF_SLICE.md)).
- Added full Cursor AI agent team (6 new skills + 6 new rules + architecture diagrams skill + AGENTS.md update).
- **`mln-cli maker onboard`**: Bundle-signing for LitVM `deposit` + `registerMaker`; dry-run by default; `-execute`, `-force-reregister`; `nostrKeyHash` from hex, **`npub1…`**, or `nsec`. See [PHASE_10_TAKER_CLI.md](PHASE_10_TAKER_CLI.md) Phase 10.4.
- **`mlnd` loopback Maker dashboard** (when `MLND_DASHBOARD_ADDR` is set): read-only UI at **`http://127.0.0.1:9842/`** (or your chosen host:port), JSON status, SSE + `opslog` for operator narrative (LitVM / Nostr / MWEB). *Landed on `main` before Phase B CLI; listed here for the next release tag.*
- **Maker operator blueprint**: [research/WALLET_MAKER_FLOW_V1.md](research/WALLET_MAKER_FLOW_V1.md) — *Implementation gap: today vs. target* (A→B→C roadmap).

### Changed

- **`mlnd`**: Reject placeholder / zero registry–court–operator addresses; relays without `MLND_NOSTR_NSEC` → read-only (no publish). Optional **`MLND_FEE_MIN_SAT` / `MLND_FEE_MAX_SAT`**, **`MLND_SWAP_X25519_PUB_HEX`** in maker ads.
- **Docs**: `PHASE_10_TAKER_CLI.md` (Phase 10.4), `WALLET_MAKER_FLOW_V1.md` (Phase B shipped), `mlnd/README.md` (pointer to onboard CLI).

### Fixed

- **`.gitignore`**: `/mlnd.db` at repo root.
