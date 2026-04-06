# MLN capabilities — questions and answers

**Audience:** Anyone evaluating the repo, onboarding operators, or scoping integrations.  
**Not a substitute for:** [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) (full architecture and economics), [`research/THREAT_MODEL_MLN.md`](../research/THREAT_MODEL_MLN.md) (risks), or an independent security audit.

---

## Production vs proof-of-concept

**Q: Is this production-ready?**  
**A:** **No.** The README states that phases 1–16 are **feature-complete in-tree** for documented bring-up paths, but **production** still depends on **official LitVM testnet broadcast** (public RPC), an **independent audit**, and **production-hardened MWEB / `coinswapd` integration** (see [`README.md`](../README.md) roadmap disclaimer). Treat the tree as a **research and integration PoC** until those gates clear.

**Q: What does “verified in-repo” mean for Phase 3a?**  
**A:** The **MWEB handoff path** (`mln-sidecar -mode=rpc` → `mweb_submitRoute` / `mweb_runBatch` / `mweb_getRouteStatus`) is exercised end-to-end with **`mw-rpc-stub`** and documented **`E2E_MWEB_FULL=1`** / optional **`MWEB_RPC_BACKEND=coinswapd`** / **`E2E_MWEB_FUNDED=1`** scripts. That proves the **wiring and fork contract**, not that **live multi-hop Tor P2P** or **public LitVM slash** is done. Details: [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../PHASE_3_MWEB_HANDOFF_SLICE.md).

---

## What the stack is

**Q: What problem does MLN solve?**  
**A:** A **trust-minimized** way to run **CoinSwap-style MWEB mixing** without a single human coordinator: **MWEB** does the mix math and (baseline) **per-hop fees**; **LitVM** holds **stake and grievances**; **Nostr** carries **discovery and gossip**; **Tor** hides **IP-level** linkage. See [`README.md`](../README.md) architecture bullets and [`AGENTS.md`](../AGENTS.md) layer boundaries.

**Q: Does LitVM execute my mix on-chain?**  
**A:** **No** for the happy path. LitVM is the **registry / bonds / slashing / grievance** layer. Verifying every successful mix on-chain is **out of scope** by design (cost and metadata). See [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) sections 5–6.

**Q: Is Nostr authoritative for who is staked?**  
**A:** **No.** Maker ads (kind **31250**) are **hints**. Wallets should confirm **stake and identity** via **LitVM RPC**. Kind **31251** grievance pointers must be **verified on-chain**. Normative wire: [`research/NOSTR_MLN.md`](../research/NOSTR_MLN.md).

---

## Capabilities by component

**Q: What ships for LitVM contracts?**  
**A:** Foundry project under [`contracts/`](../contracts/): `MwixnetRegistry`, `GrievanceCourt`, `EvidenceLib` (evidence hash helpers per product spec appendix 13), fuzz/invariant tests, Slither in CI on contract changes. **Contracts are not audited.** Local deploy: [`scripts/deploy-local-anvil.sh`](../scripts/deploy-local-anvil.sh), [`Makefile`](../Makefile). Testnet packaging: [`PHASE_16_PUBLIC_TESTNET.md`](../PHASE_16_PUBLIC_TESTNET.md).

**Q: What ships for Nostr?**  
**A:** Normative profile for kinds **31250–31251**, golden fixtures, CI validation (`nostr/validate_fixtures.py`), `mln-cli` Scout with deployment filters, `mlnd` ad publishing. Playbook: [`PHASE_2_NOSTR.md`](../PHASE_2_NOSTR.md).

**Q: What ships for takers?**  
**A:** **`mln-cli`**: Scout, pathfind (including optional self-included routing), forger posting route JSON to **`mln-sidecar`** (not vanilla upstream `swap_Swap` alone). **`mln-cli maker onboard`**: LitVM `deposit` / `registerMaker` plan or execute (dry-run by default). Optional **Wails** desktop wallet (`make build-mln-wallet`). See [`PHASE_10_TAKER_CLI.md`](../PHASE_10_TAKER_CLI.md).

