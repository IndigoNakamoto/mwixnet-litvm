# Proof A from the Mac wallet (2026-08-12)

## Goal

Use `/Users/indigo/Dev/ltc-wallet-mac` as the taker coin source for a funded Proof A mix in this repo: inspect MWEB coins first, then (after amount confirmation) run a 3-hop `coinswapd` shuffle on one Mac. Dest wallet scans a new same-value coin. No CoinSwap UI in Tauri.

## In scope / out of scope

**In:** `wallet-cli` inspect + lab export; `coinswapd -mln-directory` / `-mln-hop-index`; gitignored operator directory; three Tor HS; Scout-free `route from-directory`; dated Proof A attempt record.

**Out:** Spending before a confirmed coin; Proof B; CoinSwap screens in the Mac wallet; Scout / `mlnd` / LitVM on the happy path.

## Primary files and canonical docs

- [`scripts/proof-a-inspect.sh`](../../scripts/proof-a-inspect.sh)
- [`deploy/mixnet.operator.env.example`](../../deploy/mixnet.operator.env.example), [`deploy/mixnet.directory.example.json`](../../deploy/mixnet.directory.example.json)
- [`research/coinswapd/mln_directory.go`](../../research/coinswapd/mln_directory.go)
- [`research/PHASE_3_OPERATOR_PLAYBOOK.md`](../../research/PHASE_3_OPERATOR_PLAYBOOK.md) Part B0
- [`LIVE_COINSWAP_ATTEMPT_2026-08-12-mac-wallet.md`](../../LIVE_COINSWAP_ATTEMPT_2026-08-12-mac-wallet.md)

## Execution results

- `wallet-cli combined-summary`, `mweb-unspent`, and `mweb-coinswapd-env --out` shipped in `ltc-wallet-mac` (merge-safe chmod 600 writer; secrets not printed).
- After `WALLET_PASSPHRASE` was in `deploy/mixnet.operator.env` (chmod 600), inspect listed a mature unlocked **10000000 sat** coin. That amount was confirmed and used as directory `amountSat`.
- Mixnodes used `-mln-directory` + `-mln-hop-index` (public `getNodes` skipped). First live submit: 502 `output pubkey mismatch` (master spend vs tweaked `b_i`); **no spend**. Fix: `spendKeyForRewoundCoin`.
- Second submit accepted; `mweb_runBatch` ran. Immediate `pendingOnions=0` with **unchanged** balance was **not** treated as a pass. After Neutrino restart: **10000000** gone; new **9970000** dest coin + **3×7300** hop-fee outputs; net **−8100 sat**.
- Pass record: [`LIVE_COINSWAP_ATTEMPT_2026-08-12-proof-a-pass.md`](../../LIVE_COINSWAP_ATTEMPT_2026-08-12-proof-a-pass.md). Not Proof B.

## Verification

```bash
cd research/coinswapd && go test ./...
./bin/mln-cli route from-directory -directory deploy/mixnet.directory.example.json -out /tmp/route.json
make phase3-operator-preflight
./scripts/proof-a-inspect.sh   # needs WALLET_PASSPHRASE
```

## Layer-boundary check

MWEB (`coinswapd`) owns the shuffle and hop fees. Tor is transport. The Mac wallet only supplies keys/UTXO/dest scan. LitVM and Nostr were not used.

## Follow-ups

Proof B (stranger / 1 LTC / published first hop) stays parked. Tauri CoinSwap UI was not added in this session (gate is now a Proof A pass file, not an invitation to ship wallet mix UX unasked).
