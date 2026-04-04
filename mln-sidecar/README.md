# `mln-sidecar`

Lightweight MLN HTTP shim between the taker wallet / `mln-cli` forger and the MWEB engine. It does **not** implement wallet or onion cryptography; it only validates the MLN HTTP contract and forwards to JSON-RPC when configured.

## Modes

- **`-mode`** — default `mock`. Use `mock` for Phase 12 E2E (fixed balance, simulated onion log) or `rpc` to forward swap/balance to `-rpc-url`.
- **`-rpc-url`** — default `http://127.0.0.1:8546`. JSON-RPC base URL for a **coinswapd fork** when `-mode=rpc`; ignored in `mock`. If this URL is only reachable via Tor, set **`HTTP_PROXY` / `ALL_PROXY`** (e.g. **`socks5h://127.0.0.1:9050`**) on the sidecar process; go-ethereum’s RPC client uses the default transport’s proxy-from-environment behavior.
- **`-port`** — default `8080`. HTTP listen port for `GET /v1/balance` and `POST /v1/swap`.

**Hop `tor` strings:** `POST /v1/swap` normalizes each hop’s `tor` field by trimming whitespace and adding **`http://`** when no `scheme://` is present so payloads match **`rpc.Dial`** expectations (common for Nostr ads that publish `host.onion:port` only). Operators should still prefer explicit `http://` or `https://` in ads.

## Fork JSON-RPC contract (`-mode=rpc`)

The coinswapd fork (outside this repo) should expose:

- **`mweb_submitRoute`** — params: one object matching [`mln-cli` `RequestPayload`](../mln-cli/internal/forger/client.go) (`route`, `destination`, `amount`; each hop may include optional `swapX25519PubHex` per [`research/COINSWAPD_MLN_FORK_SPEC.md`](../research/COINSWAPD_MLN_FORK_SPEC.md)). Builds/persists the swap server-side.
- **`mweb_getBalance`** — no params. Result object: `availableSat`, `spendableSat` (uint64), optional `detail` string — same semantics as [`PHASE_10_TAKER_CLI.md`](../PHASE_10_TAKER_CLI.md) / `GET /v1/balance`.

Vanilla ltcmweb exposes `swap_Swap(onion.Onion)` only; see [`research/COINSWAPD_TEARDOWN.md`](../research/COINSWAPD_TEARDOWN.md).

Fork integration spec (wire contract, onion build checklist, optional `swapX25519PubHex` on each hop): [`research/COINSWAPD_MLN_FORK_SPEC.md`](../research/COINSWAPD_MLN_FORK_SPEC.md).

## HTTP API

- **`GET /v1/balance`** — in `mock`, fixed mock balances; in `rpc`, `mweb_getBalance`. On RPC failure returns **502** with `ok: false`.
- **`POST /v1/swap`** — MLN route JSON; in `mock`, validates and logs simulated onion; in `rpc`, `mweb_submitRoute`. Validation errors **400**; RPC errors **502**.

## Run

```bash
# repo root
make build-mln-sidecar
./bin/mln-sidecar -port 8080 -mode mock

# production-style bridge (fork must implement mweb_* methods)
./bin/mln-sidecar -port 8080 -mode rpc -rpc-url http://127.0.0.1:8546
```

## Docker (E2E)

[`deploy/docker-compose.e2e.yml`](../deploy/docker-compose.e2e.yml) runs the sidecar with **`-mode=mock`** explicitly. Image default `CMD` is also `-port 8080 -mode mock`.

See [`PHASE_12_E2E_CRUCIBLE.md`](../PHASE_12_E2E_CRUCIBLE.md).

## `mw-rpc-stub` (Phase 3a integration helper)

The repo ships **`cmd/mw-rpc-stub`**: a minimal HTTP JSON-RPC server that implements **`mweb_getBalance`** and **`mweb_submitRoute`** so **`mln-sidecar -mode=rpc`** can be tested without running the full **`research/coinswapd`** stack (Neutrino, MWEB keys, etc.). Build from the repository root: **`make build-mw-rpc-stub`** → **`bin/mw-rpc-stub`** (default **`-addr :8546`**). Runbook: [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../PHASE_3_MWEB_HANDOFF_SLICE.md).

Phase 14 **self-included** routes do not change this service: hop identity and `swap_forward` handling remain in **`mlnd` / `coinswapd`** ([`PHASE_14_SELF_INCLUSION.md`](../PHASE_14_SELF_INCLUSION.md)).
