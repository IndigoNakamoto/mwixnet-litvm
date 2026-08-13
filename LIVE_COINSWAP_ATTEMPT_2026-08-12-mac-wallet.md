# Live coinswap attempt — 2026-08-12 (Mac wallet inspect)

**Bar:** **Proof A** (lab shuffle). Not Proof B.

Proof A means three mixnode `coinswapd` processes complete peel → `swap_forward` / `swap_backward` → one aggregated MWEB tx, and the dest wallet scans a **new same-value coin**. Three Tor HS on one host is enough. It does **not** show “one honest mixer.”

**Outcome:** **Not a pass.** Gate 1 inspect CLI is in place; coin list and funded submit did **not** run. `WALLET_PASSPHRASE` was not available (no `deploy/mixnet.operator.env` passphrase). Plan said stop until a mature coin amount is confirmed. No mainnet MWEB was spent.

**LitVM / Nostr / `mlnd`:** not used. No scout, no registry `eth_call`, no maker ads. **Do not** put CoinSwap in the Tauri Mac wallet until a file like this is a **pass**.

Do not overwrite [`LIVE_COINSWAP_ATTEMPT_2026-08-12.md`](LIVE_COINSWAP_ATTEMPT_2026-08-12.md) or [`LIVE_COINSWAP_ATTEMPT_2026-04-15.md`](LIVE_COINSWAP_ATTEMPT_2026-04-15.md).

## Environment

- **Mixnet:** `mwixnet-litvm` (this repo).
- **Taker coin source:** `/Users/indigo/Dev/ltc-wallet-mac` data dir `~/Library/Application Support/com.indigonakamoto.ltc-wallet/` (scheme `litecoin-core`). GUI was not holding the data-dir lock.
- **Secrets:** No scan/spend hex, no mnemonic, no passphrase in this file.

## 1. Inspect CLI (Gate 1)

**Result:** **Shipped.** Listing coins **blocked** (no passphrase).

`wallet-cli` commands (no Tauri CoinSwap UI):

- `combined-summary`
- `mweb-unspent`
- `mweb-coinswapd-env --out FILE` (chmod 600, merges into existing env, never prints scan/spend)

Operator helper: [`scripts/proof-a-inspect.sh`](scripts/proof-a-inspect.sh) (sources gitignored `deploy/mixnet.operator.env`).

```text
./scripts/proof-a-inspect.sh
# error: WALLET_PASSPHRASE is not set.
```

## 2. Mixnode directory flag

**Result:** **Code in-tree.** Mixnodes are not on the public `getNodes` mesh.

`coinswapd-research -mln-directory FILE -mln-hop-index N` pins `mlnPeers` and checks `-k` against that hop’s `swapX25519PubHex`. Tests: `TestLoadMLNDirectoryPeers`. Playbook Part B0 and checklist updated.

## 3. Tor HS + Scout-free route

**Result:** **HS published.** Funded 3-hop **not run**.

`make phase3-operator-preflight` passed (SOCKS + `IsTor:true`). Three Proof A hidden services were added on this host (hop 0 virtual port 8334 → taker `8546`; hops 1–2 on 8335/8336). Operator directory is gitignored (`deploy/mixnet.directory.json`); example placeholders remain in [`deploy/mixnet.directory.example.json`](deploy/mixnet.directory.example.json).

```text
./bin/mln-cli route from-directory -directory deploy/mixnet.directory.json -out /tmp/proof-a-route.json
```

Wrote three hops, `feeMinSat=10000`, `amountSat=10000000` (lab default until a coin is confirmed). No `operator` / `epochId` / `accuser` / `swapId`.

Mixnodes were **not** started: `-a` needs a real `ltcmweb1…` from the wallet export. Forger / `mweb_runBatch` **not** run. **Never** `-mweb-dev-clear-pending-after-batch` for a claimed pass.

## What would make the next Proof A a pass

1. `chmod 600 deploy/mixnet.operator.env` with `WALLET_PASSPHRASE`, quit the Mac wallet GUI, `./scripts/proof-a-inspect.sh`.
2. Confirm one **mature unlocked** coin with `amount_sats > 30000`. Set directory `amountSat` to that exact value (or split in the existing wallet UI first).
3. `wallet-cli mweb-coinswapd-env --out deploy/mixnet.operator.env` (scan/spend stay in that file).
4. Start hop 0 (`-mln-local-taker -mln-directory … -mln-hop-index 0 -l 8546`) + hops 1–2 (`-mln-hop-index 1|2`); sidecar; Neutrino sees the exact coin; forger without LitVM; dest scans a new same-value coin.
5. New dated file labeled **Proof A**. Do not overwrite this one.

**Proof B** remains parked.

---

*This file does not claim README Phase 3-mix is complete. It does not claim Proof B.*
