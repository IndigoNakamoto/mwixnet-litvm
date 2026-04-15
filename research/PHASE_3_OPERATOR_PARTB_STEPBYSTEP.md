# Phase 3 — Part B step-by-step (Tor, three makers, one taker)

This is an **expanded, literal** walkthrough for [Part B of `PHASE_3_OPERATOR_PLAYBOOK.md`](PHASE_3_OPERATOR_PLAYBOOK.md). Read the **warnings** first; then follow **in order**.

**Shorter overview:** [PHASE_3_OPERATOR_PLAYBOOK.md](PHASE_3_OPERATOR_PLAYBOOK.md) Part B · **Tor rationale:** [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md) · **Checklist:** [PHASE_3_OPERATOR_CHECKLIST.md](PHASE_3_OPERATOR_CHECKLIST.md) · **Funded `coinswapd` flags / amounts:** [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md) · **Maker Docker patterns:** [PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md)

---

## 0) Read this before you type anything

- **Real Litecoin MWEB:** On the **taker** (and any maker that signs or holds funds), `coinswapd-research` uses **mainnet Neutrino**. Route amounts and keys are **real money** if you submit routes and run batches. Use **small** test outputs you can afford to lose.
- **Part A vs Part B:** Part A uses **Docker** (Anvil + stub or local RPC on `127.0.0.1`). Part B uses **live `.onion` URLs** in Nostr ads. If Part A Compose is still up, it is **easy to confuse** “cleartext lab” with “Tor lab.” For your first Part B attempt, **stop Part A** when you are ready:

  ```bash
  cd /path/to/mwixnet-litvm
  docker compose -f deploy/docker-compose.e2e.yml -f deploy/docker-compose.e2e.sidecar-rpc.yml down
  ```

  (`docker compose logs -f` stops with Ctrl+C but **does not** stop containers; `down` does.)

- **Four roles, four processes:** You need **three maker stacks** (each: Tor HS + `coinswapd-research` + `mlnd`) and **one taker stack** (Tor client + proxy env + taker `coinswapd-research` + `mln-sidecar` + `mln-cli`). They can be **four VMs**, **four terminals on one PC**, or a **split** (e.g. makers on a server, taker on a laptop). The **relay and LitVM RPC** must be the **same** for all makers and the taker’s `mln-cli scout` / `pathfind`.

---

## 1) Choose ports and names (write them down)

For **each maker** `i` in `{1,2,3}`, decide:

| Item | Example | Notes |
|------|---------|--------|
| JSON-RPC listen port | `8334`, `8335`, `8336` | Must be **unique** on that host. |
| Onion virtual port | Often **same** as RPC port | What appears in `http://xxx.onion:PORT`. |
| `coinswapd` **`-l`** | Same as RPC port | Listen port for HTTP JSON-RPC. |
| Fee address **`-a`** | `ltcmweb1…` | Valid MWEB fee address per fork. |
| Mesh key **`-k`** | ECDH private key | **Mesh makers** use **`-k`**; topology must match alive nodes (see Tor lab). **Taker** uses **`-mln-local-taker`** and does **not** rely on the same mesh probe. |

**Taker** (one machine):

| Item | Example |
|------|---------|
| Taker `coinswapd` **`-l`** | `8546` (any free port) |
| `mln-sidecar` **`-rpc-url`** | `http://127.0.0.1:8546` |
| `mln-sidecar` **`-port`** | `8080` |

---

## 2) Terminal layout (same as the short playbook)

| Terminal | Role |
|----------|------|
| **1** | **Taker:** Tor SOCKS OK, proxy file, taker `coinswapd`, `mln-sidecar`, `mln-cli`. |
| **2** | **Maker 1** |
| **3** | **Maker 2** |
| **4** | **Maker 3** |

---

## 3) Maker side — repeat for Maker 1, then 2, then 3

Do **all substeps** on **that maker’s host** before moving to the next maker.

### 3a) Install and run Tor

