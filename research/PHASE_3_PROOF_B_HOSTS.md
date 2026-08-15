# Proof B — from this Mac to three other computers

Walk this **top to bottom**. It assumes **this Mac** is the only machine that already has the MLN repo, Go builds, Tor client, and an MWEB wallet. The **other three computers** start with none of that. They never get a wallet or scan/spend keys.

Canonical flags and product bar: [PHASE_3_OPERATOR_PLAYBOOK.md](PHASE_3_OPERATOR_PLAYBOOK.md) Part B1. Tor SOCKS: [PHASE_3_TOR_OPERATOR_LAB.md](PHASE_3_TOR_OPERATOR_LAB.md). Checklist: [PHASE_3_OPERATOR_CHECKLIST.md](PHASE_3_OPERATOR_CHECKLIST.md).

**Do not spend 1 LTC** until hop 0’s Neutrino is at tip and you can JSON-RPC hop 0’s `.onion` from this Mac. **Do not** overwrite [`LIVE_COINSWAP_ATTEMPT_2026-08-12-proof-a-pass.md`](../LIVE_COINSWAP_ATTEMPT_2026-08-12-proof-a-pass.md). Success is a new dest coin of **99970000 sat**, not `pendingOnions==0`.

LitVM, Nostr, Scout, and `mlnd` stay off every machine in this run.

---

## Who does what

| Computer | Role | MWEB wallet? | What it runs |
| -------- | ---- | ------------ | ------------ |
| **This Mac** | Coordinator + **taker** | **Yes** — scan/spend and dest stay here | Tor **client**, `coinswapd-research` (`-mln-local-taker -mln-submit-remote`), `mln-sidecar`, `mln-cli` |
| **Hop 0 host** | Published first mixnode | **No** | Tor **hidden service** + `coinswapd-research` (`-mln-directory -mln-hop-index 0`). Operator runs `mweb_runBatch` |
| **Hop 1 host** | Middle mixnode | **No** | Tor HS + `coinswapd-research` (`-mln-hop-index 1`) |
| **Hop 2 host** | Last mixnode | **No** | Tor HS + `coinswapd-research` (`-mln-hop-index 2`) |

Fill this in before you copy files (hostnames, SSH, OS):

| Role | SSH / hostname | OS + CPU (`uname -m`) |
| ---- | -------------- | --------------------- |
| Hop 0 | | linux amd64 / linux arm64 / darwin arm64 / … |
| Hop 1 | | |
| Hop 2 | | |

The taker is **not** a hop. Hop 0’s `.onion` is the address a stranger would be given.

---

## What mixnodes need (and what they must not get)

**Each hop computer needs**

- Always-on box, outbound internet (Litecoin **clearnet** P2P for Neutrino — this is **not** over Tor).
- Disk for Neutrino (`neutrino.db` grows; plan several GB).
- **Tor** with SOCKS on `127.0.0.1:9050` and one **v3 hidden service**.
- **Python 3** (the start script parses the env file).
- `curl` on hop 0 (batch script).
- The **`coinswapd-research` binary matching that OS/CPU**, plus a thin repo-shaped directory (below).
- One **32-byte hex** mix key (`MIX_K`) whose **public** hex is in the shared directory.
- One **`ltcmweb1…` fee address** on `-a`. That is a **public** receive string, not a wallet.

**Each hop must not have**

- `MWEB_SCAN_SECRET` / `MWEB_SPEND_SECRET` / `WALLET_PASSPHRASE`.
- `deploy/mixnet.operator.env` from this Mac.
- `-mln-local-taker` or `-mln-submit-remote`.
- `mlnd`, LitVM RPC, Nostr relays, Docker compose stacks.

**Neutrino vs Tor:** hop-to-hop JSON-RPC (`swap_swap` / `swap_forward` / `swap_backward`) goes through Tor (`HTTP_PROXY=socks5h://127.0.0.1:9050`). Chain sync is ordinary Litecoin P2P. A hop that can only talk Tor and cannot reach Litecoin peers will never `FetchCoin` the taker UTXO.

---

## Never (all computers)

