# Nostr end-to-end demo (MLN Phase 2)

This runbook connects **local LitVM contracts** (Anvil) with **live Nostr relays** using the normative wire in [`NOSTR_MLN.md`](NOSTR_MLN.md) (kinds **31250** and **31251**). Use **throwaway Nostr keys**; public relays store what you publish.

## Prerequisites

- Docker (for `make contracts-test` / Foundry image) and Python 3.9+.
- Anvil reachable at `ANVIL_RPC_URL` (default `http://127.0.0.1:8545`) when running the grievance smoke test.

```bash
pip install -r scripts/requirements.txt
```

## 1. Local chain and golden grievance

Start Anvil (host must listen on all interfaces if you use Docker port mapping):

```bash
docker run --rm -p 8545:8545 --entrypoint anvil ghcr.io/foundry-rs/foundry:latest --host 0.0.0.0 --port 8545
```

In another terminal, deploy and open the appendix-13 golden grievance (writes `contracts/broadcast/Deploy.s.sol/<chainId>/run-latest.json`):

```bash
make test-grievance
```

After this, the golden on-chain grievance id is:

`0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3`

with epoch **42** and accused maker **`0x000000000000000000000000000000000000CAfE`** (see [`scripts/test-grievance-local.sh`](../scripts/test-grievance-local.sh)).

## 2. Watch the relay

Subscribe for MLN maker ads and grievance pointers (filters match [`scripts/nostr_watch.py`](../scripts/nostr_watch.py)):

```bash
python3 scripts/nostr_watch.py --relay wss://relay.damus.io --duration 120
```

Leave this running.

## 3. Publish a kind-31251 pointer

Generate a **32-byte hex** Nostr private key (64 hex chars, no `0x`). Then:

```bash
export NOSTR_PRIVKEY_HEX='<64_hex_chars>'

python3 scripts/publish_grievance.py \
  0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3 \
  42 \
  "$NOSTR_PRIVKEY_HEX" \
  --broadcast-json contracts/broadcast/Deploy.s.sol/31337/run-latest.json \
  --accused 0x000000000000000000000000000000000000CAfE \
  --chain-id 31337 \
  > /tmp/mln-grievance-event.json
```

Publish that JSON to the same relay:

```bash
python3 scripts/nostr_watch.py --relay wss://relay.damus.io --publish-json /tmp/mln-grievance-event.json
```

You should see **`[EVENT] kind=31251`** (or similar) in the watcher terminal. Relays may deduplicate by event `id`; changing `created_at` requires re-signing (regenerate with `publish_grievance.py`).

## 4. One-shot stack (optional)

`make test-full-stack-with-nostr` runs the grievance smoke test and then prints filter hints for [`scripts/mln-nostr-demo.py`](../scripts/mln-nostr-demo.py). For a **live** watch + publish flow, use steps 2–3 above.

## Safety

- Do not put `evidenceHash` preimages or raw MWEB material on Nostr; only pointers and hashes per [`NOSTR_MLN.md`](NOSTR_MLN.md).
- Default relay URLs are third-party infrastructure; expect latency and rate limits.