- **macOS (Homebrew):** `brew install tor` → edit `torrc` (often `/opt/homebrew/etc/tor/torrc` or `/usr/local/etc/tor/torrc`) → `brew services start tor` (or run `tor` in foreground for debugging).
- **Linux:** package `tor`, enable `tor.service`, edit `/etc/tor/torrc` (paths vary by distro).

### 3b) Hidden service directory

Pick a **directory on disk** Tor can read/write (example: `/var/lib/tor/mln-maker1` on Linux; on macOS Homebrew often under `$(brew --prefix)/var/tor/mln-maker1`). **Create it** and fix ownership to the user Tor runs as (on Linux commonly `debian-tor` / `tor`).

### 3c) `torrc` lines (one block per maker)

Use a **fresh** `HiddenServiceDir` per maker. Example for Maker 1 exposing RPC on local port **8334**:

```text
HiddenServiceDir /path/to/tor/mln-maker1
HiddenServicePort 8334 127.0.0.1:8334
```

Meaning: world reaches **`http://<onion>:8334`**; Tor forwards to **`127.0.0.1:8334`** where `coinswapd` listens.

**Maker 2 / 3:** duplicate with **`mln-maker2`**, **`mln-maker3`** and ports **8335**, **8336** (or your plan).

Reload Tor (`brew services restart tor` / `systemctl restart tor`). Read the onion hostname:

```bash
sudo cat /path/to/tor/mln-maker1/hostname
```

You get one line: `something56chars.onion`. Your **public** maker URL is:

```text
http://something56chars.onion:8334
```

(use your real port). **No** `https://` unless you have terminated TLS yourself (default lab is `http://`).

### 3d) Start `coinswapd-research` (maker / mesh)

From repo root on that host:

```bash
cd /path/to/mwixnet-litvm
make build-research-coinswapd
```

Start **listening on loopback** so only Tor (and local tools) reach JSON-RPC:

```bash
./bin/coinswapd-research -l 8334 -a 'ltcmweb1YOUR_FEE_ADDRESS' -k 'YOUR_MESH_ECDH_KEY' \
  # add other flags your topology needs; mesh makers omit -mln-local-taker
```

Exact **`-k`** / mesh flags depend on your **alive-node** graph; if you are unsure, read [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md) **Topology** and [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md). **Leave this process running.**

**Sanity from another machine** (e.g. taker, with Tor running):

```bash
curl --socks5-hostname 127.0.0.1:9050 -fsS -m 60 \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"mweb_getBalance","params":[],"id":1}' \
  'http://YOUR.onion:8334'
```

(or set `PHASE3_ONION_JSONRPC_URL` and run `make phase3-operator-preflight` on the taker).

### 3e) Start `mlnd` with matching Nostr + LitVM + Tor URL

`mlnd` must publish **the same** Tor URL peers will dial. Set at least:

| Variable | Purpose |
|----------|---------|
| `MLND_WS_URL` | LitVM WebSocket JSON-RPC (`wss://…`) |
| `MLND_REGISTRY_ADDR` | Registry contract |
| `MLND_COURT_ADDR` | Grievance court |
| `MLND_OPERATOR_ADDR` | This maker’s **registered** EVM operator |
| `MLND_LITVM_CHAIN_ID` | Decimal chain id |
| `MLND_NOSTR_RELAYS` | Same relay(s) as everyone else |
| `MLND_NOSTR_NSEC` | This maker’s Nostr secret |
| `MLND_TOR_ONION` | Full **`http://….onion:port`** for **this** maker |
| `MLND_SWAP_X25519_PUB_HEX` | **64 hex** swap pubkey for pathfind/forger (see E2E bootstrap pattern) |

Template keys (no real secrets): [`.env.compose.example`](../.env.compose.example). **Each maker** needs **different** `MLND_OPERATOR_ADDR`, `MLND_NOSTR_NSEC`, `MLND_TOR_ONION`, and usually **different** `MLND_SWAP_X25519_PUB_HEX`.

**Bridge to `coinswapd` (optional but common in Docker):** `MLND_BRIDGE_COINSWAPD=1` and paths per [PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md).

