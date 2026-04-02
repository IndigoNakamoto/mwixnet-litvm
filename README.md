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
- **[~] Phase 2 — Nostr profile (in progress)** — Event kinds / NIPs for maker ads and grievance pointers are defined in [`research/NOSTR_EVENTS.md`](research/NOSTR_EVENTS.md); local full-stack validation is available via `make test-full-stack`.
- **[ ] Phase 3 — End-to-end integration** — Nostr discovery → Tor → MWixnet round → L2 settlement / slash path.

### Phase 1 local (already shipped)

Handoff checklist for anyone picking up the repo:

- **Spec helpers:** [`contracts/src/EvidenceLib.sol`](contracts/src/EvidenceLib.sol) (`evidenceHash`, `grievanceId` per appendix 13.5); [`contracts/test/EvidenceHash.t.sol`](contracts/test/EvidenceHash.t.sol).
- **Fuzz:** [`contracts/test/FuzzRegistry.t.sol`](contracts/test/FuzzRegistry.t.sol), [`contracts/test/FuzzGrievanceCourt.t.sol`](contracts/test/FuzzGrievanceCourt.t.sol).
- **Local deploy:** [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh) (run Anvil separately), [`Makefile`](Makefile) (`make contracts-test`, `make deploy-local`, `make test-grievance`), [`contracts/deployments/anvil-local.example.json`](contracts/deployments/anvil-local.example.json) — generated `anvil-local.json` stays gitignored.
- **CI:** [`.github/workflows/contracts.yml`](.github/workflows/contracts.yml) (`forge build` / `forge test` via Docker Foundry).

### Next steps (pickup after a break)

1. **LitVM testnet** — When [official RPC and chain ID](https://docs.litvm.com/get-started-on-testnet/add-to-wallet) are published, fund a throwaway deployer with testnet `zkLTC`, copy [`contracts/.env.example`](contracts/.env.example) → `contracts/.env`, and run [`forge script`](contracts/README.md) with `--broadcast`. Record deployed addresses.
2. **Judicial economics** — [`GrievanceCourt`](contracts/src/GrievanceCourt.sol) remains a **scaffold** (bond refunds, no real slash split). Harden for a testnet demo or keep **non-production** until reviewed.
3. **Optional** — Static analysis (e.g. Slither) on `contracts/src/`.
4. **Phase 2 (in progress)** — Nostr event profile landed in [`research/NOSTR_EVENTS.md`](research/NOSTR_EVENTS.md); run `make test-full-stack` for local grievance + Nostr pointer validation, then finalize addresses/tags once LitVM testnet registry values are stable.

Appendix 13 hashing is implemented in [`contracts/src/EvidenceLib.sol`](contracts/src/EvidenceLib.sol) and covered by [`contracts/test/EvidenceHash.t.sol`](contracts/test/EvidenceHash.t.sol).

---

This repository holds the **product specification**, research notes, and Cursor configuration—**not** a production implementation yet (spec v0.1, draft).

## Documentation

| Document | Purpose |
| -------- | ------- |
| [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) | Full architecture, economics, roadmap, evidence preimage (appendix 13), MWEB appendix (14), open questions |
| [`AGENTS.md`](AGENTS.md) | Contributor / agent orientation (layer boundaries, canonical sources) |
| [`contracts/README.md`](contracts/README.md) | Solidity layout, local Anvil deploy, `make contracts-test` |
| [`Makefile`](Makefile) | Docker Foundry: `contracts-build`, `contracts-test`, `deploy-local`, `test-grievance` |
| [`research/LITVM.md`](research/LITVM.md) | LitVM testnet, env, Docker Foundry, Phase 1 local |
| [`research/NOSTR_EVENTS.md`](research/NOSTR_EVENTS.md) | Draft Nostr event kinds (`30001`, `31001`, `31002`) for maker ads and grievance visibility |
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
