# Phase 12: E2E Crucible (local simulation)

This phase adds a **self-contained Docker stack** for closed-loop testing: **Anvil** (LitVM stand-in), a local **Nostr relay**, and **three `mlnd` maker daemons** that publish kind **31250** maker ads. A **bootstrap script** deploys [`MwixnetRegistry`](contracts/src/MwixnetRegistry.sol) and [`GrievanceCourt`](contracts/src/GrievanceCourt.sol), funds makers, and calls **`deposit()`** plus **`registerMaker(bytes32 nostrKeyHash)`** with the binding from [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md).

Normative scout path for the taker: [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md). Desktop wallet: [`mln-cli/desktop/README.md`](mln-cli/desktop/README.md).

## What you get

- **Ledger:** JSON-RPC/WebSocket on `http://127.0.0.1:8545` and `ws://127.0.0.1:8545` (Anvil).
- **Relay:** WebSocket to the relay at **`ws://127.0.0.1:7080/`** (host maps `7080` → container `8080` for `scsibug/nostr-rs-relay`).
- **Sidecar:** **`mln-sidecar`** on **`http://127.0.0.1:8080`** — mock MLN HTTP service (`-mode=mock` in [`deploy/docker-compose.e2e.yml`](deploy/docker-compose.e2e.yml)) implementing **`GET /v1/balance`** and **`POST /v1/swap`** ([`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md)) so the Wails wallet and `mln-cli forger` can complete the local loop without `coinswapd`.
- **Makers:** Three containers (`maker1`, `maker2`, `maker3`) with distinct operator keys and `MLND_TOR_ONION` URLs (`http://127.0.0.1:8081` … `8083`) so the taker forger sees non-empty hop endpoints (cleartext is acceptable for this local matrix).

## Prerequisites

- **Docker** and **Docker Compose** v2.
- **`jq`** (`brew install jq` / `apt install jq`).
- **`ghcr.io/foundry-rs/foundry:latest`** (pulled implicitly by [`scripts/e2e-bootstrap.sh`](scripts/e2e-bootstrap.sh), same as [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh)).

On **Apple Silicon**, `scsibug/nostr-rs-relay` may only publish an `linux/amd64` image; Docker will run it under emulation (you may see a platform warning). That is normal for this dev stack.

Optional: **`python3`** + `pip install -r scripts/requirements.txt` if you use [`scripts/e2e_nostr_key_hash.py`](scripts/e2e_nostr_key_hash.py) to recompute `nostrKeyHash` for new test keys.

## 1. Start Anvil and the relay

From the repo root:

```bash
docker compose -f deploy/docker-compose.e2e.yml up -d
```

This starts **`anvil`**, **`nostr`**, and **`mln-sidecar`** (port **8080**). Maker services use the Compose **`makers`** profile so they do not start before env files exist.

## 2. Bootstrap contracts and makers

With Anvil listening on `http://127.0.0.1:8545` (default):

```bash
./scripts/e2e-bootstrap.sh
```

The script:

1. Waits for RPC.
2. Runs `forge script script/Deploy.s.sol:Deploy` inside the Foundry image (same as `deploy-local-anvil.sh`).
3. Reads `contracts/broadcast/Deploy.s.sol/<chainId>/run-latest.json` for **MwixnetRegistry** and **GrievanceCourt** addresses (that directory is gitignored but is written on each deploy).
4. For each of three fixed test makers: **`deposit()`** with `0.11 ether` (above default `minStake` of `0.1 ether`), then **`registerMaker(bytes32)`** with `keccak256(x-only Nostr pubkey)`.

It writes (all under **`deploy/`**, gitignored except the example JSON):

| File | Purpose |
|------|---------|
| `e2e.generated.env` | Contract addresses and URLs for your notes |
| `e2e.maker1.env` … `e2e.maker3.env` | Full `MLND_*` env for each Compose service |
| `e2e.wallet-settings.generated.json` | Taker **network settings** JSON aligned with [`NetworkSettings`](mln-cli/internal/config/settings.go) |

**Fresh chain:** If Anvil already had a prior deploy (different nonces), addresses in `run-latest.json` change; re-run bootstrap and use the new generated files. For a clean slate: `docker compose -f deploy/docker-compose.e2e.yml down -v` and bring the stack up again.

## 3. Start the three makers

```bash
docker compose -f deploy/docker-compose.e2e.yml --profile makers up -d --build
```

`docker logs deploy-maker1-1` (and `maker2`, `maker3`) should show **Nostr maker-ad broadcaster enabled** and **published kind=31250**.

## 4. Point the Wails desktop wallet at the stack

The MLN Wallet does **not** read `.env` or `.env.local`. It persists **[`NetworkSettings`](mln-cli/internal/config/settings.go)** in the OS app config directory (e.g. on macOS, `~/Library/Application Support/mln-wallet/settings.json`).

After bootstrap, open **`deploy/e2e.wallet-settings.generated.json`** and either:

- Copy the fields into the app **Network** form (relays, LitVM HTTP URL, chain id, registry, grievance court), or  
- Merge the JSON into your existing `settings.json` (keep valid JSON; backup first).

Typical values (host → Docker):

- **Nostr relays:** `ws://127.0.0.1:7080/`
- **LitVM HTTP URL:** `http://127.0.0.1:8545`
- **Chain id:** `31337`
- **Registry / court:** from `e2e.generated.env` or the wallet JSON

A blank template lives at [`deploy/e2e-wallet-settings.example.json`](deploy/e2e-wallet-settings.example.json).

Then build and run the desktop app (`make build-mln-wallet` and run the binary, or `wails dev` per [`mln-cli/desktop/README.md`](mln-cli/desktop/README.md)). Use **Scout**; you should see **three verified makers** in the table.

## 5. “Send Privately” and the sidecar

**Scout** and **Build route** exercise Nostr + LitVM verification.

**Send Privately** POSTs the built route to the default MLN sidecar URL (`http://127.0.0.1:8080/v1/swap`). With **`mln-sidecar`** from the Compose file, that service answers **`GET /v1/balance`** (mock **1.25 LTC** available / **1.2 LTC** spendable) and **`POST /v1/swap`** (validates the three-hop JSON, logs a simulated MWEB onion handoff, returns success). The desktop wallet can therefore run a **closed-loop** local simulation: balance panel, spendable check, and submit all succeed without a real `coinswapd`. For production, deploy the same binary (or a fork) next to **`coinswapd`** to translate route JSON into **`swap_Swap(onion.Onion)`** JSON-RPC per [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md). The hop URLs in the route (`8081`–`8083` in this matrix) are what makers advertise for the engine path; the wallet does not dial them directly in the Phase 10 forger flow.

## Security note

`e2e-bootstrap.sh` uses **well-known Anvil keys** and fixed Nostr test secrets. They are for **local development only**; never use them on a public network or mainnet.

## Related scripts

- [`scripts/e2e_nostr_key_hash.py`](scripts/e2e_nostr_key_hash.py) — print `nostrKeyHash` for a 64-char hex Nostr secret (for updating bootstrap constants if you change keys).
- [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh) — deploy only (no maker registration).