- Paste passphrase, scan/spend, or `MIX_K` into chat, tickets, or screenshots.
- Commit `deploy/mixnet.operator.env`, `deploy/mixnet.hop.env`, or `deploy/mixnet.directory.json`.
- Run `pgrep -lf coinswapd` (argv prints keys). Use `lsof -iTCP:8334 -sTCP:LISTEN` (or the listen port you chose) and kill by PID.
- Copy this Mac’s wallet data dir to a hop.
- Use `-trigger-batch` on the taker forger, or `-mweb-dev-clear-pending-after-batch` for a claimed pass.
- Claim Proof B from Proof A.

---

## Step 0 — This Mac: confirm you still have the stack

```bash
cd /path/to/mwixnet-litvm
make build-research-coinswapd build-mln-sidecar build-mln-cli
make phase3-operator-preflight
```

Quit the Tauri wallet GUI if you will inspect coins later (exclusive data-dir lock).

If Proof A hops are still bound on this Mac (`8546`, `8334`/`8335`/`8336`, `8080`), stop those PIDs **before** the taker steps. Mixnodes on other computers do not use those local ports.

---

## Step 1 — This Mac: build a binary for each hop OS

Default `make build-research-coinswapd` produces a **darwin** binary for *this* Mac. Other computers need a binary for **their** `GOOS`/`GOARCH`.

From repo root (example: Linux VPS x86_64):

```bash
mkdir -p bin
cd research/coinswapd
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../../bin/coinswapd-research-linux-amd64 .
cd ../..
```

Other common targets:

| Hop `uname -s` / `uname -m` | Build env |
| --------------------------- | --------- |
| Linux x86_64 | `GOOS=linux GOARCH=amd64` |
| Linux aarch64 (many cloud ARM / Pi 64-bit) | `GOOS=linux GOARCH=arm64` |
| macOS Apple Silicon | `GOOS=darwin GOARCH=arm64` (or just `make build-research-coinswapd` on this Mac and copy `bin/coinswapd-research`) |
| macOS Intel | `GOOS=darwin GOARCH=amd64` |

If all three hops share one OS/CPU, build once. Name the output so you do not scp the Mac binary onto Linux by mistake.

---

## Step 2 — This Mac: mix keys and fee addresses

Mixnodes have no wallet. They only need (1) an X25519 **private** hex for `-k` and (2) an `ltcmweb1…` string for hop fees.

### 2a) Fee address

Until a hop operator has their own MWEB wallet, use a receive address from **this Mac’s** wallet (`E2E_MWEB_DEST` in gitignored `deploy/mixnet.operator.env`, or another receive string from the same wallet). Put that **public** string in each hop’s `E2E_MWEB_DEST`. Fees will scan here. You are not putting scan/spend on the hop.

When a hop operator later has a wallet, they replace that field with **their** `ltcmweb1…` and restart. Do not send them scan/spend from this Mac.

### 2b) Three mix keys (keep private)

On this Mac, generate three 32-byte hex secrets. Store them only in files you will copy to **one** hop each (`chmod 600`). Do not echo them.

```bash
openssl rand -hex 32   # hop 0 MIX_K — write into hop0 env only
openssl rand -hex 32   # hop 1
openssl rand -hex 32   # hop 2
```

### 2c) Public hex for the directory

`swapX25519PubHex` is the X25519 **public** key (64 lowercase hex) for that `MIX_K`. Run this on this Mac (stdlib only; do not paste `MIX_K` into chat). Replace the placeholder with that hop’s private hex **in your own terminal**:

```bash
cat > /tmp/x25519pub.go << 'EOF'
package main

import (
	"crypto/ecdh"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	b, err := hex.DecodeString(os.Args[1])
	if err != nil {
		panic(err)
	}
	k, err := ecdh.X25519().NewPrivateKey(b)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(k.PublicKey().Bytes()))
}
EOF
go run /tmp/x25519pub.go "$MIX_K"
```

Write down **only** the three public hex strings (and which hop they belong to). You will put those in `mixnet.directory.json`. Keep `MIX_K` out of that file.

---

## Step 3 — Each hop computer: OS packages only

SSH to hop 0, then hop 1, then hop 2. Repeat this step on each. No git clone required yet.

