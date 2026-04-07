# Phase 10: The Taker Client (`mln-cli`)

This document describes the **taker-side** Go CLI used to discover makers on Nostr, verify them against LitVM, pick a three-hop route (wallet PoC policy), and **hand off the route to a local `coinswapd` sidecar** over HTTP (pure Go in `mln-cli`; MWEB/Tor crypto stays in the sidecar process). It also documents **`mln-cli maker onboard`**, a **maker-operator** helper to plan or submit LitVM **`deposit` + `registerMaker`** (see [`research/WALLET_MAKER_FLOW_V1.md`](research/WALLET_MAKER_FLOW_V1.md)).

Normative maker-ad wire: [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md). Wallet route policy (PoC): [`research/USER_STORIES_MLN.md`](research/USER_STORIES_MLN.md). MWEB / RPC shape: [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md). Maker daemon: [`mlnd/README.md`](mlnd/README.md).

## Architectural phases

- **Phase 10.1 — Scout:** Nostr kind **31250** ingest, Schnorr check, LitVM `eth_call` to `MwixnetRegistry` (`makerNostrKeyHash`, `stake`, `minStake`, `stakeFrozen`, **`exitUnlockTime` must be zero** so makers in the exit queue are not advertised as routable).
- **Phase 10.2 — Pathfind:** Ordered **N1 → N2 → N3** selection from verified makers (minimize sum of optional per-hop fee hints, then prefer higher total stake, random tie-break).
- **Phase 10.3 — Forger:** Validate Tor endpoints on a saved route (`-dry-run`), or **POST JSON** to a local **MLN extension** URL (`-dry-run=false`). Vanilla ltcmweb only exposes `swap_Swap(onion.Onion)` on JSON-RPC `/`; the JSON route body is implemented by a **fork or proxy** (see [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md)).
- **Phase 10.4 — Maker onboard:** Read `MwixnetRegistry` (`stake`, `minStake`, `makerNostrKeyHash`, `exitUnlockTime`, `stakeFrozen`), derive **`nostrKeyHash`** from **`MLN_NOSTR_PUBKEY_HEX`** or **`MLN_NOSTR_NSEC`**, then by default **print a dry-run plan**; with **`-execute`**, sign **`deposit()`** (if stake &lt; minStake) and **`registerMaker(bytes32)`** using **`MLN_OPERATOR_ETH_KEY`**.

Shared maker-ad types live in [`mlnd/pkg/makerad`](mlnd/pkg/makerad) so `mlnd` and `mln-cli` stay aligned.

## Phase 10.1: Scout — operational flow

1. **Nostr ingest:** Subscribe to configured relays for `kind=31250` and `#t=mln-maker-ad` (until EOSE or timeout).
2. **Signature:** Verify each event with NIP-01 Schnorr (`go-nostr`).
3. **Parse:** Decode `content` JSON, validate `v=1`, and ensure `d`-tag chain id matches `litvm.chainId`.
4. **Deployment filter:** Keep events whose `litvm.chainId`, `litvm.registry`, and (if you set env) `litvm.grievanceCourt` match your expected deployment.
5. **Dedup:** For each maker address, keep the ad with the latest `created_at` (replaceable stream).
6. **LitVM:** For each remaining ad, `eth_call` the registry at `MLN_REGISTRY_ADDR`: `makerNostrKeyHash` must equal `keccak256(P)` for the event pubkey; `stake >= minStake`; `stakeFrozen` must be false; **`exitUnlockTime(maker)` must be zero** (makers who called `requestWithdrawal` are excluded so pathfind does not route new swaps through them).
7. **Output:** Table to stdout, or `mln-cli scout -json`. Rejections go to stderr with a short reason (unless `-quiet`).

### Quick start (Scout)

