# Phase 3 — Operator checklist (real Tor, multi-hop)

Single-page sequence for **Phase 3-mix** (permissioned `coinswapd` shuffle). **Phase 3-gov** (Nostr, Scout, LitVM court) is parked until a dated live-attempt file is a **Proof A pass**.

**Prefer a linear walkthrough?** Start with [PHASE_3_OPERATOR_PLAYBOOK.md](PHASE_3_OPERATOR_PLAYBOOK.md) **Part B0**. Deep dives: [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md), [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md).

## North star (README checkboxes)

**Phase 3-mix** (this checklist) is complete only when:

1. **Proof A — lab shuffle:** three mixnode `coinswapd` processes (three Tor HS on one host is enough) complete peel → **`swap_forward` / `swap_backward`** → **one aggregated MWEB tx**, and the **dest wallet scans a new same-value coin**. No LitVM, Nostr, or **`mlnd`** in the log. Label this **Proof A**; it does **not** show “one honest mixer.”
2. **`pendingOnions == 0` without** `-mweb-dev-clear-pending-after-batch` is **operator hygiene**, not the product bar.

**Proof B** (stranger / 1 LTC / published first hop / 1-of-N honest) is a **later** bar. Do not claim it from Proof A. **Do not** put CoinSwap in the Mac wallet until a [`LIVE_COINSWAP_ATTEMPT_*.md`](../LIVE_COINSWAP_ATTEMPT_2026-04-15.md) file is a pass (at least Proof A).

**Phase 3-gov** (public LitVM grievance / Nostr ads) is **not** a mix completion condition. Court work waits until rounds complete without it.

Until then, keep **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** and optional **`MWEB_RPC_BACKEND=coinswapd`** smoke green after any handoff change ([PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md)).

**Quick preflight:** `make phase3-operator-preflight` (Tor SOCKS + printed `HTTP_PROXY` / `NO_PROXY` template + optional onion JSON-RPC ping).

---

## A — Tor and `coinswapd` proxy (every hop process)

- [ ] Run **`./scripts/tor-preflight.sh`** or **`make tor-preflight`** (9150 for Tor Browser if needed).
- [ ] On **each `coinswapd-research` that dials `.onion` peers**, export before start:

```bash
export HTTP_PROXY="socks5h://127.0.0.1:9050"   # host/port match your SOCKS
export HTTPS_PROXY="$HTTP_PROXY"                # if any hop URL uses https://
export NO_PROXY="127.0.0.1,localhost"
```

**Why:** go-ethereum **`rpc.Dial`** uses **`net/http`**; **`ProxyFromEnvironment`** reads **`HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`** — **not** **`ALL_PROXY`** ([PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md)).

- [ ] **1-hop sanity:** With the same exports, prove HTTP JSON-RPC to **one** maker `.onion` (e.g. set **`PHASE3_ONION_JSONRPC_URL`** and run **`make phase3-operator-preflight`**, or curl manually with **`--socks5-hostname`** as in the Tor lab doc).

---

## B — Topology: MLN taker vs mesh maker

- [ ] **Taker** (route from **`mweb_submitRoute`**, dynamic peers): run **`coinswapd-research`** with **`-mln-local-taker`** so startup does not depend on **`getNodes()` / `AliveNodes`** mesh match.
- [ ] **Listed mixnode (permissioned directory):** **`-mln-directory FILE -mln-hop-index N`** so startup does not probe the public mesh; **`-k`** must match that hop’s `swapX25519PubHex`.
- [ ] **Listed mesh maker (3-gov only):** omit **`-mln-local-taker`**; **`-k`** must match your row in the probed topology ([PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md)).

---

## C — Three mixnodes: hidden service + published directory (no `mlnd`)

For **each** of three **mixnodes** (`coinswapd-research` only):

- [ ] **Tor hidden service** exposes mixnode JSON-RPC (port you publish in the directory, e.g. `8334`).
- [ ] **`-k`** X25519 matches the directory **`swapX25519PubHex`**; **`-a`** fee MWEB is a full **`ltcmweb1…`** address.
- [ ] Publish hops in a hop directory (copy [`deploy/mixnet.directory.example.json`](../deploy/mixnet.directory.example.json)): `tor`, `swapX25519PubHex`, **`feeMinSat`** that covers the `backward()` weight floor, and **`amountSat`** (one denomination).
- [ ] Taker builds **`route.json`** with **`mln-cli route from-directory`** (no Scout, no LitVM). Do **not** run **`mlnd`** or Nostr ads for 3-mix.

**3-gov only (parked):** `mlnd` ads, Scout, and registry registration — see Part F.

---

## D — Taker client (not a hop)

The taker is a **client** (sidecar + local onion build / **`-mln-local-taker`** plumbing). Users do not run a mixnode to mix.

- [ ] Neutrino synced; **`-mweb-scan-secret`**, **`-mweb-spend-secret`**, **`-a`** fee MWEB (**`ltcmweb1…`** full address).
- [ ] Exact UTXO value **`== amountSat`** from the directory.
- [ ] **`mln-sidecar -mode=rpc`** → taker **`coinswapd`** JSON-RPC; **`HTTP_PROXY=socks5h://…`** on **taker `coinswapd`** (and on mixnodes that dial `.onion` peers).
- [ ] **`mln-cli route from-directory`** → **`route.json`** (no `epochId` / `accuser` / `operator`).
- [ ] **`mln-cli forger`** … **`-trigger-batch`** against sidecar **`POST /v1/swap`**. Wait for epoch / **`mweb_runBatch`**.

**Product success bar:** dest wallet **scans a new same-value MWEB coin**. **`pendingOnions == 0` without** **`E2E_MWEB_FUNDED_DEV_CLEAR`** is supporting hygiene.

Optional: **`./scripts/phase3-funded-env-check.sh`** before forger (warns if dev-clear env is set when you claim a production-shaped run).

---

## E — Maker evidence bridge (parallel)

- [ ] Patched **`coinswapd`** NDJSON receipts + **`MLND_BRIDGE_RECEIPTS_DIR`** shared mount ([PHASE_9_ENABLEMENT.md](../PHASE_9_ENABLEMENT.md)).
- [ ] **`MLND_DEFEND_AUTO`** / key hygiene only after correlators and **LitVM** addresses are correct; see [THREAT_MODEL_MLN.md](THREAT_MODEL_MLN.md).

---

## F — Phase 3-gov (parked until Proof A passes)

LitVM registry/court, Nostr ads, and **`mlnd`** on mixnodes are **not** required for 3-mix. After a Proof A pass, follow [PHASE_16_PUBLIC_TESTNET.md](../PHASE_16_PUBLIC_TESTNET.md) section 0 if you still want a court.

---

## Failure triage (no secret logging)

| Symptom | Check |
|--------|--------|
| Dial timeout to `.onion` | Tor, **`HTTP_PROXY`**, HS port, firewall |
| **`swap_forward:`** / **`swap_backward:`** errors | Peer URLs, **`-mln-local-taker`** vs mesh, logs ([swap.go](coinswapd/swap.go)) |
| Stuck **`pendingOnions`** | Peer reachability vs missing finalize — see [PHASE_3_MWEB_HANDOFF_SLICE.md](../PHASE_3_MWEB_HANDOFF_SLICE.md) |

Do not paste live onions, keys, or payloads into public tickets.
