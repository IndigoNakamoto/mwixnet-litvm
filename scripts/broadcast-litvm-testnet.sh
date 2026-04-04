#!/usr/bin/env bash
# Broadcast MwixnetRegistry + GrievanceCourt to a real chain using Foundry on the host PATH.
# Prerequisites: forge installed; contracts/.env with PRIVATE_KEY + RPC_URL (from LitVM docs).
#
# Usage (repo root):
#   ./scripts/broadcast-litvm-testnet.sh
#
# Next: record addresses for mlnd / CLI
#   make record-litvm-deploy
#   # or: python3 scripts/record-litvm-deploy.py --write deploy/litvm-addresses.generated.env
#
# See PHASE_16_PUBLIC_TESTNET.md and research/LITVM.md.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/contracts"

if [[ ! -f .env ]]; then
  echo "Copy contracts/.env.example to contracts/.env and set PRIVATE_KEY and RPC_URL (LitVM testnet HTTP JSON-RPC)."
  exit 1
fi

set -a
# shellcheck disable=SC1091
source .env
set +a

if [[ -z "${RPC_URL:-}" ]]; then
  echo "RPC_URL must be set in contracts/.env"
  exit 1
fi
if [[ -z "${PRIVATE_KEY:-}" ]]; then
  echo "PRIVATE_KEY must be set in contracts/.env"
  exit 1
fi

if ! command -v forge >/dev/null 2>&1; then
  echo "forge not on PATH. Install Foundry or use Docker (see research/LITVM.md) and run:"
  echo "  forge script script/Deploy.s.sol:Deploy --rpc-url \"\$RPC_URL\" --broadcast -vvv"
  exit 1
fi

echo "Broadcasting via forge to RPC_URL (chain id comes from the endpoint) ..."
forge script script/Deploy.s.sol:Deploy --rpc-url "$RPC_URL" --broadcast -vvv

echo ""
echo "=== Next steps ==="
echo "1) Record addresses:  make record-litvm-deploy"
echo "   or: python3 \"$ROOT/scripts/record-litvm-deploy.py\" --write \"$ROOT/deploy/litvm-addresses.generated.env\""
echo "2) Merge fragment into deploy/.env.testnet and set MLND_WS_URL + keys (see deploy/.env.testnet.example)."
echo "3) docker compose -f deploy/docker-compose.testnet.yml up -d"
