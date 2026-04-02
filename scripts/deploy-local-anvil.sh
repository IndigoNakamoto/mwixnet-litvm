#!/usr/bin/env bash
# Deploy MwixnetRegistry + GrievanceCourt to a local Anvil (no LitVM testnet).
# Prerequisites: Anvil listening on ANVIL_RPC_URL (default http://127.0.0.1:8545).
# Start Anvil in another terminal, e.g.:
#   docker run --rm -p 8545:8545 ghcr.io/foundry-rs/foundry:latest anvil --host 0.0.0.0
# Uses the well-known first Anvil private key — LOCAL TESTING ONLY.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/contracts"

IMAGE="${FOUNDRY_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"
RPC="${ANVIL_RPC_URL:-http://127.0.0.1:8545}"
export PRIVATE_KEY="${ANVIL_PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"

forge_docker() {
  docker run --rm -e PRIVATE_KEY -e MIN_STAKE -e CHALLENGE_WINDOW -e GRIEVANCE_BOND_MIN \
    -v "$ROOT/contracts:/work" -w /work "$IMAGE" --entrypoint forge "$@"
}

if ! forge_docker cast block-number --rpc-url "$RPC" &>/dev/null; then
  echo "Error: nothing listening at $RPC"
  echo "Start Anvil, e.g.: docker run --rm -p 8545:8545 $IMAGE anvil --host 0.0.0.0"
  exit 1
fi

echo "Deploying to $RPC ..."
forge_docker script script/Deploy.s.sol:Deploy --rpc-url "$RPC" --broadcast -vvv

echo
echo "Broadcast JSON: contracts/broadcast/ (see latest run-* folder)"
echo "Optional: copy contract addresses to contracts/deployments/anvil-local.json (gitignored)"
