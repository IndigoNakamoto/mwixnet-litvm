# coinswapd Appendix 13 receipts (completed 2026-04-07)

## Goal

Real appendix 13 correlators on `swap_forward` failure (post-peel commitment hash + Keccak of forward payload `P`), per-hop LitVM `operator` on the route when epoch/accuser/swapId are set, async delivery via `mweb_getLastReceipt`, sidecar `GET /v1/route/receipt`, forger vault polling after `mweb_runBatch`, and threat-model notes for `unsigned-swap-forward-failure-v1`.

## Key files

- `research/coinswapd/litvmreceipt/` — `sha256(commitment[:])`, `Keccak256(P)`, JSON marshal
- `research/coinswapd/swap.go` — `recordSwapForwardFailure` (dial vs `rpc_application`), mutex `lastReceipt*`
- `research/coinswapd/mweb_service.go` — `GetLastReceipt`, `SubmitRoute` sets `mlnPeerOperators`
- `research/coinswapd/mlnroute/request.go` — `Hop.Operator`, validation with LitVM trio
- `mln-sidecar/internal/mweb/rpc_bridge.go` — `HandleLastReceipt`; `internal/api/server.go` — `/v1/route/receipt`
- `mln-cli/internal/forger/` — `HopRequest.Operator`, `GetRouteLastReceipt`, `pollAppendixReceipt`, `PersistLastReceiptHTTP`
- `research/THREAT_MODEL_MLN.md` — new row + history for sentinel / fairness

## Verification

- `go test ./...` in `research/coinswapd`, `mln-sidecar`, `mln-cli/internal/forger`

## Note

`litvmreceipt` uses `golang.org/x/crypto/sha3` (not `go-ethereum/crypto`) to avoid duplicate secp256k1 symbols when linking the coinswapd test binary.
