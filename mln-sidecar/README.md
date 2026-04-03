# `mln-sidecar`

Lightweight MLN HTTP shim between the taker wallet / `mln-cli` forger and the MWEB engine. It does **not** implement wallet or onion cryptography; it only validates the MLN HTTP contract and forwards to JSON-RPC when configured.

## Modes

- **`-mode`** — default `mock`. Use `mock` for Phase 12 E2E (fixed balance, simulated onion log) or `rpc` to forward swap/balance to `-rpc-url`.
- **`-rpc-url`** — default `http://127.0.0.1:8546`. JSON-RPC base URL for a **coinswapd fork** when `-mode=rpc`; ignored in `mock`.
- **`-port`** — default `8080`. HTTP listen port for `GET /v1/balance` and `POST /v1/swap`.

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

Phase 14 **self-included** routes do not change this service: hop identity and `swap_forward` handling remain in **`mlnd` / `coinswapd`** ([`PHASE_14_SELF_INCLUSION.md`](../PHASE_14_SELF_INCLUSION.md)).
