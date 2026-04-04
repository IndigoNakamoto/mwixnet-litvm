# Agent instructions (mwixnet-litvm)

## What this project is

**MLN stack (working title):** Litecoin **MWEB** for CoinSwap-style mixing, **LitVM** for registry / stake / slashing / grievances, **Nostr** for discovery and gossip, **Tor** for transport. See `PRODUCT_SPEC.md` for the full architecture, threat model, and phased roadmap.

## Where truth lives

| Topic | Canonical source |
|--------|------------------|
| Product, layers, roadmap, open questions | `PRODUCT_SPEC.md` |
| Grievance `evidenceHash` preimage (LitVM correlators) | `PRODUCT_SPEC.md` §13 |
| LitVM Foundry contracts and testnet notes | `contracts/`, `research/LITVM.md` |
| Accepted code review + threat model (scaffold risks, ops, CI gaps) | `research/THREAT_MODEL_MLN.md` |
| Red-team narratives + tabletop exercise (extends threat model; not a separate audit) | `research/RED_TEAM_MLN.md` |
| Maker exit queue / cooldown / slashing window (registry + grievances) | `PRODUCT_SPEC.md` section 5.1.1, `contracts/src/MwixnetRegistry.sol`, `PHASE_15_ECONOMIC_HARDENING.md` |
| Appendix 13 hash helpers (`evidenceHash`, `grievanceId`) | `contracts/src/EvidenceLib.sol` |
| Nostr wire profile (MLN kinds, tags, `content` JSON, `nostrKeyHash` binding) | `research/NOSTR_MLN.md` |
| Archived Nostr doc stub (historical `NOSTR_EVENTS` filename) | `research/NOSTR_EVENTS.md` |
| User stories, coordination / epochs, wallet route policy (PoC), taker-first UX principles | `research/USER_STORIES_MLN.md` |
| Wallet UX wireframes (taker / maker) | `research/WALLET_TAKER_FLOW_V1.md`, `research/WALLET_MAKER_FLOW_V1.md` |
| Maker dashboard setup (operator surface) | `mlnd/MAKER_DASHBOARD_SETUP.md` |
| Product/UX design pass (personas, trust UI, IA, onboarding) | `.cursor/skills/mln-web3-product-design/SKILL.md` |
| Local Anvil deploy / CI | `scripts/deploy-local-anvil.sh`, `.github/workflows/contracts.yml` |
| Phase 15 LitVM economics (slash, bonds, exit locks; Foundry invariants + Slither in CI) | `PHASE_15_ECONOMIC_HARDENING.md`, `contracts/test/InvariantRegistryStake.t.sol`, `.github/workflows/contracts.yml`, `contracts/src/MwixnetRegistry.sol`, `contracts/src/GrievanceCourt.sol` |
| Phase 16 public testnet readiness (RPC_URL deploy, verification env, operator compose without Anvil/local relay) | `PHASE_16_PUBLIC_TESTNET.md`, `deploy/docker-compose.testnet.yml`, `deploy/.env.testnet.example`, `contracts/.env.example`, `mln-cli/internal/config/settings.go` |
| Local E2E stack (Anvil + Nostr relay + `mln-sidecar` + 3× `mlnd`, bootstrap) | `PHASE_12_E2E_CRUCIBLE.md`, `deploy/docker-compose.e2e.yml`, `scripts/e2e-bootstrap.sh`, `mln-sidecar/` |
| Phase 3a MWEB handoff (`mln-sidecar -mode=rpc`, stub or fork; no official LitVM testnet) | `PHASE_3_MWEB_HANDOFF_SLICE.md`, `scripts/e2e-mweb-handoff-stub.sh`, `deploy/docker-compose.e2e.sidecar-rpc.yml`, `make build-mw-rpc-stub`; rpc wire regression: `mln-sidecar/internal/mweb/rpc_bridge_test.go`, `research/coinswapd/mlnroute/sidecar_wire_test.go` |
| MWEB tx / onion baseline vs Grin (normative for `coinswapd` path) | `PRODUCT_SPEC.md` §14 |
| How `coinswapd` is structured (RPC, onion JSON, code paths) | `research/COINSWAPD_TEARDOWN.md`; fork + `mweb_*` implementation in `research/coinswapd/` |
| `mweb_submitRoute` / `mweb_getBalance` fork contract and MLN → `onion.Onion` mapping | `research/COINSWAPD_MLN_FORK_SPEC.md` |
| Taker CLI (`mln-cli`); maker onboard (`mln-cli maker onboard`); Wails taker wallet (`mln-cli/desktop/`, build tag `wails`); shared maker-ad structs for `mlnd` + client; Forger → MLN HTTP sidecar (`GET /v1/balance`, `POST` route JSON, not vanilla `swap_Swap`); mock sidecar for local E2E (`mln-sidecar`); optional self-as-N2 routing (Phase 14); optional `mlnd` loopback Maker dashboard (`MLND_DASHBOARD_ADDR`) | `PHASE_10_TAKER_CLI.md`, `PHASE_14_SELF_INCLUSION.md`, `mln-cli/desktop/README.md`, `mlnd/MAKER_DASHBOARD_SETUP.md`, `research/COINSWAPD_TEARDOWN.md` (sidecar + `swap_forward`), `research/COINSWAPD_MLN_FORK_SPEC.md`, `mln-cli/internal/forger/`, `mln-cli/internal/takerflow/`, `mln-cli/internal/pathfind/`, `mlnd/pkg/makerad`, `mln-sidecar/` |
| Documentation sync pass (README `PHASE_*` index parity, git-aligned status blurbs, PoC vs production, link and CI/RPC audit) | `.cursor/skills/doc-sync/SKILL.md`, `.cursor/rules/doc-sync.mdc` |
| Program hygiene (priorities, blockers, milestone/release readiness from canonical docs + git; not a second roadmap) | `.cursor/skills/mln-pm/SKILL.md` |
| Implementation Cursor Skills | mln-go-engineer, mln-contracts, mln-frontend-wails, mln-qa, mln-deployment, mln-observability, mln-architecture-diagrams | Handles Go daemons, Solidity, Wails/React desktop, testing, operator packaging, and architecture diagrams while respecting AGENTS.md boundaries and PRODUCT_SPEC.md |

