# Live coinswap attempt — 2026-08-12 (Proof A pass)

**Bar:** **Proof A** (lab shuffle). Not Proof B.

Proof A means three mixnode `coinswapd` processes complete peel → `swap_forward` / `swap_backward` → one aggregated MWEB tx, and the dest wallet scans a **new same-value coin**. Three Tor HS on one host is enough. It does **not** show “one honest mixer.”

**Outcome:** **Pass.** Taker coin **10000000 sat** (0.1 LTC) was spent. Hop-0 Neutrino (same scan key as the Mac wallet dest) lists a **new 9970000 sat** coin plus **three 7300 sat** hop-fee outputs. Net wallet drop **8100 sat** (kernel/weight fees). LitVM / Nostr / `mlnd` were not used. **Not Proof B.**

Do not overwrite [`LIVE_COINSWAP_ATTEMPT_2026-08-12.md`](LIVE_COINSWAP_ATTEMPT_2026-08-12.md), [`LIVE_COINSWAP_ATTEMPT_2026-08-12-mac-wallet.md`](LIVE_COINSWAP_ATTEMPT_2026-08-12-mac-wallet.md), or [`LIVE_COINSWAP_ATTEMPT_2026-04-15.md`](LIVE_COINSWAP_ATTEMPT_2026-04-15.md).

**Secrets:** No scan/spend hex, no mnemonic, no passphrase, no mix `-k` in this file.

## What ran

- **Taker coin source:** Mac wallet (`ltc-wallet-mac`) data dir only. No CoinSwap UI in Tauri.
- **Mixnet:** three `coinswapd-research` processes + Tor HS on this host; hop 0 is `-mln-local-taker` on `:8546`; hops 1–2 on `:8335` / `:8336`; sidecar `:8080` → hop 0.
- **Route:** `mln-cli route from-directory` (no `epochId` / `accuser` / `operator` / `swapId`). Directory `amountSat=10000000`, `feeMinSat=10000` × 3.
- **Submit:** `mln-cli forger` → sidecar `POST /v1/swap` → `mweb_submitRoute`. First funded try: HTTP 502 `output pubkey mismatch` (master spend vs tweaked `b_i`); **no spend**. Fix: `spendKeyForRewoundCoin` in `research/coinswapd/mln_wallet.go`. Second try: route accepted; `POST /v1/route/batch` → `mweb_runBatch`.
- **Never** `-mweb-dev-clear-pending-after-batch`.

## Why `pendingOnions=0` was not the pass signal

Immediately after batch, sidecar reported `pendingOnions=0` while hop-0 **availableSat was still 3699736359**. That is queue hygiene (or a too-fast status poll), **not** Proof A. After Neutrino restarted against the same DB at tip, availableSat was **3699728259** (−8100) and the exact **10000000** coin was gone.

`mweb_submitRoute` then failed exact-match on 10000000 and listed spendable values including **9970000** and **7300, 7300, 7300**. Pre-mix Mac-wallet unspent list had **10000000** and neither of those values.

Dest output = `amount − 3×feeMinSat` = **9970000**. Three hop-fee outputs after kernel share ≈ **7300** each. **9970000 + 3×7300 + 8100 = 10000000**.

## Mac wallet GUI

`wallet-cli combined-summary` during inspect reported `mweb_stale: true` / “MWEB not synced yet” and still listed the old **10000000** coin. Trust hop-0 Neutrino (caught up) for this pass. Open the Tauri app after quitting any CLI lock and wait for MWEB sync to see **9970000**.

## Not claimed

- **Proof B** (stranger / 1 LTC / published first hop / 1-of-N honest).
- README **Phase 3-mix** complete (Proof B still open).
- Unlinkability (same host, same dest as fee address).

**Proof B** remains parked. 3-gov (Nostr + LitVM court) remains parked until you choose to unpark it after this Proof A pass.