Repeat **3a–3e** for makers 2 and 3. **Wait** until you can see **three** distinct maker ads on the relay before the taker runs `pathfind`.

---

## 4) Taker — Terminal 1 (Tor + proxy file)

### 4a) Prove Tor

```bash
cd /path/to/mwixnet-litvm
make phase3-operator-preflight
# Tor Browser users:
# TOR_SOCKS_PORT=9150 make phase3-operator-preflight
```

### 4b) Save proxy exports (create the file once)

```bash
cat > ~/mln-taker-proxy.sh <<'EOF'
export HTTP_PROXY="socks5h://127.0.0.1:9050"
export HTTPS_PROXY="socks5h://127.0.0.1:9050"
export NO_PROXY="127.0.0.1,localhost"
EOF
chmod 600 ~/mln-taker-proxy.sh
```

If you use **9150**, edit the file accordingly.

**Every time** you open a new Terminal 1 for the taker:

```bash
source ~/mln-taker-proxy.sh
```

**Do not** set `PHASE3_ONION_JSONRPC_URL` to the playbook placeholder `yourmaker.onion` — that is **not** real and will fail. Use a **real** URL for optional checks, or leave it unset.

### 4c) Why this matters

Taker **`coinswapd`** dials maker **`http://*.onion:…`** via Go `net/http`, which reads **`HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`**, **not** `ALL_PROXY`. See [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md).

---

## 5) Taker — build binaries

With proxy **sourced** in Terminal 1:

```bash
cd /path/to/mwixnet-litvm
source ~/mln-taker-proxy.sh
make build-research-coinswapd build-mln-sidecar build-mln-cli
```

---

## 6) Taker — start `coinswapd-research` (taker flags)

Still **same shell** (proxy env must be **inherited** by `coinswapd`):

```bash
source ~/mln-taker-proxy.sh
./bin/coinswapd-research \
  -mln-local-taker \
  -l 8546 \
  -a 'ltcmweb1YOUR_TAKER_FEE_ADDRESS' \
  -mweb-scan-secret YOUR_64_HEX_SCAN \
  -mweb-spend-secret YOUR_64_HEX_SPEND
```

Add any other flags your wallet / Neutrino setup needs. **Leave running.**

Preconditions (exact coin amount, Neutrino sync): [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md) **Real funded operator path**.

---

## 7) Taker — start `mln-sidecar` (second window)

Open **another** terminal tab/window **on the taker machine** (proxy on **coinswapd** is what matters for `.onion`; sidecar only talks to **local** `127.0.0.1:8546` by default):

```bash
cd /path/to/mwixnet-litvm
./bin/mln-sidecar -mode=rpc -rpc-url http://127.0.0.1:8546 -port 8080
```

Leave running. See [mln-sidecar/README.md](../mln-sidecar/README.md).

---

## 8) Taker — `mln-cli` env (must match makers)

`mln-cli scout` / `pathfind` need the **same** LitVM and Nostr view as the makers. Typical sources:

- **Local Anvil lab:** after `./scripts/e2e-bootstrap.sh`, `source deploy/e2e.generated.env` (makers must use **matching** registry/court/relay from that deploy).
- **Your own testnet/mainnet:** set `MLN_*` per `mln-cli` help ([environment list in `mln-cli`](../mln-cli/cmd/mln-cli/main.go)).

Minimum set (names only):

- `MLN_NOSTR_RELAYS` (or `MLN_NOSTR_RELAY_URL`)
- `MLN_LITVM_HTTP_URL`
- `MLN_REGISTRY_ADDR`
- `MLN_LITVM_CHAIN_ID`
- `MLN_GRIEVANCE_COURT_ADDR` (if ads include court)

Then:

```bash
source deploy/e2e.generated.env   # or export your own MLN_* first
./bin/mln-cli scout
./bin/mln-cli pathfind -json > route.json
jq '.hops[].tor' route.json    # each hop should be a real http://*.onion:port
```

If **`tor`** is empty or wrong, fix **`MLND_TOR_ONION`** on the makers and wait for new ads.

