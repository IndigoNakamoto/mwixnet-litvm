# Proof B published hop group — software first (2026-08-12)

## Goal

Unblock Proof B in software: the taker is a client, not hop 0. Mixnodes keep no wallet keys; published hop 0 accepts a prebuilt onion (`swap_swap`). One-hop start scripts and a Part B1 runbook. Do not spend 1 LTC or claim a live pass until hosts are named.

## In scope / out of scope

**In:** `-mln-submit-remote`; directory `Swap` keeps `mlnPeers`; mixnode start/batch scripts; forger batch split; playbook Part B1; README/CHANGELOG.

**Out:** Spending 1 LTC; Proof B pass file; Tauri CoinSwap; 3-gov on mixnodes; new `PHASE_*.md` playbooks.

## Primary files and canonical docs

- [`research/coinswapd/mweb_service.go`](../../research/coinswapd/mweb_service.go), [`research/coinswapd/main.go`](../../research/coinswapd/main.go)
- [`scripts/mixnode-start.sh`](../../scripts/mixnode-start.sh), [`deploy/mixnet.hop.env.example`](../../deploy/mixnet.hop.env.example)
- [`research/PHASE_3_OPERATOR_PLAYBOOK.md`](../../research/PHASE_3_OPERATOR_PLAYBOOK.md) Part B1
- [`research/COINSWAPD_MLN_FORK_SPEC.md`](../../research/COINSWAPD_MLN_FORK_SPEC.md) §5.4a

## Execution results

- Taker `mweb_submitRoute` with `-mln-submit-remote` Dials hop 0 `swap_swap` (go-ethereum v1.14 name) and does not `saveOnion` locally. `performSwap` is a no-op on the taker.
- Optional `-mln-directory` on the taker pins hop URLs without matching `-k` to hop 0.
- Directory hop `Swap` keeps `mlnPeers` when length is 3.
- Tests: `TestSwapSwapRemote_dialsHop0`, `TestClearVanillaSwapMeta_keepsDirectoryPeers`, `TestPerformSwap_submitRemoteNoOp`.
- [`scripts/mixnode-start.sh`](../../scripts/mixnode-start.sh) / [`scripts/mixnode-run-batch.sh`](../../scripts/mixnode-run-batch.sh); hop env has no scan/spend.
- Forger `-trigger-batch` warns that Proof B batch is hop 0 operator / midnight.
- `E2E_MWEB_FULL=1` stub not re-run here (port 8546 in use; Docker daemon down). Unit tests passed.

## Verification

```bash
cd research/coinswapd && go test .
go test ./internal/forger ./internal/mixnetdir   # mln-cli
go test ./internal/mweb ./internal/api           # mln-sidecar
```

## Layer-boundary check

MWEB (`coinswapd`) owns onion build (taker) and peel/forward (mixnodes). Tor is transport for `swap_swap` / `swap_forward`. LitVM and Nostr were not added to the happy path.

## Follow-ups

Name three hop hosts; directory `amountSat=100000000`; forger without `-trigger-batch`; dest **99970000**. New `LIVE_COINSWAP_ATTEMPT_*.md` labeled Proof B.
