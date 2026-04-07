# mln-judge (interim LitVM dispute judge)

Daemon that subscribes to `Contested` on `GrievanceCourt`, replays `defendGrievance` calldata from the emitting transaction, checks `keccak256(defenseData)` matches the log, decodes defense tuple (v1), and optionally broadcasts `adjudicateGrievance`.

## Environment

| Variable | Meaning |
|----------|---------|
| `JUDGE_LITVM_WS_URL` | WebSocket JSON-RPC (default `ws://127.0.0.1:8545`) |
| `JUDGE_COURT_ADDR` | GrievanceCourt (`0x` + 40 hex) |
| `JUDGE_PRIVATE_KEY` | Interim judge key (must match contract `judge`) |
| `JUDGE_DRY_RUN` | If `1`, log `cast` hint and decoded defense fields only |
| `JUDGE_AUTO_ADJUDICATE` | If `1` and not dry-run, submit `adjudicateGrievance` |
| `JUDGE_VERDICT` | Required when auto-adjudicating: `exonerate` or `uphold` (slash accuser claim) |

Automated **signature verification** against `nextHopPubkey` / `signature` is not implemented (v1 stub): treat `JUDGE_DRY_RUN=1` as the safe default until PRODUCT_SPEC §13.6 message formats are fixed; then extend `internal/judge/service.go`.

## Manual fallback (`cast`)

When dry-run logs a hint:

```bash
cast send "$COURT" "adjudicateGrievance(bytes32,bool)" "$GRIEVANCE_ID" true \
  --rpc-url "$RPC_URL" --private-key "$JUDGE_PRIVATE_KEY"
```

Use `false` to uphold the accuser (slash economics aligned with timeout slash).

## Build

```bash
cd mln-judge && go build -o bin/mln-judge ./cmd/mln-judge
```