Prefer quoting or linking paths into those docs instead of inventing APIs or economics.

## Layer boundaries (do not blur)

- **MWEB:** Privacy engine and, per baseline design, **per-hop routing fees** — not duplicated on EVM for the default path.
- **LitVM:** **Registry, bonds, slashing, grievances** — a judicial / economic layer. It must **not** try to verify full happy-path mix execution on-chain (too costly and metadata-leaky). See `PRODUCT_SPEC.md` §5–6.
- **Nostr:** Discovery and signed operational events — **not** authoritative for stake (LitVM is).
- **Tor:** IP-level anonymity; complements onion payloads, does not replace MW cryptography.

## Reference code

The team maintains **`research/coinswapd/`** as an in-repo fork of `ltcmweb/coinswapd` (MLN `mweb_*` RPC and route handoff). Upstream **ltcmweb** remains the baseline for comparison; merge or submodule updates are manual.

The teardown documents **entry points** and the **`ltcd`** dependency boundary (not `mwebd` as a direct import in that binary). When extending or comparing designs, align with the teardown before assuming Grin/mwixnet details map 1:1 to MWEB.

## Current phase

**Phase 1 (local):** contracts, `EvidenceLib`, fuzz and invariant tests, Slither on `contracts/**` in GitHub Actions, `make contracts-test`, `scripts/deploy-local-anvil.sh` — see `README.md` roadmap. **Implementation phases 10–16** (through public testnet readiness packaging) are **shipped in-tree**, including **`mln-cli maker onboard`** and an **optional `mlnd` Maker dashboard** (loopback by default); **LitVM testnet broadcast** on the official chain remains pending public RPC from LitVM. **Phase 3a** (MWEB route JSON → **`mln-sidecar -mode=rpc`** → **`mweb_submitRoute`**) is **verified** on the **`mw-rpc-stub`** path with **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** as of **2026-04-03** (`PHASE_3_MWEB_HANDOFF_SLICE.md`); full Phase 3 integration remains open. **Security docs:** `research/THREAT_MODEL_MLN.md` (accepted audit snapshot + threat tables), `research/RED_TEAM_MLN.md` (red-team narratives; extends the threat model).

**Phase 1 (testnet):** when LitVM publishes endpoints, broadcast and record addresses (`research/LITVM.md`, `PHASE_16_PUBLIC_TESTNET.md`).

Earlier: **protocol clarity** (`PRODUCT_SPEC.md` §9). **`evidenceHash` preimage** — spec §13 + `EvidenceLib`; validate against nodes before freezing production ABIs. **L1 inclusion proofs** for defenses: TBD (`PRODUCT_SPEC.md` §10).

## Editing norms

- Keep speculative implementation out of `PRODUCT_SPEC.md` unless it is clearly marked as draft / TBD.
- When adding new research notes under `research/`, cross-link from the teardown or spec if it changes agreed boundaries.
