# Maker dashboard: end-to-end setup

This walks you from zero to opening the **mlnd** Maker control center in a browser. The UI is served only when **`MLND_DASHBOARD_ADDR`** is set; it binds to **loopback** by default.

For product context (three pillars, grievance flows), see [`research/WALLET_MAKER_FLOW_V1.md`](../research/WALLET_MAKER_FLOW_V1.md).

---

## 0. What you need installed

| Tool | Why |
|------|-----|
| **Go 1.22+** and a **C compiler** | `mlnd` uses CGO + SQLite |
| **Docker** (optional but easiest) | Run Anvil + Foundry deploy without a local `forge` install |
| A **web browser** | Open the dashboard |

Work from the **repository root** (`mwixnet-litvm`), not `mln-cli/`, when you run `make build` and `./bin/mlnd`.

---

## 1. Shell hygiene (avoid silent bugs)

**Each `export` must be a separate shell command.** If you paste one long line like:

`export A=1export B=2`

the shell sets `A` to garbage and never sets `B` correctly.

**Ways that work:**

- Paste **multiple lines** (real newlines between `export` lines), or  
- Use **semicolons** on one line: `export A=1; export B=2; export C=3`

**zsh:** Do not put `# comments` on the **same** line as `export` unless you ran `setopt interactivecomments`.

**Addresses:** `MLND_COURT_ADDR`, `MLND_OPERATOR_ADDR`, and (with the dashboard) `MLND_REGISTRY_ADDR` must be real **`0x` + 40 hex digits**, not README placeholders. `mlnd` rejects the zero address, invalid hex, and strings containing `YOUR`.

---

## 2. Path A — Local Anvil (recommended to explore the UI)

Use this when you do **not** have public LitVM RPC yet. Chain id is **31337**.

### Terminal 1: Anvil

```bash
docker run --rm -p 8545:8545 --entrypoint anvil ghcr.io/foundry-rs/foundry:latest --host 0.0.0.0
```

Leave this running.

### Terminal 2: Deploy contracts

From **repo root**:

```bash
make deploy-local
```

(or `./scripts/deploy-local-anvil.sh`)

Watch the script output. You should see lines like:

```text
MwixnetRegistry: 0x...
GrievanceCourt: 0x...
```

Copy those two addresses exactly (lowercase is fine).

### Operator address (who you are “watching” as)

The deploy script uses Anvil’s **first** test private key. The matching address (local testing only) is:

`0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266`

Use that as **`MLND_OPERATOR_ADDR`** unless you deliberately register and run as another key.

### Terminal 3: Build and run `mlnd` with the dashboard

Still from **repo root**, set variables using **your** registry and court addresses from step 2. Example shape (replace `0x…registry…` and `0x…court…`):

```bash
export MLND_WS_URL=ws://127.0.0.1:8545
export MLND_COURT_ADDR=0x…court…
export MLND_OPERATOR_ADDR=0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
export MLND_REGISTRY_ADDR=0x…registry…
export MLND_LITVM_CHAIN_ID=31337
export MLND_DASHBOARD_ADDR=127.0.0.1:9842
```

Optional — **read-only** relay check (no publishing without `MLND_NOSTR_NSEC`):

```bash
export MLND_NOSTR_RELAYS=wss://relay.damus.io
```

Build and start:

```bash
make build
./bin/mlnd
```

You should see:

- `mlnd dashboard: listening on http://127.0.0.1:9842/`
- `mlnd: watching GrievanceOpened accused=0xf39f… court=0x…` (non-zero court and accused)

### Open the UI

In a browser: **http://127.0.0.1:9842/**

If you set **`MLND_HTTP_TOKEN`**, open:

**http://127.0.0.1:9842/?token=YOUR_TOKEN**

(EventSource cannot send custom headers; the query param is how the page authenticates.)

### Quick API check (optional)

```bash
curl -sS http://127.0.0.1:9842/api/v1/status | head -c 500
```

---

## 3. Path B — LitVM testnet (production-shaped)

1. Get **WebSocket RPC**, **HTTP RPC** (for tooling), **chain id**, and deployment addresses from [LitVM documentation](https://docs.litvm.com/) and [`research/LITVM.md`](../research/LITVM.md).
2. Set `MLND_WS_URL` to the **WebSocket** URL (not the HTTP URL).
3. Set `MLND_COURT_ADDR`, `MLND_REGISTRY_ADDR`, and `MLND_OPERATOR_ADDR` to **your** maker identity on that chain.
4. Set `MLND_LITVM_CHAIN_ID` to the **decimal** chain id LitVM documents.
5. Add `MLND_DASHBOARD_ADDR` as in Path A.

Do not guess chain id or contract addresses.

---

## 4. Publishing Nostr maker ads (optional)

To **publish** kind **31250** (not just read relays in the dashboard), you also need:

| Variable | Purpose |
|----------|---------|
| `MLND_NOSTR_NSEC` | Nostr signing key (**nsec1…** or 64-char hex) |
| `MLND_NOSTR_RELAYS` | Comma-separated `wss://…` |
| `MLND_LITVM_CHAIN_ID` | Decimal string |
| `MLND_REGISTRY_ADDR` | Registry contract |
| `MLND_COURT_ADDR` | Grievance court |
| `MLND_OPERATOR_ADDR` | Maker address (must match on-chain registration when you go live) |

Optional: `MLND_SWAP_X25519_PUB_HEX` (64 hex digits) for onion routing in the ad — see [`research/COINSWAPD_MLN_FORK_SPEC.md`](../research/COINSWAPD_MLN_FORK_SPEC.md).

If `MLND_NOSTR_RELAYS` is set but **`MLND_NOSTR_NSEC` is empty**, `mlnd` **does not** publish; it logs once and the dashboard can still query relays.

---

## 5. Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `watching … accused=0x000… court=0x000…` | Bad paste (one-line exports) or invalid placeholder addresses. Fix shell pastes; redeploy and copy real addresses. |
| `export: not valid in this context: d-tag` | Inline `#` comment on same line as `export` in zsh. Remove or use `setopt interactivecomments`. |
| `nostr broadcaster config: … empty signing key` then exit | **Older binary.** Rebuild from current `main`: relays without `nsec` should only disable publishing, not exit. |
| Dashboard 401 | Set `MLND_HTTP_TOKEN` and open `/?token=…` or send `X-MLND-Token`. |
| `dashboard bind must be loopback` | Use `127.0.0.1:port` or set `MLND_DASHBOARD_ALLOW_LAN=1` (only if you understand the risk). |

---

## 6. Where this is documented elsewhere

- Operator env tables: [`mlnd/README.md`](README.md)
- Local deploy script: [`scripts/deploy-local-anvil.sh`](../scripts/deploy-local-anvil.sh)
- Contracts quick start: [`contracts/README.md`](../contracts/README.md)