### Linux (Debian/Ubuntu)

```bash
sudo apt-get update
sudo apt-get install -y tor python3 curl
sudo systemctl enable --now tor
```

### macOS hop

```bash
brew install tor python3
brew services start tor
```

Confirm SOCKS (on that hop):

```bash
nc -z -w 3 127.0.0.1 9050 && echo socks-ok
```

If that hop uses Tor Browser instead of system Tor, SOCKS is often **9150**. Then every `HTTP_PROXY` on that box must use `9150`.

---

## Step 4 — Each hop computer: hidden service, then send the hostname back

One **fresh** `HiddenServiceDir` per hop. Virtual port **8334** must match `LISTEN_PORT` you will put in that hop’s env.

### Linux `torrc` (`/etc/tor/torrc`) — hop 0 example

```text
HiddenServiceDir /var/lib/tor/mln-hop0
HiddenServicePort 8334 127.0.0.1:8334
```

Create the directory and give it to the Tor daemon user (often `debian-tor`):

```bash
sudo mkdir -p /var/lib/tor/mln-hop0
sudo chown debian-tor:debian-tor /var/lib/tor/mln-hop0
sudo chmod 700 /var/lib/tor/mln-hop0
sudo systemctl restart tor
sudo cat /var/lib/tor/mln-hop0/hostname
```

Use `mln-hop1` / `mln-hop2` on the other two hosts (do not share one `HiddenServiceDir`).

### macOS hop

`torrc` is often `/opt/homebrew/etc/tor/torrc`. `HiddenServiceDir` under `$(brew --prefix)/var/tor/mln-hop0`. Restart with `brew services restart tor`. Read `hostname` from that directory.

You get one line: `…………………….onion`. The public mix URL is:

```text
http://…………………….onion:8334
```

Send **only that URL** (or hostname + port) back to this Mac. Not `MIX_K`. Not wallet keys.

Do this for all three hops before Step 5. You cannot finish the directory without three hostnames.

---

## Step 5 — This Mac: write the shared directory

Copy the example and **do not commit** the real file:

```bash
cp deploy/mixnet.directory.example.json deploy/mixnet.directory.json
```

Edit `deploy/mixnet.directory.json`:

- `amountSat`: **100000000** (1 LTC). The example file’s 10000000 is Proof A.
- Hop 0 / 1 / 2 `tor`: the three `http://….onion:8334` URLs from Step 4, in order.
- Each `swapX25519PubHex`: that hop’s **public** hex from Step 2c, **64 lowercase** characters.
- Each `feeMinSat`: **10000**.

Same file goes to every hop and stays on this Mac for the taker. If you change a hop’s onion or key, rewrite the file, copy it again, and restart that hop.

---

## Step 6 — This Mac: one env file per hop, then a tarball

For **each** hop, create a file you will copy **only to that host**. Do not put all three `MIX_K` values on every hop.

```bash
mkdir -p /tmp/mln-hop0 /tmp/mln-hop1 /tmp/mln-hop2
```

Layout the start script expects (repo-shaped):

```text
bin/coinswapd-research          # rename your GOOS binary to this name inside the tarball
deploy/mixnet.hop.env           # chmod 600
deploy/mixnet.directory.json    # same file for all hops
scripts/mixnode-start.sh
scripts/mixnode-run-batch.sh    # required on hop 0; optional on 1–2
```

`deploy/mixnet.hop.env` (from [`deploy/mixnet.hop.env.example`](../deploy/mixnet.hop.env.example)):

```text
MIX_K=<that hop only>
E2E_MWEB_DEST=<ltcmweb1… fee address from Step 2a>
MLN_HOP_INDEX=0
LISTEN_PORT=8334
MIXNET_DIRECTORY=
```

Set `MLN_HOP_INDEX` to `0`, `1`, or `2`. Leave `MIXNET_DIRECTORY` empty to use `deploy/mixnet.directory.json` next to the scripts’ repo root. `chmod 600` the env file.

Example pack for hop 0 (adjust binary name):

