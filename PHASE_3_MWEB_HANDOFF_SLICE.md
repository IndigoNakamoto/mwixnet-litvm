# Phase 3a — MWEB handoff slice (no official LitVM testnet)

This is a **sub-milestone** toward README **Phase 3** (full end-to-end integration). It does **not** check the Phase 3 box: it proves the **taker → `mln-sidecar` HTTP → JSON-RPC `mweb_*`** bridge that [`mln-sidecar`](mln-sidecar/README.md) uses in **`-mode=rpc`**, while keeping **local Anvil** as the registry/court stand-in (same as [Phase 12](PHASE_12_E2E_CRUCIBLE.md)).

**Out of scope for this slice:** official [LitVM testnet](https://docs.litvm.com/) RPC, real Tor hops, Neutrino-backed [`research/coinswapd/`](research/coinswapd/) in CI, and on-chain slash resolution on a public chain.

## Completion (stub + full CLI path)

**Status: complete** for the **documented stub stack** as of **2026-04-03**.

**Verification:** `E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh` — `mw-rpc-stub` on **:8546**, Docker Compose (**Anvil + Nostr + `mln-sidecar -mode=rpc` + three `mlnd` makers**), [`scripts/e2e-bootstrap.sh`](scripts/e2e-bootstrap.sh) deploy + maker registration, then **`mln-cli pathfind -json`** and **`mln-cli forger`** posting to **`http://127.0.0.1:8080/v1/swap`**; stub logged **`mweb_submitRoute ok`** and forger reported route accepted. Quick **`curl`**-only path: `./scripts/e2e-mweb-handoff-stub.sh` (no `E2E_MWEB_FULL`).

This **does not** close README **Phase 3** (full Nostr → Tor → MWixnet round → L2 path). **Promote path** to real MWEB JSON-RPC remains [`research/coinswapd/`](research/coinswapd/) on a separate port — see [Promote path](#promote-path-researchcoinswapd) below.

## Status (regression / hardening)

- **`mln-sidecar -mode=rpc`:** [`internal/mweb/rpc_bridge.go`](mln-sidecar/internal/mweb/rpc_bridge.go) normalizes `-rpc-url` (trim space, trailing slash) and calls **`mweb_submitRoute`** / **`mweb_getBalance`** via go-ethereum `rpc.Client` (same method names as [`research/coinswapd/mweb_service.go`](research/coinswapd/mweb_service.go)).
- **Tests:** [`mln-sidecar/internal/mweb/rpc_bridge_test.go`](mln-sidecar/internal/mweb/rpc_bridge_test.go) asserts JSON-RPC **params** for `mweb_submitRoute` decode to the fork’s route object (including optional per-hop **`swapX25519PubHex`**). [`mln-sidecar/internal/api/server_rpc_test.go`](mln-sidecar/internal/api/server_rpc_test.go) covers HTTP 502 on upstream RPC errors for swap and balance. [`research/coinswapd/mlnroute/sidecar_wire_test.go`](research/coinswapd/mlnroute/sidecar_wire_test.go) golden-unmarshals the same JSON as [`scripts/e2e-mweb-handoff-stub.sh`](scripts/e2e-mweb-handoff-stub.sh).
- **`mw-rpc-stub`:** validates **`mweb_submitRoute`** payload shape (3 hops, destination, amount) so Compose/curl exercises fail loudly if the sidecar wire drifts.

## Goal

Validate the **integration contract** from [research/COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md): `mln-cli forger` / Wails **Send Privately** → **`GET /v1/balance`** and **`POST /v1/swap`** on the sidecar → **`mweb_getBalance`** / **`mweb_submitRoute`** on a JSON-RPC peer.

## Port layout (local E2E + rpc sidecar)

| Service | Port | Notes |
| ------- | ---- | ----- |
| Anvil (registry HTTP/WS) | **8545** | Taker `MLN_LITVM_HTTP_URL`; matches Phase 12 |
| MWEB JSON-RPC (`mweb_*`) | **8546** | Stub (`mw-rpc-stub`) or **`coinswapd` fork** — must **not** collide with sidecar **8080** |
| `mln-sidecar` HTTP | **8080** | Forger default `-coinswapd-url` … `/v1/swap` |
| Nostr relay (host) | **7080** | Same as Phase 12 Compose map |

Upstream **`coinswapd`** defaults to **`-l 8080`** ([research/COINSWAPD_TEARDOWN.md](research/COINSWAPD_TEARDOWN.md)). When running next to this stack, start the fork with **`-l 8546`** (or another free port) and point the sidecar at it.

## Quick path: stub + Compose

1. From repo root: **`./scripts/e2e-mweb-handoff-stub.sh`**  
   - Starts **`mw-rpc-stub`** on **`:8546`**, brings up Anvil + relay + sidecar with [deploy/docker-compose.e2e.sidecar-rpc.yml](deploy/docker-compose.e2e.sidecar-rpc.yml), and checks the HTTP→RPC path with **`curl`**.

2. Full Scout → Pathfind → Forger (optional):  
   **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`**  
   - Runs [scripts/e2e-bootstrap.sh](scripts/e2e-bootstrap.sh), starts the **makers** profile, exports **`MLN_*`** from generated `deploy/e2e.generated.env`, then **`mln-cli pathfind -json`** and **`mln-cli forger`** against `http://127.0.0.1:8080/v1/swap`.

Build the stub: **`make build-mw-rpc-stub`** (output **`bin/mw-rpc-stub`**).

## Acceptance criteria

- **`POST /v1/swap`** with a valid **three-hop** MLN route returns **200** and `"ok":true` from the sidecar while it is in **`-mode=rpc`**.
- **`GET /v1/balance`** returns **200** and balances forwarded from **`mweb_getBalance`**.
- Stub logs (or fork logs) show **`mweb_submitRoute`** was invoked (integration parity with [mln-sidecar/internal/api/server_rpc_test.go](mln-sidecar/internal/api/server_rpc_test.go)).

## Promote path: `research/coinswapd`

Replace the stub with a built binary from [`research/coinswapd/`](research/coinswapd/) listening on **8546**, with flags such as **`-l 8546`**, **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, **`-a`** (MWEB fee address), **`-k`** (server ECDH key), and optionally **`-mweb-pubkey-map`** per [COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md).

**Expectation:** that process starts **Neutrino** against **mainnet** in the current fork ([research/coinswapd/main.go](research/coinswapd/main.go)) — sync time, disk, and keys are **operator** concerns, not part of the default automated stub flow.

## Related

- Phase 12 closed loop (mock sidecar): [PHASE_12_E2E_CRUCIBLE.md](PHASE_12_E2E_CRUCIBLE.md)  
- Threat model note on mock vs rpc: [research/THREAT_MODEL_MLN.md](research/THREAT_MODEL_MLN.md)  
- Sidecar modes: [mln-sidecar/README.md](mln-sidecar/README.md)
