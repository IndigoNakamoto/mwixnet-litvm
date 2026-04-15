# Phase 3 — Operator checklist (real Tor, multi-hop)

Single-page sequence for advancing **README Phase 3** toward live operators.

**Prefer a linear, terminal-by-terminal walkthrough?** Start with [PHASE_3_OPERATOR_PLAYBOOK.md](PHASE_3_OPERATOR_PLAYBOOK.md). Deep dives: [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md), [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md), [PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md).

## North star (README checkbox)

Do **not** mark README Phase 3 complete until **both**:

1. **Live `.onion` multi-hop** reaches **`pendingOnions == 0` without** `-mweb-dev-clear-pending-after-batch`** (real finalize / broadcast after P2P **`swap_forward` / `swap_backward`**).
2. **Public LitVM** registry/court is deployed and operators align grievance flows per [PHASE_16_PUBLIC_TESTNET.md](../PHASE_16_PUBLIC_TESTNET.md) when official RPC exists.

Until then, keep **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** and optional **`MWEB_RPC_BACKEND=coinswapd`** smoke green after any handoff change ([PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md)).

**Quick preflight:** `make phase3-operator-preflight` (Tor SOCKS + printed `HTTP_PROXY` / `NO_PROXY` template + optional onion JSON-RPC ping).

---

## A — Tor and `coinswapd` proxy (every hop process)

- [ ] Run **`./scripts/tor-preflight.sh`** or **`make tor-preflight`** (9150 for Tor Browser if needed).
- [ ] On **each `coinswapd-research` that dials `.onion` peers**, export before start:

```bash
export HTTP_PROXY="socks5h://127.0.0.1:9050"   # host/port match your SOCKS
export HTTPS_PROXY="$HTTP_PROXY"                # if any hop URL uses https://
export NO_PROXY="127.0.0.1,localhost"
```

**Why:** go-ethereum **`rpc.Dial`** uses **`net/http`**; **`ProxyFromEnvironment`** reads **`HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`** — **not** **`ALL_PROXY`** ([PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md)).

- [ ] **1-hop sanity:** With the same exports, prove HTTP JSON-RPC to **one** maker `.onion` (e.g. set **`PHASE3_ONION_JSONRPC_URL`** and run **`make phase3-operator-preflight`**, or curl manually with **`--socks5-hostname`** as in the Tor lab doc).

---

## B — Topology: MLN taker vs mesh maker

- [ ] **Taker** (route from **`mweb_submitRoute`**, dynamic peers): run **`coinswapd-research`** with **`-mln-local-taker`** so startup does not depend on **`getNodes()` / `AliveNodes`** mesh match.
- [ ] **Listed mesh maker:** omit **`-mln-local-taker`**; **`-k`** must match your row in the probed topology ([PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md)).

---

## C — Three makers: hidden service + Nostr + LitVM ads

For **each** of three makers:

- [ ] **Tor hidden service** exposes maker JSON-RPC (port you publish in the ad, e.g. `8334`).
- [ ] **`mlnd`** env: **`MLND_WS_URL`**, **`MLND_COURT_ADDR`**, **`MLND_REGISTRY_ADDR`**, **`MLND_OPERATOR_ADDR`**, **`MLND_LITVM_CHAIN_ID`**, **`MLND_NOSTR_RELAYS`**, **`MLND_NOSTR_NSEC`** (if publishing ads).
- [ ] **`MLND_TOR_ONION`** (or equivalent) so kind **31250** content carries the **same** Tor **HTTP** URL takers need (scheme + host + port). **`mln-cli pathfind`** requires **non-empty `tor`** on verified makers.
- [ ] Optional: **`MLND_SWAP_X25519_PUB_HEX`** so routes include **`swapX25519PubHex`** per hop ([PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md)).
- [ ] Makers registered on the **same** registry the taker Scout/pathfind uses (local Anvil for lab, or public LitVM when deployed).

Template grouping: [`.env.compose.example`](../.env.compose.example).

---

## D — Taker funded path (production-shaped completion)

- [ ] Neutrino synced; **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, **`-a`** fee MWEB (**`ltcmweb1…`** full address).
- [ ] **`mln-sidecar -mode=rpc`** → taker **`coinswapd`** JSON-RPC (see port layout in [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md)); proxy on **taker `coinswapd`**, not sidecar, unless **`-rpc-url`** is onion.
- [ ] **`mln-cli pathfind`** → **`route.json`** with three hops, operators, `tor`, keys.
- [ ] **`mln-cli forger`** … **`-trigger-batch -wait-batch`** against sidecar **`/v1/swap`** / **`/v1/route/*`**.

**Success bar:** **`mweb_getRouteStatus`** → **`pendingOnions == 0` without** **`E2E_MWEB_FUNDED_DEV_CLEAR`** or **`-mweb-dev-clear-pending-after-batch`**.

Optional: **`./scripts/phase3-funded-env-check.sh`** before forger (warns if dev-clear env is set when you claim a production-shaped run).

---

## E — Maker evidence bridge (parallel)

- [ ] Patched **`coinswapd`** NDJSON receipts + **`MLND_BRIDGE_RECEIPTS_DIR`** shared mount ([PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md)).
- [ ] **`MLND_DEFEND_AUTO`** / key hygiene only after correlators and **LitVM** addresses are correct; see [THREAT_MODEL_MLN.md](THREAT_MODEL_MLN.md).

---

## F — When LitVM public RPC exists (README Phase 3, second half)

Follow [PHASE_16_PUBLIC_TESTNET.md](../PHASE_16_PUBLIC_TESTNET.md) section 0: **`make broadcast-litvm`**, **`make record-litvm-deploy`**, merge generated env into **`deploy/.env.testnet`**, **`docker compose -f deploy/docker-compose.testnet.yml`**, then point **`mlnd` / `mln-cli`** at published registry/court RPC off docs — **do not guess** URL or chain ID.

---

## Failure triage (no secret logging)

| Symptom | Check |
|--------|--------|
| Dial timeout to `.onion` | Tor, **`HTTP_PROXY`**, HS port, firewall |
| **`swap_forward:`** / **`swap_backward:`** errors | Peer URLs, **`-mln-local-taker`** vs mesh, logs ([swap.go](coinswapd/swap.go)) |
| Stuck **`pendingOnions`** | Peer reachability vs missing finalize — see [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md) |

Do not paste live onions, keys, or payloads into public tickets.
