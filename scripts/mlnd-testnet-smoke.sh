#!/usr/bin/env bash
# Cheap RPC + contract sanity checks for LitVM testnet (or any EVM JSON-RPC over HTTP).
# Does not open grievances or spend funds.
#
# Required:
#   MLND_HTTP_URL — HTTP JSON-RPC base URL for `cast` (not WebSocket; often https://…)
#   MLND_COURT_ADDR — GrievanceCourt contract (0x-prefixed hex)
#
# Optional alias: LITVM_TESTNET_HTTP_URL (used if MLND_HTTP_URL is unset)
#
# Values are not hardcoded here; copy RPC and addresses from official LitVM docs and
# research/LITVM.md.
#
# Usage (from repo root):
#   MLND_HTTP_URL=https://… MLND_COURT_ADDR=0x… ./scripts/mlnd-testnet-smoke.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${FOUNDRY_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"

HTTP_URL="${MLND_HTTP_URL:-${LITVM_TESTNET_HTTP_URL:-}}"
COURT="${MLND_COURT_ADDR:-}"

if [[ -z "$HTTP_URL" || -z "$COURT" ]]; then
  echo "Error: set MLND_HTTP_URL (HTTP JSON-RPC for cast) and MLND_COURT_ADDR."
  echo "Optional: LITVM_TESTNET_HTTP_URL if MLND_HTTP_URL is unset."
  echo "See research/LITVM.md and https://docs.litvm.com/"
  exit 1
fi

# When using cast via Docker, 127.0.0.1 / localhost refer to the container — use host gateway.
RPC_CAST="$HTTP_URL"
if ! command -v cast >/dev/null 2>&1; then
  case "$HTTP_URL" in
    http://127.0.0.1:*)
      RPC_CAST="${HTTP_URL/http:\/\/127.0.0.1/http://host.docker.internal}"
      ;;
    http://localhost:*)
      RPC_CAST="${HTTP_URL/http:\/\/localhost/http://host.docker.internal}"
      ;;
  esac
fi

if command -v cast >/dev/null 2>&1; then
  cast_run() { cast "$@"; }
  RPC_FOR_CAST="$HTTP_URL"
else
  cast_run() {
    docker run --rm --add-host=host.docker.internal:host-gateway \
      --entrypoint cast "$IMAGE" "$@"
  }
  RPC_FOR_CAST="$RPC_CAST"
fi

echo "== mlnd testnet smoke (cast) =="
echo "RPC: $HTTP_URL"
echo "Court: $COURT"

echo "→ chain-id"
chain_id="$(cast_run chain-id --rpc-url "$RPC_FOR_CAST")"
echo "  $chain_id"

echo "→ block-number"
bn="$(cast_run block-number --rpc-url "$RPC_FOR_CAST")"
echo "  $bn"

echo "→ code at GrievanceCourt"
code="$(cast_run code "$COURT" --rpc-url "$RPC_FOR_CAST")"
if [[ "$code" == "0x" || "$code" == "0x0" ]]; then
  echo "Error: no contract code at MLND_COURT_ADDR (check address and network)."
  exit 1
fi
echo "  (bytecode present, ${#code} chars)"

echo "OK: RPC reachable and court address holds code."
