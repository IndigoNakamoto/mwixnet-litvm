# MLN Stack (Mwixnet × LitVM × Nostr)

**A trust-minimized, non-custodial privacy routing network for Litecoin MWEB.**

The MLN Stack adapts the [Mimblewimble CoinSwap proposal](https://forum.grin.mw/t/mimblewimble-coinswap-proposal/8322) to Litecoin, eliminating the need for a single, trusted human coordinator. By decoupling the privacy engine from economic enforcement and network discovery, MLN targets scalable anonymity sets without bloating Layer 2 state or leaking metadata.

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

- **[x] Phase 0 — Protocol clarity** — MWEB transaction layer vs Grin baseline ([appendix 14](PRODUCT_SPEC.md)), native fee path via [`ltcmweb/coinswapd`](https://github.com/ltcmweb/coinswapd), and **canonical `evidenceHash` preimage** in [appendix 13](PRODUCT_SPEC.md) (validate against nodes before freezing registry ABIs).
- **[x] Phase 1 — LitVM contracts (local complete)** — Foundry project in [`contracts/`](contracts/): `MwixnetRegistry`, `GrievanceCourt`, [`EvidenceLib`](contracts/src/EvidenceLib.sol) (appendix 13.5), fuzz tests, [`Makefile`](Makefile), [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh), [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml). **Not audited.**
- **[ ] Phase 1 — LitVM testnet broadcast (pending)** — Needs [public RPC, chain ID, and zkLTC](https://docs.litvm.com/get-started-on-testnet/add-to-wallet). Then: `forge script` with [`contracts/.env`](contracts/.env.example), verify contracts, record addresses. See [`research/LITVM.md`](research/LITVM.md).
- **[~] Phase 2 — Nostr profile (in progress)** — Normative wire (kinds **31250–31251**, `content` JSON, `nostrKeyHash` binding) is in [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md); JSON fixtures are validated in CI under [`nostr/`](nostr/). Demo CLIs under [`scripts/`](scripts/) use the same kinds and shapes. Local stack: `make test-full-stack`. **Live relay walkthrough:** [`research/E2E_NOSTR_DEMO.md`](research/E2E_NOSTR_DEMO.md) (`scripts/nostr_watch.py` + `publish_grievance.py`).
- **[ ] Phase 3 — End-to-end integration** — Nostr discovery → Tor → MWixnet round → L2 settlement / slash path.
- **[x] Phase 8 — Testnet packaging (local complete)** — [`PHASE_8_TESTNET_RELEASE.md`](PHASE_8_TESTNET_RELEASE.md): `mlnd` Dockerfile, `make build` / `make docker-build`, `make testnet-smoke`, [`mlnd/.env.example`](mlnd/.env.example), GitHub Releases on `v*` tags ([`.github/workflows/mlnd-release.yml`](.github/workflows/mlnd-release.yml)). **Binaries attach on tag push; LitVM testnet RPC still pending for live operator runs.**
- **[x] Phase 9 — Enablement and hardening (operator packaging)** — [`PHASE_9_ENABLEMENT.md`](PHASE_9_ENABLEMENT.md): Docker Compose ([`docker-compose.yml`](docker-compose.yml)), [`.env.compose.example`](.env.compose.example), NDJSON bridge ops with patched `coinswapd`, defense and Nostr runbooks.
- **[~] Phase 10 — Taker client (`mln-cli`)** — [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md): Scout (Nostr 31250 + LitVM verification), Pathfind (3-hop route), Forger (Tor dry-run + HTTP POST of route JSON to a local **MLN sidecar** URL; onion build remains in `coinswapd` fork/proxy). Build: `make build-mln-cli`. Shared wire types: [`mlnd/pkg/makerad`](mlnd/pkg/makerad).
- **[~] Phase 11 — Taker wallet (Wails)** — Desktop GUI in [`mln-cli/desktop/`](mln-cli/desktop/) (React + Wails v2, `wails` build tag). Build: `make build-mln-wallet`. Developer notes: [`mln-cli/desktop/README.md`](mln-cli/desktop/README.md).

### Phase 1 local (already shipped)

Handoff checklist for anyone picking up the repo:

- **Spec helpers:** [`contracts/src/EvidenceLib.sol`](contracts/src/EvidenceLib.sol) (`evidenceHash`, `grievanceId` per appendix 13.5); [`contracts/test/EvidenceHash.t.sol`](contracts/test/EvidenceHash.t.sol).
- **Fuzz:** [`contracts/test/FuzzRegistry.t.sol`](contracts/test/FuzzRegistry.t.sol), [`contracts/test/FuzzGrievanceCourt.t.sol`](contracts/test/FuzzGrievanceCourt.t.sol).
- **Local deploy:** [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh) (run Anvil separately), [`Makefile`](Makefile) (`make contracts-test`, `make deploy-local`, `make test-grievance`), [`contracts/deployments/anvil-local.example.json`](contracts/deployments/anvil-local.example.json) — generated `anvil-local.json` stays gitignored.
- **CI:** [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml) (`forge build` / `forge test` via Docker Foundry).

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

1. **LitVM testnet** — When [official RPC and chain ID](https://docs.litvm.com/get-started-on-testnet/add-to-wallet) are published, fund a throwaway deployer with testnet `zkLTC`, copy [`contracts/.env.example`](contracts/.env.example) → `contracts/.env`, and run [`forge script`](contracts/README.md) with `--broadcast`. Record deployed addresses.
2. **Judicial economics** — [`GrievanceCourt`](contracts/src/GrievanceCourt.sol) remains a **scaffold** (bond refunds, no real slash split). Harden for a testnet demo or keep **non-production** until reviewed.
3. **Optional** — Static analysis (e.g. Slither) on `contracts/src/`.
4. **Phase 2 (in progress)** — Keep [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md), [`nostr/fixtures/`](nostr/), and [`scripts/`](scripts/) in lockstep; run `make test-full-stack` for local grievance + Nostr pointer validation. Finalize addresses/tags once LitVM testnet registry values are stable.

Appendix 13 hashing is implemented in [`contracts/src/EvidenceLib.sol`](contracts/src/EvidenceLib.sol) and covered by [`contracts/test/EvidenceHash.t.sol`](contracts/test/EvidenceHash.t.sol).

---

This repository holds the **product specification**, research notes, and Cursor configuration—**not** a production implementation yet (spec v0.1, draft).

## Documentation

| Document | Purpose |
| -------- | ------- |
| [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) | Full architecture, economics, roadmap, evidence preimage (appendix 13), MWEB appendix (14), open questions |
| [`AGENTS.md`](AGENTS.md) | Contributor / agent orientation (layer boundaries, canonical sources) |
| [`contracts/README.md`](contracts/README.md) | Solidity layout, local Anvil deploy, `make contracts-test` |
| [`Makefile`](Makefile) | Docker Foundry: `contracts-build`, `contracts-test`, `deploy-local`, `test-grievance`, `test-operator-smoke` (mlnd bridge + golden grievance; see [`PHASE_7_END_TO_END.md`](PHASE_7_END_TO_END.md)); `build`, `build-mln-cli`, `build-mln-wallet` (Wails taker GUI; see [`mln-cli/desktop/README.md`](mln-cli/desktop/README.md)), `docker-build`, `testnet-smoke` ([`PHASE_8_TESTNET_RELEASE.md`](PHASE_8_TESTNET_RELEASE.md)) |
| [`PHASE_9_ENABLEMENT.md`](PHASE_9_ENABLEMENT.md) | Operator packaging: Compose, env template, NDJSON bridge + `coinswapd`, defense and Nostr ops |
| [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md) | Taker CLI (`mln-cli`): Scout, Pathfind, Forger (dry-run + sidecar POST); env and trust model |
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

Rules under [`.cursor/rules/`](.cursor/rules/), skills under [`.cursor/skills/`](.cursor/skills/).

## License

Not specified; add a `LICENSE` when you publish.
