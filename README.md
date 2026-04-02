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
- **[ ] Phase 1 — LitVM testnet** *(current focus)* — Foundry project in [`contracts/`](contracts/) (`MwixnetRegistry`, `GrievanceCourt`). Tooling and RPC notes: [`research/LITVM.md`](research/LITVM.md). Deploy to testnet when official RPC/chain ID are published.
- **[ ] Phase 2 — Nostr profile** — Event kinds / NIPs for maker ads and discovery.
- **[ ] Phase 3 — End-to-end integration** — Nostr discovery → Tor → MWixnet round → L2 settlement / slash path.

### Next steps (pickup after a break)

1. **LitVM testnet** — When [official RPC and chain ID](https://docs.litvm.com/get-started-on-testnet/add-to-wallet) are published, fund a throwaway deployer with testnet `zkLTC`, copy [`contracts/.env.example`](contracts/.env.example) → `contracts/.env`, and run [`forge script`](contracts/README.md) with `--broadcast`. Record deployed addresses (README or a small `contracts/deployments/` note).
2. **Spec ↔ contracts** — Re-read [appendix 13](PRODUCT_SPEC.md) and confirm off-chain `evidenceHash` matches [`GrievanceCourt`](contracts/src/GrievanceCourt.sol) (only `bytes32` on-chain). Adjust ABIs or spec if anything diverges.
3. **Judicial economics** — [`GrievanceCourt`](contracts/src/GrievanceCourt.sol) is a **scaffold**: bond handling and slash/bounty splits are minimal. Either harden for a testnet demo or explicitly label as **non-production** until reviewed.
4. **Tooling** — Optionally run `forge fmt` and a static analyzer (e.g. Slither) on `contracts/src/`.
5. **Phase 2** — Nostr event profile / NIPs once registry + deploy story are stable, so ads and stake pointers do not churn.

---

This repository holds the **product specification**, research notes, and Cursor configuration—**not** a production implementation yet (spec v0.1, draft).

## Documentation

| Document | Purpose |
| -------- | ------- |
| [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) | Full architecture, economics, roadmap, evidence preimage (appendix 13), MWEB appendix (14), open questions |
| [`AGENTS.md`](AGENTS.md) | Contributor / agent orientation (layer boundaries, canonical sources) |
| [`contracts/README.md`](contracts/README.md) | Solidity layout, `forge build` / test |
| [`research/LITVM.md`](research/LITVM.md) | LitVM testnet, env, Docker Foundry, links |
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
