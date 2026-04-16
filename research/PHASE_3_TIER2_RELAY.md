# Phase 3 Tier 2 — curated Nostr relay

This note closes the **discovery gap** that blocked the 2026-04-15 live attempt: `mln-cli scout` on a public relay (`wss://relay.damus.io`) returned `(no verified makers)` with stderr `chainId mismatch` because the relay carried no kind-31250 ads bound to LitVM chain **4441** and the deployed registry / court.

Tier 2 treats the relay as **your** infrastructure, not a public good. Pointing three `mlnd` makers and the taker at the **same** curated relay is what makes `mln-cli scout` return a non-empty verified set.

## What counts as "curated"

A Nostr relay (plain [`scsibug/nostr-rs-relay`](https://hub.docker.com/r/scsibug/nostr-rs-relay) is fine) where:

1. The three Tier 2 makers publish kind **31250** ads with `litvm.chainId=4441` and `litvm.registry` / `litvm.grievanceCourt` matching [`deploy/litvm-addresses.generated.env`](../deploy/litvm-addresses.generated.env).
2. The taker's `MLN_NOSTR_RELAYS` points at the same `wss://` URL.
3. Optionally, NIP-42 AUTH is required so `.onion` endpoints are not scraped by anonymous clients (the 2026-04-15 hardening plan: [`.cursor/plans/2026-04-15-nip42-auth-relay-hardening.md`](../.cursor/plans/2026-04-15-nip42-auth-relay-hardening.md)).

Public relays (Damus, `nos.lol`, etc.) *can* work for Tier 2 if all three makers aggressively publish there and the taker filters correctly, but their rate limits and churn make them unreliable for coordinated tests. Prefer running your own.

## Three deployment shapes

### Shape A — single shared host, cleartext wss

Easiest. One operator runs `scsibug/nostr-rs-relay` on a host reachable by all four stacks:

```bash
docker run -d --name mln-tier2-relay -p 7080:8080 \
  -v "$(pwd)/deploy/nostr-rs-relay.toml:/usr/src/app/config.toml:ro" \
  scsibug/nostr-rs-relay:latest
```

Then every maker env gets `MLND_NOSTR_RELAYS=ws://relay-host:7080/` and the taker exports `MLN_NOSTR_RELAYS=ws://relay-host:7080/`. Same as [`deploy/docker-compose.e2e.yml`](../deploy/docker-compose.e2e.yml) but bound to a reachable interface instead of loopback.

Pros: trivial. Cons: exposes `.onion` fields in ads to anyone who can reach the relay.

### Shape B — NIP-42 AUTH (recommended for Tier 2)

Uncomment `[authorization]` in [`deploy/nostr-rs-relay.toml`](../deploy/nostr-rs-relay.toml) and set an allowlist of maker + taker x-only pubkeys. Operator env:

- Makers: `MLND_NOSTR_AUTH=true` (the [`mlnd` broadcaster AUTH](../mlnd/internal/nostr/broadcaster.go) derives pubkey from `MLND_NOSTR_NSEC` and signs the AUTH event).
- Taker: `MLN_NOSTR_AUTH_NSEC=<nsec1…>` (wired through [`mln-cli` Scout](../mln-cli/internal/scout/scout.go)).

Rejection is fail-fast with backoff, so misconfigured keys surface quickly.

### Shape C — relay reachable only over Tor

Run the relay on loopback on the relay host, expose via a Tor hidden service, distribute the `.onion` as `MLND_NOSTR_RELAYS=wss://<onion>:8080/`. Cleanest for threat-model purposes; requires every participant to have Tor SOCKS configured for their Nostr client. `go-nostr` / `SimplePool` respect `HTTP_PROXY` similarly to the `coinswapd` dialer (see [`research/PHASE_3_TOR_OPERATOR_LAB.md`](PHASE_3_TOR_OPERATOR_LAB.md)).

## Verifying

After makers start publishing ads:

```bash
export MLN_NOSTR_RELAYS=wss://your-tier2-relay/
export MLN_LITVM_HTTP_URL=https://liteforge.rpc.caldera.xyz/http
export MLN_LITVM_CHAIN_ID=4441
export MLN_REGISTRY_ADDR=0x01bd8c4fca29cddd354472b3f31ef243ba92ffe7
export MLN_GRIEVANCE_COURT_ADDR=0xc303368899eac7508cfdaaedf9b8d03f75462593
./bin/mln-cli doctor
```

The `doctor` summary should show: LitVM `eth_chainId` matches **4441**, relay TCP reachable, and scout verified count **≥ 3** with every row having a non-empty `tor`. That is the Tier 2 success bar; Tier 3 (funded swap) adds `make phase3-funded-preflight`.

## What this does **not** solve

- **L1 inclusion proofs** / happy-path mix validation — stays off-chain per [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) §10.
- **Stake source of truth** — Nostr ads are hints; `mln-cli scout` still cross-checks each ad against `MwixnetRegistry` on LitVM 4441.
- **Maker-to-maker `.onion` reachability** — a curated relay does not fix Tor transport between makers; that remains [`PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md`](PHASE_3_OPERATOR_PARTB_STEPBYSTEP.md) §3.