Copy **HTTP JSON-RPC URL** and **chain id** from [LitVM documentation](https://docs.litvm.com/) and [`research/LITVM.md`](research/LITVM.md); do not guess hostnames.

```bash
export MLN_NOSTR_RELAYS=wss://relay.damus.io
# Or a single relay: export MLN_NOSTR_RELAY_URL=wss://relay.example (if MLN_NOSTR_RELAYS is unset)
export MLN_LITVM_HTTP_URL=<HTTP_JSON_RPC_FROM_LITVM_DOCS>
export MLN_LITVM_CHAIN_ID=<DECIMAL_CHAIN_ID_STRING>
export MLN_REGISTRY_ADDR=0xYourRegistryAddress
# Optional: require ads to name the same grievance court you expect
# export MLN_GRIEVANCE_COURT_ADDR=0x...
# Optional: subscription wait (default 30s)
# export MLN_SCOUT_TIMEOUT=45s
```

Build from repo root: `make build-mln-cli` (output `bin/mln-cli`). Requires **Go 1.22+**.

```bash
./bin/mln-cli scout
./bin/mln-cli scout -json
```

## Phase 10.2: Pathfind

Uses the **same environment variables** as Scout, runs discovery, then prints an ordered route:

```bash
./bin/mln-cli pathfind
./bin/mln-cli pathfind -json > route.json
```

**One-shot route file (same JSON as `pathfind -json`):**

```bash
./bin/mln-cli route build              # writes route.json in cwd
./bin/mln-cli route build -out /tmp/route.json
./bin/mln-cli route build -self-included -out route.json
```

You need **at least three** verified makers.

**Self-included middle hop (Phase 14):** `pathfind -self-included` or `route build -self-included` fixes **N2** to the maker derived from **`MLN_OPERATOR_ETH_KEY`** (64-hex LitVM operator ECDSA key). See [`PHASE_14_SELF_INCLUSION.md`](PHASE_14_SELF_INCLUSION.md).

## Phase 10.3: Forger

**Dry-run (default):** checks that each hop has a **Tor** URL from the maker ad and prints the three hop endpoints.

```bash
./bin/mln-cli forger -route-json route.json -dry-run
# route.json from pathfind -json or mln-cli route build
```

**Submit to sidecar:** with `-dry-run=false`, `mln-cli` POSTs a JSON payload to the URL from `-coinswapd-url` (default `http://127.0.0.1:8080/v1/swap`). You must pass **`-dest`** (MWEB destination address) and **`-amount`** (satoshis). The request uses a **10s** HTTP timeout.

```bash
./bin/mln-cli forger -route-json route.json -dry-run=false \
  -dest mweb1... -amount 100000000
# optional: -coinswapd-url http://127.0.0.1:8080/v1/swap
```

## Phase 10.4: Maker onboard (`maker onboard`)

Operator flow for [`MwixnetRegistry`](contracts/src/MwixnetRegistry.sol): **default is dry-run** (no txs). Pass **`-execute`** to broadcast. Use the **same funded Anvil account** as in [`scripts/deploy-local-anvil.sh`](scripts/deploy-local-anvil.sh) for local testing (`PRIVATE_KEY` / first default account).

**Environment**

| Variable | Required | Role |
|----------|----------|------|
| `MLN_LITVM_HTTP_URL` | yes | LitVM HTTP JSON-RPC |
| `MLN_REGISTRY_ADDR` | yes | `MwixnetRegistry` address |
| `MLN_LITVM_CHAIN_ID` | yes | Decimal chain id (e.g. `31337` for Anvil) |
| `MLN_OPERATOR_ETH_KEY` | yes | 64-hex ECDSA key (optional `0x`); **maker address** and tx signer |
| `MLN_NOSTR_PUBKEY_HEX` | one of pubkey/nsec | 64-hex x-only Nostr pubkey **or** `npub1…` (nip19) |
| `MLN_NOSTR_NSEC` | one of pubkey/nsec | Hex secret or `nsec1…` (nip19) |

**Flags:** `-execute` — send transactions; `-force-reregister` — allow `registerMaker` when on-chain `makerNostrKeyHash` already differs (overwrites binding).

**Example (local Anvil after `make deploy-local`)**

```bash
export MLN_LITVM_HTTP_URL=http://127.0.0.1:8545
export MLN_REGISTRY_ADDR=0x…   # from deploy output
export MLN_LITVM_CHAIN_ID=31337
export MLN_OPERATOR_ETH_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
export MLN_NOSTR_PUBKEY_HEX=<64_hex_from_your_nostr_key>

./bin/mln-cli maker onboard              # plan only
./bin/mln-cli maker onboard -execute     # deposit if needed + registerMaker
```

### Sidecar request body (MLN extension)

Not supported by stock ltcmweb; the server must accept JSON like:

```json
{
  "route": [
    { "tor": "<mix API from maker ad>", "feeMinSat": 1000 },
    { "tor": "...", "feeMinSat": 1000 },
    { "tor": "...", "feeMinSat": 1000 }
  ],
  "destination": "mweb1...",
  "amount": 100000000
}
```

### Sidecar response

`mln-cli` expects a JSON object with `ok` (boolean), optional `detail`, and optional `error`. On success it reminds that **coinswapd batching** runs at **local midnight** in the reference implementation, so there may be **no immediate chain txid**.

### Optional: MWEB balance for wallets (`GET /v1/balance`)

The Wails taker wallet and [`forger.FetchMwebBalance`](mln-cli/internal/forger/balance.go) call **`GET`** on the same service as `POST /v1/swap`, with path **`/v1/balance`** when the swap URL ends in `/v1/swap` (otherwise `…/balance` is appended). **Stock ltcmweb does not implement this**; add it on your MLN fork or proxy.

Success JSON:

```json
{
  "ok": true,
  "availableSat": 125000000,
  "spendableSat": 120000000,
  "detail": "optional human-readable note"
}
```

- **`availableSat`** — total MWEB balance visible to the swap wallet (satoshis).
- **`spendableSat`** — optional; if omitted, clients treat it as equal to `availableSat`. Use when some funds are reserved or below dust for the next batch.

Implementation: [`mln-cli/internal/forger/`](mln-cli/internal/forger/).

## Trust model

Nostr relays are **untrusted transport**. LitVM registry state is the trust anchor for **identity binding** (`nostrKeyHash`) and **stake**. Stale or malicious ads may affect liveness; they cannot forge a passing registry check without the correct keys and stake.

## Build notes

Module path: `github.com/IndigoNakamoto/mwixnet-litvm/mln-cli` with `replace ../mlnd` for local development. CI runs `go test` in both `mlnd/` and `mln-cli/` when Go paths change.
