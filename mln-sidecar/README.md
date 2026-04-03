# `mln-sidecar`

Lightweight MLN HTTP shim between the taker wallet / `mln-cli` forger and the MWEB engine.

- **`GET /v1/balance`** — mock balances for local E2E ([`PHASE_10_TAKER_CLI.md`](../PHASE_10_TAKER_CLI.md)).
- **`POST /v1/swap`** — accepts the Phase 10 route JSON, validates it, logs a simulated onion build, returns success. In production, the same process boundary should translate the route into **`swap_Swap(onion.Onion)`** on `coinswapd` ([`research/COINSWAPD_TEARDOWN.md`](../research/COINSWAPD_TEARDOWN.md)).

## Run

```bash
# repo root
make build-mln-sidecar
./bin/mln-sidecar -port 8080
```

## Docker (E2E)

Started by [`deploy/docker-compose.e2e.yml`](../deploy/docker-compose.e2e.yml) on host port **8080**. See [`PHASE_12_E2E_CRUCIBLE.md`](../PHASE_12_E2E_CRUCIBLE.md).

Phase 14 **self-included** routes do not change this service: hop identity and `swap_forward` handling remain in **`mlnd` / `coinswapd`** ([`PHASE_14_SELF_INCLUSION.md`](../PHASE_14_SELF_INCLUSION.md)).
