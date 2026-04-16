# Maker box setup (macOS, Apple Silicon or Intel)

One-page runbook to turn a Mac into a Tier 2 / Tier 3 MLN maker. Do this on each maker box. Canonical context: [`research/PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md`](../research/PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md), [`research/PHASE_3_TIER2_RELAY.md`](../research/PHASE_3_TIER2_RELAY.md), [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../PHASE_3_MWEB_HANDOFF_SLICE.md).

## 0. Prereqs (once per box)

```bash
brew install go tor jq git coreutils openssl@3 curl
# Apple Silicon: make sure Homebrew Go wins the PATH race over any old /usr/local/go install.
echo 'export PATH="/opt/homebrew/opt/go/bin:$PATH"' >> ~/.zshrc && exec zsh
go version   # expect 1.22+
```

Clone the repo and build binaries:

```bash
git clone <your fork/origin> ~/Projects/mwixnet-litvm
cd ~/Projects/mwixnet-litvm
make build build-research-coinswapd
```

Only the relay host also needs Docker (for `nostr-rs-relay`).

## 1. Tor hidden service

Edit `/opt/homebrew/etc/tor/torrc` (Apple Silicon) or `/usr/local/etc/tor/torrc` (Intel). Append:

```
HiddenServiceDir /opt/homebrew/var/lib/tor/mln-maker
HiddenServicePort 8334 127.0.0.1:8334
```

Start Tor and read the onion hostname:

```bash
brew services start tor
sleep 2
cat /opt/homebrew/var/lib/tor/mln-maker/hostname
```

The printed string is this box's maker URL: `http://<56char>.onion:8334`. Save it.

## 2. Generate this box's identities (once)

Do this on **whichever box has the repo and funded LitVM gas** (typically the MacBook Pro). You will run it three times total — once per maker.

```bash
# EVM operator key + nsec + X25519 swap pub (public half — you keep the private half).
openssl rand -hex 32 | sed 's/^/0x/'                                # EVM private key
openssl rand -hex 32                                                # Nostr nsec hex
openssl genpkey -algorithm X25519 -outform DER | tail -c 32 | xxd -p -c 64   # X25519 pub hex
```

Fund each EVM address with a little LitVM 4441 gas. Onboard each maker on chain 4441 **once** (from whichever box has funded gas; doesn't need to run on the maker box):

```bash
export MLN_LITVM_HTTP_URL=https://liteforge.rpc.caldera.xyz/http
export MLN_LITVM_CHAIN_ID=4441
export MLN_REGISTRY_ADDR=0x01bd8c4fca29cddd354472b3f31ef243ba92ffe7

MLN_OPERATOR_ETH_KEY=<this maker's EVM PK> \
MLN_NOSTR_NSEC=<this maker's nsec hex> \
  ./bin/mln-cli maker onboard -execute
```

## 3. Env file (per box)

On this maker box:

```bash
cd ~/Projects/mwixnet-litvm
cp deploy/tier2.maker-box.env.example deploy/tier2.maker-box.env
chmod 600 deploy/tier2.maker-box.env
$EDITOR deploy/tier2.maker-box.env
```

Fill in the `REPLACE_THIS_MAKER_*` values for this box only. The shared lines (relay URL, LitVM endpoints, registry/court, chain id) stay identical across all three boxes.

## 4. Start the daemons

```bash
./scripts/maker-box-up.sh            # builds if needed, starts coinswapd-research + mlnd in background
# or, to watch coinswapd in the foreground:
./scripts/maker-box-up.sh foreground
```

Logs:

```bash
tail -f deploy/.maker-box.logs/coinswapd.log
tail -f deploy/.maker-box.logs/mlnd.log
```

Stop:

```bash
./scripts/maker-box-up.sh stop
```

## 5. Verify (from the taker box)

On the MacBook Pro (or wherever you run `mln-cli`):

```bash
export MLN_NOSTR_RELAYS=ws://<relay-host>:7080/
export MLN_LITVM_HTTP_URL=https://liteforge.rpc.caldera.xyz/http
export MLN_LITVM_CHAIN_ID=4441
export MLN_REGISTRY_ADDR=0x01bd8c4fca29cddd354472b3f31ef243ba92ffe7
export MLN_GRIEVANCE_COURT_ADDR=0xc303368899eac7508cfdaaedf9b8d03f75462593

./bin/mln-cli doctor
```

Gate: every maker box should add +1 to the `verified` count. When it reads `verified=3, with tor=3`, Tier 2 is done.

If a maker is missing, check in this order:

1. `curl -X POST -H 'content-type: application/json' -d '{"jsonrpc":"2.0","method":"mweb_getBalance","params":[],"id":1}' http://127.0.0.1:8334` on the maker host — proves `coinswapd-research` is listening.
2. `curl --socks5-hostname 127.0.0.1:9050 <same> http://<that-maker>.onion:8334` from the taker host with Tor running — proves Tor HS reachability.
3. `tail deploy/.maker-box.logs/mlnd.log` on the maker — look for Nostr publish lines, then for `chainId mismatch` / onboarding issues.
4. `./bin/mln-cli scout` (no `-quiet`) on the taker — prints rejection reasons per event.

## 6. Tier 3 (funded) extras

Everything above is Tier 2 (no money moves). For Tier 3:

- Fill real `MWEB_SCAN_SECRET` / `MWEB_SPEND_SECRET` for the coin the taker will spend (on the taker box, not the makers).
- Wait for Neutrino sync on each `coinswapd-research` — hours on first boot. Kick off early.
- On the taker box only: `make phase3-funded-preflight` — must exit 0 or 2 (warnings only).
- Success bar: `mln-cli forger -trigger-batch -wait-batch` drives `pendingOnions` to 0 without any dev-clear flag.

## Security hygiene

- `chmod 600` anything holding `MWEB_*_SECRET`, `MLND_OPERATOR_PRIVATE_KEY`, `MLND_NOSTR_NSEC`, `COINSWAPD_MESH_K`.
- Never commit `deploy/tier2.maker-box.env` — only the `.example` template is tracked.
- Don't paste full onion hostnames or route payloads into public tickets while a live run is in progress.
