# Agent instructions (mwixnet-litvm)

## What this project is

**MLN stack (working title):** Litecoin **MWEB** for CoinSwap-style mixing, **LitVM** for registry / stake / slashing / grievances, **Nostr** for discovery and gossip, **Tor** for transport. See `PRODUCT_SPEC.md` for the full architecture, threat model, and phased roadmap.

## Where truth lives

| Topic | Canonical source |
|--------|------------------|
| Product, layers, roadmap, open questions | `PRODUCT_SPEC.md` |
| Grievance `evidenceHash` preimage (LitVM correlators) | `PRODUCT_SPEC.md` §13 |
| LitVM Foundry contracts and testnet notes | `contracts/`, `research/LITVM.md` |
| MWEB tx / onion baseline vs Grin (normative for `coinswapd` path) | `PRODUCT_SPEC.md` §14 |
| How `coinswapd` is structured (RPC, onion JSON, code paths) | `research/COINSWAPD_TEARDOWN.md` (local clone optional under `research/coinswapd/`, gitignored) |

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

Solidity scaffolding lives in `contracts/` (Foundry); see `research/LITVM.md` and **Next steps** in `README.md` for what to do when resuming (testnet deploy, evidence alignment, hardening, then Nostr).

Earlier focus: **protocol clarity** (`PRODUCT_SPEC.md` §9). **`evidenceHash` preimage** is in `PRODUCT_SPEC.md` §13 — validate against nodes before freezing registry ABIs; **L1 inclusion proof format** for defenses remains TBD (`PRODUCT_SPEC.md` §10).

## Editing norms

- Keep speculative implementation out of `PRODUCT_SPEC.md` unless it is clearly marked as draft / TBD.
- When adding new research notes under `research/`, cross-link from the teardown or spec if it changes agreed boundaries.
