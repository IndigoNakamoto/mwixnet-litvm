#!/usr/bin/env bash
# Trigger mweb_runBatch on a local mixnode hop 0 JSON-RPC (Proof B operator, not the taker sidecar).
# Usage: MIXNODE_RPC_URL=http://127.0.0.1:8334 scripts/mixnode-run-batch.sh
set -euo pipefail
URL="${MIXNODE_RPC_URL:-http://127.0.0.1:8334}"
curl -sS -m 60 -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"mweb_runBatch","params":[]}' \
  "$URL"
echo
