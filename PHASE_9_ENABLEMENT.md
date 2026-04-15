# Phase 9: Enablement and Hardening (Operator Packaging)

This document is the operational guide for deploying an MLN maker node: the administrative daemon (`mlnd`), the MWEB privacy engine (`coinswapd` when you run it), Nostr discovery, and LitVM. **Do not guess** RPC URLs, chain IDs, or contract addresses; copy them from [official LitVM documentation](https://docs.litvm.com/) and [`research/LITVM.md`](research/LITVM.md). For live `.onion` multi-hop sequencing (Phase 3 operators), see [`research/PHASE_3_OPERATOR_CHECKLIST.md`](research/PHASE_3_OPERATOR_CHECKLIST.md).

## 1. Quick start (Docker Compose)

Recommended layout: Docker Compose so `mlnd` has stable volumes for SQLite and (optionally) a shared receipt directory for a patched `coinswapd`.

1. Clone this repository on the host.
2. Create local data dirs (example): `mkdir -p data/mlnd data/receipts`
3. Copy the environment template: `cp .env.compose.example .env` (**required** — [`docker-compose.yml`](docker-compose.yml) loads `.env`, which is gitignored).
4. Edit `.env` with your keys and LitVM endpoints (placeholders only in the example file).
5. Start: `docker compose up -d`

For per-variable semantics and non-Docker runs, see [`mlnd/README.md`](mlnd/README.md). For HTTP RPC smoke checks (e.g. before going live), see `make testnet-smoke` in the root [`Makefile`](Makefile) and [`PHASE_8_TESTNET_RELEASE.md`](PHASE_8_TESTNET_RELEASE.md).

## 2. Paired deployment architecture (NDJSON bridge)

Two processes are involved in a full maker stack:

1. **`coinswapd` (patched):** MWEB / Tor / hop handling. Stock `coinswapd` does not emit the receipt stream this repo expects.
2. **`mlnd`:** LitVM log watching, SQLite receipt vault, optional Nostr maker ads (kind **31250**), optional auto-`defendGrievance`.

**Do not pipe `coinswapd` stdout into `mlnd`.** The [`mlnd/internal/bridge`](mlnd/internal/bridge) implementation scans a directory for `*.ndjson` / `*.jsonl` and ingests **complete lines** into SQLite (`MLND_BRIDGE_RECEIPTS_DIR`).

Operational pattern:

- Apply or maintain the patch described in [`research/COINSWAPD_INTEGRATION.md`](research/COINSWAPD_INTEGRATION.md) (see also [`research/coinswapd-receipt-ndjson.patch`](research/coinswapd-receipt-ndjson.patch)).
- Configure the fork to append one JSON object per line under a shared path (e.g. `/receipts` in containers).
- Run `mlnd` with `MLND_BRIDGE_COINSWAPD=1` and `MLND_BRIDGE_RECEIPTS_DIR` pointing at that same path.

In [`docker-compose.yml`](docker-compose.yml), mount e.g. `./data/receipts:/receipts` on `mlnd`. When you have a patched `coinswapd` image, mount the **same** host path into that service so both processes share the directory.

## 3. Defense operations playbook

If a grievance is opened against your operator on LitVM, `mlnd` can submit `defendGrievance` when **`MLND_DEFEND_AUTO`** is truthy (`1`, `true`, or `yes`) and a matching receipt exists in SQLite.

### Key management and gas

- **Hot key:** Set **`MLND_OPERATOR_PRIVATE_KEY`** (64 hex characters, optional `0x` prefix). The derived address **must** match **`MLND_OPERATOR_ADDR`** (the accused maker on-chain).
- **Gas budgeting:** Treat this key as a **low-balance operational key**. `defendGrievance` is a normal contract call; keep only a small float (e.g. enough for several defenses), monitor balance, and top up as needed. Do not park main holdings on this key.
- **Dry run:** Set **`MLND_DEFEND_DRY_RUN`** to `1` / `true` / `yes` while validating setup. `mlnd` will build **`defenseData`** and log a **DRY-RUN** line with the calldata bytes; it **does not** broadcast a transaction (no gas estimate is printed in that path today).
- **Nonce management:** Run **one** `mlnd` process per `MLND_OPERATOR_PRIVATE_KEY`. Multiple instances will contend for the same on-chain nonce and can fail against the LitVM RPC.

### Nostr discovery pulse

Maker ads are **optional** but required for takers to discover you via relays. Set **`MLND_NOSTR_RELAYS`** (comma-separated `wss://` URLs) and **`MLND_NOSTR_NSEC`** (or hex), plus registry/court/operator/chain env as in [`mlnd/README.md`](mlnd/README.md). Republish interval defaults to **30m** (`MLND_NOSTR_INTERVAL`).

## Completion (shipped in repo)

- [x] [`PHASE_9_ENABLEMENT.md`](PHASE_9_ENABLEMENT.md) (this playbook)
- [x] [`docker-compose.yml`](docker-compose.yml) (`mlnd` service + commented `coinswapd` stub)
- [x] [`.env.compose.example`](.env.compose.example) (grouped operator template)
- [x] Root [`README.md`](README.md) deployment section