**Q: Where does PSBTv2 MWEB fit in?**  
**A:** **PSBTv2** adds **MWEB extensions** for **interoperable partial signing** across wallets (reference definitions in **ltcd**; desktop implementations include **[Sparrow-LTC](https://github.com/sparrow-ltc/sparrow)**). That is **separate** from this repo’s default mix handoff, which uses **`mweb_*` JSON-RPC** and **route / onion JSON** via **`mln-sidecar`** and **`coinswapd`**. PSBT matters for **funding, change, and co-signing** workflows that use PSBT carriers, not for the documented **`E2E_MWEB_*`** stub path unless you explicitly integrate it. See [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) appendix **14.0** and Tier B.

**Q: What ships for makers / operators?**  
**A:** **`mlnd`** daemon, Docker/Compose paths, optional loopback **Maker dashboard** when `MLND_DASHBOARD_ADDR` is set, NDJSON receipt bridge scaffolds per phase playbooks, release workflow for `mlnd` binaries. Operator notes: [`PHASE_9_ENABLEMENT.md`](../PHASE_9_ENABLEMENT.md), [`mlnd/MAKER_DASHBOARD_SETUP.md`](../mlnd/MAKER_DASHBOARD_SETUP.md), [`research/PHASE_3_TOR_OPERATOR_LAB.md`](../research/PHASE_3_TOR_OPERATOR_LAB.md).

**Q: What is `mln-sidecar`?**  
**A:** HTTP shim: **`GET /v1/balance`**, **`POST /v1/swap`**, route batch/status endpoints. **`-mode=mock`** for local E2E without a real node; **`-mode=rpc`** forwards to **`coinswapd`** JSON-RPC (`mweb_submitRoute`, etc.). See [`mln-sidecar/README.md`](../mln-sidecar/README.md).

**Q: Where is the MWEB engine code?**  
**A:** Reference fork and **`mweb_*`** RPC extensions live in **[`research/coinswapd/`](../research/coinswapd/)**. RPC and onion shapes: [`research/COINSWAPD_TEARDOWN.md`](../research/COINSWAPD_TEARDOWN.md), [`research/COINSWAPD_MLN_FORK_SPEC.md`](../research/COINSWAPD_MLN_FORK_SPEC.md).

---

## Local and CI workflows

**Q: How do I run contracts tests?**  
**A:** From repo root: `make contracts-test` (see [`contracts/README.md`](../contracts/README.md)).

**Q: How do I run the full local E2E stack?**  
**A:** `make test-full-stack` and [`PHASE_12_E2E_CRUCIBLE.md`](../PHASE_12_E2E_CRUCIBLE.md) (Anvil + Nostr relay + sidecar + three `mlnd` makers).

**Q: What regression anchors should releasers know?**  
**A:** README lists **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** before merging certain changes or tagging **`v*`**; Nostr fixture validation with `python3 nostr/validate_fixtures.py`. See [`README.md`](../README.md) release section.

---

## Privacy and security (high level)

**Q: What are the main residual risks?**  
**A:** Read [`research/THREAT_MODEL_MLN.md`](../research/THREAT_MODEL_MLN.md) and [`research/RED_TEAM_MLN.md`](../research/RED_TEAM_MLN.md). Examples called out in product docs: **discovery surveillance** on Nostr, **bridge/contract** risk, **honest-maker** assumptions in path privacy, and **TBD** paths for **L1/MWEB inclusion proofs** in some grievance defenses.

**Q: Does the stack guarantee anonymity if all makers collude?**  
**A:** The product spec ties meaningful privacy to **at least one honest** hop in the MWixnet-style model; **all colluding** makers is a different threat class. See [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) success criteria and threat table.

---

## Where to go next

| Goal | Start here |
| ---- | ---------- |
| Full product intent | [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) |
| Contributor / agent map | [`AGENTS.md`](../AGENTS.md) |
| Layer diagram (Nostr + LitVM) | [`docs/MLN_NOSTR_LITVM_ARCHITECTURE.md`](MLN_NOSTR_LITVM_ARCHITECTURE.md) |
| Phase-by-phase implementation | [`README.md`](../README.md) roadmap + linked `PHASE_*.md` files |
| Pairing AI tools safely | [`docs/AGENT_HANDOFF.md`](AGENT_HANDOFF.md) |