---

## 9) Taker — forger + batch (real swap attempt)

Advisory script:

```bash
./scripts/phase3-funded-env-check.sh
```

Run forger (adjust **dest**, **amount**, paths to your wallet):

```bash
./bin/mln-cli forger -route-json route.json -dry-run=false \
  -dest "$E2E_MWEB_DEST" -amount "$E2E_MWEB_AMOUNT" \
  -coinswapd-url http://127.0.0.1:8080/v1/swap \
  -trigger-batch -wait-batch
```

**README Phase 3 bar:** `pendingOnions` reaches **0** **without** dev-only clear flags (`E2E_MWEB_FUNDED_DEV_CLEAR`, `-mweb-dev-clear-pending-after-batch`). See [PHASE_3_OPERATOR_CHECKLIST.md](PHASE_3_OPERATOR_CHECKLIST.md).

---

## 9a) Automation — what “full Part B” means

| Goal | Automatable in one script / CI? | Notes |
|------|---------------------------------|--------|
| **Same control plane as Part B** (`scout` → `pathfind` → `forger` → batch → `mweb_getRouteStatus`) with **stub or local RPC**, **cleartext** hop URLs | **Yes (already)** | **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** — this is **Part A / Phase 3a**, not the README **Phase 3** checkbox. See [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md). |
| **Tor SOCKS + `HTTP_PROXY` sanity** | **Yes** | **`make phase3-operator-preflight`** / **`./scripts/tor-preflight.sh`**. |
| **Dial one real maker `.onion` JSON-RPC** (optional POST) | **Yes, if you supply the URL** | Set **`PHASE3_ONION_JSONRPC_URL`** to a **real** `http://….onion:port`; not the playbook placeholder. |
| **Full Part B as in this doc** — **three** distinct HS, **three** maker `coinswapd` + `mlnd`, **taker** mainnet **`coinswapd-research`**, live **`swap_forward` / `swap_backward`**, **`pendingOnions == 0`** without dev-clear | **No (not as a single checked-in CI job)** | Needs **long-lived Tor**, **real or lab LitVM + Nostr alignment**, **keys / funds**, and **stable inter-hop RPC**; too slow and flaky for default GitHub Actions, and **mainnet** money is unsafe to run unattended. |
| **Future: Tor-“shaped” stub lab** (Compose: Tor + multiple **`mw-rpc-stub`** backends + dynamic `hostname` → `MLND_TOR_ONION`) | **Possible engineering** | Would **stress Tor transport + ads + pathfind** only; it still would **not** prove live MWEB P2P or close the README Phase 3 bar. |

**Practical split:** Automate **everything you can** (builds, preflight, Part A regression, optional curl through Tor). Treat **operator Part B** as **semi-automated**: you script **your** `torrc`, **systemd**/Compose for processes, and env files — the repo does not ship a one-button **mainnet** multi-hop because of safety and environment variance.

---

## 10) When something fails (first places to look)

| Symptom | Check |
|---------|--------|
| `pathfind` finds no route | `mln-cli scout` — are there **3** verified makers with **non-empty** `tor`? |
| curl / preflight cannot reach `.onion` | Tor running? Correct **SOCKS** port? Maker `coinswapd` listening on **127.0.0.1:PORT**? Firewall? |
| taker cannot dial next hop | Taker **`coinswapd`** process has **`HTTP_PROXY=socks5h://…`** (not only the shell you used for `mln-cli`). |
| hangs mid swap | Maker logs (`swap_forward` / `swap_backward`); HS port matches `route.json` **`tor`** field. |

---

## 11) Security hygiene

- **`chmod 600`** on any file holding `nsec`, EVM keys, or MWEB scan/spend secrets.
- Never commit **real** `.env` with secrets; use **`.example`** files only in git.
- Do not paste **full** route payloads or onion strings into public tickets if they correlate a live run.

---

**You are done with this doc** when you either complete a **live three-hop** attempt per your lab goals or have a **specific failing step** — use **Section 10** and the **Tor lab** triage bullets next.
