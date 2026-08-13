# Permissioned mixnet first (2026-08-12)

## Goal

Treat `mwixnet-litvm` as the mixnet product: three permissioned `coinswapd` hops that shuffle same-value MWEB self-spends. Split Phase 3 into **3-mix** vs **3-gov**. Prove JSON without LitVM, ship a hop directory + Scout-free route, reject mixed-amount batches, freeze the wallet HTTP contract. Do not put CoinSwap in the Mac wallet until Proof A is a pass.

## In scope / out of scope

**In:** README / spec §9 / AGENTS / operator docs gate split; `mweb_submitRoute` JSON without LitVM; `deploy/mixnet.directory.example.json` + `mln-cli route from-directory`; one denomination per `performSwap`; sidecar taker API freeze; dated Proof A attempt record.

**Out:** Proof B (stranger / 1 LTC / 1-of-N honest); CoinSwap UI in the external Tauri wallet; Nostr, Scout, `mlnd` on mixnodes; LitVM court on the happy path; new `PHASE_*.md` playbooks.

## Primary files and canonical docs

- [`README.md`](../../README.md), [`PRODUCT_SPEC.md`](../../PRODUCT_SPEC.md) §9 status, [`AGENTS.md`](../../AGENTS.md)
- [`deploy/mixnet.directory.example.json`](../../deploy/mixnet.directory.example.json)
- [`mln-cli/internal/mixnetdir/`](../../mln-cli/internal/mixnetdir/)
- [`research/coinswapd/swap_denom.go`](../../research/coinswapd/swap_denom.go), [`research/coinswapd/mweb_service.go`](../../research/coinswapd/mweb_service.go)
- [`mln-sidecar/README.md`](../../mln-sidecar/README.md) frozen taker contract
- [`research/PHASE_3_OPERATOR_PLAYBOOK.md`](../../research/PHASE_3_OPERATOR_PLAYBOOK.md) Part B0
- [`LIVE_COINSWAP_ATTEMPT_2026-08-12.md`](../../LIVE_COINSWAP_ATTEMPT_2026-08-12.md)

## Execution results

- Phase 3 checkbox split into **3-mix** (shuffle) and **3-gov** (discovery + court, parked). Mix success bar is dest-coin appearance, not an empty onion queue and not LitVM.
- Forger / sidecar / `mlnroute` tests accept `mweb_submitRoute` JSON with no `epochId` / `accuser` / `operator` / `swapId`.
- `mln-cli route from-directory` writes slim `route.json` (no zero-address operators).
- `performSwap` rejects mixed recorded amounts; vanilla `swap_Swap` onions without amounts still skip the gate.
- **Proof A live mix:** not a pass. Preflight passed (Tor SOCKS + check.torproject.org). No funded UTXO or three operator HS in this session. Attempt file is labeled Proof A, not Proof B. Mac wallet not given CoinSwap UI.

## Verification

```bash
go test ./...   # from research/coinswapd
go test ./internal/mixnetdir ./internal/forger   # from mln-cli
go test ./internal/mweb ./...   # from mln-sidecar
./bin/mln-cli route from-directory -directory deploy/mixnet.directory.example.json -out /tmp/route.json
make phase3-operator-preflight
```

## Layer-boundary check

MWEB (`coinswapd`) owns the shuffle and hop fees. Tor is transport. LitVM and Nostr were not wired onto the happy path. The taker is a client of sidecar HTTP, not mixnode 0.

## Follow-ups

Proof A funded 3-hop (operator keys + exact UTXO). Proof B and Tauri CoinSwap stay parked until a Proof A pass file exists.
