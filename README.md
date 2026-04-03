# MLN Stack (Mwixnet × LitVM × Nostr)

**A trust-minimized, non-custodial privacy routing network for Litecoin MWEB.**

The MLN Stack adapts the [Mimblewimble CoinSwap proposal](https://forum.grin.mw/t/mimblewimble-coinswap-proposal/8322) to Litecoin, eliminating the need for a single, trusted human coordinator. By decoupling the privacy engine from economic enforcement and network discovery, MLN targets scalable anonymity sets without bloating Layer 2 state or leaking metadata.

## Prerequisites

- **Go 1.22+** for [`mln-cli/`](mln-cli/) and [`mlnd/`](mlnd/) (`go test`, `make build-mln-cli`, Wails). Dependencies such as `go-ethereum` v1.14.x require the standard library packages `maps`, `slices`, and `log/slog`.
- **Which `go` is running:** After `brew install go`, Apple Silicon Homebrew puts the compiler at **`/opt/homebrew/opt/go/bin/go`**. If `which go` still shows **`/usr/local/go/bin/go`** (an older installer), that binary stays on your `PATH` first and you will keep seeing errors like `package maps is not in GOROOT (/usr/local/go/...)`. Fix: put Homebrew first, e.g. `export PATH="/opt/homebrew/opt/go/bin:$PATH"` (add to `~/.zshrc`), or remove/renamed the old `/usr/local/go` install. Confirm with `go version` (expect 1.22+).
- **Where to run tests:** `mlnd` and `mln-cli` are **sibling** directories under the repo root. From inside `mln-cli/`, use `cd ../mlnd` (not `cd mlnd`).
- [`mln-sidecar/`](mln-sidecar/) declares a lower Go version and an older `go-ethereum`; it may still build on older toolchains—see its `go.mod`.

## 🏗️ Architecture: separation of concerns

The MLN Stack intentionally isolates the four pillars of a trustless mixing network:

1. **Privacy engine (Litecoin MWEB):** Handles the core cryptography. Takers build layered ChaCha20 onions. Makers participate in multi-hop processing, range proofs, and sorting. Mix routing and fee compensation are settled natively on the MWEB extension block.
2. **Economic enforcement (LitVM):** The “judicial layer.” LitVM (an EVM L2 on Litecoin) acts as a deposit and slash pool using bridged `zkLTC`. It does **not** process happy-path mixes. It is for registry stake, bonds, slashing, and grievance flows when a maker drops a payload or fails to broadcast—aligned with [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) (sections 5–6).
3. **Discovery & coordination (Nostr):** Replaces a central coordinator API. Makers can broadcast reachability, LitVM stake pointers, and fee signals; takers use relays as an ephemeral bulletin board to construct routes. Stake authority remains on LitVM, not Nostr.
4. **Transport (Tor):** Onion routing hides client-to-node and node-to-node IP linkage, aligned with the cryptographic payload layers.

## 🛡️ Core guarantees

- **Non-custodial:** Mix nodes never take ownership of user coins. A failed mix leaves the taker’s UTXOs unspent on L1/MWEB.
- **No single coordinator of authority:** Route construction and epochs are decentralized; there is no one human or server with custody of the protocol.
- **Cryptographic accountability:** Slashing on LitVM is meant to rely on verifiable claims (canonical evidence hashes, hop receipts; **L1 inclusion proofs where that defense path is used**—see open questions in the spec)—not informal moderation.
- **Minimal L2 footprint (v1):** Per-hop routing fees stay in the MWEB fee budget; LitVM is for staking and dispute resolution, not a second fee rail ([`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) section 5.2).

## 🗺️ Roadmap status

Spec detail lives in [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) (roadmap table, section 9); milestones here are a summary.

**Production disclaimer:** While this repository’s **Phases 1–16** implementation PoC is **feature-complete in-tree** for documented bring-up paths, it is **not production** until **LitVM testnet broadcast** on the official chain, an independent **security audit**, and **C++ `coinswapd` integration** are finalized.

**Recently completed in-repo (Phases 12–16):** local **E2E stack** (Anvil + Nostr + three `mlnd` makers + bootstrap), **`mln-sidecar`** with **`mock` and `rpc`** modes and JSON-RPC forwarding to **`mweb_submitRoute`** / **`mweb_getBalance`** (fork-side implementation still out of repo), **self-included routing** (wallet + pathfind), **Phase 15 LitVM economic hardening** (slash, bonds, exit locks, reentrancy guard) plus **registry invariants** in Foundry fuzzing, and **Phase 16 public testnet readiness** (public relay + RPC defaults, `RPC_URL` / verification env, [`deploy/docker-compose.testnet.yml`](deploy/docker-compose.testnet.yml)). **Slither** runs on `contracts/**` changes in [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml).

- **[x] Phase 0 — Protocol clarity** — MWEB transaction layer vs Grin baseline ([appendix 14](PRODUCT_SPEC.md)), native fee path via [`ltcmweb/coinswapd`](https://github.com/ltcmweb/coinswapd), and **canonical `evidenceHash` preimage** in [appendix 13](PRODUCT_SPEC.md) (validate against nodes before freezing registry ABIs).
- **[x] Phase 1 — LitVM contracts (local complete)** — Foundry project in [`contracts/`](contracts/): `MwixnetRegistry`, `GrievanceCourt`, [`EvidenceLib`](contracts/src/EvidenceLib.sol) (appendix 13.5), fuzz tests, [`Makefile`](Makefile), [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh), [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml). **Not audited.**
- **[ ] Phase 1 — LitVM testnet broadcast (pending)** — Needs [public RPC, chain ID, and zkLTC](https://docs.litvm.com/get-started-on-testnet/add-to-wallet). Runbook and operator compose: [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md). Set `PRIVATE_KEY`, `RPC_URL`, and optional `ETHERSCAN_API_KEY` in `contracts/.env` ([`contracts/.env.example`](contracts/.env.example)). See [`research/LITVM.md`](research/LITVM.md).
- **[~] Phase 2 — Nostr profile (in progress)** — Normative wire (kinds **31250–31251**, `content` JSON, `nostrKeyHash` binding) is in [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md); JSON fixtures are validated in CI under [`nostr/`](nostr/). Demo CLIs under [`scripts/`](scripts/) use the same kinds and shapes. Local stack: `make test-full-stack`. **Live relay walkthrough:** [`research/E2E_NOSTR_DEMO.md`](research/E2E_NOSTR_DEMO.md) (`scripts/nostr_watch.py` + `publish_grievance.py`).
- **[ ] Phase 3 — End-to-end integration** — Nostr discovery → Tor → MWixnet round → L2 settlement / slash path.
- **[x] Phase 8 — Testnet packaging (local complete)** — [`PHASE_8_TESTNET_RELEASE.md`](PHASE_8_TESTNET_RELEASE.md): `mlnd` Dockerfile, `make build` / `make docker-build`, `make testnet-smoke`, [`mlnd/.env.example`](mlnd/.env.example), GitHub Releases on `v*` tags ([`.github/workflows/mlnd-release.yml`](.github/workflows/mlnd-release.yml)). **Binaries attach on tag push; LitVM testnet RPC still pending for live operator runs.**
- **[x] Phase 9 — Enablement and hardening (operator packaging)** — [`PHASE_9_ENABLEMENT.md`](PHASE_9_ENABLEMENT.md): Docker Compose ([`docker-compose.yml`](docker-compose.yml)), [`.env.compose.example`](.env.compose.example), NDJSON bridge ops with patched `coinswapd`, defense and Nostr runbooks.
- **[x] Phase 10 — Taker client (`mln-cli`)** — [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md): Scout (Nostr 31250 + LitVM verification), Pathfind (3-hop route), Forger (Tor dry-run + HTTP POST of route JSON to a local **MLN sidecar** URL; onion build remains in `coinswapd` fork/proxy). Build: `make build-mln-cli`. Shared wire types: [`mlnd/pkg/makerad`](mlnd/pkg/makerad).
- **[x] Phase 11 — Taker wallet (Wails)** — Desktop GUI in [`mln-cli/desktop/`](mln-cli/desktop/) (React + Wails v2, `wails` build tag). Build: `make build-mln-wallet`. Developer notes: [`mln-cli/desktop/README.md`](mln-cli/desktop/README.md).
- **[x] Phase 12 — E2E Crucible (local simulation)** — [`PHASE_12_E2E_CRUCIBLE.md`](PHASE_12_E2E_CRUCIBLE.md): [`deploy/docker-compose.e2e.yml`](deploy/docker-compose.e2e.yml) (Anvil + `nostr-rs-relay` + **`mln-sidecar`** + three `mlnd` makers), [`scripts/e2e-bootstrap.sh`](scripts/e2e-bootstrap.sh), generated wallet settings under `deploy/` (gitignored).
- **[x] Phase 13 — Sidecar shim (`mln-sidecar`)** — Pure-Go HTTP service: **`GET /v1/balance`**, **`POST /v1/swap`** ([`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md)). **`-mode=mock`** (default, Phase 12 E2E) simulates balance and onion handoff; **`-mode=rpc`** dials **`-rpc-url`** (default `http://127.0.0.1:8546`) and calls **`mweb_submitRoute`** / **`mweb_getBalance`** on a **`coinswapd` fork** (see [`mln-sidecar/README.md`](mln-sidecar/README.md)). Build: `make build-mln-sidecar`. Module: [`mln-sidecar/`](mln-sidecar/).
- **[x] Phase 14 — Self-inclusion UX** — [`PHASE_14_SELF_INCLUSION.md`](PHASE_14_SELF_INCLUSION.md): optional **Self-Included Routing** (wallet + `mln-cli pathfind -self-included` + `MLN_OPERATOR_ETH_KEY`); pathfind fixes **N2** to the local registered maker; Scout marks **Local node**; `mln-sidecar` unchanged (middle-hop relay stays `coinswapd` / `mlnd`).
- **[x] Phase 15 — Economic hardening (LitVM contracts)** — [`PHASE_15_ECONOMIC_HARDENING.md`](PHASE_15_ECONOMIC_HARDENING.md): real **`slashStake`** (bounty / burn to `address(0)`), **`slashBps`**, exoneration **bond forfeit** to accused, **`withdrawalLockUntil`** + `slashingWindow`, auto-deregister when stake falls below `minStake`, OpenZeppelin **`ReentrancyGuard`** on registry exits and slash; dual registry invariants covered in Foundry fuzz runs. **Not audited.**
- **[x] Phase 16 — Public testnet readiness** — [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md): public Nostr relay + Sepolia HTTP/chain defaults in the wallet config, Foundry **`RPC_URL`** / **`ETHERSCAN_API_KEY`**, [`deploy/docker-compose.testnet.yml`](deploy/docker-compose.testnet.yml) + [`deploy/.env.testnet.example`](deploy/.env.testnet.example) for makers (no Anvil / local relay).

### Phase 1 local (already shipped)

Handoff checklist for anyone picking up the repo:

- **Spec helpers:** [`contracts/src/EvidenceLib.sol`](contracts/src/EvidenceLib.sol) (`evidenceHash`, `grievanceId` per appendix 13.5); [`contracts/test/EvidenceHash.t.sol`](contracts/test/EvidenceHash.t.sol).
- **Fuzz:** [`contracts/test/FuzzRegistry.t.sol`](contracts/test/FuzzRegistry.t.sol), [`contracts/test/FuzzGrievanceCourt.t.sol`](contracts/test/FuzzGrievanceCourt.t.sol).
- **Local deploy:** [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh) (run Anvil separately), [`Makefile`](Makefile) (`make contracts-test`, `make deploy-local`, `make test-grievance`), [`contracts/deployments/anvil-local.example.json`](contracts/deployments/anvil-local.example.json) — generated `anvil-local.json` stays gitignored.
- **CI:** [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml) (`forge build` / `forge test` via Docker Foundry; **Slither** on `contracts/**` changes).

### Run `mlnd` on LitVM testnet

When [official RPC and chain ID](https://docs.litvm.com/get-started-on-testnet/add-to-wallet) are available, use [`research/LITVM.md`](research/LITVM.md) and [`mlnd/.env.example`](mlnd/.env.example). Quick RPC check: set `MLND_HTTP_URL` (HTTP JSON-RPC) and `MLND_COURT_ADDR`, then `make testnet-smoke`. Run the daemon with `MLND_WS_URL` and the same court/operator settings (see [`mlnd/README.md`](mlnd/README.md)).

### Release process (`mlnd` binaries)

Push a version tag matching `v*` (e.g. `v0.1.0`). [`.github/workflows/mlnd-release.yml`](.github/workflows/mlnd-release.yml) builds **linux/amd64** and **linux/arm64** with CGO (SQLite), then attaches `mlnd-linux-amd64` and `mlnd-linux-arm64` to the GitHub Release. ARM runners need a **public** repository (or adjust the workflow). Docker: `make docker-build` → image `mlnd:local`.

## Deployment (operators)

The maker daemon (`mlnd`) can run under Docker Compose with persistent SQLite and a shared receipt directory for a patched [`coinswapd`](https://github.com/ltcmweb/coinswapd).

### Quick start

1. `mkdir -p data/mlnd data/receipts`
2. `cp .env.compose.example .env` and fill in LitVM endpoints and keys (see [`research/LITVM.md`](research/LITVM.md)).
3. `docker compose up -d mlnd`

For NDJSON bridge layout, auto-defend key practice, Nostr relays, and paired `coinswapd` mounting, read the **[Phase 9 enablement playbook](PHASE_9_ENABLEMENT.md)**.

### Next steps (pickup after a break)

1. **LitVM testnet** — When [official RPC and chain ID](https://docs.litvm.com/get-started-on-testnet/add-to-wallet) are published, follow [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md): fund a throwaway deployer, set `PRIVATE_KEY` and `RPC_URL` in `contracts/.env`, run [`forge script`](contracts/README.md) with `--broadcast` (and `--verify` when applicable). Record deployed addresses.
2. **Judicial layer** — Phase 15 economics and locks are implemented in [`GrievanceCourt`](contracts/src/GrievanceCourt.sol) and [`MwixnetRegistry`](contracts/src/MwixnetRegistry.sol) ([`PHASE_15_ECONOMIC_HARDENING.md`](PHASE_15_ECONOMIC_HARDENING.md)). Contracts remain **not audited**; on-chain **`defenseData` verification** and formal review are still open.
3. **`coinswapd` fork** — Implement **`mweb_submitRoute`** / **`mweb_getBalance`** so **`mln-sidecar -mode=rpc`** can hand off MLN route JSON to the real MWEB engine (see [`mln-sidecar/README.md`](mln-sidecar/README.md), [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md)).
4. **Phase 2 (in progress)** — Keep [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md), [`nostr/fixtures/`](nostr/), and [`scripts/`](scripts/) in lockstep; run `make test-full-stack` for local grievance + Nostr pointer validation. Finalize addresses/tags once LitVM testnet registry values are stable.

Appendix 13 hashing is implemented in [`contracts/src/EvidenceLib.sol`](contracts/src/EvidenceLib.sol) and covered by [`contracts/test/EvidenceHash.t.sol`](contracts/test/EvidenceHash.t.sol).

---

This repository holds the **product specification**, research notes, and Cursor configuration (spec **v0.1**, draft). **Production vs PoC** is defined under **Roadmap status** above.

## Documentation

| Document | Purpose |
| -------- | ------- |
| [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) | Full architecture, economics, roadmap, evidence preimage (appendix 13), MWEB appendix (14), open questions |
| [`AGENTS.md`](AGENTS.md) | Contributor / agent orientation (layer boundaries, canonical sources) |
| [`contracts/README.md`](contracts/README.md) | Solidity layout, local Anvil deploy, `make contracts-test` |
| [`Makefile`](Makefile) | Contracts (`contracts-build`, `contracts-test`, `deploy-local`, `test-grievance`), operator smoke (`test-operator-smoke`, `test-full-stack`), `mlnd` / CLI / wallet / sidecar builds, Docker images — see phase playbooks below for context |
| [`PHASE_5_NOSTR_TOR_BRIDGE.md`](PHASE_5_NOSTR_TOR_BRIDGE.md) | Phase 5: Nostr relay behavior, Tor URL clarity, receipt bridge scaffold (`mlnd`) |
| [`PHASE_6_BRIDGE_INTEGRATION.md`](PHASE_6_BRIDGE_INTEGRATION.md) | Phase 6: NDJSON receipt bridge → `mlnd` SQLite, identity threading vs `coinswapd` |
| [`PHASE_7_END_TO_END.md`](PHASE_7_END_TO_END.md) | Phase 7: golden NDJSON → `mlnd` bridge → LitVM grievance operator smoke (`make test-operator-smoke`) |
| [`PHASE_8_TESTNET_RELEASE.md`](PHASE_8_TESTNET_RELEASE.md) | Phase 8: `mlnd` Docker, release workflow, `make testnet-smoke`, GitHub Releases |
| [`PHASE_9_ENABLEMENT.md`](PHASE_9_ENABLEMENT.md) | Phase 9: operator packaging — Compose, env template, NDJSON bridge + `coinswapd`, defense and Nostr ops |
| [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md) | Phase 10: taker CLI (`mln-cli`) — Scout, Pathfind, Forger (dry-run + sidecar POST); env and trust model |
| [`mln-cli/desktop/README.md`](mln-cli/desktop/README.md) | Phase 11: Wails taker wallet (`make build-mln-wallet`, `wails` build tag) |
| [`PHASE_12_E2E_CRUCIBLE.md`](PHASE_12_E2E_CRUCIBLE.md) | Phase 12: local Docker E2E — Anvil + Nostr relay + 3× `mlnd`, `scripts/e2e-bootstrap.sh`, [`deploy/docker-compose.e2e.yml`](deploy/docker-compose.e2e.yml) |
| [`mln-sidecar/README.md`](mln-sidecar/README.md) | Phase 13: `mln-sidecar` HTTP shim — `GET /v1/balance`, `POST /v1/swap`; `-mode=mock` vs `-mode=rpc` (`mweb_submitRoute` / `mweb_getBalance`) |
| [`PHASE_14_SELF_INCLUSION.md`](PHASE_14_SELF_INCLUSION.md) | Phase 14: optional self-included routing (wallet + `mln-cli pathfind -self-included`) |
| [`PHASE_15_ECONOMIC_HARDENING.md`](PHASE_15_ECONOMIC_HARDENING.md) | Phase 15: LitVM slash economics, bond forfeit, slashing window, registry reentrancy guard; Foundry invariant fuzzing; Slither in CI |
| [`PHASE_16_PUBLIC_TESTNET.md`](PHASE_16_PUBLIC_TESTNET.md) | Phase 16: public testnet readiness — Foundry `RPC_URL` / verification, operator `docker-compose.testnet.yml`, wallet defaults vs local E2E |
| [`research/THREAT_MODEL_MLN.md`](research/THREAT_MODEL_MLN.md) | Accepted code review snapshot, threat tables, and residual risks (not a substitute for audit) |
| [`docker-compose.yml`](docker-compose.yml) | `mlnd` service + commented `coinswapd` stub; use with [`.env.compose.example`](.env.compose.example) |
| [`scripts/requirements.txt`](scripts/requirements.txt) | `pip install -r scripts/requirements.txt` for Nostr demo CLIs (`nostr` PyPI package) |
| [`research/LITVM.md`](research/LITVM.md) | LitVM testnet, env, Docker Foundry, Phase 1 local |
| [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md) | Phase 2 Nostr wire: kinds 31250–31251, `nostrKeyHash` binding, maker ads + grievance pointers |
| [`research/E2E_NOSTR_DEMO.md`](research/E2E_NOSTR_DEMO.md) | Relay E2E: Anvil + golden grievance + `publish_grievance.py` + `nostr_watch.py` |
| [`research/NOSTR_EVENTS.md`](research/NOSTR_EVENTS.md) | Archived pointer (historical filename); normative spec is `NOSTR_MLN.md` |
| [`research/USER_STORIES_MLN.md`](research/USER_STORIES_MLN.md) | User stories, coordination model, epoch semantics, wallet auto-route policy (PoC) |
| [`research/WALLET_TAKER_FLOW_V1.md`](research/WALLET_TAKER_FLOW_V1.md) | Wallet wireframe-level taker flow, UTC-midnight epoch UX behavior, and edge-case actions |
| [`research/WALLET_MAKER_FLOW_V1.md`](research/WALLET_MAKER_FLOW_V1.md) | Operator maker flow: register, Nostr ad, dashboard, batch participation, timelocked exit, grievance defense |
| [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md) | Map of `coinswapd` (RPCs, onion shape, `ltcd` boundary) |

## Local reference code (optional)

Clone a `coinswapd` tree beside the spec to follow code references in the teardown:

```bash
git clone https://github.com/ltcmweb/coinswapd.git research/coinswapd
```

`research/coinswapd/` is **gitignored** and not part of this repository.

## Cursor

Rules under [`.cursor/rules/`](.cursor/rules/) (e.g. [`.cursor/rules/doc-sync.mdc`](.cursor/rules/doc-sync.mdc) when editing top-level docs), skills under [`.cursor/skills/`](.cursor/skills/) (including [`.cursor/skills/doc-sync/SKILL.md`](.cursor/skills/doc-sync/SKILL.md) for documentation synchronization passes).

## License

Not specified; add a `LICENSE` when you publish.
