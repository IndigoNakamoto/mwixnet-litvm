# Phase 3a — MWEB handoff slice (no official LitVM testnet)

This is a **sub-milestone** toward README **Phase 3** (full end-to-end integration). It does **not** check the Phase 3 box: it proves the **taker → `mln-sidecar` HTTP → JSON-RPC `mweb_*`** bridge that [`mln-sidecar`](mln-sidecar/README.md) uses in **`-mode=rpc`**, while keeping **local Anvil** as the registry/court stand-in (same as [Phase 12](PHASE_12_E2E_CRUCIBLE.md)).

**Still out of scope for closing README Phase 3:** official [LitVM testnet](https://docs.litvm.com/) RPC as the live registry, **automated CI** running Neutrino-backed [`research/coinswapd/`](research/coinswapd/), a **completed** MWixnet round to chain with **LitVM slash/defense** on a public deployment, and **live** `.onion` connectivity in the default stub script.

## Live Tor operator lab (toward full Phase 3)

Runbook, **`HTTP_PROXY` / `socks5h://`** details (Go `net/http` vs `ALL_PROXY`), 1-hop then 3-hop checklist, and README Phase 3 **gate** wording: **[`research/PHASE_3_TOR_OPERATOR_LAB.md`](research/PHASE_3_TOR_OPERATOR_LAB.md)**.

**Preflight:** [`scripts/tor-preflight.sh`](scripts/tor-preflight.sh) or **`make tor-preflight`**.

## Completion (stub + full CLI path)

**Status: complete** for the **documented stub stack** as of **2026-04-03**.

**Verification:** `E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh` — `mw-rpc-stub` on **:8546**, Docker Compose (**Anvil + Nostr + `mln-sidecar -mode=rpc` + three `mlnd` makers**), [`scripts/e2e-bootstrap.sh`](scripts/e2e-bootstrap.sh) (contracts deploy + three makers registered), then **`mln-cli pathfind -json`** and **`mln-cli forger`** with **`-trigger-batch -wait-batch`** (POST **`/v1/route/batch`** → **`mweb_runBatch`**, poll **`GET /v1/route/status`** until **`pendingOnions`** is **0** on the stub). Stub logs **`mweb_submitRoute ok`**; forger prints route accepted and cleared pending queue. **Regression anchor:** script exits **`0`** after **`Phase 3a stub handoff checks passed.`** Quick **`curl`**-only path: `./scripts/e2e-mweb-handoff-stub.sh` (no `E2E_MWEB_FULL`).

## Real funded swap path achieved

**Status (in-repo, 2026-04-03):** The handoff script and fork support a **non-stub** submit path: real **`MWEB_SCAN_SECRET` / `MWEB_SPEND_SECRET`**, **`E2E_MWEB_DEST`** and **`E2E_MWEB_AMOUNT`** (must equal a spendable MWEB coin value exactly), per-hop **`swapX25519PubHex`** (via **`MLND_SWAP_X25519_PUB_HEX`** in [`scripts/e2e-bootstrap.sh`](scripts/e2e-bootstrap.sh) maker envs → **`mln-cli pathfind -json`**), then **`mln-cli forger -trigger-batch -wait-batch`**. Optional **`E2E_MWEB_FUNDED_DEV_CLEAR=1`** enables **`-mweb-dev-clear-pending-after-batch`** on **`coinswapd-research`** so **`pendingOnions`** returns to **0** after **`mweb_runBatch`** without chain finalize (**DEV ONLY** — see [research/COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md) section 2.7a). Production-shaped **`pendingOnions=0`** still requires **`finalize`** / **`SendTransaction`** after live P2P hops.

**Verification command** (host runs **`coinswapd-research`**; Docker stack from the script; replace placeholders with your wallet values):

```bash
MWEB_RPC_BACKEND=coinswapd \
  COINSWAPD_FEE_MWEB='ltcmweb1<your full fee MWEB address>' \
  MWEB_SCAN_SECRET='<64-hex>' MWEB_SPEND_SECRET='<64-hex>' \
  E2E_MWEB_DEST='ltcmweb1<your full receive>' \
  E2E_MWEB_AMOUNT=<satoshis matching one spendable coin exactly> \
  E2E_MWEB_FUNDED=1 E2E_MWEB_FUNDED_DEV_CLEAR=1 \
  ./scripts/e2e-mweb-handoff-stub.sh
```

(Omit **`E2E_MWEB_FUNDED_DEV_CLEAR=1`** only if you depend on a real multi-hop finalize; **`forger -wait-batch`** may time out without live maker RPCs.)

## Completed swap path (operators): submit → batch → status

**Status — completed swap path achieved (in-repo):** Operators and CI can run **submit → `mweb_runBatch` → poll `mweb_getRouteStatus`** end-to-end via **`mln-sidecar`** + **`mln-cli forger -trigger-batch -wait-batch`**; **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** passes with **`pendingOnions`** returning to **0** on **`mw-rpc-stub`**. README **Phase 3** stays unchecked until **live multi-hop maker RPCs**, **`.onion`**, and **public LitVM** grievance path (see gaps in this doc and **`PRODUCT_SPEC.md`**).

**Normative wire:** [research/COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md) §2.6–2.7 (`mweb_getRouteStatus`, `mweb_runBatch`).

1. **Preconditions (real `coinswapd-research`):** Neutrino synced, **`-mweb-scan-secret`** + **`-mweb-spend-secret`**, **`-a`** fee MWEB address, hop **`swapX25519PubHex`** (or **`-mweb-pubkey-map`** JSON), funded MWEB coin matching **`-amount`**, **`-mln-local-taker`** for MLN-only taker topology. Set **Tor/SOCKS** env on **coinswapd** if hop URLs are **`.onion`**.

2. **`mweb_getBalance`** / **`GET /v1/balance`** — confirm spendable funds.

3. **`mweb_submitRoute`** / **`POST /v1/swap`** — same MLN route JSON as before; success returns **`{ "accepted": true }`** on the fork.

4. **Trigger batch (do not rely only on UTC midnight):** **`mweb_runBatch`** or **`POST /v1/route/batch`** through **`mln-sidecar`**, or start **`coinswapd`** with **`-f`** once (legacy “swap at startup”). This runs **`performSwap()`** synchronously up to async **`swap_forward`** calls.

5. **Poll completion:** **`mweb_getRouteStatus`** or **`GET /v1/route/status`**. **`pendingOnions`** should reach **0** after **`finalize`** + **`SendTransaction`** clears the local DB (fork behavior). If peers are unreachable, onions may remain; fix **RPC/Tor** to next hops.

6. **`mln-cli` one-liner (sidecar URL):**  
   `mln-cli forger -route-json route.json -dry-run=false -dest <ltcmweb1…> -amount <sat> -coinswapd-url http://127.0.0.1:8080/v1/swap -trigger-batch -wait-batch`

**`mw-rpc-stub`:** simulates **`pendingOnions`** increment on submit and clears it on **`mweb_runBatch`** so **`-wait-batch`** can pass in CI without Neutrino. When **`mweb_submitRoute`** carries LitVM metadata (**`epochId`**, **`accuser`**, **`swapId`**) and each hop includes **`operator`** (as **`mln-cli forger`** sends from **`route.json`**), the stub’s golden receipt sets **`accusedMaker`** to the **lowercased** **`route[0].operator`** (same correlator intent as **`MockBridge`** in **`-mode=mock`**). If hop 0 **`operator`** is missing or not a 20-byte hex address, the receipt keeps the legacy placeholder **`0x000…0001`** so minimal RPC tests stay stable.

### Correlator-aligned receipts and maker auto-defend (local)

To close the loop **taker vault → `openGrievance` → `mlnd` receipt match → `defendGrievance`** without hand-editing receipts:

1. **Shared SQLite:** use the **same file path** for **`mln-cli forger -vault …`** and the accused maker’s **`MLND_DB_PATH`** (both use the receipt store schema; the accused hop is **N1**, first hop in **`route.json`**).
2. **Before `grievance file`:** start **`mlnd`** for **N1** with **`MLND_OPERATOR_ADDR`** equal to **`jq -r '.hops[0].operator'`** from that route, **`MLND_DEFEND_AUTO=1`**, **`MLND_OPERATOR_PRIVATE_KEY`** for that address, plus usual **`MLND_WS_URL`**, **`MLND_COURT_ADDR`**, **`MLND_REGISTRY_ADDR`**, **`MLND_LITVM_CHAIN_ID`**. The grievance watcher only sees **new** **`GrievanceOpened`** logs after subscribe — if **`mln-cli grievance file`** runs first, the maker will **not** auto-defend that case.
3. **Accuser key:** set **`MLN_ACCUSER_ETH_KEY`** (or **`MLN_OPERATOR_ETH_KEY`**) and **`MLN_RECEIPT_EPOCH_ID`** consistently for **`forger -vault`** and **`grievance file`** (see **`mln-cli` forger** help). Local Anvil-only dev keys are documented under **`PHASE_12_E2E_CRUCIBLE.md`** / **`scripts/e2e-bootstrap.sh`** — never reuse on a public network.
4. **Helper script:** **[`scripts/grievance-correlated-stub-e2e.sh`](scripts/grievance-correlated-stub-e2e.sh)** maps **`deploy/e2e.generated.env`** to **`MLN_*`**, runs **`route build`**, optionally **`forger`** with a vault against the Phase 3a sidecar URL, prints **`mlnd`** env for N1 and the **`grievance file`** command line (flags **before** positional **`swap_id`**).

**`mln-judge`:** **`defendGrievance`** calldata must be the **v1 ABI-encoded tuple** from **`litvmevidence.BuildDefenseData`**. Minimal synthetic bytes (e.g. **`0xdeadbeef`**) will fail **`UnpackDefenseV1`** with length errors — that is expected. After **`mlnd`** auto-defend (or any path that submits real defense data), run **`mln-judge`** with **`JUDGE_DRY_RUN=1`** to confirm the daemon logs decoded defense fields without broadcasting **`adjudicateGrievance`** (see **`mln-judge/README.md`**).

**Remaining gaps vs on-chain “mixed” proof:** multi-hop **`swap_forward` / `swap_backward`** still require **live maker `coinswapd` RPC** endpoints; LitVM grievance path and L1 inclusion proofs remain per **`PRODUCT_SPEC.md`** / Phase 15 docs.

## Permanent regression anchor (PRs + release tags)

**Run before merging any PR** that touches **`mln-sidecar/`**, **`research/coinswapd/`**, or **`mln-cli`** paths that affect the forger / sidecar handoff (**`internal/forger/`**, **`internal/pathfind/`**, **`internal/takerflow/`**, **`cmd/mln-cli/`** forger flags), **and** before every **`v*`** release tag on the stack. From repo root:

1. **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** — expect exit **`0`** and **`Phase 3a stub handoff checks passed.`** (build **`make build-mw-rpc-stub`**, **`make build-mln-cli`**; Docker for Compose).

2. **Research fork variant (host JSON-RPC; no `E2E_MWEB_FULL`):** on a suitable host, **`make build-research-coinswapd`**, full **`ltcmweb1…`** **`COINSWAPD_FEE_MWEB`**, then  
   `MWEB_RPC_BACKEND=coinswapd COINSWAPD_FEE_MWEB="$ADDR" ./scripts/e2e-mweb-handoff-stub.sh`  
   — expect balance path OK and stub-shaped **`POST /v1/swap`** → **502** (see [Quick path](#quick-path-stub--compose) step 3).

Skipping these after handoff-affecting edits risks silent regressions in **`mweb_*`**, **`/v1/route/*`**, or **`mln-cli forger`**.

## Release candidate regression (before every `v*` tag)

Same commands as **Permanent regression anchor** above; release candidates must run both stub **`E2E_MWEB_FULL=1`** and, when feasible, the **`MWEB_RPC_BACKEND=coinswapd`** smoke on a dev host.

## Integration slice (research fork + Tor-shaped URLs) — 2026-04-03

**Shipped in this repo (not a full Phase 3 close):**

- **`make build-research-coinswapd`** → **`bin/coinswapd-research`** (Go **1.23** toolchain; Neutrino mainnet on start — see [`research/coinswapd/main.go`](research/coinswapd/main.go)).
- **Optional E2E backend:** `MWEB_RPC_BACKEND=coinswapd` in [`scripts/e2e-mweb-handoff-stub.sh`](scripts/e2e-mweb-handoff-stub.sh) starts the fork on **`STUB_ADDR`** (default **`:8546`**) when **`COINSWAPD_FEE_MWEB`** is set (mainnet MWEB fee address for **`-a`**), passing **`-mln-local-taker`** so startup skips **`getNodes()`** / **`config.AliveNodes`**. Default script path still uses a **stub-shaped** **`POST /v1/swap`** body and expects **502** (missing keys / UTXO). For **`E2E_MWEB_FUNDED=1`**, see [Real funded operator path](#real-funded-operator-path-coinswapd-research) below ( **`E2E_MWEB_FULL=1`** alone with **`coinswapd`** remains invalid unless **`E2E_MWEB_FUNDED=1`** ).
- **Tor / mix URL normalization:** [`mln-sidecar/internal/mweb/translator.go`](mln-sidecar/internal/mweb/translator.go) and [`mln-cli/internal/forger/torurl.go`](mln-cli/internal/forger/torurl.go) prefix **`http://`** when a hop **`tor`** string has no URI scheme (typical for ads that publish `something.onion:port` only). **`mln-cli pathfind`** only considers makers with **non-empty** **`tor`** in the verified set so routes are viable for real transport.
- **Where real Tor applies:** [`research/coinswapd/swap.go`](research/coinswapd/swap.go) uses **`rpc.Dial(node.Url)`** (go-ethereum over **`http://`**) for inter-node **`swap_forward` / `swap_backward`**. Go’s **`net/http`** default transport uses **`ProxyFromEnvironment`**, which reads **`HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`** (not **`ALL_PROXY`**). Use **`socks5h://127.0.0.1:9050`** so **`.onion`** hostnames resolve at the Tor proxy. Set those vars on the **`coinswapd`** process; see [`research/PHASE_3_TOR_OPERATOR_LAB.md`](research/PHASE_3_TOR_OPERATOR_LAB.md). The **sidecar** only forwards route JSON; it does not dial maker `.onion` hosts (set proxy on **mln-sidecar** only if **`-rpc-url`** itself points through Tor).

### Research backend smoke (operator-verified)

**Status:** **passed** (**2026-04-03**) with **`MWEB_RPC_BACKEND=coinswapd`**, valid mainnet **`COINSWAPD_FEE_MWEB`** (**`ltcmweb1…`** per ltcmweb/ltcd), script-supplied **`-mln-local-taker`**, and **`make build-research-coinswapd`**.

**Observed:** **`mln-local-taker: skipping getNodes`** at fork startup; Neutrino header sync on the host; **`GET /v1/balance`** via **`mln-sidecar -mode=rpc`** → **200** / **`ok: true`** (**`mweb_getBalance`** live); **`POST /v1/swap`** with the **stub-shaped** body → sidecar **502** (**expected**: missing **`swapX25519PubHex`** / no UTXO); exit **0** and **`Phase 3a stub handoff checks passed.`**

This **does not** close README **Phase 3** (full Nostr → Tor → MWixnet round → L2 path). **Promote path** details: [Promote path](#promote-path-researchcoinswapd) below.

### Real funded operator path (`coinswapd-research`)

**Goal:** **`mweb_getBalance`** → **`mweb_submitRoute`** (full MLN JSON with per-hop **`swapX25519PubHex`**) → **`mweb_runBatch`** → poll **`mweb_getRouteStatus`** until **`pendingOnions`** is **0**, without relying on a stub-shaped **502** body.

**Preconditions**

1. **Neutrino synced** far enough that **`mweb_getBalance`** reflects your coins; **`-mweb-scan-secret`** and **`-mweb-spend-secret`** (hex **32-byte** keys) match the wallet that owns the MWEB UTXO.
2. **`-a`** / **`COINSWAPD_FEE_MWEB`:** valid mainnet MWEB stealth fee address (**`ltcmweb1…`**).
3. **Exact UTXO value:** [`pickCoinExactAmount`](research/coinswapd/mln_wallet.go) requires a spendable coin whose **`Value`** **equals** the route **`amount`** (satoshis). **`sum(feeMinSat) ≤ amount`** (pathfind / sidecar rules).
4. **Three hop pubkeys:** every hop must carry **`swapX25519PubHex`** (64 lowercase hex digits). Local E2E: [`scripts/e2e-bootstrap.sh`](scripts/e2e-bootstrap.sh) now writes **`MLND_SWAP_X25519_PUB_HEX`** into each **`e2e.maker*.env`** so **`mln-cli pathfind -json`** includes keys for **`mln-cli forger`**.

**Happy-path RPC sequence (normative)**

| Step | JSON-RPC | Notes |
| ---- | -------- | ----- |
| 1 | **`mweb_getBalance`** | Confirm **`spendableSat`**; optional via sidecar **`GET /v1/balance`**. |
| 2 | **`mweb_submitRoute`** | Single object: **`route`**, **`destination`**, **`amount`** (same shape as **`POST /v1/swap`** / [`COINSWAPD_MLN_FORK_SPEC.md`](research/COINSWAPD_MLN_FORK_SPEC.md) §2.3). |
| 3 | **`mweb_runBatch`** | Same synchronous entry as midnight batch; **`swap_forward`** may still run async. |
| 4 | **`mweb_getRouteStatus`** | Poll until **`pendingOnions == 0`** (or timeout). |

**Production-shaped completion:** **`pendingOnions`** returns to **0** only after **`finalize`** + **`SendTransaction`** removes spent onions ([`swap.go`](research/coinswapd/swap.go)), which requires a full **backward** pass and live maker RPCs for **`swap_forward` / `swap_backward`**.

**Controlled-host smoke without live makers (DEV ONLY):** start the fork with **`-mweb-dev-clear-pending-after-batch`** (see [COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md) §2.7a). After **`mweb_runBatch`**, the fork **deletes persisted onions** without broadcasting — use only to verify HTTP/RPC + DB status wiring.

**One-shot script (host runs `coinswapd-research`; Docker runs Anvil + relay + rpc sidecar + makers)**

```bash
MWEB_RPC_BACKEND=coinswapd \
  COINSWAPD_FEE_MWEB='ltcmweb1…' \
  MWEB_SCAN_SECRET='<64-hex scan>' MWEB_SPEND_SECRET='<64-hex spend>' \
  E2E_MWEB_DEST='ltcmweb1…' E2E_MWEB_AMOUNT=1000000 \
  E2E_MWEB_FUNDED=1 E2E_MWEB_FUNDED_DEV_CLEAR=1 \
  ./scripts/e2e-mweb-handoff-stub.sh
```

**Flags / env**

| Variable | Meaning |
| -------- | ------- |
| **`E2E_MWEB_FUNDED`** | Run bootstrap (unless **`E2E_MWEB_SKIP_BOOTSTRAP=1`**), makers, **`pathfind -json`**, **`forger -trigger-batch -wait-batch`** against **`coinswapd`** on the host. |
| **`E2E_MWEB_FUNDED_DEV_CLEAR`** | Pass **`-mweb-dev-clear-pending-after-batch`** to **`coinswapd-research`** (DEV ONLY). |
| **`E2E_MWEB_BATCH_POLL`**, **`E2E_MWEB_BATCH_TIMEOUT`** | Override forger wait polling (defaults **500ms**, **5m**). |
| **`jq`** | Required for funded mode (validates pathfind output has three **64-char** swap keys). |

**`mln-cli` equivalent (after you have a route JSON file):**

```bash
mln-cli forger -route-json deploy/e2e.mweb-funded.route.json -dry-run=false \
  -dest "$E2E_MWEB_DEST" -amount "$E2E_MWEB_AMOUNT" \
  -coinswapd-url http://127.0.0.1:8080/v1/swap \
  -trigger-batch -wait-batch -batch-poll 500ms -batch-timeout 5m
```

## Documented gaps vs full round + L2

| Gap | Notes |
| --- | ----- |
| **Research smoke scope** | Proves **HTTP → JSON-RPC** to **`bin/coinswapd-research`** only; does **not** assert **`mweb_submitRoute`** persist, **`swap_forward`**, or broadcast. |
| **No live Tor in CI** | Stub and cleartext **`http://127.0.0.1`** hops only; `.onion` + SOCKS are operator bring-up. |
| **coinswapd Neutrino + UTXO** | Successful **`mweb_submitRoute`** needs keys, optional **`-mweb-pubkey-map`**, funded MWEB coin; smoke uses a body the fork correctly rejects (**502**). |
| **`swap_forward` / epoch** | Multi-node forward/backward and midnight batching are **coinswapd** internals, not covered by this handoff script. |
| **LitVM L2** | Grievance / slash path is **not** wired here; see [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) and Phase 12/15 docs. |
| **Mesh vs MLN taker** | **`-mln-local-taker`** skips **`-k`** / **`getNodes`** mesh match; public mesh nodes omit it and use **`-k`** consistent with [config/nodes.go](research/coinswapd/config/nodes.go). |

## Status (regression / hardening)

- **`mln-sidecar -mode=rpc`:** [`internal/mweb/rpc_bridge.go`](mln-sidecar/internal/mweb/rpc_bridge.go) normalizes `-rpc-url` (trim space, trailing slash), normalizes hop **`tor`** strings before validation, and calls **`mweb_submitRoute`** / **`mweb_getBalance`** via go-ethereum `rpc.Client` (same method names as [`research/coinswapd/mweb_service.go`](research/coinswapd/mweb_service.go)).
- **Tests:** [`mln-sidecar/internal/mweb/rpc_bridge_test.go`](mln-sidecar/internal/mweb/rpc_bridge_test.go) asserts JSON-RPC **params** for `mweb_submitRoute` decode to the fork’s route object (including optional per-hop **`swapX25519PubHex`**). [`mln-sidecar/internal/api/server_rpc_test.go`](mln-sidecar/internal/api/server_rpc_test.go) covers HTTP 502 on upstream RPC errors for swap and balance. [`research/coinswapd/mlnroute/sidecar_wire_test.go`](research/coinswapd/mlnroute/sidecar_wire_test.go) golden-unmarshals the same JSON as [`scripts/e2e-mweb-handoff-stub.sh`](scripts/e2e-mweb-handoff-stub.sh).
- **`mw-rpc-stub`:** validates **`mweb_submitRoute`** payload shape (3 hops, destination, amount) so Compose/curl exercises fail loudly if the sidecar wire drifts.

## Goal

Validate the **integration contract** from [research/COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md): `mln-cli forger` / Wails **Send Privately** → **`GET /v1/balance`** and **`POST /v1/swap`** on the sidecar → **`mweb_getBalance`** / **`mweb_submitRoute`** on a JSON-RPC peer.

## Port layout (local E2E + rpc sidecar)

| Service | Port | Notes |
| ------- | ---- | ----- |
| Anvil (registry HTTP/WS) | **8545** | Taker `MLN_LITVM_HTTP_URL`; matches Phase 12 |
| MWEB JSON-RPC (`mweb_*`) | **8546** | Stub (`mw-rpc-stub`) or **`coinswapd` fork** — must **not** collide with sidecar **8080** |
| `mln-sidecar` HTTP | **8080** | Forger default `-coinswapd-url` … `/v1/swap` |
| Nostr relay (host) | **7080** | Same as Phase 12 Compose map |

Upstream **`coinswapd`** defaults to **`-l 8080`** ([research/COINSWAPD_TEARDOWN.md](research/COINSWAPD_TEARDOWN.md)). When running next to this stack, start the fork with **`-l 8546`** (or another free port) and point the sidecar at it.

## Quick path: stub + Compose

1. From repo root: **`./scripts/e2e-mweb-handoff-stub.sh`**  
   - Starts **`mw-rpc-stub`** on **`:8546`**, brings up Anvil + relay + sidecar with [deploy/docker-compose.e2e.sidecar-rpc.yml](deploy/docker-compose.e2e.sidecar-rpc.yml), and checks the HTTP→RPC path with **`curl`**.

2. Full Scout → Pathfind → Forger (optional):  
   **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`**  
   - Runs [scripts/e2e-bootstrap.sh](scripts/e2e-bootstrap.sh), starts the **makers** profile, exports **`MLN_*`** from generated `deploy/e2e.generated.env`, then **`mln-cli pathfind -json`** and **`mln-cli forger`** against `http://127.0.0.1:8080/v1/swap`.

3. Optional **research fork** smoke (host JSON-RPC only; no full **`E2E_MWEB_FULL`**):  
   **`ADDR='<paste full mainnet MWEB stealth from wallet>' MWEB_RPC_BACKEND=coinswapd COINSWAPD_FEE_MWEB="$ADDR" ./scripts/e2e-mweb-handoff-stub.sh`** — the string must be complete Bech32 (usually **60+** characters). On **mainnet**, [`github.com/ltcmweb/ltcd`](https://github.com/ltcmweb/ltcd) uses **`Bech32HRPMweb: "ltcmweb"`**, so valid addresses begin with **`ltcmweb1`** (what compatible wallets show), not **`mweb1`**. Do **not** use Unicode `…`, ASCII `...`, or truncated examples; **`coinswapd`** then prints `decoded address is of unknown format` and never listens. The script rejects obvious shorthand before starting the binary.  
   - Builds **`bin/coinswapd-research`**, waits for **`mweb_getBalance`**, then same Compose + sidecar checks as above.

Build targets: **`make build-mw-rpc-stub`** → **`bin/mw-rpc-stub`**; **`make build-research-coinswapd`** → **`bin/coinswapd-research`**.

## Acceptance criteria

- **`POST /v1/swap`** with a valid **three-hop** MLN route returns **200** and `"ok":true` from the sidecar while it is in **`-mode=rpc`** against the **stub** (default script).
- **`GET /v1/balance`** returns **200** and balances forwarded from **`mweb_getBalance`**.
- Stub logs (or fork logs) show **`mweb_submitRoute`** was invoked when the stub accepts the body (integration parity with [mln-sidecar/internal/api/server_rpc_test.go](mln-sidecar/internal/api/server_rpc_test.go)).

## Promote path: `research/coinswapd`

Run a built binary from [`research/coinswapd/`](research/coinswapd/) listening on **8546**, with flags such as **`-l 8546`**, **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, **`-a`** (MWEB fee address), **`-k`** (server ECDH key), and optionally **`-mweb-pubkey-map`** per [COINSWAPD_MLN_FORK_SPEC.md](research/COINSWAPD_MLN_FORK_SPEC.md). For **MLN `mweb_*` / local sidecar** without joining the public swap mesh, add **`-mln-local-taker`** (random **`-k`** is fine; peers come from **`mweb_submitRoute`**). Omit it when operating as a listed mesh node (your **`-k`** must match your row in the probed topology).

**Expectation:** that process starts **Neutrino** against **mainnet** in the current fork ([research/coinswapd/main.go](research/coinswapd/main.go)) — sync time, disk, and keys are **operator** concerns, not part of the default automated stub flow.

## Related

- Live Tor operator lab + README Phase 3 gate: [research/PHASE_3_TOR_OPERATOR_LAB.md](research/PHASE_3_TOR_OPERATOR_LAB.md)  
- Phase 12 closed loop (mock sidecar): [PHASE_12_E2E_CRUCIBLE.md](PHASE_12_E2E_CRUCIBLE.md)  
- Threat model note on mock vs rpc: [research/THREAT_MODEL_MLN.md](research/THREAT_MODEL_MLN.md)  
- Sidecar modes: [mln-sidecar/README.md](mln-sidecar/README.md)
