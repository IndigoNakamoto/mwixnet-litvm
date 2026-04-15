# Phase 3 operator push — completed record

## Goal

Advance README Phase 3 toward real operators by shipping **in-repo** operator runbooks and tooling: a single checklist (Tor, proxy, topology, 3× makers, production-shaped `pendingOnions`, LitVM gate), preflight scripts, compose env hints, cross-links from canonical docs, and on-code operator guidance for inter-hop JSON-RPC over Tor. **North star** remains operator-executed: live `.onion` multi-hop without dev-clear plus public LitVM when RPC exists ([`research/PHASE_3_TOR_OPERATOR_LAB.md`](../../research/PHASE_3_TOR_OPERATOR_LAB.md), [`PHASE_16_PUBLIC_TESTNET.md`](../../PHASE_16_PUBLIC_TESTNET.md)).

## In scope / out of scope

**In scope:** Documentation, shell helpers, Makefile target, `.env.compose.example` comments, `swap.go` comments re `HTTP_PROXY` / `socks5h`, README and phase doc cross-links, Phase 16 subsection for the LitVM half of the README Phase 3 gate.

**Out of scope:** Running a live 3-hop mix (requires operator hosts, Tor HS, keys, Neutrino); changing Nostr kinds or LitVM contracts; a non-stub **`grievance-correlated-*`** sibling script (deferred until a successful P2P lab—see plan optional artifacts).

## Primary files and canonical docs

- [`research/PHASE_3_OPERATOR_CHECKLIST.md`](../../research/PHASE_3_OPERATOR_CHECKLIST.md)
- [`scripts/phase3-operator-preflight.sh`](../../scripts/phase3-operator-preflight.sh), [`scripts/phase3-funded-env-check.sh`](../../scripts/phase3-funded-env-check.sh), [`scripts/tor-preflight.sh`](../../scripts/tor-preflight.sh)
- [`Makefile`](../../Makefile) (`phase3-operator-preflight`)
- [`.env.compose.example`](../../.env.compose.example)
- [`research/coinswapd/swap.go`](../../research/coinswapd/swap.go)
- Cross-links: [`README.md`](../../README.md), [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../../PHASE_3_MWEB_HANDOFF_SLICE.md), [`research/PHASE_3_TOR_OPERATOR_LAB.md`](../../research/PHASE_3_TOR_OPERATOR_LAB.md), [`PHASE_9_ENABLEMENT.md`](../../PHASE_9_ENABLEMENT.md), [`PHASE_16_PUBLIC_TESTNET.md`](../../PHASE_16_PUBLIC_TESTNET.md)
- Layer map: [`AGENTS.md`](../../AGENTS.md)

## Execution results

- Added **PHASE_3_OPERATOR_CHECKLIST** (workstreams A–F, north star, triage table, no-secret logging reminder).
- **`phase3-operator-preflight.sh`:** delegates to Tor preflight, prints copy-paste `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` for `coinswapd`, optional `PHASE3_ONION_JSONRPC_URL` one-hop `mweb_getBalance` POST via `curl --socks5-hostname`.
- **`phase3-funded-env-check.sh`:** advisory only; warns on `E2E_MWEB_FUNDED_DEV_CLEAR` and incomplete funded env.
- **`make phase3-operator-preflight`** wired in Makefile.
- **`.env.compose.example:** commented Phase 3 proxy block for paired `coinswapd`.
- **Phase 16:** short “README Phase 3 gate (LitVM half)” pointer to section 0 + checklist.
- **`swap.go`:** comments at both `rpc.Dial` sites (`swap_forward`, `swap_backward`) documenting `ProxyFromEnvironment` and Tor.
- **`go build`** in `research/coinswapd` succeeded after comment edits.
- **Tor preflight** without a local SOCKS listener exits non-zero (expected); operators need Tor on 9050/9150 or `TOR_PREFLIGHT_SKIP_CURL=1` for TCP-only.

## Verification

- `cd research/coinswapd && go build .`
- With Tor running: `make phase3-operator-preflight`; optional `PHASE3_ONION_JSONRPC_URL=http://….onion:PORT make phase3-operator-preflight`
- Handoff regression (unchanged): `E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh` per [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../../PHASE_3_MWEB_HANDOFF_SLICE.md)

## Layer-boundary check

**Tor** and **MWEB** (`coinswapd` transport, operator docs); **LitVM** and **Nostr** only as cross-references and existing gate wording—no stake authority on Nostr, no new on-chain behavior.

## Follow-ups

- After first successful live multi-hop lab: consider a **`grievance-correlated-*.sh`** variant for non-stub routes (plan optional artifact).
- README Phase 3 checkbox still requires operator proof of `.onion` completion **without** dev-clear and public LitVM deployment per official RPC.