```bash
STAGE=/tmp/mln-hop0
mkdir -p "$STAGE/bin" "$STAGE/deploy" "$STAGE/scripts"
cp bin/coinswapd-research-linux-amd64 "$STAGE/bin/coinswapd-research"
chmod +x "$STAGE/bin/coinswapd-research"
cp deploy/mixnet.directory.json "$STAGE/deploy/"
cp scripts/mixnode-start.sh scripts/mixnode-run-batch.sh "$STAGE/scripts/"
# write $STAGE/deploy/mixnet.hop.env (MIX_K for hop 0 only), then:
chmod 600 "$STAGE/deploy/mixnet.hop.env"
chmod +x "$STAGE/scripts/"*.sh
tar -C /tmp -czf /tmp/mln-hop0.tar.gz mln-hop0
```

Repeat for hop 1 and hop 2 with **their** `MIX_K` and `MLN_HOP_INDEX`. Then copy:

```bash
scp /tmp/mln-hop0.tar.gz user@hop0-host:
scp /tmp/mln-hop1.tar.gz user@hop1-host:
scp /tmp/mln-hop2.tar.gz user@hop2-host:
```

After a successful copy, delete the tarballs from `/tmp` on this Mac if you do not want three private keys sitting in a shared temp directory. Do not scp `deploy/mixnet.operator.env`.

---

## Step 7 — Each hop: unpack and start

On that hop:

```bash
mkdir -p ~/mln-mixnode
tar -C ~/mln-mixnode --strip-components=1 -xzf ~/mln-hop0.tar.gz   # use the matching tarball name
cd ~/mln-mixnode
chmod 600 deploy/mixnet.hop.env
./scripts/mixnode-start.sh
```

Leave it running (tmux/systemd). Stderr should show `mixnode-start: hop-index=N` and `mln-directory: hop M of 3` (hop-index `0` logs as hop **1 of 3**). Then Neutrino `Syncing height …`.

If it exits with “swapX25519PubHex does not match this process `-k` public key”, the directory pub for that index does not match `MIX_K` on this box. Fix the JSON or the env; do not “try another `-k`”.

**Hop 0** must reach chain tip before anyone submits. `validateOnion` / `FetchCoin` runs there.

Optional systemd sketch (Linux): `WorkingDirectory=` the unpack dir, `ExecStart=` `/home/…/mln-mixnode/scripts/mixnode-start.sh`, `Environment=HTTP_PROXY=socks5h://127.0.0.1:9050` (the script also sets that default). Do not put `MIX_K` in a world-readable unit file; keep it in `deploy/mixnet.hop.env` mode 600.

---

## Step 8 — This Mac: prove hop 0’s onion over Tor

SOCKS on **this Mac** (system Tor `9050` or Tor Browser `9150`):

```bash
cd /path/to/mwixnet-litvm
make phase3-operator-preflight
```

Then (replace with hop 0’s real URL; this does not print keys):

```bash
export HTTP_PROXY=socks5h://127.0.0.1:9050
export NO_PROXY=127.0.0.1,localhost
curl --socks5-hostname 127.0.0.1:9050 -sS -m 60 \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"mweb_getRouteStatus","params":[]}' \
  'http://YOURHOP0.onion:8334'
```

You want JSON back, not a SOCKS/timeout error. Repeat a cheap ping to hop 1 and hop 2 if you want. Until hop 0 answers, do not run the forger.

---

## Step 9 — This Mac (taker): local coinswapd + sidecar + submit

Taker keys stay on this Mac. Sidecar talks to **local** `coinswapd` on **8546**, not hop 0.

### 9a) Confirm a 1 LTC coin

```bash
./scripts/proof-a-inspect.sh
```

Pick a mature unlocked coin with `amount_sats: 100000000`. If scan/spend are missing from `deploy/mixnet.operator.env`:

```bash
# GUI must be quit
cargo run --manifest-path "$HOME/Dev/ltc-wallet-mac/Cargo.toml" -p wallet-cli -- \
  --data-dir "$HOME/Library/Application Support/com.indigonakamoto.ltc-wallet" \
  mweb-coinswapd-env --out /path/to/mwixnet-litvm/deploy/mixnet.operator.env
chmod 600 deploy/mixnet.operator.env
```

