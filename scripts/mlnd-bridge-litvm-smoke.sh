#!/usr/bin/env bash
# Fixture smoke: golden NDJSON (EvidenceGoldenVectors.t.sol) → mlnd bridge → SQLite →
# openGrievance on Anvil → mlnd logs "validated receipt".
#
# Prerequisites: Anvil on ANVIL_RPC_URL, cast or Docker Foundry, python3.
# Runs mlnd with host `go` when Go is 1.22+, otherwise `docker run golang:1.22` (needs Docker).
# Does not run coinswapd.
#
# Usage: from repo root, after Anvil is up:
#   ./scripts/mlnd-bridge-litvm-smoke.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${FOUNDRY_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"
RPC="${ANVIL_RPC_URL:-http://127.0.0.1:8545}"
RPC_DOCKER="$RPC"
case "$RPC" in
  http://127.0.0.1:*)
    RPC_DOCKER="${RPC/http:\/\/127.0.0.1/http://host.docker.internal}"
    ;;
  http://localhost:*)
    RPC_DOCKER="${RPC/http:\/\/localhost/http://host.docker.internal}"
    ;;
esac
CHAIN_ID="${ANVIL_CHAIN_ID:-31337}"

if command -v cast >/dev/null 2>&1; then
  cast_run() { cast "$@"; }
  RPC_CAST="$RPC"
else
  cast_run() {
    docker run --rm --add-host=host.docker.internal:host-gateway \
      -v "$ROOT/contracts:/work" -w /work --entrypoint cast "$IMAGE" "$@"
  }
  RPC_CAST="$RPC_DOCKER"
fi

GO_IMAGE="${MLND_SMOKE_GO_IMAGE:-golang:1.22}"
use_docker_mlnd=false
if ! command -v go >/dev/null 2>&1; then
  use_docker_mlnd=true
elif ! go version 2>/dev/null | grep -qE 'go1\.(2[2-9]|[3-9][0-9])\.'; then
  use_docker_mlnd=true
fi
if [[ "$use_docker_mlnd" == true ]]; then
  if ! command -v docker >/dev/null 2>&1; then
    echo "Error: host Go is below 1.22 (or missing) and Docker is not available to run mlnd."
    exit 1
  fi
fi

# Golden vectors (must match EvidenceGoldenVectors.t.sol and test-grievance-local.sh)
readonly ACCUSER="0x000000000000000000000000000000000000bEef"
readonly ACCUSED="0x000000000000000000000000000000000000CAfE"
readonly EPOCH_ID=42
readonly EVIDENCE_HASH="0x2d4d7ae96f39e2d5037f21782bc831874261ffe22743f74bbf865a39ec4df112"
readonly EXPECT_GRIEVANCE_ID="0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3"

if ! cast_run block-number --rpc-url "$RPC_CAST" &>/dev/null; then
  echo "Error: nothing listening at $RPC"
  echo "Start Anvil, e.g.: docker run --rm -p 8545:8545 --entrypoint anvil $IMAGE --host 0.0.0.0"
  exit 1
fi

echo "== Deploy contracts =="
"$ROOT/scripts/deploy-local-anvil.sh"

LATEST="$ROOT/contracts/broadcast/Deploy.s.sol/$CHAIN_ID/run-latest.json"
if [[ ! -f "$LATEST" ]]; then
  echo "Error: expected broadcast artifact at $LATEST"
  exit 1
fi

COURT="$(python3 -c "
import json, sys
path = sys.argv[1]
with open(path) as f:
    d = json.load(f)
court = None
for t in d.get('transactions', []):
    if t.get('transactionType') != 'CREATE':
        continue
    if t.get('contractName') == 'GrievanceCourt':
        court = t['contractAddress']
        break
if not court:
    print('missing GrievanceCourt in broadcast', file=sys.stderr)
    sys.exit(1)
print(court)
" "$LATEST")"

echo "GrievanceCourt: $COURT"

BRIDGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mln-bridge-XXXXXX")"
MLND_LOG="$(mktemp "${TMPDIR:-/tmp}/mln-smoke-XXXXXX.log")"
MLND_DB=""
MLND_PID=""
MLND_DOCKER_NAME=""

cleanup_files() {
  rm -rf "$BRIDGE_DIR" 2>/dev/null || true
  rm -f "$MLND_LOG" "$MLND_DB" 2>/dev/null || true
}

