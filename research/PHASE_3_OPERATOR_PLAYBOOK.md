# Phase 3 — One long operator playbook (by terminal)

Read this top to bottom. **Part A** builds confidence on one machine (mostly **Terminal 1**). **Part B** is the Tor multi-hop shape README Phase 3 is aiming at, using **four terminals** as four windows or tabs on one laptop (or four SSH sessions).

Deep reference (not step order): [PHASE_3_OPERATOR_CHECKLIST.md](PHASE_3_OPERATOR_CHECKLIST.md), [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md), [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md).

**Real Litecoin?**

- **Part A (stub + Docker):** **No.** The MWEB path is **`mw-rpc-stub`** or scripted checks; LitVM is **local Anvil**. You are not spending mainnet LTC/MWEB there.
- **Part B / `coinswapd-research` (funded path):** **Yes — mainnet MWEB today.** The tracked fork starts **Neutrino against Litecoin mainnet** and expects **`ltcmweb1…`** addresses ([`research/coinswapd/main.go`](../research/coinswapd/main.go), [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md)). That means **real keys and real funds** if you submit routes and run batches. Treat amounts as real money; use small outputs you can afford to lose while debugging.
- **LitVM (`zkLTC` / registry)** is separate: local Anvil in the lab, or **LitVM testnet** when RPC exists — not the same as “is this L1 Litecoin?”

**Convention**

- **Terminal 1–4** = separate shell windows unless noted.
- Do not paste private keys into shell history on shared machines; prefer env files with `chmod 600`.
- Ports below match the default Phase 12 / 3a compose layout ([PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md)): Anvil **8545**, MWEB JSON-RPC **8546**, sidecar **8080**, relay **7080**.

### Two computers — is that a better test?

**For learning the stack:** Not required. **Part A** is fastest on **one** machine (everything on `127.0.0.1`). Adding a second PC before that works usually adds firewall, “which IP is the relay?”, and SSH sync without improving the stub path.

**For realism (Tor + multi-hop):** **Yes, it can be a better exercise** once you are past Part A. You force real cross-host behavior: taker Tor exits dialing maker **`.onion`** URLs (not loopback), separate clocks and disks, and you mimic “I’m the user at home, makers are elsewhere.” It does **not** change the formal README Phase 3 bar (still: live `.onion` multi-hop + `pendingOnions` without dev-clear + public LitVM when RPC exists)—it just makes mistakes like wrong relay URL or unpublished HS more like production.

**Practical split with two PCs**

| Machine | Typical role |
| ------- | ------------ |
| **PC 1 (makers / lab)** | Nostr relay + registry (Anvil or testnet RPC reachable from PC 2), **all three makers** (`mlnd` + `coinswapd`), each with a real **hidden service** for JSON-RPC. |
| **PC 2 (taker)** | Tor (SOCKS), **`coinswapd-research`** (taker, **`-mln-local-taker`**), **`mln-sidecar`**, **`mln-cli`** pathfind/forger. **`MLN_*`** / Scout must point at the **same** relay and registry **PC 1** uses (use LAN IP or public relay, not `127.0.0.1` on PC 2). |

**Caveat:** If PC 1 still publishes **`tor`** as `http://127.0.0.1:…` in ads, PC 2 cannot use those hops. Ads must carry **reachable `.onion` (or routable) URLs** for the taker.

---

## Part A — Local confidence (stub + Docker, first session)

Goal: prove **Scout → pathfind → forger → sidecar → `mweb_*`** without Tor or mainnet keys.

### Step 0 — Open terminals (Part A)

| Terminal | Action |
| -------- | ------ |
| **1** | You will run builds and the handoff script here. |
| **2** | Optional: `docker compose logs -f` for the stack (attach after Step 2). |
| **3** | Optional: spare (e.g. `curl` experiments). |
| **4** | Optional: notes / copy `deploy/e2e.generated.env` lines after bootstrap. |

### Step 1 — Terminal 1: builds

```bash
cd /path/to/mwixnet-litvm
make build-mln-cli build-mw-rpc-stub
```