Load scan/spend/dest in your shell from that file. Do not print them.

### 9b) Taker `coinswapd`

`-k` on this process must **not** be hop 0’s `MIX_K` (omit `-k` to get a random key, or use any unused 32-byte hex). Directory is for hop **URLs** only.

```bash
export HTTP_PROXY=socks5h://127.0.0.1:9050
export HTTPS_PROXY="$HTTP_PROXY"
export NO_PROXY=127.0.0.1,localhost

./bin/coinswapd-research \
  -mln-local-taker -mln-submit-remote \
  -mln-directory deploy/mixnet.directory.json \
  -mweb-scan-secret "$MWEB_SCAN_SECRET" \
  -mweb-spend-secret "$MWEB_SPEND_SECRET" \
  -a "$E2E_MWEB_DEST" \
  -l 8546
```

Wait until this process’s Neutrino is at tip and the 1 LTC coin is spendable.

### 9c) Sidecar

```bash
./bin/mln-sidecar -mode=rpc -rpc-url http://127.0.0.1:8546 -port 8080
```

### 9d) Forger — no `-trigger-batch`

```bash
./bin/mln-cli route from-directory -directory deploy/mixnet.directory.json -out route.json
./bin/mln-cli forger -route-json route.json -dry-run=false \
  -dest "$E2E_MWEB_DEST" -amount 100000000 \
  -coinswapd-url http://127.0.0.1:8080/v1/swap
```

The onion is queued on **hop 0**, not on this Mac. Local `pendingOnions` stays **0**. That is expected.

---

## Step 10 — Hop 0 host: run the batch

SSH to hop 0 (or wait until UTC midnight on that process):

```bash
cd ~/mln-mixnode
MIXNODE_RPC_URL=http://127.0.0.1:8334 ./scripts/mixnode-run-batch.sh
```

`mweb_getRouteStatus` on hop 0 is **queue hygiene**, not the pass bar.

---

## Step 11 — This Mac: dest coin

In the Mac wallet (GUI or `proof-a-inspect.sh` after quitting GUI), look for a **new 99970000 sat** coin (1 LTC minus `10000×3`). Hop-fee outputs of about **7300 sat** each may appear at the `-a` addresses (same wallet if you reused dest in Step 2a).

Record a **new** `LIVE_COINSWAP_ATTEMPT_YYYY-MM-DD-proof-b.md` at the repo root. Label it **Proof B**. Include host roles (not keys), `amountSat`, dest amount scanned, and that LitVM/Nostr/`mlnd` were unused.

---

## If something fails

| Symptom | Likely cause |
| ------- | ------------ |
| `curl` to `.onion` times out | Tor down, wrong HS port, `HiddenServicePort` ≠ `LISTEN_PORT`, firewall, or this Mac missing `socks5h` |
| Submit / `swap_swap` dial error | Taker `HTTP_PROXY` not set; hop 0 not listening; directory `tor` URL wrong |
| `FetchCoin` / validate onion | Hop 0 Neutrino not at tip, or UTXO not 100000000 sat / not visible on mainnet yet |
| Startup: pub hex mismatch | Directory row ≠ this hop’s `MIX_K` |
| Startup: missing directory | Unpack path does not contain `deploy/mixnet.directory.json` |
| Batch rejects mixed values | Another onion with a different `amountSat` is still queued on hop 0 |
| Dest never scans 99970000 | Mix did not finalize; wrong dest; or you looked at `pendingOnions` on the taker |

---

## Copy checklist (no secrets in the list)

**To every hop:** `coinswapd-research` (correct GOOS), `scripts/mixnode-start.sh`, `deploy/mixnet.directory.json`.

**To hop 0 only (in addition):** `scripts/mixnode-run-batch.sh`.

**To each hop, unique:** `deploy/mixnet.hop.env` with **that** hop’s `MIX_K` and `MLN_HOP_INDEX`.

**Never to a hop:** `deploy/mixnet.operator.env`, wallet data dir, scan/spend, this Mac’s Proof A `MIX_K0/1/2` unless you intentionally reuse those pubs in the new directory (usually you generate fresh keys in Step 2).
