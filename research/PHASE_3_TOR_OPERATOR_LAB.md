# Phase 3 — Live Tor operator lab (coinswapd + MLN)

This note is the **operator runbook** for advancing from Phase 3a (cleartext / stub) toward README **Phase 3** (live `.onion`, multi-hop P2P). **Step-by-step terminals (long playbook):** [`PHASE_3_OPERATOR_PLAYBOOK.md`](PHASE_3_OPERATOR_PLAYBOOK.md). **Checklist:** [`PHASE_3_OPERATOR_CHECKLIST.md`](PHASE_3_OPERATOR_CHECKLIST.md). Canonical handoff wire remains [`COINSWAPD_MLN_FORK_SPEC.md`](COINSWAPD_MLN_FORK_SPEC.md) and [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../PHASE_3_MWEB_HANDOFF_SLICE.md).

## Preflight: Tor SOCKS

Before debugging `swap_forward` / MWEB, confirm Tor is usable:

```bash
./scripts/tor-preflight.sh
```

- Default SOCKS: `127.0.0.1:9050` (override with `TOR_SOCKS_HOST`, `TOR_SOCKS_PORT`; Tor Browser often uses **9150**).
- The script uses **`curl --socks5-hostname`** so the **hostname is resolved at the proxy** (required for `.onion`). That matches why we recommend **`socks5h://`** in proxy URLs below.
- Air-gapped / no curl: `TOR_PREFLIGHT_SKIP_CURL=1 ./scripts/tor-preflight.sh` (TCP-only).

Makefile: `make tor-preflight`.

## Proxy env for taker `coinswapd` (go-ethereum `rpc.Dial`)

Inter-hop calls use **`github.com/ethereum/go-ethereum/rpc.Dial`** with **`http://` or `https://` URLs** ([`research/coinswapd/swap.go`](../research/coinswapd/swap.go)). That path uses Go’s **`net/http.Client`** with the default transport, which honors **`ProxyFromEnvironment`** (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`) — it does **not** read **`ALL_PROXY`** (curl convention).

**Set on the `coinswapd` process (example):**

```bash
export HTTP_PROXY="socks5h://127.0.0.1:9050"
export HTTPS_PROXY="socks5h://127.0.0.1:9050"   # if any hop uses https://
# Optional: exclude local sidecar / JSON-RPC that must not go through Tor
export NO_PROXY="127.0.0.1,localhost"
```

Use **`socks5h://`** (not `socks5://`) so **`.onion` names are not resolved by the local stub resolver** before the SOCKS handshake.

You may still set **`ALL_PROXY`** for other tools in the same shell (e.g. `curl`); **coinswapd** needs **`HTTP_PROXY`** for `http://` hop URLs.

## Maker side: hidden service + Nostr

- Each maker publishes a **reachable** JSON-RPC base URL for peers. In MLN, that is the maker’s **Tor hidden service** URL (with scheme), e.g. `http://abcd1234…onion:8334`.
- Configure **`MLND_TOR_ONION`** (and related `mlnd` env) so kind **31250** ads match what pathfind expects — see [`NOSTR_MLN.md`](NOSTR_MLN.md), [`PHASE_2_NOSTR.md`](../PHASE_2_NOSTR.md), and [`scripts/e2e-bootstrap.sh`](../scripts/e2e-bootstrap.sh) (local E2E uses cleartext `http://127.0.0.1:808n` **on purpose**; production lab uses real `.onion`).
- **`mln-cli pathfind`** requires **non-empty `tor`** on verified makers so routes are viable for Tor transport.

## Topology: `-mln-local-taker` vs public mesh

- **MLN taker + dynamic route from `mweb_submitRoute`:** use **`-mln-local-taker`** on the taker’s `coinswapd` so startup does not depend on **`getNodes()`** / `AliveNodes` matching a static **`-k`** mesh. See [Mesh vs MLN taker](../PHASE_3_MWEB_HANDOFF_SLICE.md#documented-gaps-vs-full-round--l2) in the Phase 3 slice doc and [`research/coinswapd/config/nodes.go`](../research/coinswapd/config/nodes.go).
- **Listed mesh node:** omit **`-mln-local-taker`**; **`-k`** must match your row in the probed topology.

## Warm-up: 1-hop, then 3-hop

1. **1-hop / proxy sanity:** With one maker’s `.onion` RPC reachable through Tor, confirm **`rpc.Dial`-equivalent** behavior by hitting that HTTP JSON-RPC (e.g. a trivial method) **with `HTTP_PROXY=socks5h://…` set** in the same environment you use for `coinswapd`.
2. **3-hop lab:** Three makers, distinct `.onion` endpoints, Nostr discovery + **`mln-cli pathfind`** route including **`swapX25519PubHex`** per hop; then **`mweb_getBalance`** → **`mweb_submitRoute`** → **`mweb_runBatch`** → poll **`mweb_getRouteStatus`**.
3. **Success bar:** **`pendingOnions == 0` without `-mweb-dev-clear-pending-after-batch`** (real finalize / `SendTransaction` after live **`swap_forward` / `swap_backward`**). Dev-clear is **DEV ONLY** — see [`COINSWAPD_MLN_FORK_SPEC.md`](COINSWAPD_MLN_FORK_SPEC.md) §2.7a.

## Failure triage (no secret logging)

- **Dial / timeout:** verify **`HTTP_PROXY`**, Tor running, `.onion` port, firewall.
- **`swap_forward:` / `swap_backward:` errors in logs:** see [`swap.go`](../research/coinswapd/swap.go); avoid pasting full onions + payloads into public tickets if they correlate real runs.
- **Sidecar / Docker:** [`deploy/docker-compose.e2e.sidecar-rpc.yml`](../deploy/docker-compose.e2e.sidecar-rpc.yml) uses `host.docker.internal` for **host** JSON-RPC; **taker `coinswapd`** must still dial **maker** onions — proxy env belongs on **`coinswapd`**, not only the sidecar.

## README Phase 3 checkbox (gate)

Do **not** mark README Phase 3 complete until **both**:

1. Live **`.onion` multi-hop** completes **without** dev-clear (this doc).
2. **Public LitVM** grievance/slash path scope is agreed and deployed per [`PHASE_16_PUBLIC_TESTNET.md`](../PHASE_16_PUBLIC_TESTNET.md) when RPC is available.

Until then, keep running regression anchors from [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../PHASE_3_MWEB_HANDOFF_SLICE.md) (**`E2E_MWEB_FULL=1`**, optional **`MWEB_RPC_BACKEND=coinswapd`** smoke).