Docker must be running for the next step.

### Step 2 — Terminal 1: stub handoff (curl + compose)

```bash
./scripts/e2e-mweb-handoff-stub.sh
```

Expect **`GET /v1/balance OK`** and **`POST /v1/swap OK`**. This starts **`mw-rpc-stub`** on **:8546** and brings up **Anvil + relay + `mln-sidecar -mode=rpc`** via [deploy/docker-compose.e2e.yml](../deploy/docker-compose.e2e.yml) + [deploy/docker-compose.e2e.sidecar-rpc.yml](../deploy/docker-compose.e2e.sidecar-rpc.yml).

### Step 3 — Terminal 2 (optional): watch sidecar / relay

```bash
cd /path/to/mwixnet-litvm
docker compose -f deploy/docker-compose.e2e.yml -f deploy/docker-compose.e2e.sidecar-rpc.yml logs -f mln-sidecar
```

(Use another pane for `nostr-rs-relay` or `anvil` if you prefer.)

### Step 4 — Terminal 1: full CLI path on stub

Stop the previous run (**Ctrl+C** kills the stub; `docker compose ... down` if you want a clean slate). Then:

```bash
E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh
```

This runs [scripts/e2e-bootstrap.sh](../scripts/e2e-bootstrap.sh), starts the **makers** profile, exports **`MLN_*`** from **`deploy/e2e.generated.env`**, runs **`mln-cli pathfind`** and **`mln-cli forger`** with **`-trigger-batch -wait-batch`**. Success ends with **`Phase 3a stub handoff checks passed.`**

### Step 5 — Terminal 4 (optional): save env for later

```bash
grep '^MLN_' deploy/e2e.generated.env | head
```

You will reuse the same variable names when you move to funded **`coinswapd`** or Tor labs.

**Part A done.** You have validated the integration contract. Part B adds Tor and real maker RPCs.

---

## Part B — Tor + three makers + taker (four-terminal habit)

**Verbose step-by-step (Tor HS lines, port table, taker/maker order):** [PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md](PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md).

Goal: same logical path as Part A, but hop URLs are **`http://….onion:port`**, and **taker `coinswapd-research`** must see **`HTTP_PROXY`**.

Assume: **three maker hosts** (can be three VMs or three directories on one machine with three Tor hidden services—advanced). **One taker laptop** with Tor client. Adjust hostnames/paths to your layout.

### Step 0 — Open terminals (Part B)

| Terminal | Role for the rest of Part B |
| -------- | ---------------------------- |
| **1** | **Taker:** exports, `coinswapd-research`, optional `mln-sidecar`, `mln-cli`. |
| **2** | **Maker 1:** `mlnd` + `coinswapd` (or Docker) for first hop. |
| **3** | **Maker 2:** second hop. |
| **4** | **Maker 3:** third hop. |

If you only have one machine for all makers, you can run **2–4** as **tmux panes** or **sequential** “start maker 1, then 2, then 3” in one terminal—but separate processes are still required.

### Step 1 — Terminal 1: Tor and proxy template

```bash
cd /path/to/mwixnet-litvm
make phase3-operator-preflight
# or: TOR_SOCKS_PORT=9150 make phase3-operator-preflight   # Tor Browser
```

Copy the printed **`export HTTP_PROXY=…`**, **`HTTPS_PROXY`**, **`NO_PROXY`** into a small file (e.g. `~/mln-taker-proxy.sh`) and **`source`** it in **Terminal 1** before every **`coinswapd-research`** start.

Optional one-hop check (replace URL with a real maker):

```bash
export PHASE3_ONION_JSONRPC_URL='http://yourmaker.onion:8334'
make phase3-operator-preflight
```

### Step 2 — Terminal 2, 3, 4: each maker (repeat pattern)

On **each** maker host, in order:

