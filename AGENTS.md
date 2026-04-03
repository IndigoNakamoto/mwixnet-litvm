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
| Maker exit queue / cooldown / slashing window (registry + grievances) | `PRODUCT_SPEC.md` section 5.1.1, `contracts/src/MwixnetRegistry.sol`, `PHASE_15_ECONOMIC_HARDENING.md` |
| Appendix 13 hash helpers (`evidenceHash`, `grievanceId`) | `contracts/src/EvidenceLib.sol` |
| Nostr wire profile (MLN kinds, tags, `content` JSON, `nostrKeyHash` binding) | `research/NOSTR_MLN.md` |
| Archived Nostr doc stub (historical `NOSTR_EVENTS` filename) | `research/NOSTR_EVENTS.md` |
| User stories, coordination / epochs, wallet route policy (PoC) | `research/USER_STORIES_MLN.md` |
| Wallet UX wireframes (taker / maker) | `research/WALLET_TAKER_FLOW_V1.md`, `research/WALLET_MAKER_FLOW_V1.md` |
| Local Anvil deploy / CI | `scripts/deploy-local-anvil.sh`, `.github/workflows/contracts.yml` |
| Phase 15 LitVM economics (slash, bonds, exit locks; Foundry invariants + Slither in CI) | `PHASE_15_ECONOMIC_HARDENING.md`, `contracts/test/InvariantRegistryStake.t.sol`, `.github/workflows/contracts.yml`, `contracts/src/MwixnetRegistry.sol`, `contracts/src/GrievanceCourt.sol` |
| Local E2E stack (Anvil + Nostr relay + `mln-sidecar` + 3× `mlnd`, bootstrap) | `PHASE_12_E2E_CRUCIBLE.md`, `deploy/docker-compose.e2e.yml`, `scripts/e2e-bootstrap.sh`, `mln-sidecar/` |
| MWEB tx / onion baseline vs Grin (normative for `coinswapd` path) | `PRODUCT_SPEC.md` §14 |
| How `coinswapd` is structured (RPC, onion JSON, code paths) | `research/COINSWAPD_TEARDOWN.md` (local clone optional under `research/coinswapd/`, gitignored) |
| Taker CLI (`mln-cli`); Wails taker wallet (`mln-cli/desktop/`, build tag `wails`); shared maker-ad structs for `mlnd` + client; Forger → MLN HTTP sidecar (`GET /v1/balance`, `POST` route JSON, not vanilla `swap_Swap`); mock sidecar for local E2E (`mln-sidecar`); optional self-as-N2 routing (Phase 14) | `PHASE_10_TAKER_CLI.md`, `PHASE_14_SELF_INCLUSION.md`, `mln-cli/desktop/README.md`, `research/COINSWAPD_TEARDOWN.md` (sidecar + `swap_forward`), `mln-cli/internal/forger/`, `mln-cli/internal/takerflow/`, `mln-cli/internal/pathfind/`, `mlnd/pkg/makerad`, `mln-sidecar/` |
| Documentation sync pass (README `PHASE_*` index parity, git-aligned status blurbs, PoC vs production, link and CI/RPC audit) | `.cursor/skills/doc-sync/SKILL.md`, `.cursor/rules/doc-sync.mdc` |

Prefer quoting or linking paths into those docs instead of inventing APIs or economics.

## Layer boundaries (do not blur)

- **MWEB:** Privacy engine and, per baseline design, **per-hop routing fees** — not duplicated on EVM for the default path.
- **LitVM:** **Registry, bonds, slashing, grievances** — a judicial / economic layer. It must **not** try to verify full happy-path mix execution on-chain (too costly and metadata-leaky). See `PRODUCT_SPEC.md` §5–6.
- **Nostr:** Discovery and signed operational events — **not** authoritative for stake (LitVM is).
- **Tor:** IP-level anonymity; complements onion payloads, does not replace MW cryptography.

## Reference code

The team uses a **fork** of `ltcmweb/coinswapd` for implementation work and Q&A; keep that tree **outside this repo** or under `research/coinswapd/` locally (ignored by git). Upstream **ltcmweb** remains the baseline for what the public reference does.

The teardown documents **entry points** and the **`ltcd`** dependency boundary (not `mwebd` as a direct import in that binary). When extending or comparing designs, align with the teardown before assuming Grin/mwixnet details map 1:1 to MWEB.

## Current phase

**Phase 1 (local):** contracts, `EvidenceLib`, fuzz and invariant tests, Slither on `contracts/**` in GitHub Actions, `make contracts-test`, `scripts/deploy-local-anvil.sh` — see `README.md` roadmap. **Implementation phases 10–15** (taker CLI, wallet, E2E crucible, sidecar, self-inclusion, economic hardening) are **shipped in-tree**; **LitVM testnet broadcast** remains pending public RPC.

**Phase 1 (testnet):** when LitVM publishes endpoints, broadcast and record addresses (`research/LITVM.md`).

Earlier: **protocol clarity** (`PRODUCT_SPEC.md` §9). **`evidenceHash` preimage** — spec §13 + `EvidenceLib`; validate against nodes before freezing production ABIs. **L1 inclusion proofs** for defenses: TBD (`PRODUCT_SPEC.md` §10).

## Editing norms

- Keep speculative implementation out of `PRODUCT_SPEC.md` unless it is clearly marked as draft / TBD.
- When adding new research notes under `research/`, cross-link from the teardown or spec if it changes agreed boundaries.
