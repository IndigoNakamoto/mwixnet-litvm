# Live coinswap attempt — 2026-04-15

**Scope:** Operator-playbook path after LitVM testnet (chain **4441**) addresses landed in `README.md` / `deploy/litvm-addresses.generated.env` (see repo state at this date).

**Outcome:** **Not completed.** No successful **`mweb_submitRoute`** → **`mweb_runBatch`** → **`pendingOnions=0`** on **`coinswapd-research`**. Below is what was actually run and what blocked a full funded attempt.

## Environment

- **Workspace:** `mwixnet-litvm` (local checkout).
- **LitVM RPC (HTTP):** `https://liteforge.rpc.caldera.xyz/http` (from `README.md`).
- **Registry / court:** `deploy/litvm-addresses.generated.env` (`MLND_REGISTRY_ADDR`, `MLND_COURT_ADDR`, `MLND_LITVM_CHAIN_ID=4441`).
- **Nostr (scout sample):** `wss://relay.damus.io` (public; not team-curated).
- **Secrets:** No mainnet MWEB scan/spend keys, no `ltcmweb1…` fee/receive material — by design nothing wallet-bearing was pasted into the shell or this file.

## 1. `make phase3-operator-preflight`

**Result:** **Passed** (exit 0).

- Tor SOCKS **`127.0.0.1:9050`**: TCP connect OK.
- **`curl`** via Tor to `https://check.torproject.org/api/ip`: **`"IsTor":true`** (truncated JSON in log).
- Script printed recommended **`HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`** for **`coinswapd-research`** (`.onion` inter-hop JSON-RPC).

## 2. Builds

**Result:** **Passed** (exit 0).

```text
make build-research-coinswapd build-mln-sidecar build-mln-cli
```

Artifacts: `bin/coinswapd-research`, `bin/mln-sidecar`, `bin/mln-cli`.

## 3. `coinswapd-research` (non-stub) — end-to-end handoff

**Result:** **Not run** to live JSON-RPC completion.

**Reason:** The supported automation path is `MWEB_RPC_BACKEND=coinswapd ./scripts/e2e-mweb-handoff-stub.sh` (see `PHASE_3_MWEB_HANDOFF_SLICE.md`). That path requires **`COINSWAPD_FEE_MWEB`** as a **full mainnet MWEB** stealth address with prefix **`ltcmweb1`** (Bech32 per `ltcmweb/ltcd`); the script **rejects** documentation placeholders. Starting the fork also implies Neutrino sync against **mainnet** and, for a real submit, **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, and a spendable coin whose value **exactly** matches the route amount. None of that was available in this session without operator-owned keys.

**Partial verification:** `bin/coinswapd-research -h` lists **`-mln-local-taker`**, **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, **`-a`** (fee MWEB), consistent with `PHASE_3_MWEB_HANDOFF_SLICE.md`.

## 4. `mln-cli` vs **live** LitVM registry/court (chain 4441)

Exports used for **`scout`** / **`pathfind`**:

- `MLN_LITVM_CHAIN_ID=4441`
- `MLN_LITVM_HTTP_URL=https://liteforge.rpc.caldera.xyz/http`
- `MLN_REGISTRY_ADDR` / `MLN_GRIEVANCE_COURT_ADDR` from `deploy/litvm-addresses.generated.env`
- `MLN_NOSTR_RELAYS=wss://relay.damus.io`

### `mln-cli scout`

**Result:** Exit **0**, table empty — **`(no verified makers)`**.

**Sample rejection (stderr):** `rejected <event-id>: chainId mismatch` — public relay ads did not match the configured **4441** filter / deployment pairing for this scout run.

### `mln-cli pathfind -json`

**Result:** Exit **1**.

```text
pathfind: need at least 3 verified makers with Tor endpoints, got 0
```

So there was **no** `route.json` to feed **`mln-cli forger`** against a live sidecar in this attempt.

## Blockers (honest)

1. **Discovery:** On the sampled public relay, **no** set of **three** LitVM-verified makers with **non-empty Tor** endpoints for the **4441** + deployed registry/court pairing used here.
2. **MWEB:** A real **`coinswapd-research`** submit requires **mainnet** MWEB wallet material and matching UTXO topology; not exercised without an operator wallet.

## What would complete the next attempt

- **MWEB:** Operator runs `E2E_MWEB_FUNDED=1` + `MWEB_RPC_BACKEND=coinswapd` per `scripts/e2e-mweb-handoff-stub.sh` with real **`COINSWAPD_FEE_MWEB`**, **`MWEB_SCAN_SECRET`**, **`MWEB_SPEND_SECRET`**, **`E2E_MWEB_DEST`**, **`E2E_MWEB_AMOUNT`** (exact coin), and optional **`E2E_MWEB_FUNDED_DEV_CLEAR=1`** only if the doc’s dev-only semantics apply.
- **LitVM + Nostr:** Either publish **kind 31250** ads that pass **`mln-cli scout`** for **4441** and the deployed addresses, or use the same relay/bootstrap the Phase 12 E2E stack uses when testing routes — still distinct from “public internet” maker availability.

---

*This file satisfies the repo gate for documenting a **real attempt** with **minimal logs** and **no keys**. It does not claim README **Phase 3** is complete.*