1. Configure Tor hidden service → JSON-RPC port (your ops guide; not duplicated here).
2. Start **patched `coinswapd-research`** listening on the HS port (and **`-l`** as needed). Mesh makers: **omit** **`-mln-local-taker`** and align **`-k`** with topology. If unsure, use MLN taker path only on the **taker** side.
3. Start **`mlnd`** with LitVM + Nostr env filled from [`.env.compose.example`](../.env.compose.example) / [PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md): **`MLND_WS_URL`**, **`MLND_REGISTRY_ADDR`**, **`MLND_COURT_ADDR`**, **`MLND_OPERATOR_ADDR`**, **`MLND_LITVM_CHAIN_ID`**, **`MLND_TOR_ONION`** (full **`http://….onion:port`**), **`MLND_NOSTR_RELAYS`**, **`MLND_NOSTR_NSEC`**, optional **`MLND_SWAP_X25519_PUB_HEX`**.

Wait until all three ads are visible on your relay before the taker runs pathfind.

### Step 3 — Terminal 1: taker `coinswapd-research` + sidecar

Still with **`HTTP_PROXY`** sourced:

```bash
source ~/mln-taker-proxy.sh   # your file from Step 1
make build-research-coinswapd build-mln-sidecar
```

Start **taker** **`coinswapd-research`** with **`-mln-local-taker`**, **`-l 8546`** (or another free port), **`-a`** fee MWEB (**full `ltcmweb1…`**), **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, and Neutrino allowed to sync. See [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md) funded path.

In **the same Terminal 1** (second line, or use a **split pane**):

```bash
./bin/mln-sidecar -mode=rpc -rpc-url http://127.0.0.1:8546 -port 8080
```

(See [mln-sidecar/README.md](../mln-sidecar/README.md).)

### Step 4 — Terminal 1: pathfind

```bash
source deploy/e2e.generated.env   # or your own MLN_* exports pointing at the same registry/relay as makers
./bin/mln-cli pathfind -json > route.json
# inspect: jq '.hops[].tor' route.json   # must be non-empty .onion URLs
```

### Step 5 — Terminal 1: forger + batch wait

Advisory (warns if dev-clear env is on):

```bash
./scripts/phase3-funded-env-check.sh
```

Then (adjust flags to your `route.json`, dest, amount, sidecar URL):

```bash
./bin/mln-cli forger -route-json route.json -dry-run=false \
  -dest "$E2E_MWEB_DEST" -amount "$E2E_MWEB_AMOUNT" \
  -coinswapd-url http://127.0.0.1:8080/v1/swap \
  -trigger-batch -wait-batch
```

**README Phase 3 success bar:** **`pendingOnions`** returns to **0** **without** **`E2E_MWEB_FUNDED_DEV_CLEAR`** or **`-mweb-dev-clear-pending-after-batch`**. If you only see **0** with dev-clear, you are still on the Phase 3a smoke path, not the full gate.

### Step 6 — Terminals 2–4: if something hangs

- Watch **`coinswapd`** stderr for **`swap_forward:`** / **`swap_backward:`** (do not post full payloads publicly).
- Confirm each maker’s HS port matches the **`tor`** field the taker pathfound.
- Re-run **Step 1** proxy check: **`ALL_PROXY` alone is not enough** for **`coinswapd`**.

---

## Part C — After public LitVM RPC exists (same day or later)

Use **Terminal 1** for deploy tooling; makers/takers still use **2–4** as above but LitVM env comes from official docs and [PHASE_16_PUBLIC_TESTNET.md](../PHASE_16_PUBLIC_TESTNET.md) section 0 (**`make broadcast-litvm`**, **`make record-litvm-deploy`**, merge **`deploy/litvm-addresses.generated.env`**). Registry/court addresses in **`mlnd`** and **`mln-cli scout`** must match.

---

## Quick map (which doc if stuck)

| Question | Open |
| -------- | ---- |
| Why `socks5h` and not `ALL_PROXY`? | [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md) |
| Port matrix stub vs sidecar | [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md) |
| Maker Docker / NDJSON bridge | [PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md) |
| Checkbox criteria | [PHASE_3_OPERATOR_CHECKLIST.md](PHASE_3_OPERATOR_CHECKLIST.md) |
