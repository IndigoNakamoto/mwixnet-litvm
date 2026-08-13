# Live coinswap attempt — 2026-08-12

**Bar:** **Proof A** (lab shuffle). Not Proof B.

Proof A means three mixnode `coinswapd` processes complete peel → `swap_forward` / `swap_backward` → one aggregated MWEB tx, and the dest wallet scans a **new same-value coin**. Three Tor HS on one host is enough. It does **not** show “one honest mixer.”

Proof B (stranger / 1 LTC / published first hop / 1-of-N honest) stays **parked**. Do not claim it from this file. **Do not** put CoinSwap in the external Tauri Mac wallet until a file like this is a **pass**.

**Outcome:** **Not a pass.** No funded 3-hop MWEB round; dest wallet did not scan a new coin. This session shipped the 3-mix gate (directory, Scout-free route, same-value reject, JSON without LitVM) and ran preflight. It did **not** spend mainnet MWEB.

**LitVM / Nostr / `mlnd`:** not used. No scout, no registry `eth_call`, no maker ads.

## Environment

- **Workspace:** `mwixnet-litvm` (local checkout).
- **Secrets:** No mainnet MWEB scan/spend keys, no `ltcmweb1…` material in the shell or this file.

## 1. `make phase3-operator-preflight`

**Result:** **Passed** (exit 0).

- Tor SOCKS `127.0.0.1:9050`: TCP connect OK.
- HTTPS via Tor to `https://check.torproject.org/api/ip`: `"IsTor":true`.
- Printed `HTTP_PROXY=socks5h://127.0.0.1:9050` template for `coinswapd-research`.

## 2. Scout-free route JSON

**Result:** **Passed**.

```text
./bin/mln-cli route from-directory -directory deploy/mixnet.directory.example.json -out route.json
```

Wrote `route.json` with three hops, `feeMinSat=10000`, `amountSat=10000000` (lab denom 0.1 LTC). File contains **no** `operator`, `epochId`, `accuser`, or `swapId`.

Unit coverage: `go test` in `research/coinswapd` (including `mlnroute` forger JSON without LitVM → `mweb_submitRoute` stub), `mln-cli/internal/mixnetdir`, `mln-cli/internal/forger`, `mln-sidecar/internal/mweb`.

## 3. Same-value gate

**Result:** **Code in-tree**, not exercised on live onions.

`performSwap` rejects a batch when recorded `mweb_submitRoute` amounts differ (`research/coinswapd/swap_denom.go`). Tests: `TestRequireSameDenomination_rejectMixed`.

## 4. Three mixnodes + funded submit

**Result:** **Not run.**

Requires operator-owned `-k` / `-a` / scan+spend secrets, an exact `amountSat` UTXO, three Tor HS, hop fees covering `backward()`, and `mweb_runBatch` without `-mweb-dev-clear-pending-after-batch`. None of that wallet material was available in this session (same class of blocker as [`LIVE_COINSWAP_ATTEMPT_2026-04-15.md`](LIVE_COINSWAP_ATTEMPT_2026-04-15.md) item 2, without repeating the Scout-vs-LitVM mistake).

## What would make the next Proof A a pass

1. Three `coinswapd-research` mixnodes + Tor HS; taker **client** (`-mln-local-taker` + sidecar).
2. Real directory (not the example placeholders) with matching `swapX25519PubHex` and `feeMinSat` ≥ 10000.
3. Exact `amountSat` MWEB coin; `mln-cli forger` without LitVM flags; epoch `mweb_runBatch`.
4. Dest wallet scans a new same-value coin. Log has no LitVM, Nostr, or `mlnd`.
5. New dated file labeled **Proof A**. Do not overwrite this one.

**Proof B** remains parked until after a Proof A pass.

---

*This file does not claim README Phase 3-mix is complete. It does not claim Proof B.*
