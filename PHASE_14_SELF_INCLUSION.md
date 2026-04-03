# Phase 14: Self-Inclusion UX (Participant-Observer)

This phase lets a **taker wallet** optionally fix **its own LitVM maker** as the **middle hop (N2)** in a three-hop route. The goal is UX and trust framing: you control one hop’s behavior while still using two external makers for N1 and N3.

**Normative context:** Maker ads and registry binding — [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md). Taker flow and sidecar contract — [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md). Hop-to-hop relay after entry — [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md) (`swap_forward` / `swap_backward`). Local stack — [`PHASE_12_E2E_CRUCIBLE.md`](PHASE_12_E2E_CRUCIBLE.md).

## Scope

| In scope | Out of scope |
|----------|----------------|
| Wallet settings: LitVM **operator** secp256k1 private key (hex), **Self-Included Routing** toggle | Encrypting or HSM-wrapping keys (document threat model; settings file is user-readable on disk) |
| Scout: mark verified row as **local** when `operator` matches derived address | Changing Nostr or registry wire formats |
| Pathfind: **Self-Route** mode — N1/N3 external, **N2 = self** (same min-fee + stake tie-break policy over valid triples) | `mln-sidecar` behavior ([`mln-sidecar/`](mln-sidecar/) stays a dumb route translator; **no** self-detection flag) |
| Wails: toggle + copy + gated **Build route** | Implementing peel/forward inside the sidecar (that remains **`mlnd` + `coinswapd`**) |
| `mln-cli pathfind`: `-self-included` + `MLN_OPERATOR_ETH_KEY` | New LitVM contracts |

## Security copy (assumption-scoped)

Self-included N2 means **you** are not outsourcing that hop’s integrity to a third party. **N1 and N3** remain external; timing, batching, and side-channel correlation are **not** automatically eliminated. Describe benefits in product copy with explicit assumptions; avoid unqualified “probability zero” claims in [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) unless marked draft/TBD per repo norms.

## Identity binding

- **Operator key:** Same **Ethereum-style ECDSA** secret that controls the **LitVM maker `operator` address** (the address in the maker-ad **d-tag** and registry). Hex: optional `0x` prefix, **64** hex characters (32 bytes).
- **Matching:** [`scout.VerifiedMaker`](mln-cli/internal/scout/scout.go) `Operator` (`common.Address`) is compared to `crypto.PubkeyToAddress(key.PublicKey)`.
- **Nostr `nostrKeyHash`:** Not required for row matching in v1; registry + ad already tie `operator` to the published Nostr key. Optional future: cross-check pubkey hash if the wallet also holds `MLND_NOSTR_NSEC`.

## Pathfind semantics

- **Default:** [`pathfind.PickRoute`](mln-cli/internal/pathfind/pathfind.go) — three **distinct** verified makers, min total `feeMinSat`, then max sum stake, random tie-break.
- **Self-Route:** [`pathfind.PickRouteSelfMiddle`](mln-cli/internal/pathfind/pathfind.go) — require **self** in the verified set and **at least two** other makers. Enumerate pairs `(N1, N3)` from externals (`N1 != N3`), score `fee(N1)+fee(self)+fee(N3)` then stake sum of the three, same tie-break as default.
- **Eligibility:** `len(verified) >= 3` and self appears in `verified`.

## Wails UX

- **Toggle label:** `Self-Included Routing`
- **Subtext:** `You will act as the middle hop in your own transactions, ensuring total privacy even if other nodes collude. (Requires local node to be active and staked).`
- **Settings:** Toggle + password-style field for operator key hex. Saving with toggle **on** requires a parseable key.
- **Scout table:** Show **Local** (or similar) when `verified[i].local` is true.

## `mln-cli` CLI

- **`pathfind`:** `-self-included` — use self-middle selection. Operator key from env **`MLN_OPERATOR_ETH_KEY`** (hex, same format as wallet).
- **`scout`:** If `MLN_OPERATOR_ETH_KEY` is set and valid, STATUS column shows `verified (local)` for the matching row.

## Phase 13 sidecar

**No protocol or code changes** to [`mln-sidecar`](mln-sidecar/): it does not inspect hop identity. Real N2 processing is **`swap_forward`** on your maker stack when your Tor URL appears in the onion path.

## Implementation checklist (for agents)

1. [`mln-cli/internal/config/settings.go`](mln-cli/internal/config/settings.go) — new fields + `ValidateSelfInclusion` + `TryOperatorAddress` / `OperatorEthereumAddress`.
2. [`mln-cli/internal/identity/`](mln-cli/internal/identity/) — parse hex secp256k1, derive address.
3. [`mln-cli/internal/scout/scout.go`](mln-cli/internal/scout/scout.go) — `Local` on `VerifiedMaker`.
4. [`mln-cli/internal/pathfind/pathfind.go`](mln-cli/internal/pathfind/pathfind.go) — `PickRouteSelfMiddle`.
5. [`mln-cli/internal/takerflow/takerflow.go`](mln-cli/internal/takerflow/takerflow.go) — annotate scout; branch `BuildRoute`.
6. [`mln-cli/desktop/app.go`](mln-cli/desktop/app.go) — validate self-inclusion on save.
7. [`mln-cli/desktop/frontend/src/App.jsx`](mln-cli/desktop/frontend/src/App.jsx) + [`App.js`](mln-cli/desktop/frontend/src/wailsjs/go/main/App.js) — toggle, key field, scout column.
8. [`mln-cli/cmd/mln-cli/main.go`](mln-cli/cmd/mln-cli/main.go) — scout/pathfind env + flag.
9. Tests in `pathfind` / `config` / `identity`.
10. [`README.md`](README.md) roadmap line; [`AGENTS.md`](AGENTS.md) table row.