kill_mlnd() {
  if [[ -n "${MLND_DOCKER_NAME:-}" ]]; then
    docker stop "$MLND_DOCKER_NAME" 2>/dev/null || true
    MLND_DOCKER_NAME=""
  elif [[ -n "${MLND_PID:-}" ]] && kill -0 "$MLND_PID" 2>/dev/null; then
    kill "$MLND_PID" 2>/dev/null || true
    wait "$MLND_PID" 2>/dev/null || true
  fi
}

refresh_mlnd_log() {
  if [[ -n "${MLND_DOCKER_NAME:-}" ]]; then
    docker logs "$MLND_DOCKER_NAME" >"$MLND_LOG" 2>&1 || true
  fi
}

finish() {
  kill_mlnd
  cleanup_files
}
trap finish EXIT

# hopIndex 2; peeled / forwardCt match Solidity golden test (uint256 0x1111 / 0x2222 as bytes32)
NDJSON='{"epochId":"42","accuser":"'"$ACCUSER"'","accusedMaker":"'"$ACCUSED"'","hopIndex":2,"peeledCommitment":"0x0000000000000000000000000000000000000000000000000000000000001111","forwardCiphertextHash":"0x0000000000000000000000000000000000000000000000000000000000002222","nextHopPubkey":"golden-smoke","signature":"golden-smoke"}'
printf '%s\n' "$NDJSON" >"$BRIDGE_DIR/golden.ndjson"
echo "Wrote fixture NDJSON → $BRIDGE_DIR/golden.ndjson"

echo "== Start mlnd (bridge + watcher, no auto-defend) =="
if [[ "$use_docker_mlnd" == true ]]; then
  MLND_DOCKER_NAME="mln-operator-smoke-$$"
  docker rm -f "$MLND_DOCKER_NAME" 2>/dev/null || true
  docker run -d --rm --name "$MLND_DOCKER_NAME" \
    --add-host=host.docker.internal:host-gateway \
    -v "$ROOT:/w" \
    -v "$BRIDGE_DIR:/mln-bridge-in:ro" \
    -e MLND_WS_URL="ws://host.docker.internal:8545" \
    -e MLND_COURT_ADDR="$COURT" \
    -e MLND_OPERATOR_ADDR="$ACCUSED" \
    -e MLND_DB_PATH="/tmp/mlnd-smoke.db" \
    -e MLND_BRIDGE_COINSWAPD=1 \
    -e MLND_BRIDGE_RECEIPTS_DIR="/mln-bridge-in" \
    -e MLND_BRIDGE_POLL_INTERVAL=200ms \
    -w /w/mlnd \
    "$GO_IMAGE" \
    go run ./cmd/mlnd >/dev/null
  # Module download + subscribe can take a while on first run.
  for _ in {1..90}; do
    if docker logs "$MLND_DOCKER_NAME" 2>&1 | grep -q "watching GrievanceOpened"; then
      break
    fi
    sleep 2
  done
  refresh_mlnd_log
else
  MLND_DB="$(mktemp "${TMPDIR:-/tmp}/mln-smoke-XXXXXX.db")"
  (
    cd "$ROOT/mlnd"
    export MLND_WS_URL="${MLND_WS_URL:-ws://127.0.0.1:8545}"
    export MLND_COURT_ADDR="$COURT"
    export MLND_OPERATOR_ADDR="$ACCUSED"
    export MLND_DB_PATH="$MLND_DB"
    export MLND_BRIDGE_COINSWAPD=1
    export MLND_BRIDGE_RECEIPTS_DIR="$BRIDGE_DIR"
    export MLND_BRIDGE_POLL_INTERVAL=200ms
    go run ./cmd/mlnd
  ) >"$MLND_LOG" 2>&1 &
  MLND_PID=$!
  sleep 6
fi

echo "== openGrievance (golden evidenceHash) =="
cast_run rpc --rpc-url "$RPC_CAST" anvil_setBalance "$ACCUSER" 0x21e19e0c9bab2400000 &>/dev/null
cast_run rpc --rpc-url "$RPC_CAST" anvil_impersonateAccount "$ACCUSER" &>/dev/null
cast_run send --rpc-url "$RPC_CAST" "$COURT" \
  "openGrievance(address,uint256,bytes32)" "$ACCUSED" "$EPOCH_ID" "$EVIDENCE_HASH" \
  --value 0.1ether --from "$ACCUSER" --unlocked

sleep 3

refresh_mlnd_log

if ! grep -q "validated receipt for grievance" "$MLND_LOG"; then
  echo "Error: expected mlnd to log validated receipt. Last 40 lines of mlnd log:"
  tail -40 "$MLND_LOG"
  exit 1
fi

echo "OK — mlnd validated receipt for grievance $EXPECT_GRIEVANCE_ID (golden vectors)."
