#!/usr/bin/env bash
# Deploy MwixnetRegistry + GrievanceCourt to a local Anvil (no LitVM testnet).
# Prerequisites: Anvil listening on ANVIL_RPC_URL (default http://127.0.0.1:8545).
# Start Anvil in another terminal (image entrypoint must not bind loopback-only):
#   docker run --rm -p 8545:8545 --entrypoint anvil ghcr.io/foundry-rs/foundry:latest --host 0.0.0.0
# Uses the well-known first Anvil private key — LOCAL TESTING ONLY.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/contracts"

IMAGE="${FOUNDRY_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"
RPC="${ANVIL_RPC_URL:-http://127.0.0.1:8545}"
# forge runs inside Docker; 127.0.0.1 there is not the host. macOS/Win Docker maps host.docker.internal.
RPC_DOCKER="$RPC"
case "$RPC" in
  http://127.0.0.1:*)
    RPC_DOCKER="${RPC/http:\/\/127.0.0.1/http://host.docker.internal}"
    ;;
  http://localhost:*)
    RPC_DOCKER="${RPC/http:\/\/localhost/http://host.docker.internal}"
    ;;
esac
export PRIVATE_KEY="${ANVIL_PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"

forge_docker() {
  docker run --rm --add-host=host.docker.internal:host-gateway \
    -e PRIVATE_KEY -e MIN_STAKE -e COOLDOWN_PERIOD -e CHALLENGE_WINDOW -e GRIEVANCE_BOND_MIN \
    -v "$ROOT/contracts:/work" -w /work --entrypoint forge "$IMAGE" "$@"
}

cast_docker() {
  docker run --rm --add-host=host.docker.internal:host-gateway \
    -v "$ROOT/contracts:/work" -w /work --entrypoint cast "$IMAGE" "$@"
}

if ! cast_docker block-number --rpc-url "$RPC_DOCKER" &>/dev/null; then
  echo "Error: nothing listening at $RPC (from Foundry container: $RPC_DOCKER)"
  echo "Start Anvil, e.g.: docker run --rm -p 8545:8545 --entrypoint anvil $IMAGE --host 0.0.0.0"
  exit 1
fi

echo "Deploying to $RPC ..."
forge_docker script script/Deploy.s.sol:Deploy --rpc-url "$RPC_DOCKER" --broadcast -vvv

echo
echo "Broadcast JSON: contracts/broadcast/ (see latest run-* folder)"
echo "Optional: copy contract addresses to contracts/deployments/anvil-local.json (gitignored)"
